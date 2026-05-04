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
	TypesenseBinaryName  = "typesense-server"
	TypesenseAPIKey      = "routa-local-dev"
	TypesenseAddr        = "127.0.0.1"
	TypesenseDefaultPort = "8108"
)

const typesenseUnitTmpl = `[Unit]
Description=routa Typesense {{.Version}}
After=network.target

[Service]
Type=simple
ExecStart={{.Binary}} --data-dir {{.DataDir}} --api-key {{.APIKey}} --listen-address {{.Addr}} --api-port {{.Port}} --enable-cors
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.LogPath}}
Restart=on-failure
RestartSec=2
TimeoutStopSec=30

[Install]
WantedBy=default.target
`

type typesenseUnitData struct {
	Version string
	Binary  string
	DataDir string
	APIKey  string
	Addr    string
	Port    string
	LogPath string
}

type TypesenseInstance struct {
	Version string
	Unit    string
	DataDir string
}

func ValidateTypesenseVersion(version string) error {
	return validateVersionLabel("Typesense", version)
}

func TypesenseUnitName(version string) string {
	return "routa-typesense@" + version + ".service"
}

func TypesenseDataDir(version string) string {
	return filepath.Join(paths.DataDir(), "services", "typesense", version)
}

func TypesenseLogPath(version string) string {
	return filepath.Join(paths.LogDir(), "typesense-"+version+".log")
}

func RenderTypesenseUnit(version, binary string) (string, error) {
	return RenderTypesenseUnitWithPort(version, binary, TypesenseDefaultPort)
}

func RenderTypesenseUnitWithPort(version, binary, port string) (string, error) {
	if err := ValidateTypesenseVersion(version); err != nil {
		return "", err
	}
	if err := ValidateTCPPort("Typesense", port); err != nil {
		return "", err
	}
	if binary == "" {
		return "", fmt.Errorf("typesense binary path cannot be empty")
	}
	return render("typesense-unit", typesenseUnitTmpl, typesenseUnitData{
		Version: version,
		Binary:  binary,
		DataDir: TypesenseDataDir(version),
		APIKey:  TypesenseAPIKey,
		Addr:    TypesenseAddr,
		Port:    port,
		LogPath: TypesenseLogPath(version),
	})
}

func WriteTypesenseFiles(version, binary string) error {
	return WriteTypesenseFilesWithPort(version, binary, TypesenseDefaultPort)
}

func WriteTypesenseFilesWithPort(version, binary, port string) error {
	if err := ValidateTypesenseVersion(version); err != nil {
		return err
	}
	if err := ValidateTCPPort("Typesense", port); err != nil {
		return err
	}
	for _, dir := range []string{
		TypesenseDataDir(version),
		paths.LogDir(),
		paths.SystemdUserDir(),
	} {
		if err := ensureDir(dir); err != nil {
			return err
		}
	}
	content, err := RenderTypesenseUnitWithPort(version, binary, port)
	if err != nil {
		return err
	}
	return writeUserUnit(TypesenseUnitName(version), content)
}

func EnsureTypesense(version string) error {
	return EnsureTypesenseWithPort(version, TypesenseDefaultPort)
}

func EnsureTypesenseWithPort(version, port string) error {
	bin, err := FindTypesenseBinary(version)
	if err != nil {
		return err
	}
	if err := WriteTypesenseFilesWithPort(version, bin, port); err != nil {
		return err
	}
	return systemd.DaemonReload()
}

func FindTypesenseBinary(version string) (string, error) {
	return findTypesenseBinaryWith(version, exec.LookPath, commandOutput)
}

func findTypesenseBinaryWith(version string, lookPath lookPathFunc, output commandOutputFunc) (string, error) {
	if err := ValidateTypesenseVersion(version); err != nil {
		return "", err
	}
	return versionedCommand("Typesense binary", version, typesenseBinaryCandidates(version), []string{"--version"}, lookPath, output)
}

func typesenseBinaryCandidates(version string) []string {
	major := majorVersionLabel(version)
	return []string{
		"typesense-server-" + version,
		"typesense-server" + version,
		"typesense-" + version,
		"typesense" + version,
		filepath.Join("/usr/local/typesense", version, "typesense-server"),
		filepath.Join("/opt/typesense", version, "typesense-server"),
		"typesense-server-" + major,
		"typesense-server" + major,
		"typesense-" + major,
		"typesense" + major,
		filepath.Join("/usr/local/typesense", major, "typesense-server"),
		filepath.Join("/opt/typesense", major, "typesense-server"),
		TypesenseBinaryName,
	}
}

func InstalledTypesenseInstances() ([]TypesenseInstance, error) {
	versions := map[string]bool{}
	entries, err := readVersionDirs(filepath.Join(paths.DataDir(), "services", "typesense"), ValidateTypesenseVersion)
	if err != nil {
		return nil, err
	}
	for _, version := range entries {
		versions[version] = true
	}
	for _, unit := range TypesenseUnitNamesForUninstall() {
		version := strings.TrimSuffix(strings.TrimPrefix(unit, "routa-typesense@"), ".service")
		versions[version] = true
	}

	out := make([]TypesenseInstance, 0, len(versions))
	for version := range versions {
		out = append(out, TypesenseInstance{
			Version: version,
			Unit:    TypesenseUnitName(version),
			DataDir: TypesenseDataDir(version),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func TypesenseUnitNamesForUninstall() []string {
	return versionedUnitNamesForUninstall("routa-typesense@", ValidateTypesenseVersion)
}
