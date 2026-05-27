package util

import (
	"crypto/sha1"
	"fmt"
	"os"
)

func ComputeQuickSHA1(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	h := sha1.New()
	buffer := make([]byte, 32*1024)
	n, _ := file.Read(buffer)
	h.Write(buffer[:n])
	return fmt.Sprintf("%x", h.Sum(nil))
}
