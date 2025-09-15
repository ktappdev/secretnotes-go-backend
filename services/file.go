package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// FileService handles encrypted file operations
type FileService struct {
	App        *pocketbase.PocketBase
	Encryption *Service
}

// NewFileService creates a new file service
func NewFileService(app *pocketbase.PocketBase, encryption *Service) *FileService {
	return &FileService{
		App:        app,
		Encryption: encryption,
	}
}

// StoreEncryptedFile stores an encrypted file (encrypted bytes go into the file_data field)
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

	// Find or create the record in encrypted_files
	filesCollection, err := f.App.FindCollectionByNameOrId("encrypted_files")
	if err != nil {
		return "", fmt.Errorf("files collection not found: %w", err)
	}

	// Delete any existing files with the same phrase hash
	existingRecords, _ := f.App.FindRecordsByFilter(
		"encrypted_files",
		"phrase_hash = {:phrase_hash}",
		"",
		-1, // get all
		0,
		dbx.Params{"phrase_hash": phraseHash},
	)
	for _, existingRec := range existingRecords {
		f.App.Delete(existingRec)
	}

	// Create a new record
	rec := core.NewRecord(filesCollection)
	rec.Set("phrase_hash", phraseHash)

	// Set metadata fields
	rec.Set("file_name", filename)
	rec.Set("content_type", contentType)

	// Create a file from the encrypted bytes and attach it to the file field
	encFile, err := filesystem.NewFileFromBytes(encryptedContent, filename)
	if err != nil {
		return "", fmt.Errorf("failed to create file from bytes: %w", err)
	}
	// File fields expect a slice of files
	rec.Set("file_data", []*filesystem.File{encFile})

	if err := f.App.Save(rec); err != nil {
		return "", fmt.Errorf("failed to save encrypted file: %w", err)
	}

	return fileHash, nil
}

// RetrieveDecryptedFile retrieves and decrypts a file from the file_data field
func (f *FileService) RetrieveDecryptedFile(phrase string) ([]byte, string, string, error) {
	phraseHash := f.hashPhrase(phrase)

	records, err := f.App.FindRecordsByFilter(
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
	if len(records) == 0 {
		return nil, "", "", fmt.Errorf("encrypted file not found")
	}

	rec := records[0]
	contentType := rec.GetString("content_type")
	filename := rec.GetString("file_name")

	// Extract the stored filename from the file_data field
	// PocketBase stores this as a string reference to the actual file
	fileData := rec.Get("file_data")
	var storedFilename string
	switch v := fileData.(type) {
	case string:
		storedFilename = v
	case []*filesystem.File:
		if len(v) > 0 {
			storedFilename = v[0].Name
		}
	case *filesystem.File:
		storedFilename = v.Name
	default:
		return nil, "", "", fmt.Errorf("invalid file data format")
	}

	if storedFilename == "" {
		return nil, "", "", fmt.Errorf("no file stored")
	}

	// Access the file through PocketBase's filesystem
	// Use the original BaseFilesPath approach but fix the file access method
	fs, err := f.App.NewFilesystem()
	if err != nil {
		return nil, "", "", fmt.Errorf("filesystem init: %w", err)
	}
	defer fs.Close()

	// Construct the file storage key using PocketBase's BaseFilesPath
	// Files are stored directly under the record path (no /file_data/ subdirectory)
	fileKey := rec.BaseFilesPath() + "/" + storedFilename

	// Use GetReader to access the encrypted file through PocketBase's filesystem API
	reader, err := fs.GetReader(fileKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to access encrypted file: %w", err)
	}
	defer reader.Close()

	encryptedBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", "", fmt.Errorf("read file content: %w", err)
	}

	decryptedContent, err := f.Encryption.DecryptData(encryptedBytes, phrase)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to decrypt file: %w", err)
	}

	return decryptedContent, filename, contentType, nil
}

// DeleteEncryptedFile deletes an encrypted file record (file bytes are removed by PocketBase)
func (f *FileService) DeleteEncryptedFile(phrase string) error {
	phraseHash := f.hashPhrase(phrase)

	records, err := f.App.FindRecordsByFilter(
		"encrypted_files",
		"phrase_hash = {:phrase_hash}",
		"",
		1,
		0,
		dbx.Params{"phrase_hash": phraseHash},
	)
	if err != nil || len(records) == 0 {
		return fmt.Errorf("encrypted file not found")
	}

	rec := records[0]
	if err := f.App.Delete(rec); err != nil {
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
