package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

const (
	v2Collection       = "notes_v2"
	v2TokenHeader      = "X-Lookup-Token"
	v2RealtimePrefix   = "secretnotez:v2:"
	v2MaxJSONBody      = 400_000
	v2MaxImageBytes    = 15 << 20
	v2MaxMultipartBody = 16 << 20
	v2MaxCiphertext    = 300_000
	v2MaxMetadata      = 20_000
	legacyHashHeader   = "X-Legacy-Lookup-Hash"
	legacyHashPattern  = "^[a-f0-9]{64}$"
)

var (
	errV2Conflict = errors.New("version conflict")
	hexHashRE     = regexp.MustCompile(legacyHashPattern)
)

type v2NoteResponse struct {
	ID         string `json:"id"`
	Ciphertext string `json:"ciphertext"`
	HasImage   bool   `json:"hasImage"`
	Version    int    `json:"version"`
	Created    string `json:"created"`
	Updated    string `json:"updated"`
}

type v2Envelope struct {
	Version    int    `json:"v"`
	Nonce      string `json:"n"`
	Ciphertext string `json:"c"`
}

type fixedWindowLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	entries map[string]rateEntry
	checks  int
}

type rateEntry struct {
	started time.Time
	count   int
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{limit: limit, window: window, entries: make(map[string]rateEntry)}
}

