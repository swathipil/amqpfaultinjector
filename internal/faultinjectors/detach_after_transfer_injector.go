package faultinjectors

import (
	"context"
	"sync/atomic"

	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/encoding"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
)

// NewDetachAfterTransferInjector creates an injector that detaches an AMQP sender after a set number of
// TRANSFER frames. Note, if your transfer is split across several frames each one counts, even if they're
// logically part of the same message.
func NewDetachAfterTransferInjector(times int, amqpError encoding.Error) *DetachAfterTransferInjector {
	return &DetachAfterTransferInjector{
		detachesRemaining: int64(times),
		amqpError:         amqpError,
	}
}

type DetachAfterTransferInjector struct {
	detachesRemaining int64
	amqpError         encoding.Error
}

func (dat *DetachAfterTransferInjector) outbound(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	slogger := logging.SloggerFromContext(ctx)

	myFrames := []MetaFrame{
		{Action: MetaFrameActionPassthrough, Frame: params.Frame},
	}

	if transferBody, isTransfer := params.Frame.Body.(*frames.PerformTransfer); isTransfer {
		localAttachFrame := params.StateMap.LookupLocalAttachFrame(params.Frame.Header.Channel, transferBody.Handle)

		if !params.ManagementOrCBS() {
			if atomic.AddInt64(&dat.detachesRemaining, -1) >= 0 {
				slogger.Info("Adding artificial DETACH frame", "address", localAttachFrame.Body.Address(params.Out))

				// To the service, it looks like the client has asked to DETACH
				// To the client, when the service replies, it'll look like the service has initiated the DETACH.
				detachMetaFrame := MetaFrame{
					Action: MetaFrameActionAdded,
					Frame: &frames.Frame{
						Header: params.Frame.Header,
						Body: &frames.PerformDetach{
							Handle: transferBody.Handle,
							Closed: true,
						},
					},
					Description: "DETACHing after TRANSFER",
				}

				myFrames = []MetaFrame{
					{Action: MetaFrameActionDropped, Frame: params.Frame},
					detachMetaFrame,
				}
			}
		}
	}

	return myFrames, nil
}

func (dat *DetachAfterTransferInjector) inbound(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	if detachBody, isDetach := params.Frame.Body.(*frames.PerformDetach); isDetach {
		slogger := logging.SloggerFromContext(ctx)

		stateFrame := params.StateMap.LookupCorrespondingAttachFrame(false, params.Frame.Header.Channel, detachBody.Handle)

		slogger.Info("Enhancing DETACH frame from service", "entity", stateFrame.Body.Address(params.Out))

		// update DETACH frame to have our configured error in it
		detachBody.Error = &dat.amqpError

		return []MetaFrame{
			{Action: MetaFrameActionModified, Frame: params.Frame, Description: "Updating DETACH with specific error"},
		}, nil
	}

	return []MetaFrame{
		{Action: MetaFrameActionPassthrough, Frame: params.Frame},
	}, nil
}

func (dat *DetachAfterTransferInjector) Callback(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	if params.Out {
		return dat.outbound(ctx, params)
	}

	return dat.inbound(ctx, params)
}
