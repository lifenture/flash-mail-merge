package merge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"com/lifenture/flash-mail-merge/internal/docx"
	"com/lifenture/flash-mail-merge/internal/fields"
)

func TestReplaceFieldValues(t *testing.T) {
	// Test XML with merge fields in the expected format
	xml := `<w:document>
		<w:body>
			<w:p>
				<w:r><w:t>«name»</w:t></w:r>
				<w:r><w:t> is </w:t></w:r>
				<w:r><w:t>«age»</w:t></w:r>
				<w:r><w:t> years old</w:t></w:r>
			</w:p>
		</w:body>
	</w:document>`

	// Test merge data
	data := fields.MergeData{
		"name": "John Doe",
		"age":  30,
	}

	result, skipped, err := replaceFieldValues(xml, data)
	if err != nil {
		t.Fatalf("replaceFieldValues failed: %v", err)
	}

	// Check that fields were replaced
	if !strings.Contains(result, "John Doe") {
		t.Error("Name field was not replaced")
	}
	if !strings.Contains(result, "30") {
		t.Error("Age field was not replaced")
	}

	// Should have no skipped fields
	if len(skipped) != 0 {
		t.Errorf("Expected no skipped fields, got %v", skipped)
	}
}

func TestReplaceFieldValuesWithMultipleRuns(t *testing.T) {
	// Test field replacement with formatting
	xml := `<w:document>
		<w:body>
			<w:p>
				<w:r><w:rPr><w:b/></w:rPr><w:t>«fullname»</w:t></w:r>
			</w:p>
		</w:body>
	</w:document>`

	data := fields.MergeData{
		"fullname": "John Doe",
	}

	result, skipped, err := replaceFieldValues(xml, data)
	if err != nil {
		t.Fatalf("replaceFieldValues failed: %v", err)
	}

	if !strings.Contains(result, "John Doe") {
		t.Error("Fullname field was not replaced correctly")
	}

	// Should preserve the run's formatting (bold)
	if !strings.Contains(result, "<w:rPr><w:b/></w:rPr>") {
		t.Error("Run's formatting (bold) was not preserved")
	}

	if len(skipped) != 0 {
		t.Errorf("Expected no skipped fields, got %v", skipped)
	}
}

func TestReplaceFieldValuesWithSkipped(t *testing.T) {
	// Test XML with merge fields, one missing
	xml := `<w:document>
		<w:body>
			<w:p>
				<w:r><w:t>«name»</w:t></w:r>
				<w:r><w:t> has email: </w:t></w:r>
				<w:r><w:t>«email»</w:t></w:r>
			</w:p>
		</w:body>
	</w:document>`

	// Test merge data - missing email field
	data := fields.MergeData{
		"name": "John Doe",
	}

	result, skipped, err := replaceFieldValues(xml, data)
	if err != nil {
		t.Fatalf("replaceFieldValues failed: %v", err)
	}

	// Check that name field was replaced
	if !strings.Contains(result, "John Doe") {
		t.Error("Name field was not replaced")
	}

	// Check that email field was not replaced - should contain original placeholder
	if !strings.Contains(result, "«email»") {
		t.Error("Email field should remain as placeholder")
	}

	// Should have one skipped field
	if len(skipped) != 1 || skipped[0] != "email" {
		t.Errorf("Expected skipped fields [email], got %v", skipped)
	}
}


func TestPerformMerge(t *testing.T) {
	// Create a minimal DOCX structure
	doc := &docx.DocxFile{
		Files: map[string][]byte{
			"word/document.xml": []byte(`<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>Hello {{name}}!</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`),
			"[Content_Types].xml": []byte(`<?xml version="1.0"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="xml" ContentType="application/xml"/>
	<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`),
			"_rels/.rels": []byte(`<?xml version="1.0"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`),
		},
	}

	// Test merge data
	data := fields.MergeData{
		"name": "Jane Smith",
	}

	mergedDoc, skipped, err := PerformMerge(doc, data)
	if err != nil {
		t.Fatalf("PerformMerge failed: %v", err)
	}

	// Should have no skipped fields
	if len(skipped) != 0 {
		t.Errorf("Expected no skipped fields, got %v", skipped)
	}

	// Should return valid DOCX bytes
	if len(mergedDoc) == 0 {
		t.Error("Expected non-empty merged document")
	}

	// Check that it starts with ZIP signature
	if len(mergedDoc) < 4 || mergedDoc[0] != 0x50 || mergedDoc[1] != 0x4B {
		t.Error("Merged document does not have valid ZIP signature")
	}
}

