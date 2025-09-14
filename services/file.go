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

	records, _ := f.App.FindRecordsByFilter(
		"encrypted_files",
		"phrase_hash = {:phrase_hash}",
		"-created",
		1,
		0,
		dbx.Params{"phrase_hash": phraseHash},
	)

	var rec *core.Record
	if len(records) > 0 {
		rec = records[0]
	} else {
		rec = core.NewRecord(filesCollection)
		rec.Set("phrase_hash", phraseHash)
	}

	// Set metadata fields
	rec.Set("file_name", filename)
	rec.Set("content_type", contentType)

	// Use a form to attach the encrypted bytes to the file field
	// Attach the encrypted bytes to the file field directly via the record
	encFile, err := filesystem.NewFileFromBytes(encryptedContent, filename)
	if err != nil {
		return "", fmt.Errorf("failed to create file from bytes: %w", err)
	}
	rec.Set("file_data", encFile)

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
		return nil, "", "", fmt.Errorf("image file not found")
	}

	rec := records[0]
	contentType := rec.GetString("content_type")
	filename := rec.GetString("file_name")

	storedName := rec.GetString("file_data")
	if storedName == "" {
		return nil, "", "", fmt.Errorf("stored file not found")
	}

	fs, err := f.App.NewFilesystem()
	if err != nil {
		return nil, "", "", fmt.Errorf("filesystem init: %w", err)
	}
	defer fs.Close()

	// Construct the file storage key and read bytes
	key := rec.BaseFilesPath() + "/file_data/" + storedName
	file, err := fs.GetFile(key)
	if err != nil {
		return nil, "", "", fmt.Errorf("get stored file: %w", err)
	}
encryptedBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, "", "", fmt.Errorf("read stored file: %w", err)
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
		"-created",
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
