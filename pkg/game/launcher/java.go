package launcher

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// GetJavaPath finds a java executable that matches version spec (e.g., "17", "1.8", "17.0.8")
// and, on macOS, matches archSpec ("", "arm64", "x86_64", "universal").
// Returns absolute path to the java binary.
func GetJavaPath(verSpec, archSpec string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var candidates []string

	switch runtime.GOOS {
	case "darwin":
		candidates = append(candidates, darwinCandidates(ctx)...)
	case "linux":
		candidates = append(candidates, linuxCandidates(ctx)...)
	case "windows":
		candidates = append(candidates, windowsCandidates(ctx)...)
	default:
		candidates = append(candidates, whichAll(ctx, "java")...)
	}

	// Always include PATH last as fallback
	if p, _ := exec.LookPath("java"); p != "" {
		candidates = append(candidates, p)
	}

	seen := map[string]struct{}{}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		abs, _ := filepath.Abs(c)
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}

		// macOS: enforce arch if requested
		if runtime.GOOS == "darwin" && archSpec != "" {
			ok, _ := candidateHasArch(ctx, abs, archSpec)
			if !ok {
				continue
			}
		}

		if v, err := javaVersion(ctx, abs); err == nil && versionMatches(verSpec, v) {
			return abs, nil
		}
	}

	return "", fmt.Errorf("no java matching version %q%v found",
		verSpec, func() string {
			if runtime.GOOS == "darwin" && archSpec != "" {
				return " and arch " + archSpec
			}
			return ""
		}())
}

// -------------------- macOS (incl. Homebrew + Temurin + Zulu) --------------------

func darwinCandidates(ctx context.Context) []string {
	var homes []string

	// 1) Apple & user JVM folders (incl. casks like temurin-17.jdk, zulu-17.jdk)
	globs := []string{
		"/Library/Java/JavaVirtualMachines/*/Contents/Home",
		filepath.Join(os.Getenv("HOME"), "Library/Java/JavaVirtualMachines/*/Contents/Home"),

		// Explicit vendor-friendly globs (helpful when you want to see vendors distinctly)
		"/Library/Java/JavaVirtualMachines/temurin-*.jdk/Contents/Home",
		"/Library/Java/JavaVirtualMachines/zulu-*.jdk/Contents/Home",
		filepath.Join(os.Getenv("HOME"), "Library/Java/JavaVirtualMachines/temurin-*.jdk/Contents/Home"),
		filepath.Join(os.Getenv("HOME"), "Library/Java/JavaVirtualMachines/zulu-*.jdk/Contents/Home"),
	}

	// 2) Homebrew Cellar/opt layouts (OpenJDK/Temurin variants installed as formulae)
	if prefix := brewPrefix(ctx); prefix != "" {
		globs = append(globs,
			// openjdk formulae
			filepath.Join(prefix, "Cellar", "openjdk*", "*", "libexec", "openjdk.jdk", "Contents", "Home"),
			filepath.Join(prefix, "opt", "openjdk", "libexec", "openjdk.jdk", "Contents", "Home"),
			filepath.Join(prefix, "opt", "openjdk@*", "libexec", "openjdk.jdk", "Contents", "Home"),

			// Temurin (some taps package it as formula too)
			filepath.Join(prefix, "Cellar", "temurin*", "*", "libexec", "openjdk.jdk", "Contents", "Home"),
			filepath.Join(prefix, "opt", "temurin", "libexec", "openjdk.jdk", "Contents", "Home"),
			filepath.Join(prefix, "opt", "temurin@*", "libexec", "openjdk.jdk", "Contents", "Home"),

			// Zulu (rare as formula, mostly cask, but include for completeness)
			filepath.Join(prefix, "Cellar", "zulu*", "*", "libexec", "openjdk.jdk", "Contents", "Home"),
			filepath.Join(prefix, "opt", "zulu*", "libexec", "openjdk.jdk", "Contents", "Home"),
		)
	}

	// 3) Parse `/usr/libexec/java_home -V` list (don’t trust -v picker)
	if out := javaHomeVerbose(ctx); out != "" {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Matching Java Virtual Machines") {
				continue
			}
			if i := strings.LastIndex(line, "/Contents/Home"); i != -1 {
				start := strings.LastIndex(line[:i], " ")
				if start == -1 {
					start = 0
				}
				home := strings.TrimSpace(line[start : i+len("/Contents/Home")])
				if home != "" {
					homes = append(homes, home)
				}
			}
		}
	}

	// Expand globs
	for _, g := range globs {
		if matches, _ := filepath.Glob(g); len(matches) > 0 {
			homes = append(homes, matches...)
		}
	}

	// Convert to java binary paths
	var cands []string
	seen := map[string]struct{}{}
	for _, h := range homes {
		java := filepath.Join(h, "bin", "java")
		if _, ok := seen[java]; ok {
			continue
		}
		seen[java] = struct{}{}
		cands = append(cands, java)
	}
	// Include PATH (sometimes user linked a preferred JDK)
	cands = append(cands, whichAll(ctx, "java")...)
	return cands
}

