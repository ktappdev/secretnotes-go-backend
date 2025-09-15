# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Secret Notes Go Backend is a secure, self-hosted notes application built with Go and PocketBase. It provides end-to-end encryption for both text notes and file attachments using AES-256-GCM with passphrase-based security.

## Development Commands

### Running the Application
```bash
# Run the development server (default: 127.0.0.1:8091)
go run main.go

# Run with custom host/port
go run main.go serve --http 0.0.0.0:8090

# Run PocketBase admin commands
go run main.go serve --http 0.0.0.0:8090
go run main.go migrate collections
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific service
go test ./services/...

# Run tests with verbose output
go test -v ./services/...
```

### Dependencies
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Update dependencies
go get -u
```

### Database Management
```bash
# Reset database (removes pb_data directory)
rm -rf pb_data/
go run main.go serve

# The database is automatically created in pb_data/ directory
```

## Architecture

### Core Components

1. **PocketBase Framework** (`main.go`): Main application using PocketBase for database, routing, and file storage
2. **Encryption Service** (`services/encryption.go`): AES-256-GCM encryption with PBKDF2 key derivation
3. **Note Service** (`services/note.go`): Manages note CRUD operations with encryption
4. **File Service** (`services/file.go`): Handles encrypted file storage and retrieval
5. **Database Models** (`models/`): Data structure definitions
6. **Migrations** (`migrations/`): Database schema changes

### Security Architecture

- **Encryption**: AES-256-GCM with PBKDF2 (10,000 iterations, SHA-256)
- **Key Storage**: Passphrases are never stored; only SHA-256 hashes for lookup
- **File Storage**: Files encrypted before storage, decrypted on retrieval
- **Zero-Knowledge**: Server cannot access user data without the passphrase

### API Endpoints

All endpoints are under `/api/secretnotes/` and use passphrase authentication via `X-Passphrase` header or JSON body:

- `GET /api/secretnotes/` - Health check
- `GET /api/secretnotes/notes` - Get or create note
- `POST /api/secretnotes/notes` - Create note (same as GET)
- `PATCH /api/secretnotes/notes` - Update note
- `PUT /api/secretnotes/notes` - Upsert note
- `POST /api/secretnotes/notes/image` - Upload encrypted image
- `GET /api/secretnotes/notes/image` - Retrieve decrypted image
- `DELETE /api/secretnotes/notes/image` - Delete image

### Database Collections

1. **notes**: Stores encrypted note content
   - `phrase_hash`: SHA-256 hash of passphrase for lookup
   - `message`: Encrypted note content
   - `image_hash`: Reference to encrypted file

2. **encrypted_files**: Stores encrypted file data
   - `phrase_hash`: SHA-256 hash of passphrase
   - `file_name`: Original filename
   - `content_type`: MIME type
   - `file_data`: Encrypted file content via PocketBase file system

## Development Patterns

### Adding New Endpoints

1. Add route handler in `main.go` using the existing pattern
2. Implement encryption using the `EncryptionService`
3. Validate passphrase length (minimum 3 characters)
4. Handle errors consistently with appropriate HTTP status codes
5. Use SHA-256 hashing for phrase storage, never store raw passphrases

### Service Layer Pattern

- All business logic is in the `services/` directory
- Services accept PocketBase app instance and encryption service
- Database operations use PocketBase's record system
- Error handling follows Go conventions with wrapped errors

### Encryption Usage

```go
// Encrypt data
encryptionService := services.NewEncryptionService()
encrypted, err := encryptionService.EncryptData([]byte(data), passphrase)

// Decrypt data
decrypted, err := encryptionService.DecryptData(encryptedData, passphrase)
```

### File Operations

Files are encrypted before storage and decrypted on retrieval. The `FileService` handles all file operations including:

- Encryption using user's passphrase
- Storage via PocketBase's file system
- Metadata preservation (filename, content type)
- Secure lookup using phrase hashes

## Testing

### Unit Tests

- Tests are located alongside services (`services/encryption_test.go`)
- Use Go's built-in testing framework
- Test encryption/decryption with different passphrases
- Test edge cases and error conditions

### API Testing

- Bruno API test collection in `api-test/` directory
- Tests cover all endpoints with various scenarios
- Include tests for invalid passphrases and error handling

## Security Considerations

### Critical Security Rules

1. **Never store raw passphrases** - always use SHA-256 hashes
2. **Always encrypt sensitive data** before database storage
3. **Use strong encryption** - AES-256-GCM with proper key derivation
4. **Validate input** - minimum passphrase length, sanitize inputs
5. **No logging of sensitive data** - passphrases, decrypted content
6. **Use HTTPS in production** - protect passphrases in transit

### Passphrase Handling

- Minimum 3 characters (enforced in `main.go:297`)
- Hashed with SHA-256 for storage/lookup
- Used as encryption key via PBKDF2 derivation
- Never exposed in logs or error messages

### Error Handling

- Use generic error messages for encryption failures
- Don't expose internal details in API responses
- Log technical errors for debugging without sensitive data
- Return appropriate HTTP status codes (400, 404, 500)

## Deployment

### Production Requirements

1. **HTTPS mandatory** - use reverse proxy with SSL/TLS
2. **Environment variables** for configuration
3. **File permissions** - restrict access to pb_data/ directory
4. **Backups** - regular backups of pb_data/ directory
5. **Monitoring** - application health and error logs

### Environment Variables

- No specific environment variables required
- PocketBase uses default data directory (`./pb_data`)
- Server binds to `127.0.0.1:8091` by default

## File Structure

```
.
├── main.go                    # Application entry point
├── go.mod/go.sum             # Go module dependencies
├── services/                 # Business logic services
│   ├── encryption.go         # AES-256-GCM encryption service
│   ├── encryption_test.go    # Encryption tests
│   ├── note.go              # Note management service
│   └── file.go              # File handling service
├── models/                   # Database model definitions
│   ├── note.go              # Note data model
│   └── encrypted_file.go    # Encrypted file model
├── migrations/               # Database schema migrations
│   ├── 001_init.go          # Initial schema
│   ├── 002_increase_encrypted_content_limit.go
│   └── 003_remove_encrypted_content.go
├── pb_data/                  # PocketBase data directory (auto-created)
├── api-test/                 # Bruno API test collection
└── README.md                # Project documentation
```

## Common Issues and Solutions

### Database Issues

- **Schema errors**: Run `go run main.go migrate collections`
- **Corrupted data**: Delete `pb_data/` and restart
- **Permission errors**: Check file permissions on `pb_data/`

### Encryption Issues

- **Decryption failures**: Verify passphrase is correct
- **Invalid data**: Check data wasn't corrupted during storage
- **Performance**: Large files may take time to encrypt/decrypt

### File Upload Issues

- **Size limits**: Check multipart form size limits in handlers
- **Content types**: Verify file types are properly handled
- **Storage**: Ensure sufficient disk space in `pb_data/` directory