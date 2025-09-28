package connectors

import (
	"io/fs"
	"strings"
)

type ChecksumType int

const (
	ChecksumTypeSHA1 ChecksumType = iota + 1
	ChecksumTypeSHA256
)

type Connector interface {
	NewFromURI(uri string) Connector

	GetPath() string
	GetURI() string

	Connect() error
	Login() error
	// ReadFile reads the file from the remote path and unmarshals it into the destination
	ReadFile(remotePath string, dest any) error
	ReadFileBytes(remotePath string, size int64) ([]byte, error)

	SendFile(remotePath string, localPath string) error
	SendFileFromBytes(remotePath string, bytes []byte, perm ...fs.FileMode) error

	GetScheme() string // ftp, sftp, etc.
	IsConnected() bool
	Close() error

	HasFile(remotePath string) bool
	HasFileWithChecksum(remotePath string, checksumType ChecksumType, checksum string) bool
}

var CONNECTORS = map[string]Connector{
	"sftp":  new(SFTPConnector),
	"file":  new(FileConnector),
	"http":  new(HttpConnector),
	"https": new(HttpConnector),
}

func FindConnectorFromURI(uri string) Connector {
	for k, connector := range CONNECTORS {
		if strings.HasPrefix(uri, k+"://") {
			return connector.NewFromURI(uri)
		}
	}

	return nil
}
