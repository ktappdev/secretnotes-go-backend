package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ktappdev/secretnotes-go-backend/cli/internal/e2ee"
)

var (
	ErrNoteNotFound      = errors.New("encrypted note not found")
	ErrNoteAlreadyExists = errors.New("encrypted note already exists")
)

const (
	notePurpose     = "secretnotez:v2:note"
	imagePurpose    = "secretnotez:v2:image"
	metadataPurpose = "secretnotez:v2:image-metadata"
)

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

type legacyNote struct {
	Ciphertext string `json:"ciphertext"`
	HasImage   bool   `json:"hasImage"`
}

type imageMetadata struct {
	FileName    string `json:"fileName"`
	FileSize    int    `json:"fileSize"`
	ContentType string `json:"contentType"`
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

func (c *Client) RecoverNote(ctx context.Context, passphrase []byte) (*Note, error) {
	keys, err := c.sessionKeys(passphrase)
	if err != nil {
		return nil, err
	}
	defer keys.Clear()
	hash := e2ee.LegacyLookupHash(passphrase)
	legacy, err := c.getLegacyNote(ctx, hash)
	if err != nil {
		return nil, err
	}
	message, err := e2ee.DecryptLegacyBase64(legacy.Ciphertext, passphrase)
	if err != nil {
		return nil, err
	}
	ciphertext, err := e2ee.Encrypt(message, keys.Note, notePurpose)
	if err != nil {
		return nil, err
	}

	var image []byte
	var metadata string
	if legacy.HasImage {
		legacyImage, filenameCiphertext, contentType, imageErr := c.getLegacyImage(ctx, hash)
		if imageErr != nil {
			return nil, imageErr
		}
		plaintextImage, imageErr := e2ee.DecryptLegacy(legacyImage, passphrase)
		if imageErr != nil {
			return nil, imageErr
		}
		filename, imageErr := e2ee.DecryptLegacyBase64(filenameCiphertext, passphrase)
		if imageErr != nil {
			return nil, imageErr
		}
		imageEnvelope, imageErr := e2ee.Encrypt(plaintextImage, keys.Image, imagePurpose)
		if imageErr != nil {
			return nil, imageErr
		}
		image = []byte(imageEnvelope)
		metadataValue, _ := json.Marshal(imageMetadata{
			FileName: string(filename), FileSize: len(plaintextImage), ContentType: contentType,
		})
		metadata, imageErr = e2ee.Encrypt(metadataValue, keys.ImageMetadata, metadataPurpose)
		if imageErr != nil {
			return nil, imageErr
		}
	}
	imported, err := c.importNote(ctx, keys, ciphertext, image, metadata)
	if err != nil && !errors.Is(err, ErrNoteAlreadyExists) {
		return nil, err
	}
	verified, err := c.getEncryptedNote(ctx, keys)
	if err != nil {
		return nil, err
	}
	note, err := decryptNote(verified, keys)
	if err != nil {
		return nil, err
	}
	if (imported != nil && note.ID != imported.ID) || note.Message != string(message) || note.HasImage != legacy.HasImage {
		return nil, errors.New("recovered note verification failed")
	}
	if legacy.HasImage {
		if err := c.verifyImage(ctx, keys); err != nil {
			return nil, err
		}
	}
	return note, nil
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

func (c *Client) getLegacyNote(ctx context.Context, hash string) (*legacyNote, error) {
	req, err := c.request(ctx, http.MethodGet, "/api/secretnotes/v2/recovery", nil, "")
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Legacy-Lookup-Hash", hash)
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, ErrNoteNotFound
	}
	if res.StatusCode != http.StatusOK {
		return nil, responseError("recover note", res)
	}
	var result legacyNote
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) getLegacyImage(ctx context.Context, hash string) ([]byte, string, string, error) {
	req, err := c.request(ctx, http.MethodGet, "/api/secretnotes/v2/recovery/image", nil, "")
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Set("X-Legacy-Lookup-Hash", hash)
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, "", "", responseError("recover image", res)
	}
	value, err := io.ReadAll(io.LimitReader(res.Body, 15<<20+1))
	if err != nil || len(value) > 15<<20 {
		return nil, "", "", errors.New("legacy image exceeds limit")
	}
	return value, res.Header.Get("X-Encrypted-Metadata"), res.Header.Get("X-Legacy-Content-Type"), nil
}

func (c *Client) importNote(ctx context.Context, keys *e2ee.Keys, ciphertext string, image []byte, metadata string) (*encryptedNote, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("ciphertext", ciphertext); err != nil {
		return nil, err
	}
	if len(image) > 0 {
		part, err := writer.CreateFormFile("image", "image.bin")
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(image); err != nil {
			return nil, err
		}
		if err := writer.WriteField("image_metadata", metadata); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req, err := c.request(ctx, http.MethodPost, "/api/secretnotes/v2/import", &body, keys.LookupHeader())
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusConflict {
		return nil, ErrNoteAlreadyExists
	}
	if res.StatusCode != http.StatusCreated {
		return nil, responseError("import note", res)
	}
	return decodeEncryptedNote(res.Body)
}

func (c *Client) verifyImage(ctx context.Context, keys *e2ee.Keys) error {
	req, err := c.request(ctx, http.MethodGet, "/api/secretnotes/v2/notes/image", nil, keys.LookupHeader())
	if err != nil {
		return err
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return responseError("verify image", res)
	}
	encrypted, err := io.ReadAll(io.LimitReader(res.Body, 15<<20+1))
	if err != nil || len(encrypted) > 15<<20 {
		return errors.New("encrypted image exceeds limit")
	}
	if _, err := e2ee.Decrypt(string(encrypted), keys.Image, imagePurpose); err != nil {
		return err
	}
	_, err = e2ee.Decrypt(res.Header.Get("X-Encrypted-Metadata"), keys.ImageMetadata, metadataPurpose)
	return err
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
