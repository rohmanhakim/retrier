package retrier_test

import (
	"errors"
	"testing"

	retrier "github.com/rohmanhakim/retrier"
)

// TestRetryError_NewRetryError tests the creation of RetryError with various parameters.
func TestRetryError_NewRetryError(t *testing.T) {
	innerErr := errors.New("original error")

	tests := []struct {
		name        string
		cause       retrier.RetryErrorCause
		message     string
		policy      retrier.RetryPolicy
		wrapped     error
		wantCause   retrier.RetryErrorCause
		wantMessage string
		wantPolicy  retrier.RetryPolicy
		wantWrapped bool
	}{
		{
			name:        "ErrZeroAttempt with auto policy",
			cause:       retrier.ErrZeroAttempt,
			message:     "cannot retry with zero attempts",
			policy:      retrier.RetryPolicyAuto,
			wrapped:     innerErr,
			wantCause:   retrier.ErrZeroAttempt,
			wantMessage: "cannot retry with zero attempts",
			wantPolicy:  retrier.RetryPolicyAuto,
			wantWrapped: true,
		},
		{
			name:        "ErrExhaustedAttempts with manual policy",
			cause:       retrier.ErrExhaustedAttempts,
			message:     "max retries exceeded",
			policy:      retrier.RetryPolicyManual,
			wrapped:     innerErr,
			wantCause:   retrier.ErrExhaustedAttempts,
			wantMessage: "max retries exceeded",
			wantPolicy:  retrier.RetryPolicyManual,
			wantWrapped: true,
		},
		{
			name:        "ErrExhaustedAttempts with never policy",
			cause:       retrier.ErrExhaustedAttempts,
			message:     "permanent failure",
			policy:      retrier.RetryPolicyNever,
			wrapped:     nil,
			wantCause:   retrier.ErrExhaustedAttempts,
			wantMessage: "permanent failure",
			wantPolicy:  retrier.RetryPolicyNever,
			wantWrapped: false,
		},
		{
			name:        "nil wrapped error",
			cause:       retrier.ErrZeroAttempt,
			message:     "test",
			policy:      retrier.RetryPolicyNever,
			wrapped:     nil,
			wantCause:   retrier.ErrZeroAttempt,
			wantMessage: "test",
			wantPolicy:  retrier.RetryPolicyNever,
			wantWrapped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := retrier.NewRetryError(tt.cause, tt.message, tt.policy, tt.wrapped)

			if err.Cause != tt.wantCause {
				t.Errorf("Cause = %v, want %v", err.Cause, tt.wantCause)
			}

			if err.Message != tt.wantMessage {
				t.Errorf("Message = %v, want %v", err.Message, tt.wantMessage)
			}

			if err.RetryPolicy() != tt.wantPolicy {
				t.Errorf("RetryPolicy() = %v, want %v", err.RetryPolicy(), tt.wantPolicy)
			}

			gotWrapped := err.Unwrap() != nil
			if gotWrapped != tt.wantWrapped {
				t.Errorf("Unwrap() nil = %v, want %v", gotWrapped, tt.wantWrapped)
			}

			if tt.wantWrapped && err.Unwrap() != tt.wrapped {
				t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), tt.wrapped)
			}
		})
	}
}

// TestRetryError_Error tests the Error() method format.
func TestRetryError_Error(t *testing.T) {
	innerErr := errors.New("original error")

	tests := []struct {
		name         string
		cause        retrier.RetryErrorCause
		message      string
		wrapped      error
		wantContains []string
	}{
		{
			name:         "with wrapped error",
			cause:        retrier.ErrExhaustedAttempts,
			message:      "max retries exceeded",
			wrapped:      innerErr,
			wantContains: []string{"retry error", "exhausted attempt", "max retries exceeded", "original error"},
		},
		{
			name:         "without wrapped error",
			cause:        retrier.ErrZeroAttempt,
			message:      "zero attempts not allowed",
			wrapped:      nil,
			wantContains: []string{"retry error", "zero attempt", "zero attempts not allowed"},
		},
		{
			name:         "empty message with wrapped",
			cause:        retrier.ErrExhaustedAttempts,
			message:      "",
			wrapped:      innerErr,
			wantContains: []string{"retry error", "exhausted attempt"},
		},
		{
			name:         "empty message without wrapped",
			cause:        retrier.ErrZeroAttempt,
			message:      "",
			wrapped:      nil,
			wantContains: []string{"retry error", "zero attempt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := retrier.NewRetryError(tt.cause, tt.message, retrier.RetryPolicyAuto, tt.wrapped)
			errStr := err.Error()

			for _, want := range tt.wantContains {
				if !containsString(errStr, want) {
					t.Errorf("Error() = %q, should contain %q", errStr, want)
				}
			}
		})
	}
}

