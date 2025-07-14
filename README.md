# Flash Mail Merge

A high-performance AWS Lambda service for processing DOCX mail merge operations.

## Overview

Flash Mail Merge is a serverless solution that extracts merge fields from DOCX documents and performs mail merge operations. It's designed to be fast, scalable, and cost-effective for processing large volumes of documents.

## Features

- **DOCX Processing**: Extracts and processes Microsoft Word documents
- **DOCX Validation**: Validates DOCX file signature and structure
- **Merge Field Detection**: Automatically detects merge fields in documents
- **Data Validation**: Validates merge data against field requirements
- **Serverless Architecture**: Runs on AWS Lambda for scalability
- **Type Safety**: Full type checking for merge field data

## Architecture

The project is structured as follows:

```
flash-mail-merge/
├── cmd/handler/           # Lambda entrypoint
│   └── main.go           # Main Lambda handler
├── internal/
│   ├── docx/             # DOCX file processing
│   │   └── unzip.go      # ZIP archive handling
│   └── fields/           # Merge field operations
│       ├── extract.go    # Field extraction logic
│       └── models.go     # Data models
├── tests/                # Unit tests and sample files
└── deploy/               # SAM/CloudFormation templates
```

## Field Types

The system supports the following merge field types:

- **String**: Text fields
- **Number**: Numeric fields (integers and floats)
- **Date**: Date fields with format validation
- **Boolean**: True/false fields
- **Image**: Image insertion fields
- **Table**: Table merge fields

## Usage

### Local Development

1. Install Go 1.19 or later
2. Clone the repository
3. Run tests: `go test ./...`
4. Build: `go build ./cmd/handler`

### Deployment

The service is deployed using AWS SAM (Serverless Application Model) with a Makefile workflow:

1. **Configure AWS credentials**
   ```bash
   aws configure
   ```

2. **Build and deploy**
   ```bash
   make build && sam package && sam deploy
   ```
   
   Or use the simplified workflow:
   ```bash
   make deploy-all
   ```

3. **Available Make targets**:
   - `make build` - Build the Go binary for Lambda (creates `bootstrap`)
   - `make test` - Run tests
   - `make package` - Package the SAM application
   - `make deploy` - Deploy the SAM application
   - `make deploy-all` - Build, package, and deploy in one command
   - `make validate` - Validate the SAM template
   - `make local` - Start local API for testing
   - `make clean` - Clean build artifacts
   - `make delete` - Delete the CloudFormation stack

4. **Configuration**:
   - Runtime: `go1.x`
   - Handler: `bootstrap`
   - Memory: 256 MB
   - API Gateway: POST `/detect`
   - Binary media types enabled for DOCX files

### API

The Lambda function accepts events with the following structure:

```json
{
  "docx": "base64-encoded DOCX content"
}
```

## Examples

### Reading Document XML

```go
import (
    "com/lifenture/flash-mail-merge/internal/docx"
)

// Direct approach: Get document XML from DOCX bytes with validation
documentXML, err := docx.ReadDocumentXML(docxBytes)
if err != nil {
    // Handle error (invalid DOCX, missing signature, etc.)
}

// documentXML now contains the word/document.xml content as a string
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
    fmt.Printf("Field: %s, Type: %s\n", field.Name, field.Type)
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
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build ./cmd/handler
```

### Linting

```bash
golint ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions, please create an issue in the GitHub repository.
