package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func CopyFile(src, dst string) error {
	// 1. Open the source file for reading.
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	// Defer closing the source file to ensure it's closed even if an error occurs.
	defer sourceFile.Close()

	// 2. Create the destination file.
	// This will create a new file or truncate an existing one.
	os.MkdirAll(filepath.Dir(dst), 0755)
	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	// Defer closing the destination file.
	defer destinationFile.Close()

	// 3. Use io.Copy to efficiently copy the contents.
	// This function handles the reading from the source and writing to the destination
	// in an optimized way, without loading the entire file into memory.
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

func CopyFileWithMeta(src, dst string, buf []byte) error {
	// Ensure parent exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir parents: %w", err)
	}

	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer sf.Close()

	si, err := sf.Stat()
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, si.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open dst: %w", err)
	}
	defer func() {
		_ = df.Close()
	}()

	// Pre-size (helps on some FS)
	if si.Size() > 0 {
		_ = df.Truncate(si.Size())
	}

	if _, err := io.CopyBuffer(df, sf, buf); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	// Preserve mode
	if err := os.Chmod(dst, si.Mode()); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	return nil
}

func DownloadAndExtractNative(srcJar, destDir string, extensions []string) error {
	// Open the source JAR file as a zip archive.
	readCloser, err := os.Open(srcJar)
	if err != nil {
		return fmt.Errorf("failed to open source JAR file: %w", err)
	}
	defer readCloser.Close()

	// Get the file info to get the file size.
	fileInfo, err := readCloser.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	zipReader, err := zip.NewReader(readCloser, fileInfo.Size())
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	foundCount := 0
	// Iterate over all the files in the archive to find the .dylib.
	for _, file := range zipReader.File {
		// Check if the file entry is a directory. If it is, create it and continue.
		if file.FileInfo().IsDir() {
			continue
		}

		validExtensions := append([]string{".MF", ".LIST"}, extensions...)

		// We are only interested in .dylib files and want to skip directories.
		if slices.Contains(validExtensions, filepath.Ext(file.Name)) {
			// Construct the output file path, preserving the folder structure.
			destPath := filepath.Join(destDir, file.Name)

			// Ensure the destination directory exists before creating the file.
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create destination directory: %w", err)
			}

			destFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create destination file: %w", err)
			}

			// Open the file inside the zip archive.
			sourceFile, err := file.Open()
			if err != nil {
				destFile.Close() // Ensure the file is closed on error
				return fmt.Errorf("failed to open file in zip: %w", err)
			}

			// Copy the contents and then close the files immediately.
			_, err = io.Copy(destFile, sourceFile)
			sourceFile.Close()
			destFile.Close()
			if err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}

			foundCount++
		}
	}

	if foundCount == 0 {
		return fmt.Errorf("no .dylib files found in the archive")
	}

	return nil
}

func BuildDownloadURLFromMavenPath(base, coord string) (string, string, error) {
	// transform: base = https://maven.fabricmc.net/ and coord = org.ow2.asm:asm:9.8
	// to: https://maven.fabricmc.net/org/ow2/asm/asm/9.8/asm-9.8.jar

	// Split the coordinate by colons
	parts := strings.Split(coord, ":")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid maven coordinate format: %s (expected groupId:artifactId:version)", coord)
	}

	groupId := parts[0]
	artifactId := parts[1]
	version := parts[2]

	// Convert groupId dots to slashes for URL path
	groupIdPath := strings.ReplaceAll(groupId, ".", "/")

	// Build the file path
	fileName := fmt.Sprintf("%s-%s.jar", artifactId, version)

	// Ensure base URL ends with a slash
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}

	// Construct the full URL
	destDir := fmt.Sprintf("%s/%s/%s/%s", groupIdPath, artifactId, version, fileName)
	url := fmt.Sprintf("%s%s", base, destDir)

	return url, destDir, nil
}
