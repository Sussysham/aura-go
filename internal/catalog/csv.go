package catalog

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

func LoadBooks(path string) ([]Book, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}

	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	var books []Book
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		sizeMB, _ := strconv.ParseFloat(record[headerMap["SizeMB"]], 64)
		
		tagsVal := ""
		if idx, exists := headerMap["Tags"]; exists {
			tagsVal = record[idx]
		}

		books = append(books, Book{
			Title:             record[headerMap["Title"]],
			Author:            record[headerMap["Author"]],
			Format:            record[headerMap["Format"]],
			SizeMB:            sizeMB,
			LocationCategory:  record[headerMap["LocationCategory"]],
			DuplicateGroup:    record[headerMap["DuplicateGroup"]],
			FilenameGibberish: record[headerMap["FilenameGibberish"]],
			SHA1:              record[headerMap["SHA1"]],
			FileName:          record[headerMap["FileName"]],
			FullPath:          record[headerMap["FullPath"]],
			Modified:          record[headerMap["Modified"]],
			Tags:              tagsVal,
		})
	}
	return books, nil
}

func AppendBooksToCatalog(path string, newBooks []Book) error {
	existingMap := make(map[string]bool)
	var allBooks []Book
	
	file, err := os.Open(path)
	if err == nil {
		file.Close()
		loaded, err := LoadBooks(path)
		if err == nil {
			allBooks = loaded
			for _, b := range loaded {
				existingMap[b.SHA1] = true
			}
		}
	}

	addedCount := 0
	for _, nb := range newBooks {
		if !existingMap[nb.SHA1] {
			allBooks = append(allBooks, nb)
			existingMap[nb.SHA1] = true
			addedCount++
		}
	}

	if addedCount == 0 && len(allBooks) > 0 {
		return nil
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	header := []string{"Title", "Author", "Format", "SizeMB", "LocationCategory", "DuplicateGroup", "FilenameGibberish", "SHA1", "FileName", "FullPath", "Modified", "Tags"}
	_ = writer.Write(header)

	for _, b := range allBooks {
		record := []string{
			b.Title,
			b.Author,
			b.Format,
			fmt.Sprintf("%.2f", b.SizeMB),
			b.LocationCategory,
			b.DuplicateGroup,
			b.FilenameGibberish,
			b.SHA1,
			b.FileName,
			b.FullPath,
			b.Modified,
			b.Tags,
		}
		_ = writer.Write(record)
	}
	return nil
}