// Helper function to create a valid DOCX file with specified merge fields
func createSampleDocx(documentXML string) *docx.DocxFile {
	return &docx.DocxFile{
		Files: map[string][]byte{
			"word/document.xml": []byte(documentXML),
			"[Content_Types].xml": []byte(`<?xml version="1.0"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="xml" ContentType="application/xml"/>
	<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`),
			"_rels/.rels": []byte(`<?xml version="1.0"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`),
			"word/_rels/document.xml.rels": []byte(`<?xml version="1.0"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`),
		},
	}
}

// Helper function to create a DOCX as bytes
func createSampleDocxBytes(documentXML string) []byte {
	doc := createSampleDocx(documentXML)
	mergedBytes, err := rebuildDocxArchive(doc)
	if err != nil {
		panic(err)
	}
	return mergedBytes
}

// TestPerformMergeAllFieldsReplaced tests the scenario where all fields are replaced
func TestPerformMergeAllFieldsReplaced(t *testing.T) {
	// Create a DOCX with multiple merge fields using the expected format
	documentXML := `<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r><w:t>«firstname»</w:t></w:r>
			<w:r><w:t> </w:t></w:r>
			<w:r><w:t>«lastname»</w:t></w:r>
		</w:p>
		<w:p>
			<w:r><w:t>Age: </w:t></w:r>
			<w:r><w:t>«age»</w:t></w:r>
		</w:p>
		<w:p>
			<w:r><w:t>Email: </w:t></w:r>
			<w:r><w:rPr><w:b/></w:rPr><w:t>«email»</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`

	doc := createSampleDocx(documentXML)

	// Provide data for all fields
	data := fields.MergeData{
		"firstname": "Alice",
		"lastname":  "Johnson",
		"age":       35,
		"email":     "alice.johnson@company.com",
	}

	mergedDoc, skipped, err := PerformMerge(doc, data)
	if err != nil {
		t.Fatalf("PerformMerge failed: %v", err)
	}

	// Should have no skipped fields
	if len(skipped) != 0 {
		t.Errorf("Expected no skipped fields, got %v", skipped)
	}

	// Should return valid DOCX bytes
	if len(mergedDoc) == 0 {
		t.Error("Expected non-empty merged document")
	}

	// Check that it starts with ZIP signature
	if len(mergedDoc) < 4 || mergedDoc[0] != 0x50 || mergedDoc[1] != 0x4B {
		t.Error("Merged document does not have valid ZIP signature")
	}

	// Verify the merged document contains the replaced values
	mergedDocx, err := docx.UnzipDocx(mergedDoc)
	if err != nil {
		t.Fatalf("Failed to unzip merged document: %v", err)
	}

	mergedXML, err := mergedDocx.GetDocumentXML()
	if err != nil {
		t.Fatalf("Failed to get merged document XML: %v", err)
	}

	mergedContent := string(mergedXML)
	if !strings.Contains(mergedContent, "Alice") {
		t.Errorf("Expected 'Alice' in merged content, got: %s", mergedContent)
	}
	if !strings.Contains(mergedContent, "Johnson") {
		t.Errorf("Expected 'Johnson' in merged content, got: %s", mergedContent)
	}
	if !strings.Contains(mergedContent, "35") {
		t.Errorf("Expected '35' in merged content, got: %s", mergedContent)
	}
	if !strings.Contains(mergedContent, "alice.johnson@company.com") {
		t.Errorf("Expected 'alice.johnson@company.com' in merged content, got: %s", mergedContent)
	}

	// Check formatting
	if !strings.Contains(mergedContent, "<w:b/>") {
		t.Errorf("Expected formatting in merged content, got: %s", mergedContent)
	}
}

