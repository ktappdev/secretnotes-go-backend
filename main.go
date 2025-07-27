package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/dbx"
	_ "secretnotes/migrations" // Import migrations
	"secretnotes/services"
)

func main() {
	app := pocketbase.New()

	// Initialize services
	encryptionService := services.NewEncryptionService()
	// Initialize services but use them directly in handlers
	_ = services.NewNoteService(app, encryptionService)
	_ = services.NewFileService(app, encryptionService)

	// Register custom routes
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Create a route group for our API
		api := se.Router.Group("/api/secretnotes")

		// Health check endpoint
		api.GET("/", func(e *core.RequestEvent) error {
			return e.JSON(http.StatusOK, map[string]string{
				"message": "Secret Notes API is live",
				"version": "1.0.0",
			})
		})

		// Get note by phrase
		api.GET("/notes/{phrase}", func(e *core.RequestEvent) error {
			phrase := e.Request.PathValue("phrase")
			if len(phrase) < 32 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Passphrase must be at least 32 characters long",
				})
			}
			return handleGetOrCreateNote(e, phrase)
		})
		
		// Create note by phrase (explicit POST endpoint)
		api.POST("/notes/{phrase}", func(e *core.RequestEvent) error {
			phrase := e.Request.PathValue("phrase")
			if len(phrase) < 32 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Passphrase must be at least 32 characters long",
				})
			}
			return handleGetOrCreateNote(e, phrase)
		})

		// Update note by phrase
		api.PATCH("/notes/{phrase}", func(e *core.RequestEvent) error {
			phrase := e.Request.PathValue("phrase")
			if len(phrase) < 32 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Passphrase must be at least 32 characters long",
				})
			}
			return handleUpdateNote(e, phrase)
		})

		// Upload image for note
		api.POST("/notes/{phrase}/image", func(e *core.RequestEvent) error {
			phrase := e.Request.PathValue("phrase")
			if len(phrase) < 32 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Passphrase must be at least 32 characters long",
				})
			}
			return handleUploadImage(e, phrase)
		})

		// Get image for note
		api.GET("/notes/{phrase}/image", func(e *core.RequestEvent) error {
			phrase := e.Request.PathValue("phrase")
			if len(phrase) < 32 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Passphrase must be at least 32 characters long",
				})
			}
			return handleGetImage(e, phrase)
		})

		// Delete image for note
		api.DELETE("/notes/{phrase}/image", func(e *core.RequestEvent) error {
			phrase := e.Request.PathValue("phrase")
			if len(phrase) < 32 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Passphrase must be at least 32 characters long",
				})
			}
			return handleDeleteImage(e, phrase)
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// Handler functions
func handleGetOrCreateNote(e *core.RequestEvent, phrase string) error {
	app := e.App
	encryptionService := services.NewEncryptionService()
	
	// Hash the phrase for secure lookup
	phraseHash := hashPhrase(phrase)
	
	// Try to find existing note
	log.Printf("Looking for note with phrase_hash: %s", phraseHash)
	
	// Check if collection exists
	collection, err := app.FindCollectionByNameOrId("notes")
	if err != nil {
		log.Printf("Error finding collection: %v", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Notes collection not found: " + err.Error(),
		})
	}
	log.Printf("Found collection: %s", collection.Name)
	
	// Try to find existing note
	records, err := app.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "", 1, 0, dbx.Params{"phrase_hash": phraseHash})
	
	if err != nil {
		log.Printf("Error querying notes: %v", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to query notes: " + err.Error(),
		})
	}
	
	if len(records) > 0 {
		// Note exists, decrypt and return
		record := records[0]
		
		// Get message
		encryptedMessage := record.GetString("message")
		var message string
		
		log.Printf("Retrieved message from record: %s (length: %d)", encryptedMessage, len(encryptedMessage))
		
		if encryptedMessage != "" {
			// Try to decrypt the message
			log.Printf("Attempting to decrypt message with phrase: %s (hash: %s)", phrase[:5]+"...", phraseHash[:10]+"...")
			decryptedBytes, err := encryptionService.DecryptData([]byte(encryptedMessage), phrase)
			if err != nil {
				// If decryption fails, assume it's a plaintext message from direct API
				log.Printf("Decryption failed, assuming plaintext message: %v", err)
				message = encryptedMessage
			} else {
				log.Printf("Decryption succeeded, message length: %d", len(decryptedBytes))
				message = string(decryptedBytes)
			}
		} else {
			message = ""
		}
		
		return e.JSON(http.StatusOK, map[string]any{
			"id": record.Id,
			"message": message,
			"hasImage": record.GetString("image_hash") != "",
			"created": record.GetDateTime("created"),
			"updated": record.GetDateTime("updated"),
		})
	}
	
	// Create new note
	// Collection already retrieved above, no need to get it again
	if collection == nil {
		var findErr error
		collection, findErr = app.FindCollectionByNameOrId("notes")
		if findErr != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Notes collection not found: " + findErr.Error(),
			})
		}
	}
	
	record := core.NewRecord(collection)
	record.Set("phrase_hash", phraseHash)
	
	// Create an encrypted empty message to satisfy validation
	encryptedMessage, err := encryptionService.EncryptData([]byte("Welcome to your new secure note!"), phrase)
	if err != nil {
		log.Printf("Error encrypting initial message: %v", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to encrypt initial message: " + err.Error(),
		})
	}
	
	record.Set("message", string(encryptedMessage))
	
	if err := app.Save(record); err != nil {
		log.Printf("Error creating note: %v", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create note: " + err.Error(),
		})
	}
	
	return e.JSON(http.StatusCreated, map[string]any{
		"id": record.Id,
		"message": "",
		"hasImage": false,
		"created": record.GetDateTime("created"),
		"updated": record.GetDateTime("updated"),
	})
}

