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
2.  **We Encrypt Your Data**: When you save a note, your passphrase is used to encrypt the content immediately.
3.  **We Store Only the Lock**: We save the encrypted "lock" (your note) and a one-way fingerprint of your passphrase so we can find it later.
4.  **We Forget the Key**: We **never** store your passphrase or the encryption keys. We discard them immediately after the operation is done.

When you want to read your note, you provide the passphrase again. We use it to unlock the note on the fly and send the content back to you.

## 🔐 How Secure Is Your Data?

### **Your Data is Safe at Rest**
If someone were to steal the database or hard drives, they would see only gibberish.
- **No Stored Passwords**: We verify your identity using a secure hash (SHA-256), meaning we can't reverse-engineer your passphrase from our database.
- **Strong Encryption**: We use **AES-256-GCM**, a military-grade encryption standard, to lock your files and text.
- **Unique Keys**: Every single note and file is encrypted with a unique, randomly generated salt and nonce.

### **What We Don't Do**
- ❌ We don't store your real name, email, or IP address.
- ❌ We don't have a "master key" to unlock your notes.
- ❌ We can't reset your passphrase if you lose it.

**⚠️ Important**: Because we don't store your passphrase, **if you forget it, your data is gone forever.** We cannot recover it for you.

## 🤝 Contributing

We welcome contributions! If you're a developer looking to improve Secret Notes, please check out the codebase.

## 📄 License

This project is licensed under the MIT License.
