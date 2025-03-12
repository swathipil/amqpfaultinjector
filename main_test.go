package amqpfaultinjector_test

import (
	"os"
	"testing"

	"log/slog"

	"github.com/joho/godotenv"
)

var (
	liveTests          bool
	serviceBusEndpoint string
	serviceBusQueue    string
)

func TestMain(m *testing.M) {
	ec := func() int {
		slog.SetLogLoggerLevel(slog.LevelDebug)

		if err := godotenv.Load(); err != nil {
			liveTests = false
			slog.Warn("No .env file - live tests will not run")
			return 0
		}

		serviceBusEndpoint = os.Getenv("SERVICEBUS_ENDPOINT")
		serviceBusQueue = os.Getenv("SERVICEBUS_QUEUE")

		if serviceBusEndpoint == "" || serviceBusQueue == "" {
			slog.Error("SERVICEBUS_ENDPOINT and SERVICEBUS_QUEUE must be defined in the environment")
			return 1
		}

		return m.Run()
	}()

	os.Exit(ec)
}
