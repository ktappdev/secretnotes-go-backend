package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ktappdev/secretnotes-go-backend/internal/e2ee"
	"github.com/ktappdev/secretnotes-go-backend/services"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
)

func TestValidV2Envelope(t *testing.T) {
	valid := `{"v":2,"n":"AAAAAAAAAAAAAAAA","c":"AAAAAAAAAAAAAAAAAAAAAA"}`
	if !validV2Envelope(valid, len(valid)) {
		t.Fatal("expected valid envelope")
	}

	invalid := []string{
		`{"v":1,"n":"AAAAAAAAAAAAAAAA","c":"AAAAAAAAAAAAAAAAAAAAAA"}`,
		`{"v":2,"n":"short","c":"AAAAAAAAAAAAAAAAAAAAAA"}`,
		`{"v":2,"n":"AAAAAAAAAAAAAAAA","c":"short"}`,
		valid + "x",
	}
	for _, value := range invalid {
		if validV2Envelope(value, len(valid)) {
			t.Fatalf("accepted invalid envelope: %s", value)
		}
	}
}

func TestLookupTokenEncoding(t *testing.T) {
	token := make([]byte, 32)
	encoded := base64.RawURLEncoding.EncodeToString(token)
	if encoded != "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" {
		t.Fatalf("unexpected token encoding: %s", encoded)
	}
	if got := sha256Hex(token); got != "66687aadf862bd776c8fc18b8e9f8e20089714856ee233b3902a591d0d5f2925" {
		t.Fatalf("unexpected lookup hash: %s", got)
	}
}

func TestFixedWindowLimiter(t *testing.T) {
	limiter := newFixedWindowLimiter(2, time.Minute)
	if !limiter.allow("client") || !limiter.allow("client") {
		t.Fatal("expected requests within limit to pass")
	}
	if limiter.allow("client") {
		t.Fatal("expected request over limit to fail")
	}
}

func TestV2NoteAndImageHTTPRoutes(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	pbRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	registerV2Routes(pbRouter.Group("/api/secretnotes"), app)
	mux, err := pbRouter.BuildMux()
	if err != nil {
		t.Fatal(err)
	}

	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	envelope := `{"v":2,"n":"AAAAAAAAAAAAAAAA","c":"AAAAAAAAAAAAAAAAAAAAAA"}`
	do := func(method, path, contentType string, body io.Reader) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, body)
		request.Header.Set(v2TokenHeader, token)
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}

	created := do(http.MethodPost, "/api/secretnotes/v2/notes", "application/json", bytes.NewBufferString(`{"ciphertext":`+strconvQuote(envelope)+`}`))
	if created.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", created.Code, created.Body.String())
	}
	duplicate := do(http.MethodPost, "/api/secretnotes/v2/notes", "application/json", bytes.NewBufferString(`{"ciphertext":`+strconvQuote(envelope)+`}`))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate create status %d: %s", duplicate.Code, duplicate.Body.String())
	}
	loaded := do(http.MethodGet, "/api/secretnotes/v2/notes", "", nil)
	var loadedNote v2NoteResponse
	decodeErr := json.Unmarshal(loaded.Body.Bytes(), &loadedNote)
	if loaded.Code != http.StatusOK || decodeErr != nil || loadedNote.Ciphertext != envelope {
		t.Fatalf("load failed %d: %s", loaded.Code, loaded.Body.String())
	}

	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	part, err := writer.CreateFormFile("image", "image.bin")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte(envelope))
	_ = writer.WriteField("image_metadata", envelope)
	_ = writer.WriteField("version", "1")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	uploaded := do(http.MethodPost, "/api/secretnotes/v2/notes/image", writer.FormDataContentType(), &form)
	if uploaded.Code != http.StatusOK {
		t.Fatalf("image upload status %d: %s", uploaded.Code, uploaded.Body.String())
	}
	image := do(http.MethodGet, "/api/secretnotes/v2/notes/image", "", nil)
	if image.Code != http.StatusOK || image.Header().Get("X-Encrypted-Metadata") != envelope {
		t.Fatalf("image download failed %d: headers=%v body=%s", image.Code, image.Header(), image.Body.String())
	}
	if !strings.Contains(image.Header().Get("Access-Control-Expose-Headers"), "X-Encrypted-Metadata") {
		t.Fatal("encrypted metadata header is not exposed to web clients")
	}

	missingVersion := do(http.MethodPut, "/api/secretnotes/v2/notes", "application/json", bytes.NewBufferString(`{"ciphertext":`+strconvQuote(envelope)+`}`))
	if missingVersion.Code != http.StatusConflict {
		t.Fatalf("missing version status %d: %s", missingVersion.Code, missingVersion.Body.String())
	}
}

