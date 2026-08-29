package auth

import (
	"errors"
	"fmt"
	"os"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func secret() ([]byte, error) {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return nil, errors.New("JWT_SECRET environment variable is not set")
	}
	return []byte(s), nil
}

func CreateToken(userID uuid.UUID) (string, error) {
	claims := Claims{
		UserID:           userID,
		RegisteredClaims: jwt.RegisteredClaims{
			// ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token expires in 24 hours
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secretKey, err := secret()

	if err != nil {
		return "", err
	}

	return token.SignedString(secretKey)
}

func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		secretKey, err := secret()
		if err != nil {
			return nil, err
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
