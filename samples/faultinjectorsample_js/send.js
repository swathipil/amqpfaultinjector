// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

/**
 * This sample demonstrates how to create ServiceBusClient that connects to a custom endpoint
 * for sending messages
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

async function main() {
  const credential = new DefaultAzureCredential();
  const sbClient = new ServiceBusClient(fqdn, credential, {
    customEndpointAddress: "sb://localhost:5671",
  });

  const sender = sbClient.createSender(queueName);

  try {
    // Send a single message
    console.log(`Sending one scientists`);
    const message = {
      contentType: "application/json",
      subject: "Scientist",
      body: { firstName: "Albert", lastName: "Einstein" },
      timeToLive: 2 * 60 * 1000, // message expires in 2 minutes
    };
    await sender.sendMessages(message);

    // Close the sender
    console.log(`Done sending, closing...`);
    await sender.close();
  } finally {
    await sbClient.close();
  }
}

main().catch(console.error);
