package input

import (
	"context"
	"log"
	"os"
	"path/filepath"
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

// Start satisfies the Input interface.
func (d *DockerInput) Start(ctx context.Context, out chan<- []byte) error {
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

	var wg sync.WaitGroup
	for _, path := range logFiles {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			log.Printf("[docker-input] tailing %s", p)
			if err := tailFile(ctx, p, out); err != nil && ctx.Err() == nil {
				log.Printf("[docker-input] tail stopped for %s: %v", p, err)
			}
		}(path)
	}

	// Block until all tailers exit (context cancelled)
	wg.Wait()
	return ctx.Err()
}

// discoverDockerLogs finds all *-json.log files under the Docker container
// storage directory.
func discoverDockerLogs(root string) ([]string, error) {
	pattern := filepath.Join(root, "*", "*-json.log")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Filter to regular files only
	var result []string
	for _, m := range matches {
		info, statErr := os.Stat(m)
		if statErr == nil && !info.IsDir() {
			result = append(result, m)
		}
	}
	return result, nil
}

