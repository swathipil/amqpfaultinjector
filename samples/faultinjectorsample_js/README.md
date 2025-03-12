# JavaScript Samples

JavaScript setup and samples for common scenarios.

## Setup JavaScript environment

### Install NodeJS

- Download and install [LTS versions of Node.js](https://github.com/nodejs/release#release-schedule).

### Install dependencies

1. Install the packages required to run the samples using `npm`:

   ```sh
   npm install
   ```

## Run amqpproxy

Run the amqpproxy in parallel with the sample to capture frames. Instructions in main README.

## Running the samples

Two samples are provided:

- `send.js`
- `recv.js`

Find more samples [here](https://github.com/Azure/azure-sdk-for-js/tree/main/sdk/servicebus/service-bus/samples).

1. Add the following to a `.env` file under this directory:

```
SERVICEBUS_FQDN=<your-namespace>.servicebus.windows.net
SERVICEBUS_QUEUE=<your-queue-name>

# disable NodeJS error about self-issued certs
NODE_TLS_REJECT_UNAUTHORIZED=0
```

2. In the terminal, run the sample:

```sh
cd samples/faultinjectorsample_js
node send.js
```

Frames will be output to the `cmd/amqpproxy/test.json` file.
