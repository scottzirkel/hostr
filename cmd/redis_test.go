package cmd

import (
	"strings"
	"testing"

	"github.com/scottzirkel/routa/internal/services"
)

func TestRedisStartedMessageIncludesAddress(t *testing.T) {
	got := redisStartedMessage("6380")

	for _, want := range []string{
		"started " + services.RedisUnitName,
		services.RedisAddr("6380"),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("message missing %q: %q", want, got)
		}
	}
}

func TestRedisStatusHeaderIncludesAddress(t *testing.T) {
	got := redisStatusHeader("6380")

	for _, want := range []string{
		services.RedisUnitName,
		services.RedisAddr("6380"),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("status header missing %q: %q", want, got)
		}
	}
}

func TestRedisPortConflictMessageIncludesAddress(t *testing.T) {
	got := redisPortConflictError("6380").Error()

	if !strings.Contains(got, services.RedisAddr("6380")) {
		t.Fatalf("conflict message missing address: %q", got)
	}
}

func TestRedisPortFromCommandRejectsFlagAndOnAlias(t *testing.T) {
	redisStartPort = "6380"
	redisStartCmd.Flags().Set("port", "6380")
	t.Cleanup(func() {
		redisStartCmd.Flags().Set("port", services.RedisDefaultPort)
		redisStartPort = services.RedisDefaultPort
	})

	if _, err := redisPortFromCommand(redisStartCmd, []string{"on", "6381"}, redisStartPort, services.RedisDefaultPort); err == nil {
		t.Fatal("expected error")
	}
}
