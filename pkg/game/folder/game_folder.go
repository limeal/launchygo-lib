package folder

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"limeal.fr/launchygo/pkg/connectors"
	"limeal.fr/launchygo/pkg/game/authenticator"
	"limeal.fr/launchygo/pkg/game/folder/rules"
	"limeal.fr/launchygo/pkg/game/folder/shared"
	"limeal.fr/launchygo/pkg/utils"
)

type GameFolder struct {
	Path      string
	Manifest  Manifest
	Connector connectors.Connector

	KeepFiles []string // Files to keep in the game folder, even if they are not in the manifest, must be dynamic
}

func GetGameFolderPathForFolder(folderName string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", folderName), nil
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), folderName), nil
	case "linux":
		return filepath.Join(os.Getenv("HOME"), "."+folderName), nil
	}
	return "", fmt.Errorf("unsupported OS")
}

func InitGameFolder(connector connectors.Connector, folderName string, testPackModeArgs ...bool) (*GameFolder, error) {
	testPackMode := false
	if len(testPackModeArgs) > 0 {
		testPackMode = testPackModeArgs[0]
	}

	var path string
	if testPackMode == false {
		var err error
		path, err = GetGameFolderPathForFolder(folderName)
		if err != nil {
			return nil, fmt.Errorf("❌ Failed to get game folder path: %w", err)
		}
	} else {
		pwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("❌ Failed to get pwd: %w", err)
		}
		path = filepath.Join(pwd, "packs", folderName)
	}

	var manifest Manifest
	err := connector.ReadFile(shared.MANIFEST_FILE, &manifest)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to read manifest at location %s: %w", shared.MANIFEST_FILE, err)
	}

	keepFiles := []string{"options.txt", "logs/*", "resourcepacks/*"}

	return &GameFolder{
		Path:      path,
		Manifest:  manifest,
		Connector: connector,
		KeepFiles: keepFiles,
	}, nil
}

func (d *GameFolder) GetPath() string {
	return d.Path
}

func (g *GameFolder) GetMCVersion() string {
	return g.Manifest.McVersion
}

func (g *GameFolder) GetVersion() string {
	return g.Manifest.Version
}

func (g *GameFolder) GetDirectory(directory shared.Directory) string {
	path := filepath.Join(g.Path, string(directory))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatal(path + " not found")
	}
	return path
}

func (g *GameFolder) GetCP() (string, error) {
	librariesDir := g.GetDirectory(shared.DirectoryLibraries)
	cpStr := filepath.Join(g.Path, shared.JAR_FILE)
	err := filepath.WalkDir(librariesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			sep := ":"
			if runtime.GOOS == "windows" {
				sep = ";"

				// Transform the path to be a windows path
				path = strings.ReplaceAll(path, "/", "\\")
			}

			cpStr += sep + path
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return cpStr, nil
}

func (g *GameFolder) GetArguments() ManifestArguments {
	return g.Manifest.Arguments
}

func (g *GameFolder) GetMainClass() string {
	return g.Manifest.MainClass
}

func (g *GameFolder) GetAssetIndex() string {
	return g.Manifest.AssetIndex
}

func (g *GameFolder) GetRuntime() (string, error) {
	if g.Manifest.JavaBinaries == nil {
		return "", fmt.Errorf("java binaries not found")
	}

	if _, ok := g.Manifest.JavaBinaries[shared.PLATFORM]; ok {
		return filepath.Join(g.Path, g.Manifest.JavaBinaries[shared.PLATFORM]), nil
	}
	return "", fmt.Errorf("java binaries not found")
}

func (g *GameFolder) AddFileToKeep(file string) error {
	path := filepath.Join(g.Path, file)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf(path + " not found")
	}

	g.KeepFiles = append(g.KeepFiles, path)
	return nil
}

/////////////////////////////////////////////////////////////////////
// Session
/////////////////////////////////////////////////////////////////////

func (g *GameFolder) HasSession() bool {
	sessionPath := filepath.Join(g.Path, "session.json")
	_, err := os.Stat(sessionPath)
	if err != nil {
		return false
	}
	return true
}

