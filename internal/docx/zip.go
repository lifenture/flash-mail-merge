package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
)

// Rebuild creates a new DOCX file (ZIP archive) from a DocxFile struct
// with the updated document.xml content (merged XML).
// It preserves all original files except word/document.xml, which uses
// the merged XML content from the DocxFile.
func Rebuild(doc *DocxFile) ([]byte, error) {
	// Create a bytes buffer and ZIP writer
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Iterate over all files in the original document
	for filename, content := range doc.Files {
		// Create a file in the ZIP archive
		fileWriter, err := zipWriter.Create(filename)
		if err != nil {
			zipWriter.Close()
			return nil, fmt.Errorf("failed to create file %s in ZIP: %w", filename, err)
		}

		// Write the file content
		// For word/document.xml, this will be the merged XML content
		// For all other files, this will be the original content
		_, err = fileWriter.Write(content)
		if err != nil {
			zipWriter.Close()
			return nil, fmt.Errorf("failed to write content for file %s: %w", filename, err)
		}
	}

	// Close the ZIP writer
	err := zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	// Return the buffer bytes
	return buf.Bytes(), nil
}
