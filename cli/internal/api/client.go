package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ktappdev/secretnotes-go-backend/internal/e2ee"
)

var (
	ErrNoteNotFound      = errors.New("encrypted note not found")
	ErrNoteAlreadyExists = errors.New("encrypted note already exists")
)

const notePurpose = "secretnotez:v2:note"

type Client struct {
	BaseURL    string
	VerifyTLS  bool
	hc         *http.Client
	mu         sync.Mutex
	passphrase []byte
	keys       *e2ee.Keys
}

type Note struct {
	ID       string `json:"id"`
	Message  string `json:"-"`
	HasImage bool   `json:"hasImage"`
	Version  int    `json:"version"`
	Created  string `json:"created"`
	Updated  string `json:"updated"`
}

type encryptedNote struct {
	ID         string `json:"id"`
	Ciphertext string `json:"ciphertext"`
	HasImage   bool   `json:"hasImage"`
	Version    int    `json:"version"`
	Created    string `json:"created"`
	Updated    string `json:"updated"`
}

func NewClient(baseURL string, verifyTLS bool) *Client {
	tr := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: !verifyTLS}}
	return &Client{
		BaseURL: trimTrailingSlash(baseURL), VerifyTLS: verifyTLS,
		hc: &http.Client{Transport: tr, Timeout: 20 * time.Second},
	}
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clearKeys()
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/secretnotes/", nil)
	if err != nil {
		return err
	}
	attachUserAgent(req)
	res, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("health %d: %s", res.StatusCode, string(body))
	}
	return nil
}

func (c *Client) GetNote(ctx context.Context, passphrase []byte) (*Note, error) {
	keys, err := c.sessionKeys(passphrase)
	if err != nil {
		return nil, err
	}
	defer keys.Clear()
	encrypted, err := c.getEncryptedNote(ctx, keys)
	if err != nil {
		return nil, err
	}
	return decryptNote(encrypted, keys)
}

func (c *Client) CreateNote(ctx context.Context, passphrase []byte) (*Note, error) {
	keys, err := c.sessionKeys(passphrase)
	if err != nil {
		return nil, err
	}
	defer keys.Clear()
	ciphertext, err := e2ee.Encrypt(nil, keys.Note, notePurpose)
	if err != nil {
		return nil, err
	}
	encrypted, err := c.postEncryptedNote(ctx, keys, ciphertext)
	if errors.Is(err, ErrNoteAlreadyExists) {
		encrypted, err = c.getEncryptedNote(ctx, keys)
	}
	if err != nil {
		return nil, err
	}
	return decryptNote(encrypted, keys)
}

func (c *Client) UpdateNote(ctx context.Context, passphrase []byte, message string, version int) (*Note, error) {
	keys, err := c.sessionKeys(passphrase)
	if err != nil {
		return nil, err
	}
	defer keys.Clear()
	ciphertext, err := e2ee.Encrypt([]byte(message), keys.Note, notePurpose)
	if err != nil {
		return nil, err
	}
	encrypted, err := c.putEncryptedNote(ctx, keys, ciphertext, version)
	if err != nil {
		return nil, err
	}
	return decryptNote(encrypted, keys)
}

func (c *Client) sessionKeys(passphrase []byte) (*e2ee.Keys, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keys != nil && bytes.Equal(c.passphrase, passphrase) {
		return cloneKeys(c.keys), nil
	}
	c.clearKeys()
	keys, err := e2ee.DeriveKeys(passphrase)
	if err != nil {
		return nil, err
	}
	c.passphrase = bytes.Clone(passphrase)
	c.keys = keys
	return cloneKeys(keys), nil
}

func cloneKeys(keys *e2ee.Keys) *e2ee.Keys {
	return &e2ee.Keys{
		LookupToken:   bytes.Clone(keys.LookupToken),
		Note:          bytes.Clone(keys.Note),
		Image:         bytes.Clone(keys.Image),
		ImageMetadata: bytes.Clone(keys.ImageMetadata),
	}
}

func (c *Client) clearKeys() {
	if c.keys != nil {
		c.keys.Clear()
	}
	for index := range c.passphrase {
		c.passphrase[index] = 0
	}
	c.keys = nil
	c.passphrase = nil
}

func (c *Client) getEncryptedNote(ctx context.Context, keys *e2ee.Keys) (*encryptedNote, error) {
	req, err := c.request(ctx, http.MethodGet, "/api/secretnotes/v2/notes", nil, keys.LookupHeader())
	if err != nil {
		return nil, err
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, ErrNoteNotFound
	}
	if res.StatusCode != http.StatusOK {
		return nil, responseError("get note", res)
	}
	return decodeEncryptedNote(res.Body)
}

func (c *Client) putEncryptedNote(ctx context.Context, keys *e2ee.Keys, ciphertext string, version int) (*encryptedNote, error) {
	body, err := json.Marshal(struct {
		Ciphertext string `json:"ciphertext"`
		Version    int    `json:"version,omitempty"`
	}{ciphertext, version})
	if err != nil {
		return nil, err
	}
	req, err := c.request(ctx, http.MethodPut, "/api/secretnotes/v2/notes", bytes.NewReader(body), keys.LookupHeader())
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, responseError("save note", res)
	}
	return decodeEncryptedNote(res.Body)
}

func (c *Client) postEncryptedNote(ctx context.Context, keys *e2ee.Keys, ciphertext string) (*encryptedNote, error) {
	body, err := json.Marshal(struct {
		Ciphertext string `json:"ciphertext"`
	}{ciphertext})
	if err != nil {
		return nil, err
	}
	req, err := c.request(ctx, http.MethodPost, "/api/secretnotes/v2/notes", bytes.NewReader(body), keys.LookupHeader())
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusConflict {
		return nil, ErrNoteAlreadyExists
	}
	if res.StatusCode != http.StatusCreated {
		return nil, responseError("create note", res)
	}
	return decodeEncryptedNote(res.Body)
}

func (c *Client) request(ctx context.Context, method, path string, body io.Reader, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("X-Lookup-Token", token)
	}
	attachUserAgent(req)
	return req, nil
}

func decryptNote(encrypted *encryptedNote, keys *e2ee.Keys) (*Note, error) {
	message, err := e2ee.Decrypt(encrypted.Ciphertext, keys.Note, notePurpose)
	if err != nil {
		return nil, err
	}
	return &Note{
		ID: encrypted.ID, Message: string(message), HasImage: encrypted.HasImage,
		Version: encrypted.Version, Created: encrypted.Created, Updated: encrypted.Updated,
	}, nil
}

func decodeEncryptedNote(reader io.Reader) (*encryptedNote, error) {
	var note encryptedNote
	if err := json.NewDecoder(io.LimitReader(reader, 400_000)).Decode(&note); err != nil {
		return nil, err
	}
	return &note, nil
}

func responseError(operation string, res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
	return fmt.Errorf("%s %d: %s", operation, res.StatusCode, string(body))
}

func attachUserAgent(req *http.Request)     { req.Header.Set("User-Agent", "SecretNotes-CLI/2.0") }
func trimTrailingSlash(value string) string { return strings.TrimRight(value, "/") }
