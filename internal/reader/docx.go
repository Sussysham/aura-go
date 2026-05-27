package reader

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

func ExtractDOCXText(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var docFile *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}
	if docFile == nil {
		return "", fmt.Errorf("invalid docx file: word/document.xml not found")
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var builder strings.Builder
	var inText bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if se.Name.Local == "t" {
				inText = false
			} else if se.Name.Local == "p" {
				builder.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				builder.Write(se)
			}
		}
	}

	return builder.String(), nil
}
