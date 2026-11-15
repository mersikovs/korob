package crypt

import (
	"crypto/hpke"
	"fmt"
)

func Encrypt(publicKey hpke.PublicKey, plaintext []byte) ([]byte, error) {
	ciphertext, err := hpke.Seal(
		publicKey,
		KDF(),
		AEAD(),
		Info,
		plaintext,
	)
	if err != nil {
		return nil, fmt.Errorf("hpke seal: %w", err)
	}
	return ciphertext, nil
}

func Decrypt(privateKey hpke.PrivateKey, ciphertext []byte) ([]byte, error) {
	plaintext, err := hpke.Open(
		privateKey,
		KDF(),
		AEAD(),
		Info,
		ciphertext,
	)
	if err != nil {
		return nil, fmt.Errorf("hpke open: %w", err)
	}
	return plaintext, nil
}
