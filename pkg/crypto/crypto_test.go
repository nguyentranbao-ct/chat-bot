package crypto

import (
	"encoding/base64"
	"testing"
)

func TestCryptoClient(t *testing.T) {
	// Create a 32-byte key and encode it as base64
	key := "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=" // base64 of 32 bytes

	client, err := NewClient(key)
	if err != nil {
		t.Fatalf("Failed to create crypto client: %v", err)
	}

	// Test encryption and decryption
	plaintext := "Hello, World!"

	encrypted, err := client.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if encrypted == "" {
		t.Fatal("Encrypted text should not be empty")
	}

	if encrypted == plaintext {
		t.Fatal("Encrypted text should be different from plaintext")
	}

	decrypted, err := client.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("Decrypted text doesn't match original. Expected: %s, Got: %s", plaintext, decrypted)
	}
}

func TestCryptoClientEmptyStrings(t *testing.T) {
	key := "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI="

	client, err := NewClient(key)
	if err != nil {
		t.Fatalf("Failed to create crypto client: %v", err)
	}

	// Test empty string encryption
	encrypted, err := client.Encrypt("")
	if err != nil {
		t.Fatalf("Failed to encrypt empty string: %v", err)
	}

	if encrypted != "" {
		t.Fatal("Encrypted empty string should return empty string")
	}

	// Test empty string decryption
	decrypted, err := client.Decrypt("")
	if err != nil {
		t.Fatalf("Failed to decrypt empty string: %v", err)
	}

	if decrypted != "" {
		t.Fatal("Decrypted empty string should return empty string")
	}
}

func TestInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty key", ""},
		{"invalid base64", "not-base64!"},
		{"wrong key length", base64.StdEncoding.EncodeToString([]byte("short"))},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewClient(test.key)
			if err == nil {
				t.Fatalf("Expected error for %s, but got none", test.name)
			}
		})
	}
}