func (l *fixedWindowLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.checks++
	if l.checks%1000 == 0 {
		for entryKey, candidate := range l.entries {
			if now.Sub(candidate.started) >= l.window {
				delete(l.entries, entryKey)
			}
		}
	}
	entry := l.entries[key]
	if entry.started.IsZero() || now.Sub(entry.started) >= l.window {
		l.entries[key] = rateEntry{started: now, count: 1}
		return true
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

func registerV2Routes(api *router.RouterGroup[*core.RequestEvent], app *pocketbase.PocketBase) {
	generalLimiter := newFixedWindowLimiter(120, time.Minute)
	recoveryLimiter := newFixedWindowLimiter(10, time.Minute)

	api.GET("/v2/notes", func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !generalLimiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		lookupHash, err := v2LookupHash(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		record, err := findV2Note(e.App, lookupHash)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load note"})
		}
		if record == nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "Note not found"})
		}
		return e.JSON(http.StatusOK, v2Response(record))
	})

	api.PUT("/v2/notes", func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !generalLimiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		lookupHash, err := v2LookupHash(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		data := struct {
			Ciphertext string `json:"ciphertext"`
			Version    int    `json:"version"`
		}{}
		if err := bindLimitedJSON(e, &data); err != nil || !validV2Envelope(data.Ciphertext, v2MaxCiphertext) {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid encrypted note"})
		}

		var saved *core.Record
		created := false
		err = e.App.RunInTransaction(func(txApp core.App) error {
			record, findErr := findV2Note(txApp, lookupHash)
			if findErr != nil {
				return findErr
			}
			if record == nil {
				created = true
				collection, collectionErr := txApp.FindCollectionByNameOrId(v2Collection)
				if collectionErr != nil {
					return collectionErr
				}
				record = core.NewRecord(collection)
				record.Set("lookup_hash", lookupHash)
				record.Set("version", 1)
			} else {
				current := int(record.GetInt("version"))
				if data.Version > 0 && data.Version != current {
					return fmt.Errorf("%w:%d", errV2Conflict, current)
				}
				record.Set("version", current+1)
			}
			record.Set("ciphertext", data.Ciphertext)
			if saveErr := txApp.Save(record); saveErr != nil {
				return saveErr
			}
			saved = record
			return nil
		})
		if err != nil {
			if errors.Is(err, errV2Conflict) {
				parts := strings.Split(err.Error(), ":")
				current, _ := strconv.Atoi(parts[len(parts)-1])
				return e.JSON(http.StatusConflict, map[string]any{"error": "version_conflict", "currentVersion": current})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save note"})
		}
		publishV2Update(app, lookupHash, saved)
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		return e.JSON(status, v2Response(saved))
	})

	api.POST("/v2/import", func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !recoveryLimiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		lookupHash, err := v2LookupHash(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := parseLimitedMultipart(e); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid import"})
		}
		ciphertext := e.Request.FormValue("ciphertext")
		metadata := e.Request.FormValue("image_metadata")
		if !validV2Envelope(ciphertext, v2MaxCiphertext) || (metadata != "" && !validV2Envelope(metadata, v2MaxMetadata)) {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid encrypted import"})
		}
		imageFile, _, imageErr := e.Request.FormFile("image")
		if imageErr != nil && !errors.Is(imageErr, http.ErrMissingFile) {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid encrypted image"})
		}
		if imageFile != nil {
			defer imageFile.Close()
		}

		var saved *core.Record
		err = e.App.RunInTransaction(func(txApp core.App) error {
			existing, findErr := findV2Note(txApp, lookupHash)
			if findErr != nil {
				return findErr
			}
			if existing != nil {
				return errV2Conflict
			}
			collection, collectionErr := txApp.FindCollectionByNameOrId(v2Collection)
			if collectionErr != nil {
				return collectionErr
			}
			record := core.NewRecord(collection)
			record.Set("lookup_hash", lookupHash)
			record.Set("ciphertext", ciphertext)
			record.Set("version", 1)
			if imageFile != nil {
				contents, readErr := io.ReadAll(io.LimitReader(imageFile, v2MaxImageBytes+1))
				if readErr != nil || len(contents) > v2MaxImageBytes {
					return fmt.Errorf("encrypted image exceeds limit")
				}
				stored, fileErr := filesystem.NewFileFromBytes(contents, "image.bin")
				if fileErr != nil {
					return fileErr
				}
				record.Set("image_file", stored)
				record.Set("image_metadata", metadata)
			}
			if saveErr := txApp.Save(record); saveErr != nil {
				return saveErr
			}
			saved = record
			return nil
		})
		if err != nil {
			if errors.Is(err, errV2Conflict) {
				return e.JSON(http.StatusConflict, map[string]string{"error": "Note already exists"})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to import note"})
		}
		publishV2Update(app, lookupHash, saved)
		return e.JSON(http.StatusCreated, v2Response(saved))
	})

	api.POST("/v2/notes/image", func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !generalLimiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		lookupHash, err := v2LookupHash(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := parseLimitedMultipart(e); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid encrypted image"})
		}
		metadata := e.Request.FormValue("metadata")
		version, _ := strconv.Atoi(e.Request.FormValue("version"))
		if !validV2Envelope(metadata, v2MaxMetadata) {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid encrypted image metadata"})
		}
		file, _, err := e.Request.FormFile("image")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Encrypted image is required"})
		}
		defer file.Close()
		contents, err := io.ReadAll(io.LimitReader(file, v2MaxImageBytes+1))
		if err != nil || len(contents) > v2MaxImageBytes {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Encrypted image exceeds limit"})
		}
		stored, err := filesystem.NewFileFromBytes(contents, "image.bin")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid encrypted image"})
		}

		var saved *core.Record
		err = e.App.RunInTransaction(func(txApp core.App) error {
			record, findErr := findV2Note(txApp, lookupHash)
			if findErr != nil || record == nil {
				return fmt.Errorf("note not found")
			}
			current := int(record.GetInt("version"))
			if version > 0 && version != current {
				return fmt.Errorf("%w:%d", errV2Conflict, current)
			}
			record.Set("image_file", stored)
			record.Set("image_metadata", metadata)
			record.Set("version", current+1)
			if saveErr := txApp.Save(record); saveErr != nil {
				return saveErr
			}
			saved = record
			return nil
		})
		if err != nil {
			if errors.Is(err, errV2Conflict) {
				return e.JSON(http.StatusConflict, map[string]string{"error": "version_conflict"})
			}
			return e.JSON(http.StatusNotFound, map[string]string{"error": "Note not found"})
		}
		publishV2Update(app, lookupHash, saved)
		return e.JSON(http.StatusOK, v2Response(saved))
	})

	api.GET("/v2/notes/image", func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !generalLimiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		lookupHash, err := v2LookupHash(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		record, err := findV2Note(e.App, lookupHash)
		if err != nil || record == nil || record.GetString("image_file") == "" {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
		}
		return writeRecordFile(e, record, "image_file", record.GetString("image_metadata"), "")
	})

	api.DELETE("/v2/notes/image", func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !generalLimiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		lookupHash, err := v2LookupHash(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		data := struct {
			Version int `json:"version"`
		}{}
		if err := bindLimitedJSON(e, &data); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		}
		var saved *core.Record
		err = e.App.RunInTransaction(func(txApp core.App) error {
			record, findErr := findV2Note(txApp, lookupHash)
			if findErr != nil || record == nil {
				return fmt.Errorf("note not found")
			}
			current := int(record.GetInt("version"))
			if data.Version > 0 && data.Version != current {
				return fmt.Errorf("%w:%d", errV2Conflict, current)
			}
			record.Set("image_file", "")
			record.Set("image_metadata", "")
			record.Set("version", current+1)
			if saveErr := txApp.Save(record); saveErr != nil {
				return saveErr
			}
			saved = record
			return nil
		})
		if err != nil {
			if errors.Is(err, errV2Conflict) {
				return e.JSON(http.StatusConflict, map[string]string{"error": "version_conflict"})
			}
			return e.JSON(http.StatusNotFound, map[string]string{"error": "Note not found"})
		}
		publishV2Update(app, lookupHash, saved)
		return e.JSON(http.StatusOK, v2Response(saved))
	})

	api.GET("/v2/recovery", func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !recoveryLimiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		legacyHash, err := legacyLookupHash(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		records, err := e.App.FindRecordsByFilter("notes", "phrase_hash = {:hash}", "", 1, 0, dbx.Params{"hash": legacyHash})
		if err != nil || len(records) == 0 {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "Legacy note not found"})
		}
		record := records[0]
		return e.JSON(http.StatusOK, map[string]any{
			"ciphertext": record.GetString("message"),
			"hasImage":   record.GetString("image_hash") != "",
		})
	})

	api.GET("/v2/recovery/image", func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !recoveryLimiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		legacyHash, err := legacyLookupHash(e)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		records, err := e.App.FindRecordsByFilter("encrypted_files", "phrase_hash = {:hash}", "", 1, 0, dbx.Params{"hash": legacyHash})
		if err != nil || len(records) == 0 {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "Legacy image not found"})
		}
		record := records[0]
		return writeRecordFile(e, record, "file_data", record.GetString("file_name"), record.GetString("content_type"))
	})

	app.OnRealtimeSubscribeRequest().BindFunc(func(e *core.RealtimeSubscribeRequestEvent) error {
		for _, rawSubscription := range e.Subscriptions {
			u, err := url.Parse(rawSubscription)
			if err != nil || !strings.HasPrefix(u.Path, v2RealtimePrefix) {
				continue
			}
			lookupHash := strings.TrimPrefix(u.Path, v2RealtimePrefix)
			if !hexHashRE.MatchString(lookupHash) {
				return e.ForbiddenError("Invalid encrypted note subscription", nil)
			}
			options := struct {
				Headers map[string]any `json:"headers"`
			}{}
			if err := json.Unmarshal([]byte(u.Query().Get("options")), &options); err != nil {
				return e.ForbiddenError("Encrypted note subscription requires a lookup token", nil)
			}
			var token string
			for key, value := range options.Headers {
				if strings.EqualFold(strings.ReplaceAll(key, "_", "-"), v2TokenHeader) {
					token, _ = value.(string)
				}
			}
			decoded, decodeErr := base64.RawURLEncoding.DecodeString(token)
			if decodeErr != nil || len(decoded) != 32 || sha256Hex(decoded) != lookupHash {
				return e.ForbiddenError("Invalid encrypted note subscription", nil)
			}
			record, findErr := findV2Note(e.App, lookupHash)
			if findErr != nil || record == nil {
				return e.ForbiddenError("Encrypted note not found", nil)
			}
		}
		return e.Next()
	})
}

