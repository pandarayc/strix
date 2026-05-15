package llm

import (
	"context"
	"fmt"
	"math"
	"time"
)

// APIError represents an error from the LLM API.
type APIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status=%d, type=%s): %s", e.Status, e.Type, e.Message)
}

// IsRetryable returns true if the error is likely transient.
func (e *APIError) IsRetryable() bool {
	switch e.Status {
	case 429, 529, 503, 502, 504:
		return true
	case 500:
		return true
	default:
		return false
	}
}

// RetryWithBackoff executes fn with exponential backoff on retryable errors.
func RetryWithBackoff[T any](
	ctx context.Context,
	maxRetries int,
	fn func() (T, error),
	isRetryable func(error) bool,
) (T, error) {
	var zero T

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		if !isRetryable(err) || attempt == maxRetries {
			return zero, err
		}

		// Calculate backoff: 1s, 2s, 4s, 8s, ...
		delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	return zero, fmt.Errorf("max retries exceeded")
}
