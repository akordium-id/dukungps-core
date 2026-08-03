package user

import (
	"net/mail"
	"strings"
	"time"
)

const (
	RoleAdmin  = "admin"
	RoleUser   = "user"
	RoleViewer = "viewer"
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
		IsValidEmail(u.Email) &&
		len(u.Password) >= 8
}

func (u *User) NormalizeEmail() {
	u.Email = NormalizeEmail(u.Email)
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func IsValidEmail(email string) bool {
	email = NormalizeEmail(email)

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	if addr.Address != email {
		return false
	}

	local, domain, ok := strings.Cut(addr.Address, "@")
	if !ok || local == "" || domain == "" {
		return false
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}

	return true
}
