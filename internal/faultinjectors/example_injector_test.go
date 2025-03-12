package faultinjectors

import (
	"context"

	"github.com/Azure/amqpfaultinjector/internal/logging"
	"github.com/Azure/amqpfaultinjector/internal/proto/encoding"
)

func NewExampleInjector(exampleValue int) *ExampleInjector {
	return &ExampleInjector{exampleValue: exampleValue}
}

type ExampleInjector struct {
	exampleValue int
}

// Callback is called by the FaultInjector framework. The current frame is passed in params.Frame.
// Your injector should look at the frame and transform it into the result you want, using the returned
// faultinjector.MetaFrame(s).
func (inj *ExampleInjector) Callback(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	// `slog` is the official logging package for Go. We provide some convenience functions to set and retrieve
	// the current slogger, using the passed in context. Logging from fault injectors should _always_ use slog.
	slogger := logging.SloggerFromContext(ctx)

	// When dealing with a particular link's frames you'll often want to see what the original
	// connection parameters were, the entity path. You can get the current link's ATTACH frame properties
	// at any time using [MirrorCallbackParams.AttachFrame].
	if attachFrame := params.AttachFrame(); attachFrame != nil {
		// slog loggers take a single message as the text, and additional context as "string, any" pairs.
		slogger.Debug("Some ATTACH frame details",
			"desiredcapabilities", attachFrame.Body.DesiredCapabilities,
			"managementOrCBS", params.ManagementOrCBS(),
			"outbound", params.Out,
			"isSender", *params.Role() == encoding.RoleSender,
			"exampleValue", inj.exampleValue)
	}

	// In this example we're just sending the frame to the client/server without any changes, but we could
	// change the frame, add _more_ frames, or even drop the frame altogether. This is all controlled by the
	// returned slice of MetaFrames.
	return []MetaFrame{
		// [MetaFrameActionPassthrough] indicates the frame should be passed on, without changes.
		// Other options include:
		// - MetaFrameActionAdded
		// - MetaFrameActionModified
		// - MetaFrameActionDropped
		//
		// See the documentation of each constant for more details.
		{Action: MetaFrameActionPassthrough, Frame: params.Frame},
	}, nil
}