// TestPerformMergeSomeFieldsMissing tests the scenario where some fields are missing
func TestPerformMergeSomeFieldsMissing(t *testing.T) {
	// Create a DOCX with multiple merge fields using the expected format
	documentXML := `<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r><w:t>«firstname»</w:t></w:r>
			<w:r><w:t> </w:t></w:r>
			<w:r><w:t>«lastname»</w:t></w:r>
		</w:p>
		<w:p>
			<w:r><w:t>Age: </w:t></w:r>
			<w:r><w:t>«age»</w:t></w:r>
		</w:p>
		<w:p>
			<w:r><w:t>Phone: </w:t></w:r>
			<w:r><w:t>«phone»</w:t></w:r>
		</w:p>
		<w:p>
			<w:r><w:t>Department: </w:t></w:r>
			<w:r><w:t>«department»</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`

	doc := createSampleDocx(documentXML)

	// Provide data for only some fields - missing 'age' and 'department'
	data := fields.MergeData{
		"firstname": "Bob",
		"lastname":  "Smith",
		"phone":     "555-9876",
		// Missing: age, department
	}

	mergedDoc, skipped, err := PerformMerge(doc, data)
	if err != nil {
		t.Fatalf("PerformMerge failed: %v", err)
	}

	// Should have exactly 2 skipped fields
	if len(skipped) != 2 {
		t.Errorf("Expected 2 skipped fields, got %d: %v", len(skipped), skipped)
	}

	// Check that the skipped fields are correct
	expectedSkipped := []string{"age", "department"}
	for _, expected := range expectedSkipped {
		found := false
		for _, actual := range skipped {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field '%s' to be skipped, but it wasn't in %v", expected, skipped)
		}
	}

	// Should return valid DOCX bytes
	if len(mergedDoc) == 0 {
		t.Error("Expected non-empty merged document")
	}

	// Check that it starts with ZIP signature
	if len(mergedDoc) < 4 || mergedDoc[0] != 0x50 || mergedDoc[1] != 0x4B {
		t.Error("Merged document does not have valid ZIP signature")
	}

	// Verify the merged document contains the replaced values
	mergedDocx, err := docx.UnzipDocx(mergedDoc)
	if err != nil {
		t.Fatalf("Failed to unzip merged document: %v", err)
	}

	mergedXML, err := mergedDocx.GetDocumentXML()
	if err != nil {
		t.Fatalf("Failed to get merged document XML: %v", err)
	}

	mergedContent := string(mergedXML)
	
	// Should contain replaced values
	if !strings.Contains(mergedContent, "Bob") {
		t.Error("Merged document should contain 'Bob'")
	}
	if !strings.Contains(mergedContent, "Smith") {
		t.Error("Merged document should contain 'Smith'")
	}
	if !strings.Contains(mergedContent, "555-9876") {
		t.Error("Merged document should contain '555-9876'")
	}

	// Should still contain the original field placeholders for skipped fields
	if !strings.Contains(mergedContent, "«age»") {
		t.Error("Merged document should still contain '«age»' placeholder")
	}
	if !strings.Contains(mergedContent, "«department»") {
		t.Error("Merged document should still contain '«department»' placeholder")
	}
}

