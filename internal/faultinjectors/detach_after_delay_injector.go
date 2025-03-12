package faultinjectors

import (
	"context"
	"time"

	"github.com/Azure/amqpfaultinjector/internal/logging"
	"github.com/Azure/amqpfaultinjector/internal/proto/encoding"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/Azure/amqpfaultinjector/internal/utils"
)

// NewDetachAfterDelayInjector returns a callback that automatically causes links to be DETACH'd 2
// seconds after they're created. When the client has sent an ATTACH for any link we then schedule
// a DETACH to be sent, to the service, for that specific link. Two seconds later, when that
// DETACH is sent we then intercept the DETACH frame from the service, add in an arbitrary error,
// and then send that to the client.
//
// The end result is the client thinks that the service has initiated a DETACH, with error.
func NewDetachAfterDelayInjector(detachAfter time.Duration, detachError *encoding.Error) *DetachAfterDelayInjector {
	if detachAfter == 0 {
		utils.Panicf("detachAfter cannot be zero")
	}

	return &DetachAfterDelayInjector{
		DetachAfter: detachAfter,
		DetachError: detachError,
	}
}

type DetachAfterDelayInjector struct {
	DetachError *encoding.Error
	DetachAfter time.Duration
}

func (rtd *DetachAfterDelayInjector) outbound(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	// TODO: it's a bit inefficient to spawn up a slice just to return a single value.
	myFrames := []MetaFrame{{Action: MetaFrameActionPassthrough, Frame: params.Frame}}

	if attachBody, isAttach := params.Frame.Body.(*frames.PerformAttach); isAttach {
		// schedule a detach for 2 seconds later.
		// To the service, it looks like the client has asked to DETACH
		// To the client, when the service replies, it'll look like the service has initiated the DETACH.
		slogger := logging.SloggerFromContext(ctx)

		slogger.Info("Adding delayed DETACH frame", "entity", attachBody.Address(params.Out), "delay", rtd.DetachAfter)

		detachMetaFrame := MetaFrame{
			Action: MetaFrameActionAdded,
			Frame: &frames.Frame{
				Header: params.Frame.Header,
				Body: &frames.PerformDetach{
					Handle: attachBody.Handle,
					Closed: true,
				},
			},
			Description: "Detaching after a 2 second delay",
			Delay:       rtd.DetachAfter,
		}

		myFrames = append(myFrames, detachMetaFrame)
	}

	return myFrames, nil
}

func (rtd *DetachAfterDelayInjector) inbound(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	if detachBody, isDetach := params.Frame.Body.(*frames.PerformDetach); isDetach {
		slogger := logging.SloggerFromContext(ctx)

		stateFrame := params.StateMap.LookupCorrespondingAttachFrame(false, params.Frame.Header.Channel, detachBody.Handle)

		slogger.Info("Enhancing DETACH frame from service", "entity", stateFrame.Body.Address(params.Out), "delay", rtd.DetachAfter)

		// update DETACH frame to have our configured error in it
		detachBody.Error = rtd.DetachError

		return []MetaFrame{
			{Action: MetaFrameActionModified, Frame: params.Frame, Description: "Updating DETACH with specific error"},
		}, nil
	}

	return []MetaFrame{
		{Action: MetaFrameActionPassthrough, Frame: params.Frame},
	}, nil
}

func (rtd *DetachAfterDelayInjector) Callback(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	if params.Out {
		return rtd.outbound(ctx, params)
	}

	return rtd.inbound(ctx, params)
}