func brewPrefix(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "brew", "--prefix").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	// Heuristics if brew not on PATH
	if fi, err := os.Stat("/opt/homebrew"); err == nil && fi.IsDir() {
		return "/opt/homebrew" // Apple Silicon default
	}
	return "/usr/local" // Intel default
}

func javaHomeVerbose(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "/usr/libexec/java_home", "-V")
	out, _ := cmd.CombinedOutput() // -V prints to stderr on some macOS versions
	return string(out)
}

// candidateHasArch returns true if binary supports requested archSpec: "arm64", "x86_64", "universal".
func candidateHasArch(ctx context.Context, javaPath, archSpec string) (bool, error) {
	if archSpec == "" {
		return true, nil
	}
	archSpec = normalizeArch(archSpec)

	// Resolve symlinks (Homebrew often symlinks)
	if tgt, err := filepath.EvalSymlinks(javaPath); err == nil && tgt != "" {
		javaPath = tgt
	}

	// Prefer lipo
	arches, err := lipoArches(ctx, javaPath)
	if err != nil || len(arches) == 0 {
		// Fallback to `file` output
		arches = fileArches(ctx, javaPath)
	}
	if len(arches) == 0 {
		// If we cannot determine, don't block; assume OK
		return true, nil
	}

	hasArm := contains(arches, "arm64") || contains(arches, "arm64e")
	hasX86 := contains(arches, "x86_64")

	switch archSpec {
	case "universal":
		return hasArm && hasX86, nil
	case "arm64":
		return hasArm, nil
	case "x86_64":
		return hasX86, nil
	default:
		return contains(arches, archSpec), nil
	}
}

func lipoArches(ctx context.Context, path string) ([]string, error) {
	out, err := exec.CommandContext(ctx, "/usr/bin/lipo", "-archs", path).Output()
	if err != nil {
		return nil, err
	}
	var res []string
	for _, a := range strings.Fields(string(out)) {
		res = append(res, normalizeArch(a))
	}
	return res, nil
}

func fileArches(ctx context.Context, path string) []string {
	out, err := exec.CommandContext(ctx, "/usr/bin/file", "-b", path).Output()
	if err != nil {
		return nil
	}
	s := strings.ToLower(string(out))
	var arches []string
	if strings.Contains(s, "x86_64") {
		arches = append(arches, "x86_64")
	}
	if strings.Contains(s, "arm64e") || strings.Contains(s, "arm64") {
		arches = append(arches, "arm64")
	}
	return arches
}

