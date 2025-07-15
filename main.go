package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"com/lifenture/flash-mail-merge/internal/docx"
	"com/lifenture/flash-mail-merge/internal/fields"
	"com/lifenture/flash-mail-merge/internal/logging"
	"com/lifenture/flash-mail-merge/internal/merge"
)

// Common headers for all responses
func getCommonHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/json"}
}

// Helper function to create error responses
func createErrorResponse(statusCode int, errorMessage string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    getCommonHeaders(),
		Body:       fmt.Sprintf(`{"error": "%s"}`, errorMessage),
	}
}

// Helper function to create successful response
func createSuccessResponse(data interface{}) (events.APIGatewayProxyResponse, error) {
	responseBody, err := json.Marshal(data)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to marshal response: %w", err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    getCommonHeaders(),
		Body:       string(responseBody),
	}, nil
}

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
	// Unmarshal the body into MergeRequest
	var req MergeRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		logging.Error("failed to unmarshal request body: %v", err)
		return createErrorResponse(http.StatusBadRequest, "Invalid input"), nil
	}

	// Check if docx field is present
	if req.Docx == "" {
		logging.Error("'docx' field is empty")
		return createErrorResponse(http.StatusBadRequest, "'docx' key missing"), nil
	}

	// Decode the DOCX exactly as today
	docxBytes, err := base64.StdEncoding.DecodeString(req.Docx)
	if err != nil {
		logging.Error("failed to decode base64 string: %v", err)
		return createErrorResponse(http.StatusBadRequest, "Failed to decode base64 input"), nil
	}

	// Create a DocxFile from the bytes to use ExtractFields
	docxFile, err := docx.UnzipDocx(docxBytes)
	if err != nil {
		logging.Error("failed to create DOCX file: %v", err)
		return createErrorResponse(http.StatusInternalServerError, "Failed to process document"), nil
	}

	// Extract fields to get MergeFieldSet
	fieldSet, err := fields.ExtractFields(docxFile)
	if err != nil {
		logging.Error("failed to extract fields: %v", err)
		return createErrorResponse(http.StatusInternalServerError, "Failed to extract fields"), nil
	}

	// Prepare response structure
	response := map[string]interface{}{}

	// If req.Data is present, parse merge data and validate
	if req.Data != nil {
		duplicates := fields.DetectDuplicates(req.Data)
		if len(duplicates) > 0 {
			logging.Warn("Duplicate keys detected: %v", duplicates)
		}

		mergeData, err := parseMergeData(req.Data)
		if err != nil {
			logging.Error("failed to parse merge data: %v", err)
			return createErrorResponse(http.StatusBadRequest, "Failed to parse merge data"), nil
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

		// Include validation output in response
		response["validation"] = validationResult

		// Only execute merge if validation passed
		if !validationResult.Valid {
			// Return validation error with response including validation details
			responseBody, err := json.Marshal(response)
			if err != nil {
				logging.Error("failed to marshal response: %v", err)
				return createErrorResponse(http.StatusInternalServerError, "Failed to create response"), nil
			}

			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Headers:    getCommonHeaders(),
				Body:       string(responseBody),
			}, nil
		}

		// After successful validation, perform merge
		mergedBytes, skipped, err := merge.PerformMerge(docxFile, mergeData)
		if err != nil {
			logging.Error("failed to perform merge: %v", err)
			return createErrorResponse(http.StatusInternalServerError, "Failed to perform merge"), nil
		}

		// Base64-encode the merged document
		mergedDocumentB64 := base64.StdEncoding.EncodeToString(mergedBytes)

		// Add merged document and skipped fields to response
		response["mergedDocument"] = mergedDocumentB64
		response["skippedFields"] = skipped
	}

	// Use helper function to create successful response
	successResponse, err := createSuccessResponse(response)
	if err != nil {
		logging.Error("failed to create success response: %v", err)
		return createErrorResponse(http.StatusInternalServerError, "Failed to create response"), nil
	}

	return successResponse, nil
}

func main() {
	lambda.Start(handler)
}

