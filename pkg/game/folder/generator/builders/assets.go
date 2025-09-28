package builders

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"slices"

	"limeal.fr/launchygo/pkg/connectors"
	"limeal.fr/launchygo/pkg/game/folder"
	"limeal.fr/launchygo/pkg/game/folder/generator/manifests"
	"limeal.fr/launchygo/pkg/game/folder/shared"
	"limeal.fr/launchygo/pkg/utils"
)

const resourcesBase = "https://resources.download.minecraft.net"

type AssetBuilder struct {
	Connector      connectors.Connector
	AssetsManifest *manifests.AssetsManifest
	AssetsIdx      string
}

func NewAssetBuilder(connector connectors.Connector, assetsManifest *manifests.AssetsManifest, assetsIdx string) *AssetBuilder {
	return &AssetBuilder{
		Connector:      connector,
		AssetsManifest: assetsManifest,
		AssetsIdx:      assetsIdx,
	}
}

func (a *AssetBuilder) GetFolderPath() string {
	return "assets"
}

func (a *AssetBuilder) GetAssetsIndexPath() string {
	return filepath.Join(a.GetFolderPath(), "indexes", a.AssetsIdx+".json")
}

func (a *AssetBuilder) filterElements() []string {
	assetsObjectsDir := filepath.Join(a.GetFolderPath(), "objects")

	assetsToDownload := []string{}
	for _, asset := range a.AssetsManifest.Objects {
		hashMin := asset.Hash[:2]
		dest := filepath.Join(assetsObjectsDir, hashMin, asset.Hash)
		exists := a.Connector.HasFileWithChecksum(dest, connectors.ChecksumTypeSHA1, asset.Hash)

		if !exists {
			assetsToDownload = append(assetsToDownload, dest)
		}
	}

	return assetsToDownload
}

func (a *AssetBuilder) Download(pcb shared.ProgressCallback) ([]folder.FolderFile, error) {
	assetsStr, err := json.Marshal(a.AssetsManifest)
	if err != nil {
		log.Fatal("failed to marshal assets manifest")
	}

	assets := []folder.FolderFile{}
	assets = append(assets, folder.FolderFile{
		Size: int64(len(assetsStr)),
		Path: a.GetAssetsIndexPath(),
		Sha:  utils.BytesSHA1(assetsStr),
		Type: "assets",
	})

	a.Connector.SendFileFromBytes(a.GetAssetsIndexPath(), assetsStr)

	assetsObjectsDir := filepath.Join(a.GetFolderPath(), "objects")

	assetsToDownload := a.filterElements()
	totalAssets := len(assetsToDownload)
	downloadedAssets := 0

	for _, asset := range a.AssetsManifest.Objects {
		hashMin := asset.Hash[:2]

		existingAssetPath := filepath.Join(assetsObjectsDir, hashMin, asset.Hash)

		assets = append(assets, folder.FolderFile{
			Size: asset.Size,
			Path: existingAssetPath,
			Sha:  asset.Hash,
			Type: "assets",
		})

		if !slices.Contains(assetsToDownload, existingAssetPath) {
			continue
		}

		dest := filepath.Join(assetsObjectsDir, hashMin, asset.Hash)
		url := resourcesBase + "/" + hashMin + "/" + asset.Hash
		bytes, err := utils.DoRequest[[]byte](http.MethodGet, url, nil)
		if err != nil {
			log.Fatal("\nFailed to download asset: ", url)
		}

		a.Connector.SendFileFromBytes(dest, bytes)

		downloadedAssets++
		if pcb != nil {
			pcb("Downloading assets", downloadedAssets, totalAssets, asset.Hash)
		} else {
			utils.PrintProgress("Downloading assets", downloadedAssets, totalAssets, asset.Hash)
		}
	}

	return assets, nil
}