// TestRetryError_Unwrap tests the Unwrap() method for error chain support.
func TestRetryError_Unwrap(t *testing.T) {
	innerErr := errors.New("network error")

	// Test with wrapped error
	err := retrier.NewRetryError(retrier.ErrExhaustedAttempts, "max retries", retrier.RetryPolicyManual, innerErr)
	if err.Unwrap() != innerErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), innerErr)
	}

	// Test with nil wrapped error
	errNil := retrier.NewRetryError(retrier.ErrZeroAttempt, "test", retrier.RetryPolicyNever, nil)
	if errNil.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", errNil.Unwrap())
	}
}

// TestRetryError_RetryPolicy tests that RetryPolicy() returns the cached policy.
func TestRetryError_RetryPolicy(t *testing.T) {
	policies := []retrier.RetryPolicy{
		retrier.RetryPolicyAuto,
		retrier.RetryPolicyManual,
		retrier.RetryPolicyNever,
	}

	for _, policy := range policies {
		t.Run("policy", func(t *testing.T) {
			err := retrier.NewRetryError(retrier.ErrExhaustedAttempts, "test", policy, nil)
			if err.RetryPolicy() != policy {
				t.Errorf("RetryPolicy() = %v, want %v", err.RetryPolicy(), policy)
			}
		})
	}
}

// TestRetryError_Is tests the Is() method for errors.Is support.
func TestRetryError_Is(t *testing.T) {
	err := retrier.NewRetryError(retrier.ErrExhaustedAttempts, "test", retrier.RetryPolicyManual, nil)

	// Should match RetryError
	if !err.Is(&retrier.RetryError{}) {
		t.Error("Is() should return true for RetryError target")
	}

	// Should not match other error types
	if err.Is(errors.New("other error")) {
		t.Error("Is() should return false for non-RetryError target")
	}

	// Should not match nil
	if err.Is(nil) {
		t.Error("Is() should return false for nil target")
	}
}

// TestRetryError_ImplementsRetryableError verifies that RetryError implements
// the RetryableError interface.
func TestRetryError_ImplementsRetryableError(t *testing.T) {
	err := retrier.NewRetryError(retrier.ErrZeroAttempt, "test", retrier.RetryPolicyAuto, nil)

	// Verify all interface methods exist and return valid values
	var _ retrier.RetryableError = err

	// Verify basic interface contract
	if err.Error() == "" {
		t.Error("Error() should not be empty")
	}
	if err.RetryPolicy() < 0 {
		t.Error("RetryPolicy() should return valid policy")
	}
}

// TestRetryError_Causes tests both RetryErrorCause constants.
func TestRetryError_Causes(t *testing.T) {
	causes := []retrier.RetryErrorCause{
		retrier.ErrZeroAttempt,
		retrier.ErrExhaustedAttempts,
	}

	for _, cause := range causes {
		t.Run(string(cause), func(t *testing.T) {
			err := retrier.NewRetryError(cause, "test", retrier.RetryPolicyAuto, nil)
			if err.Cause != cause {
				t.Errorf("Cause = %v, want %v", err.Cause, cause)
			}
		})
	}
}

// TestRetryError_ErrorChain tests errors.Is and errors.As support.
func TestRetryError_ErrorChain(t *testing.T) {
	innerErr := errors.New("original error")
	err := retrier.NewRetryError(retrier.ErrExhaustedAttempts, "max retries", retrier.RetryPolicyManual, innerErr)

	// Test errors.Is
	if !errors.Is(err, innerErr) {
		t.Error("errors.Is should find wrapped error")
	}

	// Test errors.As
	var retryErr *retrier.RetryError
	if !errors.As(err, &retryErr) {
		t.Error("errors.As should find RetryError")
	}
}

// containsString is a helper to check if a string contains a substring.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
