#!/usr/bin/env python

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

import os
from dotenv import find_dotenv, load_dotenv
import ssl
from azure.servicebus import ServiceBusClient, TransportType
from azure.identity import DefaultAzureCredential


import logging
handler = logging.FileHandler('out.log', mode='w')
log_fmt = logging.Formatter(fmt='%(asctime)s | %(threadName)s | %(levelname)s | %(name)s | %(message)s')
handler.setFormatter(log_fmt)
logger = logging.getLogger('azure.servicebus')
logger.setLevel(logging.DEBUG)
logger.addHandler(handler)


find_dotenv()
load_dotenv()
FULLY_QUALIFIED_NAMESPACE = os.environ["SERVICEBUS_ENDPOINT"]
QUEUE_NAME = os.environ["SERVICEBUS_QUEUE"]
# The custom endpoint address to use for establishing a connection to the Service Bus service,
# allowing network requests to be routed through any application gateways
# or other paths needed for the host environment.
CUSTOM_ENDPOINT_ADDRESS = "sb://localhost:5671"

context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
context.check_hostname = False
context.verify_mode = ssl.CERT_NONE

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

    receiver = servicebus_client.get_queue_receiver(queue_name=QUEUE_NAME)
    with receiver:
        received_msgs = receiver.receive_messages(max_message_count=1)
        print('Received messages.')
        for msg in received_msgs:
            print(msg)
            receiver.complete_message(msg)
        print('Completed messages.')
