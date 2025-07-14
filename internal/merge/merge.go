package merge

import (
	"archive/zip"
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"

	"com/lifenture/flash-mail-merge/internal/docx"
	"com/lifenture/flash-mail-merge/internal/fields"
)

// PerformMerge performs mail merge on a DOCX document with the provided data
func PerformMerge(doc *docx.DocxFile, data fields.MergeData) (mergedDoc []byte, skipped []string, err error) {
	log.Printf("Starting mail merge with %d available data fields", len(data))
	
	// Get the document XML content
	documentXML, err := doc.GetDocumentXML()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get document XML: %w", err)
	}
	log.Printf("Retrieved document XML content (%d bytes)", len(documentXML))

	// Replace field values in the document XML
	updatedXML, skippedFields, err := replaceFieldValues(string(documentXML), data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to replace field values: %w", err)
	}
	log.Printf("Field replacement completed - processed fields with %d skipped", len(skippedFields))
	if len(skippedFields) > 0 {
		log.Printf("Skipped fields: %v", skippedFields)
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
	log.Printf("Copied %d files from original document to updated document", fileCount)
	
	// Replace the document XML with the updated version
	updatedDoc.Files["word/document.xml"] = []byte(updatedXML)
	log.Printf("Updated document XML content (%d bytes)", len(updatedXML))

	// Rebuild the DOCX (ZIP) archive
	log.Printf("Starting ZIP archive rebuild")
	mergedBytes, err := rebuildDocxArchive(updatedDoc)
	if err != nil {
		log.Printf("ZIP rebuild failed: %v", err)
		return nil, nil, fmt.Errorf("failed to rebuild DOCX archive: %w", err)
	}
	log.Printf("ZIP rebuild successful - generated %d bytes", len(mergedBytes))

	return mergedBytes, skippedFields, nil
}

// replaceFieldValues replaces merge fields in the XML with actual values
func replaceFieldValues(documentXML string, data fields.MergeData) (string, []string, error) {
	var skipped []string
	processedFields := make(map[string]bool)

	// Process the XML to replace merge fields
	result := documentXML

	// 1. Find and replace simple merge fields
	log.Printf("Processing simple merge fields")
	result, simpleSkipped := replaceSimpleFields(result, data, processedFields)
	skipped = append(skipped, simpleSkipped...)
	log.Printf("Simple field processing completed: %d fields skipped", len(simpleSkipped))

	// 2. Find and replace complex merge fields
	log.Printf("Processing complex merge fields")
	result, complexSkipped := replaceComplexFields(result, data, processedFields)
	skipped = append(skipped, complexSkipped...)
	log.Printf("Complex field processing completed: %d fields skipped", len(complexSkipped))

	log.Printf("Total fields processed: %d, Total fields skipped: %d", len(processedFields), len(skipped))
	return result, skipped, nil
}

// replaceSimpleFields handles simple merge fields: <w:fldSimple w:instr="MERGEFIELD fieldname">
func replaceSimpleFields(documentXML string, data fields.MergeData, processedFields map[string]bool) (string, []string) {
	var skipped []string

	// Regular expression to find simple merge fields
	simpleFieldRegex := regexp.MustCompile(`(?is)<w:fldSimple[^>]*w:instr="MERGEFIELD\s+([^\s"]+)"[^>]*>.*?</w:fldSimple>`)
	
	// Count simple merge fields detected
	simpleFieldMatches := simpleFieldRegex.FindAllString(documentXML, -1)
	log.Printf("Detected %d simple merge fields in document", len(simpleFieldMatches))

	// Find all simple merge fields
	result := simpleFieldRegex.ReplaceAllStringFunc(documentXML, func(match string) string {
		// Extract field name from the match
		fieldNameRegex := regexp.MustCompile(`(?i)w:instr="MERGEFIELD\s+([^\s"]+)"`)
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
			log.Printf("Simple field replacement: '%s' -> '%s'", fieldName, value)
			// Extract and collapse multiple runs if they exist, preserving the first run's structure
			replacementRun := createReplacementRun(match, value)
			return replacementRun
		}

		// Field not found in data, add to skipped list
		if !contains(skipped, fieldName) {
			log.Printf("Simple field skipped: '%s' (no data available)", fieldName)
			skipped = append(skipped, fieldName)
		}

		// Return the original field
		return match
	})

	return result, skipped
}

