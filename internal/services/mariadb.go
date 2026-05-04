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
	MariaDBBinaryName  = "mariadbd"
	MariaDBDefaultPort = "3306"
)

const mariadbUnitTmpl = `[Unit]
Description=routa MariaDB {{.Version}}
After=network.target

[Service]
Type=simple
ExecStart={{.Binary}} --defaults-file={{.ConfigPath}}
Restart=on-failure
RestartSec=2
TimeoutStopSec=30

[Install]
WantedBy=default.target
`

const mariadbConfigTmpl = `[mysqld]
bind-address=127.0.0.1
port={{.Port}}
datadir={{.DataDir}}
socket={{.SocketPath}}
pid-file={{.PIDFile}}
log-error={{.LogPath}}
skip-networking=0

[client]
socket={{.SocketPath}}
`

type mariadbUnitData struct {
	Version    string
	Binary     string
	ConfigPath string
}

type mariadbConfigData struct {
	DataDir    string
	SocketPath string
	PIDFile    string
	LogPath    string
	Port       string
}

type MariaDBInstance struct {
	Version string
	Unit    string
	DataDir string
}

func MariaDB(version string) Definition {
	return MariaDBWithPort(version, MariaDBDefaultPort)
}

func MariaDBWithPort(version, port string) Definition {
	return Definition{
		Name:        "mariadb",
		UnitName:    MariaDBUnitName(version),
		BinaryName:  MariaDBBinaryName,
		ConfigPath:  MariaDBConfigPath(version),
		DataDir:     MariaDBDataDir(version),
		RenderUnit:  func(binary string) (string, error) { return RenderMariaDBUnit(version, binary) },
		WriteConfig: func() error { return WriteMariaDBConfigWithPort(version, port) },
	}
}

func ValidateMariaDBVersion(version string) error {
	return validateVersionLabel("MariaDB", version)
}

func MariaDBUnitName(version string) string {
	return "routa-mariadb@" + version + ".service"
}

func MariaDBDataDir(version string) string {
	return filepath.Join(paths.DataDir(), "services", "mariadb", version)
}

func MariaDBConfigPath(version string) string {
	return filepath.Join(paths.ConfigDir(), "services", "mariadb", version, "my.cnf")
}

func MariaDBSocketPath(version string) string {
	return filepath.Join(paths.RunDir(), "mariadb-"+version+".sock")
}

func MariaDBPIDFile(version string) string {
	return filepath.Join(paths.RunDir(), "mariadb-"+version+".pid")
}

func MariaDBLogPath(version string) string {
	return filepath.Join(paths.LogDir(), "mariadb-"+version+".log")
}

func RenderMariaDBUnit(version, binary string) (string, error) {
	if err := ValidateMariaDBVersion(version); err != nil {
		return "", err
	}
	if binary == "" {
		return "", fmt.Errorf("mariadb binary path cannot be empty")
	}
	return render("mariadb-unit", mariadbUnitTmpl, mariadbUnitData{
		Version:    version,
		Binary:     binary,
		ConfigPath: MariaDBConfigPath(version),
	})
}

func RenderMariaDBConfig(version string) (string, error) {
	return RenderMariaDBConfigWithPort(version, MariaDBDefaultPort)
}

func RenderMariaDBConfigWithPort(version, port string) (string, error) {
	if err := ValidateMariaDBVersion(version); err != nil {
		return "", err
	}
	if err := ValidateTCPPort("MariaDB", port); err != nil {
		return "", err
	}
	return render("mariadb-config", mariadbConfigTmpl, mariadbConfigData{
		DataDir:    MariaDBDataDir(version),
		SocketPath: MariaDBSocketPath(version),
		PIDFile:    MariaDBPIDFile(version),
		LogPath:    MariaDBLogPath(version),
		Port:       port,
	})
}

func WriteMariaDBConfig(version string) error {
	return WriteMariaDBConfigWithPort(version, MariaDBDefaultPort)
}

func WriteMariaDBConfigWithPort(version, port string) error {
	if err := ValidateMariaDBVersion(version); err != nil {
		return err
	}
	if err := ValidateTCPPort("MariaDB", port); err != nil {
		return err
	}
	for _, dir := range []string{
		MariaDBDataDir(version),
		filepath.Dir(MariaDBConfigPath(version)),
		paths.RunDir(),
		paths.LogDir(),
	} {
		if err := ensureDir(dir); err != nil {
			return err
		}
	}
	content, err := RenderMariaDBConfigWithPort(version, port)
	if err != nil {
		return err
	}
	return os.WriteFile(MariaDBConfigPath(version), []byte(content), 0o644)
}

func EnsureMariaDB(version string) error {
	return EnsureMariaDBWithPort(version, MariaDBDefaultPort)
}

