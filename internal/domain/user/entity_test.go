package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_IsValid(t *testing.T) {
	tests := []struct {
		name string
		user User
		want bool
	}{
		{
			name: "valid user",
			user: User{Name: "Faiq", Email: "faiq@akordium.id", Password: "secret123"},
			want: true,
		},
		{
			name: "invalid email",
			user: User{Name: "Faiq", Email: "bukan-email", Password: "secret123"},
			want: false,
		},
		{
			name: "password too short",
			user: User{Name: "Faiq", Email: "faiq@akordium.id", Password: "short"},
			want: false,
		},
		{
			name: "missing name",
			user: User{Name: "", Email: "faiq@akordium.id", Password: "secret123"},
			want: false,
		},
		{
			name: "email with consecutive dots in domain",
			user: User{Name: "Faiq", Email: "a@..com", Password: "secret123"},
			want: false,
		},
		{
			name: "email with leading dot in domain",
			user: User{Name: "Faiq", Email: "a@.b.com", Password: "secret123"},
			want: false,
		},
		{
			name: "email with hyphen-bounded label",
			user: User{Name: "Faiq", Email: "a@b-.com", Password: "secret123"},
			want: false,
		},
		{
			name: "email without tld",
			user: User{Name: "Faiq", Email: "a@b", Password: "secret123"},
			want: false,
		},
		{
			name: "email with display name",
			user: User{Name: "Faiq", Email: "Faiq <faiq@akordium.id>", Password: "secret123"},
			want: false,
		},
		{
			name: "uppercase email accepted",
			user: User{Name: "Faiq", Email: "FAIQ@Akordium.Id", Password: "secret123"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.IsValid())
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	assert.Equal(t, "faiq@akordium.id", NormalizeEmail("  FAIQ@Akordium.Id  "))
}

func TestUser_NormalizeEmail(t *testing.T) {
	u := User{Name: "Faiq", Email: "  FAIQ@Akordium.Id  ", Password: "secret123"}
	u.NormalizeEmail()
	assert.Equal(t, "faiq@akordium.id", u.Email)
}

func TestRoleConstants(t *testing.T) {
	assert.Equal(t, "admin", RoleAdmin)
	assert.Equal(t, "user", RoleUser)
	assert.Equal(t, "viewer", RoleViewer)
}
