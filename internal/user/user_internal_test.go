package user

// Tests for unexported helpers; public-API tests are in user_test.go.

import "testing"

func TestFirstNameParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, fullName, username, want string
	}{
		{"first and last", "Jane Doe", "jdoe", "Jane"},
		{"single name", "Jane", "jdoe", "Jane"},
		{"extra whitespace", "  Jane   Mary Doe ", "jdoe", "Jane"},
		{"empty falls back to username", "", "jdoe", "jdoe"},
		{"blank falls back to username", "   ", "jdoe", "jdoe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := firstName(tt.fullName, tt.username); got != tt.want {
				t.Errorf("firstName(%q, %q) = %q, want %q", tt.fullName, tt.username, got, tt.want)
			}
		})
	}
}
