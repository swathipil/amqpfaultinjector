#!/usr/bin/env python

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

"""
Examples to show sending events with different options to an Event Hub partition.
"""

import time
import os
import ssl
from dotenv import find_dotenv, load_dotenv

from azure.eventhub import EventHubProducerClient, EventData, TransportType
from azure.eventhub.exceptions import EventHubError
from azure.identity import DefaultAzureCredential

import logging
handler = logging.FileHandler('out.log', mode='w')
log_fmt = logging.Formatter(fmt='%(asctime)s | %(threadName)s | %(levelname)s | %(name)s | %(message)s')
handler.setFormatter(log_fmt)
logger = logging.getLogger('azure.eventhub')
logger.setLevel(logging.DEBUG)
logger.addHandler(handler)

load_dotenv(find_dotenv())

FULLY_QUALIFIED_NAMESPACE = os.environ["EVENT_HUB_HOSTNAME"]
EVENTHUB_NAME = os.environ["EVENT_HUB_NAME"]

def send_event_data_batch(producer):
    # Without specifying partition_id or partition_key
    # the events will be distributed to available partitions via round-robin.
    event_data_batch = producer.create_batch()
    for _ in range(1):
        event_data_batch.add(EventData("Hello world!"))
    producer.send_batch(event_data_batch)

CUSTOM_ENDPOINT_ADDRESS = "sb://localhost:5671"

context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
context.check_hostname = False
context.verify_mode = ssl.CERT_NONE

producer = EventHubProducerClient(
    fully_qualified_namespace=FULLY_QUALIFIED_NAMESPACE,
    eventhub_name=EVENTHUB_NAME,
    credential=DefaultAzureCredential(),
    custom_endpoint_address=CUSTOM_ENDPOINT_ADDRESS,
    ssl_context=context,
    transport_type=TransportType.Amqp,
)

start_time = time.time()
with producer:
    send_event_data_batch(producer)

print("Send messages in {} seconds.".format(time.time() - start_time))
