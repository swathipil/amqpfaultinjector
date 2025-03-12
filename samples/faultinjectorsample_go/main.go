package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/joho/godotenv"
)

const envFilePath = "../../.env"

func main() {
	err := func() error {
		if err := godotenv.Load(envFilePath); err != nil {
			return fmt.Errorf("failed to load .env file from %s: %w", envFilePath, err)
		}

		endpoint := os.Getenv("SERVICEBUS_FULLY_QUALIFIED_NAMESPACE")
		queue := os.Getenv("SERVICEBUS_QUEUE_NAME")

		dac, err := azidentity.NewDefaultAzureCredential(nil)

		if err != nil {
			return err
		}

		// TODO: I'm reversed from what I think other people are doing - the endpoint is the "real"
		// endpoint, and the hostname is supposed to be localhost:5671.
		client, err := azservicebus.NewClient(endpoint, dac, &azservicebus.ClientOptions{
			CustomEndpoint: "localhost:5671",
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		})

		if err != nil {
			return err
		}

		sender, err := client.NewSender(queue, nil)

		if err != nil {
			return err
		}

		err = sender.SendMessage(context.Background(), &azservicebus.Message{
			Body: []byte("hello world!"),
		}, nil)

		return err
	}()

	if err != nil {
		panic(err)
	}
}