func normalizeArch(a string) string {
	a = strings.ToLower(strings.TrimSpace(a))
	switch a {
	case "amd64":
		return "x86_64"
	case "arm64e":
		return "arm64"
	default:
		return a
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// -------------------- Linux --------------------

func linuxCandidates(ctx context.Context) []string {
	var cands []string

	// update-alternatives
	if out, err := exec.CommandContext(ctx, "update-alternatives", "--list", "java").Output(); err == nil {
		for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if l != "" {
				cands = append(cands, strings.TrimSpace(l))
			}
		}
	}

	// Common JVM roots
	for _, g := range []string{
		"/usr/lib/jvm/*/bin/java",
		"/usr/java/*/bin/java",
	} {
		if matches, _ := filepath.Glob(g); len(matches) > 0 {
			cands = append(cands, matches...)
		}
	}

	// PATH
	cands = append(cands, whichAll(ctx, "java")...)
	return cands
}

// -------------------- Windows --------------------

func windowsCandidates(ctx context.Context) []string {
	var cands []string
	cands = append(cands, whereAll(ctx, "java.exe")...)
	if jh := os.Getenv("JAVA_HOME"); jh != "" {
		cands = append(cands, filepath.Join(jh, "bin", "java.exe"))
	}
	for _, root := range []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		`C:\Program Files`,
		`C:\Program Files (x86)`,
	} {
		if root == "" {
			continue
		}
		for _, g := range []string{
			filepath.Join(root, "Java", "*", "bin", "java.exe"),
			filepath.Join(root, "Eclipse Adoptium", "jdk-*", "bin", "java.exe"),
			filepath.Join(root, "AdoptOpenJDK", "jdk-*", "bin", "java.exe"),
			filepath.Join(root, "Zulu", "zulu*", "bin", "java.exe"),
		} {
			if matches, _ := filepath.Glob(g); len(matches) > 0 {
				cands = append(cands, matches...)
			}
		}
	}
	return cands
}

// -------------------- Generic helpers --------------------

func whichAll(ctx context.Context, bin string) []string {
	out, err := exec.CommandContext(ctx, "which", "-a", bin).Output()
	if err != nil {
		if p, _ := exec.LookPath(bin); p != "" {
			return []string{p}
		}
		return nil
	}
	var res []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			res = append(res, l)
		}
	}
	return res
}

func whereAll(ctx context.Context, bin string) []string {
	out, err := exec.CommandContext(ctx, "where", bin).Output()
	if err != nil {
		return nil
	}
	var res []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			res = append(res, l)
		}
	}
	return res
}

// --- Version parsing & checks ---

var versionRe = regexp.MustCompile(`"(.*?)"`) // extracts "17.0.10" or "1.8.0_392"

func javaVersion(ctx context.Context, javaPath string) (string, error) {
	cmd := exec.CommandContext(ctx, javaPath, "-version")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr // java -version prints to stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	m := versionRe.FindStringSubmatch(stderr.String())
	if len(m) < 2 {
		return "", fmt.Errorf("failed to parse version from: %s", stderr.String())
	}
	return m[1], nil
}

func versionMatches(spec, actual string) bool {
	spec = strings.TrimSpace(spec)
	actual = strings.TrimSpace(actual)
	if spec == "" || actual == "" {
		return false
	}

	// Compare majors first (normalize "1.8" -> 8)
	specMajor := majorOf(spec)
	actMajor := majorOf(actual)
	if specMajor > 0 && actMajor > 0 && specMajor != actMajor {
		return false
	}

	// If spec pins minor/patch (e.g., "17.0.8", "1.8.0_392"), require prefix match.
	// If spec is just major ("17", "11", "8", or "1.8"), accept any with same major.
	if pinsMinorOrPatch(spec) {
		return strings.HasPrefix(actual, spec)
	}
	return true
}

func majorOf(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if strings.HasPrefix(v, "1.") { // legacy Java 8 style
		parts := strings.SplitN(v, ".", 3)
		if len(parts) >= 2 {
			return atoi(parts[1])
		}
		return 0
	}
	parts := strings.SplitN(v, ".", 2)
	return atoi(parts[0])
}

func pinsMinorOrPatch(spec string) bool {
	spec = strings.TrimSpace(spec)
	switch spec {
	case "17", "16", "15", "14", "13", "12", "11", "10", "9":
		return false
	case "8", "1.8":
		return false
	default:
		return true
	}
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
