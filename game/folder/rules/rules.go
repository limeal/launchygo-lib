package rules

import (
	"runtime"
	"strings"

	"limeal.fr/launchygo/game/folder/generator/manifests"
	"limeal.fr/launchygo/game/folder/shared"
)

type Env struct {
	Platform shared.Platform
	Arch     string // x86_64 | aarch64 | x86 | arm
}

type Feature struct {
	AKey  string // is_quick_play_singleplayer
	Flag  string // quickPlaySingleplayer
	Value string // true
}

type NativeClassifier struct {
	Artifact manifests.Artifact
	Rules    []manifests.Rule
}

func DetectEnv() Env {
	arch := runtime.GOARCH // amd64, arm64, 386, arm
	plat := shared.PlatformLinux
	switch runtime.GOOS {
	case "windows":
		plat = shared.PlatformWindows
	case "darwin":
		if arch == "arm64" {
			plat = shared.PlatformMacosArm
		} else {
			plat = shared.PlatformMacosIntel
		}
	case "linux":
		plat = shared.PlatformLinux
	}
	return Env{Platform: plat, Arch: map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "x86", "arm": "arm"}[arch]}
}

func ShouldInclude(rulesList []manifests.Rule, env Env) bool {
	if len(rulesList) == 0 {
		return true
	}
	allowed := false
	for _, r := range rulesList {
		applies := true
		if r.OS != nil {
			name := strings.ToLower(r.OS.Name)
			// Mojang historically uses "osx"; newer use "macos". Treat both as mac.
			if env.Platform == shared.PlatformWindows || env.Platform == shared.PlatformWindowsArm || env.Platform == shared.PlatformWindowsX86 {
				applies = name == "windows"
			}
			if env.Platform == shared.PlatformLinux {
				applies = name == "linux"
			}
			if env.Platform == shared.PlatformMacosIntel || env.Platform == shared.PlatformMacosArm {
				applies = (name == "osx" || name == "macos")
			}
			if applies && r.OS.Arch != "" {
				applies = (strings.ToLower(r.OS.Arch) == strings.ToLower(env.Arch))
			}
		}

		// If the rule has features, it is not applicable
		if r.Features != nil {
			return false
		}

		if applies {
			if r.Action == "disallow" {
				return false
			}
			if r.Action == "allow" {
				allowed = true
			}
		}
	}
	return allowed
}

func ShouldIncludeFeatures(rulesList []manifests.Rule, features ...Feature) bool {
	for _, rule := range rulesList {
		if rule.Features == nil {
			continue
		}
		for _, feature := range features {
			if val, ok := rule.Features[feature.AKey]; ok && val {
				return true
			}
		}
	}

	return false
}

// Function that will check if the library is a native, if so it return the classifier key + ok
// if not it return nil, false
func ExtractNativeClassifier(artifact *manifests.Artifact, cls map[string]*manifests.Artifact) ([]NativeClassifier, bool) {
	// Predefine classifier keys for each platform to avoid repeated code and unnecessary checks
	platformClassifiers := map[shared.Platform][]string{
		shared.PlatformWindows:    {"windows", "natives-windows"},
		shared.PlatformWindowsArm: {"windows", "natives-windows-arm64"},
		shared.PlatformWindowsX86: {"windows", "natives-windows-x86"},
		shared.PlatformLinux:      {"linux", "natives-linux", "natives-linux-64", "natives-linux-32"},
		shared.PlatformMacosIntel: {"osx", "natives-macos", "natives-osx"},
		shared.PlatformMacosArm:   {"osx", "natives-macos-arm64"},
	}

	// First, check for all existing platforms if the suffix matches
	nativeClassifiers := []NativeClassifier{}
	for p, classifierKeys := range platformClassifiers {
		for _, k := range classifierKeys {
			if artifact != nil && strings.Contains(artifact.Path, "natives-") && strings.HasSuffix(artifact.Path, k+".jar") {
				nativeClassifiers = append(nativeClassifiers, NativeClassifier{
					Artifact: *artifact,
					Rules:    p.CreateRules(),
				})
				return nativeClassifiers, true
			}
			if cls[k] != nil {
				nativeClassifiers = append(nativeClassifiers, NativeClassifier{
					Artifact: *cls[k],
					Rules:    p.CreateRules(),
				})
				break
			}
		}
	}

	if len(nativeClassifiers) == 0 {
		return nil, false
	}

	return nativeClassifiers, true
}

func ToFolderRules(rules []manifests.Rule) []manifests.Rule {
	os := []manifests.Rule{}
	for _, rule := range rules {
		if rule.OS != nil {
			os = append(os, manifests.Rule{Action: rule.Action, OS: rule.OS, Features: rule.Features})
		}
	}

	if len(os) == 0 {
		return nil
	}
	return os
}
