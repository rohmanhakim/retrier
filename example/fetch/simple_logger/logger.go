// Package simple_logger provides a simple implementation of the retrier.DebugLogger interface
// that prints retry information to stdout.
package simple_logger

import (
	"context"
	"fmt"
	"time"

	"github.com/rohmanhakim/retrier"
)

// SimpleLogger implements retrier.DebugLogger interface.
// It prints retry attempts to stdout with timestamps.
type SimpleLogger struct{}

// NewSimpleLogger creates a new SimpleLogger instance.
func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{}
}

// Enabled returns true to enable debug logging.
func (l *SimpleLogger) Enabled() bool {
	return true
}

// LogRetry prints retry information to stdout.
// It shows the attempt number, max attempts, backoff duration, and error (if any).
func (l *SimpleLogger) LogRetry(_ context.Context, attempt, maxAttempts int, backoff time.Duration, err error, _ ...any) {
	timestamp := time.Now().Format("15:04:05.000")

	if err == nil {
		// Success case
		fmt.Printf("[%s] ✅ Success on attempt %d/%d\n", timestamp, attempt, maxAttempts)
	} else if backoff > 0 {
		// Retry case - will retry after backoff
		fmt.Printf("[%s] ❌ Attempt %d/%d failed: %v\n", timestamp, attempt, maxAttempts, err)
		fmt.Printf("[%s]    ↳ Retrying in %v...\n", timestamp, backoff)
	} else {
		// Exhausted case - no more retries
		fmt.Printf("[%s] 🛑 Attempt %d/%d failed (exhausted): %v\n", timestamp, attempt, maxAttempts, err)
	}
}

// Interface assertion to ensure SimpleLogger implements DebugLogger
var _ retrier.DebugLogger = (*SimpleLogger)(nil)
