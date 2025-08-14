# CRUSH.md - Codebase Guidelines

## Build/Lint/Test Commands

- **Run server**: `go run main.go serve`
- **Run tests**: `go test ./services/...`
- **Run single test**: `go test -run TestEncryptionService ./services/...`
- **Lint**: `golangci-lint run` (if installed)
- **Format**: `go fmt ./...`

## Code Style Guidelines

### Imports
- Group imports by standard library, third-party, and local packages
- Use blank imports for package initialization (`import _ "package"`)

### Formatting
- Use `go fmt` for consistent formatting
- Indent with tabs (standard Go convention)
- Keep line length under 120 characters where possible

### Types
- Use descriptive type names with CamelCase
- Prefer explicit error handling over panics
- Use `interface{}` sparingly, prefer concrete types

### Naming Conventions
- **Packages**: lowercase, single word (e.g., `services`, `models`)
- **Structs**: CamelCase (e.g., `EncryptionService`)
- **Functions**: CamelCase (e.g., `EncryptData`)
- **Variables**: camelCase (e.g., `phraseHash`)
- **Constants**: ALL_CAPS (e.g., `SALT_SIZE`)

### Error Handling
- Use `errors.New()` or `fmt.Errorf()` for creating errors
- Use error wrapping with `%w` for context
- Handle errors explicitly, don't ignore them

### Comments
- Use comments sparingly, focus on "why" not "what"
- Document public APIs and complex logic
- Keep comments up-to-date with code changes

### Security
- Never log sensitive information (passphrases, decrypted content)
- Use crypto/rand for all random values
- Validate all inputs from external sources

## Project Structure

- `main.go`: Application entry point
- `services/`: Business logic and encryption
- `models/`: Data structures (reference only)
- `middleware/`: Request validation
- `migrations/`: Database schema changes
- `pb_data/`: PocketBase data (not committed)