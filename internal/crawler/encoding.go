// Package crawler provides data crawling services for fund information.
package crawler

import (
	"bytes"
	"io"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// GBKToUTF8 converts GBK encoded bytes to UTF-8.
func GBKToUTF8(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	return io.ReadAll(reader)
}

// UTF8ToGBK converts UTF-8 encoded bytes to GBK.
func UTF8ToGBK(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewEncoder())
	return io.ReadAll(reader)
}
