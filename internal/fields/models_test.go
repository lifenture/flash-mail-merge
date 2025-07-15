package fields

import (
	"testing"
	"time"
)

func TestMergeFieldSet_Validate_NoWarningsForUnknownFields(t *testing.T) {
	// Build a dummy MergeFieldSet with one known field
	fieldSet := MergeFieldSet{
		Fields: []MergeField{
			{
				Name:     "known_field",
				Type:     FieldTypeString,
				Required: false,
				Position: FieldPosition{
					XMLPath:     "/test/path",
					NodeIndex:   0,
					StartOffset: 0,
					EndOffset:   10,
				},
			},
		},
		DocumentName: "test_document",
		ExtractedAt:  time.Now(),
		TotalFields:  1,
	}

	// Provide MergeData containing both the known field and an extra unknown key
	mergeData := MergeData{
		"known_field":   "test_value",
		"unknown_field": "extra_value",
	}

	// Validate and expect no warnings
	result := fieldSet.Validate(mergeData)

	// Assert that no warnings are produced
	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings, but got %d warnings: %v", len(result.Warnings), result.Warnings)
	}

	// The validation should still be valid since the unknown field is ignored
	if !result.Valid {
		t.Errorf("Expected validation to be valid, but got invalid result with errors: %v", result.Errors)
	}

	// Should have no errors since the known field is provided and valid
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, but got %d errors: %v", len(result.Errors), result.Errors)
	}

	// Should have no missing fields since the known field is provided
	if len(result.MissingFields) != 0 {
		t.Errorf("Expected no missing fields, but got: %v", result.MissingFields)
	}
}

func TestMergeFieldSet_Validate_RequiredFieldMissing(t *testing.T) {
	// Build a MergeFieldSet with a required field
	fieldSet := MergeFieldSet{
		Fields: []MergeField{
			{
				Name:     "required_field",
				Type:     FieldTypeString,
				Required: true,
				Position: FieldPosition{
					XMLPath:     "/test/path",
					NodeIndex:   0,
					StartOffset: 0,
					EndOffset:   10,
				},
			},
		},
		DocumentName: "test_document",
		ExtractedAt:  time.Now(),
		TotalFields:  1,
	}

	// Provide MergeData with only unknown fields
	mergeData := MergeData{
		"unknown_field": "extra_value",
	}

	// Validate and expect errors for missing required field
	result := fieldSet.Validate(mergeData)

	// Should be invalid due to missing required field
	if result.Valid {
		t.Errorf("Expected validation to be invalid due to missing required field")
	}

	// Should have errors for missing required field
	if len(result.Errors) == 0 {
		t.Errorf("Expected errors for missing required field, but got none")
	}

	// Should have missing fields
	if len(result.MissingFields) != 1 {
		t.Errorf("Expected 1 missing field, but got %d: %v", len(result.MissingFields), result.MissingFields)
	}

	// Should still have no warnings for unknown fields
	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings, but got %d warnings: %v", len(result.Warnings), result.Warnings)
	}
}

func TestMergeFieldSet_Validate_ValidData(t *testing.T) {
	// Build a MergeFieldSet with mixed field types
	fieldSet := MergeFieldSet{
		Fields: []MergeField{
			{
				Name:     "string_field",
				Type:     FieldTypeString,
				Required: true,
				Position: FieldPosition{XMLPath: "/test/string"},
			},
			{
				Name:     "number_field",
				Type:     FieldTypeNumber,
				Required: false,
				Position: FieldPosition{XMLPath: "/test/number"},
			},
			{
				Name:     "date_field",
				Type:     FieldTypeDate,
				Required: false,
				Position: FieldPosition{XMLPath: "/test/date"},
			},
		},
		DocumentName: "test_document",
		ExtractedAt:  time.Now(),
		TotalFields:  3,
	}

	// Provide valid MergeData with extra unknown fields
	mergeData := MergeData{
		"string_field":  "hello",
		"number_field":  42,
		"date_field":    "2023-01-01",
		"unknown_field": "ignored",
		"extra_data":    123.45,
	}

	// Validate
	result := fieldSet.Validate(mergeData)

	// Should be valid
	if !result.Valid {
		t.Errorf("Expected validation to be valid, but got errors: %v", result.Errors)
	}

	// Should have no errors
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, but got: %v", result.Errors)
	}

	// Should have no warnings (unknown fields are ignored)
	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings, but got: %v", result.Warnings)
	}

	// Should have no missing fields
	if len(result.MissingFields) != 0 {
		t.Errorf("Expected no missing fields, but got: %v", result.MissingFields)
	}
}

func TestMergeFieldSet_Validate_InvalidDataType(t *testing.T) {
	// Build a MergeFieldSet with specific type requirements
	fieldSet := MergeFieldSet{
		Fields: []MergeField{
			{
				Name:     "string_field",
				Type:     FieldTypeString,
				Required: true,
				Position: FieldPosition{XMLPath: "/test/string"},
			},
		},
		DocumentName: "test_document",
		ExtractedAt:  time.Now(),
		TotalFields:  1,
	}

	// Provide MergeData with wrong type and unknown field
	mergeData := MergeData{
		"string_field":  123, // Should be string, not number
		"unknown_field": "ignored",
	}

	// Validate
	result := fieldSet.Validate(mergeData)

	// Should be invalid due to type mismatch
	if result.Valid {
		t.Errorf("Expected validation to be invalid due to type mismatch")
	}

	// Should have errors for type mismatch
	if len(result.Errors) == 0 {
		t.Errorf("Expected errors for type mismatch, but got none")
	}

	// Should still have no warnings for unknown fields
	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings, but got: %v", result.Warnings)
	}
}
