# Flash Mail Merge

A high-performance AWS Lambda service for processing DOCX mail merge operations with field extraction, validation, and document merging capabilities.

## Overview

Flash Mail Merge is a serverless solution that extracts merge fields from DOCX documents, validates merge data, and performs complete mail merge operations. It's designed to be fast, scalable, and cost-effective for processing large volumes of documents with comprehensive logging and error handling.

## Features

- **DOCX Processing**: Extracts and processes Microsoft Word documents with full ZIP archive handling
- **DOCX Validation**: Validates DOCX file signature and structure integrity
- **Merge Field Detection**: Automatically detects merge fields in documents with support for complex field types
- **Data Validation**: Validates merge data against field requirements with detailed error reporting
- **Mail Merge Execution**: Performs complete mail merge operations with field replacement
- **Duplicate Key Detection**: Detects and handles duplicate keys in merge data with first-win logic
- **Comprehensive Logging**: Structured logging with configurable log levels
- **Serverless Architecture**: Runs on AWS Lambda with API Gateway and S3 integration
- **Type Safety**: Full type checking for merge field data with Go's strong typing

## Architecture

The project is structured as follows:

```
flash-mail-merge/
├── main.go               # Main Lambda handler (current entry point)
├── cmd/handler/          # Alternative handler (template)
│   └── main.go          # Basic Lambda handler template
├── internal/
│   ├── docx/            # DOCX file processing
│   │   ├── unzip.go     # ZIP archive handling and extraction
│   │   ├── zip.go       # ZIP archive creation
│   │   └── *_test.go    # Unit tests
│   ├── fields/          # Merge field operations
│   │   ├── extract.go   # Field extraction logic
│   │   ├── models.go    # Data models and validation
│   │   ├── parse.go     # Field parsing utilities
│   │   └── *_test.go    # Unit tests
│   ├── merge/           # Mail merge operations
│   │   ├── merge.go     # Core merge functionality
│   │   └── merge_test.go # Unit tests
│   └── logging/         # Logging utilities
│       └── log.go       # Structured logging
├── tests/               # Unit tests and sample files
├── tests_manual/        # Manual testing files and events
└── deploy/              # SAM/CloudFormation templates
    └── template.yaml    # AWS SAM template
```

## Usage

### Local Development

1. Install Go 1.21 or later (project uses Go 1.24.5)
2. Clone the repository
3. Run tests: `go test ./...`
4. Build: `go build -o main main.go`
5. Test locally: `make local` (starts SAM local API)

### Deployment

The service is deployed using AWS SAM (Serverless Application Model) with a Makefile workflow:

1. **Configure AWS credentials**
   ```bash
   aws configure
   ```

2. **Create S3 deployment bucket** (if needed)
   ```bash
   make create-bucket
   ```

3. **Build and deploy**
   ```bash
   make build && make package && make deploy
   ```
   
   Or use the simplified workflow:
   ```bash
   make deploy-all
   ```

4. **Available Make targets**:
   - `make build` - Build the Go binary for Lambda (creates `bootstrap`)
   - `make test` - Run tests
   - `make package` - Package the SAM application
   - `make deploy` - Deploy the SAM application
   - `make deploy-all` - Build, package, and deploy in one command
   - `make validate` - Validate the SAM template
   - `make local` - Start local API for testing
   - `make create-bucket` - Create S3 bucket for deployment
   - `make clean` - Clean build artifacts
   - `make delete` - Delete the CloudFormation stack

5. **Configuration**:
   - Runtime: `go1.x`
   - Handler: `bootstrap`
   - Memory: 256 MB
   - Timeout: 30 seconds
   - API Gateway: POST `/merge` and POST `/detect`
   - Binary media types enabled for DOCX files
   - S3 buckets for document storage and results
   - API Key authentication with usage plans

### API

The service provides two main endpoints for different use cases:

## 1. `/merge` Endpoint - Field Detection and Mail Merge

Performs field extraction, data validation, and optional mail merge operations.

### Request Schema
```json
{
  "docx": "base64-encoded DOCX content",     // Required
  "data": {                                  // Optional
    "FieldName1": "value1",
    "FieldName2": "value2"
  }
}
```

### Response Schema
```json
{
  "validation": {
    "valid": true,
    "errors": [],
    "warnings": []
  },
  "mergedDocument": "base64-encoded-result-docx",  // Only when data provided
  "skippedFields": []                             // Only when data provided
}
```

