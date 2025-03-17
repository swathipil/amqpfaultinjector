// A simple example of making your own fault injector command - see the

package main

import (
	"context"
	"log/slog"
	"path"

	"github.com/Azure/amqpfaultinjector/internal/faultinjectors"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/Azure/amqpfaultinjector/internal/utils"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{}

	host := cmd.Flags().String("host", "", "The hostname of the service we're proxying to (ex: <server>.servicebus.windows.net)")
	logs := cmd.Flags().String("logs", ".", "The directory to write any logs or trace files")
	_ = cmd.MarkFlagRequired("host")

	localEndpoint := "127.0.0.1:5671"

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		fi, err := faultinjectors.NewFaultInjector(
			localEndpoint,
			*host,
			injectorCallback,
			&faultinjectors.FaultInjectorOptions{
				// enable logging all traffic to a JSON file
				JSONLFile: path.Join(*logs, "sample-rawfaultinjector-traffic.json"),
			})

		if err != nil {
			return err
		}

		slog.Info("Starting fault injection server...", "endpoint", localEndpoint)
		return fi.ListenAndServe()
	}

	if err := cmd.Execute(); err != nil {
		slog.Error("Failed to execute fault injector command", "error", err)
	}

	slog.Info("Fault injector done")
}

func injectorCallback(ctx context.Context, params faultinjectors.MirrorCallbackParams) ([]faultinjectors.MetaFrame, error) {
	// this function is the heart of any fault injection. You get access to both incoming and outgoing traffic:
	if params.Out {
		// outbound frames (client -> service)
		slog.Info("Outbound frame", "type", params.Type())
	} else {
		// incoming frames (service -> client)
		slog.Info("Inbound frame", "type", params.Type())
	}

	// you can make decisions about these frames, and alter traffic.

	// for instance, modify a frame in-place:
	switch frameBody := params.Frame.Body.(type) {
	case *frames.PerformFlow:
		slog.Info("MOD: adding an extra credit to each FLOW frame")
		// You can use the marshalled data and just let the fault injector take care of encoding your frame.

		// small modification - let's give them _more_ link credit than they asked for
		frameBody.LinkCredit = utils.Ptr(*frameBody.LinkCredit + 1)

		slightlyModifiedFrame := frames.Frame{
			Header: params.Frame.Header,
			Body:   frameBody,
		}

		return []faultinjectors.MetaFrame{
			{Action: faultinjectors.MetaFrameActionPassthrough, Frame: &slightlyModifiedFrame},
		}, nil
	case *frames.PerformDisposition:
		slog.Info("RAW: taking full control over encoding an AMQP frame")

		// NOTE: as an example we'll just take the raw bytes for the current frame
		// You're welcome to get the raw bytes from _any_ source, "raw frames" ignores all validation and encoding on our end.
		frameBytes := params.Frame.Raw()

		//
		// Exercise left to the reader: "change some bytes up, or encode our own custom replacement and pass those to [frames.NewRawFrame]
		//

		rawFrame := frames.NewRawFrame(frameBytes)

		return []faultinjectors.MetaFrame{
			{Action: faultinjectors.MetaFrameActionPassthrough, Frame: rawFrame},
		}, nil
	default:
		slog.Info("UNCHANGED: don't change a frame")
		// or you could just not change a frame at all
		return []faultinjectors.MetaFrame{
			{Action: faultinjectors.MetaFrameActionPassthrough, Frame: params.Frame},
		}, nil

		// other frame types available:
		// case *frames.EmptyFrame:
		// case *frames.PerformAttach:
		// case *frames.PerformBegin:
		// case *frames.PerformClose:
		// case *frames.PerformDetach:
		// case *frames.PerformDisposition:
		// case *frames.PerformEnd:
		// case *frames.PerformTransfer:
		// default:
		// 	panic(fmt.Sprintf("unexpected frames.Body: %#v", fr))
	}
}
