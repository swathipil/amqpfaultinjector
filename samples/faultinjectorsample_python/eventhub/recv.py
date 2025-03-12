#!/usr/bin/env python

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

"""
An example to show receiving events from an Event Hub with checkpoint store.
In the `receive` method of `EventHubConsumerClient`:
If no partition id is specified, the checkpoint_store are used for load-balance and checkpoint.
If partition id is specified, the checkpoint_store can only be used for checkpoint.
"""
import os
import ssl
from dotenv import find_dotenv, load_dotenv
from azure.eventhub import EventHubConsumerClient, TransportType
from azure.identity import DefaultAzureCredential

import logging
handler = logging.FileHandler('out.log', mode='w')
log_fmt = logging.Formatter(fmt='%(asctime)s | %(threadName)s | %(levelname)s | %(name)s | %(message)s')
handler.setFormatter(log_fmt)
logger = logging.getLogger('azure')
logger.setLevel(logging.DEBUG)
logger.addHandler(handler)

find_dotenv()
load_dotenv()

FULLY_QUALIFIED_NAMESPACE = os.environ["EVENT_HUB_HOSTNAME"]
EVENTHUB_NAME = os.environ["EVENT_HUB_NAME"]
CUSTOM_ENDPOINT_ADDRESS = "sb://localhost:5671"

def on_event(partition_context, event):
    # Put your code here.
    # Avoid time-consuming operations.
    print("Received event from partition: {}.".format(partition_context.partition_id))
    print(event)
    partition_context.update_checkpoint(event)

def on_error(partition_context, error):
    # Put your code here.
    print("An exception: {} occurred during receiving from partition: {}.".format(partition_context, error))
    print((type(error)))
    #print("Listing checkpoints: ", checkpoint_store.list_checkpoints(FULLY_QUALIFIED_NAMESPACE, EVENTHUB_NAME, "$Default"))
    #consumer_client.close()

if __name__ == "__main__":
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    context.check_hostname = False
    context.verify_mode = ssl.CERT_NONE

    consumer_client = EventHubConsumerClient(
        fully_qualified_namespace=FULLY_QUALIFIED_NAMESPACE,
        eventhub_name=EVENTHUB_NAME,
        credential=DefaultAzureCredential(),
        consumer_group="$Default",
        logging_enable=True,
        custom_endpoint_address=CUSTOM_ENDPOINT_ADDRESS,
        ssl_context=context,
        transport_type=TransportType.Amqp,
        retry_total=0
    )

    try:
        with consumer_client:
            """
            Without specified partition_id, the receive will try to receive events from all partitions and if provided
            with a checkpoint store, the client will load-balance partition assignment with other EventHubConsumerClient
            instances which also try to receive events from all partitions and use the same storage resource.
            """
            consumer_client.receive(
                on_event=on_event,
                on_error=on_error,
                starting_position="-1",  # "-1" is from the beginning of the partition.
            )
            # With specified partition_id, load-balance will be disabled, for example:
            # client.receive(on_event=on_event, partition_id='0')
    except KeyboardInterrupt:
        print("Stop receiving.")
