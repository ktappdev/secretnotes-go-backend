# Secret Notes Frontend Developer Guide

🔐 **Welcome to the Secret Notes Project!** This guide will help you build an exceptional frontend for our zero-knowledge, encrypted notes application.

## 🎯 Project Overview

Secret Notes is a **zero-knowledge encrypted notes app** where users can store text notes and file attachments using only a passphrase—no accounts, no logins, just pure encryption magic. Think of it as a "digital safe" where only the person with the passphrase can access the contents.

### 🔑 Core Concept
- **One Passphrase = One Secure Vault**: Each passphrase creates an isolated, encrypted space
- **Zero Knowledge**: Even if hackers get the database, they can't read your data without your passphrase
- **No Accounts**: No usernames, emails, or sign-ups—just your passphrase
- **Military-Grade Security**: AES-256-GCM encryption with PBKDF2 key derivation

## 🏗️ Architecture Overview

```
┌─────────────────┐    HTTPS     ┌─────────────────┐    Encrypted    ┌─────────────────┐
│   React App     │ ◄──────────► │   Go Backend    │ ◄─────────────► │   PocketBase    │
│  (Your Code)    │   REST API   │  (Encryption)   │   File Storage  │   (Database)    │
└─────────────────┘              └─────────────────┘                 └─────────────────┘
```

**Data Flow:**
1. User enters passphrase in your app
2. Your app sends passphrase via `X-Passphrase` header
3. Backend encrypts/decrypts data server-side
4. Your app receives plain text content

## 🚀 Getting Started

### Prerequisites
- Node.js 18+ or React Native development environment
- Backend running at `http://127.0.0.1:8091` (use `go run main.go`)
- Basic understanding of REST APIs and file uploads

### Quick Test
```bash
# Test if backend is running
curl http://127.0.0.1:8091/api/secretnotes/

# Expected response:
# {"message":"Secret Notes API is live","version":"1.0.0"}
```

## 🛡️ Security Model (Important!)

### What Makes This Secure
1. **Server-Side Encryption**: All encryption happens on the backend
2. **Passphrase-Based**: Your passphrase is the ONLY key to your data
3. **No Storage of Secrets**: Passphrases are never stored anywhere
4. **Hash-Based Lookup**: We use SHA-256(passphrase) to find your data
5. **Unique Encryption**: Every note gets a unique salt and nonce

### Your Frontend Responsibilities
- ✅ **Use HTTPS in production** (protects passphrase in transit)
- ✅ **Validate passphrase length** (minimum 3 characters, recommend 12+)
- ✅ **Show clear error messages** for wrong passphrases
- ✅ **Implement secure passphrase input** (password fields, no autocomplete)
- ❌ **Don't store passphrases** (not even in local storage)
- ❌ **Don't log passphrases** (even for debugging)

## 🔌 API Reference

### Base URL
- **Development**: `http://127.0.0.1:8091/api/secretnotes`
- **Production**: `https://your-domain.com/api/secretnotes`

### Authentication
No traditional auth! Just include the passphrase in every request:

```javascript
const headers = {
  'X-Passphrase': userPassphrase,
  'Content-Type': 'application/json'
}
```

### 📝 Notes API

#### 1. Get or Create Note
```javascript
// GET /api/secretnotes/notes
const response = await fetch(`${baseUrl}/api/secretnotes/notes`, {
  method: 'GET',
  headers: {
    'X-Passphrase': passphrase
  }
});

// Response (200 for existing, 201 for new):
{
  "id": "abc123",
  "message": "Your decrypted note content",
  "hasImage": false,
  "created": "2023-01-01T00:00:00Z",
  "updated": "2023-01-01T00:00:00Z"
}
```

#### 2. Update Note (PATCH)
```javascript
// PATCH /api/secretnotes/notes
const response = await fetch(`${baseUrl}/api/secretnotes/notes`, {
  method: 'PATCH',
  headers: {
    'X-Passphrase': passphrase,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    message: "Updated note content"
  })
});
```

