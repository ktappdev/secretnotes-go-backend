package e2ee

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const (
	scryptN = 65536
	scryptR = 8
	scryptP = 2
)

var protocolSalt = []byte("secretnotez/e2ee/v2")

type Keys struct {
	LookupToken   []byte
	Note          []byte
	Image         []byte
	ImageMetadata []byte
}

type envelope struct {
	Version    int    `json:"v"`
	Nonce      string `json:"n"`
	Ciphertext string `json:"c"`
}

func DeriveKeys(passphrase []byte) (*Keys, error) {
	root, err := scrypt.Key(passphrase, protocolSalt, scryptN, scryptR, scryptP, 32)
	if err != nil {
		return nil, fmt.Errorf("derive root key: %w", err)
	}
	defer zero(root)
	derive := func(label string) ([]byte, error) {
		key := make([]byte, 32)
		_, err := io.ReadFull(hkdf.New(sha256.New, root, protocolSalt, []byte(label)), key)
		return key, err
	}
	lookup, err := derive("secretnotez:v2:lookup")
	if err != nil {
		return nil, err
	}
	note, err := derive("secretnotez:v2:note")
	if err != nil {
		return nil, err
	}
	image, err := derive("secretnotez:v2:image")
	if err != nil {
		return nil, err
	}
	metadata, err := derive("secretnotez:v2:image-metadata")
	if err != nil {
		return nil, err
	}
	return &Keys{LookupToken: lookup, Note: note, Image: image, ImageMetadata: metadata}, nil
}

func (k *Keys) Clear() {
	zero(k.LookupToken)
	zero(k.Note)
	zero(k.Image)
	zero(k.ImageMetadata)
}

func (k *Keys) LookupHeader() string {
	return base64.RawURLEncoding.EncodeToString(k.LookupToken)
}

func Encrypt(plaintext, key []byte, purpose string) (string, error) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, []byte(purpose))
	encoded, err := json.Marshal(envelope{
		Version:    2,
		Nonce:      base64.RawURLEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawURLEncoding.EncodeToString(ciphertext),
	})
	return string(encoded), err
}

func Decrypt(encoded string, key []byte, purpose string) ([]byte, error) {
	var value envelope
	if err := json.Unmarshal([]byte(encoded), &value); err != nil || value.Version != 2 {
		return nil, errors.New("unsupported encrypted content")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(value.Nonce)
	if err != nil || len(nonce) != 12 {
		return nil, errors.New("malformed encrypted content")
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(value.Ciphertext)
	if err != nil || len(ciphertext) < 16 {
		return nil, errors.New("malformed encrypted content")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(purpose))
	if err != nil {
		return nil, errors.New("encrypted content could not be authenticated")
	}
	return plaintext, nil
}

func LegacyLookupHash(passphrase []byte) string {
	sum := sha256.Sum256(passphrase)
	return hex.EncodeToString(sum[:])
}

func DecryptLegacyBase64(encoded string, passphrase []byte) ([]byte, error) {
	value, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("legacy content is not valid base64")
	}
	return DecryptLegacy(value, passphrase)
}

func DecryptLegacy(encrypted, passphrase []byte) ([]byte, error) {
	if len(encrypted) < 44 {
		return nil, errors.New("legacy encrypted content is malformed")
	}
	salt, nonce, ciphertext := encrypted[:16], encrypted[16:28], encrypted[28:]
	key := pbkdf2.Key(passphrase, salt, 10000, 32, sha256.New)
	defer zero(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("legacy content could not be authenticated")
	}
	return plaintext, nil
}

func zero(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
