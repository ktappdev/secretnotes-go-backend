package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/dbx"
)

// Note represents a secret note
type Note struct {
	ID        string    `json:"id"`
    Phrase    string    `json:"phrase"`    // Encrypted identifier
	Message   string    `json:"message"`   // Encrypted note content
	ImageHash string    `json:"image_hash"` // Hash for encrypted image lookup
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
}

// NoteService handles note operations
type NoteService struct {
	App        *pocketbase.PocketBase
	Encryption *Service
}

// NewNoteService creates a new note service
func NewNoteService(app *pocketbase.PocketBase, encryption *Service) *NoteService {
	return &NoteService{
		App:        app,
		Encryption: encryption,
	}
}

// GetOrCreateNote retrieves an existing note or creates a new one
func (n *NoteService) GetOrCreateNote(phrase string) (*Note, error) {
	// Validate phrase length
	if len(phrase) < 3 {
		return nil, fmt.Errorf("phrase must be at least 3 characters long")
	}

	// Hash the phrase for secure lookup
	phraseHash := n.hashPhrase(phrase)

	// Try to find existing note
	records, err := n.App.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "", 1, 0, dbx.Params{"phrase_hash": phraseHash})
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}

	if len(records) > 0 {
		// Note exists, decrypt and return
		record := records[0]
		encryptedMessage := record.GetString("message")
		var message string

		if encryptedMessage != "" {
			// Try to decrypt the message
			decryptedBytes, err := n.Encryption.DecryptData([]byte(encryptedMessage), phrase)
			if err != nil {
				// If decryption fails, assume it's plaintext
				message = encryptedMessage
			} else {
				message = string(decryptedBytes)
			}
		}

		return &Note{
			ID:        record.Id,
			Phrase:    phraseHash, // Store hash, not original phrase
			Message:   message,
			ImageHash: record.GetString("image_hash"),
			Created:   record.GetDateTime("created").Time(),
			Updated:   record.GetDateTime("updated").Time(),
		}, nil
	}

	// Create new note
	collection, err := n.App.FindCollectionByNameOrId("notes")
	if err != nil {
		return nil, fmt.Errorf("notes collection not found: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("phrase_hash", phraseHash)

	// Create an encrypted welcome message
	encryptedMessage, err := n.Encryption.EncryptData([]byte("Welcome to your new secure note!"), phrase)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt initial message: %w", err)
	}

	record.Set("message", string(encryptedMessage))

	if err := n.App.Save(record); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	return &Note{
		ID:        record.Id,
		Phrase:    phraseHash,
		Message:   "Welcome to your new secure note!",
		ImageHash: "",
		Created:   record.GetDateTime("created").Time(),
		Updated:   record.GetDateTime("updated").Time(),
	}, nil
}

// UpdateNote updates an existing note
func (n *NoteService) UpdateNote(phrase, message string) (*Note, error) {
	// Validate phrase length
	if len(phrase) < 3 {
		return nil, fmt.Errorf("phrase must be at least 3 characters long")
	}

	// Hash the phrase for secure lookup
	phraseHash := n.hashPhrase(phrase)

	// Find the existing note
	records, err := n.App.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "", 1, 0, dbx.Params{"phrase_hash": phraseHash})
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("note not found")
	}

	record := records[0]

	// Encrypt the message
	encryptedMessage, err := n.Encryption.EncryptData([]byte(message), phrase)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Update the record
	record.Set("message", string(encryptedMessage))

	if err := n.App.Save(record); err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	return &Note{
		ID:        record.Id,
		Phrase:    phraseHash,
		Message:   message, // Return unencrypted message
		ImageHash: record.GetString("image_hash"),
		Created:   record.GetDateTime("created").Time(),
		Updated:   record.GetDateTime("updated").Time(),
	}, nil
}

// DeleteNote deletes a note
func (n *NoteService) DeleteNote(phrase string) error {
	// Validate phrase length
	if len(phrase) < 3 {
		return fmt.Errorf("phrase must be at least 3 characters long")
	}

	// Hash the phrase for secure lookup
	phraseHash := n.hashPhrase(phrase)

	// Find the note to delete
	records, err := n.App.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "", 1, 0, dbx.Params{"phrase_hash": phraseHash})
	if err != nil || len(records) == 0 {
		return fmt.Errorf("note not found")
	}

	record := records[0]

	// Also delete any associated encrypted files
	fileRecords, err := n.App.FindRecordsByFilter("encrypted_files", "phrase_hash = {:phrase_hash}", "", -1, 0, dbx.Params{"phrase_hash": phraseHash})
	if err == nil {
		for _, fileRecord := range fileRecords {
			if deleteErr := n.App.Delete(fileRecord); deleteErr != nil {
				log.Printf("Warning: failed to delete associated file: %v", deleteErr)
			}
		}
	}

	// Delete the note
	if err := n.App.Delete(record); err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	return nil
}

// hashPhrase creates a SHA-256 hash of the phrase for secure storage and lookup
func (n *NoteService) hashPhrase(phrase string) string {
	hash := sha256.Sum256([]byte(phrase))
	return hex.EncodeToString(hash[:])
}
