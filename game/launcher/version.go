package launcher

import (
	"strconv"
	"strings"
)

func GetJavaVersionForVersion(mcVersion string) string {
	versionMap := map[string]string{
		"< 1.16":    "8",
		"1.16":      "11",
		"1.17":      "16",
		"1.18":      "17",
		">= 1.20.5": "21",
	}

	operatorMap := map[string]func(a, b string) bool{
		"<":  VersionLT,
		"<=": VersionLTE,
		">":  VersionGT,
		">=": VersionGTE,
		"=":  VersionEQ,
		"!=": VersionNE,
	}

	for version, javaVersion := range versionMap {
		// If starts with <, then it's less than the version
		for operator, operatorFunc := range operatorMap {
			if strings.HasPrefix(version, operator) {
				// Remove the operator from the version
				version = strings.TrimPrefix(version, operator)
				if operatorFunc(mcVersion, strings.TrimSpace(version)) {
					return javaVersion
				}
			}
		}

		// If it's not a operator, then it's equal to the version
		if VersionEQ(mcVersion, version) {
			return javaVersion
		}
	}

	return "21"
}

func VersionLT(a, b string) bool  { return versionCmp(a, b) < 0 }
func VersionLTE(a, b string) bool { return versionCmp(a, b) <= 0 }
func VersionGT(a, b string) bool  { return versionCmp(a, b) > 0 }
func VersionGTE(a, b string) bool { return versionCmp(a, b) >= 0 }
func VersionEQ(a, b string) bool  { return versionCmp(a, b) == 0 }
func VersionNE(a, b string) bool  { return versionCmp(a, b) != 0 }

func versionCmp(a, b string) int {
	parse := func(s string) (maj, min, patch int) {
		parts := strings.Split(s, ".")
		if len(parts) > 0 {
			maj, _ = strconv.Atoi(parts[0])
		}
		if len(parts) > 1 {
			min, _ = strconv.Atoi(parts[1])
		}
		if len(parts) > 2 {
			patch, _ = strconv.Atoi(parts[2])
		}
		return
	}
	am, an, ap := parse(a)
	bm, bn, bp := parse(b)
	if am != bm {
		if am < bm {
			return -1
		} else {
			return 1
		}
	}
	if an != bn {
		if an < bn {
			return -1
		} else {
			return 1
		}
	}
	if ap != bp {
		if ap < bp {
			return -1
		} else {
			return 1
		}
	}
	return 0
}
