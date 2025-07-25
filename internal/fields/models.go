package fields

import (
	"fmt"
	"strings"
	"time"
)

// normalize standardizes field names for consistent comparison
func normalize(s string) string {
	return strings.ToLower(s)
}

// MergeField represents a merge field found in a document
type MergeField struct {
	// Name is the field name/identifier
	Name string `json:"name"`
	
	// Type indicates the expected data type
	Type FieldType `json:"type"`
	
	// Position information for the field in the document
	Position FieldPosition `json:"position"`
	
	// DefaultValue is the default value if no data is provided
	DefaultValue interface{} `json:"default_value,omitempty"`
	
	// Required indicates if this field must have a value
	Required bool `json:"required"`
	
	// Format specifies formatting options for the field
	Format *FieldFormat `json:"format,omitempty"`
}

// FieldType represents the data type of a merge field
type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeNumber   FieldType = "number"
	FieldTypeDate     FieldType = "date"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeImage    FieldType = "image"
	FieldTypeTable    FieldType = "table"
	FieldTypeUnknown  FieldType = "unknown"
)

// FieldPosition represents the location of a field in the document
type FieldPosition struct {
	// XMLPath is the path to the field in the XML structure
	XMLPath string `json:"xml_path"`
	
	// NodeIndex is the index of the node containing the field
	NodeIndex int `json:"node_index"`
	
	// StartOffset is the character offset where the field starts
	StartOffset int `json:"start_offset"`
	
	// EndOffset is the character offset where the field ends
	EndOffset int `json:"end_offset"`
	
	// Page number (if determinable)
	Page int `json:"page,omitempty"`
}

// FieldFormat contains formatting options for a field
type FieldFormat struct {
	// DateFormat for date fields (e.g., "2006-01-02", "January 2, 2006")
	DateFormat string `json:"date_format,omitempty"`
	
	// NumberFormat for number fields (e.g., "currency", "percentage")
	NumberFormat string `json:"number_format,omitempty"`
	
	// TextTransform for string fields (e.g., "uppercase", "lowercase", "title")
	TextTransform string `json:"text_transform,omitempty"`
	
	// Prefix to add before the field value
	Prefix string `json:"prefix,omitempty"`
	
	// Suffix to add after the field value
	Suffix string `json:"suffix,omitempty"`
}

// MergeFieldSet represents a collection of merge fields
type MergeFieldSet struct {
	// Fields is the list of discovered merge fields
	Fields []MergeField `json:"fields"`
	
	// DocumentName is the name of the source document
	DocumentName string `json:"document_name"`
	
	// ExtractedAt is when the fields were extracted
	ExtractedAt time.Time `json:"extracted_at"`
	
	// TotalFields is the count of fields found
	TotalFields int `json:"total_fields"`
	
	// normalizedFieldMap is a cached map for fast case-insensitive field lookups
	// Maps normalized field names to MergeField pointers
	normalizedFieldMap map[string]*MergeField `json:"-"`
}

// MergeData represents the data to be merged into fields
type MergeData map[string]interface{}

// ValidationResult represents the result of field validation
type ValidationResult struct {
	// Valid indicates if the data is valid for merging
	Valid bool `json:"valid"`
	
	// Errors contains validation error messages
	Errors []string `json:"errors,omitempty"`
	
	// Warnings contains validation warnings
	Warnings []string `json:"warnings,omitempty"`
	
	// MissingFields lists required fields that are missing
	MissingFields []string `json:"missing_fields,omitempty"`
}

// String returns a string representation of the merge field
func (mf MergeField) String() string {
	return fmt.Sprintf("MergeField{Name: %s, Type: %s, Required: %v}", mf.Name, mf.Type, mf.Required)
}


// GetRequiredFields returns only the required fields from the set
func (mfs MergeFieldSet) GetRequiredFields() []MergeField {
	var required []MergeField
	for _, field := range mfs.Fields {
		if field.Required {
			required = append(required, field)
		}
	}
	return required
}

// buildNormalizedFieldMap builds the normalized field map if it doesn't exist
func (mfs *MergeFieldSet) buildNormalizedFieldMap() {
	if mfs.normalizedFieldMap == nil {
		mfs.normalizedFieldMap = make(map[string]*MergeField)
		for i := range mfs.Fields {
			normalizedName := normalize(mfs.Fields[i].Name)
			mfs.normalizedFieldMap[normalizedName] = &mfs.Fields[i]
		}
	}
}

// GetFieldByName returns a field by its name, or nil if not found
func (mfs *MergeFieldSet) GetFieldByName(name string) *MergeField {
	mfs.buildNormalizedFieldMap()
	normalizedName := normalize(name)
	return mfs.normalizedFieldMap[normalizedName]
}

// HasField checks if a field with the given name exists
func (mfs *MergeFieldSet) HasField(name string) bool {
	return mfs.GetFieldByName(name) != nil
}

// Validate checks if the provided merge data is valid for this field set.
// Extra keys in the data that don't match any field are silently ignored.
// Warnings in the result only come from duplicate-key detection performed in main.go.
func (mfs *MergeFieldSet) Validate(data MergeData) ValidationResult {
	result := ValidationResult{
		Valid:         true,
		Errors:        []string{},
		Warnings:      []string{},
		MissingFields: []string{},
	}

	// Check required fields
	for _, field := range mfs.GetRequiredFields() {
		// Check if field exists using normalized name matching
		found := false
		normalizedFieldName := normalize(field.Name)
		for dataKey := range data {
			if normalize(dataKey) == normalizedFieldName {
				found = true
				break
			}
		}
		if !found {
			result.Valid = false
			result.MissingFields = append(result.MissingFields, field.Name)
			result.Errors = append(result.Errors, fmt.Sprintf("Required field '%s' is missing", field.Name))
		}
	}

	// Validate data types and formats
	for fieldName, value := range data {
		field := mfs.GetFieldByName(fieldName)
		if field == nil {
			continue   // silently ignore data keys not present in template
		}

		if err := validateFieldValue(field, value); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid value for field '%s': %s", fieldName, err.Error()))
		}
	}

	return result
}

// validateFieldValue validates a single field value against its type
func validateFieldValue(field *MergeField, value interface{}) error {
	if value == nil {
		if field.Required {
			return fmt.Errorf("field is required but value is nil")
		}
		return nil
	}

	switch field.Type {
	case FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case FieldTypeNumber:
		switch value.(type) {
		case int, int64, float64, float32:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case FieldTypeDate:
		switch v := value.(type) {
		case string:
			if _, err := time.Parse("2006-01-02", v); err != nil {
				return fmt.Errorf("invalid date format: %s", err.Error())
			}
		case time.Time:
			// Valid
		default:
			return fmt.Errorf("expected date string or time.Time, got %T", value)
		}
	case FieldTypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	}

	return nil
}


// ToLower converts all string values to lowercase
func (md MergeData) ToLower() MergeData {
	result := make(MergeData)
	for key, value := range md {
		if str, ok := value.(string); ok {
			result[key] = strings.ToLower(str)
		} else {
			result[key] = value
		}
	}
	return result
}
