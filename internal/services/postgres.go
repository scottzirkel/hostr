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
	PostgresBinaryName  = "postgres"
	PostgresDefaultPort = "5432"
)

const postgresUnitTmpl = `[Unit]
Description=routa Postgres {{.Version}}
After=network.target

[Service]
Type=simple
ExecStart={{.Binary}} -D {{.DataDir}} -c config_file={{.ConfigPath}}
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.LogPath}}
Restart=on-failure
RestartSec=2
TimeoutStopSec=30

[Install]
WantedBy=default.target
`

const postgresConfigTmpl = `listen_addresses = '127.0.0.1'
port = {{.Port}}
unix_socket_directories = '{{.RunDir}}'
external_pid_file = '{{.PIDFile}}'
logging_collector = off
log_destination = 'stderr'
`

type postgresUnitData struct {
	Version    string
	Binary     string
	DataDir    string
	ConfigPath string
	LogPath    string
}

type postgresConfigData struct {
	RunDir  string
	PIDFile string
	Port    string
}

type PostgresInstance struct {
	Version string
	Unit    string
	DataDir string
}

func Postgres(version string) Definition {
	return PostgresWithPort(version, PostgresDefaultPort)
}

func PostgresWithPort(version, port string) Definition {
	return Definition{
		Name:        "postgres",
		UnitName:    PostgresUnitName(version),
		BinaryName:  PostgresBinaryName,
		ConfigPath:  PostgresConfigPath(version),
		DataDir:     PostgresDataDir(version),
		RenderUnit:  func(binary string) (string, error) { return RenderPostgresUnit(version, binary) },
		WriteConfig: func() error { return WritePostgresConfigWithPort(version, port) },
	}
}

func ValidatePostgresVersion(version string) error {
	return validateVersionLabel("Postgres", version)
}

func PostgresUnitName(version string) string {
	return "routa-postgres@" + version + ".service"
}

func PostgresDataDir(version string) string {
	return filepath.Join(paths.DataDir(), "services", "postgres", version)
}

func PostgresConfigPath(version string) string {
	return filepath.Join(paths.ConfigDir(), "services", "postgres", version, "postgresql.conf")
}

func PostgresPIDFile(version string) string {
	return filepath.Join(paths.RunDir(), "postgres-"+version+".pid")
}

func PostgresLogPath(version string) string {
	return filepath.Join(paths.LogDir(), "postgres-"+version+".log")
}

func RenderPostgresUnit(version, binary string) (string, error) {
	if err := ValidatePostgresVersion(version); err != nil {
		return "", err
	}
	if binary == "" {
		return "", fmt.Errorf("postgres binary path cannot be empty")
	}
	return render("postgres-unit", postgresUnitTmpl, postgresUnitData{
		Version:    version,
		Binary:     binary,
		DataDir:    PostgresDataDir(version),
		ConfigPath: PostgresConfigPath(version),
		LogPath:    PostgresLogPath(version),
	})
}

func RenderPostgresConfig(version string) (string, error) {
	return RenderPostgresConfigWithPort(version, PostgresDefaultPort)
}

func RenderPostgresConfigWithPort(version, port string) (string, error) {
	if err := ValidatePostgresVersion(version); err != nil {
		return "", err
	}
	if err := ValidateTCPPort("Postgres", port); err != nil {
		return "", err
	}
	return render("postgres-config", postgresConfigTmpl, postgresConfigData{
		RunDir:  paths.RunDir(),
		PIDFile: PostgresPIDFile(version),
		Port:    port,
	})
}

func WritePostgresConfig(version string) error {
	return WritePostgresConfigWithPort(version, PostgresDefaultPort)
}

func WritePostgresConfigWithPort(version, port string) error {
	if err := ValidatePostgresVersion(version); err != nil {
		return err
	}
	if err := ValidateTCPPort("Postgres", port); err != nil {
		return err
	}
	for _, dir := range []string{
		PostgresDataDir(version),
		filepath.Dir(PostgresConfigPath(version)),
		paths.RunDir(),
		paths.LogDir(),
	} {
		if err := ensureDir(dir); err != nil {
			return err
		}
	}
	content, err := RenderPostgresConfigWithPort(version, port)
	if err != nil {
		return err
	}
	return os.WriteFile(PostgresConfigPath(version), []byte(content), 0o644)
}

func EnsurePostgres(version string) error {
	return EnsurePostgresWithPort(version, PostgresDefaultPort)
}

func EnsurePostgresWithPort(version, port string) error {
	if err := ValidatePostgresVersion(version); err != nil {
		return err
	}
	if err := ValidateTCPPort("Postgres", port); err != nil {
		return err
	}
	bin, err := FindPostgresBinary(version)
	if err != nil {
		return err
	}
	if err := WriteFiles(PostgresWithPort(version, port), bin); err != nil {
		return err
	}
	if err := initializePostgresDataDir(version, bin); err != nil {
		return err
	}
	return systemd.DaemonReload()
}

