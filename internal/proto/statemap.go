package proto

import (
	"fmt"

	"github.com/Azure/amqpfaultinjector/internal/proto/encoding"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/Azure/amqpfaultinjector/internal/utils"
)

type StateMap struct {
	localByRoleAndName attachFramesByRoleAndName // used when we map the remote's ATTACH frame to our own

	localAttach  attachFramesByChannelAndHandle
	remoteAttach attachFramesByChannelAndHandle

	// Reverse-lookup versions of localAttach and remoteAttach. These maps aren't populated until we receive
	// the remote service's ATTACH responses.
	localToRemote attachFramesByChannelAndHandle
	remoteToLocal attachFramesByChannelAndHandle

	remoteOpenFrame *StateFrame[*frames.PerformOpen]
	localOpenFrame  *StateFrame[*frames.PerformOpen]
}

func NewStateMap() *StateMap {
	return &StateMap{
		localToRemote: attachFramesByChannelAndHandle{},
		remoteToLocal: attachFramesByChannelAndHandle{},
		localAttach:   attachFramesByChannelAndHandle{},
		remoteAttach:  attachFramesByChannelAndHandle{},

		localByRoleAndName: attachFramesByRoleAndName{},
	}
}

func (sm *StateMap) AddFrame(out bool, fr *frames.Frame) {
	switch fr.Body.(type) {
	case *frames.PerformOpen:
		sm.SetOpenFrame(out, NewStateFrame[*frames.PerformOpen](fr))
	case *frames.PerformAttach:
		if out {
			sm.outboundAttach(NewStateFrame[*frames.PerformAttach](fr))
		} else {
			sm.incomingAttach(NewStateFrame[*frames.PerformAttach](fr))
		}
	}
}

func (sm *StateMap) GetOpenFrame(out bool) *StateFrame[*frames.PerformOpen] {
	if out {
		return sm.localOpenFrame
	} else {
		return sm.remoteOpenFrame
	}
}

func (sm *StateMap) SetOpenFrame(out bool, openFrame *StateFrame[*frames.PerformOpen]) {
	if out {
		sm.localOpenFrame = openFrame
	} else {
		sm.remoteOpenFrame = openFrame
	}
}

// LookupCorrespondingAttachFrame looks up the corresponding link's ATTACH frame.
// - If localToRemote is true, pass in a local channel and local handle. The ATTACH frame will contain the values the service sent us for THEIR side of the link.
// - If localToRemote is false, pass in a remote channel and remote handle. The ATTACH frame will contain the values we sent to the service for OUR side of the link.
func (sm *StateMap) LookupCorrespondingAttachFrame(localToRemote bool, channel uint16, handle uint32) *StateFrame[*frames.PerformAttach] {
	if localToRemote {
		return sm.localToRemote.Load(channelAndHandle{channel, handle})
	} else {
		return sm.remoteToLocal.Load(channelAndHandle{channel, handle})
	}
}

func (sm *StateMap) LookupAttachFrame(out bool, channel uint16, handle uint32) *StateFrame[*frames.PerformAttach] {
	if out {
		return sm.LookupLocalAttachFrame(channel, handle)
	} else {
		return sm.LookupRemoteAttachFrame(channel, handle)
	}
}

func (sm *StateMap) LookupLocalAttachFrame(channel uint16, handle uint32) *StateFrame[*frames.PerformAttach] {
	return sm.localAttach.Load(channelAndHandle{channel, handle})
}

func (sm *StateMap) LookupRemoteAttachFrame(channel uint16, handle uint32) *StateFrame[*frames.PerformAttach] {
	return sm.remoteAttach.Load(channelAndHandle{channel, handle})
}

// outboundAttach handles the ATTACH frame, initiated from the client.
func (sm *StateMap) outboundAttach(fr *StateFrame[*frames.PerformAttach]) {
	sm.localByRoleAndName.Store(linkAndRole{
		Role:     fr.Body.Role,
		LinkName: fr.Body.Name,
	}, fr)

	sm.localAttach.Store(channelAndHandle{fr.Header.Channel, fr.Body.Handle}, fr)
}

// incomingAttach handles the ATTACH frame reply, from the service.
func (sm *StateMap) incomingAttach(remoteAttachFrame *StateFrame[*frames.PerformAttach]) {
	sm.remoteAttach.Store(channelAndHandle{remoteAttachFrame.Header.Channel, remoteAttachFrame.Body.Handle}, remoteAttachFrame)

	// find our corresponding link, which'll have the opposite role of theirs, but with the
	// same link name.
	// our local counterpart will have the opposite role (ie, sender -> receiver, receiver -> sender)
	// but should share the same link name.
	localAttachFrame := sm.localByRoleAndName.Load(linkAndRole{!remoteAttachFrame.Body.Role, remoteAttachFrame.Body.Name})

	if localAttachFrame == nil {
		utils.Panicf("No corresponding attach frame for %s, receiver:%t", remoteAttachFrame.Body.Name, remoteAttachFrame.Body.Role)
	}

	// let's associate our local ATTACH frame with the remote's equivalent of their channel and handle.
	// we can look it up later if we want to correlate detaches.
	sm.localToRemote.Store(channelAndHandle{localAttachFrame.Header.Channel, localAttachFrame.Body.Handle}, remoteAttachFrame)
	sm.remoteToLocal.Store(channelAndHandle{remoteAttachFrame.Header.Channel, remoteAttachFrame.Body.Handle}, localAttachFrame)
}

type attachFramesByRoleAndName = utils.SyncMap[linkAndRole, *StateFrame[*frames.PerformAttach]]
type attachFramesByChannelAndHandle = utils.SyncMap[channelAndHandle, *StateFrame[*frames.PerformAttach]]

type channelAndHandle struct {
	Channel uint16
	Handle  uint32
}

type linkAndRole struct {
	Role     encoding.Role
	LinkName string
}

type StateFrame[T frames.Body] struct {
	*frames.Frame
	Body T
}

func NewStateFrame[T frames.Body](fr *frames.Frame) *StateFrame[T] {
	if body, ok := fr.Body.(T); !ok {
		var zeroT T
		panic(fmt.Errorf("invalid state frame type - expected %T, got %T", zeroT, fr.Body))
	} else {
		return &StateFrame[T]{Frame: fr, Body: body}
	}
}
