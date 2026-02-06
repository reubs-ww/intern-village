// Package repository provides cryptographic utilities for token encryption.
package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid.
	ErrInvalidKey = errors.New("encryption key must be 32 bytes for AES-256")

	// ErrDecryptionFailed is returned when decryption fails.
	ErrDecryptionFailed = errors.New("decryption failed: ciphertext too short or invalid")

	// ErrEmptyPlaintext is returned when attempting to encrypt empty data.
	ErrEmptyPlaintext = errors.New("plaintext cannot be empty")
)

// Crypto provides AES-256-GCM encryption and decryption for tokens.
type Crypto struct {
	key []byte
}

// NewCrypto creates a new Crypto instance with the given key.
// The key must be exactly 32 bytes (256 bits) for AES-256.
func NewCrypto(key []byte) (*Crypto, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	return &Crypto{key: key}, nil
}

// NewCryptoFromString creates a new Crypto instance from a base64-encoded key string.
func NewCryptoFromString(keyStr string) (*Crypto, error) {
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %w", err)
	}
	return NewCrypto(key)
}

// EncryptToken encrypts a plaintext token using AES-256-GCM.
// Returns the base64-encoded ciphertext (nonce prepended to ciphertext).
func (c *Crypto) EncryptToken(plaintext string) (string, error) {
	if plaintext == "" {
		return "", ErrEmptyPlaintext
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce with the standard size
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded result
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptToken decrypts a base64-encoded ciphertext using AES-256-GCM.
// Expects the nonce to be prepended to the ciphertext.
func (c *Crypto) DecryptToken(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", ErrDecryptionFailed
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", ErrDecryptionFailed
	}

	// Extract nonce and ciphertext
	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// GenerateKey generates a new random 32-byte key for AES-256.
// Returns the key as a base64-encoded string.
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