func InitializePostgresDataDir(version string) error {
	bin, err := FindPostgresBinary(version)
	if err != nil {
		return err
	}
	return initializePostgresDataDir(version, bin)
}

func initializePostgresDataDir(version, serverBin string) error {
	initialized, err := PostgresDataDirInitialized(version)
	if err != nil {
		return err
	}
	if initialized {
		return nil
	}
	initBin, err := findPostgresInitCommand(version, serverBin)
	if err != nil {
		return err
	}
	cmd := exec.Command(initBin, "-D", PostgresDataDir(version))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func FindPostgresBinary(version string) (string, error) {
	return findPostgresBinaryWith(version, exec.LookPath, commandOutput)
}

func findPostgresBinaryWith(version string, lookPath lookPathFunc, output commandOutputFunc) (string, error) {
	if err := ValidatePostgresVersion(version); err != nil {
		return "", err
	}
	return versionedCommand("Postgres binary", version, postgresBinaryCandidates(version), []string{"--version"}, lookPath, output)
}

func postgresBinaryCandidates(version string) []string {
	major := majorVersionLabel(version)
	return []string{
		"postgres-" + version,
		"postgres" + version,
		filepath.Join("/usr/lib/postgresql", version, "bin", "postgres"),
		filepath.Join("/usr/pgsql-"+version, "bin", "postgres"),
		filepath.Join("/opt/postgresql", version, "bin", "postgres"),
		"postgres-" + major,
		"postgres" + major,
		filepath.Join("/usr/lib/postgresql", major, "bin", "postgres"),
		filepath.Join("/usr/pgsql-"+major, "bin", "postgres"),
		filepath.Join("/opt/postgresql", major, "bin", "postgres"),
		PostgresBinaryName,
	}
}

func findPostgresInitCommand(version, serverBin string) (string, error) {
	candidates := postgresInitCommandCandidates(version, serverBin)
	return versionedCommand("Postgres initdb", version, candidates, []string{"--version"}, exec.LookPath, commandOutput)
}

func postgresInitCommandCandidates(version, serverBin string) []string {
	major := majorVersionLabel(version)
	var out []string
	if serverBin != "" {
		out = append(out, filepath.Join(filepath.Dir(serverBin), "initdb"))
	}
	return append(out,
		"initdb-"+version,
		"initdb"+version,
		filepath.Join("/usr/lib/postgresql", version, "bin", "initdb"),
		filepath.Join("/usr/pgsql-"+version, "bin", "initdb"),
		filepath.Join("/opt/postgresql", version, "bin", "initdb"),
		"initdb-"+major,
		"initdb"+major,
		filepath.Join("/usr/lib/postgresql", major, "bin", "initdb"),
		filepath.Join("/usr/pgsql-"+major, "bin", "initdb"),
		filepath.Join("/opt/postgresql", major, "bin", "initdb"),
		"initdb",
	)
}

func PostgresDataDirInitialized(version string) (bool, error) {
	versionFile := filepath.Join(PostgresDataDir(version), "PG_VERSION")
	info, err := os.Stat(versionFile)
	if err == nil {
		return !info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func InstalledPostgresInstances() ([]PostgresInstance, error) {
	versions := map[string]bool{}
	for _, root := range []string{
		filepath.Join(paths.DataDir(), "services", "postgres"),
		filepath.Join(paths.ConfigDir(), "services", "postgres"),
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
			if err := ValidatePostgresVersion(entry.Name()); err == nil {
				versions[entry.Name()] = true
			}
		}
	}

	for _, unit := range PostgresUnitNamesForUninstall() {
		version := strings.TrimSuffix(strings.TrimPrefix(unit, "routa-postgres@"), ".service")
		versions[version] = true
	}

	out := make([]PostgresInstance, 0, len(versions))
	for version := range versions {
		out = append(out, PostgresInstance{
			Version: version,
			Unit:    PostgresUnitName(version),
			DataDir: PostgresDataDir(version),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func PostgresUnitNamesForUninstall() []string {
	seen := map[string]bool{}
	for _, pattern := range []string{
		filepath.Join(paths.SystemdUserDir(), "routa-postgres@*.service"),
		filepath.Join(paths.SystemdUserDir(), "default.target.wants", "routa-postgres@*.service"),
	} {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			unit := filepath.Base(match)
			version := strings.TrimSuffix(strings.TrimPrefix(unit, "routa-postgres@"), ".service")
			if err := ValidatePostgresVersion(version); err == nil {
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
