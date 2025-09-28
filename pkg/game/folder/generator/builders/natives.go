package builders

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"slices"

	"limeal.fr/launchygo/pkg/game/folder"
	"limeal.fr/launchygo/pkg/game/folder/generator/manifests"
	"limeal.fr/launchygo/pkg/game/folder/rules"
	"limeal.fr/launchygo/pkg/game/folder/shared"
	"limeal.fr/launchygo/pkg/connectors"
	"limeal.fr/launchygo/pkg/utils"
)

type NativesBuilder struct {
	Connector connectors.Connector
	Natives   []rules.NativeClassifier
}

func NewNativesBuilder(connector connectors.Connector, artifacts []rules.NativeClassifier) *NativesBuilder {
	return &NativesBuilder{connector, artifacts}
}

func (l *NativesBuilder) GetFolderPath() string {
	return "natives"
}

func (l *NativesBuilder) filterElements() []string {
	return []string{}
}

func (l *NativesBuilder) constructDynamicLibrary(zipReader *zip.Reader, rules []manifests.Rule) ([]folder.FolderFile, error) {
	foundCount := 0
	nativeFilesLocalized := []folder.FolderFile{}
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		validExtensions := []string{".so", ".dll", ".dylib"}

		if !slices.Contains(validExtensions, filepath.Ext(file.Name)) {
			continue
		}

		destPath := filepath.Join(l.GetFolderPath(), file.Name)

		sourceFile, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file in zip: %w", err)
		}
		defer sourceFile.Close()

		bytes, err := io.ReadAll(sourceFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		checksum := utils.BytesSHA1(bytes)
		if l.Connector.HasFileWithChecksum(destPath, connectors.ChecksumTypeSHA1, checksum) {
			continue
		}

		nativeFilesLocalized = append(nativeFilesLocalized, folder.FolderFile{
			Size:  int64(len(bytes)),
			Path:  destPath,
			Sha:   checksum,
			Type:  "natives",
			Rules: rules,
		})

		l.Connector.SendFileFromBytes(destPath, bytes)
		foundCount++

	}

	return nativeFilesLocalized, nil
}

func (l *NativesBuilder) Download(pcb shared.ProgressCallback, debug bool) ([]folder.FolderFile, error) {
	totalNatives := len(l.Natives)
	processedNatives := 0

	natives := []folder.FolderFile{}
	for _, artifact := range l.Natives {
		artifactBytes, err := shared.DownloadArtifact(&artifact.Artifact)
		if err != nil {
			log.Fatal("failed to download artifact")
		}

		readCloser := bytes.NewReader(artifactBytes)

		zipReader, err := zip.NewReader(readCloser, int64(len(artifactBytes)))
		if err != nil {
			log.Fatal("failed to create zip reader")
		}

		nativeFilesLocalized, err := l.constructDynamicLibrary(zipReader, artifact.Rules)
		if err != nil {
			log.Fatal("failed to construct dynamic library")
		}

		natives = append(natives, nativeFilesLocalized...)
		processedNatives++

		if pcb != nil {
			pcb("Downloading natives", processedNatives, totalNatives, artifact.Artifact.Path)
		} else if !debug {
			utils.PrintProgress("Downloading natives", processedNatives, totalNatives, artifact.Artifact.Path)
		}
	}

	return natives, nil
}
