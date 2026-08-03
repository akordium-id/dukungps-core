package user

import (
	"regexp"
	"strings"
	"time"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

const (
	RoleAdmin    = "admin"
	RoleUser     = "user"
	RoleViewer   = "viewer"
)

type User struct {
	ID        int64
	Name      string
	Email     string
	Password  string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) IsValid() bool {
	return strings.TrimSpace(u.Name) != "" &&
		emailRegex.MatchString(u.Email) &&
		len(u.Password) >= 8
}
