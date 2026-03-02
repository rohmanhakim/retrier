package retrier_test

import (
	"context"
	"testing"
	"time"

	retrier "github.com/rohmanhakim/retrier"
)

// mockRetryableError is a simple retryable error for testing.
type mockRetryableError struct {
	msg string
}

func (e *mockRetryableError) Error() string                    { return e.msg }
func (e *mockRetryableError) RetryPolicy() retrier.RetryPolicy { return retrier.RetryPolicyAuto }

// mockLogger captures backoff delays for verification.
type backoffMockLogger struct {
	enabled       bool
	logRetryCalls []backoffLogCall
}

type backoffLogCall struct {
	attempt int
	backoff time.Duration
}

func (m *backoffMockLogger) Enabled() bool { return m.enabled }

func (m *backoffMockLogger) LogRetry(_ context.Context, attempt int, _ int, backoff time.Duration, _ error, _ ...any) {
	m.logRetryCalls = append(m.logRetryCalls, backoffLogCall{
		attempt: attempt,
		backoff: backoff,
	})
}

// TestBackoff_ZeroJitter tests that jitter=0 produces deterministic delays.
// This indirectly tests computeJitter(max <= 0) returning 0.
func TestBackoff_ZeroJitter(t *testing.T) {
	initialDuration := 20 * time.Millisecond
	multiplier := 2.0

	// Run multiple times and collect first backoff delays
	// With jitter=0, all delays should be exactly equal to initialDuration
	uniqueDelays := make(map[time.Duration]int)

	for i := 0; i < 10; i++ {
		mock := &backoffMockLogger{enabled: true}
		callCount := 0

		fn := func() (string, error) {
			callCount++
			if callCount == 1 {
				return "", &mockRetryableError{msg: "error"}
			}
			return "success", nil
		}

		opts := []retrier.RetryOption{
			retrier.WithMaxAttempts(2),
			retrier.WithJitter(0), // Zero jitter
			retrier.WithInitialDuration(initialDuration),
			retrier.WithMultiplier(multiplier),
			retrier.WithMaxDuration(1 * time.Minute),
		}

		retrier.Retry(context.Background(), mock, fn, opts...)

		if len(mock.logRetryCalls) > 0 {
			uniqueDelays[mock.logRetryCalls[0].backoff]++
		}
	}

	// With zero jitter, all delays should be identical (exactly initialDuration)
	if len(uniqueDelays) != 1 {
		t.Errorf("Expected exactly 1 unique delay with jitter=0, got %d unique delays: %v", len(uniqueDelays), uniqueDelays)
	}

	for delay := range uniqueDelays {
		if delay != initialDuration {
			t.Errorf("Expected delay=%v with jitter=0, got %v", initialDuration, delay)
		}
	}
}

// TestBackoff_NegativeJitter tests that negative jitter is treated as no jitter.
// This indirectly tests computeJitter(max <= 0) returning 0.
func TestBackoff_NegativeJitter(t *testing.T) {
	initialDuration := 20 * time.Millisecond

	// Run multiple times with negative jitter
	uniqueDelays := make(map[time.Duration]int)

	for i := 0; i < 10; i++ {
		mock := &backoffMockLogger{enabled: true}
		callCount := 0

		fn := func() (string, error) {
			callCount++
			if callCount == 1 {
				return "", &mockRetryableError{msg: "error"}
			}
			return "success", nil
		}

		opts := []retrier.RetryOption{
			retrier.WithMaxAttempts(2),
			retrier.WithJitter(-10 * time.Millisecond), // Negative jitter
			retrier.WithInitialDuration(initialDuration),
			retrier.WithMultiplier(2.0),
			retrier.WithMaxDuration(1 * time.Minute),
		}

		retrier.Retry(context.Background(), mock, fn, opts...)

		if len(mock.logRetryCalls) > 0 {
			uniqueDelays[mock.logRetryCalls[0].backoff]++
		}
	}

	// With negative jitter, all delays should be identical (no jitter applied)
	if len(uniqueDelays) != 1 {
		t.Errorf("Expected exactly 1 unique delay with negative jitter, got %d unique delays: %v", len(uniqueDelays), uniqueDelays)
	}

	for delay := range uniqueDelays {
		if delay != initialDuration {
			t.Errorf("Expected delay=%v with negative jitter, got %v", initialDuration, delay)
		}
	}
}

