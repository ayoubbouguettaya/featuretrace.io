package input

import "context"

// Input is the contract every log source must satisfy.
// Start begins producing raw log lines into out. It blocks until ctx is
// cancelled, at which point it must drain cleanly and return.
type Input interface {
	Start(ctx context.Context, out chan<- RawLog) error
}