#### 3. Create/Replace Note (PUT)
```javascript
// PUT /api/secretnotes/notes - Creates new or replaces existing
const response = await fetch(`${baseUrl}/api/secretnotes/notes`, {
  method: 'PUT',
  headers: {
    'X-Passphrase': passphrase,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    message: "Complete note content"
  })
});
```

### 📎 File Attachment API

#### 1. Upload File
```javascript
// POST /api/secretnotes/notes/image
const formData = new FormData();
formData.append('image', fileInput.files[0]);

const response = await fetch(`${baseUrl}/api/secretnotes/notes/image`, {
  method: 'POST',
  headers: {
    'X-Passphrase': passphrase
  },
  body: formData
});

// Response:
{
  "message": "Image uploaded successfully",
  "fileName": "document.pdf",
  "fileSize": 12345,
  "contentType": "application/pdf",
  "fileHash": "sha256_hash_of_encrypted_file"
}
```

#### 2. Download File
```javascript
// GET /api/secretnotes/notes/image
const response = await fetch(`${baseUrl}/api/secretnotes/notes/image`, {
  method: 'GET',
  headers: {
    'X-Passphrase': passphrase
  }
});

if (response.ok) {
  const blob = await response.blob();
  const contentType = response.headers.get('Content-Type');
  const filename = response.headers.get('Content-Disposition')
    ?.match(/filename="(.+)"/)?.[1] || 'download';
  
  // Create download link
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
}
```

#### 3. Delete File
```javascript
// DELETE /api/secretnotes/notes/image
const response = await fetch(`${baseUrl}/api/secretnotes/notes/image`, {
  method: 'DELETE',
  headers: {
    'X-Passphrase': passphrase
  }
});
```

## 🎨 UI/UX Recommendations

### 🔐 Passphrase Input Design
```jsx
// Recommended passphrase input component
const PassphraseInput = ({ value, onChange, onSubmit }) => (
  <div className="passphrase-container">
    <input
      type="password"
      value={value}
      onChange={onChange}
      onKeyPress={(e) => e.key === 'Enter' && onSubmit()}
      placeholder="Enter your secure passphrase..."
      autoComplete="new-password" // Prevent browser saving
      className="passphrase-input"
      minLength={3}
      required
    />
    <div className="passphrase-strength">
      {value.length >= 12 ? "🟢 Strong" : 
       value.length >= 6  ? "🟡 Moderate" : 
       value.length >= 3  ? "🔴 Weak" : ""}
    </div>
  </div>
);
```

### 📝 Note Editor Interface
```jsx
const NoteEditor = ({ note, onSave, passphrase }) => {
  const [content, setContent] = useState(note?.message || '');
  const [isSaving, setIsSaving] = useState(false);
  
  const handleSave = async () => {
    setIsSaving(true);
    try {
      await saveNote(passphrase, content);
      // Show success feedback
    } catch (error) {
      // Show error (likely wrong passphrase)
    }
    setIsSaving(false);
  };

  return (
    <div className="note-editor">
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder="Start writing your secure note..."
        className="note-textarea"
        rows={10}
      />
      <div className="editor-actions">
        <button onClick={handleSave} disabled={isSaving}>
          {isSaving ? 'Saving...' : 'Save Note'}
        </button>
        <div className="character-count">{content.length} characters</div>
      </div>
    </div>
  );
};
```

### 📎 File Upload Component
```jsx
const FileUpload = ({ passphrase, onUploadComplete }) => {
  const [isUploading, setIsUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  
  const handleFileUpload = async (file) => {
    if (!file) return;
    
    setIsUploading(true);
    const formData = new FormData();
    formData.append('image', file);
    
    try {
      const response = await fetch(`${baseUrl}/api/secretnotes/notes/image`, {
        method: 'POST',
        headers: { 'X-Passphrase': passphrase },
        body: formData
      });
      
      if (response.ok) {
        const result = await response.json();
        onUploadComplete(result);
      }
    } catch (error) {
      console.error('Upload failed:', error);
    }
    setIsUploading(false);
  };
  
  return (
    <div 
      className={`file-upload ${dragOver ? 'drag-over' : ''}`}
      onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
      onDragLeave={() => setDragOver(false)}
      onDrop={(e) => {
        e.preventDefault();
        setDragOver(false);
        handleFileUpload(e.dataTransfer.files[0]);
      }}
    >
      {isUploading ? (
        <div>Uploading...</div>
      ) : (
        <>
          <div>Drop file here or</div>
          <input 
            type="file" 
            onChange={(e) => handleFileUpload(e.target.files[0])}
            accept="*/*"
          />
        </>
      )}
    </div>
  );
};
```