func bindLimitedJSON(e *core.RequestEvent, destination any) error {
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, v2MaxJSONBody)
	return e.BindBody(destination)
}

func parseLimitedMultipart(e *core.RequestEvent) error {
	e.Request.Body = http.MaxBytesReader(e.Response, e.Request.Body, v2MaxMultipartBody)
	return e.Request.ParseMultipartForm(1 << 20)
}

func preparePrivateResponse(e *core.RequestEvent) {
	e.Response.Header().Set("Cache-Control", "no-store, max-age=0")
	e.Response.Header().Set("Pragma", "no-cache")
	e.Response.Header().Set("X-Content-Type-Options", "nosniff")
}

func rateLimited(e *core.RequestEvent) error {
	e.Response.Header().Set("Retry-After", "60")
	return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "Too many requests"})
}

func v2LookupHash(e *core.RequestEvent) (string, error) {
	token := e.Request.Header.Get(v2TokenHeader)
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return "", fmt.Errorf("invalid lookup token")
	}
	return sha256Hex(decoded), nil
}

func legacyLookupHash(e *core.RequestEvent) (string, error) {
	lookupHash := e.Request.Header.Get(legacyHashHeader)
	if !hexHashRE.MatchString(lookupHash) {
		return "", fmt.Errorf("invalid legacy lookup hash")
	}
	return lookupHash, nil
}

