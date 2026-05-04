// Package services manages optional routa user-space services such as Redis.
package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/scottzirkel/routa/internal/paths"
	"github.com/scottzirkel/routa/internal/systemd"
)

type Definition struct {
	Name        string
	UnitName    string
	BinaryName  string
	ConfigPath  string
	DataDir     string
	RenderUnit  func(binary string) (string, error)
	WriteConfig func() error
}

func Ensure(def Definition) error {
	bin, err := exec.LookPath(def.BinaryName)
	if err != nil {
		return fmt.Errorf("missing required command %q. Install it with your system package manager", def.BinaryName)
	}
	if err := WriteFiles(def, bin); err != nil {
		return err
	}
	return systemd.DaemonReload()
}

func WriteFiles(def Definition, bin string) error {
	if def.WriteConfig != nil {
		if err := def.WriteConfig(); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(paths.SystemdUserDir(), 0o755); err != nil {
		return err
	}
	content, err := def.RenderUnit(bin)
	if err != nil {
		return err
	}
	if err := writeUserUnit(def.UnitName, content); err != nil {
		return err
	}
	return nil
}

func writeUserUnit(unitName, content string) error {
	return os.WriteFile(filepath.Join(paths.SystemdUserDir(), unitName), []byte(content), 0o644)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func validateVersionLabel(kind, version string) error {
	if version == "" {
		return fmt.Errorf("%s version cannot be empty", kind)
	}
	if !isVersionLabelEdge(version[0]) || !isVersionLabelEdge(version[len(version)-1]) {
		return fmt.Errorf("invalid %s version %q", kind, version)
	}
	for i := 0; i < len(version); i++ {
		if isVersionLabelEdge(version[i]) || version[i] == '.' || version[i] == '-' || version[i] == '_' {
			continue
		}
		return fmt.Errorf("invalid %s version %q: use letters, numbers, dots, dashes, or underscores", kind, version)
	}
	if strings.Contains(version, "..") {
		return fmt.Errorf("invalid %s version %q", kind, version)
	}
	return nil
}

func isVersionLabelEdge(b byte) bool {
	return ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z') || ('0' <= b && b <= '9')
}

func majorVersionLabel(version string) string {
	major, _, _ := strings.Cut(version, ".")
	return major
}

func ValidateTCPPort(label, port string) error {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("%s port must be 1-65535", label)
	}
	return nil
}

func readVersionDirs(root string, validate func(string) error) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err := validate(entry.Name()); err == nil {
			out = append(out, entry.Name())
		}
	}
	return out, nil
}

func versionedUnitNamesForUninstall(prefix string, validate func(string) error) []string {
	seen := map[string]bool{}
	for _, pattern := range []string{
		filepath.Join(paths.SystemdUserDir(), prefix+"*.service"),
		filepath.Join(paths.SystemdUserDir(), "default.target.wants", prefix+"*.service"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			unit := filepath.Base(match)
			version := strings.TrimSuffix(strings.TrimPrefix(unit, prefix), ".service")
			if err := validate(version); err == nil {
				seen[unit] = true
			}
		}
	}
	units := make([]string, 0, len(seen))
	for unit := range seen {
		units = append(units, unit)
	}
	sort.Strings(units)
	return units
}

func render(name, tmpl string, data any) (string, error) {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return b.String(), nil
}