// TestPerformMergeInvalidDocxStructure tests the scenario with invalid DOCX structure
func TestPerformMergeInvalidDocxStructure(t *testing.T) {
	tests := []struct {
		name        string
		doc         *docx.DocxFile
		expectedErr string
	}{
		{
			name: "missing document.xml",
			doc: &docx.DocxFile{
				Files: map[string][]byte{
					"[Content_Types].xml": []byte(`<?xml version="1.0"?><Types/>`),
					"_rels/.rels":         []byte(`<?xml version="1.0"?><Relationships/>`),
					// Missing word/document.xml
				},
			},
			expectedErr: "document.xml not found",
		},
		{
			name: "empty document.xml",
			doc: &docx.DocxFile{
				Files: map[string][]byte{
					"word/document.xml":   []byte(""), // Empty document
					"[Content_Types].xml": []byte(`<?xml version="1.0"?><Types/>`),
					"_rels/.rels":         []byte(`<?xml version="1.0"?><Relationships/>`),
				},
			},
			expectedErr: "", // Empty document should not fail, just return empty result
		},
		{
			name: "corrupted document.xml",
			doc: &docx.DocxFile{
				Files: map[string][]byte{
					"word/document.xml":   []byte(`<invalid>xml<content`), // Malformed XML
					"[Content_Types].xml": []byte(`<?xml version="1.0"?><Types/>`),
					"_rels/.rels":         []byte(`<?xml version="1.0"?><Relationships/>`),
				},
			},
			expectedErr: "", // Malformed XML should not fail the merge process itself
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test merge data
			data := fields.MergeData{
				"name": "Test User",
			}

			mergedDoc, skipped, err := PerformMerge(tt.doc, data)

			if tt.expectedErr != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.expectedErr)
				} else if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.expectedErr, err)
				}
				return
			}

			// For cases where no error is expected
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Should return some result (even if empty)
			if mergedDoc == nil {
				t.Error("Expected non-nil merged document")
			}

			// Skipped fields should be empty or contain expected values
			_ = skipped // We don't have specific expectations for skipped fields in these tests
		})
	}
}

// TestPerformMergeWithCorruptedZip tests the scenario with corrupted ZIP data
func TestPerformMergeWithCorruptedZip(t *testing.T) {
	// Create a DOCX with valid structure but then corrupt the ZIP during rebuild
	documentXML := `<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:fldSimple w:instr="MERGEFIELD name">
					<w:t>John</w:t>
				</w:fldSimple>
			</w:r>
		</w:p>
	</w:body>
</w:document>`

	doc := createSampleDocx(documentXML)

	// Test merge data
	data := fields.MergeData{
		"name": "Test User",
	}

	// This should succeed since the issue is in ZIP creation, not in the merge logic
	mergedDoc, skipped, err := PerformMerge(doc, data)
	if err != nil {
		t.Errorf("PerformMerge should succeed even with potential ZIP issues: %v", err)
	}

	// Should have no skipped fields
	if len(skipped) != 0 {
		t.Errorf("Expected no skipped fields, got %v", skipped)
	}

	// Should return valid DOCX bytes
	if len(mergedDoc) == 0 {
		t.Error("Expected non-empty merged document")
	}

	// Test the actual ZIP corruption scenario by trying to read invalid ZIP data
	invalidZip := []byte{0x50, 0x4B, 0x03, 0x04} // Valid ZIP signature but corrupted content
	_, err = docx.UnzipDocx(invalidZip)
	if err == nil {
		t.Error("Expected error when trying to unzip corrupted ZIP data")
	}
}

