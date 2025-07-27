# Secret Notes Frontend Integration Guide

## Overview

This document provides guidance for frontend developers on how to integrate with the Secret Notes Go backend. The backend offers two approaches for integration:

1. **Direct PocketBase API** - Using PocketBase's JavaScript SDK to interact directly with the collections
2. **Custom API Endpoints** - Using our custom endpoints that handle encryption/decryption

## Base URL

All API endpoints are relative to the base URL where the backend is deployed. For local development, this is typically `http://localhost:8090`.

- Custom API endpoints are prefixed with `/api/secretnotes/`
- Direct PocketBase API endpoints are prefixed with `/api/collections/`

## Authentication

The Secret Notes backend does not use traditional authentication. Instead, all operations are protected by a passphrase that must be provided as part of the URL path or hashed for direct API access. This passphrase is used for both identification and encryption/decryption.

### API Approach Recommendation

**We recommend using the Custom API Endpoints (Option 1)** for the following reasons:

1. **Server-side encryption/decryption**: No need to implement encryption logic in the frontend
2. **Unified encryption**: Both notes and files use the same passphrase-based encryption
3. **Secure file handling**: Files are encrypted at rest and decrypted on-demand
4. **Simplified integration**: Just send passphrases and receive decrypted content
5. **Verified implementation**: The custom endpoints have been thoroughly tested and verified

**Direct PocketBase API (Option 2)** is also available but requires:
- Client-side encryption/decryption implementation
- More complex file handling
- Manual encryption key management

### Passphrase Requirements

- Minimum length: 32 characters
- Should be randomly generated for security
- Never stored on the client
- Sent only in URL paths (over HTTPS in production)

## API Endpoints

All endpoints are prefixed with `/api/secretnotes/`.

### Health Check

**Endpoint**: `GET /api/secretnotes/`

**Description**: Check if the API is running.

**Response**:
```json
{
  "message": "Secret Notes API is live",
  "version": "1.0.0"
}
```

### Create Note

**Endpoint**: `POST /api/secretnotes/notes/{phrase}`

**Description**: Creates a new note with the specified content.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 32 characters)

**Request Body**:
```json
{
  "title": "Note Title",
  "message": "Note content to encrypt"
}
```

**Response**:
```json
{
  "id": "record_id",
  "message": "decrypted_note_content",
  "hasImage": false,
  "created": "",
  "updated": ""
}
```

### Get Note

**Endpoint**: `GET /api/secretnotes/notes/{phrase}`

**Description**: Retrieves an existing note or creates a new one if it doesn't exist.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 32 characters)

**Response**:
```json
{
  "id": "record_id",
  "message": "decrypted_note_content",
  "hasImage": false,
  "created": "",
  "updated": ""
}
```

### Update Note

**Endpoint**: `PATCH /api/secretnotes/notes/{phrase}`

**Description**: Update an existing note with new content.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 32 characters)

**Request Body**:
```json
{
  "message": "new_note_content"
}
```

**Response**:
```json
{
  "id": "note_id",
  "message": "decrypted_note_content",
  "hasImage": true,
  "created": "2023-01-01T00:00:00Z",
  "updated": "2023-01-01T00:00:00Z"
}
```

### Upload Image

**Endpoint**: `POST /api/secretnotes/notes/{phrase}/image`

**Description**: Upload an image associated with a note.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 32 characters)

**Request Body**: Multipart form data with the image file.

**Form Fields**:
- `image`: The image file to upload

**Response**:
```json
{
  "id": "file_id",
  "fileName": "image.jpg",
  "contentType": "image/jpeg",
  "created": "2023-01-01T00:00:00Z"
}
```

### Retrieve Image

**Endpoint**: `GET /api/secretnotes/notes/{phrase}/image`

**Description**: Retrieve an image associated with a note.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 32 characters)

**Response**: The decrypted image file (binary data)

**Content-Type**: The original content type of the image

### Delete Image

**Endpoint**: `DELETE /api/secretnotes/notes/{phrase}/image`

**Description**: Delete an image associated with a note.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 32 characters)

**Response**:
```json
{
  "message": "Image deleted successfully"
}
```

## Error Handling

The API returns standard HTTP status codes:

- `200`: Success
- `400`: Bad request (e.g., missing parameters, invalid passphrase length)
- `404`: Note or image not found
- `500`: Internal server error

Error responses follow this format:
```json
{
  "error": "description of the error"
}
```

## Implementation Examples

### Option 1: Using Custom API Endpoints (with Encryption/Decryption)

#### Creating/Retrieving a Note

