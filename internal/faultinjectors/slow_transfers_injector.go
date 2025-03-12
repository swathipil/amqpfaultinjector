package faultinjectors

import (
	"context"
	"strconv"
	"time"

	"github.com/Azure/amqpfaultinjector/internal/logging"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/Azure/amqpfaultinjector/internal/utils"
)

// NewSlowTransfersInjector creates a SlowTransferFrames injector, which slows down any incoming
// TRANSFER frames, to non-cbs/non-management links.
//   - delayForFrame controls how long we hold onto a TRANSFER frame, before forwarding the frame to the Receiver.
func NewSlowTransfersInjector(delayForFrame time.Duration) *SlowTransfersInjector {
	return &SlowTransfersInjector{
		delayForFrame: delayForFrame,
	}
}

type SlowTransfersInjector struct {
	delayForFrame time.Duration
}

func (inj *SlowTransfersInjector) Callback(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	// `slog` is the official logging package for Go. We provide some convenience functions to set and retrieve
	// the current slogger, using the passed in context. Logging from fault injectors should _always_ use slog.
	slogger := logging.SloggerFromContext(ctx)

	// pass on, without changes:
	if params.Out || // outbound frames
		params.ManagementOrCBS() || // frames involving the management/cbs
		params.Type() != frames.BodyTypeTransfer { // non-TRANSFER frames
		return []MetaFrame{{Action: MetaFrameActionPassthrough, Frame: params.Frame}}, nil
	}

	transferFrame, ok := params.Frame.Body.(*frames.PerformTransfer)

	if !ok {
		utils.Panicf("BodyType was %s, but the actual body type was %T", params.Frame.BodyType, params.Frame.Body)
	}

	deliveryID := "<unspecified>"

	if transferFrame.DeliveryID != nil {
		deliveryID = strconv.FormatInt(int64(uint64(*transferFrame.DeliveryID)), 10)
	}

	slogger.Info("Slowing down TRANSFER frame", "deliveryid", deliveryID, "more", transferFrame.More)

	// if the sleep is cancelled then the injector is being closed out, we should exit.
	if err := utils.Sleep(ctx, inj.delayForFrame); err != nil {
		return nil, err
	}

	slogger.Info("Sending TRANSFER frame", "deliveryid", deliveryID, "more", transferFrame.More)

	// In this example we're just sending the frame to the client/server without any changes, but we could
	// change the frame, add _more_ frames, or even drop the frame altogether. This is all controlled by the
	// returned slice of MetaFrames.
	return []MetaFrame{
		{Action: MetaFrameActionPassthrough, Frame: params.Frame},
	}, nil
}
