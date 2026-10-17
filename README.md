# Secret Notes Go Backend

🔐 A secure, self-hosted notes application with end-to-end encryption for both text notes and file attachments.

## ✨ Features

- **📝 Secure Notes**: Write and store text notes that only you can read.
- **📁 Encrypted Files**: Upload images or documents that are encrypted before storage.
- **⚡ Instant Access**: No sign-up or login process. Just type your passphrase and go.
- **🔒 Privacy First**: No accounts, no tracking, no personal data stored.
- **☁️ Self-Hosted**: You own the data and the infrastructure.

## 🛡️ How It Works

Secret Notes operates on a simple but powerful principle: **Your passphrase is your key.**

1.  **You Choose a Passphrase**: This can be anything—a word, a sentence, or a random string.
2.  **Your Client Encrypts Your Data**: The web, mobile, and CLI clients derive separate lookup, note, image, and metadata keys locally.
3.  **The Server Stores Only Ciphertext**: The v2 API receives a one-way lookup token and authenticated ciphertext—not the passphrase, encryption keys, note, image, filename, or media type.
4.  **Keys Stay on the Device**: Keys are held in memory for the active note and discarded when you switch notes.

When you open a note, the server returns ciphertext unchanged and the client decrypts it locally.

## Safe production rollout

Do not release either half of the v2 change by itself. Use this order:

1. Back up `pb_data` and external image storage, then verify that the backup can be read.
2. Deploy the backend and its `notes_v2` migration. The v2 API becomes available while legacy v1 clients temporarily continue to work.
3. Release the updated web, mobile, and CLI clients. Updated clients never fall back to sending a plaintext passphrase. Once a passphrase has a v2 note, legacy note and image routes return HTTP 426 instead of reading or changing the old copy. Legacy-only notes remain available during the compatibility window.
4. After client adoption is confirmed, set `SECRETNOTEZ_V1_MODE=disabled` and restart the backend. Legacy routes then return HTTP 426.

The temporary v1 compatibility window closes automatically at `2026-12-31T23:59:59Z`. Override it with the RFC3339 environment variable `SECRETNOTEZ_V1_DEADLINE` only to shorten the transition. Legacy recovery closes at `2027-03-31T23:59:59Z`; `SECRETNOTEZ_RECOVERY_DEADLINE` can shorten that window. Recovery exposes a deterministic legacy lookup oracle, so it must not remain enabled indefinitely.

To update a locally installed CLI from this checkout, run `go install ./cli/cmd/sn`. Existing binaries do not update automatically.

## 🔐 How Secure Is Your Data?

### **Your Data Is End-to-End Encrypted**
The server cannot decrypt v2 content. If someone obtained the database or image storage, they would see ciphertext and lookup hashes.
- **Memory-hard derivation**: Clients use the frozen v2 scrypt parameters, then HKDF-SHA256 key separation.
- **Authenticated encryption**: Notes and images use AES-256-GCM with fresh random nonces and purpose-bound authenticated data.
- **Exact passphrases**: Case, whitespace, Unicode, and all other UTF-8 bytes are preserved.

### **What We Don't Do**
- ❌ We don't store your real name, email, or IP address.
- ❌ We don't have a "master key" to unlock your notes.
- ❌ We can't reset your passphrase if you lose it.

**⚠️ Important**: Because we don't store your passphrase, **if you forget it, your data is gone forever.** We cannot recover it for you.

## 🤝 Contributing

We welcome contributions! If you're a developer looking to improve Secret Notes, please check out the codebase.

## 📄 License

This project is licensed under the MIT License.