// TestBackoff_DelayCappedAtMaxDuration tests that exponential backoff is capped at maxDuration.
// This indirectly tests exponentialBackoffDelay() when delay > maxBackoff.
func TestBackoff_DelayCappedAtMaxDuration(t *testing.T) {
	maxDuration := 50 * time.Millisecond
	// Configure options where exponential calculation would exceed maxDuration
	// initialDuration * multiplier^5 = 100ms * 10^5 = huge number > 50ms maxDuration
	opts := []retrier.RetryOption{
		retrier.WithMaxAttempts(7),
		retrier.WithJitter(0), // No jitter for predictable testing
		retrier.WithInitialDuration(100 * time.Millisecond),
		retrier.WithMultiplier(10.0), // High multiplier
		retrier.WithMaxDuration(maxDuration),
	}

	mock := &backoffMockLogger{enabled: true}
	callCount := 0

	fn := func() (string, error) {
		callCount++
		if callCount < 6 {
			return "", &mockRetryableError{msg: "error"}
		}
		return "success", nil
	}

	retrier.Retry(context.Background(), mock, fn, opts...)

	// All backoffs after the first few should be capped at maxDuration
	for i, call := range mock.logRetryCalls {
		// Skip the success log (last entry has backoff=0)
		if call.backoff == 0 {
			continue
		}
		if call.backoff > maxDuration {
			t.Errorf("Attempt %d: backoff %v exceeds maxDuration %v", call.attempt, call.backoff, maxDuration)
		}
		// For later attempts (high exponent), backoff should be exactly maxDuration
		if i >= 2 && call.backoff != maxDuration {
			t.Logf("Attempt %d: backoff %v (expected to be capped at %v)", call.attempt, call.backoff, maxDuration)
		}
	}
}

// TestBackoff_DelayCappedAtMaxDuration_Table tests various configurations where delay exceeds maxBackoff.
func TestBackoff_DelayCappedAtMaxDuration_Table(t *testing.T) {
	tests := []struct {
		name            string
		initialDuration time.Duration
		multiplier      float64
		maxDuration     time.Duration
		attempts        int
		jitter          time.Duration
	}{
		{
			name:            "high multiplier exceeds max quickly",
			initialDuration: 100 * time.Millisecond,
			multiplier:      10.0,
			maxDuration:     50 * time.Millisecond,
			attempts:        3,
			jitter:          0,
		},
		{
			name:            "large initial equals max",
			initialDuration: 100 * time.Millisecond,
			multiplier:      2.0,
			maxDuration:     100 * time.Millisecond,
			attempts:        3,
			jitter:          0,
		},
		{
			name:            "very high multiplier",
			initialDuration: 10 * time.Millisecond,
			multiplier:      100.0,
			maxDuration:     50 * time.Millisecond,
			attempts:        3,
			jitter:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &backoffMockLogger{enabled: true}
			callCount := 0

			fn := func() (string, error) {
				callCount++
				if callCount < tt.attempts {
					return "", &mockRetryableError{msg: "error"}
				}
				return "success", nil
			}

			opts := []retrier.RetryOption{
				retrier.WithMaxAttempts(tt.attempts + 2),
				retrier.WithJitter(tt.jitter),
				retrier.WithInitialDuration(tt.initialDuration),
				retrier.WithMultiplier(tt.multiplier),
				retrier.WithMaxDuration(tt.maxDuration),
			}

			retrier.Retry(context.Background(), mock, fn, opts...)

			// Verify no backoff exceeds maxDuration
			for _, call := range mock.logRetryCalls {
				if call.backoff > 0 && call.backoff > tt.maxDuration {
					t.Errorf("backoff %v exceeds maxDuration %v", call.backoff, tt.maxDuration)
				}
			}
		})
	}
}

// TestBackoff_ExponentialGrowth tests that backoff grows exponentially until capped.
func TestBackoff_ExponentialGrowth(t *testing.T) {
	initialDuration := 10 * time.Millisecond
	multiplier := 2.0
	maxDuration := 1 * time.Minute
	jitter := time.Duration(0) // No jitter for predictable testing

	mock := &backoffMockLogger{enabled: true}
	callCount := 0

	fn := func() (string, error) {
		callCount++
		if callCount < 5 {
			return "", &mockRetryableError{msg: "error"}
		}
		return "success", nil
	}

	opts := []retrier.RetryOption{
		retrier.WithMaxAttempts(6),
		retrier.WithJitter(jitter),
		retrier.WithInitialDuration(initialDuration),
		retrier.WithMultiplier(multiplier),
		retrier.WithMaxDuration(maxDuration),
	}

	retrier.Retry(context.Background(), mock, fn, opts...)

	// Verify exponential growth pattern
	// Expected: 10ms, 20ms, 40ms, 80ms
	expectedDelays := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		40 * time.Millisecond,
		80 * time.Millisecond,
	}

	for i, call := range mock.logRetryCalls {
		// Skip success log (backoff=0)
		if call.backoff == 0 {
			continue
		}
		if i < len(expectedDelays) {
			if call.backoff != expectedDelays[i] {
				t.Errorf("Attempt %d: expected backoff %v, got %v", call.attempt, expectedDelays[i], call.backoff)
			}
		}
	}
}
