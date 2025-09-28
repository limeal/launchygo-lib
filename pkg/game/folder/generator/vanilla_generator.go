package generator

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"limeal.fr/launchygo/pkg/connectors"
	"limeal.fr/launchygo/pkg/game/folder"
	"limeal.fr/launchygo/pkg/game/folder/generator/builders"
	"limeal.fr/launchygo/pkg/game/folder/generator/manifests"
	"limeal.fr/launchygo/pkg/game/folder/shared"
	"limeal.fr/launchygo/pkg/utils"
)

// Vanilla game folder
type VanillaGenerator struct {
	PackPath string
	Version  string
	Manifest manifests.VVersionManifest
	Assets   manifests.AssetsManifest
}

func InitVanillaGenerator(packName string, version string) *VanillaGenerator {
	fmt.Println("[*] Initializing vanilla generator for version: ", version)
	fmt.Println("[*] Pack name: ", packName)

	versionInfo := manifests.VersionInfo{}
	for _, v := range shared.MC_GLOBAL_MANIFEST.Versions {
		if v.ID == version {
			versionInfo = v
			break
		}
	}

	manifestFileURL := versionInfo.URL

	fmt.Println("[*] Version info: ", versionInfo)
	fmt.Println("[*] Manifest file URL: ", manifestFileURL)

	manifest := manifests.VVersionManifest{}
	optionsManifest := utils.NewRequestOptions[manifests.VVersionManifest]("application/json", &manifest)
	_, err := utils.DoRequest(http.MethodGet, manifestFileURL, optionsManifest)
	if err != nil {
		fmt.Println("[*] Error: ", err)
		log.Fatal("failed to decode version manifest")
	}

	assetsManifest := manifests.AssetsManifest{}
	optionsAssets := utils.NewRequestOptions[manifests.AssetsManifest]("application/json", &assetsManifest)
	_, err = utils.DoRequest(http.MethodGet, manifest.AssetIndex.URL, optionsAssets)
	if err != nil {
		log.Fatal("failed to decode assets manifest")
	}

	return &VanillaGenerator{
		PackPath: filepath.Join("packs", packName),
		Version:  version,
		Manifest: manifest,
		Assets:   assetsManifest,
	}
}

// ///////////////////////////////////////////////////////////////////
// Build
// ///////////////////////////////////////////////////////////////////

func (g *VanillaGenerator) Generate(debug bool, pCb shared.ProgressCallback) {

	targetChecksum := g.Manifest.Downloads["client"].Sha1

	// Dont use connector here, we want to write the files in the pack folder
	// or we can simulate a file connector
	fileConnector := connectors.FindConnectorFromURI(fmt.Sprintf("file://./%s", g.PackPath))

	exists := fileConnector.HasFileWithChecksum(shared.JAR_FILE, connectors.ChecksumTypeSHA1, targetChecksum)
	if !exists {
		fmt.Println("[*] Downloading client jar")
		bytes, err := utils.DoRequest[[]byte](http.MethodGet, g.Manifest.Downloads["client"].URL, nil)
		if err != nil {
			log.Fatal("failed to download client jar")
		}
		fmt.Println("[*] Sending client jar to connector")
		fileConnector.SendFileFromBytes(shared.JAR_FILE, bytes)
	}

	// If Java version is defined in manifest, add the runtime builder
	var runtimeFiles []folder.FolderFile
	if g.Manifest.JavaVersion != nil && g.Manifest.JavaVersion.Component != "" {
		fmt.Println("[*] Downloading runtime")
		runtimeBuilder, err := builders.NewRuntimeBuilder(fileConnector, g.Manifest.JavaVersion.Component)
		if err != nil {
			panic(err)
		}

		runtimeFiles, err = runtimeBuilder.Download(pCb)
		if err != nil {
			log.Fatal("failed to download runtime: ", err)
		}

		fmt.Println("[*] Number of runtime files: ", len(runtimeFiles))
	}

	// Download the assets
	fmt.Println("[*] Downloading assets")
	assetsBuilder := builders.NewAssetBuilder(fileConnector, &g.Assets, g.Manifest.AssetIndex.ID)
	assets, err := assetsBuilder.Download(pCb)
	if err != nil {
		log.Fatal("failed to download assets")
	}

	fmt.Println("[*] Number of assets: ", len(assets))

	fmt.Println("\n[*] Downloading libraries")
	librariesBuilder := builders.NewLibrairiesBuilder(fileConnector, g.Manifest.Libraries)
	librairies, natives, err := librariesBuilder.Download(pCb, debug)
	if err != nil {
		log.Fatal("failed to download libraries")
	}

	fmt.Println("[*] Number of libraries: ", len(librairies))

	fmt.Println("\n[*] Downloading natives")
	nativesBuilder := builders.NewNativesBuilder(fileConnector, natives)
	nativeFiles, err := nativesBuilder.Download(pCb, debug)
	if err != nil {
		log.Fatal("failed to download natives")
	}

	fmt.Println("[*] Number of natives: ", len(nativeFiles))

	// Also append the client jar
	files := []folder.FolderFile{{
		Size: g.Manifest.Downloads["client"].Size,
		Path: shared.JAR_FILE,
		Sha:  g.Manifest.Downloads["client"].Sha1,
		Type: "jar",
	}}

	if runtimeFiles != nil && len(runtimeFiles) > 0 {
		files = append(files, runtimeFiles...)
	}

	files = append(files, assets...)
	files = append(files, librairies...)
	files = append(files, nativeFiles...)

	// Create the manifest and send it to the connector
	manifest := folder.Manifest{
		Version:    g.Version,
		MainClass:  g.Manifest.MainClass,
		McVersion:  g.Manifest.Version,
		Arguments:  g.Manifest.Arguments,
		AssetIndex: g.Manifest.AssetIndex.ID,
		Files:      files,
	}

	if runtimeFiles != nil && len(runtimeFiles) > 0 {
		manifest.JavaBinaries = make(map[shared.Platform]string)
		manifest.JavaBinaries[shared.PlatformMacosArm] = fmt.Sprintf("runtime/%s/jre.bundle/Contents/Home/bin/java", shared.PlatformMacosArm)
		manifest.JavaBinaries[shared.PlatformMacosIntel] = fmt.Sprintf("runtime/%s/jre.bundle/Contents/Home/bin/java", shared.PlatformMacosIntel)
		manifest.JavaBinaries[shared.PlatformWindows] = fmt.Sprintf("runtime\\%s\\bin\\java.exe", shared.PlatformWindows)
		manifest.JavaBinaries[shared.PlatformLinux] = fmt.Sprintf("runtime/%s/bin/java", shared.PlatformLinux)
	}

	manifestStr, err := json.MarshalIndent(manifest, "", "\t")
	if err != nil {
		log.Fatal("failed to marshal manifest")
	}
	fileConnector.SendFileFromBytes(shared.MANIFEST_FILE, manifestStr)
}
