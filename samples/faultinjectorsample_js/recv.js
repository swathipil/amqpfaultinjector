// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

/**
 * This sample demonstrates how to create ServiceBusClient that connects to a custom endpoint
 * for receiving messages
 */

import { ServiceBusClient } from "@azure/service-bus";
import { DefaultAzureCredential } from "@azure/identity";

// Load the .env file if it exists
import "dotenv/config";

// Define connection string and related Service Bus entity names here
const fqdn =
  process.env.SERVICEBUS_FQDN ||
  "<your-servicebus-namespace>.servicebus.windows.net";
const queueName = process.env.SERVICEBUS_QUEUE || "<queue name>";

export async function main() {
  const credential = new DefaultAzureCredential();
  const sbClient = new ServiceBusClient(fqdn, credential, {
    customEndpointAddress: "sb://localhost:5671",
  });

  const queueReceiver = sbClient.createReceiver(queueName);

  try {
    let allMessages = [];

    console.log(`Receiving messages...`);

    while (allMessages.length < 10) {
      const messages = await queueReceiver.receiveMessages(10, {
        maxWaitTimeInMs: 60 * 1000,
      });

      if (!messages.length) {
        console.log("No more messages to receive");
        break;
      }

      console.log(`Received ${messages.length} messages`);
      allMessages.push(...messages);

      for (let message of messages) {
        console.log(`  Message: '${message.body}'`);

        await queueReceiver.completeMessage(message);
      }
    }

    await queueReceiver.close();
  } finally {
    await sbClient.close();
  }
}

main().catch(console.error);
