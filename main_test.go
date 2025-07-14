package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandler(t *testing.T) {
	// Get the path to the sample DOCX file
	samplePath := filepath.Join("tests", "data", "sample.docx")
	
	// Check if the sample file exists
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("Sample DOCX file not found, skipping test")
	}
	
	// Read the sample DOCX file
	file, err := os.Open(samplePath)
	if err != nil {
		t.Fatalf("Failed to open sample DOCX file: %v", err)
	}
	defer file.Close()
	
	docxBytes, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read sample DOCX file: %v", err)
	}
	
	// Encode the DOCX file as base64
	encodedDocx := base64.StdEncoding.EncodeToString(docxBytes)
	
	// Create the request body
	requestBody := map[string]string{
		"docx": encodedDocx,
	}
	
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}
	
	// Create the Lambda event
	request := events.APIGatewayProxyRequest{
		Body: string(bodyJSON),
	}
	
	// Call the handler
	ctx := context.Background()
	response, err := handler(ctx, request)
	
	// Check for errors
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	
	// Check status code
	if response.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
		t.Logf("Response body: %s", response.Body)
	}
	
	// Check content type
	expectedContentType := "application/json"
	if response.Headers["Content-Type"] != expectedContentType {
		t.Errorf("Expected Content-Type %s, got %s", expectedContentType, response.Headers["Content-Type"])
	}
	
	// Parse response body
	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(response.Body), &responseData); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	
	// With no data field, the response should be empty
	if len(responseData) != 0 {
		t.Errorf("Expected empty response when no data field provided, got %v", responseData)
	}
}

func TestHandlerErrorCases(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid JSON",
			requestBody:    "{invalid json",
			expectedStatus: 400,
			expectedError:  "Invalid input",
		},
		{
			name:           "missing docx field",
			requestBody:    `{"other": "value"}`,
			expectedStatus: 400,
			expectedError:  "'docx' key missing",
		},
		{
			name:           "invalid base64",
			requestBody:    `{"docx": "invalid_base64!"}`,
			expectedStatus: 400,
			expectedError:  "Failed to decode base64 input",
		},
		{
			name:           "empty body",
			requestBody:    "",
			expectedStatus: 400,
			expectedError:  "Invalid input",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := events.APIGatewayProxyRequest{
				Body: tt.requestBody,
			}
			
			response, err := handler(ctx, request)
			
			// Handler should not return an error (errors are handled internally)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			// Check status code
			if response.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, response.StatusCode)
			}
			
			// Check that error message is present in response
			if !strings.Contains(response.Body, tt.expectedError) {
				t.Errorf("Expected error message '%s' in response body: %s", tt.expectedError, response.Body)
			}
		})
	}
}

func TestHandlerWithValidDocx(t *testing.T) {
	// Create a minimal ZIP structure with the test XML
	// This would need to be a valid ZIP file to pass the tests
	// For now, we'll skip this test if we can't create a valid DOCX
	t.Skip("Skipping test - requires creating valid DOCX programmatically")
}

