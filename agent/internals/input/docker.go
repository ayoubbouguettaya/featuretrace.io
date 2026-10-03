package input

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const defaultDockerLogRoot = "/var/lib/docker/containers"

// DockerInput tails all container JSON-log files produced by the Docker
// daemon. Each container stores its logs at:
//
//	/var/lib/docker/containers/<id>/<id>-json.log
//
// DockerInput discovers these files and spawns a tailer goroutine per file.
type DockerInput struct {
	LogRoot string // override for testing
}

// In a new or existing model file
type RawLog struct {
	Data          []byte
	ContainerID   string
	ContainerName string
}

type DiscoverDockerLogsFile struct {
	Path          string
	ContainerID   string
	ContainerName string
}

// Start satisfies the Input interface.
func (d *DockerInput) Start(ctx context.Context, out chan<- RawLog) error {
	root := d.LogRoot
	if root == "" {
		root = defaultDockerLogRoot
	}

	logFiles, err := discoverDockerLogs(root)
	if err != nil {
		return err
	}

	if len(logFiles) == 0 {
		log.Printf("[docker-input] no container log files found under %s", root)
	}

	fmt.Println("logFiles", logFiles)

	var wg sync.WaitGroup
	for _, path := range logFiles {
		wg.Add(1)
		go func(p DiscoverDockerLogsFile) {
			defer wg.Done()

			log.Printf("[docker-input] tailing %s", p.Path)
			if err := tailFile(ctx, p.Path, p, out); err != nil && ctx.Err() == nil {
				log.Printf("[docker-input] tail stopped for %s: %v", p.Path, err)
			}
		}(path)
	}

	// Block until all tailers exit (context cancelled)
	wg.Wait()
	return ctx.Err()
}

// enabledLabel is the container label that opts a container into log
// collection, e.g. in docker-compose:
//
//	labels:
//	  - feature-trace.enabled=true
const enabledLabel = "feature-trace.enabled"

// dockerConfigV2 is the minimal subset of config.v2.json we need.
type dockerConfigV2 struct {
	Name   string `json:"Name"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

// readContainerConfig reads config.v2.json in the container's directory.
// Returns nil on any error.
func readContainerConfig(containerDir string) *dockerConfigV2 {
	configPath := filepath.Join(containerDir, "config.v2.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("[docker-input] cannot read %s: %v", configPath, err)
		return nil
	}
	var cfg dockerConfigV2
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[docker-input] cannot parse %s: %v", configPath, err)
		return nil
	}
	return &cfg
}

// containerName returns the container name without the leading slash
// (config.v2.json stores it as "/redis").
func (c *dockerConfigV2) containerName() string {
	return strings.TrimPrefix(c.Name, "/")
}

// traceEnabled reports whether the container has opted in via enabledLabel.
func (c *dockerConfigV2) traceEnabled() bool {
	enabled, _ := strconv.ParseBool(c.Config.Labels[enabledLabel])
	return enabled
}

// discoverDockerLogs finds all *-json.log files under the Docker container
// storage directory, keeping only containers labelled enabledLabel=true.
func discoverDockerLogs(root string) ([]DiscoverDockerLogsFile, error) {
	pattern := filepath.Join(root, "*", "*-json.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Filter to regular files only
	var result []DiscoverDockerLogsFile
	for _, matchedPath := range matches {
		info, statErr := os.Stat(matchedPath)
		if statErr == nil && !info.IsDir() {
			containerDir := filepath.Dir(matchedPath)
			containerID := filepath.Base(containerDir)
			cfg := readContainerConfig(containerDir)
			if cfg == nil {
				continue
			}
			containerName := cfg.containerName()

			if !cfg.traceEnabled() {
				log.Printf("[docker-input] skipping %s (no %s=true label)", containerName, enabledLabel)
				continue
			}

			result = append(result, DiscoverDockerLogsFile{
				Path:          matchedPath,
				ContainerID:   containerID,
				ContainerName: containerName,
			})
		}
	}
	return result, nil
}
