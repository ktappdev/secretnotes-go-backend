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
- Alt+S: Toggle Autosave (persists to config)
- Ctrl+Y: Copy note content to clipboard
- Ctrl+Q: Quit and wipe screen + scrollback (privacy)
- Ctrl+C: Quit and clear screen only

Features
- First-run setup storing non-sensitive config in the user config directory
- Passphrase prompt at start (never stored); in-app passphrase change (Ctrl+P)
- Load note via GET /api/secretnotes/notes (creates if missing)
- Save via PATCH /api/secretnotes/notes with {"message": "..."}
- Autosave ON by default (1200ms debounce), toggleable in-app and persisted
- Plain view toggle for clean selection/copy (Ctrl+T)
- One-shot copy to clipboard (Ctrl+Y)

Build
- cd cli
- go mod tidy
- go build ./cmd/sn

Run
- ./sn
- ./sn --url https://secret-note-backend.lugetech.com --autosave --autosave-debounce-ms 1200
- ./sn --insecure (only if using dev HTTPS with self-signed certs)

Install
- Requirements: Go 1.22+
- Local (from this repo):
  - cd cli && go install ./cmd/sn
- From GitHub (always latest from main):
  ```bash
  go install github.com/ktappdev/secretnotes-go-backend/cli/cmd/sn@main
  ```
  - Alternative (@latest may be cached by proxies):
    ```bash
    go install github.com/ktappdev/secretnotes-go-backend/cli/cmd/sn@latest
    # If @latest fails due to proxy cache:
    go env -w GOPROXY=direct
    go install github.com/ktappdev/secretnotes-go-backend/cli/cmd/sn@latest
    go env -u GOPROXY
    ```

Autosave
- Default: ON (1200 ms debounce)
- Toggle in the app: press Alt+S (persisted to config immediately)
- One-time for this run: ./sn --autosave (optional: --autosave-debounce-ms 1200)
- You can also edit preferences.autosaveEnabled and preferences.autosaveDebounceMs in the config

First run
- You’ll be asked:
  - Server name (default: local)
- Server URL (default: https://secret-note-backend.lugetech.com)
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
- No recovery: If you forget your passphrase, your note is permanently unrecoverable.

Privacy & exit behavior
- Ctrl+Q: wipes the screen and scrollback to remove any trace of your note
- Ctrl+C: clears the visible screen but preserves scrollback history
- Terminal support varies: not all terminals honor scrollback clear (CSI 3 J). If privacy is critical, manually clear or close your terminal after quitting. For stubborn terminals, set SN_WIPE_AGGRESSIVE=1 when launching.

Why clear the screen?
- Your note content can remain in the terminal’s visible buffer or scrollback history
- Screen recording, streaming, or remote support tools can capture that history
- Clearing (and optionally wiping scrollback) reduces the chance of accidental disclosure

Troubleshooting
- Status shows Connected/Offline; the backend host is not displayed.
- If you see a server health warning, ensure your backend is running and the URL is correct.
- Saving issues often mean the passphrase doesn’t match the note or the server is unreachable.
