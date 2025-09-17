package shared

import (
	"testing"
)

func TestNewIdempotencyKey(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    string
	}{
		{
			name:        "valid alphanumeric key",
			input:       "abc123XYZ0",
			expectError: false,
			expected:    "abc123XYZ0",
		},
		{
			name:        "valid all letters",
			input:       "abcdefghij",
			expectError: false,
			expected:    "abcdefghij",
		},
		{
			name:        "valid all numbers",
			input:       "1234567890",
			expectError: false,
			expected:    "1234567890",
		},
		{
			name:        "valid mixed case",
			input:       "AbC123XyZ0",
			expectError: false,
			expected:    "AbC123XyZ0",
		},
		{
			name:        "invalid too short",
			input:       "abc123",
			expectError: true,
		},
		{
			name:        "invalid too long",
			input:       "abc123XYZ01",
			expectError: true,
		},
		{
			name:        "invalid with special characters",
			input:       "abc123-XY0",
			expectError: true,
		},
		{
			name:        "invalid with spaces",
			input:       "abc123 XY0",
			expectError: true,
		},
		{
			name:        "invalid with underscore",
			input:       "abc123_XY0",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := NewIdempotencyKey(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input %q, but got none", tt.input)
				}
				if err != ErrInvalidIdempotencyKey {
					t.Errorf("expected ErrInvalidIdempotencyKey, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if key.Value() != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, key.Value())
				}
			}
		})
	}
}

func TestIdempotencyKey_String(t *testing.T) {
	key, _ := NewIdempotencyKey("abc123XYZ0")
	expected := "abc123XYZ0"

	if key.String() != expected {
		t.Errorf("expected %q, got %q", expected, key.String())
	}
}

func TestIdempotencyKey_Equals(t *testing.T) {
	key1, _ := NewIdempotencyKey("abc123XYZ0")
	key2, _ := NewIdempotencyKey("abc123XYZ0")
	key3, _ := NewIdempotencyKey("xyz789ABC1")

	if !key1.Equals(key2) {
		t.Error("expected equal keys to return true for Equals()")
	}

	if key1.Equals(key3) {
		t.Error("expected different keys to return false for Equals()")
	}
}
