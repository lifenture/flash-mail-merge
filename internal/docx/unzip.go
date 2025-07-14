package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// DocxFile represents a DOCX file structure
type DocxFile struct {
	Files map[string][]byte
}

// UnzipDocx extracts the contents of a DOCX file from byte data
func UnzipDocx(data []byte) (*DocxFile, error) {
	reader := bytes.NewReader(data)
	zipReader, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	docx := &DocxFile{
		Files: make(map[string][]byte),
	}

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		content, err := readZipFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file.Name, err)
		}

		docx.Files[file.Name] = content
	}

	return docx, nil
}

// readZipFile reads the content of a single file from the zip archive
func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// GetDocumentXML returns the main document XML content
func (d *DocxFile) GetDocumentXML() ([]byte, error) {
	content, exists := d.Files["word/document.xml"]
	if !exists {
		return nil, fmt.Errorf("document.xml not found in DOCX file")
	}
	return content, nil
}

// GetContentTypes returns the content types XML
func (d *DocxFile) GetContentTypes() ([]byte, error) {
	content, exists := d.Files["[Content_Types].xml"]
	if !exists {
		return nil, fmt.Errorf("[Content_Types].xml not found in DOCX file")
	}
	return content, nil
}

// ListFiles returns a list of all files in the DOCX archive
func (d *DocxFile) ListFiles() []string {
	files := make([]string, 0, len(d.Files))
	for filename := range d.Files {
		files = append(files, filename)
	}
	return files
}

// HasFile checks if a specific file exists in the DOCX archive
func (d *DocxFile) HasFile(filename string) bool {
	_, exists := d.Files[filename]
	return exists
}

// GetFile retrieves the content of a specific file from the DOCX archive
func (d *DocxFile) GetFile(filename string) ([]byte, error) {
	content, exists := d.Files[filename]
	if !exists {
		return nil, fmt.Errorf("file %s not found in DOCX archive", filename)
	}
	return content, nil
}

// IsValidDocx performs basic validation to ensure this is a valid DOCX file
func (d *DocxFile) IsValidDocx() bool {
	// Check for essential DOCX files
	requiredFiles := []string{
		"word/document.xml",
		"[Content_Types].xml",
		"_rels/.rels",
	}

	for _, file := range requiredFiles {
		if !d.HasFile(file) {
			return false
		}
	}

	// Check content types for Word document
	contentTypes, err := d.GetContentTypes()
	if err != nil {
		return false
	}

	return strings.Contains(string(contentTypes), "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml")
}

// ReadDocumentXML opens DOCX bytes, validates the signature, and returns word/document.xml content
func ReadDocumentXML(buf []byte) (string, error) {
	// Validate that this is a ZIP file (DOCX signature)
	if len(buf) < 4 {
		return "", fmt.Errorf("invalid DOCX file: too short")
	}

	// Check for ZIP signature (PK header)
	if buf[0] != 0x50 || buf[1] != 0x4B {
		return "", fmt.Errorf("invalid DOCX file: missing ZIP signature")
	}

	// Unzip the DOCX file
	docx, err := UnzipDocx(buf)
	if err != nil {
		return "", fmt.Errorf("failed to unzip DOCX: %w", err)
	}

	// Validate that this is a valid DOCX file
	if !docx.IsValidDocx() {
		return "", fmt.Errorf("invalid DOCX file: missing required DOCX structure")
	}

	// Get the document XML content
	documentXML, err := docx.GetDocumentXML()
	if err != nil {
		return "", fmt.Errorf("failed to get document XML: %w", err)
	}

	return string(documentXML), nil
}
