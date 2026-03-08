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

// tailFile continuously reads new lines appended to a file. It handles
// truncation (log rotation) by re-seeking to the beginning.
func tailFile(ctx context.Context, path string, file DiscoverDockerLogsFile, out chan<- RawLog) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Seek to end — we only want new data
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	var lastSize int64

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

		// EOF — check for truncation (log rotation)
		info, statErr := f.Stat()
		if statErr == nil {
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
