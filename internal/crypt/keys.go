package crypt

import (
	"crypto/hpke"
	"fmt"
	"os"
)

func GenerateAndSaveKeyPair(privatePath, publicPath string) error {
	kem := KEM()

	privateKey, err := kem.GenerateKey()
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	privateBytes, err := privateKey.Bytes()
	if err != nil {
		return fmt.Errorf("serialize private key: %w", err)
	}

	publicBytes := privateKey.PublicKey().Bytes()

	if err := os.WriteFile(privatePath, privateBytes, 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	if err := os.WriteFile(publicPath, publicBytes, 0644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}
	return nil
}

func LoadPrivateKey(path string) (hpke.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %q: %w", path, err)
	}

	key, err := KEM().NewPrivateKey(raw)
	clear(raw)
	if err != nil {
		return nil, fmt.Errorf("parse private key %q: %w", path, err)
	}
	return key, nil
}

func LoadPublicKey(path string) (hpke.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key %q: %w", path, err)
	}

	key, err := KEM().NewPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("parse public key %q: %w", path, err)
	}
	return key, nil
}
