package user

import (
	"context"
	"errors"
	"fmt"
	"pulse/internal/auth"
	"pulse/internal/shared"

	"uuid"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", errors.New("email and password are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	u := &User{
		Model:        shared.Model{ID: uuid.New()},
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.repo.CreateUser(ctx, u); err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}

	return auth.CreateToken(u.ID)
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", errors.New("invalid email or password")
		}
		return "", fmt.Errorf("get user by email: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid email or password")
	}

	return auth.CreateToken(u.ID)
}