## 🔧 Error Handling

### Common API Errors
```javascript
const handleApiError = async (response) => {
  if (!response.ok) {
    const error = await response.json();
    switch (response.status) {
      case 400:
        return "Invalid passphrase (minimum 3 characters required)";
      case 404:
        return "No data found for this passphrase";
      case 413:
        return "File too large (max ~8GB)";
      case 500:
        return "Server error - please try again";
      default:
        return error.error || "Unknown error occurred";
    }
  }
};

// Usage example:
try {
  const response = await fetch(url, options);
  if (!response.ok) {
    const errorMessage = await handleApiError(response);
    setError(errorMessage);
    return;
  }
  // Handle success...
} catch (networkError) {
  setError("Network error - check your connection");
}
```

### Wrong Passphrase Detection
```javascript
const isWrongPassphrase = (error) => {
  return error.includes("not found") || 
         error.includes("decrypt") ||
         error.includes("invalid");
};

// Show user-friendly message:
if (isWrongPassphrase(errorMessage)) {
  setError("Incorrect passphrase. Please check and try again.");
}
```

## 📱 React Native Considerations

### File Handling
```javascript
import DocumentPicker from 'react-native-document-picker';
import RNFS from 'react-native-fs';

const uploadFile = async (passphrase) => {
  try {
    const result = await DocumentPicker.pickSingle({
      type: [DocumentPicker.types.allFiles],
    });
    
    const formData = new FormData();
    formData.append('image', {
      uri: result.uri,
      type: result.type,
      name: result.name,
    });
    
    const response = await fetch(`${baseUrl}/api/secretnotes/notes/image`, {
      method: 'POST',
      headers: {
        'X-Passphrase': passphrase,
        'Content-Type': 'multipart/form-data',
      },
      body: formData,
    });
    
    return await response.json();
  } catch (error) {
    console.error('File upload error:', error);
  }
};
```

### Secure Storage
```javascript
// For React Native - NEVER store passphrases!
import AsyncStorage from '@react-native-async-storage/async-storage';

// ❌ DON'T DO THIS:
// AsyncStorage.setItem('passphrase', passphrase);

// ✅ Only store non-sensitive settings:
AsyncStorage.setItem('theme', 'dark');
AsyncStorage.setItem('lastUsedDate', Date.now());
```

## 🎭 User Experience Flow

### 1. First Time User
```
1. Show welcome screen explaining the concept
2. Prompt for secure passphrase creation
3. Show passphrase strength indicator
4. Explain "no recovery" warning
5. Create first note automatically
6. Show tutorial/tips
```

### 2. Returning User
```
1. Show passphrase input immediately
2. Load note on successful passphrase entry
3. Show file attachment if exists
4. Enable editing and file upload
```

### 3. Error States
```
- Wrong passphrase → Clear, helpful error message
- Network error → Retry button and offline indicator
- Large file → Progress indicator and size warning
- Empty passphrase → Inline validation message
```

## 💡 Advanced Features to Implement

### 🚀 Priority 1 (Core Features)
- [ ] **Passphrase validation** (length, strength indicator)
- [ ] **Auto-save** notes while typing (debounced)
- [ ] **File type icons** and size display
- [ ] **Download/preview files** inline
- [ ] **Copy note content** to clipboard
- [ ] **Character/word count** for notes
- [ ] **Mobile-responsive design**

### 🌟 Priority 2 (Nice to Have)
- [ ] **Dark/light theme** toggle
- [ ] **Offline detection** and queuing
- [ ] **Markdown support** in notes
- [ ] **Rich text editor** (bold, italic, lists)
- [ ] **Multiple file attachments** (when backend supports it)
- [ ] **Drag & drop** file uploads
- [ ] **Progressive Web App** features

