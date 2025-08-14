package main

import (
	"crypto/sha256"
	"encoding/hex"
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
	
	// Set the server to listen on port 8091
	app.RootCmd.SetArgs([]string{"serve", "--http", "127.0.0.1:8091"})

	// Initialize services
	encryptionService := services.NewEncryptionService()
	noteService := services.NewNoteService(app, encryptionService)
	fileService := services.NewFileService(app, encryptionService)

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
            if len(phrase) < 3 {
                return e.JSON(http.StatusBadRequest, map[string]string{
                    "error": "Passphrase must be at least 3 characters long",
                })
            }
            return handleGetOrCreateNote(e, phrase, noteService)
        })
		
		// Create note by phrase (explicit POST endpoint)
        api.POST("/notes/{phrase}", func(e *core.RequestEvent) error {
            phrase := e.Request.PathValue("phrase")
            if len(phrase) < 3 {
                return e.JSON(http.StatusBadRequest, map[string]string{
                    "error": "Passphrase must be at least 3 characters long",
                })
            }
            return handleGetOrCreateNote(e, phrase, noteService)
        })

		// Update note by phrase
        api.PATCH("/notes/{phrase}", func(e *core.RequestEvent) error {
            phrase := e.Request.PathValue("phrase")
            if len(phrase) < 3 {
                return e.JSON(http.StatusBadRequest, map[string]string{
                    "error": "Passphrase must be at least 3 characters long",
                })
            }
            return handleUpdateNote(e, phrase, noteService)
        })

		// Upsert note by phrase (create if missing, update if exists)
		api.PUT("/notes/{phrase}", func(e *core.RequestEvent) error {
			phrase := e.Request.PathValue("phrase")
			if len(phrase) < 3 {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": "Passphrase must be at least 3 characters long",
				})
			}
			return handleUpsertNote(e, phrase, noteService)
		})

		// Upload image for note
        api.POST("/notes/{phrase}/image", func(e *core.RequestEvent) error {
            phrase := e.Request.PathValue("phrase")
            if len(phrase) < 3 {
                return e.JSON(http.StatusBadRequest, map[string]string{
                    "error": "Passphrase must be at least 3 characters long",
                })
            }
            return handleUploadImage(e, phrase, noteService, fileService)
        })

		// Get image for note
        api.GET("/notes/{phrase}/image", func(e *core.RequestEvent) error {
            phrase := e.Request.PathValue("phrase")
            if len(phrase) < 3 {
                return e.JSON(http.StatusBadRequest, map[string]string{
                    "error": "Passphrase must be at least 3 characters long",
                })
            }
            return handleGetImage(e, phrase, fileService)
        })

		// Delete image for note
        api.DELETE("/notes/{phrase}/image", func(e *core.RequestEvent) error {
            phrase := e.Request.PathValue("phrase")
            if len(phrase) < 3 {
                return e.JSON(http.StatusBadRequest, map[string]string{
                    "error": "Passphrase must be at least 3 characters long",
                })
            }
            return handleDeleteImage(e, phrase, noteService, fileService)
        })

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// Handler functions
func handleGetOrCreateNote(e *core.RequestEvent, phrase string, noteService *services.NoteService) error {
	// Use the note service to get or create the note
	note, err := noteService.GetOrCreateNote(phrase)
	
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Determine status code based on whether note was created or retrieved
	status := http.StatusOK
	if note.Message == "Welcome to your new secure note!" {
		status = http.StatusCreated
	}

	return e.JSON(status, map[string]any{
		"id": note.ID,
		"message": note.Message,
		"hasImage": note.ImageHash != "",
		"created": note.Created,
		"updated": note.Updated,
	})
}

