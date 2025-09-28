package connectors

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"limeal.fr/launchygo/utils"
)

const SFTP_SCHEME = "sftp"

type SFTPConnector struct {
	Host     string
	Port     int
	BasePath string
	Username string
	Password string

	client       *sftp.Client
	clientConfig *ssh.ClientConfig

	// Connection pool for parallel operations
	pool      []*sftp.Client
	poolMutex sync.RWMutex
	poolIndex int
	poolSize  int
}

func (c *SFTPConnector) NewFromURI(uri string) Connector {
	// Example: sftp://user:password@host:port/base_path
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil
	}

	host := parsed.Hostname()
	portStr := parsed.Port()
	basePath := parsed.Path
	port := 22 // default SFTP port
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	username := ""
	password := ""
	if parsed.User != nil {
		username = parsed.User.Username()
		pw, set := parsed.User.Password()
		if set {
			password = pw
		}
	}

	return &SFTPConnector{
		Host:     host,
		Port:     port,
		BasePath: basePath,
		Username: username,
		Password: password,
		clientConfig: &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
		poolSize: 4, // Default pool size
	}
}

func (c *SFTPConnector) GetPath() string {
	return c.BasePath
}

func (c *SFTPConnector) GetURI() string {
	if c.Username != "" {
		if c.Password != "" {
			return SFTP_SCHEME + "://" + url.QueryEscape(c.Username) + ":" + "*****" + "@" + c.Host + ":" + strconv.Itoa(c.Port) + "/"
		}
		return SFTP_SCHEME + "://" + url.QueryEscape(c.Username) + "@" + c.Host + ":" + strconv.Itoa(c.Port) + "/"
	}
	return SFTP_SCHEME + "://" + c.Host + ":" + strconv.Itoa(c.Port) + "/"
}

/**
* Internal function to format the path
* If the path starts with a /, it returns the path as is
* Otherwise, it returns the path prefixed with a /
 */
func (c *SFTPConnector) formatPath(path string) string {
	cleanPath := path
	if c.BasePath != "" {
		cleanPath = strings.TrimLeft(path, "/")
		cleanPath = c.BasePath + "/" + cleanPath
	}
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}
	return cleanPath
}

func (c *SFTPConnector) Connect() error {
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), c.clientConfig)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	c.client, err = sftp.NewClient(conn,
		sftp.UseConcurrentWrites(true),
		sftp.UseConcurrentReads(true),
		sftp.MaxPacket(1<<15), // 32KiB is a solid default
	)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}

	// Initialize connection pool
	return c.initPool()
}

// initPool creates a pool of SFTP connections for parallel operations
func (c *SFTPConnector) initPool() error {
	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()

	c.pool = make([]*sftp.Client, c.poolSize)

	for i := 0; i < c.poolSize; i++ {
		conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), c.clientConfig)
		if err != nil {
			// Clean up any connections we've already created
			c.closePool()
			return fmt.Errorf("failed to dial pool connection %d: %w", i, err)
		}

		client, err := sftp.NewClient(conn,
			sftp.UseConcurrentWrites(true),
			sftp.UseConcurrentReads(true),
			sftp.MaxPacket(1<<15),
		)
		if err != nil {
			conn.Close()
			c.closePool()
			return fmt.Errorf("failed to create SFTP pool client %d: %w", i, err)
		}

		c.pool[i] = client
	}

	return nil
}

// getPoolClient returns a client from the pool using round-robin
func (c *SFTPConnector) getPoolClient() *sftp.Client {
	c.poolMutex.RLock()
	defer c.poolMutex.RUnlock()

	if len(c.pool) == 0 {
		return c.client // Fallback to main client
	}

	client := c.pool[c.poolIndex]
	c.poolIndex = (c.poolIndex + 1) % len(c.pool)
	return client
}

// closePool closes all connections in the pool
func (c *SFTPConnector) closePool() {
	for i, client := range c.pool {
		if client != nil {
			client.Close()
			c.pool[i] = nil
		}
	}
	c.pool = nil
}

func (c *SFTPConnector) Login() error {
	return nil
}

/**
* Read the file from the remote path and unmarshals it into the destination
* Notice: file must contains a json object
*
* Example:
* ```
* type Manifest struct {
*     Version string `json:"version"`
* }
* var manifest Manifest
* err := connector.ReadFile("manifest.json", &manifest)
* ```
 */