// TestPerformMergeWithRealDocxFile tests with an actual DOCX file if available
func TestPerformMergeWithRealDocxFile(t *testing.T) {
	// Try to load the sample DOCX file
	samplePath := filepath.Join("../../tests/data/sample.docx")
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("Skipping test: sample.docx not found")
		return
	}

	// Read the sample DOCX file
	docxBytes, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("Failed to read sample DOCX file: %v", err)
	}

	// Unzip the DOCX file
	doc, err := docx.UnzipDocx(docxBytes)
	if err != nil {
		t.Fatalf("Failed to unzip sample DOCX file: %v", err)
	}

	// Perform merge with sample data matching the actual field names in sample.docx
	data := fields.MergeData{
		"Org_Name":                    "Test Organization",
		"Org_Address":                 "123 Test Street",
		"Org_City":                    "Test City",
		"Org_State":                   "CA",
		"Org_PostalCode":              "12345",
		"Today":                       "2024-01-15",
		"Contact_FullName":            "Integration Test User",
		"Contact_Title":               "Test Manager",
		"Account_Name":                "Test Account",
		"Contact_MailingAddress":      "456 Mail Street",
		"Contact_MailingCity":         "Mail City",
		"Contact_MailingState":        "NY",
		"Contact_MailingPostalCode":   "67890",
	}

	mergedDoc, skipped, err := PerformMerge(doc, data)
	if err != nil {
		t.Fatalf("PerformMerge failed with real DOCX file: %v", err)
	}

	// Should return valid DOCX bytes
	if len(mergedDoc) == 0 {
		t.Error("Expected non-empty merged document")
	}

	// Check that it starts with ZIP signature
	if len(mergedDoc) < 4 || mergedDoc[0] != 0x50 || mergedDoc[1] != 0x4B {
		t.Error("Merged document does not have valid ZIP signature")
	}

	// Log skipped fields for debugging
	if len(skipped) > 0 {
		t.Logf("Skipped fields: %v", skipped)
	}

	// Try to unzip the merged document to verify it's valid
	mergedDocx, err := docx.UnzipDocx(mergedDoc)
	if err != nil {
		t.Fatalf("Failed to unzip merged document: %v", err)
	}

	// Verify the merged document is valid
	if !mergedDocx.IsValidDocx() {
		t.Error("Merged document is not a valid DOCX file")
	}

	// Verify that some data was actually merged
	mergedXML, err := mergedDocx.GetDocumentXML()
	if err != nil {
		t.Fatalf("Failed to get merged document XML: %v", err)
	}

	mergedContent := string(mergedXML)
	
	// Should contain at least one of the merged values
	hasReplacedContent := strings.Contains(mergedContent, "Integration Test User") ||
		strings.Contains(mergedContent, "test@example.com") ||
		strings.Contains(mergedContent, "Test Company") ||
		strings.Contains(mergedContent, "2024-01-15")

	if !hasReplacedContent {
		t.Error("Merged document should contain at least one replaced value")
	}
}

// TestPerformMergeWithSpecialCharacters tests merging with special characters
func TestPerformMergeWithSpecialCharacters(t *testing.T) {
	// Create a DOCX with a simple merge field using expected format
	documentXML := `<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r><w:t>«message»</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`

	doc := createSampleDocx(documentXML)

	// Test data with special characters that need XML escaping
	data := fields.MergeData{
		"message": `Hello <world> & "friends"! It's a 'test'.`,
	}

	mergedDoc, skipped, err := PerformMerge(doc, data)
	if err != nil {
		t.Fatalf("PerformMerge failed: %v", err)
	}

	// Should have no skipped fields
	if len(skipped) != 0 {
		t.Errorf("Expected no skipped fields, got %v", skipped)
	}

	// Verify the merged document contains properly escaped content
	mergedDocx, err := docx.UnzipDocx(mergedDoc)
	if err != nil {
		t.Fatalf("Failed to unzip merged document: %v", err)
	}

	mergedXML, err := mergedDocx.GetDocumentXML()
	if err != nil {
		t.Fatalf("Failed to get merged document XML: %v", err)
	}

	mergedContent := string(mergedXML)
	
	// Should contain properly escaped XML
	if !strings.Contains(mergedContent, "&lt;world&gt;") {
		t.Error("Merged document should contain escaped '<world>'")
	}
	if !strings.Contains(mergedContent, "&amp;") {
		t.Error("Merged document should contain escaped '&'")
	}
	if !strings.Contains(mergedContent, "&quot;friends&quot;") {
		t.Error("Merged document should contain escaped quotes")
	}
	if !strings.Contains(mergedContent, "&#39;test&#39;") {
		t.Error("Merged document should contain escaped single quotes")
	}

	// Should NOT contain unescaped special characters
	if strings.Contains(mergedContent, "<world>") {
		t.Error("Merged document should not contain unescaped '<world>'")
	}
	if strings.Contains(mergedContent, `"friends"`) && !strings.Contains(mergedContent, `&quot;friends&quot;`) {
		t.Error("Merged document should not contain unescaped quotes")
	}
}
