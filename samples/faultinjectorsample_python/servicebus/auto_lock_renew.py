#!/usr/bin/env python

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

"""
Example to show usage of AutoLockRenewer:
    1. Automatically renew locks on messages received from non-sessionful entity
    2. Automatically renew locks on the session of sessionful entity

We do not guarantee that this SDK is thread-safe. We do not recommend reusing the ServiceBusClient,
 ServiceBusSender, ServiceBusReceiver across threads. It is up to the running 
 application to use these classes in a thread-safe manner.
"""

import os
import time
import ssl
from dotenv import find_dotenv, load_dotenv

from azure.servicebus import ServiceBusClient, AutoLockRenewer, ServiceBusMessage, TransportType
from azure.servicebus.exceptions import ServiceBusError
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
FULLY_QUALIFIED_NAMESPACE = os.environ["SERVICEBUS_FULLY_QUALIFIED_NAMESPACE"]
QUEUE_NAME = os.environ["SERVICEBUS_QUEUE_NAME"]
CUSTOM_ENDPOINT_ADDRESS = "sb://localhost:5671"


def renew_lock_on_message_received_from_non_sessionful_entity():
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
        with servicebus_client.get_queue_sender(queue_name=QUEUE_NAME) as sender:
            msgs_to_send = [ServiceBusMessage("message: {}".format(i)) for i in range(10)]
            sender.send_messages(msgs_to_send)
            print("Send messages to non-sessionful queue.")

        # Can also be called via "with AutoLockRenewer() as renewer" to automate shutdown.
        renewer = AutoLockRenewer()

        with servicebus_client.get_queue_receiver(queue_name=QUEUE_NAME, prefetch_count=10) as receiver:
            received_msgs = receiver.receive_messages(max_message_count=10, max_wait_time=5)

            for msg in received_msgs:
                # automatically renew the lock on each message for 100 seconds
                renewer.register(receiver, msg, max_lock_renewal_duration=100)
            print("Register messages into AutoLockRenewer done.")

            time.sleep(100)  # message handling for long period (E.g. application logic)

            for msg in received_msgs:
                receiver.complete_message(msg)  # Settling the message deregisters it from the AutoLockRenewer
            print("Complete messages.")

        renewer.close()


renew_lock_on_message_received_from_non_sessionful_entity()
