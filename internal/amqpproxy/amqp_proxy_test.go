package amqpproxy_test

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/richardpark-msft/amqpfaultinjector/internal/amqpproxy"
	"github.com/richardpark-msft/amqpfaultinjector/internal/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestAMQPProxy(t *testing.T) {
	testData := mustCreateAMQPProxy(t)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	require.NoError(t, err)

	client, err := azservicebus.NewClient(testData.ServiceBusEndpoint, cred, &azservicebus.ClientOptions{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		CustomEndpoint: "127.0.0.1:5671",
		RetryOptions: azservicebus.RetryOptions{
			MaxRetries: -1,
		},
	})
	require.NoError(t, err)

	sender, err := client.NewSender(testData.ServiceBusQueue, nil)
	require.NoError(t, err)

	err = sender.SendMessage(context.Background(), &azservicebus.Message{
		Body: []byte("hello world"),
	}, nil)
	require.NoError(t, err)

	require.NoError(t, sender.Close(context.Background()))

	receiver, err := client.NewReceiverForQueue(testData.ServiceBusQueue, nil)
	require.NoError(t, err)

	messages, err := receiver.ReceiveMessages(context.Background(), 1, nil)
	require.NoError(t, err)
	require.NotEmpty(t, messages)

	for _, m := range messages {
		err = receiver.CompleteMessage(context.Background(), m, nil)
		require.NoError(t, err)
	}

	require.NoError(t, receiver.Close(context.Background()))
	require.NoError(t, testData.Close())

	testhelpers.ValidateLog(t, testData.JSONLFile+"-1.json")
}

type testAMQPProxy struct {
	*amqpproxy.AMQPProxy
	JSONLFile          string
	ServiceBusEndpoint string
	ServiceBusQueue    string
}

func mustCreateAMQPProxy(t *testing.T) testAMQPProxy {
	dir, err := os.MkdirTemp("", "amqpproxy*")
	require.NoError(t, err)

	t.Logf("Temp folder: %s", dir)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	jsonlFile := filepath.Join(dir, "amqpproxy-traffic")

	env := testhelpers.InitLiveTests("../..")

	amqpProxy, err := amqpproxy.NewAMQPProxy(
		"localhost:5671",
		env.ServiceBusEndpoint,
		&amqpproxy.AMQPProxyOptions{
			BaseJSONName: jsonlFile,
			CertDir:      dir,
		})
	require.NoError(t, err)

	go func() {
		require.NoError(t, amqpProxy.ListenAndServe())
	}()

	time.Sleep(5 * time.Second)

	return testAMQPProxy{
		AMQPProxy:          amqpProxy,
		JSONLFile:          jsonlFile,
		ServiceBusEndpoint: env.ServiceBusEndpoint,
		ServiceBusQueue:    env.ServiceBusQueue,
	}
}
