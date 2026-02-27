package retrier_test

import (
	"context"
	"errors"
	"testing"
	"time"

	retrier "github.com/rohmanhakim/retrier"
)

// TestNoOpLogger_Enabled tests that NoOpLogger.Enabled returns false.
func TestNoOpLogger_Enabled(t *testing.T) {
	logger := retrier.NewNoOpLogger()
	if logger.Enabled() {
		t.Error("NoOpLogger.Enabled() should return false")
	}
}

// TestNoOpLogger_LogRetry tests that NoOpLogger.LogRetry is a no-op.
func TestNoOpLogger_LogRetry(t *testing.T) {
	logger := retrier.NewNoOpLogger()

	// LogRetry should not panic and should do nothing
	// This test verifies it can be called with various parameters without error
	logger.LogRetry(context.Background(), 1, 3, 100*time.Millisecond, errors.New("test error"))
	logger.LogRetry(context.Background(), 2, 3, 200*time.Millisecond, nil)
	logger.LogRetry(context.TODO(), 0, 0, 0, nil) // context.TODO and zero values
}

// TestNoOpLogger_NewNoOpLogger tests that NewNoOpLogger returns a valid instance.
func TestNoOpLogger_NewNoOpLogger(t *testing.T) {
	logger := retrier.NewNoOpLogger()
	if logger == nil {
		t.Error("NewNoOpLogger() should not return nil")
	}
}

// TestNoOpLogger_ImplementsDebugLogger verifies NoOpLogger implements DebugLogger interface.
func TestNoOpLogger_ImplementsDebugLogger(t *testing.T) {
	// This test ensures NoOpLogger properly implements the DebugLogger interface
	var _ retrier.DebugLogger = retrier.NewNoOpLogger()
}
