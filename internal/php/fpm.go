package php

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/scottzirkel/routa/internal/paths"
	"github.com/scottzirkel/routa/internal/site"
	"github.com/scottzirkel/routa/internal/systemd"
)

// fpm config — one per spec (the spec is what the user wrote: "8.3" or "8.3.30")
// so socket/log/pid paths don't collide between the major.minor instance and
// the patch instance, even though the binary is shared via symlink.
const fpmConfTmpl = `[global]
pid = {{.RunDir}}/php-fpm-{{.Spec}}.pid
error_log = {{.LogDir}}/php-fpm-{{.Spec}}.log
daemonize = no

[www]
listen = {{.RunDir}}/php-fpm-{{.Spec}}.sock
listen.mode = 0666
pm = ondemand
pm.max_children = 16
pm.process_idle_timeout = 60s
pm.max_requests = 500
clear_env = no
catch_workers_output = yes
decorate_workers_output = no
{{range .PoolINISettings}}php_admin_value[{{.Key}}] = {{.Value}}
{{end}}
{{range .SitePools}}
[{{.PoolName}}]
listen = {{.SocketPath}}
listen.mode = 0666
pm = ondemand
pm.max_children = 16
pm.process_idle_timeout = 60s
pm.max_requests = 500
clear_env = no
catch_workers_output = yes
decorate_workers_output = no
{{range $.PoolINISettings}}php_admin_value[{{.Key}}] = {{.Value}}
{{end}}{{range .Env}}{{if .Value}}env[{{.Key}}] = {{fpmQuote .Value}}
{{end}}{{end}}
{{end}}
`

// systemd template — %i is the version spec.
const fpmUnitTmpl = `[Unit]
Description=routa PHP-FPM (%i)
After=network.target

[Service]
Type=simple
ExecStart={{.PHPDir}}/%i/bin/php-fpm --nodaemonize --php-ini {{.RunDir}}/php-fpm-%i.ini --fpm-config {{.RunDir}}/php-fpm-%i.conf
ExecReload=/bin/kill -USR2 $MAINPID
Restart=on-failure
RestartSec=2
KillSignal=SIGQUIT
TimeoutStopSec=5

[Install]
WantedBy=default.target
`

type fpmTmplData struct {
	RunDir          string
	LogDir          string
	Spec            string
	PoolINISettings []INISetting
	SitePools       []SitePool
}

type SitePool struct {
	PoolName   string
	SocketPath string
	Env        []EnvSetting
}

type EnvSetting struct {
	Key   string
	Value string
}

type unitTmplData struct {
	PHPDir string
	RunDir string
}

func WriteFPMConfig(spec string) error {
	if err := os.MkdirAll(paths.RunDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(paths.LogDir(), 0o755); err != nil {
		return err
	}
	settings, err := EffectiveINISettings(spec)
	if err != nil {
		return err
	}
	iniPath := filepath.Join(paths.RunDir(), fmt.Sprintf("php-fpm-%s.ini", spec))
	if err := writeINISettingsFile(iniPath, settings); err != nil {
		return err
	}
	sitePools, err := sitePoolsForSpec(spec)
	if err != nil {
		return err
	}
	t := template.Must(template.New("fpm").Funcs(template.FuncMap{"fpmQuote": fpmQuote}).Parse(fpmConfTmpl))
	dest := filepath.Join(paths.RunDir(), fmt.Sprintf("php-fpm-%s.conf", spec))
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, fpmTmplData{
		RunDir:          paths.RunDir(),
		LogDir:          paths.LogDir(),
		Spec:            spec,
		PoolINISettings: poolINISettings(settings),
		SitePools:       sitePools,
	})
}

func poolINISettings(settings []INISetting) []INISetting {
	out := make([]INISetting, 0, len(settings))
	for _, setting := range settings {
		if setting.Key == ZendExtensionKey {
			continue
		}
		out = append(out, setting)
	}
	return out
}

func RefreshFPMConfigsForSites(sites []site.Resolved) error {
	specs := map[string]bool{}
	for _, resolved := range sites {
		if resolved.Kind == site.KindPHP && resolved.PHP != "" {
			specs[resolved.PHP] = true
		}
	}
	for spec := range specs {
		if err := WriteFPMConfig(spec); err != nil {
			return err
		}
		unit := fmt.Sprintf("routa-php@%s.service", spec)
		if systemd.IsActive(unit) {
			if err := systemd.RunSystemctl("--user", "reload", unit); err != nil {
				return fmt.Errorf("reload %s: %w", unit, err)
			}
		}
	}
	return nil
}

func sitePoolsForSpec(spec string) ([]SitePool, error) {
	st, err := site.Load()
	if err != nil {
		return nil, err
	}
	var pools []SitePool
	for _, resolved := range st.Resolve() {
		if resolved.Kind != site.KindPHP || resolved.PHP != spec || resolved.EnvFile == "" {
			continue
		}
		env, err := LoadEnvFile(resolved.EnvFile)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", resolved.Name, err)
		}
		if len(env) == 0 {
			continue
		}
		pools = append(pools, SitePool{
			PoolName:   "routa-" + resolved.Name,
			SocketPath: site.FPMSocketPath(resolved),
			Env:        env,
		})
	}
	sort.Slice(pools, func(i, j int) bool { return pools[i].PoolName < pools[j].PoolName })
	return pools, nil
}

var envKeyRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var envRefRE = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

func LoadEnvFile(path string) ([]EnvSetting, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	env := map[string]string{}
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("parse %s:%d: expected KEY=value", path, lineNo)
		}
		key = strings.TrimSpace(key)
		if !envKeyRE.MatchString(key) {
			return nil, fmt.Errorf("parse %s:%d: invalid env key %q", path, lineNo, key)
		}
		env[key] = parseEnvValue(strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for key, value := range env {
		env[key] = expandEnvReferences(value, env, map[string]bool{key: true})
	}

	out := make([]EnvSetting, 0, len(env))
	for key, value := range env {
		out = append(out, EnvSetting{Key: key, Value: value})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func parseEnvValue(value string) string {
	if len(value) >= 2 {
		quote := value[0]
		if (quote == '"' || quote == '\'') && value[len(value)-1] == quote {
			value = value[1 : len(value)-1]
		}
	}
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r", ""), "\n", "")
}

func expandEnvReferences(value string, env map[string]string, seen map[string]bool) string {
	return envRefRE.ReplaceAllStringFunc(value, func(match string) string {
		parts := envRefRE.FindStringSubmatch(match)
		key := parts[1]
		if key == "" {
			key = parts[2]
		}
		if seen[key] {
			return match
		}
		if replacement, ok := env[key]; ok {
			seen[key] = true
			expanded := expandEnvReferences(replacement, env, seen)
			delete(seen, key)
			return expanded
		}
		if replacement, ok := os.LookupEnv(key); ok {
			return replacement
		}
		return match
	})
}

func fpmQuote(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`)
	return `"` + replacer.Replace(value) + `"`
}

func EnsureSystemdTemplate() error {
	if err := os.MkdirAll(paths.SystemdUserDir(), 0o755); err != nil {
		return err
	}
	t := template.Must(template.New("u").Parse(fpmUnitTmpl))
	dest := filepath.Join(paths.SystemdUserDir(), "routa-php@.service")
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, unitTmplData{
		PHPDir: paths.PHPDir(),
		RunDir: paths.RunDir(),
	})
}

func SocketPath(spec string) string {
	return filepath.Join(paths.RunDir(), fmt.Sprintf("php-fpm-%s.sock", spec))
}
