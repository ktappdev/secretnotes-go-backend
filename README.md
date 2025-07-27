# Secret Notes Go Backend

🔐 A secure, self-hosted notes application with end-to-end encryption for both text notes and file attachments. Built with Go and PocketBase, featuring passphrase-based encryption using AES-256-GCM.

## ✨ Features

- **🔒 End-to-End Encryption**: All notes and files encrypted with AES-256-GCM
- **🔑 Passphrase-Based Security**: No accounts needed - your passphrase is your key
- **📁 Encrypted File Storage**: Upload and encrypt any file type
- **🚀 Self-Hosted**: Deploy on your own infrastructure
- **⚡ Fast & Lightweight**: Built with Go and PocketBase
- **🌐 RESTful API**: Easy integration with any frontend
- **📱 Stateless Design**: No sessions or stored authentication tokens

## 🛡️ How Secure Is Your Data?

### 🔐 **Your Data is Completely Private**

**Even if someone gains full access to the server, database, and all files, they CANNOT read your data without your passphrase.**

Here's why:

#### **🔒 Military-Grade Encryption**
- **AES-256-GCM**: The same encryption standard used by governments and banks
- **Authenticated Encryption**: Prevents tampering - any modification breaks decryption
- **Unique Per-Operation**: Every note and file gets its own random salt and nonce

#### **🔑 Zero-Knowledge Architecture**
- **Server Cannot Decrypt**: The server never sees your passphrase or decrypted data
- **No Master Keys**: There are no "backdoors" or recovery mechanisms
- **Client-Side Key Derivation**: Your passphrase becomes the encryption key using PBKDF2

#### **🛡️ What Gets Stored**
```
✅ Encrypted Data: [random_salt][random_nonce][encrypted_content]
✅ Passphrase Hash: SHA-256 hash for lookup (cannot be reversed)
✅ Metadata: File names, content types (not sensitive)

❌ Your Passphrase: NEVER stored anywhere
❌ Decrypted Content: NEVER touches the database
❌ Encryption Keys: Generated on-demand, never stored
```

#### **🔍 Security Verification**
- **Open Source**: All encryption code is visible and auditable
- **Standard Libraries**: Uses Go's crypto package (not custom crypto)
- **No Network Transmission**: Decryption happens server-side, only encrypted data in database

### **🚨 What This Means For You**

✅ **If the server is hacked**: Your data remains encrypted and unreadable  
✅ **If the database is stolen**: Attackers get encrypted gibberish  
✅ **If we're subpoenaed**: We literally cannot provide your data  
✅ **If you forget your passphrase**: Your data is permanently lost (by design)  

### **⚠️ Your Responsibilities**

- **Use a strong passphrase** (32+ characters, unique, random)
- **Never share your passphrase** with anyone
- **Use HTTPS** in production (to protect passphrase in transit)
- **Keep backups** if you want to preserve data

### **🔬 Technical Security Details**

- **PBKDF2**: 100,000 iterations with SHA-256 for key derivation
- **Random Generation**: Uses crypto/rand for all random values
- **Memory Safety**: Go prevents buffer overflows and memory leaks
- **Constant-Time Operations**: Prevents timing attacks
- **No Logging**: Passphrases and decrypted content never logged

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/secret-notes-go.git
   cd secret-notes-go
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Run the server**:
   ```bash
   go run main.go serve
   ```

4. **Access the API**:
   - Server runs on `http://localhost:8090`
   - API endpoints available at `/api/secretnotes/`
   - Admin UI available at `http://localhost:8090/_/`

## 📖 API Documentation

### Authentication

No traditional authentication required. All operations use a passphrase (minimum 32 characters) that serves as both identifier and encryption key.

### Core Endpoints

#### Notes
- `GET /api/secretnotes/notes/{phrase}` - Get or create note
- `POST /api/secretnotes/notes/{phrase}` - Create new note
- `PATCH /api/secretnotes/notes/{phrase}` - Update note

#### Files
- `POST /api/secretnotes/notes/{phrase}/image` - Upload encrypted file
- `GET /api/secretnotes/notes/{phrase}/image` - Download decrypted file
- `DELETE /api/secretnotes/notes/{phrase}/image` - Delete file

### Example Usage

```bash
# Create a note
curl -X POST "http://localhost:8090/api/secretnotes/notes/your-very-long-secure-passphrase-here" \
  -H "Content-Type: application/json" \
  -d '{"title":"My Note","message":"Secret content"}'

# Upload a file
curl -X POST "http://localhost:8090/api/secretnotes/notes/your-very-long-secure-passphrase-here/image" \
  -F "image=@document.pdf"

# Download the file
curl "http://localhost:8090/api/secretnotes/notes/your-very-long-secure-passphrase-here/image" \
  -o downloaded-document.pdf
```

## 🏗️ Architecture

```
.
├── main.go                 # Main application entry point
├── migrations/
│   └── 001_init.go        # Database schema migrations
├── models/
│   ├── encrypted_file.go  # File model definitions
│   └── note.go           # Note model definitions
├── services/
│   ├── encryption.go     # AES-256-GCM encryption service
│   ├── file.go          # File handling service
│   └── note.go          # Note management service
├── middleware/
│   └── validation.go    # Request validation middleware
├── BACKEND_DOCS.md      # Detailed API documentation
└── FRONTEND_GUIDE.md    # Frontend integration guide
```

## 🔧 Configuration

### Environment Variables

- `PORT`: Server port (default: 8090)
- `DATA_DIR`: Data directory for PocketBase (default: ./pb_data)

### Production Deployment

1. **Use HTTPS**: Always deploy behind HTTPS in production
2. **Secure Headers**: Configure proper security headers
3. **Firewall**: Restrict access to necessary ports only
4. **Backups**: Regular backup of the `pb_data` directory
5. **Monitoring**: Set up logging and monitoring

## 📚 Documentation

- **[Backend Documentation](BACKEND_DOCS.md)**: Detailed API reference and architecture
- **[Frontend Integration Guide](FRONTEND_GUIDE.md)**: How to integrate with frontends

## 🧪 Development

### Running Tests

```bash
go test ./services/...
```

### Database Reset

```bash
rm -rf pb_data/
go run main.go serve
```

### Adding New Endpoints

1. Add route in `main.go`
2. Implement handler function
3. Update documentation
4. Add tests

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes
4. Add tests if applicable
5. Update documentation
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⚠️ Security Considerations

- **Passphrase Strength**: Use strong, unique passphrases (32+ characters)
- **HTTPS Only**: Never use over unencrypted connections in production
- **Regular Updates**: Keep dependencies updated
- **Backup Security**: Encrypt backups and store securely
- **Access Control**: Implement proper network-level access controls

## 🆘 Support

For security issues, please see [SECURITY.md](SECURITY.md).

For general questions and support:
- Open an issue on GitHub
- Check the documentation in `BACKEND_DOCS.md`
- Review the frontend integration guide in `FRONTEND_GUIDE.md`

---

**Built with ❤️ using Go and PocketBase**
# secretnotes-go-backend
