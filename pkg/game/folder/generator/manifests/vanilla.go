package manifests

type MCManifest struct {
	Latest   LatestVersions `json:"latest"`
	Versions []VersionInfo  `json:"versions"`
}

type LatestVersions struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

type VersionInfo struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	SHA1 string `json:"sha1"`
	Type string `json:"type"`
}

/////////////////////////////////////////////////////////////////////
// VVersionManifest: Vanilla Version Manifest
/////////////////////////////////////////////////////////////////////

type Rule struct {
	Action string `json:"action"`
	OS     *struct {
		Name string `json:"name,omitempty"`
		Arch string `json:"arch,omitempty"`
	} `json:"os,omitempty"`
	Features map[string]bool `json:"features,omitempty"`
}

type VVersionManifest struct {
	Arguments struct {
		Game []any `json:"game"`
		JVM  []any `json:"jvm"`
	} `json:"arguments"`
	AssetIndex struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"assetIndex"`

	Downloads              map[string]DownloadEntry `json:"downloads"`
	Version                string                   `json:"id"`
	Libraries              []Library                `json:"libraries"`
	Logging                Logging                  `json:"logging"`
	MainClass              string                   `json:"mainClass"`
	MinimumLauncherVersion int                      `json:"minimumLauncherVersion"`
	JavaVersion            *struct {
		Component    string `json:"component"`    // "java-runtime-gamma"
		MajorVersion int64  `json:"majorVersion"` // "17"
	} `json:"javaVersion,omitempty"`
}

type DownloadEntry struct {
	Sha1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type LibraryDownloads struct {
	Artifact    *Artifact            `json:"artifact,omitempty"`
	Classifiers map[string]*Artifact `json:"classifiers,omitempty"`
}

type Library struct {
	Downloads LibraryDownloads `json:"downloads"`
	Name      string           `json:"name"`
	Rules     []Rule           `json:"rules,omitempty"`
}

type Artifact struct {
	Path string `json:"path"`
	Sha1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type Logging struct {
	Client struct {
		Argument string `json:"argument"`
		File     struct {
			ID   string `json:"id"`
			Sha1 string `json:"sha1"`
			Size int64  `json:"size"`
			URL  string `json:"url"`
		} `json:"file"`
		Type string `json:"type"`
	} `json:"client"`
}

/////////////////////////////////////////////////////////////////////
// AssetsManifest
/////////////////////////////////////////////////////////////////////

type AssetsManifest struct {
	Objects map[string]AssetObject `json:"objects"`
}

type AssetObject struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

/////////////////////////////////////////////////////////////////////
// RuntimeManifest
/////////////////////////////////////////////////////////////////////

// Single manifest file
type JavaRuntimeManifestFile struct {
	Type       string `json:"type"`                 // file or directory
	Executable bool   `json:"executable,omitempty"` // if type is file, true if it's executable
	Downloads  struct {
		Lzma struct {
			URL  string `json:"url"`
			Size int64  `json:"size"`
			Sha1 string `json:"sha1"`
		} `json:"lzma"`
		Raw struct {
			URL  string `json:"url"`
			Size int64  `json:"size"`
			Sha1 string `json:"sha1"`
		} `json:"raw"`
	} `json:"downloads,omitempty"` // empty if type is directory
}

type JavaRuntimeManifest struct {
	Files map[string]JavaRuntimeManifestFile `json:"files"`
}

/////////////////////////////////////////////////////////////////////

// Global file
type JavaRuntime struct {
	Availability struct {
		Group    int64 `json:"group"`
		Progress int64 `json:"progress"`
	} `json:"availability"` // NOTE: I don't know what this is
	Manifest struct {
		Sha1 string `json:"sha1"`
		Size int64  `json:"size"`
		URL  string `json:"url"`
	} `json:"manifest"`
	Version struct {
		Name     string `json:"name"`
		Released string `json:"released"` // date in iso format
	} `json:"version"`
}

type JavaRuntimes struct {
	JavaRuntimeAlpha         []JavaRuntime `json:"java-runtime-alpha"`
	JavaRuntimeBeta          []JavaRuntime `json:"java-runtime-beta"`
	JavaRuntimeGamma         []JavaRuntime `json:"java-runtime-gamma"`
	JavaRuntimeGammaSnapshot []JavaRuntime `json:"java-runtime-gamma-snapshot"`
	JavaRuntimeDelta         []JavaRuntime `json:"java-runtime-delta"`
	JreLegacy                []JavaRuntime `json:"jre-legacy"`
	MinecraftJavaExe         []JavaRuntime `json:"minecraft-java-exe,omitempty"`
}

type RuntimeManifest struct {
	GameCore   JavaRuntimes `json:"game-core"`
	Linux      JavaRuntimes `json:"linux"`
	LinuxI386  JavaRuntimes `json:"linux-i386"`
	Macos      JavaRuntimes `json:"mac-os"`
	MacosArm   JavaRuntimes `json:"mac-os-arm64"`
	WindowsArm JavaRuntimes `json:"windows-arm64"`
	WindowsX64 JavaRuntimes `json:"windows-x64"`
	WindowsX86 JavaRuntimes `json:"windows-x86"`
}
