package utils

import "testing"

func TestMaskPhoneNumber(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{
			name:     "Standard phone number",
			phone:    "+1234567890",
			expected: "+12****7890",
		},
		{
			name:     "Short phone number",
			phone:    "+12345",
			expected: "****",
		},
		{
			name:     "Exactly 6 characters",
			phone:    "123456",
			expected: "****",
		},
		{
			name:     "7 characters (first valid to mask)",
			phone:    "+123456",
			expected: "+12****3456",
		},
		{
			name:     "Very long phone number",
			phone:    "+12345678901234567890",
			expected: "+12****7890",
		},
		{
			name:     "Empty string",
			phone:    "",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskPhoneNumber(tt.phone)
			if result != tt.expected {
				t.Errorf("MaskPhoneNumber(%q) = %q, want %q", tt.phone, result, tt.expected)
			}
		})
	}
}
