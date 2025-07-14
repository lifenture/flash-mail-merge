package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"com/lifenture/flash-mail-merge/internal/docx"
	"com/lifenture/flash-mail-merge/internal/fields"
)

// MergeRequest represents the request payload for merge operations
type MergeRequest struct {
	Docx string          `json:"docx"`            // base64 DOCX (required)
	Data json.RawMessage `json:"data,omitempty"`  // raw map for merge values (optional)
}

// parseMergeData parses raw JSON data into MergeData with duplicate-key "first-win" logic.
// If a key appears multiple times in the JSON object, only the first occurrence is kept.
func parseMergeData(raw json.RawMessage) (fields.MergeData, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))

	// Read the opening brace
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read opening token: %w", err)
	}

	// Expect opening brace
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected opening brace, got %T: %v", token, token)
	}

	result := make(fields.MergeData)

	// Process key-value pairs
	for decoder.More() {
		// Read the key
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to read key token: %w", err)
		}

		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %T: %v", token, token)
		}

		// Check if key already exists (first-win logic)
		if _, exists := result[key]; exists {
			// Skip the value for this duplicate key
			var dummy interface{}
			if err := decoder.Decode(&dummy); err != nil {
				return nil, fmt.Errorf("failed to skip duplicate key value: %w", err)
			}
		} else {
			// Decode the value for this new key
			var value interface{}
			if err := decoder.Decode(&value); err != nil {
				return nil, fmt.Errorf("failed to decode value for key %s: %w", key, err)
			}
			result[key] = value
		}
	}

	// Read the closing brace
	token, err = decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read closing token: %w", err)
	}

	// Expect closing brace
	if delim, ok := token.(json.Delim); !ok || delim != '}' {
		return nil, fmt.Errorf("expected closing brace, got %T: %v", token, token)
	}

	return result, nil
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Content-Type": "application/json"}

	// Unmarshal the body into MergeRequest
	var req MergeRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		log.Printf("failed to unmarshal request body: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       `{"error": "Invalid input"}`,
		}, nil
	}

	// Check if docx field is present
	if req.Docx == "" {
		log.Println("'docx' field is empty")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       `{"error": "'docx' key missing"}`,
		}, nil
	}

	// Decode the DOCX exactly as today
	docxBytes, err := base64.StdEncoding.DecodeString(req.Docx)
	if err != nil {
		log.Printf("failed to decode base64 string: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       `{"error": "Failed to decode base64 input"}`,
		}, nil
	}


	// Create a DocxFile from the bytes to use ExtractFields
	docxFile, err := docx.UnzipDocx(docxBytes)
	if err != nil {
		log.Printf("failed to create DOCX file: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"error": "Failed to process document"}`,
		}, nil
	}

	// Extract fields to get MergeFieldSet
	fieldSet, err := fields.ExtractFields(docxFile)
	if err != nil {
		log.Printf("failed to extract fields: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"error": "Failed to extract fields"}`,
		}, nil
	}

	// Prepare response structure
	response := map[string]interface{}{
		"fields": fieldSet.GetFieldNames(),
		"count":  len(fieldSet.Fields),
	}

	// If req.Data is present, parse merge data and validate
	if req.Data != nil {
		duplicates := fields.DetectDuplicates(req.Data)
		if len(duplicates) > 0 {
			log.Printf("Duplicate keys detected: %v", duplicates)
		}

		mergeData, err := parseMergeData(req.Data)
		if err != nil {
			log.Printf("failed to parse merge data: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Headers:    headers,
				Body:       `{"error": "Failed to parse merge data"}`,
			}, nil
		}

		// Run validation
		validationResult := fieldSet.Validate(mergeData)
		
		// Add duplicate key warnings to validation result
		if len(duplicates) > 0 {
			for _, key := range duplicates {
				validationResult.Warnings = append(validationResult.Warnings, 
					fmt.Sprintf("Duplicate key '%s' detected in JSON data (first occurrence kept)", key))
			}
		}

		// Include validation output and cleaned-up merge data in response
		response["validation"] = validationResult
		response["merge_data"] = mergeData

		// Return 400 if validation failed
		if !validationResult.Valid {
			responseBody, err := json.Marshal(response)
			if err != nil {
				log.Printf("failed to marshal response: %v", err)
				return events.APIGatewayProxyResponse{
					StatusCode: http.StatusInternalServerError,
					Body:       `{"error": "Failed to create response"}`,
				}, nil
			}

			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Headers:    headers,
				Body:       string(responseBody),
			}, nil
		}
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       `{"error": "Failed to create response"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       string(responseBody),
	}, nil
}

func main() {
	lambda.Start(handler)
}

