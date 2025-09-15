# Bruno API Tests

This directory contains Bruno API tests for the Secret Notes backend.

## Test Files

### Health Check
- **Health_Check.bru** - Basic API health check (GET /api/secretnotes/)

### Note Operations
- **notes_get_or_create.bru** - Get or create note (GET /api/secretnotes/notes)
- **notes_create_post.bru** - Create note via POST (POST /api/secretnotes/notes)
- **notes_update_patch.bru** - Update note (PATCH /api/secretnotes/notes)
- **notes_upsert_put.bru** - Create or update note (PUT /api/secretnotes/notes)

### Image Operations
- **images_upload.bru** - Upload encrypted image (POST /api/secretnotes/notes/image)
- **images_get.bru** - Retrieve decrypted image (GET /api/secretnotes/notes/image)
- **images_delete.bru** - Delete uploaded image (DELETE /api/secretnotes/notes/image)

## Test Data Files
- **test-image.png** - Test image file for upload tests

## Environment Variables
All tests use variables defined in `environments/Local.bru`:
- `baseUrl` - Server URL (default: http://127.0.0.1:8091)
- `testPhrase` - Test passphrase (default: test-phrase-123)
- `testPhrase2` - Alternative test passphrase
- `imageFile` - Test image file name

## Usage
1. Open Bruno
2. Load this collection
3. Select the desired environment (Local/Production)
4. Run tests individually or as a collection

## Note
- All tests require the X-Passphrase header for authentication
- Image upload tests use multipart-form data
- Tests are ordered sequentially for logical flow