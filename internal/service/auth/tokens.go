package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tinkerborg/open-pulumi-service/internal/model"
)

// TODO token expiry

type UserClaims struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	jwt.RegisteredClaims
}

// TODO - options array for token type, expiry etc
func (s *Service) CreateToken(id string, tokenType string) (string, error) {
	claims := &UserClaims{
		ID:   id,
		Type: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			// TODO
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	value, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.key)
	if err != nil {
		return "", err
	}

	token := &model.AuthToken{
		UserID: id,
		Value:  value,
	}

	if err := s.store.Create(token); err != nil {
		return "", err
	}

	return value, nil
}

func (s *Service) GetUserClaims(token string) (*UserClaims, error) {
	claims := UserClaims{}

	parsed, err := jwt.ParseWithClaims(token, &claims, s.publicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %s", err)
	}

	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}

	return &claims, nil
}

func (s *Service) publicKey(token *jwt.Token) (interface{}, error) {
	return s.key.Public(), nil
}
