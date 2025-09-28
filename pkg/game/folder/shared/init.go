package shared

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"

	"limeal.fr/launchygo/pkg/game/folder/generator/manifests"
)

type Directory string

const (
	DirectoryAssets    Directory = "assets"
	DirectoryNatives   Directory = "natives"
	DirectoryLibraries Directory = "libraries"
)

type Platform string

const (
	PlatformMacosIntel Platform = "macos"
	PlatformMacosArm   Platform = "macos-arm64"
	PlatformWindows    Platform = "windows"
	PlatformWindowsArm Platform = "windows-arm64"
	PlatformWindowsX86 Platform = "windows-x86"
	PlatformLinux      Platform = "linux"
)

func (p Platform) GetRuntimes() (*manifests.JavaRuntimes, error) {
	switch p {
	case PlatformMacosIntel:
		return &RUNTIME_MANIFEST.Macos, nil
	case PlatformMacosArm:
		return &RUNTIME_MANIFEST.MacosArm, nil
	case PlatformWindows:
		return &RUNTIME_MANIFEST.WindowsX64, nil
	case PlatformWindowsArm:
		return &RUNTIME_MANIFEST.WindowsArm, nil
	case PlatformWindowsX86:
		return &RUNTIME_MANIFEST.WindowsX86, nil
	case PlatformLinux:
		return &RUNTIME_MANIFEST.Linux, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", p)
	}
}

func (p Platform) GetArchs() []string {
	switch p {
	case PlatformMacosIntel:
		return []string{"amd64", "x86_64"}
	case PlatformMacosArm:
		return []string{"arm64", "aarch64"}
	case PlatformWindows:
		return []string{"amd64", "x86_64"}
	case PlatformWindowsArm:
		return []string{"arm64", "aarch64"}
	case PlatformWindowsX86:
		return []string{"x86", "i386", "386"}
	case PlatformLinux:
		return []string{"amd64", "x86_64"}
	default:
		return []string{}
	}
}

func (p Platform) CreateRules() []manifests.Rule {
	archs := p.GetArchs()
	rules := []manifests.Rule{}
	for _, arch := range archs {
		name := string(p)
		if p == PlatformMacosArm || p == PlatformMacosIntel {
			name = "osx"
		}

		rules = append(rules, manifests.Rule{
			Action: "allow",
			OS: &struct {
				Name string "json:\"name,omitempty\""
				Arch string "json:\"arch,omitempty\""
			}{Name: name, Arch: arch},
		})
	}
	return rules
}

type ProgressCallback func(section string, current int, total int, description string)

const PISTON_MANIFEST_URL = "https://piston-meta.mojang.com/mc/game/version_manifest.json"
const RUNTIME_MANIFEST_URL = "https://launchermeta.mojang.com/v1/products/java-runtime/2ec0cc96c44e5a76b9c8b7c39df7210883d12871/all.json"
const (
	MainClass = "net.minecraft.client.main.Main"
)

var MC_GLOBAL_MANIFEST manifests.MCManifest
var PLATFORM Platform
var RUNTIME_MANIFEST manifests.RuntimeManifest

var MANIFEST_FILE = "manifest.json"
var JAR_FILE = "minecraft.jar"

var ASSETS_DIR = "assets"
var LIBRARIES_DIR = "libraries"
var NATIVES_DIR = "natives"

func GetVersions(releaseOnly bool) []string {
	versions := []string{}
	for _, v := range MC_GLOBAL_MANIFEST.Versions {
		if releaseOnly && v.Type != "release" {
			continue
		}
		versions = append(versions, v.ID)
	}
	return versions
}

func init() {
	// Initialize MC_GLOBAL_MANIFEST
	resp, err := http.Get(PISTON_MANIFEST_URL)
	if err != nil {
		log.Fatal("failed to get version manifest")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("failed to read version manifest")
	}
	json.Unmarshal(body, &MC_GLOBAL_MANIFEST)

	// Initialize RUNTIME_MANIFEST
	resp, err = http.Get(RUNTIME_MANIFEST_URL)
	if err != nil {
		log.Fatal("failed to get runtime manifest")
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("failed to read runtime manifest")
	}
	json.Unmarshal(body, &RUNTIME_MANIFEST)

	// Initialize PLATFORM
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			PLATFORM = PlatformMacosArm
		} else {
			PLATFORM = PlatformMacosIntel
		}
	case "windows":
		if runtime.GOARCH == "arm64" {
			PLATFORM = PlatformWindowsArm
		} else if runtime.GOARCH == "386" {
			PLATFORM = PlatformWindowsX86
		} else {
			PLATFORM = PlatformWindows
		}
	case "linux":
		PLATFORM = PlatformLinux
	}
}
