package fields

import (
	"encoding/xml"
	"strings"
	"time"
	"com/lifenture/flash-mail-merge/internal/docx"
)

// Extract extracts field names from a DOCX document XML string
func Extract(documentXML string) ([]string, error) {
	decoder := xml.NewDecoder(strings.NewReader(documentXML))
	fieldNames := make(map[string]struct{})

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch token := tok.(type) {
		case xml.StartElement:
			name := token.Name.Local

			// Check for simple fields
			if name == "fldSimple" {
				for _, attr := range token.Attr {
					if attr.Name.Local == "instr" && strings.Contains(attr.Value, "MERGEFIELD") {
						parts := strings.Fields(attr.Value)
						if len(parts) > 1 {
							fieldName := parts[1]
							fieldNames[fieldName] = struct{}{}
						}
					}
				}
			}

			// Check for complex fields
			if name == "fldChar" {
				fieldType, found := getFieldCharType(token)
				if found && fieldType == "begin" {
					str, ok := extractComplexField(decoder)
					if ok {
						fieldNames[str] = struct{}{}
					}
				}
			}
		}
	}

	// Convert fieldNames map to a slice
	uniqueFieldNames := make([]string, 0, len(fieldNames))
	for name := range fieldNames {
		uniqueFieldNames = append(uniqueFieldNames, name)
	}

	return uniqueFieldNames, nil
}

// ExtractFields extracts merge fields from the given DOCX document
func ExtractFields(doc *docx.DocxFile) (*MergeFieldSet, error) {
	docContent, err := doc.GetDocumentXML()
	if err != nil {
		return nil, err
	}

	// Get field names using the Extract function
	fieldNames, err := Extract(string(docContent))
	if err != nil {
		return nil, err
	}

	// Convert field names to MergeField structs
	fields := make([]MergeField, len(fieldNames))
	for i, name := range fieldNames {
		fields[i] = MergeField{
			Name:     name,
			Type:     FieldTypeString,
			Required: false,
		}
	}

	return &MergeFieldSet{
		Fields:       fields,
		ExtractedAt:  time.Now(),
		TotalFields:  len(fields),
		DocumentName: "document.docx",
	}, nil
}

// Extract text from complex field
func extractComplexField(decoder *xml.Decoder) (string, bool) {
	var name string
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch token := tok.(type) {
		case xml.StartElement:
			if token.Name.Local == "instrText" {
				var value string
				decoder.DecodeElement(&value, &token)
				if strings.Contains(value, "MERGEFIELD") {
					parts := strings.Fields(value)
					if len(parts) > 1 {
						name = parts[1]
					}
				}
			} else if token.Name.Local == "fldChar" {
				// Check if this is the end of the field
				fieldType, found := getFieldCharType(token)
				if found && fieldType == "end" {
					return name, name != ""
				}
			}
		}
	}
	return "", false
}

// Get field char type
func getFieldCharType(token xml.StartElement) (string, bool) {
	for _, attr := range token.Attr {
		if attr.Name.Local == "fldCharType" {
			return attr.Value, true
		}
	}
	return "", false
}
