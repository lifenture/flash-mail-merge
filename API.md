# Flash Mail Merge API Reference

This document provides detailed technical documentation for the Flash Mail Merge API endpoints.

## Base URL

```
https://your-api-gateway-url.amazonaws.com/stage
```

Replace `your-api-gateway-url` with your actual API Gateway URL and `stage` with your deployment stage (dev, staging, prod).

## Authentication

All API endpoints require authentication using API Keys:

```http
x-api-key: your-api-key-here
```

### Rate Limits

- **Rate Limit**: 100 requests per second
- **Burst Limit**: 200 requests
- **Daily Quota**: 10,000 requests per day

## Content Type

All requests must include the following header:

```http
Content-Type: application/json
```

---

## Endpoints

### 1. POST `/merge` - Field Detection and Mail Merge

Extracts merge fields from DOCX documents and optionally performs mail merge operations.

#### Request

**Headers:**
```http
Content-Type: application/json
x-api-key: your-api-key
```

**Body Schema:**
```json
{
  "docx": "string",           // Required: Base64-encoded DOCX file
  "data": {                   // Optional: Merge data key-value pairs
    "field1": "value1",
    "field2": "value2"
  }
}
```

#### Response

**Success Response (200 OK):**
```json
{
  "validation": {
    "valid": true,
    "errors": [],
    "warnings": []
  },
  "mergedDocument": "base64-encoded-docx",  // Only present when data provided
  "skippedFields": []                      // Only present when data provided
}
```

**Validation Error Response (400 Bad Request):**
```json
{
  "validation": {
    "valid": false,
    "errors": [
      "Required field 'FirstName' is missing",
      "Field 'Email' has invalid format"
    ],
    "warnings": [
      "Duplicate key 'FirstName' detected in JSON data (first occurrence kept)"
    ]
  }
}
```

#### Error Responses

**400 Bad Request:**
```json
{
  "error": "Invalid input"
}
```

**400 Bad Request:**
```json
{
  "error": "'docx' key missing"
}
```

**400 Bad Request:**
```json
{
  "error": "Failed to decode base64 input"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Failed to process document"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Failed to extract fields"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Failed to perform merge"
}
```

#### Example Usage

**Field Detection Only:**
```bash
curl -X POST https://your-api-gateway-url.amazonaws.com/dev/merge \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
  -d '{
    "docx": "UEsDBAoAAAAAAJZQV1cAAAAAAAAAAAAAAAAJAAAAZG9jUHJvcHMv..."
  }'
```

**Field Detection + Mail Merge:**
```bash
curl -X POST https://your-api-gateway-url.amazonaws.com/dev/merge \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
  -d '{
    "docx": "UEsDBAoAAAAAAJZQV1cAAAAAAAAAAAAAAAAJAAAAZG9jUHJvcHMv...",
    "data": {
      "FirstName": "John",
      "LastName": "Doe",
      "Email": "john.doe@example.com",
      "CompanyName": "Acme Corp"
    }
  }'
```

---

### 2. POST `/detect` - Field Extraction Only

Extracts merge fields from DOCX documents without validation or merge operations. This endpoint is optimized for quickly discovering what fields are available in a document.

#### Request

**Headers:**
```http
Content-Type: application/json
x-api-key: your-api-key
```

**Body Schema:**
```json
{
  "docx": "string"    // Required: Base64-encoded DOCX file
}
```

#### Response

**Success Response (200 OK):**
```json
{
  "data": {
    "FirstName": "",
    "LastName": "",
    "Email": "",
    "CompanyName": "",
    "Address": "",
    "PhoneNumber": ""
  }
}
```

#### Error Responses

**400 Bad Request:**
```json
{
  "error": "Invalid input"
}
```

**400 Bad Request:**
```json
{
  "error": "'docx' key missing"
}
```

**400 Bad Request:**
```json
{
  "error": "Failed to decode base64 input"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Failed to process document"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Failed to extract fields"
}
```

#### Example Usage

```bash
curl -X POST https://your-api-gateway-url.amazonaws.com/dev/detect \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
  -d '{
    "docx": "UEsDBAoAAAAAAJZQV1cAAAAAAAAAAAAAAAAJAAAAZG9jUHJvcHMv..."
  }'
```

---

## Error Handling

### HTTP Status Codes

- **200 OK**: Request successful
- **400 Bad Request**: Invalid request data, missing required fields, or validation errors
- **500 Internal Server Error**: Server-side processing error

### Error Response Format

All error responses follow a consistent format:

```json
{
  "error": "Human-readable error message"
}
```

### Common Error Scenarios

1. **Missing API Key**
   - Status: 403 Forbidden
   - Response: API Gateway error message

2. **Invalid JSON**
   - Status: 400 Bad Request
   - Response: `{"error": "Invalid input"}`

3. **Missing Required Fields**
   - Status: 400 Bad Request
   - Response: `{"error": "'docx' key missing"}`

4. **Invalid Base64 Encoding**
   - Status: 400 Bad Request
   - Response: `{"error": "Failed to decode base64 input"}`

5. **Corrupted DOCX File**
   - Status: 500 Internal Server Error
   - Response: `{"error": "Failed to process document"}`

