import os
import ssl

from azure.eventhub import EventHubProducerClient, TransportType
from azure.identity import AzureCliCredential
from dotenv import find_dotenv, load_dotenv

find_dotenv()
load_dotenv()

FULLY_QUALIFIED_NAMESPACE = os.environ["EVENT_HUB_HOSTNAME"]
EVENTHUB_NAME = os.environ["EVENT_HUB_NAME"]

CUSTOM_ENDPOINT_ADDRESS = "sb://localhost:5671"

context = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
context.check_hostname = False
context.verify_mode = ssl.CERT_NONE

def on_event(partition_context, event):
    # This function will be called when an event is received
    print("Received event from partition: {}".format(partition_context.partition_id))

def get_eventhub_properties():
    client = EventHubProducerClient(
        fully_qualified_namespace=FULLY_QUALIFIED_NAMESPACE,
        eventhub_name=EVENTHUB_NAME,
        credential=AzureCliCredential(),
        custom_endpoint_address=CUSTOM_ENDPOINT_ADDRESS,
        ssl_context=context,
        transport_type=TransportType.Amqp,
    )

    properties = client.get_eventhub_properties()
    print("Event Hub properties: {}".format(properties))

    client.close()

if __name__ == "__main__":
    get_eventhub_properties()