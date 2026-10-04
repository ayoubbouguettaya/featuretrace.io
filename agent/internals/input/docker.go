package input

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultDockerLogRoot      = "/var/lib/docker/containers"
	defaultDockerScanInterval = 5 * time.Second
)

// DockerInput tails all container JSON-log files produced by the Docker
// daemon. Each container stores its logs at:
//
//	/var/lib/docker/containers/<id>/<id>-json.log
//
// DockerInput spawns a tailer goroutine per opted-in container and rescans
// the directory every ScanInterval to follow containers that are started or
// removed while the agent runs.
type DockerInput struct {
	LogRoot      string        // override for testing
	ScanInterval time.Duration // defaults to defaultDockerScanInterval
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
	interval := d.ScanInterval
	if interval <= 0 {
		interval = defaultDockerScanInterval
	}

	w := &containerWatcher{
		root:    root,
		out:     out,
		tailers: make(map[string]tailer),
		skipped: make(map[string]bool),
		exited:  make(chan string),
	}

	// A missing or unreadable log root is a misconfiguration, not something
	// a later scan will fix.
	if _, err := os.ReadDir(root); err != nil {
		return err
	}

	// Containers already running: only new lines matter.
	w.scan(ctx, false)
	if len(w.tailers) == 0 {
		log.Printf("[docker-input] no containers labelled %s=true under %s yet, rescanning every %s", enabledLabel, root, interval)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.scan(ctx, true)
		case id := <-w.exited:
			// Forget tailers that died on their own so the next scan retries.
			delete(w.tailers, id)
		case <-ctx.Done():
			// Block until all tailers exit
			w.wg.Wait()
			return ctx.Err()
		}
	}
}

// tailer is a running tailFile goroutine for one container.
type tailer struct {
	name string
	stop context.CancelFunc
}

// containerWatcher keeps one tailer per opted-in container. Its maps are
// only touched from the DockerInput.Start goroutine.
type containerWatcher struct {
	root    string
	out     chan<- RawLog
	tailers map[string]tailer // by container ID
	skipped map[string]bool   // container IDs without the opt-in label
	exited  chan string       // tailers send their container ID when they exit
	wg      sync.WaitGroup
}

// scan starts a tailer for every new opted-in container and stops the
// tailer of every removed one. fromStart makes new tailers read the log
// from the beginning, so lines written before the scan noticed the
// container are not lost.
func (w *containerWatcher) scan(ctx context.Context, fromStart bool) {
	logs, err := listContainers(w.root)
	if err != nil {
		log.Printf("[docker-input] cannot scan %s: %v", w.root, err)
		return
	}

	for id, t := range w.tailers {
		if _, ok := logs[id]; !ok {
			log.Printf("[docker-input] container %s removed, stopping tailer", t.name)
			t.stop()
			delete(w.tailers, id)
		}
	}
	for id := range w.skipped {
		if _, ok := logs[id]; !ok {
			delete(w.skipped, id)
		}
	}

	for id, path := range logs {
		if _, ok := w.tailers[id]; ok || w.skipped[id] {
			continue
		}
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			continue // never started, so no log file yet
		}
		cfg := readContainerConfig(filepath.Dir(path))
		if cfg == nil {
			continue // retried on the next scan
		}
		name := cfg.containerName()
		if !cfg.traceEnabled() {
			log.Printf("[docker-input] skipping %s (no %s=true label)", name, enabledLabel)
			w.skipped[id] = true
			continue
		}
		w.startTailer(ctx, DiscoverDockerLogsFile{
			Path:          path,
			ContainerID:   id,
			ContainerName: name,
		}, fromStart)
	}
}

func (w *containerWatcher) startTailer(ctx context.Context, file DiscoverDockerLogsFile, fromStart bool) {
	tailCtx, stop := context.WithCancel(ctx)
	w.tailers[file.ContainerID] = tailer{name: file.ContainerName, stop: stop}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer stop()

		log.Printf("[docker-input] tailing %s (%s)", file.ContainerName, file.Path)
		if err := tailFile(tailCtx, file.Path, file, w.out, fromStart); err != nil && tailCtx.Err() == nil {
			log.Printf("[docker-input] tail stopped for %s: %v", file.Path, err)
		}

		select {
		case w.exited <- file.ContainerID:
		case <-ctx.Done():
		}
	}()
}

// listContainers maps the ID of every container under root to the path of
// its JSON log file. The file may be missing: containers that were never
// started have none, and Docker briefly removes it while rotating. Only a
// removed container loses its directory.
func listContainers(root string) (map[string]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	logs := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			id := e.Name()
			logs[id] = filepath.Join(root, id, id+"-json.log")
		}
	}
	return logs, nil
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
