package main

import (
	"os"
	"testing"

	"github.com/Azure/amqpfaultinjector/internal/testhelpers"
)

var testEnv = testhelpers.LoadEnv("../..")

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
