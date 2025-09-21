package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
				assert.Error(t, err, "expected error for input %q", tt.input)
				assert.Equal(t, ErrInvalidIdempotencyKey, err, "expected ErrInvalidIdempotencyKey")
			} else {
				assert.NoError(t, err, "unexpected error for input %q", tt.input)
				assert.Equal(t, tt.expected, key.Value(), "expected %q, got %q", tt.expected, key.Value())
			}
		})
	}
}

func TestIdempotencyKey_String(t *testing.T) {
	key, _ := NewIdempotencyKey("abc123XYZ0")
	expected := "abc123XYZ0"

	assert.Equal(t, expected, key.String(), "expected %q, got %q", expected, key.String())
}

func TestIdempotencyKey_Equals(t *testing.T) {
	key1, _ := NewIdempotencyKey("abc123XYZ0")
	key2, _ := NewIdempotencyKey("abc123XYZ0")
	key3, _ := NewIdempotencyKey("xyz789ABC1")

	assert.True(t, key1.Equals(key2), "expected equal keys to return true for Equals()")
	assert.False(t, key1.Equals(key3), "expected different keys to return false for Equals()")
}
