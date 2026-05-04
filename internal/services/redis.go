package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottzirkel/routa/internal/paths"
)

const (
	RedisUnitName     = "routa-redis.service"
	RedisDefaultPort  = "6379"
	RedisDefaultAddr  = "127.0.0.1:6379"
	RedisBinaryName   = "redis-server"
	redisBindHostIPv4 = "127.0.0.1"
)

const redisUnitTmpl = `[Unit]
Description=routa Redis
After=network.target

[Service]
Type=simple
ExecStart={{.Binary}} {{.ConfigPath}}
Restart=on-failure
RestartSec=2
TimeoutStopSec=5

[Install]
WantedBy=default.target
`

const redisConfigTmpl = `bind 127.0.0.1 ::1
protected-mode yes
port {{.Port}}
daemonize no
supervised no
dir {{.DataDir}}
dbfilename dump.rdb
appendonly yes
pidfile {{.PIDFile}}
`

type redisUnitData struct {
	Binary     string
	ConfigPath string
}

type redisConfigData struct {
	DataDir string
	PIDFile string
	Port    string
}

func Redis() Definition {
	return RedisWithPort(RedisDefaultPort)
}

func RedisWithPort(port string) Definition {
	return Definition{
		Name:        "redis",
		UnitName:    RedisUnitName,
		BinaryName:  RedisBinaryName,
		ConfigPath:  RedisConfigPath(),
		DataDir:     RedisDataDir(),
		RenderUnit:  RenderRedisUnit,
		WriteConfig: func() error { return WriteRedisConfigWithPort(port) },
	}
}

func RedisAddr(port string) string {
	return redisBindHostIPv4 + ":" + port
}

func RedisDataDir() string {
	return filepath.Join(paths.DataDir(), "services", "redis")
}

func RedisConfigPath() string {
	return filepath.Join(paths.ConfigDir(), "services", "redis", "redis.conf")
}

func RedisPIDFile() string {
	return filepath.Join(paths.RunDir(), "redis.pid")
}

func RenderRedisUnit(binary string) (string, error) {
	if binary == "" {
		return "", fmt.Errorf("redis binary path cannot be empty")
	}
	return render("redis-unit", redisUnitTmpl, redisUnitData{
		Binary:     binary,
		ConfigPath: RedisConfigPath(),
	})
}

func RenderRedisConfig() (string, error) {
	return RenderRedisConfigWithPort(RedisDefaultPort)
}

func RenderRedisConfigWithPort(port string) (string, error) {
	if err := ValidateRedisPort(port); err != nil {
		return "", err
	}
	return render("redis-config", redisConfigTmpl, redisConfigData{
		DataDir: RedisDataDir(),
		PIDFile: RedisPIDFile(),
		Port:    port,
	})
}

func WriteRedisConfig() error {
	return WriteRedisConfigWithPort(RedisDefaultPort)
}

func WriteRedisConfigWithPort(port string) error {
	if err := ValidateRedisPort(port); err != nil {
		return err
	}
	if err := ensureDir(RedisDataDir()); err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(RedisConfigPath())); err != nil {
		return err
	}
	if err := ensureDir(paths.RunDir()); err != nil {
		return err
	}
	content, err := RenderRedisConfigWithPort(port)
	if err != nil {
		return err
	}
	return os.WriteFile(RedisConfigPath(), []byte(content), 0o644)
}

func RedisConfiguredPort() (string, error) {
	data, err := os.ReadFile(RedisConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return RedisDefaultPort, nil
		}
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if fields[0] != "port" {
			continue
		}
		if len(fields) < 2 {
			return "", fmt.Errorf("invalid Redis port line in %s: %q", RedisConfigPath(), line)
		}
		if err := ValidateRedisPort(fields[1]); err != nil {
			return "", err
		}
		return fields[1], nil
	}
	return RedisDefaultPort, nil
}

func ValidateRedisPort(port string) error {
	return ValidateTCPPort("Redis", port)
}
