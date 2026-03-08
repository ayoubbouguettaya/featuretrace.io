package input

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// dockerConfigV2 is the minimal subset of config.v2.json we need.
type dockerConfigV2 struct {
	Name string `json:"Name"`
}

// readContainerName reads the container name from config.v2.json in the
// container's directory. The Name field looks like "/redis"; we strip the
// leading slash. Returns "" on any error.
func readContainerName(containerDir string) string {
	configPath := filepath.Join(containerDir, "config.v2.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("[docker-input] cannot read %s: %v", configPath, err)
		return ""
	}
	var cfg dockerConfigV2
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("[docker-input] cannot parse %s: %v", configPath, err)
		return ""
	}
	return strings.TrimPrefix(cfg.Name, "/")
}

// discoverDockerLogs finds all *-json.log files under the Docker container
// storage directory.
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
			containerName := readContainerName(containerDir)

			result = append(result, DiscoverDockerLogsFile{
				Path:          matchedPath,
				ContainerID:   containerID,
				ContainerName: containerName,
			})
		}
	}
	return result, nil
}
