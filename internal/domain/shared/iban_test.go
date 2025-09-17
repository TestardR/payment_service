package shared

import (
	"testing"
)

func TestNewIBAN(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    string
	}{
		{
			name:        "valid IBAN with spaces",
			input:       "GB82 WEST 1234 5698 7654 32",
			expectError: false,
			expected:    "GB82WEST12345698765432",
		},
		{
			name:        "valid IBAN without spaces",
			input:       "FR1420041010050500013M02606",
			expectError: false,
			expected:    "FR1420041010050500013M02606",
		},
		{
			name:        "valid IBAN lowercase",
			input:       "de89370400440532013000",
			expectError: false,
			expected:    "DE89370400440532013000",
		},
		{
			name:        "invalid IBAN too short",
			input:       "GB82",
			expectError: true,
		},
		{
			name:        "invalid IBAN wrong format",
			input:       "1234567890123456789012",
			expectError: true,
		},
		{
			name:        "invalid IBAN with special characters",
			input:       "GB82-WEST-1234-5698-7654-32",
			expectError: true,
		},
		{
			name:        "empty IBAN",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iban, err := NewIBAN(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input %q, but got none", tt.input)
				}
				if err != ErrInvalidIBAN {
					t.Errorf("expected ErrInvalidIBAN, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if iban.Value() != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, iban.Value())
				}
			}
		})
	}
}

func TestIBAN_String(t *testing.T) {
	iban, _ := NewIBAN("GB82WEST12345698765432")
	expected := "GB82WEST12345698765432"

	if iban.String() != expected {
		t.Errorf("expected %q, got %q", expected, iban.String())
	}
}

func TestIBAN_Equals(t *testing.T) {
	iban1, _ := NewIBAN("GB82WEST12345698765432")
	iban2, _ := NewIBAN("gb82 west 1234 5698 7654 32")
	iban3, _ := NewIBAN("FR1420041010050500013M02606")

	if !iban1.Equals(iban2) {
		t.Error("expected IBANs to be equal (normalized)")
	}

	if iban1.Equals(iban3) {
		t.Error("expected IBANs to be different")
	}
}
