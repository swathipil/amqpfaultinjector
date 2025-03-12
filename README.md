# !!NOTE!!: this project is in an _alpha_ state

# Running

## From binaries

Look in the [releases](https://github.com/richardpark-msft/amqpfaultinjector/releases) for AMQP proxy and fault injector binaries. Each binary
is self-contained, and provides help if run from the command line with no arguments, with with `-h`.

## From Source

For frame capture, run the amqpproxy in parallel with your sample.

### Install Go

- Download and install the latest Go from https://go.dev/dl.
  - Download the latest tar.gz for your OS under "Featured Downloads".
  - Extract the tar.gz file:
    ```
    tar -C /usr/local -xzf go1.x.x.linux-amd64.tar.gz
    ```
  - Add Go to your PATH:
    ```
    export PATH=$PATH:/usr/local/go/bin
    ```
- Verify the installation:
  ```
  go version
  ```

### Run the AMQP proxy

1. Run `amqpproxy` with the desired endpoint:

    ```sh
    cd cmd/amqpproxy
    go run . --host <hostname>
    ```

    Replace `<hostname>` with the hostname of the remote AMQP server you want to connect to. (ex. `"<your-namespace>.servicebus.windows.net"`)

2. Run your sample in parallel to the proxy, which will log frames to the `amqpproxy-traffic-1.json` file located in `cmd/amqpproxy`. Prior logs will be moved to the `amqpproxy-traffic-2.json` and so on. You can find samples in the `samples` folder.
3. To stop the proxy, Ctrl+C.


### Run a fault injector scenario

1. Run the desired fault injection scenario with host specified:

  ```sh
  cd cmd/faultinjector
  go run . <scenario> --host <hostname>
  ```
  Replace `<hostname>` with the hostname of the remote AMQP server you want to connect to. (ex. `"<your-namespace>.servicebus.windows.net"`)

2. For a list of all supported injection scenarios, run:

  ```sh
  cd cmd/faultinjector
  go run . --help
  ```
