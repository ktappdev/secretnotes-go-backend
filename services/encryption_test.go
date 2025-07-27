package services

import (
	"testing"
)

func TestEncryptionService(t *testing.T) {
	// Create a new encryption service
	svc := NewEncryptionService()

	// Test data
	originalText := "This is a secret message"
	phrase := "this_is_a_very_long_passphrase_that_is_at_least_32_characters_long"

	// Encrypt the text
	encryptedText, err := svc.EncryptString(originalText, phrase)
	if err != nil {
		t.Fatalf("Failed to encrypt text: %v", err)
	}

	// Decrypt the text
	decryptedText, err := svc.DecryptString(encryptedText, phrase)
	if err != nil {
		t.Fatalf("Failed to decrypt text: %v", err)
	}

	// Verify the decrypted text matches the original
	if decryptedText != originalText {
		t.Errorf("Decrypted text does not match original. Got: %s, Expected: %s", decryptedText, originalText)
	}
}

func TestEncryptionServiceWithDifferentPhrases(t *testing.T) {
	// Create a new encryption service
	svc := NewEncryptionService()

	// Test data
	originalText := "This is a secret message"
	phrase1 := "this_is_a_very_long_passphrase_that_is_at_least_32_characters_long"
	phrase2 := "this_is_a_different_very_long_passphrase_that_is_at_least_32_characters_long"

	// Encrypt the text with the first phrase
	encryptedText1, err := svc.EncryptString(originalText, phrase1)
	if err != nil {
		t.Fatalf("Failed to encrypt text with phrase1: %v", err)
	}

	// Try to decrypt with the second phrase (should fail)
	_, err = svc.DecryptString(encryptedText1, phrase2)
	if err == nil {
		t.Error("Expected decryption to fail with different phrase, but it succeeded")
	}
}