### Example Request
```bash
curl -X POST https://your-api-gateway-url/merge \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
  -d '{
    "docx": "UEsDBAoAAAAAAJZQV1cAAAAAAAAAAAAAAAAJAAAAZG9jUHJvcHMv...",
    "data": {
      "FirstName": "John",
      "LastName": "Doe",
      "Email": "john.doe@example.com"
    }
  }'
```

### Error Codes
- **400 Bad Request**: Invalid JSON, missing 'docx' field, invalid base64, or validation errors
- **500 Internal Server Error**: Document processing failure or merge operation error

## 2. `/detect` Endpoint - Field Extraction Only

Extracts merge fields from DOCX documents without performing validation or merge operations.

### Request Schema
```json
{
  "docx": "base64-encoded DOCX content"  // Required
}
```

### Response Schema
```json
{
  "data": {
    "FieldName1": "",
    "FieldName2": "",
    "FieldName3": ""
  }
}
```

### Example Request
```bash
curl -X POST https://your-api-gateway-url/detect \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
  -d '{
    "docx": "UEsDBAoAAAAAAJZQV1cAAAAAAAAAAAAAAAAJAAAAZG9jUHJvcHMv..."
  }'
```

### Example Response
```json
{
  "data": {
    "FirstName": "",
    "LastName": "",
    "Email": "",
    "CompanyName": "",
    "Address": ""
  }
}
```

### Error Codes
- **400 Bad Request**: Invalid JSON, missing 'docx' field, or invalid base64
- **500 Internal Server Error**: Document processing failure or field extraction error

## Common Error Response Format

All endpoints return errors in the following format:

```json
{
  "error": "Error message description"
}
```

## Authentication

Both endpoints require API Key authentication:
- Include `x-api-key` header with your API key
- API keys are managed through AWS API Gateway
- Rate limiting: 100 requests/second, 200 burst
- Daily quota: 10,000 requests

## Features

- **Field Extraction**: Automatically detects merge fields in DOCX documents
- **Data Validation**: Validates provided merge data against field requirements (merge endpoint)
- **Duplicate Detection**: Handles duplicate keys with first-win logic (merge endpoint)
- **Mail Merge**: Performs complete merge operation when data is provided (merge endpoint)
- **Error Handling**: Comprehensive error reporting and validation feedback

## Examples

### Processing DOCX Files

```go
import (
    "com/lifenture/flash-mail-merge/internal/docx"
)

// Extract DOCX file from bytes with validation
docxFile, err := docx.UnzipDocx(docxBytes)
if err != nil {
    // Handle error (invalid DOCX, missing signature, etc.)
}

// Access document XML
documentXML := docxFile.DocumentXML
fmt.Printf("Document XML: %s\n", documentXML)
```

### Extracting Fields

```go
import (
    "com/lifenture/flash-mail-merge/internal/docx"
    "com/lifenture/flash-mail-merge/internal/fields"
)

// Load DOCX file
doc, err := docx.UnzipDocx(docxData)
if err != nil {
    // Handle error
}

// Extract merge fields
fieldSet, err := fields.ExtractFields(doc)
if err != nil {
    // Handle error
}

// Use extracted fields
for _, field := range fieldSet.Fields {
    fmt.Printf("Field: %s, Type: %s, Required: %t\n", field.Name, field.Type, field.Required)
}
```

### Validating Merge Data

```go
// Prepare merge data
mergeData := fields.MergeData{
    "FirstName": "John",
    "LastName":  "Doe",
    "Email":     "john.doe@example.com",
}

// Validate against field set
result := fieldSet.Validate(mergeData)
if !result.Valid {
    for _, error := range result.Errors {
        fmt.Printf("Validation error: %s\n", error)
    }
}

// Check for warnings (e.g., duplicate keys)
for _, warning := range result.Warnings {
    fmt.Printf("Warning: %s\n", warning)
}
```

### Performing Mail Merge

```go
import (
    "com/lifenture/flash-mail-merge/internal/merge"
)

// Perform mail merge operation
mergedBytes, skippedFields, err := merge.PerformMerge(docxFile, mergeData)
if err != nil {
    // Handle error
}

// Save or process the merged document
fmt.Printf("Merge completed. Skipped fields: %v\n", skippedFields)
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with make
make test
```

### Building

```bash
# Build for local development
go build -o main main.go

# Build for Lambda deployment
make build
```

### Linting

```bash
# Run standard Go tools
go fmt ./...
go vet ./...

# If you have golint installed
golint ./...
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## API Documentation

For detailed API reference including request/response schemas, error codes, and SDK examples, see [API.md](API.md).

## Support

For issues and questions, please create an issue in the GitHub repository.
