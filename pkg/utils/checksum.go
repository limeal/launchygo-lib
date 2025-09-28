package utils

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func BytesSHA1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func BytesSHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func FileSHA1(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return BytesSHA1(data)
}

func FileSHA256(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return BytesSHA256(data)
}

func ReaderSHA1(rc io.ReadCloser) (string, error) {
	h := sha1.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ReaderSHA256(rc io.ReadCloser) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
