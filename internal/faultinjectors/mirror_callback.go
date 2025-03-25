package faultinjectors

import (
	"context"
	"strings"

	"github.com/Azure/amqpfaultinjector/internal/logging"
	"github.com/Azure/amqpfaultinjector/internal/proto"
	"github.com/Azure/amqpfaultinjector/internal/proto/encoding"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
)

// MirrorCallback's take a frame and decide what to send afterwards.
// - To stop mirroring immediately, return (nil, io.EOF)
// - To stop mirroring but send some last packets, return (<metaframes>, io.EOF)
// - Otherwise, return <metaframes>, nil
//
// NOTE, for actual mirroring, see [Mirror].
type MirrorCallback func(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error)

type MirrorCallbackParams struct {
	Out         bool
	Frame       *frames.Frame
	StateMap    *proto.StateMap
	FrameLogger *logging.FrameLogger

	// attachFrame is the cached attach frame, set on first call to [AttachFrame].
	attachFrame *proto.StateFrame[*frames.PerformAttach]
}

const ManagementEntityPathSuffix = "$management"
const CBSEntityPath = "$cbs"

// Handle gets the handle for the current frame or nil if this frame doesn't
// have a handle (for instance, a BEGIN frame).
func (mcp *MirrorCallbackParams) Handle() *uint32 {
	return mcp.Frame.Body.GetHandle()
}

// Channel returns the channel for the current frame.
func (mcp *MirrorCallbackParams) Channel() uint16 {
	return mcp.Frame.Header.Channel
}

// Type is the BodyType of the underlying frame.
func (mcp *MirrorCallbackParams) Type() frames.BodyType {
	return mcp.Frame.Body.Type()
}

// Address is the address from the ATTACH frame that corresponds to
// the link's frame. If the frame is not a link frame (for instance, a
// BEGIN frame) then this function returns an empty string.
func (mcp *MirrorCallbackParams) Address() string {
	if mcp.AttachFrame() != nil {
		return mcp.AttachFrame().Body.Address(mcp.Out)
	}

	return ""
}

// AttachFrame looks up the attach frame corresponding to the link this frame is addressed
// to, or sent from. Note, if the frame is incoming it will give you the ATTACH frame
// details for the _remote_, otherwise it'll give you the ATTACH frame details for your
// local link.
func (mcp *MirrorCallbackParams) AttachFrame() *proto.StateFrame[*frames.PerformAttach] {
	if mcp.Handle() == nil {
		return nil
	}

	if mcp.attachFrame == nil {
		if _, isAttach := mcp.Frame.Body.(*frames.PerformAttach); isAttach {
			mcp.attachFrame = proto.NewStateFrame[*frames.PerformAttach](mcp.Frame)
		} else {
			mcp.attachFrame = mcp.StateMap.LookupAttachFrame(mcp.Out, mcp.Channel(), *mcp.Handle())
		}
	}

	return mcp.attachFrame
}

// Role is the role from the link's attach frame.
// NOTE: if this is an incoming frame the Role will be reversed your local
// links role, as it represents the other end (ie, receiver paired with sender, etc)
func (mcp *MirrorCallbackParams) Role() *encoding.Role {
	if af := mcp.AttachFrame(); af != nil {
		return &af.Body.Role
	}

	return nil
}

// ManagementOrCBS returns true if the link is connected to the $management or $cbs endpoints,
// false if the frame isn't a link frame (ie, BEGIN) or otherwise.
func (mcp *MirrorCallbackParams) ManagementOrCBS() bool {
	attachFrame := mcp.AttachFrame()

	if attachFrame == nil {
		return false
	}

	return strings.HasSuffix(attachFrame.Body.Address(mcp.Out), ManagementEntityPathSuffix) ||
		attachFrame.Body.Address(mcp.Out) == CBSEntityPath
}
