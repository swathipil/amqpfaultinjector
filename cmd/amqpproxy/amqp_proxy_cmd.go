package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/Azure/amqpfaultinjector"
)

func main() {
	endpoint := flag.String("h", "", "The hostname to connect to")
	logdir := flag.String("d", ".", "Directory where the amqpproxy-traffic.json and amqpproxy-tlskeys.txt will be created")
	disableTLS := flag.Bool("disable_tls", false, "Disables TLS for the local endpoint ONLY. All traffic is still sent, via TLS, to Azure.")
	disableStateTracking := flag.Bool("disable_state_tracing", false, "Disables state tracing")
	enableBinFiles := flag.Bool("enable_bin_files", false, "Enables writing out amqpproxy-bin files. Note, these files do not redact secrets.")
	flag.Parse()

	if *endpoint == "" {
		fmt.Printf("Hostname must be specified\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Printf("Connect to %s\n", *endpoint)

	var baseBinName string

	if *enableBinFiles {
		baseBinName = path.Join(*logdir, "amqpproxy-bin")
	}

	fi, err := amqpfaultinjector.NewAMQPProxy(
		"localhost:5671",
		*endpoint,
		&amqpfaultinjector.AMQPProxyOptions{
			BaseJSONName:               path.Join(*logdir, "amqpproxy-traffic"),
			TLSKeyLogFile:              path.Join(*logdir, "amqpproxy-tlskeys.txt"),
			BaseBinName:                baseBinName,
			DisableTLSForLocalEndpoint: *disableTLS,
			DisableStateTracing:        *disableStateTracking,
		})

	if err != nil {
		panic(err)
	}

	if err := fi.ListenAndServe(); err != nil {
		panic(err)
	}
}
