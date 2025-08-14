package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/dbx"
)

// FileService handles encrypted file operations
type FileService struct {
	App         *pocketbase.PocketBase
	Encryption  *Service
}

// NewFileService creates a new file service
func NewFileService(app *pocketbase.PocketBase, encryption *Service) *FileService {
	return &FileService{
		App:        app,
		Encryption: encryption,
	}
}

// StoreEncryptedFile stores an encrypted file
func (f *FileService) StoreEncryptedFile(phrase string, file multipart.File, filename, contentType string) (string, error) {
	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Encrypt the file content
	encryptedContent, err := f.Encryption.EncryptData(content, phrase)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt file: %w", err)
	}

	// Hash the phrase for secure lookup
	phraseHash := f.hashPhrase(phrase)

	// Generate a hash for the encrypted file
	fileHash := f.hashBytes(encryptedContent)

	// Store in encrypted_files collection
	filesCollection, err := f.App.FindCollectionByNameOrId("encrypted_files")
	if err != nil {
		return "", fmt.Errorf("files collection not found: %w", err)
	}

	// Check if file already exists for this phrase
	existingFiles, err := f.App.FindRecordsByFilter(
		"encrypted_files",
		"phrase_hash = {:phrase_hash}",
		"-created",
		1,
		0,
		dbx.Params{"phrase_hash": phraseHash},
	)

	var fileRecord *core.Record

	if err == nil && len(existingFiles) > 0 {
		// Update existing file
		fileRecord = existingFiles[0]
	} else {
		// Create new file record
		fileRecord = core.NewRecord(filesCollection)
		fileRecord.Set("phrase_hash", phraseHash)
	}

	// Set file data
	fileRecord.Set("file_name", filename)
	fileRecord.Set("content_type", contentType)
	fileRecord.Set("encrypted_content", encryptedContent)
	fileRecord.Set("file_data", file)

	if err := f.App.Save(fileRecord); err != nil {
		return "", fmt.Errorf("failed to save encrypted file: %w", err)
	}

	return fileHash, nil
}

// RetrieveDecryptedFile retrieves and decrypts a file
func (f *FileService) RetrieveDecryptedFile(phrase string) ([]byte, string, string, error) {
	// Hash the phrase for secure lookup
	phraseHash := f.hashPhrase(phrase)

	// Find the encrypted file
	fileRecords, err := f.App.FindRecordsByFilter(
		"encrypted_files",
		"phrase_hash = {:phrase_hash}",
		"",
		1,
		0,
		dbx.Params{"phrase_hash": phraseHash},
	)

	if err != nil {
		return nil, "", "", fmt.Errorf("error finding encrypted file: %w", err)
	}

	if len(fileRecords) == 0 {
		return nil, "", "", fmt.Errorf("image file not found")
	}

	fileRecord := fileRecords[0]
	contentType := fileRecord.GetString("content_type")
	filename := fileRecord.GetString("file_name")

	// Get encrypted content
	encryptedContent := fileRecord.Get("encrypted_content")
	if encryptedContent == nil {
		return nil, "", "", fmt.Errorf("encrypted file content not found")
	}

	// Convert to byte array
	var encryptedBytes []byte
	switch v := encryptedContent.(type) {
	case []byte:
		encryptedBytes = v
	case string:
		encryptedBytes = []byte(v)
	default:
		return nil, "", "", fmt.Errorf("invalid encrypted content format")
	}

	// Decrypt the file content
	decryptedContent, err := f.Encryption.DecryptData(encryptedBytes, phrase)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to decrypt file: %w", err)
	}

	return decryptedContent, filename, contentType, nil
}

// DeleteEncryptedFile deletes an encrypted file
func (f *FileService) DeleteEncryptedFile(phrase string) error {
	// Hash the phrase for secure lookup
	phraseHash := f.hashPhrase(phrase)

	// Find the encrypted file
	fileRecords, err := f.App.FindRecordsByFilter(
		"encrypted_files",
		"phrase_hash = {:phrase_hash}",
		"-created",
		1,
		0,
		dbx.Params{"phrase_hash": phraseHash},
	)

	if err != nil || len(fileRecords) == 0 {
		return fmt.Errorf("encrypted file not found")
	}

	// Delete the file record
	fileRecord := fileRecords[0]
	if err := f.App.Delete(fileRecord); err != nil {
		return fmt.Errorf("failed to delete encrypted file: %w", err)
	}

	return nil
}

// hashPhrase creates a SHA-256 hash of the phrase for secure storage and lookup
func (f *FileService) hashPhrase(phrase string) string {
	hash := sha256.Sum256([]byte(phrase))
	return hex.EncodeToString(hash[:])
}

// hashBytes creates a SHA-256 hash of a byte array
func (f *FileService) hashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
