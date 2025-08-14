# Secret Notes Frontend Integration Guide

## Overview

This document provides guidance for frontend developers on how to integrate with the Secret Notes Go backend API for a React Native application. The backend uses custom API endpoints that handle encryption/decryption server-side.

## Base URL

**Production**: `https://secret-note-backend.lugetech.com`
**Local Development**: `http://localhost:8091`

All custom API endpoints are prefixed with `/api/secretnotes/`

## Authentication

The Secret Notes backend does not use traditional authentication. Instead, all operations are protected by a passphrase that must be provided as part of the URL path. This passphrase is used for both identification and encryption/decryption.

### Key Features

1. **Server-side encryption/decryption**: No need to implement encryption logic in the frontend
2. **Unified encryption**: Both notes and files use the same passphrase-based encryption
3. **Secure file handling**: Files are encrypted at rest and decrypted on-demand
4. **Simplified integration**: Just send passphrases and receive decrypted content
5. **Auto-creation**: Notes are automatically created when accessed if they don't exist

### Passphrase Requirements

- Minimum length: 3 characters
- Should be randomly generated for security
- Never stored on the client
- Sent only in URL paths (over HTTPS in production)

## API Endpoints

All endpoints are prefixed with `/api/secretnotes/`.

### Health Check

**Endpoint**: `GET /api/secretnotes/`

**Description**: Check if the API is running.

**Response**:
```json
{
  "message": "Secret Notes API is live",
  "version": "1.0.0"
}
```

### Get or Create Note

**Endpoint**: `GET /api/secretnotes/notes/{phrase}`

**Description**: Retrieves an existing note or automatically creates a new one with a welcome message if it doesn't exist.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 3 characters)

**Response**:
- **200 OK** if note exists
- **201 Created** if new note was created

```json
{
  "id": "record_id",
  "message": "decrypted_note_content",
  "hasImage": false,
  "created": "2023-01-01T00:00:00Z",
  "updated": "2023-01-01T00:00:00Z"
}
```

### Create Note (Explicit)

**Endpoint**: `POST /api/secretnotes/notes/{phrase}`

**Description**: Same behavior as GET - retrieves existing note or creates new one. Provided for semantic clarity.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 3 characters)

**Response**: Same as GET endpoint

### Update Note

**Endpoint**: `PATCH /api/secretnotes/notes/{phrase}`

**Description**: Updates an existing note's content. Returns 404 if note doesn't exist.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 3 characters)

**Request Body**:
```json
{
  "message": "Updated note content"
}
```

**Response**:
- **200 OK** on successful update
- **404 Not Found** if note doesn't exist

```json
{
  "id": "record_id",
  "message": "Updated note content",
  "hasImage": false,
  "created": "2023-01-01T00:00:00Z",
  "updated": "2023-01-01T00:00:01Z"
}
```

### Upsert Note (Recommended)

**Endpoint**: `PUT /api/secretnotes/notes/{phrase}`

**Description**: Creates a new note or updates existing one in a single call. This is the recommended endpoint for most use cases.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 3 characters)

**Request Body**:
```json
{
  "message": "Note content to save"
}
```

**Response**:
- **200 OK** if existing note was updated
- **201 Created** if new note was created

```json
{
  "id": "record_id",
  "message": "Note content to save",
  "hasImage": false,
  "created": "2023-01-01T00:00:00Z",
  "updated": "2023-01-01T00:00:00Z"
}
```

### Upload Image

**Endpoint**: `POST /api/secretnotes/notes/{phrase}/image`

**Description**: Upload an image associated with a note.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 3 characters)

**Request Body**: Multipart form data with the image file.

**Form Fields**:
- `image`: The image file to upload

**Response**:
```json
{
  "id": "file_id",
  "fileName": "image.jpg",
  "contentType": "image/jpeg",
  "created": "2023-01-01T00:00:00Z"
}
```

### Retrieve Image

**Endpoint**: `GET /api/secretnotes/notes/{phrase}/image`

**Description**: Retrieve an image associated with a note.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 3 characters)

**Response**: The decrypted image file (binary data)

**Content-Type**: The original content type of the image

### Delete Image

**Endpoint**: `DELETE /api/secretnotes/notes/{phrase}/image`

**Description**: Delete an image associated with a note.

**Path Parameters**:
- `phrase`: The passphrase for the note (minimum 3 characters)

**Response**:
```json
{
  "message": "Image deleted successfully"
}
```

## Error Handling

The API returns standard HTTP status codes:

- `200`: Success
- `400`: Bad request (e.g., missing parameters, invalid passphrase length)
- `404`: Note or image not found
- `500`: Internal server error

Error responses follow this format:
```json
{
  "error": "description of the error"
}
```

## Implementation Examples

### React Native Implementation

#### Creating/Updating a Note (Recommended Approach)