func EnsureMariaDBWithPort(version, port string) error {
	if err := ValidateMariaDBVersion(version); err != nil {
		return err
	}
	if err := ValidateTCPPort("MariaDB", port); err != nil {
		return err
	}
	bin, err := FindMariaDBBinary(version)
	if err != nil {
		return err
	}
	if err := WriteFiles(MariaDBWithPort(version, port), bin); err != nil {
		return err
	}
	if err := initializeMariaDBDataDir(version, bin); err != nil {
		return err
	}
	return systemd.DaemonReload()
}

func InitializeMariaDBDataDir(version string) error {
	bin, err := FindMariaDBBinary(version)
	if err != nil {
		return err
	}
	return initializeMariaDBDataDir(version, bin)
}

func initializeMariaDBDataDir(version, serverBin string) error {
	initialized, err := MariaDBDataDirInitialized(version)
	if err != nil {
		return err
	}
	if initialized {
		return nil
	}
	initBin, err := findMariaDBInitCommand(version, serverBin)
	if err != nil {
		return err
	}
	cmd := exec.Command(initBin,
		"--defaults-file="+MariaDBConfigPath(version),
		"--datadir="+MariaDBDataDir(version),
		"--skip-test-db",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func FindMariaDBBinary(version string) (string, error) {
	return findMariaDBBinaryWith(version, exec.LookPath, commandOutput)
}

func findMariaDBBinaryWith(version string, lookPath lookPathFunc, output commandOutputFunc) (string, error) {
	if err := ValidateMariaDBVersion(version); err != nil {
		return "", err
	}
	return versionedCommand("MariaDB binary", version, mariadbBinaryCandidates(version), []string{"--version"}, lookPath, output)
}

func mariadbBinaryCandidates(version string) []string {
	major := majorVersionLabel(version)
	return []string{
		"mariadbd-" + version,
		"mariadbd" + version,
		"mysqld-" + version,
		"mysqld" + version,
		filepath.Join("/usr/lib/mariadb", version, "bin", "mariadbd"),
		filepath.Join("/usr/local/mariadb", version, "bin", "mariadbd"),
		filepath.Join("/opt/mariadb", version, "bin", "mariadbd"),
		"mariadbd-" + major,
		"mariadbd" + major,
		"mysqld-" + major,
		"mysqld" + major,
		filepath.Join("/usr/lib/mariadb", major, "bin", "mariadbd"),
		filepath.Join("/usr/local/mariadb", major, "bin", "mariadbd"),
		filepath.Join("/opt/mariadb", major, "bin", "mariadbd"),
		MariaDBBinaryName,
		"mysqld",
	}
}

func MariaDBDataDirInitialized(version string) (bool, error) {
	mysqlDir := filepath.Join(MariaDBDataDir(version), "mysql")
	info, err := os.Stat(mysqlDir)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func InstalledMariaDBInstances() ([]MariaDBInstance, error) {
	versions := map[string]bool{}
	for _, root := range []string{
		filepath.Join(paths.DataDir(), "services", "mariadb"),
		filepath.Join(paths.ConfigDir(), "services", "mariadb"),
	} {
		entries, err := os.ReadDir(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if err := ValidateMariaDBVersion(entry.Name()); err == nil {
				versions[entry.Name()] = true
			}
		}
	}

	for _, unit := range MariaDBUnitNamesForUninstall() {
		version := strings.TrimSuffix(strings.TrimPrefix(unit, "routa-mariadb@"), ".service")
		versions[version] = true
	}

	out := make([]MariaDBInstance, 0, len(versions))
	for version := range versions {
		out = append(out, MariaDBInstance{
			Version: version,
			Unit:    MariaDBUnitName(version),
			DataDir: MariaDBDataDir(version),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func MariaDBUnitNamesForUninstall() []string {
	seen := map[string]bool{}
	for _, pattern := range []string{
		filepath.Join(paths.SystemdUserDir(), "routa-mariadb@*.service"),
		filepath.Join(paths.SystemdUserDir(), "default.target.wants", "routa-mariadb@*.service"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			unit := filepath.Base(match)
			version := strings.TrimSuffix(strings.TrimPrefix(unit, "routa-mariadb@"), ".service")
			if err := ValidateMariaDBVersion(version); err == nil {
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

func findMariaDBInitCommand(version, serverBin string) (string, error) {
	names := []string{"mariadb-install-db", "mysql_install_db"}
	if serverBin != "" {
		dir := filepath.Dir(serverBin)
		for _, name := range names {
			path := filepath.Join(dir, name)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path, nil
			}
		}
	}
	for _, name := range []string{
		"mariadb-install-db-" + version,
		"mysql_install_db-" + version,
		"mariadb-install-db" + version,
		"mysql_install_db" + version,
		"mariadb-install-db-" + majorVersionLabel(version),
		"mysql_install_db-" + majorVersionLabel(version),
		"mariadb-install-db" + majorVersionLabel(version),
		"mysql_install_db" + majorVersionLabel(version),
	} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("missing required command \"mariadb-install-db\" or \"mysql_install_db\". Install MariaDB with your system package manager")
}
