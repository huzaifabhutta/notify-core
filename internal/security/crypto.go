package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// HashAPIKey hashes an API key using bcrypt
func HashAPIKey(apiKey string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hashedBytes), nil
}

// CompareAPIKey compares a plain-text API key with its bcrypt hash
func CompareAPIKey(hashedKey, plainKey string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedKey), []byte(plainKey))
}

// EncryptString encrypts a string using AES-256-GCM
func EncryptString(plaintext, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("encryption key is required")
	}

	// Ensure key is 32 bytes for AES-256
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		return "", fmt.Errorf("encryption key must be at least 32 bytes")
	}
	if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a string using AES-256-GCM
func DecryptString(ciphertext, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("encryption key is required")
	}

	// Ensure key is 32 bytes for AES-256
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		return "", fmt.Errorf("encryption key must be at least 32 bytes")
	}
	if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptIfNotEmpty encrypts a string if it's not empty
func EncryptIfNotEmpty(plaintext, key string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	return EncryptString(plaintext, key)
}

// DecryptIfNotEmpty decrypts a string if it's not empty
func DecryptIfNotEmpty(ciphertext, key string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return DecryptString(ciphertext, key)
}
