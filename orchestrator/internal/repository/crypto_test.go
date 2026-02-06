package repository

import (
	"encoding/base64"
	"testing"
)

func TestNewCrypto(t *testing.T) {
	t.Run("valid 32-byte key", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}

		crypto, err := NewCrypto(key)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if crypto == nil {
			t.Fatal("expected crypto to not be nil")
		}
	})

	t.Run("invalid key length", func(t *testing.T) {
		testCases := []struct {
			name   string
			keyLen int
		}{
			{"too short - 16 bytes", 16},
			{"too short - 24 bytes", 24},
			{"too long - 48 bytes", 48},
			{"empty", 0},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				key := make([]byte, tc.keyLen)
				_, err := NewCrypto(key)
				if err != ErrInvalidKey {
					t.Errorf("expected ErrInvalidKey, got %v", err)
				}
			})
		}
	})
}

func TestNewCryptoFromString(t *testing.T) {
	t.Run("valid base64 key", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}
		keyStr := base64.StdEncoding.EncodeToString(key)

		crypto, err := NewCryptoFromString(keyStr)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if crypto == nil {
			t.Fatal("expected crypto to not be nil")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := NewCryptoFromString("not-valid-base64!!!")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	crypto, err := NewCrypto(key)
	if err != nil {
		t.Fatalf("failed to create crypto: %v", err)
	}

	t.Run("round trip", func(t *testing.T) {
		testCases := []string{
			"simple-token",
			"ghp_abcdefghijklmnopqrstuvwxyz123456",
			"a",
			"with spaces and special chars: !@#$%^&*()",
			"unicode: 你好世界 🚀",
		}

		for _, plaintext := range testCases {
			t.Run(plaintext[:min(len(plaintext), 20)], func(t *testing.T) {
				ciphertext, err := crypto.EncryptToken(plaintext)
				if err != nil {
					t.Fatalf("encryption failed: %v", err)
				}

				if ciphertext == plaintext {
					t.Error("ciphertext should not equal plaintext")
				}

				decrypted, err := crypto.DecryptToken(ciphertext)
				if err != nil {
					t.Fatalf("decryption failed: %v", err)
				}

				if decrypted != plaintext {
					t.Errorf("expected %q, got %q", plaintext, decrypted)
				}
			})
		}
	})

	t.Run("empty plaintext", func(t *testing.T) {
		_, err := crypto.EncryptToken("")
		if err != ErrEmptyPlaintext {
			t.Errorf("expected ErrEmptyPlaintext, got %v", err)
		}
	})

	t.Run("different encryptions produce different ciphertexts", func(t *testing.T) {
		plaintext := "test-token"
		ct1, _ := crypto.EncryptToken(plaintext)
		ct2, _ := crypto.EncryptToken(plaintext)

		if ct1 == ct2 {
			t.Error("same plaintext should produce different ciphertexts due to random nonce")
		}

		// Both should decrypt to the same value
		pt1, _ := crypto.DecryptToken(ct1)
		pt2, _ := crypto.DecryptToken(ct2)

		if pt1 != pt2 {
			t.Errorf("both ciphertexts should decrypt to same plaintext: %q vs %q", pt1, pt2)
		}
	})
}

func TestDecryptToken_InvalidInput(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	crypto, _ := NewCrypto(key)

	t.Run("empty ciphertext", func(t *testing.T) {
		_, err := crypto.DecryptToken("")
		if err != ErrDecryptionFailed {
			t.Errorf("expected ErrDecryptionFailed, got %v", err)
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := crypto.DecryptToken("not-valid-base64!!!")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("too short ciphertext", func(t *testing.T) {
		// Valid base64 but too short (less than nonce size)
		shortData := base64.StdEncoding.EncodeToString([]byte("short"))
		_, err := crypto.DecryptToken(shortData)
		if err != ErrDecryptionFailed {
			t.Errorf("expected ErrDecryptionFailed, got %v", err)
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		// Encrypt something valid
		ct, _ := crypto.EncryptToken("test-token")

		// Decode, tamper, re-encode
		data, _ := base64.StdEncoding.DecodeString(ct)
		data[len(data)-1] ^= 0xFF // Flip bits in last byte
		tampered := base64.StdEncoding.EncodeToString(data)

		_, err := crypto.DecryptToken(tampered)
		if err != ErrDecryptionFailed {
			t.Errorf("expected ErrDecryptionFailed for tampered data, got %v", err)
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		// Encrypt with original key
		ct, _ := crypto.EncryptToken("test-token")

		// Try to decrypt with different key
		otherKey := make([]byte, 32)
		for i := range otherKey {
			otherKey[i] = byte(i + 100)
		}
		otherCrypto, _ := NewCrypto(otherKey)

		_, err := otherCrypto.DecryptToken(ct)
		if err != ErrDecryptionFailed {
			t.Errorf("expected ErrDecryptionFailed for wrong key, got %v", err)
		}
	})
}

func TestGenerateKey(t *testing.T) {
	t.Run("generates valid key", func(t *testing.T) {
		keyStr, err := GenerateKey()
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}

		// Should be valid base64
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			t.Fatalf("generated key is not valid base64: %v", err)
		}

		// Should be 32 bytes
		if len(key) != 32 {
			t.Errorf("expected 32-byte key, got %d bytes", len(key))
		}

		// Should be usable
		crypto, err := NewCryptoFromString(keyStr)
		if err != nil {
			t.Fatalf("generated key is not usable: %v", err)
		}

		// Should work for encrypt/decrypt
		ct, err := crypto.EncryptToken("test")
		if err != nil {
			t.Fatalf("encryption with generated key failed: %v", err)
		}

		pt, err := crypto.DecryptToken(ct)
		if err != nil {
			t.Fatalf("decryption with generated key failed: %v", err)
		}

		if pt != "test" {
			t.Errorf("round trip failed: expected 'test', got %q", pt)
		}
	})

	t.Run("generates different keys", func(t *testing.T) {
		key1, _ := GenerateKey()
		key2, _ := GenerateKey()

		if key1 == key2 {
			t.Error("generated keys should be different")
		}
	})
}