```javascript
// Using the PUT endpoint for upsert functionality
const saveNote = async (passphrase, message) => {
  try {
    const response = await fetch(`${BASE_URL}/api/secretnotes/notes/${encodeURIComponent(passphrase)}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ message }),
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    const result = await response.json();
    return result;
  } catch (error) {
    console.error('Error saving note:', error);
    throw error;
  }
};

// Usage
const passphrase = 'my-secure-passphrase-123';
const noteContent = 'This is my secret note content';
const savedNote = await saveNote(passphrase, noteContent);
```

#### Retrieving a Note

```javascript
const getNote = async (passphrase) => {
  try {
    const response = await fetch(`${BASE_URL}/api/secretnotes/notes/${encodeURIComponent(passphrase)}`);
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    const note = await response.json();
    return note;
  } catch (error) {
    console.error('Error retrieving note:', error);
    throw error;
  }
};

// Usage
const passphrase = 'my-secure-passphrase-123';
const note = await getNote(passphrase);
console.log('Note content:', note.message);
console.log('Has image:', note.hasImage);
```

#### Updating an Existing Note (PATCH)

```javascript
// Only use PATCH if you specifically need to update an existing note
// and want a 404 error if the note doesn't exist
const updateExistingNote = async (passphrase, message) => {
  try {
    const response = await fetch(`${BASE_URL}/api/secretnotes/notes/${encodeURIComponent(passphrase)}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ message }),
    });
    
    if (response.status === 404) {
      throw new Error('Note not found');
    }
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    const updatedNote = await response.json();
    return updatedNote;
  } catch (error) {
    console.error('Error updating note:', error);
    throw error;
  }
};
```

#### Working with Images

```javascript
// Upload an image
const uploadImage = async (passphrase, imageFile) => {
  try {
    const formData = new FormData();
    formData.append('image', imageFile);
    
    const response = await fetch(`${BASE_URL}/api/secretnotes/notes/${encodeURIComponent(passphrase)}/image`, {
      method: 'POST',
      body: formData,
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    const result = await response.json();
    return result;
  } catch (error) {
    console.error('Error uploading image:', error);
    throw error;
  }
};

// Retrieve an image
const getImage = async (passphrase) => {
  try {
    const response = await fetch(`${BASE_URL}/api/secretnotes/notes/${encodeURIComponent(passphrase)}/image`);
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    // Returns the image as a blob
    const imageBlob = await response.blob();
    return imageBlob;
  } catch (error) {
    console.error('Error retrieving image:', error);
    throw error;
  }
};

// Delete an image
const deleteImage = async (passphrase) => {
  try {
    const response = await fetch(`${BASE_URL}/api/secretnotes/notes/${encodeURIComponent(passphrase)}/image`, {
      method: 'DELETE',
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    const result = await response.json();
    return result;
  } catch (error) {
    console.error('Error deleting image:', error);
    throw error;
  }
};
```

### Complete React Native Example

```javascript
import React, { useState, useEffect } from 'react';
import { View, Text, TextInput, TouchableOpacity, Alert } from 'react-native';

const BASE_URL = 'https://secret-note-backend.lugetech.com';

const SecretNoteApp = () => {
  const [passphrase, setPassphrase] = useState('');
  const [noteContent, setNoteContent] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  
  const saveNote = async () => {
    if (passphrase.length < 3) {
      Alert.alert('Error', 'Passphrase must be at least 3 characters long');
      return;
    }
    
    setIsLoading(true);
    try {
      const response = await fetch(`${BASE_URL}/api/secretnotes/notes/${encodeURIComponent(passphrase)}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ message: noteContent }),
      });
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const result = await response.json();
      Alert.alert('Success', 'Note saved successfully!');
    } catch (error) {
      Alert.alert('Error', 'Failed to save note: ' + error.message);
    } finally {
      setIsLoading(false);
    }
  };
  
  const loadNote = async () => {
    if (passphrase.length < 3) {
      Alert.alert('Error', 'Passphrase must be at least 3 characters long');
      return;
    }
    
    setIsLoading(true);
    try {
      const response = await fetch(`${BASE_URL}/api/secretnotes/notes/${encodeURIComponent(passphrase)}`);
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      
      const note = await response.json();
      setNoteContent(note.message);
    } catch (error) {
      Alert.alert('Error', 'Failed to load note: ' + error.message);
    } finally {
      setIsLoading(false);
    }
  };
  
  return (
    <View style={{ padding: 20 }}>
      <Text>Secret Notes</Text>
      <TextInput
        placeholder="Enter passphrase (min 3 chars)"
        value={passphrase}
        onChangeText={setPassphrase}
        secureTextEntry
        style={{ borderWidth: 1, padding: 10, marginVertical: 10 }}
      />
      <TextInput
        placeholder="Enter note content"
        value={noteContent}
        onChangeText={setNoteContent}
        multiline
        style={{ borderWidth: 1, padding: 10, height: 100, marginVertical: 10 }}
      />
      <TouchableOpacity onPress={loadNote} disabled={isLoading}>
        <Text>Load Note</Text>
      </TouchableOpacity>
      <TouchableOpacity onPress={saveNote} disabled={isLoading}>
        <Text>Save Note</Text>
      </TouchableOpacity>
    </View>
  );
};

