package logger

import (
	"log"
	"os"
)

// Logger provides levelled, structured-ish logging for all FeatureTrace
// backend services. It is intentionally simple (stdlib log) to keep
// dependencies light; swap for zerolog/zap when needed.
type Logger struct {
	info  *log.Logger
	warn  *log.Logger
	err   *log.Logger
	debug *log.Logger
}

// New creates a Logger with the given component prefix.
func New(component string) *Logger {
	prefix := "[" + component + "] "
	return &Logger{
		info:  log.New(os.Stdout, prefix+"INFO  ", log.LstdFlags|log.Lmsgprefix),
		warn:  log.New(os.Stdout, prefix+"WARN  ", log.LstdFlags|log.Lmsgprefix),
		err:   log.New(os.Stderr, prefix+"ERROR ", log.LstdFlags|log.Lmsgprefix),
		debug: log.New(os.Stdout, prefix+"DEBUG ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (l *Logger) Info(msg string, args ...any)  { l.info.Printf(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.warn.Printf(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.err.Printf(msg, args...) }
func (l *Logger) Debug(msg string, args ...any) { l.debug.Printf(msg, args...) }

// Fatal logs and exits.
func (l *Logger) Fatal(msg string, args ...any) {
	l.err.Fatalf(msg, args...)
}