func (c *SFTPConnector) ReadFile(remotePath string, dest any) error {
	bytes, err := c.ReadFileBytes(remotePath, -1)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dest)
}

func (c *SFTPConnector) ReadFileBytes(remotePath string, size int64) ([]byte, error) {
	remotePath = c.formatPath(remotePath)

	// Use pool client for better concurrency
	client := c.getPoolClient()

	f, err := client.Open(remotePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if size <= 0 {
		st, err := f.Stat()
		if err != nil {
			return nil, err
		}
		size = st.Size()
	}

	buf := make([]byte, size)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *SFTPConnector) SendFile(remotePath string, localPath string) error {
	bytes, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	return c.SendFileFromBytes(remotePath, bytes)
}

func (c *SFTPConnector) SendFileFromBytes(remotePath string, buf []byte, perm ...fs.FileMode) error {
	remotePath = c.formatPath(remotePath)

	dir := path.Dir(remotePath) // POSIX paths over SFTP
	if err := c.client.MkdirAll(dir); err != nil {
		return fmt.Errorf("mkdirAll %s: %w", dir, err)
	}

	tmp := remotePath + ".part"
	// Remove any stale temp file
	_ = c.client.Remove(tmp)

	rf, err := c.client.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("open remote tmp: %w", err)
	}

	// Optional preallocate (server may ignore)
	if len(buf) > 0 {
		_ = rf.Truncate(int64(len(buf)))
	}

	if _, err := rf.Write(buf); err != nil {
		_ = rf.Close()
		_ = c.client.Remove(tmp)
		return fmt.Errorf("write: %w", err)
	}

	// Ensure all data is flushed to the server before rename
	if err := rf.Close(); err != nil {
		_ = c.client.Remove(tmp)
		return fmt.Errorf("close remote tmp: %w", err)
	}

	// Atomic promote
	if err := c.client.PosixRename(tmp, remotePath); err != nil {
		// Fallback path if POSIX rename isn’t supported
		_ = c.client.Remove(remotePath)
		if err2 := c.client.Rename(tmp, remotePath); err2 != nil {
			_ = c.client.Remove(tmp)
			return fmt.Errorf("rename: %w (fallback failed: %v)", err, err2)
		}
	}

	// Set the mode
	if len(perm) > 0 {
		return c.client.Chmod(remotePath, perm[0])
	}

	return nil
}

/**
* List all files and directories in the given path
* Example:
* ```
* files, err := connector.List("path/subpath")
* ```
 */
func (c *SFTPConnector) List(path string) ([]os.FileInfo, error) {
	// List all files and directories in the given path
	files, err := c.client.ReadDir(c.formatPath(path))
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return files, nil
}

func (c *SFTPConnector) GetScheme() string {
	return SFTP_SCHEME
}

func (c *SFTPConnector) IsConnected() bool {
	return c.client != nil
}

func (c *SFTPConnector) Close() error {
	// Close the main client
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}

	// Close all pool connections
	c.poolMutex.Lock()
	c.closePool()
	c.poolMutex.Unlock()

	return nil
}

// SetPoolSize sets the number of connections in the pool
func (c *SFTPConnector) SetPoolSize(size int) {
	c.poolMutex.Lock()
	defer c.poolMutex.Unlock()
	c.poolSize = size
}

func (c *SFTPConnector) HasFile(remotePath string) bool {
	// Check if the file exists using pool client
	client := c.getPoolClient()
	_, err := client.Stat(c.formatPath(remotePath))
	return err == nil
}

func (c *SFTPConnector) HasFileWithChecksum(remotePath string, checksumType ChecksumType, checksum string) bool {
	bytes, err := c.ReadFileBytes(remotePath, -1)
	if err != nil {
		return false
	}

	// Check if the checksum is equal to the file checksum
	switch checksumType {
	case ChecksumTypeSHA1:
		sha1 := utils.BytesSHA1(bytes)

		return sha1 == checksum
	case ChecksumTypeSHA256:
		sha256 := utils.BytesSHA256(bytes)
		return sha256 == checksum
	}
	return err == nil
}
