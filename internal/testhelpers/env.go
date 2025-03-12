package testhelpers

import (
	"log/slog"
	"os"
	"path"

	"github.com/joho/godotenv"
)

type TestEnv struct {
	ServiceBusEndpoint string
	ServiceBusQueue    string
	LiveTests          bool
}

func LoadEnv(dir string) TestEnv {
	var envs []string

	if dir != "" {
		envs = []string{path.Join(dir, ".env")}
	}

	if err := godotenv.Load(envs...); err != nil {
		slog.Warn("No .env file - live tests will not run")
		return TestEnv{LiveTests: false}
	}

	te := TestEnv{ServiceBusEndpoint: os.Getenv("SERVICEBUS_ENDPOINT"), ServiceBusQueue: os.Getenv("SERVICEBUS_QUEUE")}

	if te.ServiceBusEndpoint == "" || te.ServiceBusQueue == "" {
		slog.Error("SERVICEBUS_ENDPOINT and SERVICEBUS_QUEUE must be defined in the environment")
		return TestEnv{LiveTests: false}
	}

	return te
}
