package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/richardpark-msft/amqpfaultinjector/cmd/internal"
	"github.com/richardpark-msft/amqpfaultinjector/internal/faultinjectors"
	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/encoding"
	"github.com/spf13/cobra"
)

const addressFileFlagName = "address-file"

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "faultinjector",
	}

	rootCmd.PersistentFlags().String(addressFileFlagName, "", "File to write the address the faultinjector is listening on. If enabled, the faultinjector will start on a random port, instead of 5671.")

	internal.AddCommonFlags(rootCmd)
	return rootCmd
}

func runFaultInjector(ctx context.Context, cmd *cobra.Command, injector faultinjectors.MirrorCallback) error {
	port := 5671

	addressFile, err := cmd.Flags().GetString(addressFileFlagName)

	if err != nil {
		return err
	}

	if addressFile != "" {
		slog.Info("Fault injector will start up on the next free port")
		port = 0
	}

	cf, err := internal.ExtractCommonFlags(cmd)

	if err != nil {
		return err
	}

	fi, err := faultinjectors.NewFaultInjector(
		fmt.Sprintf("localhost:%d", port),
		cf.Host,
		injector,
		&faultinjectors.FaultInjectorOptions{
			JSONLFile:     filepath.Join(cf.LogsDir, "faultinjector-traffic.json"),
			TLSKeyLogFile: filepath.Join(cf.LogsDir, "faultinjector-tlskeys.txt"),
			AddressFile:   addressFile,
			CertDir:       cf.CertDir,
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

func newDetachAfterDelayCommand(ctx context.Context) *cobra.Command {
	var detachAfter *time.Duration
	var detachErrorCond *string
	var detachErrorDesc *string

	cmd := &cobra.Command{
		Use:   "detach_after_delay",
		Short: "Detaches links after a specified delay with a specified error. Useful for exercising recovery code.",
		RunE: func(cmd *cobra.Command, args []string) error {
			injector := faultinjectors.NewDetachAfterDelayInjector(*detachAfter, &encoding.Error{
				Condition:   encoding.ErrCond(*detachErrorCond),
				Description: *detachErrorDesc,
			})

			return runFaultInjector(ctx, cmd, injector.Callback)
		},
	}

	detachAfter = cmd.Flags().Duration("delay", 2*time.Second, "Amount of time to wait, after ATTACH, before initiating DETACH")
	detachErrorCond = cmd.Flags().String("cond", "amqp:link:detach-forced", "AMQP error condition to use for the returned error from DETACH")
	detachErrorDesc = cmd.Flags().String("desc", "Detached by the fault injector", "AMQP error description to use for the returned error from DETACH")

	return cmd
}

func newDetachAfterTransferCommand(ctx context.Context) *cobra.Command {
	var times *int
	var detachErrorCond *string
	var detachErrorDesc *string

	cmd := &cobra.Command{
		Use:   "detach_after_transfer",
		Short: "Detaches AMQP senders after a specified number of TRANSFER frames.",
		RunE: func(cmd *cobra.Command, args []string) error {
			injector := faultinjectors.NewDetachAfterTransferInjector(*times, encoding.Error{
				Condition:   encoding.ErrCond(*detachErrorCond),
				Description: *detachErrorDesc,
			})
			return runFaultInjector(ctx, cmd, injector.Callback)
		},
	}

	times = cmd.Flags().Int("times", 1, "Number of times to DETACH after TRANSFER frames")
	detachErrorCond = cmd.Flags().String("cond", "amqp:link:detach-forced", "AMQP error condition to use for the returned error from DETACH")
	detachErrorDesc = cmd.Flags().String("desc", "Detached by the fault injector", "AMQP error description to use for the returned error from DETACH")

	return cmd
}

func newSlowTransferFrames(ctx context.Context) *cobra.Command {
	var delay *time.Duration

	cmd := &cobra.Command{
		Use:   "transfer_delay",
		Short: "Slows down TRANSFER frames",
		RunE: func(cmd *cobra.Command, args []string) error {
			injector := faultinjectors.NewSlowTransfersInjector(*delay)
			return runFaultInjector(ctx, cmd, injector.Callback)
		},
	}

	delay = cmd.Flags().Duration("delay", 10*time.Second, "Amount of delay to introduce before each TRANSFER frame")
	return cmd
}

// newPassthroughCommand creates a command that passes all frames through, unchanged. Useful if trying to troubleshoot.
func newPassthroughCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "passthrough",
		Short: "Runs the fault injector but passes all frames through. Useful for troubleshooting.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFaultInjector(ctx, cmd, func(ctx context.Context, params faultinjectors.MirrorCallbackParams) ([]faultinjectors.MetaFrame, error) {
				return []faultinjectors.MetaFrame{{
					Action: faultinjectors.MetaFrameActionPassthrough, Frame: params.Frame,
				}}, nil
			})
		},
	}

	return cmd
}
