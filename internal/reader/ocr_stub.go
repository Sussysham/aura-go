//go:build !ocr

package reader

import "fmt"

func ExtractOCRText(path string) (string, error) {
	return "", fmt.Errorf("OCR is not compiled into this binary. Rebuild with: go build -tags ocr")
}
