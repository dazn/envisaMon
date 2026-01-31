package tpi

import (
	"errors"
	"testing"
)

func TestAuthError_Error(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "basic error message",
			message: "incorrect password",
			want:    "authentication failed: incorrect password",
		},
		{
			name:    "unexpected prompt",
			message: "unexpected prompt: Welcome",
			want:    "authentication failed: unexpected prompt: Welcome",
		},
		{
			name:    "empty message",
			message: "",
			want:    "authentication failed: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &AuthError{Message: tt.message}
			if got := e.Error(); got != tt.want {
				t.Errorf("AuthError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConnectionError_Error(t *testing.T) {
	tests := []struct {
		name    string
		message string
		err     error
		want    string
	}{
		{
			name:    "with wrapped error",
			message: "failed to dial",
			err:     errors.New("connection refused"),
			want:    "connection error: failed to dial: connection refused",
		},
		{
			name:    "without wrapped error",
			message: "connection closed",
			err:     nil,
			want:    "connection error: connection closed",
		},
		{
			name:    "read error with wrapped",
			message: "read error",
			err:     errors.New("EOF"),
			want:    "connection error: read error: EOF",
		},
		{
			name:    "empty message with error",
			message: "",
			err:     errors.New("network down"),
			want:    "connection error: : network down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ConnectionError{
				Message: tt.message,
				Err:     tt.err,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("ConnectionError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTimeoutError_Error(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		err       error
		want      string
	}{
		{
			name:      "read timeout",
			operation: "read login prompt",
			err:       errors.New("i/o timeout"),
			want:      "timeout during read login prompt: i/o timeout",
		},
		{
			name:      "write timeout",
			operation: "send password",
			err:       errors.New("deadline exceeded"),
			want:      "timeout during send password: deadline exceeded",
		},
		{
			name:      "set deadline timeout",
			operation: "set read deadline",
			err:       errors.New("invalid deadline"),
			want:      "timeout during set read deadline: invalid deadline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TimeoutError{
				Operation: tt.operation,
				Err:       tt.err,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("TimeoutError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
