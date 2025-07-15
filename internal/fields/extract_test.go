package fields

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"com/lifenture/flash-mail-merge/internal/docx"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name           string
		documentXML    string
		expectedFields []string
		expectError    bool
	}{
		{
			name: "simple merge fields",
			documentXML: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
    <w:body>
        <w:p>
            <w:fldSimple w:instr=" MERGEFIELD  FirstName  \* MERGEFORMAT ">
                <w:t>«FirstName»</w:t>
            </w:fldSimple>
            <w:fldSimple w:instr=" MERGEFIELD  LastName  \* MERGEFORMAT ">
                <w:t>«LastName»</w:t>
            </w:fldSimple>
        </w:p>
    </w:body>
</w:document>`,
			expectedFields: []string{"FirstName", "LastName"},
			expectError:    false,
		},
		{
			name: "complex merge fields",
			documentXML: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
    <w:body>
        <w:p>
            <w:r>
                <w:fldChar w:fldCharType="begin"/>
            </w:r>
            <w:r>
                <w:instrText> MERGEFIELD  Email  \* MERGEFORMAT </w:instrText>
            </w:r>
            <w:r>
                <w:fldChar w:fldCharType="separate"/>
            </w:r>
            <w:r>
                <w:t>«Email»</w:t>
            </w:r>
            <w:r>
                <w:fldChar w:fldCharType="end"/>
            </w:r>
        </w:p>
    </w:body>
</w:document>`,
			expectedFields: []string{"Email"},
			expectError:    false,
		},
		{
			name: "mixed simple and complex fields",
			documentXML: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
    <w:body>
        <w:p>
            <w:fldSimple w:instr=" MERGEFIELD  FirstName  \* MERGEFORMAT ">
                <w:t>«FirstName»</w:t>
            </w:fldSimple>
            <w:r>
                <w:fldChar w:fldCharType="begin"/>
            </w:r>
            <w:r>
                <w:instrText> MERGEFIELD  Email  \* MERGEFORMAT </w:instrText>
            </w:r>
            <w:r>
                <w:fldChar w:fldCharType="separate"/>
            </w:r>
            <w:r>
                <w:t>«Email»</w:t>
            </w:r>
            <w:r>
                <w:fldChar w:fldCharType="end"/>
            </w:r>
        </w:p>
    </w:body>
</w:document>`,
			expectedFields: []string{"FirstName", "Email"},
			expectError:    false,
		},
		{
			name: "duplicate fields",
			documentXML: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
    <w:body>
        <w:p>
            <w:fldSimple w:instr=" MERGEFIELD  FirstName  \* MERGEFORMAT ">
                <w:t>«FirstName»</w:t>
            </w:fldSimple>
            <w:fldSimple w:instr=" MERGEFIELD  FirstName  \* MERGEFORMAT ">
                <w:t>«FirstName»</w:t>
            </w:fldSimple>
        </w:p>
    </w:body>
</w:document>`,
			expectedFields: []string{"FirstName"},
			expectError:    false,
		},
		{
			name: "no merge fields",
			documentXML: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
    <w:body>
        <w:p>
            <w:r>
                <w:t>Hello World</w:t>
            </w:r>
        </w:p>
    </w:body>
</w:document>`,
			expectedFields: []string{},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := Extract(tt.documentXML)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if !tt.expectError {
				// Sort both slices to compare them properly
				sort.Strings(fields)
				sort.Strings(tt.expectedFields)
				
				if !reflect.DeepEqual(fields, tt.expectedFields) {
					t.Errorf("Expected fields %v, got %v", tt.expectedFields, fields)
				}
			}
		})
	}
}

func TestExtractFromSampleDocx(t *testing.T) {
	// Get the path to the sample DOCX file
	samplePath := filepath.Join("..", "..", "tests", "data", "sample.docx")
	
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
	
	// Unzip the DOCX file
	docxFile, err := docx.UnzipDocx(docxBytes)
	if err != nil {
		t.Fatalf("Failed to unzip DOCX: %v", err)
	}
	
	// Get document XML
	documentXMLBytes, err := docxFile.GetDocumentXML()
	if err != nil {
		t.Fatalf("Failed to get document XML: %v", err)
	}
	
	// Test the Extract function
	fields, err := Extract(string(documentXMLBytes))
	if err != nil {
		t.Fatalf("Failed to extract fields: %v", err)
	}
	
	// Expected fields from our sample DOCX
	expectedFields := []string{
		"Account_Name", "Contact_FirstName", "Contact_FullName", "Contact_MailingAddress",
		"Contact_MailingCity", "Contact_MailingPostalCode", "Contact_MailingState", "Contact_Title",
		"Org_Address", "Org_City", "Org_Name", "Org_PostalCode", "Org_State", "Today",
		"User_Company", "User_Email", "User_Fax", "User_FullName", "User_Phone", "User_Title",
	}
	
	// Sort both slices for comparison
	sort.Strings(fields)
	sort.Strings(expectedFields)
	
	if !reflect.DeepEqual(fields, expectedFields) {
		t.Errorf("Expected fields %v, got %v", expectedFields, fields)
	}
	
	// Verify we got the expected count
	if len(fields) != 20 {
		t.Errorf("Expected 20 fields, got %d", len(fields))
	}
}