export default SecretNoteApp;
```

const record = await pb.collection('notes').create({
  phrase_hash: phraseHash,
  message: encryptedMessage,
  image_hash: ""
});
```

#### Retrieving Notes

```javascript
// Fetch notes with matching phrase hash
const records = await pb.collection('notes').getList(1, 50, {
  filter: `phrase_hash='${phraseHash}'`,
});

// Decrypt the message
if (records.items.length > 0) {
  const encryptedMessage = records.items[0].message;
  const decryptedMessage = await decryptWithPassphrase(encryptedMessage, passphrase);
  console.log(decryptedMessage);
}
```

#### Uploading an Image

```javascript
// First encrypt the image data
const encryptedImageData = await encryptWithPassphrase(await imageFile.arrayBuffer(), passphrase);

// Create a FormData object
const formData = new FormData();
formData.append('phrase_hash', phraseHash);
formData.append('file_name', imageFile.name);
formData.append('content_type', imageFile.type);
formData.append('file_data', new Blob([encryptedImageData]), 'encrypted_image');

// Upload the encrypted file
const record = await pb.collection('encrypted_files').create(formData);
```

#### Retrieving an Image

```javascript
// Fetch encrypted files with matching phrase hash
const records = await pb.collection('encrypted_files').getList(1, 1, {
  filter: `phrase_hash='${phraseHash}'`,
});

if (records.items.length > 0) {
  const fileRecord = records.items[0];
  
  // Get the file URL
  const fileUrl = pb.files.getUrl(fileRecord, fileRecord.file_data);
  
  // Fetch the encrypted file
  const response = await fetch(fileUrl);
  const encryptedData = await response.arrayBuffer();
  
  // Decrypt the file data
  const decryptedData = await decryptWithPassphrase(encryptedData, passphrase);
  
  // Create a blob URL for display
  const blob = new Blob([decryptedData], { type: fileRecord.content_type });
  const objectUrl = URL.createObjectURL(blob);
  
  // Display the image
  document.getElementById('image').src = objectUrl;
}
```

### Retrieving an Image

```javascript
const response = await fetch(`/api/secretnotes/notes/${passphrase}/image`);
const imageBlob = await response.blob();
const imageUrl = URL.createObjectURL(imageBlob);
```

### Deleting an Image

```javascript
const response = await fetch(`/api/secretnotes/notes/${passphrase}/image`, {
  method: 'DELETE',
});

const result = await response.json();
```

## Security Considerations

1. **HTTPS Only**: Always use HTTPS in production to protect passphrases in transit.
2. **Passphrase Storage**: Never store passphrases in local storage or any persistent storage.
3. **Memory Management**: Clear passphrase variables from memory when no longer needed.
4. **Input Validation**: Always validate passphrase length (minimum 3 characters) on the client side.
5. **URL Encoding**: Always use `encodeURIComponent()` when including passphrases in URLs.
6. **Error Handling**: Implement proper error handling for all API calls.
7. **Rate Limiting**: Implement client-side rate limiting to avoid overwhelming the server.

## Best Practices

1. **Use PUT for Most Operations**: The `PUT /api/secretnotes/notes/{phrase}` endpoint is recommended for most use cases as it handles both creation and updates.
2. **Passphrase Generation**: Use a cryptographically secure random generator for passphrases.
3. **Error Handling**: Always implement proper try-catch blocks and user-friendly error messages.
4. **Loading States**: Show loading indicators during API calls for better UX.
5. **Input Validation**: Validate passphrase length and content before making API calls.
6. **Accessibility**: Implement full accessibility support for all UI components.
7. **Offline Handling**: Consider implementing offline detection and appropriate user feedback.

## API Endpoint Summary

| Method | Endpoint | Description | Use Case |
|--------|----------|-------------|----------|
| `GET` | `/api/secretnotes/notes/{phrase}` | Get existing note or create new one | Initial note access |
| `POST` | `/api/secretnotes/notes/{phrase}` | Same as GET (semantic clarity) | Alternative to GET |
| `PUT` | `/api/secretnotes/notes/{phrase}` | Create or update note | **Recommended for most use cases** |
| `PATCH` | `/api/secretnotes/notes/{phrase}` | Update existing note only | When you need 404 for missing notes |
| `POST` | `/api/secretnotes/notes/{phrase}/image` | Upload image | Image upload |
| `GET` | `/api/secretnotes/notes/{phrase}/image` | Retrieve image | Image download |
| `DELETE` | `/api/secretnotes/notes/{phrase}/image` | Delete image | Image removal |

## Support

For questions or issues with the API, please refer to the backend documentation or contact the development team.
