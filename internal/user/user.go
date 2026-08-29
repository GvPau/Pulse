package user

import "pulse/internal/shared"

type User struct {
	shared.Model
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}
