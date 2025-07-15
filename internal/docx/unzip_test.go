package docx

import (
	"testing"
)


func TestUnzipDocx(t *testing.T) {
	// Test with invalid data
	t.Run("invalid data", func(t *testing.T) {
		buf := []byte{0x50, 0x4B, 0x03, 0x04}
		_, err := UnzipDocx(buf)
		if err == nil {
			t.Error("expected error for invalid data")
		}
	})
}

func TestDocxFile_IsValidDocx(t *testing.T) {
	// Test with empty DocxFile
	t.Run("empty DocxFile", func(t *testing.T) {
		docx := &DocxFile{Files: make(map[string][]byte)}
		if docx.IsValidDocx() {
			t.Error("expected false for empty DocxFile")
		}
	})

	// Test with minimal required files but missing content types
	t.Run("missing content types", func(t *testing.T) {
		docx := &DocxFile{
			Files: map[string][]byte{
				"word/document.xml": []byte("<document></document>"),
				"[Content_Types].xml": []byte("<Types></Types>"),
				"_rels/.rels": []byte("<Relationships></Relationships>"),
			},
		}
		if docx.IsValidDocx() {
			t.Error("expected false for missing DOCX content type")
		}
	})

	// Test with valid DOCX structure
	t.Run("valid DOCX structure", func(t *testing.T) {
		docx := &DocxFile{
			Files: map[string][]byte{
				"word/document.xml": []byte("<document></document>"),
				"[Content_Types].xml": []byte(`<Types><Default Extension="xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`),
				"_rels/.rels": []byte("<Relationships></Relationships>"),
			},
		}
		if !docx.IsValidDocx() {
			t.Error("expected true for valid DOCX structure")
		}
	})
}
