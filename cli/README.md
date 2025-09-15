# SecretNotes CLI

A simple, secure TUI for a single encrypted note tied to your passphrase.

Concept
- One passphrase → one note. Change the passphrase and you see a different note.
- No list, no accounts. Just load, edit, and save your note.
- Images exist in the backend but are intentionally omitted in this CLI v1 (text-only).

Keybindings
- Ctrl+S: Save the note
- Ctrl+P: Change passphrase (focuses prompt, Enter to reload)
- Ctrl+T: Toggle Plain view (shows only your text, no UI chrome)
- Ctrl+Y: Copy note content to clipboard
- Ctrl+Q or Ctrl+C: Quit

Features
- First-run setup storing non-sensitive config in the user config directory
- Passphrase prompt at start (never stored); in-app passphrase change (Ctrl+P)
- Load note via GET /api/secretnotes/notes (creates if missing)
- Save via PATCH /api/secretnotes/notes with {"message": "..."}
- Optional debounced autosave
- Plain view toggle for clean selection/copy (Ctrl+T)
- One-shot copy to clipboard (Ctrl+Y)

Build
- cd cli
- go mod tidy
- go build ./cmd/sn

Run
- ./sn
- ./sn --url http://127.0.0.1:8091 --autosave --autosave-debounce-ms 1200
- ./sn --insecure (for dev HTTPS with self-signed certs)

First run
- You’ll be asked:
  - Server name (default: local)
  - Server URL (default: http://127.0.0.1:8091)
  - TLS verification (only for https)
  - Autosave and debounce settings
- After setup, you’ll be prompted for your passphrase (masked). The note for that passphrase is loaded.

Change passphrase during a session
- Press Ctrl+P to focus the passphrase prompt, enter a new passphrase, then press Enter.
- The editor reloads to show the note for that passphrase.

Copying your text (no UI borders or line numbers)
- Press Ctrl+T to enable Plain view. Only your note text is shown, so you can select and copy without any UI lines.
- Or press Ctrl+Y to copy the full note content directly to your clipboard.

Config paths
- macOS: ~/Library/Application Support/SecretNotes/config.json
- Linux/other: platform-specific os.UserConfigDir path (e.g., ~/.config/SecretNotes/config.json)
- Windows: %AppData%\SecretNotes\config.json

Security
- Passphrase lives only in memory for the session and is zeroed on exit.
- No passphrases in config or logs.

Troubleshooting
- If you see a server health warning, ensure your backend is running and the URL is correct.
- Saving issues often mean the passphrase doesn’t match the note or the server is unreachable.
