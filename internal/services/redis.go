package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/scottzirkel/routa/internal/paths"
)

const RedisUnitName = "routa-redis.service"

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
port 6379
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
}

func Redis() Definition {
	return Definition{
		Name:        "redis",
		UnitName:    RedisUnitName,
		BinaryName:  "redis-server",
		ConfigPath:  RedisConfigPath(),
		DataDir:     RedisDataDir(),
		RenderUnit:  RenderRedisUnit,
		WriteConfig: WriteRedisConfig,
	}
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
	return render("redis-config", redisConfigTmpl, redisConfigData{
		DataDir: RedisDataDir(),
		PIDFile: RedisPIDFile(),
	})
}

func WriteRedisConfig() error {
	if err := ensureDir(RedisDataDir()); err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(RedisConfigPath())); err != nil {
		return err
	}
	if err := ensureDir(paths.RunDir()); err != nil {
		return err
	}
	content, err := RenderRedisConfig()
	if err != nil {
		return err
	}
	return os.WriteFile(RedisConfigPath(), []byte(content), 0o644)
}