```javascript
// Generate a secure passphrase (32+ characters)
const passphrase = generateSecurePassphrase();

// Retrieve or create a note
const response = await fetch(`/api/secretnotes/notes/${passphrase}`);
const note = await response.json();
```

#### Updating a Note

```javascript
const response = await fetch(`/api/secretnotes/notes/${passphrase}`, {
  method: 'PATCH',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    message: 'New secret message',
  }),
});

const updatedNote = await response.json();
```

#### Uploading an Image

```javascript
const formData = new FormData();
formData.append('image', imageFile);

const response = await fetch(`/api/secretnotes/notes/${passphrase}/image`, {
  method: 'POST',
  body: formData,
});

const uploadedImage = await response.json();
```

### Option 2: Using Direct PocketBase API (with PocketBase SDK)

#### Setup

```javascript
import PocketBase from 'pocketbase';

const pb = new PocketBase('http://localhost:8090');

// Generate a secure passphrase (32+ characters) and hash it
const passphrase = generateSecurePassphrase();
const phraseHash = await sha256(passphrase);
```

#### Creating a Note

```javascript
// Create a new note with encrypted message
const encryptedMessage = await encryptWithPassphrase(message, passphrase);

const record = await pb.collection('notes').create({
  phrase_hash: phraseHash,
  message: encryptedMessage,
  image_hash: ""
});
```

#### Retrieving Notes

```javascript
// Fetch notes with matching phrase hash
const records = await pb.collection('notes').getList(1, 50, {
  filter: `phrase_hash='${phraseHash}'`,
});

// Decrypt the message
if (records.items.length > 0) {
  const encryptedMessage = records.items[0].message;
  const decryptedMessage = await decryptWithPassphrase(encryptedMessage, passphrase);
  console.log(decryptedMessage);
}
```

#### Uploading an Image

```javascript
// First encrypt the image data
const encryptedImageData = await encryptWithPassphrase(await imageFile.arrayBuffer(), passphrase);

// Create a FormData object
const formData = new FormData();
formData.append('phrase_hash', phraseHash);
formData.append('file_name', imageFile.name);
formData.append('content_type', imageFile.type);
formData.append('file_data', new Blob([encryptedImageData]), 'encrypted_image');

// Upload the encrypted file
const record = await pb.collection('encrypted_files').create(formData);
```

#### Retrieving an Image

```javascript
// Fetch encrypted files with matching phrase hash
const records = await pb.collection('encrypted_files').getList(1, 1, {
  filter: `phrase_hash='${phraseHash}'`,
});

if (records.items.length > 0) {
  const fileRecord = records.items[0];
  
  // Get the file URL
  const fileUrl = pb.files.getUrl(fileRecord, fileRecord.file_data);
  
  // Fetch the encrypted file
  const response = await fetch(fileUrl);
  const encryptedData = await response.arrayBuffer();
  
  // Decrypt the file data
  const decryptedData = await decryptWithPassphrase(encryptedData, passphrase);
  
  // Create a blob URL for display
  const blob = new Blob([decryptedData], { type: fileRecord.content_type });
  const objectUrl = URL.createObjectURL(blob);
  
  // Display the image
  document.getElementById('image').src = objectUrl;
}
```

### Retrieving an Image

```javascript
const response = await fetch(`/api/secretnotes/notes/${passphrase}/image`);
const imageBlob = await response.blob();
const imageUrl = URL.createObjectURL(imageBlob);
```

### Deleting an Image

```javascript
const response = await fetch(`/api/secretnotes/notes/${passphrase}/image`, {
  method: 'DELETE',
});

const result = await response.json();
```

## Security Considerations

1. **Passphrase Handling**: Never store passphrases in localStorage, sessionStorage, or cookies.
2. **URL Exposure**: Be aware that passphrases will be visible in browser history and server logs.
3. **HTTPS**: Always use HTTPS in production to protect passphrase transmission.
4. **CORS**: The backend should be configured with appropriate CORS headers.
5. **Rate Limiting**: Implement client-side rate limiting to avoid overwhelming the server.

## Best Practices

1. **Passphrase Generation**: Use a cryptographically secure random generator for passphrases.
2. **User Experience**: Provide clear instructions on passphrase management to users.
3. **Error Handling**: Implement graceful error handling for network issues and API errors.
4. **Loading States**: Show loading indicators during API requests.
5. **Validation**: Validate passphrase length on the client before making requests.

## Future Enhancements

1. **WebSockets**: Real-time updates for note changes.
2. **Caching**: Client-side caching of recently accessed notes.
3. **Progressive Web App**: Offline support for viewing existing notes.
4. **Accessibility**: Full accessibility support for all UI components.
