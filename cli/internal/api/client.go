package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL   string
	VerifyTLS bool
	hc        *http.Client
}

type Note struct {
	ID      string      `json:"id"`
	Message string      `json:"message"`
	HasImage bool       `json:"hasImage"`
	Created any         `json:"created"`
	Updated any         `json:"updated"`
}

func NewClient(baseURL string, verifyTLS bool) *Client {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifyTLS}}
	return &Client{
		BaseURL:   trimTrailingSlash(baseURL),
		VerifyTLS: verifyTLS,
		hc: &http.Client{Transport: tr, Timeout: 12 * time.Second},
	}
}

func (c *Client) Health(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/secretnotes/", nil)
	req.Header.Set("User-Agent", "SecretNotes-CLI/1.0")
	res, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("health %d: %s", res.StatusCode, string(b))
	}
	return nil
}

func (c *Client) GetOrCreateNote(ctx context.Context, passphrase []byte) (*Note, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/secretnotes/notes", nil)
	attachHeaders(req, passphrase)
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("get note %d: %s", res.StatusCode, string(b))
	}
	var note Note
	if err := json.NewDecoder(res.Body).Decode(&note); err != nil {
		return nil, err
	}
	return &note, nil
}

func (c *Client) UpdateNote(ctx context.Context, passphrase []byte, message string) (*Note, error) {
	body, _ := json.Marshal(map[string]string{"message": message})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, c.BaseURL+"/api/secretnotes/notes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	attachHeaders(req, passphrase)
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("update note %d: %s", res.StatusCode, string(b))
	}
	var note Note
	if err := json.NewDecoder(res.Body).Decode(&note); err != nil {
		return nil, err
	}
	return &note, nil
}

func attachHeaders(req *http.Request, passphrase []byte) {
	// Construct header string transiently
	req.Header.Set("X-Passphrase", string(passphrase))
	req.Header.Set("User-Agent", "SecretNotes-CLI/1.0")
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}