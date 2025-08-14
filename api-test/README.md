# Secret Notes API - Bruno Test Collection

This directory contains Bruno API test files for the Secret Notes backend API.

## Setup

1. Install Bruno: https://www.usebruno.com/
2. Open Bruno and import this collection
3. Select either "Local" or "Production" environment

## Environments

- **Local**: `http://localhost:8091` - For local development
- **Production**: `https://secret-note-backend.lugetech.com` - For production testing

## Test Files

1. **Health Check** - Verify API is running
2. **Get or Create Note** - Test auto-creation functionality (GET)
3. **Create Note (POST)** - Test auto-creation via POST
4. **Update Note (PATCH)** - Test updating existing notes only
5. **Upsert Note (PUT)** - Test create-or-update functionality (recommended)
6. **Upload Image** - Test image upload to a note
7. **Get Image** - Test image retrieval
8. **Delete Image** - Test image deletion
9. **Test Invalid Passphrase** - Test validation (passphrase < 3 chars)
10. **Test PATCH Non-Existent Note** - Test PATCH behavior on missing notes

## Key API Behaviors Tested

### Auto-Creation
- GET/POST `/api/secretnotes/notes/{phrase}` automatically creates notes if they don't exist
- Returns 201 for new notes, 200 for existing notes

### Upsert vs Update
- **PUT** (upsert): Creates if missing, updates if exists - **recommended**
- **PATCH** (update): Only updates existing notes, returns 404 if missing

### Validation
- All endpoints validate minimum passphrase length (3 characters)
- Returns 400 with error message for invalid passphrases

## Running Tests

1. Start your local server: `go run main.go`
2. Open Bruno and select the "Local" environment
3. Run individual tests or the entire collection
4. For production testing, switch to "Production" environment

## Notes

- Test passphrases are defined in environment variables
- Image upload test requires a test image file (test-image.jpg)
- All tests include response validation and status code checks