func handleUpdateNote(e *core.RequestEvent, phrase string) error {
	// Read request body
	data := struct {
		Message string `json:"message"`
	}{}
	
	if err := e.BindBody(&data); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}
	
	app := e.App
	encryptionService := services.NewEncryptionService()
	
	// Hash the phrase for secure lookup
	phraseHash := hashPhrase(phrase)
	
	// Find the note
	records, err := app.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "", 1, 0, dbx.Params{"phrase_hash": phraseHash})
	if err != nil || len(records) == 0 {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Note not found",
		})
	}
	
	record := records[0]
	
	// Encrypt the message
	encryptedMessage, err := encryptionService.EncryptData([]byte(data.Message), phrase)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to encrypt message",
		})
	}
	
	record.Set("message", string(encryptedMessage))
	
	if err := app.Save(record); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update note",
		})
	}
	
	return e.JSON(http.StatusOK, map[string]any{
		"id": record.Id,
		"message": data.Message, // Return the unencrypted message to the client
		"hasImage": record.GetString("image_hash") != "",
		"created": record.GetDateTime("created"),
		"updated": record.GetDateTime("updated"),
	})
}

func handleUploadImage(e *core.RequestEvent, phrase string) error {
	app := e.App
	encryptionService := services.NewEncryptionService()
	
	// Hash the phrase for secure lookup
	phraseHash := hashPhrase(phrase)
	log.Printf("Looking for note with phrase_hash: %s", phraseHash)
	
	// Check if note exists
	noteRecords, err := app.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "", 1, 0, dbx.Params{"phrase_hash": phraseHash})
	if err != nil {
		log.Printf("Error finding note: %v", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Error finding note: " + err.Error(),
		})
	}
	
	if len(noteRecords) == 0 {
		log.Printf("No notes found with phrase_hash: %s", phraseHash)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Note not found",
		})
	}
	
	log.Printf("Found note with ID: %s", noteRecords[0].Id)
	noteRecord := noteRecords[0]
	
	// Parse multipart form
	if err := e.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to parse form",
		})
	}
	
	// Get uploaded file
	file, header, err := e.Request.FormFile("image")
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "No image file provided",
		})
	}
	defer file.Close()
	
	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read file",
		})
	}
	
	// Encrypt the file content
	encryptedContent, err := encryptionService.EncryptData(fileContent, phrase)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to encrypt file",
		})
	}
	
	// Generate a hash for the encrypted file
	fileHash := hashBytes(encryptedContent)
	
	// Store in encrypted_files collection
	filesCollection, err := app.FindCollectionByNameOrId("encrypted_files")
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Files collection not found",
		})
	}
	
	// Check if file already exists for this phrase
	existingFiles, err := app.FindRecordsByFilter(
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
	fileRecord.Set("file_name", header.Filename)
	fileRecord.Set("content_type", header.Header.Get("Content-Type"))
	
	// For PocketBase, we need to store the encrypted content in a separate field
	// since file_data will be handled by PocketBase's file storage system
	
	// Store the encrypted content as a base64 string in a custom field
	log.Printf("Storing encrypted content of length: %d bytes", len(encryptedContent))
	fileRecord.Set("encrypted_content", encryptedContent)
	
	// Use the original file for PocketBase's file_data field
	// PocketBase expects the original form file to be used directly
	log.Printf("Setting file_data with the original file")
	
	// Pass the original file directly to PocketBase
	// This is the correct way to set file_data in PocketBase
	fileRecord.Set("file_data", file)
	
	// Debug: log all fields being set
	log.Printf("File record fields before save:")
	log.Printf("- phrase_hash: %s", fileRecord.GetString("phrase_hash"))
	log.Printf("- file_name: %s", fileRecord.GetString("file_name"))
	log.Printf("- content_type: %s", fileRecord.GetString("content_type"))
	log.Printf("- encrypted_content length: %d", len(fileRecord.GetString("encrypted_content")))
	log.Printf("- file_data type: %T", fileRecord.Get("file_data"))
	
	log.Printf("Attempting to save file record with ID: %s", fileRecord.Id)
	if err := app.Save(fileRecord); err != nil {
		log.Printf("Error saving encrypted file: %v", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to save encrypted file: " + err.Error(),
		})
	}
	
	// Update note with image hash reference
	noteRecord.Set("image_hash", fileHash)
	if err := app.Save(noteRecord); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update note with image reference",
		})
	}
	
	return e.JSON(http.StatusOK, map[string]any{
		"message": "Image uploaded successfully",
		"fileName": header.Filename,
		"fileSize": header.Size,
		"contentType": header.Header.Get("Content-Type"),
	})
}

