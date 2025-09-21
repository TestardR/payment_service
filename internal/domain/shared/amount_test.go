package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAmount(t *testing.T) {
	tests := []struct {
		name        string
		input       float64
		expectError bool
		expected    float64
	}{
		{
			name:        "valid positive amount",
			input:       100.50,
			expectError: false,
			expected:    100.50,
		},
		{
			name:        "valid zero amount",
			input:       0.0,
			expectError: false,
			expected:    0.0,
		},
		{
			name:        "valid small amount",
			input:       0.01,
			expectError: false,
			expected:    0.01,
		},
		{
			name:        "valid large amount",
			input:       999999.99,
			expectError: false,
			expected:    999999.99,
		},
		{
			name:        "invalid negative amount",
			input:       -10.50,
			expectError: true,
		},
		{
			name:        "invalid negative small amount",
			input:       -0.01,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := NewAmount(tt.input)

			if tt.expectError {
				assert.Error(t, err, "expected error for input %f", tt.input)
				assert.Equal(t, ErrInvalidAmount, err, "expected ErrInvalidAmount")
			} else {
				assert.NoError(t, err, "unexpected error for input %f", tt.input)
				assert.Equal(t, tt.expected, amount.Value(), "expected %f, got %f", tt.expected, amount.Value())
			}
		})
	}
}

func TestAmount_IsZero(t *testing.T) {
	zeroAmount, _ := NewAmount(0.0)
	nonZeroAmount, _ := NewAmount(10.50)

	assert.True(t, zeroAmount.IsZero(), "expected zero amount to return true for IsZero()")
	assert.False(t, nonZeroAmount.IsZero(), "expected non-zero amount to return false for IsZero()")
}

func TestAmount_Add(t *testing.T) {
	amount1, _ := NewAmount(10.50)
	amount2, _ := NewAmount(5.25)
	
	result := amount1.Add(amount2)
	expected := 15.75

	assert.Equal(t, expected, result.Value(), "expected %f, got %f", expected, result.Value())
}

func TestAmount_Subtract(t *testing.T) {
	tests := []struct {
		name        string
		amount1     float64
		amount2     float64
		expectError bool
		expected    float64
	}{
		{
			name:        "valid subtraction",
			amount1:     10.50,
			amount2:     5.25,
			expectError: false,
			expected:    5.25,
		},
		{
			name:        "subtraction resulting in zero",
			amount1:     10.00,
			amount2:     10.00,
			expectError: false,
			expected:    0.00,
		},
		{
			name:        "invalid subtraction (negative result)",
			amount1:     5.00,
			amount2:     10.00,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount1, _ := NewAmount(tt.amount1)
			amount2, _ := NewAmount(tt.amount2)

			result, err := amount1.Subtract(amount2)

			if tt.expectError {
				assert.Error(t, err, "expected error for %f - %f", tt.amount1, tt.amount2)
			} else {
				assert.NoError(t, err, "unexpected error for %f - %f", tt.amount1, tt.amount2)
				assert.Equal(t, tt.expected, result.Value(), "expected %f, got %f", tt.expected, result.Value())
			}
		})
	}
}

func TestAmount_Equals(t *testing.T) {
	amount1, _ := NewAmount(10.50)
	amount2, _ := NewAmount(10.50)
	amount3, _ := NewAmount(15.75)

	assert.True(t, amount1.Equals(amount2), "expected equal amounts to return true for Equals()")
	assert.False(t, amount1.Equals(amount3), "expected different amounts to return false for Equals()")
}
