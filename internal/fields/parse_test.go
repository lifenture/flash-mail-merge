package fields

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDetectDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    json.RawMessage
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    json.RawMessage(`{"name": "John", "age": 30, "city": "New York"}`),
			expected: nil,
		},
		{
			name:     "single duplicate",
			input:    json.RawMessage(`{"name": "John", "age": 30, "name": "Jane"}`),
			expected: []string{"name"},
		},
		{
			name:     "multiple duplicates same key",
			input:    json.RawMessage(`{"name": "John", "age": 30, "name": "Jane", "name": "Bob"}`),
			expected: []string{"name", "name"},
		},
		{
			name:     "multiple duplicates different keys",
			input:    json.RawMessage(`{"name": "John", "age": 30, "name": "Jane", "age": 25}`),
			expected: []string{"name", "age"},
		},
		{
			name:     "empty object",
			input:    json.RawMessage(`{}`),
			expected: nil,
		},
		{
			name:     "empty input",
			input:    json.RawMessage(``),
			expected: nil,
		},
		{
			name:     "invalid JSON",
			input:    json.RawMessage(`{"name": "John", "age": 30`),
			expected: nil,
		},
		{
			name:     "not an object",
			input:    json.RawMessage(`["name", "age"]`),
			expected: nil,
		},
		{
			name:     "complex values with duplicates",
			input:    json.RawMessage(`{"user": {"name": "John"}, "age": 30, "user": {"name": "Jane"}, "settings": {"theme": "dark"}}`),
			expected: []string{"user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectDuplicates(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DetectDuplicates() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectDuplicatesIntegration(t *testing.T) {
	// Test with real JSON data that could come from a request
	jsonWithDuplicates := json.RawMessage(`{
		"firstName": "John",
		"lastName": "Doe",
		"email": "john@example.com",
		"firstName": "Jane",
		"company": "ACME Corp",
		"email": "jane@example.com"
	}`)

	duplicates := DetectDuplicates(jsonWithDuplicates)
	expected := []string{"firstName", "email"}

	if !reflect.DeepEqual(duplicates, expected) {
		t.Errorf("DetectDuplicates() = %v, want %v", duplicates, expected)
	}

	// Test that parseMergeData still works correctly with first-win logic
	mergeData, err := parseMergeData(jsonWithDuplicates)
	if err != nil {
		t.Fatalf("parseMergeData() error = %v", err)
	}

	// Should keep the first occurrence of each duplicate key
	if mergeData["firstName"] != "John" {
		t.Errorf("Expected firstName='John', got '%v'", mergeData["firstName"])
	}
	if mergeData["email"] != "john@example.com" {
		t.Errorf("Expected email='john@example.com', got '%v'", mergeData["email"])
	}
	if mergeData["lastName"] != "Doe" {
		t.Errorf("Expected lastName='Doe', got '%v'", mergeData["lastName"])
	}
	if mergeData["company"] != "ACME Corp" {
		t.Errorf("Expected company='ACME Corp', got '%v'", mergeData["company"])
	}
}
