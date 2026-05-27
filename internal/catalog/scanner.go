package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"aura-go/internal/util"
)

func ScanDirectoryForBooks(dir string) ([]Book, error) {
	var books []Book
	
	cleanDir := filepath.Clean(dir)
	if _, err := os.Stat(cleanDir); err != nil {
		return nil, err
	}

	hashTracker := make(map[string][]int)

	err := filepath.Walk(cleanDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".pdf" || ext == ".epub" || ext == ".txt" {
			sizeMB := float64(info.Size()) / (1024.0 * 1024.0)
			
			shaVal := util.ComputeQuickSHA1(path)
			if shaVal == "" {
				return nil
			}

			fileName := info.Name()
			format := strings.TrimPrefix(ext, ".")
			
			title := strings.TrimSuffix(fileName, ext)
			author := "Unknown"
			
			if strings.Contains(title, " - ") {
				parts := strings.SplitN(title, " - ", 2)
				title = strings.TrimSpace(parts[0])
				author = strings.TrimSpace(parts[1])
			}

			category := filepath.Base(filepath.Dir(path))
			if category == "." || category == filepath.Base(cleanDir) {
				category = "Library"
			}

			gibberish := "N"
			if matched, _ := regexp.MatchString(`^[a-fA-F0-9\-]{25,}`, title); matched || len(title) > 20 && !strings.Contains(title, " ") {
				gibberish = "Y"
			}

			books = append(books, Book{
				Title:             title,
				Author:            author,
				Format:            format,
				SizeMB:            sizeMB,
				LocationCategory:  category,
				DuplicateGroup:    "",
				FilenameGibberish: gibberish,
				SHA1:              shaVal,
				FileName:          fileName,
				FullPath:          path,
				Modified:          info.ModTime().Format("2006-01-02"),
			})
			
			hashTracker[shaVal] = append(hashTracker[shaVal], len(books)-1)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	groupCounter := 1
	for _, indices := range hashTracker {
		if len(indices) > 1 {
			groupTag := fmt.Sprintf("G%02d", groupCounter)
			for _, idx := range indices {
				books[idx].DuplicateGroup = groupTag
			}
			groupCounter++
		}
	}

	return books, nil
}
