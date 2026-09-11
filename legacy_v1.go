package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ktappdev/secretnotes-go-backend/services"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

const defaultLegacyV1Deadline = "2026-12-31T23:59:59Z"

func registerLegacyV1Routes(
	api *router.RouterGroup[*core.RequestEvent],
	noteService *services.NoteService,
	fileService *services.FileService,
) {
	limiter := newFixedWindowLimiter(120, time.Minute)
	api.GET("/notes", legacyV1Route(limiter, func(e *core.RequestEvent) error {
		phrase, err := extractPassphrase(e, "")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return handleGetOrCreateNote(e, phrase, noteService)
	}))

	api.POST("/notes", legacyV1Route(limiter, func(e *core.RequestEvent) error {
		data := struct {
			Passphrase string `json:"passphrase"`
		}{}
		_ = e.BindBody(&data)
		phrase, err := extractPassphrase(e, data.Passphrase)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return handleGetOrCreateNote(e, phrase, noteService)
	}))

	api.PUT("/notes", legacyV1Route(limiter, func(e *core.RequestEvent) error {
		data := struct {
			Passphrase string `json:"passphrase"`
			Message    string `json:"message"`
			Version    int    `json:"version"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}
		phrase, err := extractPassphrase(e, data.Passphrase)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return handleUpsertNoteWithMessage(e, phrase, data.Message, data.Version, noteService)
	}))

	api.PATCH("/notes", legacyV1Route(limiter, func(e *core.RequestEvent) error {
		data := struct {
			Passphrase string `json:"passphrase"`
			Message    string `json:"message"`
			Version    int    `json:"version"`
		}{}
		if err := e.BindBody(&data); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}
		phrase, err := extractPassphrase(e, data.Passphrase)
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		note, err := noteService.UpdateNote(phrase, data.Message, data.Version)
		if err != nil {
			if errors.Is(err, services.ErrVersionConflict) {
				return e.JSON(http.StatusConflict, map[string]any{
					"error": "version_conflict", "currentVersion": extractCurrentVersion(err),
				})
			}
			return e.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]any{
			"id": note.ID, "message": note.Message, "hasImage": note.ImageHash != "",
			"version": note.Version, "created": note.Created, "updated": note.Updated,
		})
	}))

	api.POST("/notes/image", legacyV1Route(limiter, func(e *core.RequestEvent) error {
		phrase, err := extractPassphrase(e, "")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return handleUploadImage(e, phrase, noteService, fileService)
	}))
	api.GET("/notes/image", legacyV1Route(limiter, func(e *core.RequestEvent) error {
		phrase, err := extractPassphrase(e, "")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return handleGetImage(e, phrase, fileService)
	}))
	api.DELETE("/notes/image", legacyV1Route(limiter, func(e *core.RequestEvent) error {
		phrase, err := extractPassphrase(e, "")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return handleDeleteImage(e, phrase, noteService, fileService)
	}))
}

func legacyV1Route(limiter *fixedWindowLimiter, next func(*core.RequestEvent) error) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		preparePrivateResponse(e)
		if !legacyV1Available(time.Now().UTC()) {
			return upgradeRequired(e)
		}
		if !limiter.allow(e.RealIP()) {
			return rateLimited(e)
		}
		return next(e)
	}
}

func legacyV1Available(now time.Time) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("SECRETNOTEZ_V1_MODE")), "disabled") {
		return false
	}
	return beforeConfiguredDeadline(now, "SECRETNOTEZ_V1_DEADLINE", defaultLegacyV1Deadline)
}

func beforeConfiguredDeadline(now time.Time, envName, maximum string) bool {
	maximumDeadline, err := time.Parse(time.RFC3339, maximum)
	if err != nil {
		return false
	}
	deadline := maximumDeadline
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		configured, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			return false
		}
		if configured.Before(maximumDeadline) {
			deadline = configured
		}
	}
	return now.Before(deadline)
}
