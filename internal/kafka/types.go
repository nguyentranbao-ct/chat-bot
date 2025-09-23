package kafka

import (
	"fmt"
	"time"
)

// ErrShutdown is returned when the consumer should shutdown
var ErrShutdown = fmt.Errorf("shutdown requested")

// ErrRetry represents a retryable error with a delay
type ErrRetry struct {
	Err   error
	Delay time.Duration
}

func (e *ErrRetry) Error() string {
	return fmt.Sprintf("retry after %v: %v", e.Delay, e.Err)
}

func (e *ErrRetry) Unwrap() error {
	return e.Err
}

func (e *ErrRetry) GetDelayTime() time.Duration {
	return e.Delay
}

// NewRetryError creates a new retryable error
func NewRetryError(err error, delay time.Duration) *ErrRetry {
	return &ErrRetry{
		Err:   err,
		Delay: delay,
	}
}