func (g *GameFolder) LoadSession(authenticator authenticator.Authenticator) (string, string, error) {
	// Read the session.json file and return the username and password
	sessionPath := filepath.Join(g.Path, "session.json")
	sessionStr, err := os.ReadFile(sessionPath)
	if err != nil {
		return "", "", fmt.Errorf("error reading session: %w", err)
	}

	var usernamePassword map[string]string
	err = json.Unmarshal(sessionStr, &usernamePassword)
	if err != nil {
		return "", "", fmt.Errorf("error unmarshalling session: %w", err)
	}

	return usernamePassword["username"], usernamePassword["password"], nil
}

func (g *GameFolder) SaveSession(username string, password string) error {
	sessionStr, err := json.MarshalIndent(map[string]string{"username": username, "password": password}, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling session: %w", err)
	}

	// In the path, add a session.json file
	sessionPath := filepath.Join(g.Path, "session.json")
	err = os.WriteFile(sessionPath, sessionStr, 0644)
	if err != nil {
		return fmt.Errorf("error writing session: %w", err)
	}

	err = g.AddFileToKeep("session.json")
	if err != nil {
		return fmt.Errorf("error adding file to keep: %w", err)
	}

	return nil
}

/////////////////////////////////////////////////////////////////////
// Publish
/////////////////////////////////////////////////////////////////////

func PublishGameFolder(connector connectors.Connector, packName string) {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting pwd: ", err)
		return
	}
	// Walk dir the "packs/<pack_name>" folder and check if already in manifest if so skip, if not add to manifest
	dir := filepath.Join(pwd, "packs", packName)
	fmt.Println("Publishing game folder: ", dir)

	//  Read the manifest inside the dir
	var manifest Manifest
	f, err := os.Open(filepath.Join(dir, shared.MANIFEST_FILE))
	if err != nil {
		fmt.Println("Error opening manifest: ", err)
		return
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&manifest)
	if err != nil {
		fmt.Println("Error decoding manifest: ", err)
		return
	}

	// First create a map of the files in the manifest for fast lookup
	manifestFiles := make(map[string]FolderFile)
	for _, file := range manifest.Files {
		manifestFiles[file.Path] = file
	}

	// Count total files first for progress tracking
	totalFiles := 0
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if relPath != "manifest.json" && !d.IsDir() {
			totalFiles++
		}
		return nil
	})

	fmt.Printf("Found %d files to process\n", totalFiles)

	// Progress tracking variables
	processedFiles := 0

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Println("Error walking dir: ", err)
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			fmt.Println("Error getting rel path: ", err)
			return err
		}

		if relPath == "manifest.json" {
			return nil
		}

		if !d.IsDir() {
			// Show progress
			utils.PrintProgress("Publishing", processedFiles, totalFiles, relPath)

			// Publish the file to the connector
			bytes, err := os.ReadFile(path)
			if err != nil {
				fmt.Println("Error reading file: ", err)
				return err
			}

			if err := connector.SendFileFromBytes(relPath, bytes); err != nil {
				fmt.Println("Error sending file to connector: ", err)
				return err
			}

			if _, ok := manifestFiles[relPath]; ok {
				processedFiles++
				return nil
			}

			// Read the file stats
			stats, err := os.Stat(path)
			if err != nil {
				fmt.Println("Error stat file: ", err)
				return err
			}

			folderFile := FolderFile{Path: relPath, Sha: utils.BytesSHA1(bytes), Type: "extra", Rules: nil, Size: stats.Size()}
			manifest.Files = append(manifest.Files, folderFile)
			manifestFiles[relPath] = folderFile
			processedFiles++
		}
		return nil
	})

	// Show completion
	utils.PrintProgress("Publishing", totalFiles, totalFiles, "Complete!")
	fmt.Println() // New line after progress bar

	// Send the updated manifest to the connector
	manifestStr, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling manifest: ", err)
		return
	}

	if err := connector.SendFileFromBytes(shared.MANIFEST_FILE, manifestStr); err != nil {
		fmt.Println("Error sending manifest to connector: ", err)
		return
	}

	fmt.Println("Manifest sent to connector")
}

// ///////////////////////////////////////////////////////////////////
// Build
// ///////////////////////////////////////////////////////////////////

