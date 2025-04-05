# Python Samples

Python setup and samples for common scenarios.

## Setup Python environment

### Install Python

- Download and install latest Python 3.9+ from [python.org](https://www.python.org/downloads/).

### Create a virtual environment

1. Open a terminal or command prompt.
2. Navigate to your project directory.
3. Run the following command to create a virtual environment:

    ```sh
    python -m venv my-venv
    ```
4. Activate the virtual environment:
   - On Windows:
    ```sh
    .\my-venv\Scripts\activate
    ```
   - On macOS and Linux:
    ```sh
    source venv/bin/activate
    ```

### Install dependencies

1. Ensure you are in the virtual environment (step 4 above).
2. Install the packages required to run the samples using `pip`:

    ```sh
    pip install -r requirements.txt
    ```

### (Optional) VSCode

Set up the following for Intellisense.

1. If using Visual Studio Code, install the 'Python' extension.
2. Open the command palette in VSCode: Ctrl+Shift+P
  2a. `>Python: Select Interpreter`
  2b. `Enter interpreter path...`(ex. `/my-venv/bin/python3`)

## Run amqpproxy

Run the amqpproxy in parallel with the sample to capture frames. Instructions in main README.

## Running the samples

Two samples are provided:
- `send.py`
- `recv.py`

Find more samples [here](https://github.com/Azure/azure-sdk-for-python/tree/main/sdk/servicebus/azure-servicebus/samples).

1. Create your resources.
  - Follow this [Service Bus guide](https://github.com/Azure/azure-sdk-for-python/tree/main/sdk/servicebus/azure-servicebus#prerequisites) to create Service Bus resources. See [this](https://github.com/Azure/azure-sdk-for-python/tree/main/sdk/servicebus/azure-servicebus#create-client-using-the-azure-identity-library) for more info about authentication using Azure Authentication Directory.
  - Follow this [Event Hubs guide](https://github.com/Azure/azure-sdk-for-python/tree/main/sdk/eventhub/azure-eventhub#prerequisites) to create Event Hubs resources. Refer to [this](https://github.com/Azure/azure-sdk-for-python/tree/main/sdk/eventhub/azure-eventhub#authenticate-the-client) for info on authentication using Azure Authentication Directory.
  - When using Azure Active Directory for authentication, ensure that you have assigned a role to your principal which allows access to Service Bus, such as the [Service Bus Data Owner role](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/integration#azure-service-bus-data-owner). Otherwise, you may see a `Unauthorized access. 'Send' and 'Listen' claim(s) are required to perform this operation` error.

2. Add the following to a `.env` file:
```
SERVICEBUS_ENDPOINT=<your-sb-namespace>.servicebus.windows.net
SERVICEBUS_QUEUE=<your-queue-name>
EVENT_HUB_HOSTNAME=<your-eh-namespace>.servicebus.windows.net
EVENT_HUB_NAME=<your-hub>
```
3. In the terminal, run the sample:
```sh
cd samples/faultinjectorsample_python/servicebus
python send.py
```
OR
```sh
cd samples/faultinjectorsample_python/eventhub
python send.py
```

If running the AMQP proxy, frames will be output to the `cmd/amqpproxy/amqpproxy-traffic-1.json` file. This file will be updated upon every re-run of the proxy and previous frame logs will be moved to `amqpproxy-traffic-2.json` and so on.
If running the fault injector, frames will be output to the `cmd/faultinjector/faultinjector-traffic.json` file.