func handleGetImage(e *core.RequestEvent, phrase string) error {
	app := e.App
	encryptionService := services.NewEncryptionService()
	
	log.Printf("Getting image for phrase: %s", phrase)
	
	// Hash the phrase for secure lookup
	phraseHash := hashPhrase(phrase)
	log.Printf("Looking for note with phrase_hash: %s", phraseHash)
	
	// Find the note to get the image hash
	noteRecords, err := app.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "", 1, 0, dbx.Params{"phrase_hash": phraseHash})
	if err != nil {
		log.Printf("Error finding note: %v", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Error finding note: " + err.Error(),
		})
	}
	
	if len(noteRecords) == 0 {
		log.Printf("No notes found with phrase_hash: %s", phraseHash)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Note not found",
		})
	}
	
	log.Printf("Found note with ID: %s", noteRecords[0].Id)
	
	noteRecord := noteRecords[0]
	log.Printf("Checking image_hash for note ID %s: '%s'", noteRecord.Id, noteRecord.GetString("image_hash"))
	if noteRecord.GetString("image_hash") == "" {
		log.Printf("No image hash found for note ID: %s", noteRecord.Id)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "No image associated with this note",
		})
	}
	
	// Find the encrypted file
	log.Printf("Looking for encrypted file with phrase_hash: %s", phraseHash)
	fileRecords, err := app.FindRecordsByFilter(
		"encrypted_files", 
		"phrase_hash = {:phrase_hash}", 
		"", 
		1, 
		0, 
		dbx.Params{"phrase_hash": phraseHash},
	)
	
	if err != nil {
		log.Printf("Error finding encrypted file: %v", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Error finding image file: " + err.Error(),
		})
	}
	
	if len(fileRecords) == 0 {
		log.Printf("No encrypted files found with phrase_hash: %s", phraseHash)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Image file not found",
		})
	}
	
	log.Printf("Found encrypted file with ID: %s", fileRecords[0].Id)
	
	fileRecord := fileRecords[0]
	contentType := fileRecord.GetString("content_type")
	fileName := fileRecord.GetString("file_name")
	
	log.Printf("Content type: %s", contentType)
	log.Printf("File name: %s", fileName)
	
	// Debug: log all available fields in the file record
	log.Printf("File record fields available:")
	for key, value := range fileRecord.PublicExport() {
		log.Printf("- %s: %v (type: %T)", key, value, value)
	}
	
	// Check if we have encrypted content stored separately
	encryptedContent := fileRecord.Get("encrypted_content")
	log.Printf("encrypted_content field value: %v (type: %T)", encryptedContent, encryptedContent)
	if encryptedContent != nil {
		log.Printf("Found encrypted_content field, using it for decryption")
		
		// Convert to byte array if it's stored as a string
		var encryptedBytes []byte
		switch v := encryptedContent.(type) {
		case []byte:
			encryptedBytes = v
		case string:
			encryptedBytes = []byte(v)
		default:
			log.Printf("encrypted_content is of unexpected type: %T", encryptedContent)
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Invalid encrypted content format",
			})
		}
		
		// Decrypt the content
		decryptedData, err := encryptionService.DecryptData(encryptedBytes, phrase)
		if err != nil {
			log.Printf("Error decrypting file from encrypted_content: %v", err)
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to decrypt image",
			})
		}
		
		// Set appropriate headers for file download
		e.Response.Header().Set("Content-Type", contentType)
		e.Response.Header().Set("Content-Disposition", "attachment; filename=\"" + fileName + "\"")
		
		// Write the decrypted file directly to the response
		_, err = e.Response.Write(decryptedData)
		if err != nil {
			log.Printf("Error writing decrypted file: %v", err)
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to send image",
			})
		}
		
		return nil
	}
	
	// If we don't have encrypted_content, return an error
	// This means the file was uploaded via direct PocketBase API without encryption
	log.Printf("No encrypted_content field found, file may not be encrypted")
	return e.JSON(http.StatusNotFound, map[string]string{
		"error": "Encrypted file content not found",
	})
}

