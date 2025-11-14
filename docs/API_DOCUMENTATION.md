# Secret Notes API Documentation

**Version:** 1.0.0
**Base URL:** `http://127.0.0.1:8091/api/secretnotes` (Development)
**Production Base URL:** Your deployed server URL + `/api/secretnotes`

## Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [Security Model](#security-model)
- [Common Workflows](#common-workflows)
- [API Endpoints](#api-endpoints)
- [Data Models](#data-models)
- [Error Handling](#error-handling)
- [Code Examples](#code-examples)
- [Best Practices](#best-practices)

---

## Overview

Secret Notes is a secure, end-to-end encrypted notes application with image attachment support. All note content and images are encrypted using AES-256-GCM encryption before being stored on the server. The server cannot access your data without the passphrase.

### Key Features

- **Zero-Knowledge Encryption**: Server never sees unencrypted data
- **Passphrase-Based Access**: No traditional user accounts; passphrases act as both identifiers and encryption keys
- **Image Support**: Attach encrypted images to notes
- **Public/Private Notes**: Simple passphrases create discoverable "public" notes; complex passphrases create private notes
- **No Registration Required**: Start using immediately with any passphrase

### How It Works

1. **Passphrase = Identity + Encryption Key**: Your passphrase serves dual purposes:
   - **Identity**: SHA-256 hash used to lookup your note in the database
   - **Encryption**: Used to encrypt/decrypt your data via PBKDF2 key derivation

2. **Public vs Private Notes**:
   - **Public Notes**: Simple passphrases like "hello", "test", "public" are easily guessable and act as shared collaborative spaces
   - **Private Notes**: Complex, unique passphrases like "my-secret-diary-2024-xyz" are virtually impossible to guess

3. **Data Storage**:
   - All note content is encrypted with AES-256-GCM before storage
   - Images are encrypted and stored separately
   - Only passphrase hash is stored (SHA-256), never the actual passphrase

---

## Authentication

There is no traditional authentication system. Access is controlled via **passphrases**.

### Passphrase Requirements

- **Minimum Length**: 3 characters
- **No Maximum**: Any length supported
- **Character Set**: Any UTF-8 characters allowed
- **Case Sensitive**: "Hello" and "hello" are different notes

### Providing the Passphrase

Every API request (except health check) requires a passphrase. You can provide it in two ways:

#### Option 1: HTTP Header (Recommended)

```http
X-Passphrase: your-passphrase-here
```

**Pros**: Clean separation, works with GET requests, easier for file uploads
**Cons**: Requires HTTPS in production (passphrases visible in headers)

#### Option 2: JSON Body

```json
{
  "passphrase": "your-passphrase-here"
}
```

**Pros**: More conventional REST approach
**Cons**: Cannot be used with GET requests, more complex with multipart forms

**Priority**: If both header and JSON body are provided, header takes precedence.

---

## Security Model

### Encryption Details

- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Derivation**: PBKDF2 with 10,000 iterations using SHA-256
- **Salt**: Randomly generated per encryption operation
- **Authentication**: GCM mode provides authenticated encryption

### What is Encrypted

- ✅ Note message content
- ✅ Image file data
- ✅ File metadata (filename, content type)

### What is NOT Encrypted

- ❌ Passphrase hash (SHA-256, used for lookup)
- ❌ Image hash reference (SHA-256 of encrypted file)
- ❌ Timestamps (created, updated)
- ❌ Record IDs

### Security Best Practices

1. **Always use HTTPS in production** - Passphrases sent in headers/body must be protected in transit
2. **Strong Passphrases for Private Notes** - Use long, unique passphrases for private notes
3. **Never Log Passphrases** - Don't console.log or store passphrases in your app
4. **Handle Encryption Errors Gracefully** - Wrong passphrase = cannot decrypt, treat as authentication failure
5. **No Passphrase Recovery** - There is no password reset; lost passphrase = lost data

---

## Common Workflows

### Workflow 1: First Time User Creates a Note

```
1. User enters passphrase "my-secret-note-123"
2. App calls GET /api/secretnotes/notes with X-Passphrase header
3. Server creates new note with welcome message (returns 201)
4. User edits the message
5. App calls PATCH /api/secretnotes/notes with new message
6. Note is updated (returns 200)
```

### Workflow 2: Returning User Accesses Existing Note

```
1. User enters passphrase "my-secret-note-123"
2. App calls GET /api/secretnotes/notes with X-Passphrase header
3. Server finds existing note, decrypts, returns message (returns 200)
4. User sees their existing note content
```

### Workflow 3: User Adds Image to Note

```
1. User has existing note with passphrase "my-secret-note-123"
2. User selects image from device
3. App calls POST /api/secretnotes/notes/image with multipart form
4. Server encrypts and stores image (returns 200)
5. To display image later: GET /api/secretnotes/notes/image
6. Server decrypts and returns image file
```

### Workflow 4: Public Note Collaboration

```
1. User A creates note with passphrase "hello"
2. User A writes "Hello World!"
3. User B uses passphrase "hello" (same passphrase)
4. User B sees "Hello World!" (same note)
5. User B edits to "Hello World! - Reply from B"
6. User A refreshes and sees the update
```

---

## API Endpoints

### 1. Health Check

**Endpoint**: `GET /api/secretnotes/`
**Authentication**: None required

**Description**: Verify API is running

**Request Example**:
```http
GET /api/secretnotes/ HTTP/1.1
Host: 127.0.0.1:8091
```

**Response**:
```json
{
  "message": "Secret Notes API is live",
  "version": "1.0.0"
}
```

**Status Codes**:
- `200 OK`: API is operational

---

### 2. Get or Create Note

**Endpoint**: `GET /api/secretnotes/notes`
**Authentication**: Passphrase required

**Description**: Retrieves existing note or creates a new one if it doesn't exist. This is the primary endpoint for accessing notes.

**Request Headers**:
```http
X-Passphrase: my-secret-passphrase
```

**Request Example**:
```http
GET /api/secretnotes/notes HTTP/1.1
Host: 127.0.0.1:8091
X-Passphrase: my-secret-passphrase
```

**Response (New Note)**:
```json
{
  "id": "ab12cd34ef56gh78",
  "message": "",
  "hasImage": false,
  "created": "2024-01-15T10:30:00.000Z",
  "updated": "2024-01-15T10:30:00.000Z"
}
```

**Response (Existing Note)**:
```json
{
  "id": "ab12cd34ef56gh78",
  "message": "My existing note content here...",
  "hasImage": true,
  "created": "2024-01-15T10:30:00.000Z",
  "updated": "2024-01-16T14:22:00.000Z"
}
```

**Status Codes**:
- `200 OK`: Existing note retrieved successfully
- `201 Created`: New note created
- `400 Bad Request`: Invalid passphrase (too short)
- `500 Internal Server Error`: Server error

**Important Notes**:
- New notes start with an empty message
- The `message` field is always decrypted before being returned
- The `hasImage` boolean indicates if an image is attached
- Same passphrase always returns same note

---

### 3. Create Note (Alternative)

**Endpoint**: `POST /api/secretnotes/notes`
**Authentication**: Passphrase required

**Description**: Identical behavior to GET endpoint. Provided for REST convention. Use GET instead for simplicity.

**Request Headers**:
```http
Content-Type: application/json
X-Passphrase: my-secret-passphrase
```

**Request Body** (optional, can provide passphrase here instead of header):
```json
{
  "passphrase": "my-secret-passphrase"
}
```

**Request Example**:
```http
POST /api/secretnotes/notes HTTP/1.1
Host: 127.0.0.1:8091
Content-Type: application/json
X-Passphrase: my-secret-passphrase
```

**Response**: Same as GET endpoint

**Status Codes**: Same as GET endpoint

---

### 4. Update Note

**Endpoint**: `PATCH /api/secretnotes/notes`
**Authentication**: Passphrase required

**Description**: Updates the message content of an existing note. Note must already exist.

**Request Headers**:
```http
Content-Type: application/json
X-Passphrase: my-secret-passphrase
```

**Request Body**:
```json
{
  "message": "This is my updated note content"
}
```

**Alternative** (passphrase in body):
```json
{
  "passphrase": "my-secret-passphrase",
  "message": "This is my updated note content"
}
```

**Request Example**:
```http
PATCH /api/secretnotes/notes HTTP/1.1
Host: 127.0.0.1:8091
Content-Type: application/json
X-Passphrase: my-secret-passphrase

{
  "message": "Updated content here"
}
```

**Response**:
```json
{
  "id": "ab12cd34ef56gh78",
  "message": "Updated content here",
  "hasImage": false,
  "created": "2024-01-15T10:30:00.000Z",
  "updated": "2024-01-16T15:45:00.000Z"
}
```

**Status Codes**:
- `200 OK`: Note updated successfully
- `400 Bad Request`: Invalid request body or passphrase
- `404 Not Found`: Note doesn't exist (use GET to create first)
- `500 Internal Server Error`: Server error

**Important Notes**:
- The note must already exist; PATCH will not create a new note
- The `message` field is encrypted before storage
- Empty messages are allowed
- The `updated` timestamp is automatically updated

---

### 5. Upsert Note

**Endpoint**: `PUT /api/secretnotes/notes`
**Authentication**: Passphrase required

**Description**: Creates a new note or updates existing note in a single operation. Use this when you want to set the message without checking if the note exists first.

**Request Headers**:
```http
Content-Type: application/json
X-Passphrase: my-secret-passphrase
```

**Request Body**:
```json
{
  "message": "This message will be created or updated"
}
```

**Alternative** (passphrase in body):
```json
{
  "passphrase": "my-secret-passphrase",
  "message": "This message will be created or updated"
}
```

**Request Example**:
```http
PUT /api/secretnotes/notes HTTP/1.1
Host: 127.0.0.1:8091
Content-Type: application/json
X-Passphrase: my-secret-passphrase

{
  "message": "My note content"
}
```

**Response (Created)**:
```json
{
  "id": "ab12cd34ef56gh78",
  "message": "My note content",
  "hasImage": false,
  "created": "2024-01-16T16:00:00.000Z",
  "updated": "2024-01-16T16:00:00.000Z"
}
```

**Response (Updated)**:
```json
{
  "id": "ab12cd34ef56gh78",
  "message": "My note content",
  "hasImage": true,
  "created": "2024-01-15T10:30:00.000Z",
  "updated": "2024-01-16T16:00:00.000Z"
}
```

**Status Codes**:
- `200 OK`: Existing note updated
- `201 Created`: New note created
- `400 Bad Request`: Invalid request body or passphrase
- `500 Internal Server Error`: Server error

**Important Notes**:
- Most flexible endpoint for saving notes
- Can replace GET + PATCH workflow
- Empty messages are allowed

---

### 6. Upload Image

**Endpoint**: `POST /api/secretnotes/notes/image`
**Authentication**: Passphrase required

**Description**: Uploads and encrypts an image file to attach to the note. Only one image per note is supported; uploading a new image replaces the previous one.

**Request Headers**:
```http
Content-Type: multipart/form-data
X-Passphrase: my-secret-passphrase
```

**Request Body** (multipart/form-data):
- **Field Name**: `image`
- **Value**: Image file (binary data)

**Request Example** (pseudo-code):
```http
POST /api/secretnotes/notes/image HTTP/1.1
Host: 127.0.0.1:8091
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary
X-Passphrase: my-secret-passphrase

------WebKitFormBoundary
Content-Disposition: form-data; name="image"; filename="photo.jpg"
Content-Type: image/jpeg

[binary image data]
------WebKitFormBoundary--
```

**Response**:
```json
{
  "message": "Image uploaded successfully",
  "fileName": "photo.jpg",
  "fileSize": 245678,
  "contentType": "image/jpeg",
  "fileHash": "a1b2c3d4e5f6...",
  "created": "2024-01-16T16:30:00.000Z",
  "updated": "2024-01-16T16:30:00.000Z"
}
```

**Status Codes**:
- `200 OK`: Image uploaded successfully
- `400 Bad Request`: No image file provided, invalid form, or passphrase too short
- `500 Internal Server Error`: Encryption or storage error

**Important Notes**:
- Maximum file size: 10 MB (configurable in server)
- The image is encrypted before storage using the passphrase
- Previous image is automatically deleted when uploading a new one
- The note must exist before uploading an image (call GET endpoint first if needed)
- Supported formats: Any format (JPEG, PNG, GIF, WebP, etc.)
- The `fileHash` is a SHA-256 hash of the encrypted file content

---

### 7. Get Image

**Endpoint**: `GET /api/secretnotes/notes/image`
**Authentication**: Passphrase required

**Description**: Retrieves and decrypts the image attached to the note. Returns raw image file data.

**Request Headers**:
```http
X-Passphrase: my-secret-passphrase
```

**Request Example**:
```http
GET /api/secretnotes/notes/image HTTP/1.1
Host: 127.0.0.1:8091
X-Passphrase: my-secret-passphrase
```

**Response Headers**:
```http
HTTP/1.1 200 OK
Content-Type: image/jpeg
Content-Disposition: attachment; filename="photo.jpg"
```

**Response Body**: Raw binary image data

**Status Codes**:
- `200 OK`: Image retrieved successfully
- `400 Bad Request`: Invalid passphrase
- `404 Not Found`: No image found for this passphrase
- `500 Internal Server Error`: Decryption or retrieval error

**Important Notes**:
- Response is the actual image file, not JSON
- Use the `Content-Type` header to determine image format
- The `Content-Disposition` header provides the original filename
- To display: Create a blob URL from the response data
- Wrong passphrase will fail decryption (returns 404 or 500)

---

### 8. Delete Image

**Endpoint**: `DELETE /api/secretnotes/notes/image`
**Authentication**: Passphrase required

**Description**: Deletes the encrypted image attached to the note. The note itself remains intact.

**Request Headers**:
```http
X-Passphrase: my-secret-passphrase
```

**Request Example**:
```http
DELETE /api/secretnotes/notes/image HTTP/1.1
Host: 127.0.0.1:8091
X-Passphrase: my-secret-passphrase
```

**Response**:
```json
{
  "message": "Image deleted successfully"
}
```

**Status Codes**:
- `200 OK`: Image deleted successfully
- `400 Bad Request`: Invalid passphrase
- `404 Not Found`: No image found for this passphrase
- `500 Internal Server Error`: Deletion error

**Important Notes**:
- Only deletes the image, not the note
- After deletion, `hasImage` will be `false` when fetching the note
- Currently does NOT update the note's `image_hash` field (known limitation)

---

## Data Models

### Note Response Object

```typescript
interface NoteResponse {
  id: string;              // Unique note identifier (PocketBase record ID)
  message: string;         // Decrypted note content
  hasImage: boolean;       // True if image is attached
  created: string;         // ISO 8601 timestamp (UTC)
  updated: string;         // ISO 8601 timestamp (UTC)
}
```

**Example**:
```json
{
  "id": "ab12cd34ef56gh78",
  "message": "My secret note content",
  "hasImage": true,
  "created": "2024-01-15T10:30:00.000Z",
  "updated": "2024-01-16T14:22:00.000Z"
}
```

### Image Upload Response Object

```typescript
interface ImageUploadResponse {
  message: string;         // Success message
  fileName: string;        // Original filename
  fileSize: number;        // File size in bytes
  contentType: string;     // MIME type (e.g., "image/jpeg")
  fileHash: string;        // SHA-256 hash of encrypted file
  created: string | null;  // ISO 8601 timestamp or null
  updated: string | null;  // ISO 8601 timestamp or null
}
```

**Example**:
```json
{
  "message": "Image uploaded successfully",
  "fileName": "vacation.jpg",
  "fileSize": 245678,
  "contentType": "image/jpeg",
  "fileHash": "a1b2c3d4e5f6g7h8i9j0...",
  "created": "2024-01-16T16:30:00.000Z",
  "updated": "2024-01-16T16:30:00.000Z"
}
```

### Error Response Object

```typescript
interface ErrorResponse {
  error: string;           // Human-readable error message
}
```

**Example**:
```json
{
  "error": "Passphrase must be at least 3 characters long"
}
```

---

## Error Handling

### Error Response Format

All errors return a JSON object with an `error` field:

```json
{
  "error": "Description of what went wrong"
}
```

### Common Error Messages

| Error Message | Cause | Solution |
|--------------|-------|----------|
| `Passphrase must be at least 3 characters long` | Passphrase < 3 chars | Use longer passphrase |
| `Invalid request body` | Malformed JSON | Check JSON syntax |
| `No image file provided` | Missing `image` field in form | Include file in multipart form |
| `Failed to parse form` | Invalid multipart data | Verify multipart/form-data format |
| `note not found` | PATCH on non-existent note | Use GET or PUT first |
| `encrypted file not found` | No image for passphrase | Upload image first |
| `Failed to decrypt file` | Wrong passphrase or corrupted data | Verify passphrase is correct |
| `Failed to encrypt message` | Encryption error | Retry or report issue |

### HTTP Status Codes

| Status Code | Meaning | When It Occurs |
|-------------|---------|----------------|
| `200 OK` | Success | Successful GET, PATCH, PUT, DELETE, image operations |
| `201 Created` | Resource created | New note created via GET, POST, or PUT |
| `400 Bad Request` | Client error | Invalid passphrase, malformed request, missing fields |
| `404 Not Found` | Resource not found | Note or image doesn't exist |
| `500 Internal Server Error` | Server error | Database, encryption, or filesystem errors |

### Handling Decryption Failures

If decryption fails, it typically means:
1. **Wrong passphrase** - User entered incorrect passphrase
2. **Data corruption** - Database or file corruption (rare)

In your app, treat decryption failures as authentication failures:

```typescript
try {
  const response = await fetch('/api/secretnotes/notes', {
    headers: { 'X-Passphrase': passphrase }
  });

  if (!response.ok) {
    // Handle as "wrong passphrase" or "note not found"
    showError('Unable to access note. Check your passphrase.');
  }
} catch (error) {
  showError('Network error. Please try again.');
}
```

---

## Code Examples

### JavaScript/TypeScript (Fetch API)

#### Get or Create Note

```typescript
async function getNote(passphrase: string): Promise<NoteResponse> {
  const response = await fetch('http://127.0.0.1:8091/api/secretnotes/notes', {
    method: 'GET',
    headers: {
      'X-Passphrase': passphrase
    }
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to get note');
  }

  return await response.json();
}

// Usage
try {
  const note = await getNote('my-secret-passphrase');
  console.log('Note content:', note.message);
  console.log('Has image:', note.hasImage);
} catch (error) {
  console.error('Error:', error.message);
}
```

#### Update Note

```typescript
async function updateNote(passphrase: string, message: string): Promise<NoteResponse> {
  const response = await fetch('http://127.0.0.1:8091/api/secretnotes/notes', {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
      'X-Passphrase': passphrase
    },
    body: JSON.stringify({ message })
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to update note');
  }

  return await response.json();
}

// Usage
try {
  const updated = await updateNote('my-secret-passphrase', 'New content here');
  console.log('Updated at:', updated.updated);
} catch (error) {
  console.error('Error:', error.message);
}
```

#### Upsert Note (Create or Update)

```typescript
async function upsertNote(passphrase: string, message: string): Promise<NoteResponse> {
  const response = await fetch('http://127.0.0.1:8091/api/secretnotes/notes', {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'X-Passphrase': passphrase
    },
    body: JSON.stringify({ message })
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to upsert note');
  }

  return await response.json();
}

// Usage
try {
  const note = await upsertNote('my-secret-passphrase', 'This will be created or updated');
  console.log(note.message);
} catch (error) {
  console.error('Error:', error.message);
}
```

#### Upload Image

```typescript
async function uploadImage(passphrase: string, imageFile: File): Promise<ImageUploadResponse> {
  const formData = new FormData();
  formData.append('image', imageFile);

  const response = await fetch('http://127.0.0.1:8091/api/secretnotes/notes/image', {
    method: 'POST',
    headers: {
      'X-Passphrase': passphrase
    },
    body: formData
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to upload image');
  }

  return await response.json();
}

// Usage with file input
const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
fileInput.addEventListener('change', async (e) => {
  const file = (e.target as HTMLInputElement).files?.[0];
  if (file) {
    try {
      const result = await uploadImage('my-secret-passphrase', file);
      console.log('Uploaded:', result.fileName);
    } catch (error) {
      console.error('Error:', error.message);
    }
  }
});
```

#### Get and Display Image

```typescript
async function getImage(passphrase: string): Promise<Blob> {
  const response = await fetch('http://127.0.0.1:8091/api/secretnotes/notes/image', {
    method: 'GET',
    headers: {
      'X-Passphrase': passphrase
    }
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to get image');
  }

  return await response.blob();
}

// Usage - Display in <img> element
async function displayImage(passphrase: string, imgElement: HTMLImageElement) {
  try {
    const blob = await getImage(passphrase);
    const url = URL.createObjectURL(blob);
    imgElement.src = url;

    // Clean up object URL when no longer needed
    imgElement.onload = () => URL.revokeObjectURL(url);
  } catch (error) {
    console.error('Error:', error.message);
  }
}
```

#### Delete Image

```typescript
async function deleteImage(passphrase: string): Promise<void> {
  const response = await fetch('http://127.0.0.1:8091/api/secretnotes/notes/image', {
    method: 'DELETE',
    headers: {
      'X-Passphrase': passphrase
    }
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.error || 'Failed to delete image');
  }

  const result = await response.json();
  console.log(result.message);
}

// Usage
try {
  await deleteImage('my-secret-passphrase');
  console.log('Image deleted');
} catch (error) {
  console.error('Error:', error.message);
}
```

### React Native / Expo Example

```typescript
import { useState, useEffect } from 'react';
import * as ImagePicker from 'expo-image-picker';

const API_BASE = 'http://127.0.0.1:8091/api/secretnotes';

interface Note {
  id: string;
  message: string;
  hasImage: boolean;
  created: string;
  updated: string;
}

export function useSecretNote(passphrase: string) {
  const [note, setNote] = useState<Note | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch note
  const fetchNote = async () => {
    if (passphrase.length < 3) {
      setError('Passphrase must be at least 3 characters');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/notes`, {
        method: 'GET',
        headers: {
          'X-Passphrase': passphrase
        }
      });

      if (!response.ok) {
        const err = await response.json();
        throw new Error(err.error);
      }

      const data = await response.json();
      setNote(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch note');
    } finally {
      setLoading(false);
    }
  };

  // Update note
  const updateNote = async (message: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/notes`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'X-Passphrase': passphrase
        },
        body: JSON.stringify({ message })
      });

      if (!response.ok) {
        const err = await response.json();
        throw new Error(err.error);
      }

      const data = await response.json();
      setNote(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update note');
    } finally {
      setLoading(false);
    }
  };

  // Upload image
  const uploadImage = async () => {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ImagePicker.MediaTypeOptions.Images,
      allowsEditing: true,
      quality: 0.8
    });

    if (result.canceled) return;

    const asset = result.assets[0];
    const formData = new FormData();

    // @ts-ignore - FormData in React Native accepts this format
    formData.append('image', {
      uri: asset.uri,
      type: asset.type || 'image/jpeg',
      name: asset.fileName || 'image.jpg'
    });

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/notes/image`, {
        method: 'POST',
        headers: {
          'X-Passphrase': passphrase
        },
        body: formData
      });

      if (!response.ok) {
        const err = await response.json();
        throw new Error(err.error);
      }

      // Refresh note to update hasImage status
      await fetchNote();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to upload image');
    } finally {
      setLoading(false);
    }
  };

  // Get image URL
  const getImageUri = (): string | null => {
    if (!note?.hasImage) return null;
    return `${API_BASE}/notes/image`;
  };

  useEffect(() => {
    if (passphrase.length >= 3) {
      fetchNote();
    }
  }, [passphrase]);

  return {
    note,
    loading,
    error,
    updateNote,
    uploadImage,
    getImageUri,
    refresh: fetchNote
  };
}
```

### Axios Example

```typescript
import axios from 'axios';

const api = axios.create({
  baseURL: 'http://127.0.0.1:8091/api/secretnotes'
});

// Get note
async function getNote(passphrase: string) {
  const response = await api.get('/notes', {
    headers: { 'X-Passphrase': passphrase }
  });
  return response.data;
}

// Update note
async function updateNote(passphrase: string, message: string) {
  const response = await api.patch('/notes',
    { message },
    { headers: { 'X-Passphrase': passphrase } }
  );
  return response.data;
}

// Upload image
async function uploadImage(passphrase: string, file: File) {
  const formData = new FormData();
  formData.append('image', file);

  const response = await api.post('/notes/image', formData, {
    headers: {
      'X-Passphrase': passphrase,
      'Content-Type': 'multipart/form-data'
    }
  });
  return response.data;
}

// Get image
async function getImage(passphrase: string) {
  const response = await api.get('/notes/image', {
    headers: { 'X-Passphrase': passphrase },
    responseType: 'blob'
  });
  return response.data;
}
```

---

## Best Practices

### Frontend Implementation

1. **Passphrase Validation**
   - Validate minimum 3 characters before API calls
   - Show clear error messages for invalid passphrases
   - Consider client-side length/complexity indicators

2. **Loading States**
   - Show loading indicators during API calls
   - Disable UI elements during operations
   - Handle network timeouts gracefully

3. **Error Handling**
   - Display user-friendly error messages
   - Log technical errors for debugging
   - Provide retry mechanisms for failed operations

4. **Image Handling**
   - Check `hasImage` before attempting to fetch images
   - Cache image blobs to avoid redundant fetches
   - Clean up blob URLs with `URL.revokeObjectURL()`
   - Compress images before upload to reduce size

5. **State Management**
   - Cache note data to minimize API calls
   - Implement optimistic updates for better UX
   - Sync state after successful operations

6. **Security**
   - Never log passphrases in production
   - Clear sensitive data from memory when done
   - Use HTTPS in production
   - Warn users about passphrase strength

### UX Recommendations

1. **Public vs Private Indicators**
   - Show warning when using common passphrases like "hello", "test"
   - Suggest strong passphrases for private notes
   - Educate users about the public nature of simple passphrases

2. **Collaborative Features**
   - Show timestamp of last update
   - Implement pull-to-refresh for public notes
   - Consider real-time updates (polling or websockets)

3. **Offline Support**
   - Cache note content for offline viewing
   - Queue updates when offline, sync when online
   - Handle sync conflicts appropriately

4. **Image Management**
   - Show image preview before upload
   - Display file size limits
   - Support image deletion with confirmation
   - Show loading state during encryption/upload

### Testing Recommendations

1. **Test Cases**
   - Create new note with valid passphrase
   - Access existing note
   - Update note multiple times
   - Upload and retrieve images
   - Delete images
   - Test with various passphrase lengths (3+, 10+, 50+ chars)
   - Test with special characters in passphrases
   - Test with large images (near 10MB limit)
   - Test error scenarios (network failures, invalid data)

2. **Edge Cases**
   - Empty message updates
   - Rapid consecutive updates
   - Image upload during note update
   - Multiple users editing same public note
   - Very long passphrases (100+ characters)
   - Unicode characters in messages and passphrases

3. **Performance Testing**
   - Measure encryption/decryption time for large images
   - Test with poor network conditions
   - Monitor memory usage with large images

---

## API Versioning

Current version: **1.0.0**

The API version is returned in the health check endpoint. Future breaking changes will be communicated via:
- Version number increment
- Updated documentation
- Backward compatibility notes

---

## Rate Limiting

Currently, there is **no rate limiting** implemented. However, best practices:

- Don't spam the API with rapid requests
- Implement debouncing for update operations
- Cache data locally when possible
- Use polling intervals of 5+ seconds for public note collaboration

---

## CORS Configuration

The backend is configured to allow cross-origin requests. You should be able to call the API from:
- Web browsers (different ports)
- Mobile apps (React Native/Expo)
- Desktop apps (Electron)

If you encounter CORS issues, ensure:
- Requests include proper headers
- OPTIONS preflight requests are not blocked
- HTTPS is used in production

---

## Production Deployment

### Required Changes for Production

1. **Use HTTPS**
   - Deploy behind reverse proxy (nginx, Traefik, Caddy)
   - Obtain SSL/TLS certificate (Let's Encrypt recommended)
   - Passphrases MUST be protected in transit

2. **Update Base URL**
   - Replace `http://127.0.0.1:8091` with your production URL
   - Use environment variables for configuration
   - Support multiple environments (dev, staging, prod)

3. **Security Headers**
   - Ensure CORS is properly configured
   - Add security headers (HSTS, CSP, etc.)
   - Consider rate limiting at reverse proxy level

4. **Monitoring**
   - Log API errors (without sensitive data)
   - Monitor API response times
   - Track storage usage

### Production Checklist

- [ ] HTTPS enabled
- [ ] Base URL updated
- [ ] Environment variables configured
- [ ] Error logging implemented
- [ ] Backups configured
- [ ] Monitoring set up
- [ ] Security headers added
- [ ] CORS properly configured
- [ ] Rate limiting considered
- [ ] Performance tested

---

## Support and Issues

For issues or questions:
- Check this documentation first
- Review the code examples
- Test with curl or Postman to isolate issues
- Check server logs for detailed error messages

---

## Appendix: Quick Reference

### Base URLs

| Environment | URL |
|-------------|-----|
| Development | `http://127.0.0.1:8091/api/secretnotes` |
| Production  | `https://your-domain.com/api/secretnotes` |

### Endpoints Summary

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/` | Health check |
| GET | `/notes` | Get or create note |
| POST | `/notes` | Create note (same as GET) |
| PATCH | `/notes` | Update existing note |
| PUT | `/notes` | Upsert note |
| POST | `/notes/image` | Upload image |
| GET | `/notes/image` | Get image |
| DELETE | `/notes/image` | Delete image |

### Headers Required

| Header | Required For | Value |
|--------|-------------|-------|
| `X-Passphrase` | All note/image ops | Your passphrase (3+ chars) |
| `Content-Type` | JSON requests | `application/json` |
| `Content-Type` | Image upload | `multipart/form-data` |

### Passphrase Examples

| Type | Example | Use Case |
|------|---------|----------|
| Public | `hello` | Shared, collaborative note |
| Public | `test` | Public playground |
| Semi-Private | `team-meeting-notes` | Small group access |
| Private | `j8$kL9mP2#qR5nT` | Personal, secure note |

---

**Document Version**: 1.0.0
**Last Updated**: 2024-01-16
**API Version**: 1.0.0
