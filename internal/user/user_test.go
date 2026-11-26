package user_test

import (
	"testing"

	"github.com/ha-li/Whip/internal/user"
)

func TestFirstName(t *testing.T) {
	t.Parallel()

	name, err := user.New().FirstName()
	if err != nil {
		t.Fatalf("FirstName() error = %v", err)
	}
	if name == "" {
		t.Fatal("FirstName() returned an empty name")
	}
}
