package services

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/scottzirkel/routa/internal/paths"
	"github.com/scottzirkel/routa/internal/systemd"
)

const (
	MeilisearchBinaryName  = "meilisearch"
	MeilisearchDefaultPort = "7700"
)

const meilisearchUnitTmpl = `[Unit]
Description=routa Meilisearch {{.Version}}
After=network.target

[Service]
Type=simple
ExecStart={{.Binary}} --env development --http-addr {{.Addr}} --db-path {{.DataDir}}
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.LogPath}}
Restart=on-failure
RestartSec=2
TimeoutStopSec=30

[Install]
WantedBy=default.target
`

type meilisearchUnitData struct {
	Version string
	Binary  string
	Addr    string
	DataDir string
	LogPath string
}

type MeilisearchInstance struct {
	Version string
	Unit    string
	DataDir string
}

func ValidateMeilisearchVersion(version string) error {
	return validateVersionLabel("Meilisearch", version)
}

func MeilisearchUnitName(version string) string {
	return "routa-meilisearch@" + version + ".service"
}

func MeilisearchDataDir(version string) string {
	return filepath.Join(paths.DataDir(), "services", "meilisearch", version)
}

func MeilisearchLogPath(version string) string {
	return filepath.Join(paths.LogDir(), "meilisearch-"+version+".log")
}

func RenderMeilisearchUnit(version, binary string) (string, error) {
	return RenderMeilisearchUnitWithPort(version, binary, MeilisearchDefaultPort)
}

func RenderMeilisearchUnitWithPort(version, binary, port string) (string, error) {
	if err := ValidateMeilisearchVersion(version); err != nil {
		return "", err
	}
	if err := ValidateTCPPort("Meilisearch", port); err != nil {
		return "", err
	}
	if binary == "" {
		return "", fmt.Errorf("meilisearch binary path cannot be empty")
	}
	return render("meilisearch-unit", meilisearchUnitTmpl, meilisearchUnitData{
		Version: version,
		Binary:  binary,
		Addr:    "127.0.0.1:" + port,
		DataDir: MeilisearchDataDir(version),
		LogPath: MeilisearchLogPath(version),
	})
}

func WriteMeilisearchFiles(version, binary string) error {
	return WriteMeilisearchFilesWithPort(version, binary, MeilisearchDefaultPort)
}

func WriteMeilisearchFilesWithPort(version, binary, port string) error {
	if err := ValidateMeilisearchVersion(version); err != nil {
		return err
	}
	if err := ValidateTCPPort("Meilisearch", port); err != nil {
		return err
	}
	for _, dir := range []string{
		MeilisearchDataDir(version),
		paths.LogDir(),
		paths.SystemdUserDir(),
	} {
		if err := ensureDir(dir); err != nil {
			return err
		}
	}
	content, err := RenderMeilisearchUnitWithPort(version, binary, port)
	if err != nil {
		return err
	}
	return writeUserUnit(MeilisearchUnitName(version), content)
}

func EnsureMeilisearch(version string) error {
	return EnsureMeilisearchWithPort(version, MeilisearchDefaultPort)
}

func EnsureMeilisearchWithPort(version, port string) error {
	bin, err := FindMeilisearchBinary(version)
	if err != nil {
		return err
	}
	if err := WriteMeilisearchFilesWithPort(version, bin, port); err != nil {
		return err
	}
	return systemd.DaemonReload()
}

func FindMeilisearchBinary(version string) (string, error) {
	return findMeilisearchBinaryWith(version, exec.LookPath, commandOutput)
}

func findMeilisearchBinaryWith(version string, lookPath lookPathFunc, output commandOutputFunc) (string, error) {
	if err := ValidateMeilisearchVersion(version); err != nil {
		return "", err
	}
	return versionedCommand("Meilisearch binary", version, meilisearchBinaryCandidates(version), []string{"--version"}, lookPath, output)
}

func meilisearchBinaryCandidates(version string) []string {
	major := majorVersionLabel(version)
	return []string{
		"meilisearch-" + version,
		"meilisearch" + version,
		filepath.Join("/usr/local/meilisearch", version, "meilisearch"),
		filepath.Join("/opt/meilisearch", version, "meilisearch"),
		"meilisearch-" + major,
		"meilisearch" + major,
		filepath.Join("/usr/local/meilisearch", major, "meilisearch"),
		filepath.Join("/opt/meilisearch", major, "meilisearch"),
		MeilisearchBinaryName,
	}
}

func InstalledMeilisearchInstances() ([]MeilisearchInstance, error) {
	versions := map[string]bool{}
	for _, root := range []string{
		filepath.Join(paths.DataDir(), "services", "meilisearch"),
	} {
		entries, err := readVersionDirs(root, ValidateMeilisearchVersion)
		if err != nil {
			return nil, err
		}
		for _, version := range entries {
			versions[version] = true
		}
	}
	for _, unit := range MeilisearchUnitNamesForUninstall() {
		version := strings.TrimSuffix(strings.TrimPrefix(unit, "routa-meilisearch@"), ".service")
		versions[version] = true
	}

	out := make([]MeilisearchInstance, 0, len(versions))
	for version := range versions {
		out = append(out, MeilisearchInstance{
			Version: version,
			Unit:    MeilisearchUnitName(version),
			DataDir: MeilisearchDataDir(version),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func MeilisearchUnitNamesForUninstall() []string {
	return versionedUnitNamesForUninstall("routa-meilisearch@", ValidateMeilisearchVersion)
}
