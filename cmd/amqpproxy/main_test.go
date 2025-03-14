package main

import (
	"context"
	"crypto/tls"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Azure/amqpfaultinjector/internal/testhelpers"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/stretchr/testify/require"
)

var testEnv = testhelpers.LoadEnv("../..")

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestAMQPProxy(t *testing.T) {
	testData := mustCreateAMQPProxy(t, []string{})

	receiver, err := testData.ServiceBusClient.NewReceiverForQueue(testData.ServiceBusQueue, nil)
	require.NoError(t, err)

	defer func() {
		err := receiver.Close(context.Background())
		require.NoError(t, err)
	}()

	sender, err := testData.ServiceBusClient.NewSender(testData.ServiceBusQueue, nil)
	require.NoError(t, err)

	defer func() {
		err := sender.Close(context.Background())
		require.NoError(t, err)
	}()

	err = sender.SendMessage(context.Background(), &azservicebus.Message{Body: []byte("hello world")}, nil)
	require.NoError(t, err)

	batch, err := sender.NewMessageBatch(context.Background(), nil)
	require.NoError(t, err)

	extraData := make([]byte, 2048)
	err = batch.AddMessage(&azservicebus.Message{Body: extraData}, nil)
	require.NoError(t, err)

	err = sender.SendMessageBatch(context.Background(), batch, nil)
	require.NoError(t, err)

	messages, err := receiver.ReceiveMessages(context.Background(), 2, nil)
	require.NoError(t, err)
	require.NotEmpty(t, messages)

	for _, m := range messages {
		err = receiver.CompleteMessage(context.Background(), m, nil)
		require.NoError(t, err)
	}

	testhelpers.ValidateLog(t, testData.JSONLFile)
}

type testAMQPProxy struct {
	cancelAMQPProxy context.CancelFunc
	JSONLFile       string

	ServiceBusEndpoint string
	ServiceBusQueue    string
	ServiceBusClient   *azservicebus.Client
}

func mustCreateAMQPProxy(t *testing.T, args []string) *testAMQPProxy {
	dir, err := os.MkdirTemp("", "amqpproxy*")
	require.NoError(t, err)

	t.Logf("Temp folder: %s", dir)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := newAMQPProxyCommand(ctx)

	args = append(args,
		cmd.Name(),
		"--logs", dir,
		"--host", testEnv.ServiceBusEndpoint)

	t.Logf("Command line args for fault injector: %#v", args)
	cmd.SetArgs(args)

	jsonlFile := path.Join(dir, "amqpproxy-traffic-1.json") // note, we're assuming this test only creates a single connection

	go func() {
		t.Logf("Starting AMQP proxy command")
		require.NoError(t, cmd.Execute())
		t.Logf("AMQP Proxy has exited")
	}()

	time.Sleep(5 * time.Second)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	require.NoError(t, err)

	client, err := azservicebus.NewClient(testEnv.ServiceBusEndpoint, cred, &azservicebus.ClientOptions{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		CustomEndpoint: "127.0.0.1:5671",
		RetryOptions: azservicebus.RetryOptions{
			MaxRetries: -1,
		},
	})
	require.NoError(t, err)

	tfi := &testAMQPProxy{
		cancelAMQPProxy:    cancel,
		JSONLFile:          jsonlFile,
		ServiceBusEndpoint: testEnv.ServiceBusEndpoint,
		ServiceBusQueue:    testEnv.ServiceBusQueue,
		ServiceBusClient:   client,
	}

	t.Cleanup(func() {
		tfi.MustClose(t)
	})

	return tfi
}

func (tfi *testAMQPProxy) MustClose(t *testing.T) {
	t.Logf("Stopping AMQP Proxy")
	tfi.cancelAMQPProxy()

	t.Logf("Closing Service Bus connection")
	require.NoError(t, tfi.ServiceBusClient.Close(context.Background()))
}
