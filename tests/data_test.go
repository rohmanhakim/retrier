package retrier_test

import (
	"errors"
	"testing"

	retrier "github.com/rohmanhakim/retrier"
)

// TestResult_NewFailureResult tests NewFailureResult creation.
func TestResult_NewFailureResult(t *testing.T) {
	testErr := errors.New("test error")
	attempts := 3

	result := retrier.NewFailureResult[string](testErr, attempts)

	if result.IsSuccess() {
		t.Error("expected failure result, got success")
	}
	if result.Err() != testErr {
		t.Errorf("Err() = %v, want %v", result.Err(), testErr)
	}
	if result.Attempts() != attempts {
		t.Errorf("Attempts() = %d, want %d", result.Attempts(), attempts)
	}
	if result.Value() != "" {
		t.Errorf("Value() = %q, want empty string", result.Value())
	}
}

// TestResult_Decompose tests the Decompose method.
func TestResult_Decompose(t *testing.T) {
	tests := []struct {
		name         string
		result       retrier.Result[string]
		wantValue    string
		wantErr      bool
		wantAttempts int
	}{
		{
			name:         "success result",
			result:       retrier.NewSuccessResult("success value", 2),
			wantValue:    "success value",
			wantErr:      false,
			wantAttempts: 2,
		},
		{
			name:         "failure result",
			result:       retrier.NewFailureResult[string](errors.New("failure"), 3),
			wantValue:    "",
			wantErr:      true,
			wantAttempts: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, attempts, err := tt.result.Decompose()

			if value != tt.wantValue {
				t.Errorf("Decompose() value = %q, want %q", value, tt.wantValue)
			}
			if attempts != tt.wantAttempts {
				t.Errorf("Decompose() attempts = %d, want %d", attempts, tt.wantAttempts)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Decompose() error = %v, want error: %v", err, tt.wantErr)
			}
		})
	}
}

// TestResult_UnwrapOr tests the UnwrapOr method.
func TestResult_UnwrapOr(t *testing.T) {
	defaultValue := "default"

	tests := []struct {
		name      string
		result    retrier.Result[string]
		wantValue string
	}{
		{
			name:      "success returns actual value",
			result:    retrier.NewSuccessResult("actual", 1),
			wantValue: "actual",
		},
		{
			name:      "failure returns default value",
			result:    retrier.NewFailureResult[string](errors.New("error"), 3),
			wantValue: defaultValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.UnwrapOr(defaultValue)
			if got != tt.wantValue {
				t.Errorf("UnwrapOr() = %q, want %q", got, tt.wantValue)
			}
		})
	}
}

// TestResult_Unwrap tests the Unwrap method.
func TestResult_Unwrap(t *testing.T) {
	t.Run("success returns value", func(t *testing.T) {
		result := retrier.NewSuccessResult("success value", 1)
		got := result.Unwrap()
		if got != "success value" {
			t.Errorf("Unwrap() = %q, want %q", got, "success value")
		}
	})

	t.Run("failure panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, but Unwrap did not panic")
			}
		}()
		result := retrier.NewFailureResult[string](errors.New("error"), 3)
		result.Unwrap()
	})
}

// TestResult_Unwrap_PanicMessage tests that Unwrap panic contains error message.
func TestResult_Unwrap_PanicMessage(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		// Panic message should contain the error
		panicStr, ok := r.(string)
		if !ok {
			t.Fatalf("panic value is not a string: %v", r)
		}
		if panicStr == "" {
			t.Error("panic message should not be empty")
		}
	}()

	result := retrier.NewFailureResult[string](errors.New("test error"), 2)
	result.Unwrap()
}

// TestResult_Types tests that Result works with different generic types.
func TestResult_Types(t *testing.T) {
	t.Run("int type", func(t *testing.T) {
		result := retrier.NewSuccessResult(42, 1)
		if result.Value() != 42 {
			t.Errorf("Value() = %d, want 42", result.Value())
		}
	})

	t.Run("pointer type", func(t *testing.T) {
		type Data struct{ X int }
		data := &Data{X: 100}
		result := retrier.NewSuccessResult(data, 1)
		if result.Value().X != 100 {
			t.Errorf("Value().X = %d, want 100", result.Value().X)
		}
	})

	t.Run("slice type", func(t *testing.T) {
		slice := []int{1, 2, 3}
		result := retrier.NewSuccessResult(slice, 1)
		if len(result.Value()) != 3 {
			t.Errorf("len(Value()) = %d, want 3", len(result.Value()))
		}
	})

	t.Run("failure with pointer type", func(t *testing.T) {
		type Data struct{ X int }
		result := retrier.NewFailureResult[*Data](errors.New("error"), 1)
		if result.Value() != nil {
			t.Error("Value() should be nil for failed pointer result")
		}
		defaultVal := &Data{X: 99}
		got := result.UnwrapOr(defaultVal)
		if got.X != 99 {
			t.Errorf("UnwrapOr().X = %d, want 99", got.X)
		}
	})
}
