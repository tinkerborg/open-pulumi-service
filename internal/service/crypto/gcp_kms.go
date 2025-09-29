package crypto

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	wrapping "github.com/hashicorp/go-kms-wrapping"
	"github.com/hashicorp/go-kms-wrapping/wrappers/gcpckms"
)

type GoogleKmsCryptoService struct {
	wrapper *gcpckms.Wrapper
}

var _ Service = GoogleKmsCryptoService{}

func NewGoogleKmsCryptoService(keyID string) *GoogleKmsCryptoService {
	wrapper := gcpckms.NewWrapper(&wrapping.WrapperOptions{})

	config, err := parseKeyID(keyID)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := wrapper.SetConfig(config); err != nil {
		log.Fatalf("can't configure GCP KMS: %s", err)
	}

	return &GoogleKmsCryptoService{wrapper}
}

// Encrypt implements CryptoService.
func (g GoogleKmsCryptoService) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	blob, err := g.wrapper.Encrypt(ctx, plaintext, nil)
	if err != nil {
		return nil, err
	}

	ciphertext, err := json.Marshal(blob)
	if err != nil {
		return nil, err
	}

	b := &wrapping.EncryptedBlobInfo{}
	if err := json.Unmarshal(ciphertext, blob); err != nil {
		return nil, err
	}
	_ = b

	return ciphertext, nil
}

// Decrypt implements CryptoService.
func (g GoogleKmsCryptoService) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	blob := &wrapping.EncryptedBlobInfo{}
	if err := json.Unmarshal(ciphertext, blob); err != nil {
		return nil, err
	}

	plaintext, err := g.wrapper.Decrypt(ctx, blob, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func parseKeyID(keyID string) (map[string]string, error) {
	parts := strings.Split(keyID, "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "keyRings" || parts[6] != "cryptoKeys" {
		return nil, errors.New("invalid KMS key ID, expected: projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}")
	}
	return map[string]string{
		"project":    parts[1],
		"region":     parts[3],
		"key_ring":   parts[5],
		"crypto_key": parts[7],
	}, nil
}
