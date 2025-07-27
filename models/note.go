package models

// Note represents the structure for note data
// This is used for reference only - PocketBase handles the actual data storage
type Note struct {
	ID        string `json:"id"`
	Phrase    string `json:"phrase"`
	Message   string `json:"message"`
	ImageHash string `json:"image_hash"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`
}
