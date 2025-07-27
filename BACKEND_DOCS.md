# Secret Notes Go Backend Documentation

## Overview

This document provides technical documentation for the Secret Notes backend, which has been migrated from a Node.js/Express/MongoDB/Azure stack to a Go-based backend using PocketBase. The new backend implements unified encryption for both notes and images using the same passphrase (AES-256-GCM, PBKDF2) and leverages PocketBase's routing and file storage capabilities for enhanced security and maintainability.

## Architecture

The backend is structured as follows:

```
.
├── go.mod
├── go.sum
├── main.go
├── migrations/
│   └── 001_init.go
├── models/
│   ├── encrypted_file.go
│   └── note.go
└── services/
    ├── encryption.go
    ├── file.go
    └── note.go
```

### Core Components

1. **PocketBase App**: The main application framework that provides database, file storage, and routing capabilities.
2. **Custom Routes**: API endpoints implemented using PocketBase's routing system.
3. **Encryption Service**: Handles encryption/decryption of notes and images using AES-256-GCM with PBKDF2 key derivation.
4. **Note Service**: Manages note creation, retrieval, and updating.
5. **File Service**: Handles encrypted file storage and retrieval.
6. **Models**: Data models for notes and encrypted files.
7. **Migrations**: Database schema migrations using PocketBase's migration system.

## Encryption

The encryption service implements a unified encryption approach for both notes and images:

1. **Key Derivation**: Uses PBKDF2 with SHA-256 to derive a 256-bit key from the user's passphrase.
2. **Encryption Algorithm**: Uses AES-256-GCM for authenticated encryption.
3. **Salt**: A 16-byte random salt is generated for each encryption operation.
4. **Nonce**: A 12-byte random nonce is generated for each encryption operation.
5. **Storage Format**: The encrypted data is stored as `salt + nonce + encrypted_data`.

### Encryption Process

1. Generate a random 16-byte salt.
2. Derive a 32-byte key from the passphrase using PBKDF2 with the salt.
3. Generate a random 12-byte nonce.
4. Encrypt the data using AES-256-GCM with the derived key and nonce.
5. Store the salt, nonce, and encrypted data together.

### Decryption Process

1. Extract the salt, nonce, and encrypted data from storage.
2. Derive the key from the passphrase using PBKDF2 with the extracted salt.
3. Decrypt the data using AES-256-GCM with the derived key and extracted nonce.

## API Approaches

The backend supports two API approaches:

1. **Custom API Endpoints** - Custom endpoints that handle encryption/decryption on the server side
2. **Direct PocketBase API** - Standard PocketBase collection endpoints for direct access

### Recommendation

For most use cases, we recommend using the **Direct PocketBase API** for the following reasons:

1. Better compatibility with frontend frameworks
2. More flexible querying capabilities
3. Built-in support for pagination, filtering, and sorting
4. Simpler file handling with multipart/form-data

However, this requires implementing client-side encryption/decryption.

## Custom API Endpoints

All custom endpoints are prefixed with `/api/secretnotes/`.

### Health Check

- **Endpoint**: `GET /api/secretnotes/`
- **Description**: Health check endpoint to verify the API is running.
- **Response**: 
  ```json
  {
    "message": "Secret Notes API is live",
    "version": "1.0.0"
  }
  ```

### Get Note

- **Endpoint**: `GET /api/secretnotes/notes/{phrase}`
- **Description**: Retrieves an existing note or creates a new one if it doesn't exist.
- **Path Parameters**:
  - `phrase`: The passphrase used for encryption (minimum 32 characters).
- **Response**: 
  ```json
  {
    "id": "record_id",
    "message": "decrypted_message_content",
    "hasImage": true,
    "created": "2023-01-01T00:00:00Z",
    "updated": "2023-01-01T00:00:00Z"
  }
  ```

### Create Note

- **Endpoint**: `POST /api/secretnotes/notes/{phrase}`
- **Description**: Creates a new note with the specified content.
- **Path Parameters**:
  - `phrase`: The passphrase used for encryption (minimum 32 characters).
- **Request Body**:
  ```json
  {
    "title": "Note Title",
    "message": "Note content to encrypt"
  }
  ```
- **Response**: 
  ```json
  {
    "id": "record_id",
    "message": "decrypted_message_content",
    "hasImage": false,
    "created": "2023-01-01T00:00:00Z",
    "updated": "2023-01-01T00:00:00Z"
  }
  ```

### Update Note

- **Endpoint**: `PATCH /api/secretnotes/notes/{phrase}`
- **Description**: Updates an existing note with new content.
- **Path Parameters**:
  - `phrase`: The passphrase used for encryption (minimum 32 characters).
- **Request Body**:
  ```json
  {
    "message": "new_note_content"
  }
  ```
- **Response**: 
  ```json
  {
    "id": "record_id",
    "message": "decrypted_message_content",
    "hasImage": true,
    "created": "2023-01-01T00:00:00Z",
    "updated": "2023-01-01T00:00:00Z"
  }
  ```

### Upload Image

