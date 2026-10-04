package input

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"time"
)

const (
	tailPollInterval = 250 * time.Millisecond
)

// tailFile continuously reads new lines appended to a file. It follows both
// kinds of Docker log rotation: renaming the file away and creating a new
// one (max-file > 1), and truncating it in place (max-file = 1).
// Unless fromStart is set, it starts at the end of the file.
func tailFile(ctx context.Context, path string, file DiscoverDockerLogsFile, out chan<- RawLog, fromStart bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { f.Close() }()

	// Seek to end — we only want new data
	if !fromStart {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return err
		}
	}

	reader := bufio.NewReader(f)
	var lastSize int64
	renamed := false // path now names a new file; switch once f is drained

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err == nil {
			// Got a complete line
			cpy := make([]byte, len(line))
			copy(cpy, line)
			select {
			case out <- RawLog{
				Data:          cpy,
				ContainerID:   file.ContainerID,
				ContainerName: file.ContainerName,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		if err != io.EOF {
			log.Printf("[tail] read error on %s: %v", path, err)
			return err
		}

		// EOF on a renamed file: Docker no longer writes to it, so everything
		// has been read — continue with the new file from its start.
		if renamed {
			next, err := os.Open(path)
			if err != nil {
				return err
			}
			log.Printf("[tail] detected rotation on %s, reopening", path)
			f.Close()
			f = next
			reader.Reset(f)
			lastSize = 0
			renamed = false
			continue
		}

		// EOF — check for rotation
		info, statErr := f.Stat()
		if statErr == nil {
			// Renamed: read whatever is left in the old file first. A missing
			// path means Docker has not created the new file yet.
			if pathInfo, err := os.Stat(path); err == nil && !os.SameFile(info, pathInfo) {
				renamed = true
				continue
			}

			// Truncated in place
			currentSize := info.Size()
			if currentSize < lastSize {
				// File was truncated — seek to beginning
				log.Printf("[tail] detected rotation on %s, re-seeking", path)
				if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
					return seekErr
				}
				reader.Reset(f)
				lastSize = 0
				continue
			}
			lastSize = currentSize
		}

		// Nothing new — poll
		select {
		case <-time.After(tailPollInterval):
			reader.Reset(f)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
