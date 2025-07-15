package merge

import (
	"archive/zip"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"com/lifenture/flash-mail-merge/internal/docx"
	"com/lifenture/flash-mail-merge/internal/fields"
	"com/lifenture/flash-mail-merge/internal/logging"
)

// PerformMerge performs mail merge on a DOCX document with the provided data
func PerformMerge(doc *docx.DocxFile, data fields.MergeData) (mergedDoc []byte, skipped []string, err error) {
	logging.Debug("Starting mail merge with %d available data fields", len(data))

	// Get the document XML content
	documentXML, err := doc.GetDocumentXML()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get document XML: %w", err)
	}
	logging.Debug("Retrieved document XML content (%d bytes)", len(documentXML))

	// Replace field values in the document XML
	updatedXML, skippedFields, err := replaceFieldValues(string(documentXML), data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to replace field values: %w", err)
	}
	logging.Debug("Field replacement completed - processed fields with %d skipped", len(skippedFields))
	if len(skippedFields) > 0 {
		logging.Debug("Skipped fields: %v", skippedFields)
	}

	// Create a new DOCX file with the updated document XML
	updatedDoc := &docx.DocxFile{
		Files: make(map[string][]byte),
	}

	// Copy all files from the original document
	fileCount := 0
	for filename, content := range doc.Files {
		updatedDoc.Files[filename] = content
		fileCount++
	}
	logging.Debug("Copied %d files from original document to updated document", fileCount)

	// Replace the document XML with the updated version
	updatedDoc.Files["word/document.xml"] = []byte(updatedXML)
	logging.Debug("Updated document XML content (%d bytes)", len(updatedXML))

	// Rebuild the DOCX (ZIP) archive
	logging.Debug("Starting ZIP archive rebuild")
	mergedBytes, err := rebuildDocxArchive(updatedDoc)
	if err != nil {
		logging.Error("ZIP rebuild failed: %v", err)
		return nil, nil, fmt.Errorf("failed to rebuild DOCX archive: %w", err)
	}
	logging.Debug("ZIP rebuild successful - generated %d bytes", len(mergedBytes))

	return mergedBytes, skippedFields, nil
}

// replaceFieldValues replaces merge fields in the XML with actual values
func replaceFieldValues(documentXML string, data fields.MergeData) (string, []string, error) {
	var skipped []string
	processedFields := make(map[string]bool)

	// Process the XML to replace merge fields
	result := documentXML

	// Find and replace all merge fields by looking for <w:t>«fieldname»</w:t> pattern
	logging.Debug("Processing merge fields")
	result, fieldSkipped := replaceFields(result, data, processedFields)
	skipped = append(skipped, fieldSkipped...)
	logging.Debug("Field processing completed: %d fields skipped", len(fieldSkipped))

	logging.Debug("Total fields processed: %d, Total fields skipped: %d", len(processedFields), len(skipped))
	return result, skipped, nil
}

// replaceFields handles all merge fields by looking for <w:t>«fieldname»</w:t> pattern
func replaceFields(documentXML string, data fields.MergeData, processedFields map[string]bool) (string, []string) {
	var skipped []string

	// Simple regex to find <w:t>«fieldname»</w:t> patterns
	fieldRegex := regexp.MustCompile(`<w:t[^>]*>«([^»]+)»</w:t>`)

	// Count matches
	matches := fieldRegex.FindAllStringSubmatch(documentXML, -1)
	logging.Debug("Detected %d field placeholders in document", len(matches))

	// Replace each match
	result := fieldRegex.ReplaceAllStringFunc(documentXML, func(match string) string {
		// Extract field name from the match
		fieldNameRegex := regexp.MustCompile(`«([^»]+)»`)
		fieldNameMatch := fieldNameRegex.FindStringSubmatch(match)
		if len(fieldNameMatch) < 2 {
			return match // Return original if we can't extract field name
		}

		fieldName := strings.TrimSpace(fieldNameMatch[1])

		// Skip if already processed
		if processedFields[fieldName] {
			return match
		}
		processedFields[fieldName] = true

		// Try to get the value from merge data (case-insensitive)
		value, found := getCaseInsensitiveValue(data, fieldName)
		if found {
			logging.Debug("Field replacement: '%s' -> '%s'", fieldName, value)
			// Replace the content inside <w:t> with the value
			replacement := strings.Replace(match, "«"+fieldName+"»", escapeXML(value), 1)
			return replacement
		}

		// Field not found in data, add to skipped list
		if !contains(skipped, fieldName) {
			logging.Debug("Field skipped: '%s' (no data available)", fieldName)
			skipped = append(skipped, fieldName)
		}

		// Return the original field
		return match
	})

	return result, skipped
}

// getCaseInsensitiveValue performs case-insensitive lookup in merge data
func getCaseInsensitiveValue(data fields.MergeData, fieldName string) (string, bool) {
	// Try exact match first
	if value, exists := data[fieldName]; exists {
		return fmt.Sprintf("%v", value), true
	}

	// Try case-insensitive match
	for key, value := range data {
		if strings.EqualFold(key, fieldName) {
			return fmt.Sprintf("%v", value), true
		}
	}

	return "", false
}

// escapeXML escapes special XML characters in text content
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// rebuildDocxArchive rebuilds the DOCX file as a ZIP archive
func rebuildDocxArchive(doc *docx.DocxFile) ([]byte, error) {
	var buf bytes.Buffer

	logging.Debug("Creating ZIP writer for DOCX archive")
	// Create a new ZIP writer
	zipWriter := zip.NewWriter(&buf)

	logging.Debug("Adding %d files to ZIP archive", len(doc.Files))
	// Add all files to the ZIP archive
	fileIndex := 0
	for filename, content := range doc.Files {
		fileIndex++
		logging.Debug("Adding file %d/%d: %s (%d bytes)", fileIndex, len(doc.Files), filename, len(content))

		// Create a file in the ZIP archive
		fileWriter, err := zipWriter.Create(filename)
		if err != nil {
			logging.Error("Failed to create file %s in ZIP: %v", filename, err)
			zipWriter.Close()
			return nil, fmt.Errorf("failed to create file %s in ZIP: %w", filename, err)
		}

		// Write the file content
		_, err = fileWriter.Write(content)
		if err != nil {
			logging.Error("Failed to write content for file %s: %v", filename, err)
			zipWriter.Close()
			return nil, fmt.Errorf("failed to write content for file %s: %w", filename, err)
		}
		logging.Debug("Successfully added file: %s", filename)
	}

	logging.Debug("Closing ZIP writer")
	// Close the ZIP writer
	err := zipWriter.Close()
	if err != nil {
		logging.Error("Failed to close ZIP writer: %v", err)
		return nil, fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	logging.Debug("ZIP archive successfully created (%d bytes)", buf.Len())
	return buf.Bytes(), nil
}


// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