// TestHandlerWithDuplicateKeys tests the handler with duplicate keys in the data field
func TestHandlerWithDuplicateKeys(t *testing.T) {
	// Get the path to the sample DOCX file
	samplePath := filepath.Join("tests", "data", "sample.docx")
	
	// Check if the sample file exists
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("Sample DOCX file not found, skipping test")
	}
	
	// Read the sample DOCX file
	file, err := os.Open(samplePath)
	if err != nil {
		t.Fatalf("Failed to open sample DOCX file: %v", err)
	}
	defer file.Close()
	
	docxBytes, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read sample DOCX file: %v", err)
	}
	
	// Encode the DOCX file as base64
	encodedDocx := base64.StdEncoding.EncodeToString(docxBytes)
	
	// Create request body with duplicate keys in the data field
	// This JSON intentionally contains duplicate keys to test first-win logic
	requestBodyJSON := `{
		"docx": "` + encodedDocx + `",
		"data": {"Contact_FirstName": "John", "Contact_FullName": "John Doe", "User_Email": "john@example.com", "Contact_FirstName": "Jane", "User_Company": "ACME Corp", "User_Email": "jane@example.com", "User_Phone": "555-0123"}
	}`
	
	// Create the Lambda event
	request := events.APIGatewayProxyRequest{
		Body: requestBodyJSON,
	}
	
	// Call the handler
	ctx := context.Background()
	response, err := handler(ctx, request)
	
	// Check for errors
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	
	// Check status code - should be 200 since validation should pass
	if response.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
		t.Logf("Response body: %s", response.Body)
	}
	
	// Parse response body
	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(response.Body), &responseData); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	
	// Check that mergedDocument is present
	if _, ok := responseData["mergedDocument"]; !ok {
		t.Fatalf("Response does not contain 'mergedDocument' field")
	}
	
	// Check that skippedFields is present
	if _, ok := responseData["skippedFields"]; !ok {
		t.Fatalf("Response does not contain 'skippedFields' field")
	}
	
	// Check that validation warnings are present about duplicate keys
	validationRaw, ok := responseData["validation"]
	if !ok {
		t.Fatalf("Response does not contain 'validation' field")
	}
	
	validation, ok := validationRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("validation is not a map")
	}
	
	warningsRaw, ok := validation["warnings"]
	if !ok {
		t.Fatalf("validation does not contain 'warnings' field")
	}
	
	warnings, ok := warningsRaw.([]interface{})
	if !ok {
		t.Fatalf("warnings is not an array")
	}
	
	// Should have warnings about duplicate keys
	if len(warnings) == 0 {
		t.Errorf("Expected duplicate key warnings but found none")
	}
	
	// Check that warnings contain mentions of duplicate keys
	hasFirstNameWarning := false
	hasEmailWarning := false
	for _, warning := range warnings {
		warningStr, ok := warning.(string)
		if !ok {
			continue
		}
		if strings.Contains(warningStr, "Contact_FirstName") && strings.Contains(strings.ToLower(warningStr), "duplicate") {
			hasFirstNameWarning = true
		}
		if strings.Contains(warningStr, "User_Email") && strings.Contains(strings.ToLower(warningStr), "duplicate") {
			hasEmailWarning = true
		}
	}
	
	if !hasFirstNameWarning {
		t.Errorf("Expected warning about duplicate Contact_FirstName key")
	}
	if !hasEmailWarning {
		t.Errorf("Expected warning about duplicate User_Email key")
	}
	
	// Check that overall validation is still valid despite warnings
	validRaw, ok := validation["valid"]
	if !ok {
		t.Fatalf("validation does not contain 'valid' field")
	}
	
	valid, ok := validRaw.(bool)
	if !ok {
		t.Fatalf("valid is not a boolean")
	}
	
	if !valid {
		t.Errorf("Expected validation to be valid despite duplicate key warnings")
	}
}

// TestHandlerWithDuplicateKeysRawJSON tests the handler with raw JSON string containing duplicates
func TestHandlerWithDuplicateKeysRawJSON(t *testing.T) {
	// Get the path to the sample DOCX file
	samplePath := filepath.Join("tests", "data", "sample.docx")
	
	// Check if the sample file exists
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("Sample DOCX file not found, skipping test")
	}
	
	// Read the sample DOCX file
	file, err := os.Open(samplePath)
	if err != nil {
		t.Fatalf("Failed to open sample DOCX file: %v", err)
	}
	defer file.Close()
	
	docxBytes, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read sample DOCX file: %v", err)
	}
	
	// Encode the DOCX file as base64
	encodedDocx := base64.StdEncoding.EncodeToString(docxBytes)
	
	// Create request body with JSON string that has intentional duplicate keys
	// This simulates a more realistic scenario where the JSON might come from external sources
	requestBodyJSON := `{
		"docx": "` + encodedDocx + `",
		"data": {"Contact_FirstName": "Alice", "Contact_FullName": "Alice Smith", "User_Email": "alice@test.com", "Contact_FirstName": "Bob", "User_Company": "TestCorp", "User_Email": "bob@test.com", "User_Phone": "555-9999"}
	}`
	
	// Create the Lambda event
	request := events.APIGatewayProxyRequest{
		Body: requestBodyJSON,
	}
	
	// Call the handler
	ctx := context.Background()
	response, err := handler(ctx, request)
	
	// Check for errors
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	
	// Check status code - should be 200 since validation should pass
	if response.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", response.StatusCode)
		t.Logf("Response body: %s", response.Body)
	}
	
	// Parse response body
	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(response.Body), &responseData); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}
	
	// Check that mergedDocument is present
	if _, ok := responseData["mergedDocument"]; !ok {
		t.Fatalf("Response does not contain 'mergedDocument' field")
	}
	
	// Check that skippedFields is present
	if _, ok := responseData["skippedFields"]; !ok {
		t.Fatalf("Response does not contain 'skippedFields' field")
	}
}
