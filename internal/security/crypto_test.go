package security

import (
	"strings"
	"testing"
)

func TestHashAPIKey(t *testing.T) {
	apiKey := "test-api-key-123"

	hash, err := HashAPIKey(apiKey)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if hash == apiKey {
		t.Error("Hash should not equal plain text")
	}

	if len(hash) < 50 {
		t.Errorf("Hash too short: %d characters", len(hash))
	}

	// Hash should start with bcrypt prefix
	if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
		t.Error("Hash doesn't appear to be bcrypt format")
	}
}

func TestCompareAPIKey(t *testing.T) {
	plainKey := "test-api-key-123"

	hash, err := HashAPIKey(plainKey)
	if err != nil {
		t.Fatalf("Failed to hash: %v", err)
	}

	// Test correct key
	if err := CompareAPIKey(hash, plainKey); err != nil {
		t.Error("Expected no error for correct key")
	}

	// Test incorrect key
	if err := CompareAPIKey(hash, "wrong-key"); err == nil {
		t.Error("Expected error for incorrect key")
	}

	// Test empty key
	if err := CompareAPIKey(hash, ""); err == nil {
		t.Error("Expected error for empty key")
	}
}

func TestEncryptDecryptString(t *testing.T) {
	key := "my-32-char-encryption-key-123456"
	plaintext := "sensitive-data-12345"

	// Test encryption
	encrypted, err := EncryptString(plaintext, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if encrypted == plaintext {
		t.Error("Encrypted text should not equal plaintext")
	}

	// Test decryption
	decrypted, err := DecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected '%s', got '%s'", plaintext, decrypted)
	}
}

func TestEncryptString_ShortKey(t *testing.T) {
	key := "short-key"
	plaintext := "test"

	_, err := EncryptString(plaintext, key)
	if err == nil {
		t.Error("Expected error for short key")
	}
}

func TestDecryptString_InvalidCiphertext(t *testing.T) {
	key := "my-32-char-encryption-key-123456"

	_, err := DecryptString("invalid-ciphertext", key)
	if err == nil {
		t.Error("Expected error for invalid ciphertext")
	}
}

func TestDecryptString_WrongKey(t *testing.T) {
	key1 := "my-32-char-encryption-key-123456"
	key2 := "different-encryption-key-1234567"
	plaintext := "sensitive-data"

	encrypted, err := EncryptString(plaintext, key1)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	_, err = DecryptString(encrypted, key2)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key")
	}
}

func TestEncryptIfNotEmpty(t *testing.T) {
	key := "my-32-char-encryption-key-123456"

	// Test with non-empty string
	encrypted, err := EncryptIfNotEmpty("test", key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if encrypted == "test" {
		t.Error("Expected encrypted value")
	}

	// Test with empty string
	empty, err := EncryptIfNotEmpty("", key)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if empty != "" {
		t.Error("Expected empty string to remain empty")
	}
}

func TestHashAPIKey_EmptyString(t *testing.T) {
	_, err := HashAPIKey("")
	if err == nil {
		t.Error("Expected error for empty string")
	}
}

func TestEncryptString_Unicode(t *testing.T) {
	key := "my-32-char-encryption-key-123456"
	plaintext := "测试数据 テスト データ 🚀"

	encrypted, err := EncryptString(plaintext, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := DecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Unicode data not preserved. Expected '%s', got '%s'", plaintext, decrypted)
	}
}

func TestEncryptString_LongText(t *testing.T) {
	key := "my-32-char-encryption-key-123456"
	plaintext := strings.Repeat("This is a long text for testing encryption. ", 100)

	encrypted, err := EncryptString(plaintext, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := DecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Error("Long text not preserved correctly")
	}
}

func TestEncryptString_SpecialCharacters(t *testing.T) {
	key := "my-32-char-encryption-key-123456"
	plaintext := "!@#$%^&*()_+-=[]{}|;':\",./<>?`~"

	encrypted, err := EncryptString(plaintext, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := DecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Special characters not preserved. Expected '%s', got '%s'", plaintext, decrypted)
	}
}

func TestHashAPIKey_Deterministic(t *testing.T) {
	apiKey := "test-key-123"

	hash1, err1 := HashAPIKey(apiKey)
	hash2, err2 := HashAPIKey(apiKey)

	if err1 != nil || err2 != nil {
		t.Fatal("Hashing failed")
	}

	// Bcrypt includes salt, so hashes should be different
	if hash1 == hash2 {
		t.Error("Bcrypt hashes should be different due to salt")
	}

	// But both should verify against the same key
	if err := CompareAPIKey(hash1, apiKey); err != nil {
		t.Error("First hash should verify")
	}

	if err := CompareAPIKey(hash2, apiKey); err != nil {
		t.Error("Second hash should verify")
	}
}
