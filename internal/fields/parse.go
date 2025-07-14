package fields

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// DetectDuplicates parses raw JSON data and returns a slice of duplicate keys
// (second and later occurrences). This is useful for diagnostics when dealing
// with JSON objects that may have duplicate keys.
func DetectDuplicates(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	
	decoder := json.NewDecoder(bytes.NewReader(raw))
	
	// Read the opening brace
	token, err := decoder.Token()
	if err != nil {
		return nil
	}
	
	// Expect opening brace
	if delim, ok := token.(json.Delim); !ok || delim != '{' {
		return nil
	}
	
	seen := make(map[string]bool)
	var duplicates []string
	
	// Process key-value pairs
	for decoder.More() {
		// Read the key
		token, err := decoder.Token()
		if err != nil {
			return duplicates
		}
		
		key, ok := token.(string)
		if !ok {
			return duplicates
		}
		
	// Check if key already exists (using normalized comparison)
		normalizedKey := normalize(key)
		if seen[normalizedKey] {
			// This is a duplicate occurrence
			duplicates = append(duplicates, key)
		} else {
			// First occurrence
			seen[normalizedKey] = true
		}
		
		// Skip the value for this key
		var dummy interface{}
		if err := decoder.Decode(&dummy); err != nil {
			return duplicates
		}
	}
	
	return duplicates
}

// parseMergeData parses raw JSON data into MergeData with duplicate-key "first-win" logic.
// If a key appears multiple times in the JSON object, only the first occurrence is kept.
func parseMergeData(raw json.RawMessage) (MergeData, error) {
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
	
	result := make(MergeData)
	
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
		
	// Check if key already exists (first-win logic, using normalized comparison)
		normalizedKey := normalize(key)
		existingKey := ""
		for k := range result {
			if normalize(k) == normalizedKey {
				existingKey = k
				break
			}
		}
		if existingKey != "" {
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
