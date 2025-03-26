package faultinjectors

import (
	"time"

	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
)

type MetaFrameAction string

const (
	// MetaFrameActionAdded indicates this is a new frame. It will be encoded and sent.
	MetaFrameActionAdded = MetaFrameAction("added")

	// MetaFrameActionModified indicates the frame has been modified - the frame will be re-encoded and sent.
	MetaFrameActionModified = MetaFrameAction("modified")

	// MetaFrameActionPassthrough indicates the frame should not be modified, and should be sent as-is.
	MetaFrameActionPassthrough = MetaFrameAction("passthrough")

	// MetaFrameActionDropped indicates the frame should be logged, but should not be sent to the client/server.
	MetaFrameActionDropped = MetaFrameAction("dropped")
)

// MetaFrame adds some metadata about the frame, as well as providing overrides for routing
// and delaying the send of frames.
type MetaFrame struct {
	// Action adds some metadata about the frame, as well as allowing the frame
	// bytes to be sent "raw", without any additional encoding.
	Action MetaFrameAction `json:",omitempty"`

	// Delay indicates a frame should be sent, by the fault injector, after the delay expires. The framework
	// NOTE: there's no guarantee that multiple frames, with the same delay, will be sent in the exact same order
	// The delay here is best-effort.
	//
	// If you require absolute ordering, or greater control, you're better off doing that control inside of your
	// injector's function directly. See the [SlowTransferFrames] injector for an example.
	Delay time.Duration `json:",omitempty"`

	// Description is used as metadata in the frame logging file
	Description string `json:",omitempty"`

	// OverrideOut lets you override the direction to send this frame.
	OverrideOut *bool `json:",omitempty"`

	Frame *frames.Frame `json:",omitempty"`
}
