package fields

import (
	"encoding/json"
	"reflect"
	"testing"
)

// compareMergeData compares two MergeData maps for equality
func compareMergeData(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	
	for key, valueA := range a {
		valueB, exists := b[key]
		if !exists {
			return false
		}
		
		if !reflect.DeepEqual(valueA, valueB) {
			return false
		}
	}
	
	return true
}

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

// TestParseMergeDataFirstWin tests the first-win duplicate key logic specifically
func TestParseMergeDataFirstWin(t *testing.T) {
	tests := []struct {
		name     string
		input    json.RawMessage
		expected map[string]interface{}
		expectError bool
	}{
		{
			name:  "single duplicate key",
			input: json.RawMessage(`{"name": "first", "age": 25, "name": "second"}`),
			expected: map[string]interface{}{
				"name": "first",
				"age":  float64(25),
			},
			expectError: false,
		},
		{
			name:  "multiple duplicate keys",
			input: json.RawMessage(`{"name": "first", "age": 25, "name": "second", "age": 30, "city": "NYC"}`),
			expected: map[string]interface{}{
				"name": "first",
				"age":  float64(25),
				"city": "NYC",
			},
			expectError: false,
		},
		{
			name:  "triple duplicate key",
			input: json.RawMessage(`{"name": "first", "name": "second", "name": "third"}`),
			expected: map[string]interface{}{
				"name": "first",
			},
			expectError: false,
		},
		{
			name:  "no duplicates",
			input: json.RawMessage(`{"name": "John", "age": 30, "city": "NYC"}`),
			expected: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
				"city": "NYC",
			},
			expectError: false,
		},
		{
			name:  "complex values with duplicates",
			input: json.RawMessage(`{"user": {"name": "John"}, "settings": ["dark"], "user": {"name": "Jane"}, "settings": ["light"]}`),
			expected: map[string]interface{}{
				"user":     map[string]interface{}{"name": "John"},
				"settings": []interface{}{"dark"},
			},
			expectError: false,
		},
		{
			name:  "different data types for same key",
			input: json.RawMessage(`{"value": "string", "count": 42, "value": 123, "value": true}`),
			expected: map[string]interface{}{
				"value": "string",
				"count": float64(42),
			},
			expectError: false,
		},
		{
			name:        "invalid JSON",
			input:       json.RawMessage(`{"name": "John", "age": 30`),
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty JSON",
			input:       json.RawMessage(`{}`),
			expected:    map[string]interface{}{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMergeData(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			if !compareMergeData(result, tt.expected) {
				t.Errorf("parseMergeData() = %v, want %v", result, tt.expected)
			}
		})
	}
}
