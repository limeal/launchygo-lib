package connectors

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"limeal.fr/launchygo/utils"
)

const FILE_SCHEME = "file"

type FileConnector struct {
	Path string
}

func (c *FileConnector) NewFromURI(uri string) Connector {
	// Example: file:///path/to/file
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil
	}

	pwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	// if the path start with ./ use PWD
	finalPath := parsed.Host + parsed.Path
	if strings.HasPrefix(finalPath, "./") {
		finalPath = filepath.Join(pwd, strings.TrimPrefix(finalPath, "./"))
	}

	return &FileConnector{
		Path: finalPath,
	}
}

func (c *FileConnector) GetPath() string {
	return c.Path
}

func (c *FileConnector) GetURI() string {
	return FILE_SCHEME + "://" + c.Path
}

func (c *FileConnector) Connect() error {
	return nil
}

func (c *FileConnector) Login() error {
	return nil
}

func (c *FileConnector) ReadFile(remotePath string, dest any) error {
	// Read the file from the remote path and unmarshals it into the destination
	bytes, err := c.ReadFileBytes(remotePath, -1)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	return json.Unmarshal(bytes, dest)
}

func (c *FileConnector) ReadFileBytes(remotePath string, size int64) ([]byte, error) {
	remotePath = filepath.Join(c.Path, remotePath)

	f, err := os.Open(remotePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// If we know the size, preallocate
	if size > 0 {
		buf := make([]byte, size)
		_, err := io.ReadFull(f, buf)
		if err != nil {
			return nil, err
		}
		return buf, nil
	}

	// Fallback for unknown size
	return io.ReadAll(f)
}

func (c *FileConnector) SendFile(remotePath string, localPath string) error {
	return utils.CopyFile(localPath, filepath.Join(c.Path, remotePath))
}

func (c *FileConnector) SendFileFromBytes(remotePath string, bytes []byte, perm ...fs.FileMode) error {
	fullPath := filepath.Join(c.Path, remotePath)

	// Check if file exists, if so, remove it to ensure overwrite
	if _, err := os.Stat(fullPath); err == nil {
		// File exists, remove it
		if err := os.Remove(fullPath); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	err := os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	mode := 0644
	if len(perm) > 0 {
		mode = int(perm[0])
	}

	return os.WriteFile(filepath.Join(c.Path, remotePath), bytes, fs.FileMode(mode))
}

func (c *FileConnector) GetScheme() string {
	return FILE_SCHEME
}

func (c *FileConnector) IsConnected() bool {
	return true
}

func (c *FileConnector) Close() error {
	return nil
}

func (c *FileConnector) HasFile(remotePath string) bool {
	remotePath = filepath.Join(c.Path, remotePath)
	_, err := os.Stat(remotePath)
	return err == nil
}

func (c *FileConnector) HasFileWithChecksum(remotePath string, checksumType ChecksumType, checksum string) bool {
	bytes, err := c.ReadFileBytes(remotePath, -1)
	if err != nil {
		return false
	}

	switch checksumType {
	case ChecksumTypeSHA1:
		sha1 := utils.BytesSHA1(bytes)
		return sha1 == checksum
	case ChecksumTypeSHA256:
		sha256 := utils.BytesSHA256(bytes)
		return sha256 == checksum
	}
	return true
}
