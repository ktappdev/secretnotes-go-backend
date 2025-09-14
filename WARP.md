# WARP.md
This file provides guidance to WARP (warp.dev) when working with code in this repository.

Common development commands
- Prerequisites: Go 1.23+ (per go.mod)
- Setup dependencies: go mod download
- Run server (binds to 127.0.0.1:8091 via RootCmd.SetArgs): go run main.go
- Build: go build -o bin/secretnotes .
- Test all packages: go test ./...
- Run a single test (example): go test -run TestEncryptionService ./services/...
- Lint (if installed): golangci-lint run
- Format: go fmt ./...
- Reset local PocketBase data: rm -rf pb_data && go run main.go
- API testing via Bruno: open api-test/ in Bruno, set base URL to http://127.0.0.1:8091 and run requests
- Migrations: go run main.go migrate collections
  - Note: main.go forces serve args; to run migrations you may need to temporarily disable the RootCmd.SetArgs override

High-level architecture overview
- Runtime: PocketBase app with custom routes defined in main.go under /api/secretnotes. Startup is forced to serve at 127.0.0.1:8091 using app.RootCmd.SetArgs.
- Endpoints:
  - GET/POST /api/secretnotes/notes/{phrase}: get-or-create note; returns 201 when a new note is auto-created with a welcome message
  - PUT /api/secretnotes/notes/{phrase}: upsert (create if missing, update if exists)
  - PATCH /api/secretnotes/notes/{phrase}: update-only; 404 if note does not exist
  - POST/GET/DELETE /api/secretnotes/notes/{phrase}/image: upload, retrieve, and delete an encrypted file tied to the phrase
- Services (services/):
  - EncryptionService (encryption.go): AES-256-GCM with PBKDF2(SHA-256, 10,000 iterations), 16-byte salt, 12-byte nonce; storage format is [salt][nonce][ciphertext]. Provides EncryptData/DecryptData and string helpers.
  - NoteService (note.go): Looks up notes by phrase_hash (SHA-256 of phrase), encrypts and stores message; decrypts on read. Auto-creates a note with an encrypted welcome message on first access.
  - FileService (file.go): Encrypts file bytes with the phrase; stores metadata (file_name, content_type) and encrypted_content in encrypted_files. Also keeps a PocketBase file_data field for compatibility. Retrieval decrypts from encrypted_content; delete removes the record.
- Data model and migrations (migrations/):
  - notes: phrase_hash (text, required), message (text, required), image_hash (text)
  - encrypted_files: phrase_hash (text, required), file_name (text, required), content_type (text, required), file_data (file), encrypted_content (long text)
  - Migration 002 switches encrypted_content to a LongTextField to support larger payloads
- Validation and behavior notes:
  - Handlers enforce a minimum passphrase length of 3 characters (see main.go); this differs from SECURITY.md guidance. Treat the current code as authoritative for runtime behavior.
  - Bruno tests in api-test/ target http://localhost:8091; server in code listens on 127.0.0.1:8091. Use http://127.0.0.1:8091 to avoid host binding mismatches.
- Useful references:
  - README.md: quickstart, API summary, and commands
  - BACKEND_DOCS.md: API details, encryption overview, and migration command
  - FRONTEND_GUIDE.md: endpoint usage examples and recommended client flows
  - CRUSH.md: concise build/lint/test commands and style notes