// replaceComplexFields handles complex merge fields with <w:instrText>MERGEFIELD fieldname</w:instrText>
func replaceComplexFields(documentXML string, data fields.MergeData, processedFields map[string]bool) (string, []string) {
	var skipped []string

	// Find complex field structures
	// Pattern: <w:fldChar w:fldCharType="begin"/> ... <w:instrText>MERGEFIELD fieldname</w:instrText> ... <w:fldChar w:fldCharType="end"/>
	complexFieldRegex := regexp.MustCompile(`(?s)<w:fldChar[^>]*w:fldCharType="begin"[^>]*/?>.*?<w:instrText[^>]*>.*?MERGEFIELD\s+([^\s<]+).*?</w:instrText>.*?<w:fldChar[^>]*w:fldCharType="end"[^>]*/>`)
	
	// Count complex merge fields detected
	complexFieldMatches := complexFieldRegex.FindAllString(documentXML, -1)
	log.Printf("Detected %d complex merge fields in document", len(complexFieldMatches))

	result := complexFieldRegex.ReplaceAllStringFunc(documentXML, func(match string) string {
		// Extract field name from the instrText
		fieldNameRegex := regexp.MustCompile(`(?i)MERGEFIELD\s+([^\s<]+)`)
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
			log.Printf("Complex field replacement: '%s' -> '%s'", fieldName, value)
			// Find the display text area (between separate and end field chars)
			// Extract runs between separate and end field chars
			separateRegex := regexp.MustCompile(`(?s)<w:fldChar[^>]*w:fldCharType="separate"[^>]*/?>.*?<w:fldChar[^>]*w:fldCharType="end"[^>]*/>`)
			separateMatch := separateRegex.FindString(match)

			if separateMatch != "" {
				// Extract formatting from the separate section
				formattingRegex := regexp.MustCompile(`(?s)<w:rPr>.*?</w:rPr>`)
				formattingMatch := formattingRegex.FindString(separateMatch)
				
				if formattingMatch != "" {
					// Create replacement run with formatting
					replacementRun := fmt.Sprintf(`<w:r>%s<w:t>%s</w:t></w:r>`, formattingMatch, escapeXML(value))
					return strings.Replace(match, separateMatch, replacementRun, 1)
				}
				
				// If no formatting found, create a simple replacement
				replacementRun := fmt.Sprintf(`<w:r><w:t>%s</w:t></w:r>`, escapeXML(value))
				return strings.Replace(match, separateMatch, replacementRun, 1)
			}

			// Fallback: create a simple run
			return fmt.Sprintf(`<w:r><w:t>%s</w:t></w:r>`, escapeXML(value))
		}

		// Field not found in data, add to skipped list
		if !contains(skipped, fieldName) {
			log.Printf("Complex field skipped: '%s' (no data available)", fieldName)
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
	
	log.Printf("Creating ZIP writer for DOCX archive")
	// Create a new ZIP writer
	zipWriter := zip.NewWriter(&buf)
	
	log.Printf("Adding %d files to ZIP archive", len(doc.Files))
	// Add all files to the ZIP archive
	fileIndex := 0
	for filename, content := range doc.Files {
		fileIndex++
		log.Printf("Adding file %d/%d: %s (%d bytes)", fileIndex, len(doc.Files), filename, len(content))
		
		// Create a file in the ZIP archive
		fileWriter, err := zipWriter.Create(filename)
		if err != nil {
			log.Printf("Failed to create file %s in ZIP: %v", filename, err)
			zipWriter.Close()
			return nil, fmt.Errorf("failed to create file %s in ZIP: %w", filename, err)
		}
		
		// Write the file content
		_, err = fileWriter.Write(content)
		if err != nil {
			log.Printf("Failed to write content for file %s: %v", filename, err)
			zipWriter.Close()
			return nil, fmt.Errorf("failed to write content for file %s: %w", filename, err)
		}
		log.Printf("Successfully added file: %s", filename)
	}
	
	log.Printf("Closing ZIP writer")
	// Close the ZIP writer
	err := zipWriter.Close()
	if err != nil {
		log.Printf("Failed to close ZIP writer: %v", err)
		return nil, fmt.Errorf("failed to close ZIP writer: %w", err)
	}
	
	log.Printf("ZIP archive successfully created (%d bytes)", buf.Len())
	return buf.Bytes(), nil
}

// createReplacementRun creates a replacement run preserving the original formatting
// If the field spans multiple runs, it collapses them into the first run
func createReplacementRun(xmlContent string, value string) string {
	// Find all runs within the content
	runRegex := regexp.MustCompile(`<w:r[^>]*>.*?</w:r>`)
	runMatches := runRegex.FindAllString(xmlContent, -1)

	if len(runMatches) == 0 {
		// No runs found, create a simple run
		return fmt.Sprintf(`<w:r><w:t>%s</w:t></w:r>`, escapeXML(value))
	}

	// Use the first run as the template to preserve formatting
	firstRun := runMatches[0]

	// Extract the run properties (formatting) from the first run
	// This includes font, color, size, etc.
	runPropsRegex := regexp.MustCompile(`<w:r[^>]*>(.*?)<w:t[^>]*>.*?</w:t>(.*?)</w:r>`)
	runPropsMatch := runPropsRegex.FindStringSubmatch(firstRun)

	if len(runPropsMatch) >= 3 {
		// Reconstruct the run with preserved formatting but new text
		beforeText := runPropsMatch[1] // Run properties (w:rPr, etc.)
		afterText := runPropsMatch[2]  // Any content after w:t
		
		// Extract run attributes from the opening tag
		runAttrsRegex := regexp.MustCompile(`<w:r([^>]*)>`)
		runAttrsMatch := runAttrsRegex.FindStringSubmatch(firstRun)
		runAttrs := ""
		if len(runAttrsMatch) >= 2 {
			runAttrs = runAttrsMatch[1]
		}

		// Create the new run with preserved structure
		newRun := fmt.Sprintf(`<w:r%s>%s<w:t>%s</w:t>%s</w:r>`, 
			runAttrs, beforeText, escapeXML(value), afterText)
		return newRun
	}

	// Fallback: replace just the text content in the first run
	textRegex := regexp.MustCompile(`<w:t[^>]*>.*?</w:t>`)
	newRun := textRegex.ReplaceAllString(firstRun, fmt.Sprintf(`<w:t>%s</w:t>`, escapeXML(value)))
	return newRun
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