func (g *GameFolder) downloadMissingFiles(filesToDownload []FolderFile, pCb shared.ProgressCallback) error {
	totalFilesToDownload := len(filesToDownload)

	if totalFilesToDownload == 0 {
		return nil
	}

	var downloadedFiles int64

	// Create a worker pool with number of CPUs
	numWorkers := runtime.NumCPU()
	if numWorkers > totalFilesToDownload {
		numWorkers = totalFilesToDownload
	}

	// Configure SFTP connection pool if using SFTP connector
	if sftpConn, ok := g.Connector.(*connectors.SFTPConnector); ok {
		sftpConn.SetPoolSize(numWorkers)
	}

	// Create channels for work distribution
	fileChan := make(chan FolderFile, totalFilesToDownload)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstError error

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				// Download the file with connector
				bytes, err := g.Connector.ReadFileBytes(file.Path, file.Size)
				if err != nil {
					mu.Lock()
					if firstError == nil {
						firstError = fmt.Errorf("failed to read file %s: %w", file.Path, err)
					}
					mu.Unlock()
					continue
				}

				// Create the file in the game folder
				destPath := filepath.Join(g.Path, file.Path)
				err = os.MkdirAll(filepath.Dir(destPath), 0755)
				if err != nil {
					mu.Lock()
					if firstError == nil {
						firstError = fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
					}
					mu.Unlock()
					continue
				}

				mode := 0644
				if file.Executable {
					mode = 0755
				}

				err = os.WriteFile(destPath, bytes, fs.FileMode(mode))
				if err != nil {
					mu.Lock()
					if firstError == nil {
						firstError = fmt.Errorf("failed to write file %s: %w", file.Path, err)
					}
					mu.Unlock()
					continue
				}

				// Update progress atomically
				downloaded := atomic.AddInt64(&downloadedFiles, 1)
				if pCb != nil {
					pCb("Downloading "+file.Type+":", int(downloaded), totalFilesToDownload, file.Path)
				} else {
					utils.PrintProgress("Downloading "+file.Type+":", int(downloaded), totalFilesToDownload, file.Path)
				}
			}
		}()
	}

	// Send files to workers
	for _, file := range filesToDownload {
		fileChan <- file
	}
	close(fileChan)

	// Wait for all workers to complete
	wg.Wait()

	// Check for any errors
	if firstError != nil {
		return firstError
	}

	return nil
}

func (g *GameFolder) Build(debug bool, pCb shared.ProgressCallback) error {

	// 1. Don't download know just skip already downloaded file or file not supported for the current os
	filesToDownload := []FolderFile{}
	allowedFiles := make(map[string]bool, 0)

	for _, file := range g.Manifest.Files {
		if file.Rules != nil && len(file.Rules) > 0 && !rules.ShouldInclude(file.Rules, rules.DetectEnv()) {
			continue
		}

		dest := filepath.Join(g.Path, file.Path)
		// Check if a file at dest exists with same checksum
		if _, err := os.Stat(dest); err == nil {
			// Check if the file at dest has the same checksum as the file in the manifest
			if utils.FileSHA1(dest) == file.Sha {
				allowedFiles[dest] = true
				continue
			}
		}

		filesToDownload = append(filesToDownload, file)
		allowedFiles[dest] = true
	}

	err := g.downloadMissingFiles(filesToDownload, pCb)
	if err != nil {
		return fmt.Errorf("failed to download missing files: %w", err)
	}

	// Then with waldir go through each file in the game folder and check if it's allowed
	filepath.WalkDir(g.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && !allowedFiles[path] {
			// Check if it matches a pattern in KeepFiles
			shouldKeep := false
			for _, keepFile := range g.KeepFiles {
				// Check for exact match first
				if path == keepFile {
					shouldKeep = true
					break
				}
				// Check for glob pattern match (e.g., "logs/*")
				if matched, err := filepath.Match(keepFile, path); err == nil && matched {
					shouldKeep = true
					break
				}
				// Check for regex pattern match (for backward compatibility)
				if match, err := regexp.MatchString(keepFile, path); err == nil && match {
					shouldKeep = true
					break
				}
			}

			if !shouldKeep {
				os.Remove(path)
			}
		}
		return nil
	})

	return nil
}