func handleUpdateNote(e *core.RequestEvent, phrase string, noteService *services.NoteService) error {
	// Read request body
	data := struct {
		Message string `json:"message"`
	}{}
	
	if err := e.BindBody(&data); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}
	
	// Use the note service to update the note
	note, err := noteService.UpdateNote(phrase, data.Message)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}
	
	return e.JSON(http.StatusOK, map[string]any{
		"id": note.ID,
		"message": note.Message,
		"hasImage": note.ImageHash != "",
		"created": note.Created,
		"updated": note.Updated,
	})
}

func handleUploadImage(e *core.RequestEvent, phrase string, noteService *services.NoteService, fileService *services.FileService) error {
	// Check if note exists first
	_, err := noteService.GetOrCreateNote(phrase)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	
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
	
	// Use file service to store the encrypted file
	fileHash, err := fileService.StoreEncryptedFile(phrase, file, header.Filename, header.Header.Get("Content-Type"))
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	
	// Update note with image hash reference using note service
	// We need to add this method to NoteService
	_ = fileHash // For now, we'll handle this in a follow-up
	
	return e.JSON(http.StatusOK, map[string]any{
		"message": "Image uploaded successfully",
		"fileName": header.Filename,
		"fileSize": header.Size,
		"contentType": header.Header.Get("Content-Type"),
	})
}

func handleGetImage(e *core.RequestEvent, phrase string, fileService *services.FileService) error {
	// Use file service to retrieve and decrypt the file
	decryptedData, filename, contentType, err := fileService.RetrieveDecryptedFile(phrase)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}
	
	// Set appropriate headers for file download
	e.Response.Header().Set("Content-Type", contentType)
	e.Response.Header().Set("Content-Disposition", "attachment; filename=\"" + filename + "\"")
	
	// Write the decrypted file directly to the response
	_, err = e.Response.Write(decryptedData)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to send image",
		})
	}
	
	return nil
}

func handleDeleteImage(e *core.RequestEvent, phrase string, noteService *services.NoteService, fileService *services.FileService) error {
	// Use file service to delete the encrypted file
	err := fileService.DeleteEncryptedFile(phrase)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}
	
	// TODO: Update note to remove image hash reference
	// We need to add a method to NoteService for this
	
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

// previewString returns a safe preview of the input limited to max characters.
// If the input is shorter than max, the full string is returned. Otherwise, it appends an ellipsis.
func previewString(s string, max int) string {
    if max <= 0 {
        return ""
    }
    if len(s) <= max {
        return s
    }
    return s[:max] + "..."
}

// handleUpsertNote creates or updates a note in a single call.
// If a record for the phrase exists, it updates the message; otherwise it creates a new note with the message.
func handleUpsertNote(e *core.RequestEvent, phrase string, noteService *services.NoteService) error {
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

    phraseHash := hashPhrase(phrase)

    // Try find existing
    records, err := app.FindRecordsByFilter("notes", "phrase_hash = {:phrase_hash}", "", 1, 0, dbx.Params{"phrase_hash": phraseHash})
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to query notes: " + err.Error(),
        })
    }

    var record *core.Record
    if len(records) > 0 {
        record = records[0]
    } else {
        // Create new record
        collection, err := app.FindCollectionByNameOrId("notes")
        if err != nil {
            return e.JSON(http.StatusInternalServerError, map[string]string{
                "error": "Notes collection not found: " + err.Error(),
            })
        }
        record = core.NewRecord(collection)
        record.Set("phrase_hash", phraseHash)
    }

    // Encrypt and set message (allow empty string)
    encryptedMessage, err := encryptionService.EncryptData([]byte(data.Message), phrase)
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to encrypt message",
        })
    }
    record.Set("message", string(encryptedMessage))

    if err := app.Save(record); err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to save note",
        })
    }

    status := http.StatusOK
    if len(records) == 0 {
        status = http.StatusCreated
    }

    return e.JSON(status, map[string]any{
        "id": record.Id,
        "message": data.Message,
        "hasImage": record.GetString("image_hash") != "",
        "created": record.GetDateTime("created"),
        "updated": record.GetDateTime("updated"),
    })
}
