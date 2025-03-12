package main

import (
	"context"
	"fmt"
	"log/slog"
	"path"

	"github.com/Azure/amqpfaultinjector/internal/faultinjectors"
	"github.com/Azure/amqpfaultinjector/internal/logging"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := newRootCommand()

	// detach commands
	rootCmd.AddCommand(newDetachAfterDelayCommand(context.Background()))
	rootCmd.AddCommand(newDetachAfterTransferCommand(context.Background()))

	// transfer commands
	rootCmd.AddCommand(newSlowTransferFrames(context.Background()))

	// passthrough/diagnostics
	rootCmd.AddCommand(newPassthroughCommand(context.Background()))

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Failed to run command", "error", err)
	}
}

const hostFlagName = "host"
const addressFileFlagName = "address-file"
const logsFlagName = "logs"

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "faultinjector",
	}

	rootCmd.PersistentFlags().String(hostFlagName, "", "The hostname of the service we're proxying to (ex: <server>.servicebus.windows.net")
	rootCmd.PersistentFlags().String(logsFlagName, ".", "The directory to write any logs or trace files")
	rootCmd.PersistentFlags().String(addressFileFlagName, "", "File to write the address the fault injector is listening on. If enabled, the fault injector will start on a random port, instead of 5671.")
	_ = rootCmd.MarkPersistentFlagRequired(hostFlagName)

	return rootCmd
}

func runInjectorCommand(ctx context.Context, cmd *cobra.Command, injector faultinjectors.MirrorCallback) error {
	hostname, err := cmd.Flags().GetString(hostFlagName)

	if err != nil {
		return err
	}

	addressFile, err := cmd.Flags().GetString(addressFileFlagName)

	if err != nil {
		return err
	}

	logsDir, err := cmd.Flags().GetString(logsFlagName)

	if err != nil {
		return err
	}

	port := 5671

	if addressFile != "" {
		slog.Info("Fault injector will start up on the next free port")
		port = 0
	}

	fi, err := faultinjectors.NewFaultInjector(
		fmt.Sprintf("localhost:%d", port),
		hostname,
		injector,
		&faultinjectors.FaultInjectorOptions{
			JSONLFile:     path.Join(logsDir, "faultinjector-traffic.json"),
			TLSKeyLogFile: path.Join(logsDir, "faultinjector-tlskeys.txt"),
			AddressFile:   addressFile,
		})

	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()

		slogger := logging.SloggerFromContext(ctx)

		slogger.Info("Cancellation received, closing fault injector")
		if err := fi.Close(); err != nil {
			slogger.Error("failed when closing the fault injector", "error", err)
		}
	}()

	return fi.ListenAndServe()
}
