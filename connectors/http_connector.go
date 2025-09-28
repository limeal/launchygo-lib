package connectors

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"limeal.fr/launchygo/utils"
)

const HTTP_SCHEME = "http"
const HTTPS_SCHEME = "https"

type HttpConnector struct {
	URL string

	Secured bool // https or http
}

func (c *HttpConnector) getURL(remotePath string) string {
	if strings.HasPrefix(remotePath, "/") {
		if strings.HasSuffix(c.URL, "/") {
			return c.URL + strings.TrimPrefix(remotePath, "/")
		}
		return c.URL + remotePath
	}

	return c.URL + "/" + remotePath
}

func (c *HttpConnector) NewFromURI(uri string) Connector {
	return &HttpConnector{
		URL:     uri,
		Secured: strings.HasPrefix(uri, HTTPS_SCHEME),
	}
}

func (c *HttpConnector) GetPath() string {
	return c.URL
}

func (c *HttpConnector) GetURI() string {
	return c.URL
}

func (c *HttpConnector) Connect() error {
	return nil
}

func (c *HttpConnector) Login() error {
	return nil
}

/**
* Read the file from remote url
* e.g. https://launcher.limeal.fr/manifest.json
 */
func (c *HttpConnector) ReadFile(remotePath string, dest any) error {
	// Is like a get request to c.Path + remotePath
	url := c.getURL(remotePath)
	bytes, err := utils.DoRequest[[]byte]("GET", url, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dest)
}

func (c *HttpConnector) ReadFileBytes(remotePath string, size int64) ([]byte, error) {
	url := c.getURL(remotePath)
	return utils.DoRequest[[]byte]("GET", url, nil)
}

func (c *HttpConnector) SendFile(remotePath string, localPath string) error {
	return fmt.Errorf("Http connector does not support SendFile")
}

func (c *HttpConnector) SendFileFromBytes(remotePath string, bytes []byte, perm ...fs.FileMode) error {
	return fmt.Errorf("Http connector does not support SendFileFromBytes")
}

func (c *HttpConnector) GetScheme() string {
	return HTTP_SCHEME
}

func (c *HttpConnector) IsConnected() bool {
	return true
}

func (c *HttpConnector) Close() error {
	return nil
}

func (c *HttpConnector) HasFile(remotePath string) bool {
	url := c.getURL(remotePath)

	_, err := utils.DoRequest[[]byte]("HEAD", url, nil)
	return err == nil
}

func (c *HttpConnector) HasFileWithChecksum(remotePath string, checksumType ChecksumType, checksum string) bool {
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
