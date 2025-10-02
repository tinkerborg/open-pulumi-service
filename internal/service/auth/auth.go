package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"

	"github.com/tinkerborg/open-pulumi-service/internal/model"
	"github.com/tinkerborg/open-pulumi-service/internal/store"
)

type Service struct {
	store *store.Postgres
	key   *rsa.PrivateKey
}

func New(store *store.Postgres) (*Service, error) {
	store.RegisterModels(model.AuthToken{}, model.RSAKey{})

	key, err := getRSAKey(store)
	if err != nil {
		return nil, err
	}

	return &Service{store, key}, nil
}

const keyName = "auth-root"

func getRSAKey(s *store.Postgres) (*rsa.PrivateKey, error) {
	rsaKey := &model.RSAKey{Name: keyName}

	err := s.Read(rsaKey)

	if errors.Is(err, store.ErrNotFound) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}

		rsaKey.Value = privateKey

		if err := s.Create(rsaKey); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	rsaKey.Value.Precompute()

	return rsaKey.Value, nil
}
