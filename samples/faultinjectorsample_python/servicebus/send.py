#!/usr/bin/env python

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

"""
Examples to show how to create async EventHubProducerClient and EventHubConsumerClient that connect to custom endpoint.
"""

import os
from dotenv import find_dotenv, load_dotenv
import ssl
from azure.servicebus import ServiceBusClient, ServiceBusMessage, TransportType
from azure.identity import DefaultAzureCredential


import logging
handler = logging.FileHandler('out.log', mode='w')
log_fmt = logging.Formatter(
    fmt='%(asctime)s | %(threadName)s | %(levelname)s | %(name)s | %(message)s')
handler.setFormatter(log_fmt)
logger = logging.getLogger('azure.servicebus')
logger.setLevel(logging.DEBUG)
logger.addHandler(handler)


find_dotenv()
load_dotenv()
FULLY_QUALIFIED_NAMESPACE = os.environ["SERVICEBUS_ENDPOINT"]
QUEUE_NAME = os.environ["SERVICEBUS_QUEUE"]

print(f"Connecting to {FULLY_QUALIFIED_NAMESPACE}, using queue {QUEUE_NAME}")

# The custom endpoint address to use for establishing a connection to the Service Bus service,
# allowing network requests to be routed through any application gateways
# or other paths needed for the host environment.
CUSTOM_ENDPOINT_ADDRESS = "sb://localhost:5671"

context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
context.check_hostname = False
context.verify_mode = ssl.CERT_NONE

def send_single_message(sender):
    message = ServiceBusMessage("Single Message")
    sender.send_messages(message)

def send_large_messages(sender):
    for _ in range(200):
        message = "A" * 1024 * 600 # 600 KB
        sender.send_messages(ServiceBusMessage(message))

credential = DefaultAzureCredential()
servicebus_client = ServiceBusClient(
    FULLY_QUALIFIED_NAMESPACE,
    credential,
    logging_enable=True,
    custom_endpoint_address=CUSTOM_ENDPOINT_ADDRESS,
    ssl_context=context,
    transport_type=TransportType.Amqp,
)
with servicebus_client:
    sender = servicebus_client.get_queue_sender(queue_name=QUEUE_NAME)
    with sender:
        send_single_message(sender)
        # Namespace should be premium to send messages > 256 KB
        # send_large_messages(sender)

print('Finished sending.')
