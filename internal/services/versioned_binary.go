package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type lookPathFunc func(string) (string, error)
type commandOutputFunc func(string, ...string) ([]byte, error)

var (
	versionPattern        = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)*`)
	numericVersionPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*$`)
)

func commandOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func versionedCommand(
	kind string,
	version string,
	candidates []string,
	versionArgs []string,
	lookPath lookPathFunc,
	output commandOutputFunc,
) (string, error) {
	var mismatches []string
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true

		path, ok := resolveCommandCandidate(candidate, lookPath)
		if !ok {
			continue
		}
		out, err := output(path, versionArgs...)
		if err != nil {
			mismatches = append(mismatches, fmt.Sprintf("%s (version check failed)", path))
			continue
		}
		outputText := string(out)
		if !numericVersionPattern.MatchString(version) && strings.Contains(outputText, version) {
			return path, nil
		}
		actualVersions := versionsInOutput(outputText)
		for _, actual := range actualVersions {
			if versionLabelMatches(actual, version) {
				return path, nil
			}
		}
		if len(actualVersions) > 0 {
			mismatches = append(mismatches, fmt.Sprintf("%s (%s)", path, strings.Join(actualVersions, ", ")))
		} else {
			mismatches = append(mismatches, fmt.Sprintf("%s (unknown version)", path))
		}
	}

	detail := "no candidate binaries found"
	if len(mismatches) > 0 {
		detail = "found non-matching candidate(s): " + strings.Join(mismatches, ", ")
	}
	return "", fmt.Errorf("could not find %s matching version %s (%s)", kind, version, detail)
}

func resolveCommandCandidate(candidate string, lookPath lookPathFunc) (string, bool) {
	if filepath.IsAbs(candidate) {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			return "", false
		}
		return candidate, true
	}
	path, err := lookPath(candidate)
	if err != nil {
		return "", false
	}
	return path, true
}

func versionsInOutput(s string) []string {
	return versionPattern.FindAllString(s, -1)
}

func versionLabelMatches(actual, requested string) bool {
	return actual == requested || strings.HasPrefix(actual, requested+".")
}