### 🔮 Priority 3 (Future)
- [ ] **Note templates** (meeting notes, journal, etc.)
- [ ] **Export functionality** (encrypted backup)
- [ ] **QR code sharing** of passphrases
- [ ] **Passphrase generation** helper
- [ ] **Note versioning/history** (if backend adds support)

## 🧪 Testing Strategy

### Unit Tests
```javascript
// Test API helper functions
describe('Secret Notes API', () => {
  test('should format headers correctly', () => {
    const headers = createHeaders('test-passphrase');
    expect(headers['X-Passphrase']).toBe('test-passphrase');
  });

  test('should handle API errors gracefully', () => {
    const error = handleApiError({ status: 404 });
    expect(error).toBe('No data found for this passphrase');
  });
});
```

### Integration Tests
```javascript
// Test full user flows
describe('Note Management', () => {
  test('should create and retrieve note', async () => {
    const passphrase = 'test-passphrase-123';
    const message = 'Test note content';
    
    // Create note
    const createResponse = await createNote(passphrase, message);
    expect(createResponse.message).toBe(message);
    
    // Retrieve note
    const getResponse = await getNote(passphrase);
    expect(getResponse.message).toBe(message);
  });
});
```

### Security Tests
```javascript
describe('Security', () => {
  test('should not store passphrase in localStorage', () => {
    const passphrase = 'secret123';
    // ... simulate user interaction
    expect(localStorage.getItem('passphrase')).toBeNull();
  });

  test('should clear passphrase from memory on unmount', () => {
    // Test component cleanup
  });
});
```

## 🎯 Performance Optimization

### API Request Optimization
```javascript
// Debounce saves to avoid spam
import { debounce } from 'lodash';

const debouncedSave = debounce(async (passphrase, content) => {
  await saveNote(passphrase, content);
}, 1000);

// Usage in component:
useEffect(() => {
  if (noteContent.length > 0) {
    debouncedSave(passphrase, noteContent);
  }
}, [noteContent]);
```

### File Upload Progress
```javascript
const uploadWithProgress = async (file, passphrase, onProgress) => {
  const formData = new FormData();
  formData.append('image', file);

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    
    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) {
        const progress = (event.loaded / event.total) * 100;
        onProgress(progress);
      }
    });

    xhr.addEventListener('load', () => {
      if (xhr.status === 200) {
        resolve(JSON.parse(xhr.responseText));
      } else {
        reject(new Error('Upload failed'));
      }
    });

    xhr.open('POST', `${baseUrl}/api/secretnotes/notes/image`);
    xhr.setRequestHeader('X-Passphrase', passphrase);
    xhr.send(formData);
  });
};
```

## 🎨 Styling and Branding

### Design System Colors
```css
:root {
  /* Security/Trust Colors */
  --secure-green: #10B981;
  --warning-yellow: #F59E0B;
  --danger-red: #EF4444;
  
  /* Main Brand Colors */
  --primary-blue: #3B82F6;
  --secondary-purple: #8B5CF6;
  
  /* Neutral Colors */
  --dark-bg: #1F2937;
  --light-bg: #F9FAFB;
  --text-primary: #111827;
  --text-secondary: #6B7280;
  
  /* Encryption/Security Theme */
  --lock-gold: #D97706;
  --safe-blue: #1E40AF;
}
```

### Component Styling Examples
```css
/* Passphrase Input */
.passphrase-input {
  border: 2px solid var(--primary-blue);
  border-radius: 8px;
  padding: 12px 16px;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 16px;
  transition: all 0.2s ease;
}

.passphrase-input:focus {
  outline: none;
  border-color: var(--secure-green);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

/* Security Indicator */
.security-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  font-size: 14px;
}

.security-indicator.strong { color: var(--secure-green); }
.security-indicator.moderate { color: var(--warning-yellow); }
.security-indicator.weak { color: var(--danger-red); }

/* File Upload Zone */
.file-upload {
  border: 2px dashed var(--text-secondary);
  border-radius: 8px;
  padding: 40px;
  text-align: center;
  transition: all 0.2s ease;
  cursor: pointer;
}

.file-upload.drag-over {
  border-color: var(--primary-blue);
  background-color: rgba(59, 130, 246, 0.05);
}

.file-upload:hover {
  border-color: var(--primary-blue);
}
```

