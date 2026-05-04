// Package services manages optional routa user-space services such as Redis.
package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	if err := os.WriteFile(filepath.Join(paths.SystemdUserDir(), def.UnitName), []byte(content), 0o644); err != nil {
		return err
	}
	return nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
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
