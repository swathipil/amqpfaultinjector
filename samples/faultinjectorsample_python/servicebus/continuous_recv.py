#!/usr/bin/env python

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------

"""
Example to show sending message(s) to a Service Bus Queue.
"""

import os
from azure.servicebus import ServiceBusClient, ServiceBusMessage, ServiceBusReceiveMode, TransportType
from azure.identity import AzureCliCredential
from dotenv import load_dotenv, find_dotenv
load_dotenv(find_dotenv())

import logging
import sys
handler = logging.FileHandler('out.log', mode='w')
log_fmt = logging.Formatter(fmt='%(asctime)s | %(threadName)s | %(levelname)s | %(name)s | %(message)s')
handler.setFormatter(log_fmt)
logger = logging.getLogger('azure.servicebus')
logger.setLevel(logging.DEBUG)
logger.addHandler(handler)

FULLY_QUALIFIED_NAMESPACE = os.environ["SERVICEBUS_FULLY_QUALIFIED_NAMESPACE"]
QUEUE_NAME = os.environ["SERVICEBUS_QUEUE_NAME"]


def send_single_message(sender):
    message = ServiceBusMessage("a"*1000)
    sender.send_messages(message)


def send_a_list_of_messages(sender):
    messages = [ServiceBusMessage("a"*1000) for _ in range(1000)]
    sender.send_messages(messages)


def send_batch_message(sender):
    for _ in range(1000):
        batch_message = sender.create_message_batch()
        for _ in range(10):
            try:
                batch_message.add_message(ServiceBusMessage("Message inside a ServiceBusMessageBatch" * 100))
            except ValueError:
                # ServiceBusMessageBatch object reaches max_size.
                # New ServiceBusMessageBatch object can be created here to send more data.
                break
        sender.send_messages(batch_message)


import ssl
CUSTOM_ENDPOINT_ADDRESS = "sb://localhost:5671"

context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
context.check_hostname = False
context.verify_mode = ssl.CERT_NONE

credential = AzureCliCredential()
servicebus_client = ServiceBusClient(
    FULLY_QUALIFIED_NAMESPACE, credential, logging_enable=True,
    custom_endpoint_address=CUSTOM_ENDPOINT_ADDRESS,
    ssl_context=context,
    transport_type=TransportType.Amqp
)
with servicebus_client:
    sender = servicebus_client.get_queue_sender(queue_name=QUEUE_NAME)
    receiver = servicebus_client.get_queue_receiver(queue_name=QUEUE_NAME, receive_mode=ServiceBusReceiveMode.RECEIVE_AND_DELETE)

    with sender, receiver:
        # ensure 1000 messages are sent before receiving
        #send_single_message(sender)
        #send_a_list_of_messages(sender)
        send_batch_message(sender)
        print("Send a list of 1000 messages is done.")

        for i in range(10):
            msgs = receiver.receive_messages(max_message_count=100)
            print(f"Received {len(msgs)} messages.")
  #      
  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      
  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      
  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")

  #      send_a_list_of_messages(sender)
  #      print("Send a list of 100 messages is done.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")
  #      msgs = receiver.receive_messages(max_message_count=100)
  #      print(f"Received {len(msgs)} messages.")


print("Send message is done.")