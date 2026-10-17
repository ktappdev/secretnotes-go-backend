package main

import (
	"net/http"
	"sync"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/ktappdev/secretnotes-go-backend/internal/e2ee"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Each scrypt derivation uses about 64 MiB. Bound simultaneous legacy lookups
// across clients, in addition to the legacy routes' per-IP rate limit.
var legacyLookupSlots = make(chan struct{}, 2)

// Hold a read lock through each legacy operation so v2 creation cannot commit
// between its version check and a write to the old copy.
var legacyTransitionMu sync.RWMutex

func legacyV1Passphrase(e *core.RequestEvent, fallback string) (string, error) {
	phrase, err := extractPassphrase(e, fallback)
	if err != nil {
		return "", apis.NewBadRequestError(err.Error(), nil)
	}
	select {
	case legacyLookupSlots <- struct{}{}:
		defer func() { <-legacyLookupSlots }()
	default:
		e.Response.Header().Set("Retry-After", "1")
		return "", apis.NewApiError(http.StatusTooManyRequests, "Legacy note lookup is busy. Please retry.", nil)
	}

	passphrase := []byte(phrase)
	defer clear(passphrase)
	keys, err := e2ee.DeriveKeys(passphrase)
	if err != nil {
		return "", apis.NewInternalServerError("Failed to check note version", nil)
	}
	defer keys.Clear()
	record, err := findV2Note(e.App, sha256Hex(keys.LookupToken))
	if err != nil {
		return "", apis.NewInternalServerError("Failed to check note version", nil)
	}
	if record != nil {
		return "", apis.NewApiError(http.StatusUpgradeRequired,
			"This note uses end-to-end encryption. Update your Secret Notez app or CLI to open the current note.",
			validation.Errors{"note": validation.NewError("note_requires_v2", "Update your client to access this v2 note.")})
	}
	return phrase, nil
}
