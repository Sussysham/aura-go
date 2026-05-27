package reader

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"aura-go/internal/catalog"
)

func ExtractBookRawText(b catalog.Book) (string, error) {
	cleanPath := filepath.Clean(b.FullPath)
	ext := strings.ToLower(filepath.Ext(cleanPath))

	if ext == ".txt" {
		content, err := os.ReadFile(cleanPath)
		if err != nil { return "", err }
		return string(content), nil
	} else if ext == ".epub" {
		r, err := zip.OpenReader(cleanPath)
		if err != nil { return "", err }
		defer r.Close()

		orderedPaths, err := GetEPUBOrderedPaths(r)
		if err != nil {
			// Fallback: read all html/xhtml files in zip order (current behavior)
			var textBuilder strings.Builder
			for _, f := range r.File {
				fExt := strings.ToLower(filepath.Ext(f.Name))
				if fExt == ".xhtml" || fExt == ".html" || fExt == ".htm" {
					rc, err := f.Open()
					if err != nil { continue }
					
					content, err := io.ReadAll(rc)
					rc.Close()
					if err != nil { continue }

					cleanHTML := StripHTMLTags(string(content))
					textBuilder.WriteString(cleanHTML)
					textBuilder.WriteString("\n\n")
				}
			}
			return textBuilder.String(), nil
		}

		// Create a map of normalized zip file names to zip file pointers
		fileMap := make(map[string]*zip.File)
		for _, f := range r.File {
			normName := filepath.ToSlash(strings.ToLower(f.Name))
			fileMap[normName] = f
		}

		var textBuilder strings.Builder
		for _, path := range orderedPaths {
			normPath := filepath.ToSlash(strings.ToLower(path))
			if f, exists := fileMap[normPath]; exists {
				rc, err := f.Open()
				if err != nil { continue }
				
				content, err := io.ReadAll(rc)
				rc.Close()
				if err != nil { continue }

				cleanHTML := StripHTMLTags(string(content))
				textBuilder.WriteString(cleanHTML)
				textBuilder.WriteString("\n\n")
			}
		}
		return textBuilder.String(), nil
	}
	return "", fmt.Errorf("format %s not supported inside console (Use SumatraPDF instead)", b.Format)
}