## 🚀 Deployment Considerations

### Environment Configuration
```javascript
// config.js
const config = {
  development: {
    apiBaseUrl: 'http://127.0.0.1:8091/api/secretnotes',
    enableDebugLogs: true,
  },
  production: {
    apiBaseUrl: 'https://your-domain.com/api/secretnotes',
    enableDebugLogs: false,
  }
};

export default config[process.env.NODE_ENV || 'development'];
```

### Security Headers (Production)
```javascript
// Recommended security headers for your frontend
const securityHeaders = {
  'Content-Security-Policy': "default-src 'self'; script-src 'self' 'unsafe-inline'",
  'X-Frame-Options': 'DENY',
  'X-Content-Type-Options': 'nosniff',
  'Referrer-Policy': 'strict-origin-when-cross-origin',
  'Permissions-Policy': 'camera=(), microphone=(), location=()',
};
```

## 🐛 Debugging Tips

### API Debug Helper
```javascript
const debugApi = (url, options) => {
  console.group('🔍 API Call');
  console.log('URL:', url);
  console.log('Method:', options.method || 'GET');
  console.log('Headers:', options.headers);
  console.log('Body:', options.body);
  console.groupEnd();
};

// Usage (only in development):
if (process.env.NODE_ENV === 'development') {
  debugApi(url, options);
}
```

### Network Tab Inspection
When debugging API calls, look for:
- ✅ `X-Passphrase` header is present and not empty
- ✅ `Content-Type` matches what you're sending
- ✅ Response status codes (200, 201, 404, etc.)
- ✅ CORS headers if running on different ports

## 📚 Additional Resources

### Learning Resources
- [Fetch API Documentation](https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API)
- [FormData for File Uploads](https://developer.mozilla.org/en-US/docs/Web/API/FormData)
- [Web Security Best Practices](https://web.dev/secure/)
- [React Hook Patterns](https://reactjs.org/docs/hooks-intro.html)

### Backend API Testing
Use the provided Bruno tests to understand the API:
```bash
# In bruno-tests/ directory
./Health_Check.bru           # Test if API is running
./notes_get_or_create.bru    # Test note creation/retrieval
./images_upload.bru          # Test file upload
./images_get.bru             # Test file download
```

### Security Resources
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Web Crypto API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Crypto_API)
- [Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)

## 🎯 Success Metrics

Your frontend should achieve:
- ✅ **Zero passphrase storage** (never stored anywhere)
- ✅ **Intuitive UX** (users understand the security model)
- ✅ **Fast performance** (sub-second note saves)
- ✅ **Mobile responsive** (works on all screen sizes)
- ✅ **Error resilience** (graceful handling of network issues)
- ✅ **File support** (upload/download any file type)
- ✅ **Accessibility** (keyboard navigation, screen readers)

## 🎉 You're All Set!

You now have everything you need to build an amazing frontend for Secret Notes. The backend handles all the complex encryption, so you can focus on creating an intuitive, beautiful user experience.

### Quick Start Checklist
- [ ] Set up your React/React Native project
- [ ] Implement passphrase input with validation
- [ ] Create note editor with auto-save
- [ ] Add file upload/download functionality  
- [ ] Implement proper error handling
- [ ] Test with the Bruno collection
- [ ] Style with security/trust theme
- [ ] Deploy with HTTPS

### Questions or Suggestions?

This backend API is designed to be frontend-agnostic and developer-friendly. If you need additional endpoints, different response formats, or have ideas for improvements, the backend can be easily extended.

**Remember**: The user's passphrase is sacred—treat it with the utmost care and never store it anywhere. The beauty of this system is its simplicity and security through zero-knowledge architecture.

Happy coding! 🔐✨