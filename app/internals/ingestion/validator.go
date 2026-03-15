package ingestion

import (
	"errors"
	"time"

	"featuretrace.io/app/internals/model"
)

var (
	ErrEmptyMessage   = errors.New("message must not be empty")
	ErrInvalidLevel   = errors.New("level must be one of: debug, info, warn, error, fatal")
	ErrFutureTimestamp = errors.New("timestamp is in the future")
)

// allowedLevels is the set of recognised log levels.
var allowedLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
	"fatal": true,
}

// ValidateRecord checks that a log record is well-formed.
func ValidateRecord(rec *model.LogRecord) error {
	if rec.Message == "" {
		return ErrEmptyMessage
	}

	if !allowedLevels[rec.Level] {
		return ErrInvalidLevel
	}

	// Reject timestamps more than 1 minute in the future to catch clock skew
	// without being too strict.
	if rec.Timestamp.After(time.Now().UTC().Add(1 * time.Minute)) {
		return ErrFutureTimestamp
	}

	// Default empty level to "info".
	if rec.Level == "" {
		rec.Level = "info"
	}

	return nil
}