func sha256Hex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func findV2Note(app core.App, lookupHash string) (*core.Record, error) {
	records, err := app.FindRecordsByFilter(v2Collection, "lookup_hash = {:hash}", "", 1, 0, dbx.Params{"hash": lookupHash})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

func validV2Envelope(raw string, maxLength int) bool {
	if raw == "" || len(raw) > maxLength {
		return false
	}
	envelope := v2Envelope{}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil || envelope.Version != 2 {
		return false
	}
	nonce, nonceErr := base64.RawURLEncoding.DecodeString(envelope.Nonce)
	ciphertext, ciphertextErr := base64.RawURLEncoding.DecodeString(envelope.Ciphertext)
	return nonceErr == nil && len(nonce) == 12 && ciphertextErr == nil && len(ciphertext) >= 16
}

func v2Response(record *core.Record) v2NoteResponse {
	return v2NoteResponse{
		ID:         record.Id,
		Ciphertext: record.GetString("ciphertext"),
		HasImage:   record.GetString("image_file") != "",
		Version:    int(record.GetInt("version")),
		Created:    record.GetDateTime("created").String(),
		Updated:    record.GetDateTime("updated").String(),
	}
}

func writeRecordFile(e *core.RequestEvent, record *core.Record, fieldName, metadata, legacyContentType string) error {
	storedFilename := record.GetString(fieldName)
	if storedFilename == "" {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}
	fs, err := e.App.NewFilesystem()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load image"})
	}
	defer fs.Close()
	reader, err := fs.GetReader(record.BaseFilesPath() + "/" + storedFilename)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}
	defer reader.Close()

	e.Response.Header().Set("Content-Type", "application/octet-stream")
	e.Response.Header().Set("Content-Disposition", "attachment; filename=\"image.bin\"")
	e.Response.Header().Set("X-Encrypted-Metadata", metadata)
	if legacyContentType != "" {
		e.Response.Header().Set("X-Legacy-Content-Type", legacyContentType)
	}
	_, err = io.Copy(e.Response, reader)
	return err
}

func publishV2Update(app *pocketbase.PocketBase, lookupHash string, record *core.Record) {
	topic := v2RealtimePrefix + lookupHash
	payload, err := json.Marshal(map[string]any{
		"id":      record.Id,
		"version": record.GetInt("version"),
		"updated": record.GetDateTime("updated").String(),
	})
	if err != nil {
		return
	}
	for _, client := range app.SubscriptionsBroker().Clients() {
		for subscription := range client.Subscriptions(topic) {
			u, parseErr := url.Parse(subscription)
			if parseErr == nil && u.Path == topic {
				client.Send(subscriptions.Message{Name: topic, Data: payload})
				break
			}
		}
	}
}

func upgradeRequired(e *core.RequestEvent) error {
	preparePrivateResponse(e)
	return e.JSON(http.StatusUpgradeRequired, map[string]string{
		"error": "This version of Secret Notez must be updated to protect your passphrase.",
		"code":  "client_update_required",
	})
}