6. **Field Extraction Failure**
   - Status: 500 Internal Server Error
   - Response: `{"error": "Failed to extract fields"}`

7. **Merge Operation Failure**
   - Status: 500 Internal Server Error
   - Response: `{"error": "Failed to perform merge"}`

---

## Field Types and Validation

### Supported Field Types

The system automatically detects and categorizes merge fields:

- **Text Fields**: Standard string values
- **Numeric Fields**: Numbers and calculations
- **Date Fields**: Date and time values
- **Boolean Fields**: True/false values

### Validation Rules

1. **Required Fields**: Must be present in merge data
2. **Data Type Validation**: Values must match expected field types
3. **Duplicate Key Detection**: First occurrence wins, warnings generated
4. **Field Name Matching**: Case-sensitive matching against document fields

---

## Best Practices

### Performance Optimization

1. **Use `/detect` for Field Discovery**: When you only need to know what fields are available
2. **Batch Processing**: Group multiple documents when possible
3. **Caching**: Cache field extraction results for templates used multiple times

### Error Handling

1. **Validate Input**: Always check DOCX file integrity before upload
2. **Handle Validation Errors**: Process validation response for user feedback
3. **Retry Logic**: Implement exponential backoff for rate limit errors

### Security

1. **API Key Management**: Rotate API keys regularly
2. **Input Validation**: Validate all input data before sending requests
3. **HTTPS Only**: Always use HTTPS for API communication

---

## SDK Examples

### JavaScript/Node.js

```javascript
const axios = require('axios');

async function detectDocument(docxBase64) {
  try {
    const response = await axios.post(
      'https://your-api-gateway-url.amazonaws.com/dev/detect',
      {
        docx: docxBase64
      },
      {
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': 'your-api-key'
        }
      }
    );
    
    return response.data;
  } catch (error) {
    console.error('Error detecting document:', error.response?.data || error.message);
    throw error;
  }
}

async function performMerge(docxBase64, mergeData) {
  try {
    const response = await axios.post(
      'https://your-api-gateway-url.amazonaws.com/dev/merge',
      {
        docx: docxBase64,
        data: mergeData
      },
      {
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': 'your-api-key'
        }
      }
    );
    
    return response.data;
  } catch (error) {
    console.error('Error performing merge:', error.response?.data || error.message);
    throw error;
  }
}
```

### Python

```python
import requests
import base64
import json

class FlashMailMergeClient:
    def __init__(self, api_url, api_key):
        self.api_url = api_url
        self.api_key = api_key
        self.headers = {
            'Content-Type': 'application/json',
            'x-api-key': api_key
        }
    
    def detect_document(self, docx_file_path):
        """Extract fields from DOCX document"""
        with open(docx_file_path, 'rb') as file:
            docx_base64 = base64.b64encode(file.read()).decode('utf-8')
        
        payload = {'docx': docx_base64}
        response = requests.post(
            f'{self.api_url}/detect',
            headers=self.headers,
            data=json.dumps(payload)
        )
        
        if response.status_code == 200:
            return response.json()
        else:
            raise Exception(f"API Error: {response.status_code} - {response.text}")
    
    def perform_merge(self, docx_file_path, merge_data):
        """Perform mail merge operation"""
        with open(docx_file_path, 'rb') as file:
            docx_base64 = base64.b64encode(file.read()).decode('utf-8')
        
        payload = {
            'docx': docx_base64,
            'data': merge_data
        }
        
        response = requests.post(
            f'{self.api_url}/merge',
            headers=self.headers,
            data=json.dumps(payload)
        )
        
        if response.status_code == 200:
            return response.json()
        else:
            raise Exception(f"API Error: {response.status_code} - {response.text}")

# Usage example
client = FlashMailMergeClient(
    'https://your-api-gateway-url.amazonaws.com/dev',
    'your-api-key'
)

# Detect document fields
fields = client.detect_document('template.docx')
print("Available fields:", fields['data'].keys())

# Perform merge
merge_result = client.perform_merge('template.docx', {
    'FirstName': 'John',
    'LastName': 'Doe',
    'Email': 'john.doe@example.com'
})

# Save merged document
if 'mergedDocument' in merge_result:
    merged_bytes = base64.b64decode(merge_result['mergedDocument'])
    with open('merged_document.docx', 'wb') as f:
        f.write(merged_bytes)
```

---

## Testing

### Local Testing

When running the service locally using SAM:

```bash
# Start local API
make local

# Test detect endpoint
curl -X POST http://localhost:3000/detect \
  -H "Content-Type: application/json" \
  -d '{"docx": "base64-encoded-docx"}'

# Test merge endpoint
curl -X POST http://localhost:3000/merge \
  -H "Content-Type: application/json" \
  -d '{"docx": "base64-encoded-docx", "data": {"Field1": "value1"}}'
```

### Testing with Sample Data

The repository includes sample DOCX files for testing:

```bash
# Encode sample file for testing
base64 -i tests/data/sample.docx
```

---

## Support

For additional support or questions:

1. Review the main [README.md](README.md) for general usage
2. Check the test files for implementation examples
3. Create an issue in the GitHub repository
4. Review CloudWatch logs for debugging deployed services
