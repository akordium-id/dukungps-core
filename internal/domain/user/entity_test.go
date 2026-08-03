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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.user.IsValid())
		})
	}
}
