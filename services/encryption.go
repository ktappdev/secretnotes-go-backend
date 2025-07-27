package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// Service provides encryption and decryption functionality
type Service struct {
	SaltSize int
	KeySize  int
}

// NewEncryptionService creates a new encryption service
func NewEncryptionService() *Service {
	return &Service{
		SaltSize: 16, // 128 bits
		KeySize:  32, // 256 bits
	}
}

// DeriveKey derives a key from a passphrase using PBKDF2
func (s *Service) DeriveKey(phrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(phrase), salt, 10000, s.KeySize, sha256.New)
}

// EncryptData encrypts data using AES-256-GCM
func (s *Service) EncryptData(data []byte, phrase string) ([]byte, error) {
	// Generate random salt
	salt := make([]byte, s.SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key from phrase
	key := s.DeriveKey(phrase, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Use GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	encrypted := gcm.Seal(nil, nonce, data, nil)

	// Combine salt + nonce + encrypted data
	result := make([]byte, 0, len(salt)+len(nonce)+len(encrypted))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, encrypted...)

	return result, nil
}

// DecryptData decrypts data using AES-256-GCM
func (s *Service) DecryptData(encryptedData []byte, phrase string) ([]byte, error) {
	// Extract salt, nonce, and encrypted data
	if len(encryptedData) < s.SaltSize+12 { // 12 is minimum nonce size
		return nil, fmt.Errorf("encrypted data is too short")
	}

	// Extract components
	salt := encryptedData[:s.SaltSize]
	nonceStart := s.SaltSize
	nonceEnd := nonceStart + 12 // GCM nonce size is 12 bytes
	encryptedStart := nonceEnd

	if len(encryptedData) <= encryptedStart {
		return nil, fmt.Errorf("invalid encrypted data format")
	}

	// Extract components
	nonce := encryptedData[nonceStart:nonceEnd]
	encrypted := encryptedData[encryptedStart:]

	// Derive key from phrase
	key := s.DeriveKey(phrase, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Use GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt data
	decrypted, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return decrypted, nil
}

// EncryptString encrypts a string and returns a base64 encoded string
func (s *Service) EncryptString(text string, phrase string) (string, error) {
	encrypted, err := s.EncryptData([]byte(text), phrase)
	if err != nil {
		return "", err
	}
	return string(encrypted), nil
}

// DecryptString decrypts a base64 encoded string and returns the original string
func (s *Service) DecryptString(encryptedText string, phrase string) (string, error) {
	decrypted, err := s.DecryptData([]byte(encryptedText), phrase)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}
