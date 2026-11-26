// Package user exposes information about the logged-in OS user.
package user

import (
	osuser "os/user"
	"strings"
)

// User is bound to the frontend; its exported methods are callable from JS.
type User struct{}

// New returns a User.
func New() *User {
	return &User{}
}

// FirstName returns the first word of the user's full name (e.g. "Jane" for
// "Jane Doe"), falling back to the login name (e.g. "jdoe") when no full name
// is set.
func (u *User) FirstName() (string, error) {
	acct, err := osuser.Current()
	if err != nil {
		return "", err
	}
	return firstName(acct.Name, acct.Username), nil
}

// firstName returns the first word of fullName, or username if fullName is blank.
func firstName(fullName, username string) string {
	if fields := strings.Fields(fullName); len(fields) > 0 {
		return fields[0]
	}
	return username
}
