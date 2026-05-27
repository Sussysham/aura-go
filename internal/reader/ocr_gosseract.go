//go:build ocr

package reader

import "github.com/otiai10/gosseract/v2"

func ExtractOCRText(path string) (string, error) {
	client := gosseract.NewClient()
	defer client.Close()
	err := client.SetImage(path)
	if err != nil {
		return "", err
	}
	return client.Text()
}
