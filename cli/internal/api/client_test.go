package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ktappdev/secretnotes-go-backend/cli/internal/e2ee"
)

func TestUpdateSendsOnlyTokenAndCiphertext(t *testing.T) {
	passphrase := []byte("exact title 🔐")
	plaintext := "private message 世界"
	keys, err := e2ee.DeriveKeys(passphrase)
	if err != nil {
		t.Fatal(err)
	}
	defer keys.Clear()

	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		wire := request.URL.String() + request.Header.Get("X-Lookup-Token") + string(body)
		if strings.Contains(wire, string(passphrase)) || strings.Contains(wire, plaintext) {
			t.Fatal("plaintext credential or content crossed the API boundary")
		}
		var payload struct {
			Ciphertext string `json:"ciphertext"`
			Version    int    `json:"version"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		decrypted, err := e2ee.Decrypt(payload.Ciphertext, keys.Note, notePurpose)
		if err != nil || string(decrypted) != plaintext {
			t.Fatal("ciphertext did not contain expected note")
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(encryptedNote{
			ID: "note", Ciphertext: payload.Ciphertext, Version: 4,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, false)
	defer client.Close()
	note, err := client.UpdateNote(context.Background(), passphrase, plaintext, 3)
	if err != nil {
		t.Fatal(err)
	}
	if note.Message != plaintext || note.Version != 4 {
		t.Fatalf("unexpected note: %#v", note)
	}
}

func TestMissingNoteDoesNotAttemptLegacyRecovery(t *testing.T) {
	requests := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if strings.Contains(request.URL.Path, "recovery") || request.Header.Get("X-Legacy-Lookup-Hash") != "" {
			t.Fatal("legacy recovery must require an explicit user action")
		}
		http.Error(writer, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, false)
	defer client.Close()
	_, err := client.GetNote(context.Background(), []byte("missing title"))
	if err != ErrNoteNotFound {
		t.Fatalf("expected ErrNoteNotFound, got %v", err)
	}
	if requests != 1 {
		t.Fatalf("expected one v2 request, got %d", requests)
	}
}
