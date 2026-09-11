package e2ee

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestSharedTypeScriptVectors(t *testing.T) {
	keys, err := DeriveKeys([]byte("correct horse battery staple"))
	if err != nil {
		t.Fatal(err)
	}
	defer keys.Clear()
	if got := keys.LookupHeader(); got != "Mj5MkVeeTDWO8oVLa_lE0vALXTsYHpEbIAf9W4PB3EY" {
		t.Fatalf("lookup vector mismatch: %s", got)
	}
	legacy := "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxGmIL2lY1UoeIkAaBVXHhs7VSire+lfPJLo5HrYTaU="
	plaintext, err := DecryptLegacyBase64(legacy, []byte("legacy-title"))
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "Legacy note ✓" {
		t.Fatalf("legacy vector mismatch: %q", plaintext)
	}
	crossClient := `{"v":2,"n":"AAECAwQFBgcICQoL","c":"raCDT_nEhRPY1BNd2diOtRRKm_PtaHwZ0wJ4EUO9yAMuoHx2BQ"}`
	decrypted, err := Decrypt(crossClient, keys.Note, "secretnotez:v2:note")
	if err != nil || string(decrypted) != "Cross-client note ✓" {
		t.Fatalf("TypeScript ciphertext vector mismatch: %q, %v", decrypted, err)
	}
}

func TestEncryptionRejectsWrongKeyAndTampering(t *testing.T) {
	first, _ := DeriveKeys([]byte("first title"))
	second, _ := DeriveKeys([]byte("second title"))
	defer first.Clear()
	defer second.Clear()
	encoded, err := Encrypt([]byte("Hello, 世界"), first.Note, "secretnotez:v2:note")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(encoded, second.Note, "secretnotez:v2:note"); err == nil {
		t.Fatal("wrong key was accepted")
	}
	var value envelope
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(value.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[0] ^= 1
	value.Ciphertext = base64.RawURLEncoding.EncodeToString(ciphertext)
	tampered, _ := json.Marshal(value)
	if _, err := Decrypt(string(tampered), first.Note, "secretnotez:v2:note"); err == nil {
		t.Fatal("tampered ciphertext was accepted")
	}
	raw, _ := base64.RawURLEncoding.DecodeString(first.LookupHeader())
	raw[0] ^= 1
	if base64.RawURLEncoding.EncodeToString(raw) == first.LookupHeader() {
		t.Fatal("test mutation did not change bytes")
	}
}
