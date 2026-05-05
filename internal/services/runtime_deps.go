package services

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type RuntimeDependencyError struct {
	Service   string
	Version   string
	Binary    string
	Libraries []string
}

func (e *RuntimeDependencyError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s is installed under routa, but this system is missing runtime libraries:", e.Service, e.Version)
	for _, lib := range e.Libraries {
		fmt.Fprintf(&b, "\n  %s", lib)
	}
	if cmd := installCommandForLibraries(e.Libraries); cmd != "" {
		fmt.Fprintf(&b, "\n\nInstall them with:\n  %s", cmd)
	} else {
		fmt.Fprint(&b, "\n\nInstall the OS packages that provide these libraries, then rerun the command.")
	}
	if e.Binary != "" {
		fmt.Fprintf(&b, "\n\nChecked binary:\n  %s", e.Binary)
	}
	return b.String()
}

func installCommandForLibraries(libs []string) string {
	pkgs := packagesForLibraries(distroID(), libs)
	if len(pkgs) == 0 {
		return ""
	}
	switch distroID() {
	case "arch", "endeavouros", "manjaro":
		return "sudo pacman -S " + strings.Join(pkgs, " ")
	case "debian", "ubuntu", "linuxmint", "pop":
		return "sudo apt install " + strings.Join(pkgs, " ")
	case "fedora":
		return "sudo dnf install " + strings.Join(pkgs, " ")
	default:
		return ""
	}
}

func packagesForLibraries(distro string, libs []string) []string {
	pkgByLib := map[string]map[string]string{
		"libaio.so.1": {
			"arch":        "libaio",
			"endeavouros": "libaio",
			"manjaro":     "libaio",
			"debian":      "libaio1",
			"ubuntu":      "libaio1",
			"linuxmint":   "libaio1",
			"pop":         "libaio1",
			"fedora":      "libaio",
		},
		"libnuma.so.1": {
			"arch":        "numactl",
			"endeavouros": "numactl",
			"manjaro":     "numactl",
			"debian":      "libnuma1",
			"ubuntu":      "libnuma1",
			"linuxmint":   "libnuma1",
			"pop":         "libnuma1",
			"fedora":      "numactl-libs",
		},
	}
	seen := map[string]bool{}
	var out []string
	for _, lib := range libs {
		pkg := pkgByLib[lib][distro]
		if pkg == "" || seen[pkg] {
			continue
		}
		seen[pkg] = true
		out = append(out, pkg)
	}
	sort.Strings(out)
	return out
}

func distroID() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key != "ID" {
			continue
		}
		return strings.Trim(strings.ToLower(value), `"`)
	}
	return ""
}
