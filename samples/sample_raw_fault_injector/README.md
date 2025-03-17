# Sample Raw Fault Injector

This is a simple self-contained example of a fault injector. You can use this as a starting point for writing your own fault injector or to test out ideas.
NOTE: you'll be editing this project, so some basic Go language knowledge is required.

## Running

### Launching the fault injector

1. Install Go 1.24+.
1. Clone this repo: `git clone https://github.com/richardpark-msft/amqpfaultinjector`
1. Open a terminal: `cd samples/sample_raw_fault_injector`
1. Compile and run this app: `go run . --host <your servicebus/eventhubs namespace, like sb.servicebus.windows.net>`. The fault injector will stay persistent until you kill it.
1. Look in [main.go](./main.go), in the `injectorCallback` function for more on how to change this app, and what is possible within the fault injection framework.

### Launching your test program

Out of the box, Azure clients will not connect to the proxy, so you'll have to do some configuration. The faultinjector* samples contain examples for various languages, which you can adapt.
