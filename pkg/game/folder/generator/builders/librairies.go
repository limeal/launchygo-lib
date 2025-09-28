package builders

import (
	"fmt"
	"path/filepath"
	"slices"

	"limeal.fr/launchygo/pkg/connectors"
	"limeal.fr/launchygo/pkg/game/folder"
	"limeal.fr/launchygo/pkg/game/folder/generator/manifests"
	"limeal.fr/launchygo/pkg/game/folder/rules"
	"limeal.fr/launchygo/pkg/game/folder/shared"
	"limeal.fr/launchygo/pkg/utils"
)

type LibrairiesBuilder struct {
	// MC_FOLDER_PATH or Connector path
	Connector connectors.Connector
	Libraries []manifests.Library
}

func NewLibrairiesBuilder(connector connectors.Connector, libraries []manifests.Library) *LibrairiesBuilder {
	return &LibrairiesBuilder{connector, libraries}
}

func (l *LibrairiesBuilder) GetFolderPath() string {
	return "libraries"
}

func (l *LibrairiesBuilder) filterElements() []string {
	librairiesToDownload := []string{}
	for _, library := range l.Libraries {
		artifact := library.Downloads.Artifact

		if artifact == nil {
			continue
		}

		// If the artificat use itself as native library, we skip it
		nativeClassifiers, ok := rules.ExtractNativeClassifier(artifact, library.Downloads.Classifiers)
		if ok && len(nativeClassifiers) > 0 && nativeClassifiers[0].Artifact.Path == artifact.Path {
			continue
		}

		dest := filepath.Join(l.GetFolderPath(), artifact.Path)
		exists := l.Connector.HasFileWithChecksum(dest, connectors.ChecksumTypeSHA1, artifact.Sha1)
		if !exists {
			librairiesToDownload = append(librairiesToDownload, dest)
		}
	}

	return librairiesToDownload
}

func (l *LibrairiesBuilder) Download(pcb shared.ProgressCallback, debug bool) ([]folder.FolderFile, []rules.NativeClassifier, error) {
	librairiesToDownload := l.filterElements()
	totalLibrairies := len(librairiesToDownload)
	downloadedLibrairies := 0

	nativesArtifacts := []rules.NativeClassifier{}
	librairies := []folder.FolderFile{}
	for _, library := range l.Libraries {
		artifact := library.Downloads.Artifact
		classifiers := library.Downloads.Classifiers

		artifactPath := ""
		if artifact != nil {
			artifactPath = filepath.Join(l.GetFolderPath(), artifact.Path)
		}

		rulesR := rules.ToFolderRules(library.Rules)
		nativeClassifiers, ok := rules.ExtractNativeClassifier(artifact, classifiers)
		if ok {
			nativesArtifacts = append(nativesArtifacts, nativeClassifiers...)
			if nativeClassifiers != nil && len(nativeClassifiers) > 0 && nativeClassifiers[0].Artifact.Path == artifact.Path {
				continue
			}
		}

		if artifactPath == "" {
			// In case the artifact path is empty, we skip the library
			continue
		}

		// librairies/<artifact.Path>
		distPath := filepath.Join(l.GetFolderPath(), artifact.Path)
		librairies = append(librairies, folder.FolderFile{
			Size:  artifact.Size,
			Path:  distPath,
			Sha:   artifact.Sha1,
			Type:  "libraries",
			Rules: rulesR,
		})

		if !slices.Contains(librairiesToDownload, distPath) {
			continue
		}

		downloadedLibrairies++
		// Check if connector has the file and checksum is equal to the artifact.Sha1
		if l.Connector.HasFileWithChecksum(distPath, connectors.ChecksumTypeSHA1, artifact.Sha1) {
			continue
		}

		dlBytes, err := shared.DownloadArtifact(artifact)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to download artifact")
		}

		l.Connector.SendFileFromBytes(distPath, dlBytes)

		if pcb != nil {
			pcb("Downloading libraries", downloadedLibrairies, totalLibrairies, library.Name)
		} else if !debug {
			utils.PrintProgress("Downloading libraries", downloadedLibrairies, totalLibrairies, library.Name)
		} else {
			fmt.Println("[*] Downloading library:", library.Name)
		}
	}

	return librairies, nativesArtifacts, nil
}
