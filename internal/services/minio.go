package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/scottzirkel/routa/internal/paths"
	"github.com/scottzirkel/routa/internal/systemd"
)

const (
	MinIOBinaryName         = "minio"
	MinIODefaultPort        = "9000"
	MinIODefaultConsolePort = "9001"
	MinIORootUser           = "routa"
	MinIORootPassword       = "routa-local-dev"
	minIOEnvFileMode        = 0o600
)

const minIOUnitTmpl = `[Unit]
Description=routa MinIO {{.Version}}
After=network.target

[Service]
Type=simple
EnvironmentFile={{.ConfigPath}}
ExecStart={{.Binary}} server --address {{.Addr}} --console-address {{.ConsoleAddr}} {{.DataDir}}
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.LogPath}}
Restart=on-failure
RestartSec=2
TimeoutStopSec=30

[Install]
WantedBy=default.target
`

const minIOConfigTmpl = `MINIO_ROOT_USER={{.RootUser}}
MINIO_ROOT_PASSWORD={{.RootPassword}}
`

type minIOUnitData struct {
	Version     string
	Binary      string
	ConfigPath  string
	Addr        string
	ConsoleAddr string
	DataDir     string
	LogPath     string
}

type minIOConfigData struct {
	RootUser     string
	RootPassword string
}

type MinIOInstance struct {
	Version string
	Unit    string
	DataDir string
}

func ValidateMinIOVersion(version string) error {
	return validateVersionLabel("MinIO", version)
}

func MinIOUnitName(version string) string {
	return "routa-minio@" + version + ".service"
}

func MinIODataDir(version string) string {
	return filepath.Join(paths.DataDir(), "services", "minio", version)
}

func MinIOConfigPath(version string) string {
	return filepath.Join(paths.ConfigDir(), "services", "minio", version, "env")
}

func MinIOLogPath(version string) string {
	return filepath.Join(paths.LogDir(), "minio-"+version+".log")
}

func RenderMinIOUnit(version, binary string) (string, error) {
	return RenderMinIOUnitWithPorts(version, binary, MinIODefaultPort, MinIODefaultConsolePort)
}

func RenderMinIOUnitWithPorts(version, binary, port, consolePort string) (string, error) {
	if err := ValidateMinIOVersion(version); err != nil {
		return "", err
	}
	if err := ValidateTCPPort("MinIO", port); err != nil {
		return "", err
	}
	if err := ValidateTCPPort("MinIO console", consolePort); err != nil {
		return "", err
	}
	if binary == "" {
		return "", fmt.Errorf("minio binary path cannot be empty")
	}
	return render("minio-unit", minIOUnitTmpl, minIOUnitData{
		Version:     version,
		Binary:      binary,
		ConfigPath:  MinIOConfigPath(version),
		Addr:        "127.0.0.1:" + port,
		ConsoleAddr: "127.0.0.1:" + consolePort,
		DataDir:     MinIODataDir(version),
		LogPath:     MinIOLogPath(version),
	})
}

func RenderMinIOConfig() (string, error) {
	return render("minio-config", minIOConfigTmpl, minIOConfigData{
		RootUser:     MinIORootUser,
		RootPassword: MinIORootPassword,
	})
}

func WriteMinIOFiles(version, binary string) error {
	return WriteMinIOFilesWithPorts(version, binary, MinIODefaultPort, MinIODefaultConsolePort)
}

func WriteMinIOFilesWithPorts(version, binary, port, consolePort string) error {
	if err := ValidateMinIOVersion(version); err != nil {
		return err
	}
	if err := ValidateTCPPort("MinIO", port); err != nil {
		return err
	}
	if err := ValidateTCPPort("MinIO console", consolePort); err != nil {
		return err
	}
	for _, dir := range []string{
		MinIODataDir(version),
		filepath.Dir(MinIOConfigPath(version)),
		paths.LogDir(),
		paths.SystemdUserDir(),
	} {
		if err := ensureDir(dir); err != nil {
			return err
		}
	}
	config, err := RenderMinIOConfig()
	if err != nil {
		return err
	}
	if err := os.WriteFile(MinIOConfigPath(version), []byte(config), minIOEnvFileMode); err != nil {
		return err
	}
	unit, err := RenderMinIOUnitWithPorts(version, binary, port, consolePort)
	if err != nil {
		return err
	}
	return writeUserUnit(MinIOUnitName(version), unit)
}

func EnsureMinIO(version string) error {
	return EnsureMinIOWithPorts(version, MinIODefaultPort, MinIODefaultConsolePort)
}

func EnsureMinIOWithPorts(version, port, consolePort string) error {
	bin, err := FindMinIOBinary(version)
	if err != nil {
		return err
	}
	if err := WriteMinIOFilesWithPorts(version, bin, port, consolePort); err != nil {
		return err
	}
	return systemd.DaemonReload()
}

func FindMinIOBinary(version string) (string, error) {
	return findMinIOBinaryWith(version, exec.LookPath, commandOutput)
}

func findMinIOBinaryWith(version string, lookPath lookPathFunc, output commandOutputFunc) (string, error) {
	if err := ValidateMinIOVersion(version); err != nil {
		return "", err
	}
	return versionedCommand("MinIO binary", version, minIOBinaryCandidates(version), []string{"--version"}, lookPath, output)
}

func minIOBinaryCandidates(version string) []string {
	major := majorVersionLabel(version)
	return []string{
		"minio-" + version,
		"minio" + version,
		filepath.Join("/usr/local/minio", version, "minio"),
		filepath.Join("/opt/minio", version, "minio"),
		"minio-" + major,
		"minio" + major,
		filepath.Join("/usr/local/minio", major, "minio"),
		filepath.Join("/opt/minio", major, "minio"),
		MinIOBinaryName,
	}
}

func InstalledMinIOInstances() ([]MinIOInstance, error) {
	versions := map[string]bool{}
	for _, root := range []string{
		filepath.Join(paths.DataDir(), "services", "minio"),
		filepath.Join(paths.ConfigDir(), "services", "minio"),
	} {
		entries, err := readVersionDirs(root, ValidateMinIOVersion)
		if err != nil {
			return nil, err
		}
		for _, version := range entries {
			versions[version] = true
		}
	}
	for _, unit := range MinIOUnitNamesForUninstall() {
		version := strings.TrimSuffix(strings.TrimPrefix(unit, "routa-minio@"), ".service")
		versions[version] = true
	}

	out := make([]MinIOInstance, 0, len(versions))
	for version := range versions {
		out = append(out, MinIOInstance{
			Version: version,
			Unit:    MinIOUnitName(version),
			DataDir: MinIODataDir(version),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func MinIOUnitNamesForUninstall() []string {
	return versionedUnitNamesForUninstall("routa-minio@", ValidateMinIOVersion)
}