- **Endpoint**: `POST /api/secretnotes/notes/{phrase}/image`
- **Description**: Uploads and encrypts an image associated with a note.
- **Path Parameters**:
  - `phrase`: The passphrase used for encryption (minimum 32 characters).
- **Request Body**: Multipart form data with the image file.
- **Form Fields**:
  - `image`: The image file to upload
- **Response**: 
  ```json
  {
    "message": "Image uploaded successfully",
    "fileName": "image.jpg",
    "fileSize": 12345,
    "contentType": "image/jpeg"
  }
  ```

### Get Image

- **Endpoint**: `GET /api/secretnotes/notes/{phrase}/image`
- **Description**: Retrieves and decrypts an image associated with a note.
- **Path Parameters**:
  - `phrase`: The passphrase used for encryption (minimum 32 characters).
- **Response**: The decrypted image file (binary data)
- **Headers**:
  - `Content-Type`: The original content type of the image
  - `Content-Disposition`: Attachment with the original filename

### Delete Image

- **Endpoint**: `DELETE /api/secretnotes/notes/{phrase}/image`
- **Description**: Deletes an image associated with a note.
- **Path Parameters**:
  - `phrase`: The passphrase used for encryption (minimum 32 characters).
- **Response**: 
  ```json
  {
    "message": "Image deleted successfully"
  }
  ```

## Data Models

### Note

Represents a secret note with the following fields:

- `id`: Unique identifier (generated by PocketBase).
- `phrase_hash`: SHA-256 hash of the passphrase for secure lookup.
- `message`: Encrypted note content (AES-256-GCM).
- `image_hash`: SHA-256 hash for encrypted image lookup.
- `created`: Timestamp when the note was created.
- `updated`: Timestamp when the note was last updated.

## File Storage

The file service handles encrypted image storage and retrieval using a hybrid approach that leverages both PocketBase's file storage system and custom encryption:

1. **Upload Process**: 
   - Files are encrypted using AES-256-GCM with the user's passphrase
   - The encrypted content is stored in a custom `encrypted_content` field
   - The original file is also stored in PocketBase's `file_data` field for metadata and organization
   - File metadata (name, content type) is stored alongside the encrypted content

2. **Storage Architecture**:
   - **Custom Encryption**: Encrypted file content stored in `encrypted_content` field
   - **PocketBase Integration**: Original file stored in `file_data` field for PocketBase compatibility
   - **Metadata**: File name, content type, and phrase hash stored as separate fields

3. **Retrieval Process**:
   - Files are retrieved by looking up the `encrypted_content` field
   - Content is decrypted using the user's passphrase
   - Decrypted content is served directly to the client with appropriate headers

4. **Security**: All file content is encrypted at rest using AES-256-GCM and can only be decrypted with the correct passphrase.

### File Storage Schema

The `encrypted_files` collection contains:
- `phrase_hash`: SHA-256 hash of the passphrase for secure lookup
- `file_name`: Original filename
- `content_type`: MIME type of the file
- `file_data`: Original file stored by PocketBase (for compatibility)
- `encrypted_content`: AES-256-GCM encrypted file content (custom field)
- `created`: Timestamp when the file was created.
- `updated`: Timestamp when the file was last updated.

## Migrations

Database schema migrations are handled using PocketBase's migration system. The initial migration creates two collections:

1. `notes`: Stores encrypted note data.
2. `encrypted_files`: Stores metadata for encrypted files.

To apply migrations, run:

```bash
go run main.go migrate collections
```

## Direct PocketBase API

The backend also supports direct access to PocketBase collections. These endpoints are prefixed with `/api/collections/`.

### Notes Collection

- **Create Note**: `POST /api/collections/notes/records`
  ```json
  {
    "phrase_hash": "sha256_hash_of_passphrase",
    "message": "encrypted_message",
    "image_hash": ""
  }
  ```

- **Get Notes**: `GET /api/collections/notes/records?filter=phrase_hash='sha256_hash_of_passphrase'`

### Encrypted Files Collection

- **Upload File**: `POST /api/collections/encrypted_files/records` (multipart/form-data)
  - Form Fields:
    - `phrase_hash`: SHA-256 hash of the passphrase
    - `file_name`: Original name of the file
    - `content_type`: MIME type of the file
    - `file_data`: The file to upload (will be stored encrypted)

- **Get Files**: `GET /api/collections/encrypted_files/records?filter=phrase_hash='sha256_hash_of_passphrase'`

## Security Considerations

1. All sensitive data (notes and images) is encrypted at rest.
2. Passphrases are never stored in plain text.
3. When using direct PocketBase API, encryption/decryption must be handled by the client.
3. Each encryption operation uses a unique salt and nonce.
4. Minimum passphrase length is enforced (32 characters).
5. AES-256-GCM provides both confidentiality and authenticity.

## Future Work

1. Add middleware for rate limiting.
2. Enhance error handling and logging.
3. Add unit and integration tests.
4. Implement data migration from the old Node.js backend.
5. Add support for multiple images per note.
6. Implement secure passphrase recovery mechanism.
