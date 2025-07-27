package models

// EncryptedFile represents the structure for encrypted file metadata
// This is used for reference only - PocketBase handles the actual data storage
type EncryptedFile struct {
	ID          string `json:"id"`
	PhraseHash  string `json:"phrase_hash"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	FileData    string `json:"file_data"` // PocketBase file field
	Created     string `json:"created"`
	Updated     string `json:"updated"`
}
