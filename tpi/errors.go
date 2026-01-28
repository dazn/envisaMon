package tpi

import "fmt"

// AuthError represents authentication failures (fatal, no retry)
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication failed: %s", e.Message)
}

// ConnectionError represents network connection issues (retry with backoff)
type ConnectionError struct {
	Message string
	Err     error
}

func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("connection error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("connection error: %s", e.Message)
}

// TimeoutError represents read/write timeouts (retry with backoff)
type TimeoutError struct {
	Operation string
	Err       error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout during %s: %v", e.Operation, e.Err)
}
