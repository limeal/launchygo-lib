package manifests

import (
	"log"

	"limeal.fr/launchygo/utils"
)

/////////////////////////////////////////////////////////////////////
// FabricVersionManifest: Fabric Version Manifest
/////////////////////////////////////////////////////////////////////

type FabricManifest struct {
	ID           string `json:"id"`
	InheritsFrom string `json:"inheritsFrom"`
	Arguments    struct {
		Game []string `json:"game"`
		JVM  []string `json:"jvm"`
	} `json:"arguments"`
	Libraries []FabricLibrary `json:"libraries"`
	MainClass string          `json:"mainClass"`
}

// Path is: url/
type FabricLibrary struct {
	Name   string  `json:"name"`
	URL    string  `json:"url"`
	MD5    *string `json:"md5,omitempty"`
	SHA1   string  `json:"sha1,omitempty"`
	SHA256 *string `json:"sha256,omitempty"`
	SHA512 *string `json:"sha512,omitempty"`
	Size   int64   `json:"size,omitempty"`
}

func (f *FabricLibrary) ToVanillaLibrary() Library {
	// Build the download URL from the maven path
	downloadURL, destDir, err := utils.BuildDownloadURLFromMavenPath(f.URL, f.Name)
	if err != nil {
		log.Fatal("failed to build download URL from maven path")
	}

	return Library{
		Name: f.Name,
		Downloads: LibraryDownloads{
			Artifact: &Artifact{
				URL:  downloadURL,
				Path: destDir,
				Sha1: f.SHA1,
				Size: f.Size,
			},
		},
	}
}
