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

// HasFile checks if a specific file exists in the DOCX archive
func (d *DocxFile) HasFile(filename string) bool {
	_, exists := d.Files[filename]
	return exists
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

