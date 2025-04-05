package faultinjectors

import (
	"context"
	"fmt"

	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
)

func NewMultiTransferInjector() *MultiTransferInjector {
	return &MultiTransferInjector{}
}

type MultiTransferInjector struct {
}

// Callback is called by the FaultInjector framework. The current frame is passed in params.Frame.
// Your injector should look at the frame and transform it into the result you want, using the returned
// faultinjector.MetaFrame(s).
func (inj *MultiTransferInjector) Callback(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	// When dealing with a particular link's frames you'll often want to see what the original
	// connection parameters were, the entity path. You can get the current link's ATTACH frame properties
	// at any time using [MirrorCallbackParams.AttachFrame].
	transferFrame, isTransfer := params.Frame.Body.(*frames.PerformTransfer)

	if isTransfer && !params.Out && !params.ManagementOrCBS() {
		var newFrames []MetaFrame
		for i, singlePayloadByte := range transferFrame.Payload {
			// Copy the transfer frame, but set the payload to just a single byte
			clonedFrame := *transferFrame // shallow copy of the original transfer frame
			clonedFrame.Payload = []byte{singlePayloadByte}
			clonedFrame.More = i != len(transferFrame.Payload)-1 // the last frame should not have the "More" flag set.

			newFrames = append(newFrames, MetaFrame{
				Action:      MetaFrameActionAdded,
				Frame:       &frames.Frame{Header: params.Frame.Header, Body: &clonedFrame},
				Description: fmt.Sprintf("Frame %d of %d", i+1, len(transferFrame.Payload)),
			})
		}

		return newFrames, nil
	}

	return []MetaFrame{
		{Action: MetaFrameActionPassthrough, Frame: params.Frame},
	}, nil
}
