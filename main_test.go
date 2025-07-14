package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
	
	// Check that fields are present
	fields, ok := responseData["fields"].([]interface{})
	if !ok {
		t.Fatalf("Response does not contain 'fields' array")
	}
	
	// Convert to string slice
	fieldStrings := make([]string, len(fields))
	for i, field := range fields {
		fieldStrings[i] = field.(string)
	}
	
	// Check expected fields
	expectedFields := []string{"FirstName", "LastName", "Email", "Company", "Phone"}
	
	// Sort both slices for comparison
	sort.Strings(fieldStrings)
	sort.Strings(expectedFields)
	
	if !reflect.DeepEqual(fieldStrings, expectedFields) {
		t.Errorf("Expected fields %v, got %v", expectedFields, fieldStrings)
	}
	
	// Check count
	count, ok := responseData["count"].(float64)
	if !ok {
		t.Fatalf("Response does not contain 'count' field")
	}
	
	if int(count) != 5 {
		t.Errorf("Expected count 5, got %d", int(count))
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
		"data": {"FirstName": "John", "LastName": "Doe", "Email": "john@example.com", "FirstName": "Jane", "Company": "ACME Corp", "Email": "jane@example.com", "Phone": "555-0123"}
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
	
	// Check that merge_data is present and contains first-win values
	mergeDataRaw, ok := responseData["merge_data"]
	if !ok {
		t.Fatalf("Response does not contain 'merge_data' field")
	}
	
	mergeData, ok := mergeDataRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("merge_data is not a map")
	}
	
	// Check first-win logic - first occurrence should be kept
	if mergeData["FirstName"] != "John" {
		t.Errorf("Expected FirstName='John' (first occurrence), got '%v'", mergeData["FirstName"])
	}
	if mergeData["Email"] != "john@example.com" {
		t.Errorf("Expected Email='john@example.com' (first occurrence), got '%v'", mergeData["Email"])
	}
	if mergeData["LastName"] != "Doe" {
		t.Errorf("Expected LastName='Doe', got '%v'", mergeData["LastName"])
	}
	if mergeData["Company"] != "ACME Corp" {
		t.Errorf("Expected Company='ACME Corp', got '%v'", mergeData["Company"])
	}
	if mergeData["Phone"] != "555-0123" {
		t.Errorf("Expected Phone='555-0123', got '%v'", mergeData["Phone"])
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
		if strings.Contains(warningStr, "FirstName") && strings.Contains(strings.ToLower(warningStr), "duplicate") {
			hasFirstNameWarning = true
		}
		if strings.Contains(warningStr, "Email") && strings.Contains(strings.ToLower(warningStr), "duplicate") {
			hasEmailWarning = true
		}
	}
	
	if !hasFirstNameWarning {
		t.Errorf("Expected warning about duplicate FirstName key")
	}
	if !hasEmailWarning {
		t.Errorf("Expected warning about duplicate Email key")
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
		"data": {"FirstName": "Alice", "LastName": "Smith", "Email": "alice@test.com", "FirstName": "Bob", "Company": "TestCorp", "Email": "bob@test.com", "Phone": "555-9999"}
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
	
	// Check that merge_data contains first-win values
	mergeDataRaw, ok := responseData["merge_data"]
	if !ok {
		t.Fatalf("Response does not contain 'merge_data' field")
	}
	
	mergeData, ok := mergeDataRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("merge_data is not a map")
	}
	
	// Verify first-win logic
	if mergeData["FirstName"] != "Alice" {
		t.Errorf("Expected FirstName='Alice' (first occurrence), got '%v'", mergeData["FirstName"])
	}
	if mergeData["Email"] != "alice@test.com" {
		t.Errorf("Expected Email='alice@test.com' (first occurrence), got '%v'", mergeData["Email"])
	}
	if mergeData["LastName"] != "Smith" {
		t.Errorf("Expected LastName='Smith', got '%v'", mergeData["LastName"])
	}
	if mergeData["Company"] != "TestCorp" {
		t.Errorf("Expected Company='TestCorp', got '%v'", mergeData["Company"])
	}
	if mergeData["Phone"] != "555-9999" {
		t.Errorf("Expected Phone='555-9999', got '%v'", mergeData["Phone"])
	}
}