func handleDeleteImage(e *core.RequestEvent, phrase string) error {
	app := e.App

	// Hash the phrase for secure lookup
	phraseHash := hashPhrase(phrase)

	// Find the note
	noteRecords, err := app.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "-created", 1, 0, dbx.Params{"phrase_hash": phraseHash})
	if err != nil || len(noteRecords) == 0 {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Note not found",
		})
	}

	noteRecord := noteRecords[0]
	if noteRecord.GetString("image_hash") == "" {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "No image associated with this note",
		})
	}
	
	// Find the encrypted file
	fileRecords, err := app.FindRecordsByFilter(
		"encrypted_files", 
		"phrase_hash = {:phrase_hash}", 
		"-created", 
		1, 
		0, 
		dbx.Params{"phrase_hash": phraseHash},
	)
	
	if err == nil && len(fileRecords) > 0 {
		// Delete the file record
		fileRecord := fileRecords[0]
		if err := app.Delete(fileRecord); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to delete encrypted file",
			})
		}
	}
	
	// Update note to remove image reference
	noteRecord.Set("image_hash", "")
	if err := app.Save(noteRecord); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update note",
		})
	}
	
	return e.JSON(http.StatusOK, map[string]string{
		"message": "Image deleted successfully",
	})
}

// Helper functions

// hashPhrase creates a SHA-256 hash of the phrase for secure storage and lookup
func hashPhrase(phrase string) string {
	hash := sha256.Sum256([]byte(phrase))
	return hex.EncodeToString(hash[:])
}

// hashBytes creates a SHA-256 hash of a byte array
func hashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
