package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Azure/amqpfaultinjector/internal/testhelpers"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

var testEnv = testhelpers.LoadEnv("../..")

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestFaultInjector_Logging(t *testing.T) {
	t.Run("Send", func(t *testing.T) {
		testData := mustCreateFaultInjector(t, newPassthroughCommand, nil)

		sender, err := testData.ServiceBusClient.NewSender(testData.ServiceBusQueue, nil)
		require.NoError(t, err)

		err = sender.SendMessage(context.Background(), &azservicebus.Message{
			Body: []byte("hello world"),
		}, nil)
		require.NoError(t, err)

		testData.MustClose(t)

		found := false

		for _, logLine := range testhelpers.MustReadJSON(t, testData.JSONLFile) {
			if logLine.EntityPath != testData.ServiceBusQueue {
				continue
			}

			found = true

			// ALL the messages should say false - the direction tells us if it's coming from the service, or the client.
			require.False(t, *logLine.Receiver)
		}

		require.True(t, found)
	})

	t.Run("Receive", func(t *testing.T) {
		testData := mustCreateFaultInjector(t, newPassthroughCommand, nil)

		receiver, err := testData.ServiceBusClient.NewReceiverForQueue(testData.ServiceBusQueue, &azservicebus.ReceiverOptions{
			// TODO: there's a bug (somewhere) when I use this mode with the fault injector where it receives a message
			// without a delivery tag.
			ReceiveMode: azservicebus.ReceiveModeReceiveAndDelete,
		})
		require.NoError(t, err)

		messages, err := receiver.ReceiveMessages(context.Background(), 1, nil)
		require.NoError(t, err)
		require.NotEmpty(t, messages)

		testData.MustClose(t)

		found := false

		for _, logLine := range testhelpers.MustReadJSON(t, testData.JSONLFile) {
			if logLine.EntityPath != testData.ServiceBusQueue {
				continue
			}

			found = true

			// ALL the messages should say false - the direction tells us if it's coming from the service, or the client.
			require.True(t, *logLine.Receiver)
		}

		require.True(t, found)
	})
}

func TestFaultInjector_DetachAfterTransfer(t *testing.T) {
	testData := mustCreateFaultInjector(t, newDetachAfterTransferCommand, []string{"--times", "2"})

	t.Run("sender", func(t *testing.T) {
		{
			sender, err := testData.ServiceBusClient.NewSender(testData.ServiceBusQueue, nil)
			require.NoError(t, err)

			err = sender.SendMessage(context.Background(), &azservicebus.Message{
				Body: []byte("hello world 1"),
			}, nil)
			require.Contains(t, err.Error(), "Detached by the fault injector")

			err = sender.SendMessage(context.Background(), &azservicebus.Message{
				Body: []byte("hello world 2"),
			}, nil)
			require.NoError(t, err)

			err = sender.Close(context.Background())
			require.NoError(t, err)
		}

		testData.MustClose(t)
		testhelpers.ValidateLog(t, testData.JSONLFile)
	})
}

func TestFaultInjector_DetachAfterDelay(t *testing.T) {
	testData := mustCreateFaultInjector(t, newDetachAfterDelayCommand, nil)

	{
		sender, err := testData.ServiceBusClient.NewSender(testData.ServiceBusQueue, nil)
		require.NoError(t, err)

		err = sender.SendMessage(context.Background(), &azservicebus.Message{
			Body: []byte("hello world 1"),
		}, nil)
		require.NoError(t, err)

		// the fault injector will detach us in 2 seconds...
		time.Sleep(3 * time.Second)

		err = sender.Close(context.Background())
		require.ErrorContains(t, err, "Detached by the fault injector")
	}

	testData.MustClose(t)
	testhelpers.ValidateLog(t, testData.JSONLFile)
}

func TestFaultInjector_SlowTransferFrames(t *testing.T) {
	testData := mustCreateFaultInjector(t, newSlowTransferFrames, nil)

	{
		sender, err := testData.ServiceBusClient.NewSender(testData.ServiceBusQueue, nil)
		require.NoError(t, err)

		for i := range 2 {
			err = sender.SendMessage(context.Background(), &azservicebus.Message{
				Body: []byte(fmt.Sprintf("hello world %d", i)),
			}, nil)
			require.NoError(t, err)
		}

		err = sender.Close(context.Background())
		require.NoError(t, err)
	}

	// the default 10s delay is going to make it so we can't receive more than 1 message at a time.
	{
		receiver, err := testData.ServiceBusClient.NewReceiverForQueue(testData.ServiceBusQueue, nil)
		require.NoError(t, err)

		// ensure the receiver is warm - this doesn't cause TRANSFER frames over our link so it won't be
		// affected by the fault injector.
		_, err = receiver.PeekMessages(context.Background(), 1, nil)
		require.NoError(t, err)

		t.Logf("Starting to receive messages - TRANSFERS should start being delayed")
		messages, err := receiver.ReceiveMessages(context.Background(), 100, nil)
		require.NoError(t, err)
		require.Equal(t, 1, len(messages))
	}

	t.Logf("Receiving complete, closing fault injector")
	testData.MustClose(t)
	testhelpers.ValidateLog(t, testData.JSONLFile)
}

type testFaultInjector struct {
	cancelFaultInjector context.CancelFunc
	JSONLFile           string

	ServiceBusEndpoint string
	ServiceBusQueue    string
	ServiceBusClient   *azservicebus.Client
}

func (tfi *testFaultInjector) MustClose(t *testing.T) {
	t.Logf("Stopping fault injector")
	tfi.cancelFaultInjector()

	t.Logf("Closing Service Bus connection")
	require.NoError(t, tfi.ServiceBusClient.Close(context.Background()))
}

func mustCreateFaultInjector(t *testing.T, createCommand func(ctx context.Context) *cobra.Command, args []string) *testFaultInjector {
	dir, err := os.MkdirTemp("", "faultinjector*")
	require.NoError(t, err)

	t.Logf("Temp folder: %s", dir)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	ctx, cancel := context.WithCancel(context.Background())
	subCommand := createCommand(ctx)

	args = append(args,
		subCommand.Name(),
		"--logs", dir,
		"--host", testEnv.ServiceBusEndpoint)

	rootCmd := newRootCommand()
	rootCmd.AddCommand(subCommand)

	t.Logf("Command line args for fault injector: %#v", args)
	rootCmd.SetArgs(args)

	jsonlFile := path.Join(dir, "faultinjector-traffic.json")

	go func() {
		t.Logf("Starting fault injector command")
		require.NoError(t, subCommand.Execute())
		t.Logf("Fault injector command has exited")
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

	tfi := &testFaultInjector{
		cancelFaultInjector: cancel,
		JSONLFile:           jsonlFile,
		ServiceBusEndpoint:  testEnv.ServiceBusEndpoint,
		ServiceBusQueue:     "testqueue",
		ServiceBusClient:    client,
	}

	t.Cleanup(func() {
		tfi.MustClose(t)
	})

	return tfi
}
