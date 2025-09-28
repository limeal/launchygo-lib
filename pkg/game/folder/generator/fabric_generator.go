package generator

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"limeal.fr/launchygo/pkg/game/folder/generator/manifests"
	"limeal.fr/launchygo/pkg/game/folder/shared"
)

// Vanilla game folder
type FabricGenerator struct {
	PackPath string
	Version  string

	FabricManifest   manifests.FabricManifest
	VanillaGenerator *VanillaGenerator
}

func InitFabricGenerator(packName string, fabricVersion string) *FabricGenerator {
	file, err := os.Open(fabricVersion)
	if err != nil {
		log.Fatal("failed to open fabric manifest")
	}
	defer file.Close()

	var manifest manifests.FabricManifest
	if err := json.NewDecoder(file).Decode(&manifest); err != nil {
		log.Fatal("failed to decode fabric manifest")
	}

	vanillaGenerator := InitVanillaGenerator(packName, manifest.InheritsFrom)
	return &FabricGenerator{
		PackPath:         filepath.Join("packs", packName),
		Version:          manifest.InheritsFrom,
		FabricManifest:   manifest,
		VanillaGenerator: vanillaGenerator,
	}
}

// ///////////////////////////////////////////////////////////////////
// Build
// ///////////////////////////////////////////////////////////////////

func (g *FabricGenerator) Generate(debug bool, pCb shared.ProgressCallback) {
	libraries := g.VanillaGenerator.Manifest.Libraries
	for _, library := range g.FabricManifest.Libraries {
		libraries = append(libraries, library.ToVanillaLibrary())
	}

	g.VanillaGenerator.Version = g.FabricManifest.ID
	g.VanillaGenerator.Manifest.MainClass = g.FabricManifest.MainClass
	g.VanillaGenerator.Manifest.Libraries = libraries

	// Convert []string to []any for appending
	for _, arg := range g.FabricManifest.Arguments.Game {
		g.VanillaGenerator.Manifest.Arguments.Game = append(g.VanillaGenerator.Manifest.Arguments.Game, arg)
	}
	for _, arg := range g.FabricManifest.Arguments.JVM {
		g.VanillaGenerator.Manifest.Arguments.JVM = append(g.VanillaGenerator.Manifest.Arguments.JVM, arg)
	}

	g.VanillaGenerator.Generate(debug, pCb)
}
