package services

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase"
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

	// TODO: Check if note exists in PocketBase
	// This will be implemented when we set up PocketBase collections
	var note *Note

	// If note doesn't exist, create a new one
	if note == nil {
		// Encrypt the phrase for storage (we don't store the plain phrase)
		encryptedPhrase, err := n.Encryption.EncryptString(phrase, phrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt phrase: %w", err)
		}

		note = &Note{
			Phrase:  encryptedPhrase,
			Message: "", // Empty message initially
			Created: time.Now(),
			Updated: time.Now(),
		}

		// TODO: Save note to PocketBase
		// This will be implemented when we set up PocketBase collections
	}

	return note, nil
}

// UpdateNote updates an existing note
func (n *NoteService) UpdateNote(phrase, message string) (*Note, error) {
	// Validate phrase length
    if len(phrase) < 3 {
        return nil, fmt.Errorf("phrase must be at least 3 characters long")
	}

	// Encrypt the message
	encryptedMessage, err := n.Encryption.EncryptString(message, phrase)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt message: %w", err)
	}

	// TODO: Retrieve existing note from PocketBase
	// This will be implemented when we set up PocketBase collections
	var note *Note

	if note == nil {
		return nil, fmt.Errorf("note not found")
	}

	// Update the note
	note.Message = encryptedMessage
	note.Updated = time.Now()

	// TODO: Save updated note to PocketBase
	// This will be implemented when we set up PocketBase collections

	return note, nil
}

// DeleteNote deletes a note
func (n *NoteService) DeleteNote(phrase string) error {
	// Validate phrase length
    if len(phrase) < 3 {
        return fmt.Errorf("phrase must be at least 3 characters long")
	}

	// TODO: Delete note from PocketBase
	// This will be implemented when we set up PocketBase collections

	return nil
}
