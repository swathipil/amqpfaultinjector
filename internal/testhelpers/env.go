package testhelpers

import (
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/joho/godotenv"
)

type TestEnv struct {
	ServiceBusEndpoint string
	ServiceBusQueue    string
	SkipReason         string
}

func (te TestEnv) SkipIfNotLive(t *testing.T) {
	if te.SkipReason != "" {
		t.Skipf("Skipping test: %s", te.SkipReason)
	}
}

func LoadEnv(dir string) TestEnv {
	envs := []string{path.Join(dir, ".env")}

	if err := godotenv.Load(envs...); err != nil {
		slog.Warn("No .env file - live tests will not run")
		return TestEnv{SkipReason: "No .env file - live tests will not run"}
	}

	te := TestEnv{ServiceBusEndpoint: os.Getenv("SERVICEBUS_ENDPOINT"), ServiceBusQueue: os.Getenv("SERVICEBUS_QUEUE")}

	if te.ServiceBusEndpoint == "" || te.ServiceBusQueue == "" {
		slog.Error("SERVICEBUS_ENDPOINT and SERVICEBUS_QUEUE must be defined in the environment")
		return TestEnv{SkipReason: "SERVICEBUS_ENDPOINT and SERVICEBUS_QUEUE must be defined in the environment"}
	}

	return te
}
