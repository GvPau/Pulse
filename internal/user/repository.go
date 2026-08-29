package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

var ErrNotFound = errors.New("user not found")

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	_, err := r.pool.Exec(ctx,
		"INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)",
		user.ID,
		user.Email,
		user.PasswordHash,
	)

	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}
