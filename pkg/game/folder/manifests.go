package folder

import (
	"limeal.fr/launchygo/pkg/game/folder/generator/manifests"
	"limeal.fr/launchygo/pkg/game/folder/shared"
)

/////////////////////////////////////////////////////////////////////
// Manifest
/////////////////////////////////////////////////////////////////////

type FolderFile struct {
	Size       int64            `json:"size"`
	Path       string           `json:"path"`
	Sha        string           `json:"sha"`  // sha1
	Type       string           `json:"type"` // assets, libraries, natives
	Rules      []manifests.Rule `json:"rules,omitempty"`
	Executable bool             `json:"executable,omitempty"` // If set to true, on dl set the file with 0755 permissions
}

type ManifestArgumentWithRules struct {
	Rules []manifests.Rule `json:"rules"`
	Value any              `json:"value"`
}

type ManifestArguments struct {
	Game []any `json:"game"` // Either a string or a ManifestArgumentWithRules
	JVM  []any `json:"jvm"`  // Either a string or a ManifestArgumentWithRules
}

type Manifest struct {
	MainClass  string            `json:"mainClass"`
	Version    string            `json:"version"`
	McVersion  string            `json:"mcVersion"`
	Arguments  ManifestArguments `json:"arguments"`
	AssetIndex string            `json:"assetIndex"`

	JavaBinaries map[shared.Platform]string `json:"javaBinaries"` // Path to the java binary for the platform
	// Ex: "mac-os": "runtime/mac-os/jre.bundle/Contents/Home/bin/java"
	// "windows": "runtime/windows/bin/java.exe"
	// "linux": "runtime/linux/bin/java"

	// Served has "os book" to only pick elements for the current os
	Files []FolderFile `json:"files"`
}
