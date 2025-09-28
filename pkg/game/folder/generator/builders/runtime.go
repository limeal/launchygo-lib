package builders

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"slices"

	"limeal.fr/launchygo/pkg/game/folder"
	"limeal.fr/launchygo/pkg/game/folder/generator/manifests"
	"limeal.fr/launchygo/pkg/game/folder/shared"
	"limeal.fr/launchygo/pkg/connectors"
	"limeal.fr/launchygo/pkg/utils"
)

type RuntimeBuilder struct {
	Connector        connectors.Connector
	RuntimeManifests map[shared.Platform]manifests.JavaRuntimeManifest
}

func getJavaRuntimeFromVersion(platform shared.Platform, javaVersion string) (*manifests.JavaRuntime, error) {
	var runtimes manifests.JavaRuntimes
	switch platform {
	case shared.PlatformMacosIntel:
		runtimes = shared.RUNTIME_MANIFEST.Macos
	case shared.PlatformMacosArm:
		runtimes = shared.RUNTIME_MANIFEST.MacosArm
	case shared.PlatformWindows:
		runtimes = shared.RUNTIME_MANIFEST.WindowsX64
	case shared.PlatformWindowsArm:
		runtimes = shared.RUNTIME_MANIFEST.WindowsArm
	case shared.PlatformWindowsX86:
		runtimes = shared.RUNTIME_MANIFEST.WindowsX86
	case shared.PlatformLinux:
		runtimes = shared.RUNTIME_MANIFEST.Linux
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	// We must take the runtimes[javaVersion] maybe by unmarshal the json into a map[string]JavaRuntime
	bytes, err := json.Marshal(runtimes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal runtimes: %w", err)
	}

	var runtimesMap map[string][]manifests.JavaRuntime
	json.Unmarshal(bytes, &runtimesMap)

	runtime, ok := runtimesMap[javaVersion]
	if !ok {
		return nil, fmt.Errorf("java runtime not found for version: %s", javaVersion)
	}

	if len(runtime) == 0 {
		return nil, fmt.Errorf("java runtime not found for version: %s", javaVersion)
	}

	return &runtime[0], nil
}

func NewRuntimeBuilder(connector connectors.Connector, javaVersion string) (*RuntimeBuilder, error) {
	manifestsR := make(map[shared.Platform]manifests.JavaRuntimeManifest)
	for _, platform := range []shared.Platform{
		shared.PlatformMacosIntel,
		shared.PlatformMacosArm,
		shared.PlatformWindows,
		shared.PlatformWindowsArm,
		shared.PlatformWindowsX86,
		shared.PlatformLinux,
	} {
		runtime, err := getJavaRuntimeFromVersion(platform, javaVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to get java runtime from version: %s", err)
		}

		// Then fetch the manifest
		manifest := manifests.JavaRuntimeManifest{}
		optionsAssets := utils.NewRequestOptions[manifests.JavaRuntimeManifest]("application/json", &manifest)
		if _, err := utils.DoRequest(http.MethodGet, runtime.Manifest.URL, optionsAssets); err != nil {
			return nil, fmt.Errorf("failed to get java runtime manifest from version: %s", err)
		}

		manifestsR[platform] = manifest
	}

	return &RuntimeBuilder{Connector: connector, RuntimeManifests: manifestsR}, nil
}

func (r *RuntimeBuilder) GetFolderPath() string {
	return "runtime"
}

func (r *RuntimeBuilder) filterElements() []string {
	runtimeFiles := []string{}
	for platform, manifest := range r.RuntimeManifests {
		dirPath := filepath.Join(r.GetFolderPath(), string(platform))
		for key, file := range manifest.Files {
			if file.Type != "file" {
				continue
			}

			runtimeFiles = append(runtimeFiles, filepath.Join(dirPath, key))
		}
	}
	return runtimeFiles
}

func (r *RuntimeBuilder) Download(pcb shared.ProgressCallback) ([]folder.FolderFile, error) {
	runtimeFiles := r.filterElements()
	totalRuntimeFiles := len(runtimeFiles)
	downloadedRuntimeFiles := 0

	// Go through each files and download them
	var files []folder.FolderFile
	for platform, manifest := range r.RuntimeManifests {
		dirPath := filepath.Join(r.GetFolderPath(), string(platform))
		for key, file := range manifest.Files {

			filePath := filepath.Join(dirPath, key)
			if !slices.Contains(runtimeFiles, filePath) {
				continue
			}

			bytes, err := utils.DoRequest[[]byte](http.MethodGet, file.Downloads.Raw.URL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to download runtime file: %s", err)
			}

			mode := 0644
			if file.Executable {
				mode = 0755
			}

			err = r.Connector.SendFileFromBytes(filePath, bytes, fs.FileMode(mode))
			if err != nil {
				return nil, fmt.Errorf("failed to send runtime file to connector: %s", err)
			}

			files = append(files, folder.FolderFile{
				Size:       file.Downloads.Raw.Size,
				Path:       filePath,
				Sha:        file.Downloads.Raw.Sha1,
				Type:       "runtime",
				Executable: file.Executable,
				Rules:      platform.CreateRules(),
			})

			downloadedRuntimeFiles++
			if pcb != nil {
				pcb("Downloading runtime", downloadedRuntimeFiles, totalRuntimeFiles, fmt.Sprintf("%s/%s", platform, key))
			} else {
				utils.PrintProgress("Downloading runtime", downloadedRuntimeFiles, totalRuntimeFiles, fmt.Sprintf("%s/%s", platform, key))
			}
		}
	}

	return files, nil
}