func TestV2RealtimeSubscriptionRequiresMatchingToken(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	pbRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	registerV2Routes(pbRouter.Group("/api/secretnotes"), app)

	tokenBytes := make([]byte, 32)
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	lookupSum := sha256.Sum256(tokenBytes)
	topic := v2RealtimePrefix + hex.EncodeToString(lookupSum[:])

	collection, err := app.FindCollectionByNameOrId(v2Collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("lookup_hash", hex.EncodeToString(lookupSum[:]))
	record.Set("ciphertext", `{"v":2,"n":"AAAAAAAAAAAAAAAA","c":"AAAAAAAAAAAAAAAAAAAAAA"}`)
	record.Set("version", 1)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}

	trigger := func(subscription string) error {
		request := httptest.NewRequest(http.MethodPost, "/api/realtime", nil)
		response := httptest.NewRecorder()
		event := &core.RealtimeSubscribeRequestEvent{
			RequestEvent:  &core.RequestEvent{App: app, Event: router.Event{Request: request, Response: response}},
			Subscriptions: []string{subscription},
		}
		return app.OnRealtimeSubscribeRequest().Trigger(event, func(event *core.RealtimeSubscribeRequestEvent) error {
			return nil
		})
	}
	if err := trigger(topic); err == nil {
		t.Fatal("anonymous encrypted subscription was accepted")
	}
	options, _ := json.Marshal(map[string]any{"headers": map[string]any{v2TokenHeader: token}})
	subscription := topic + "?options=" + url.QueryEscape(string(options))
	if err := trigger(subscription); err != nil {
		t.Fatalf("matching encrypted subscription was rejected: %v", err)
	}
	client := subscriptions.NewDefaultClient()
	client.Subscribe(subscription)
	app.SubscriptionsBroker().Register(client)
	defer app.SubscriptionsBroker().Unregister(client.Id())
	received := make(chan subscriptions.Message, 1)
	go func() {
		received <- <-client.Channel()
	}()
	publishV2Update(app, hex.EncodeToString(lookupSum[:]), record)
	select {
	case message := <-received:
		if message.Name != subscription || !strings.Contains(string(message.Data), `"version":1`) {
			t.Fatalf("unexpected realtime message: %#v", message)
		}
	case <-time.After(time.Second):
		t.Fatal("encrypted note update was not published")
	}
}

func TestLegacyDeadlines(t *testing.T) {
	t.Setenv("SECRETNOTEZ_V1_MODE", "compat")
	t.Setenv("SECRETNOTEZ_V1_DEADLINE", "2026-10-01T00:00:00Z")
	if !legacyV1Available(time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)) || legacyV1Available(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("legacy v1 deadline was not enforced")
	}
	t.Setenv("SECRETNOTEZ_V1_MODE", "disabled")
	if legacyV1Available(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("disabled legacy v1 mode was accepted")
	}
	t.Setenv("SECRETNOTEZ_RECOVERY_DEADLINE", "2027-01-01T00:00:00Z")
	if legacyRecoveryAvailable(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("legacy recovery deadline was not enforced")
	}
	t.Setenv("SECRETNOTEZ_RECOVERY_DEADLINE", "2030-01-01T00:00:00Z")
	if legacyRecoveryAvailable(time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("legacy recovery deadline override extended the maximum window")
	}
}

func TestLegacyV1CompatibilityHTTPRoute(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:       t.TempDir(),
		DefaultEncryptionEnv: "pb_test_env",
		HideStartBanner:      true,
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	defer app.ResetBootstrapState()
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	pbRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	encryption := services.NewEncryptionService()
	registerLegacyV1Routes(
		pbRouter.Group("/api/secretnotes"),
		services.NewNoteService(app, encryption),
		services.NewFileService(app, encryption),
	)
	mux, err := pbRouter.BuildMux()
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("SECRETNOTEZ_V1_MODE", "compat")
	t.Setenv("SECRETNOTEZ_V1_DEADLINE", "2099-01-01T00:00:00Z")
	request := httptest.NewRequest(http.MethodGet, "/api/secretnotes/notes", nil)
	request.Header.Set("X-Passphrase", "compatibility-test-note")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusCreated && response.Code != http.StatusOK {
		t.Fatalf("legacy compatibility route status %d: %s", response.Code, response.Body.String())
	}

	t.Setenv("SECRETNOTEZ_V1_MODE", "disabled")
	request = httptest.NewRequest(http.MethodGet, "/api/secretnotes/notes", nil)
	request.Header.Set("X-Passphrase", "compatibility-test-note")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUpgradeRequired {
		t.Fatalf("disabled legacy route status %d: %s", response.Code, response.Body.String())
	}

	// A v2 note for the same passphrase must stop all access to the stale copy.
	t.Setenv("SECRETNOTEZ_V1_MODE", "compat")
	keys, err := e2ee.DeriveKeys([]byte("compatibility-test-note"))
	if err != nil {
		t.Fatal(err)
	}
	defer keys.Clear()
	collection, err := app.FindCollectionByNameOrId(v2Collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("lookup_hash", sha256Hex(keys.LookupToken))
	record.Set("ciphertext", `{"v":2,"n":"AAAAAAAAAAAAAAAA","c":"AAAAAAAAAAAAAAAAAAAAAA"}`)
	record.Set("version", 1)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/notes"},
		{http.MethodPost, "/notes"},
		{http.MethodPut, "/notes"},
		{http.MethodPatch, "/notes"},
		{http.MethodGet, "/notes/image"},
		{http.MethodPost, "/notes/image"},
		{http.MethodDelete, "/notes/image"},
	} {
		request := httptest.NewRequest(route.method, "/api/secretnotes"+route.path,
			strings.NewReader(`{"passphrase":"compatibility-test-note","message":"stale edit","version":1}`))
		request.Header.Set("Content-Type", "application/json")
		// Exercise JSON passphrase fallback as well as header-based clients.
		if route.method == http.MethodGet || strings.HasSuffix(route.path, "/image") {
			request.Header.Set("X-Passphrase", "compatibility-test-note")
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusUpgradeRequired || !strings.Contains(response.Body.String(), "note_requires_v2") {
			t.Fatalf("migrated legacy %s %s status %d: %s", route.method, route.path, response.Code, response.Body.String())
		}
	}
	legacy, err := app.FindFirstRecordByFilter("notes", "phrase_hash = {:hash}",
		dbx.Params{"hash": hashPhrase("compatibility-test-note")})
	if err != nil {
		t.Fatal(err)
	}
	if legacy.GetInt("version") != 1 {
		t.Fatal("rejected legacy requests changed the old note")
	}
}

func strconvQuote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
