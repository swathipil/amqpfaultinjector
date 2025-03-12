package proto

import (
	"testing"

	"github.com/Azure/amqpfaultinjector/internal/proto/encoding"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/stretchr/testify/require"
)

const clientSideChannel = uint16(100)
const clientSideHandle = uint32(150)

const serverSideChannel = uint16(200)
const serverSideHandle = uint32(250)

func TestStatemap(t *testing.T) {
	// some pretty basic statemap checking.
	sm := NewStateMap()

	require.Panics(t, func() {
		// it's a data corruption issue if we receive an incoming ATTACH frame when we never sent an
		// ATTACH frame from our side.
		sm.AddFrame(false, &frames.Frame{
			Header: frames.Header{Channel: serverSideChannel},
			Body: &frames.PerformAttach{
				Role:   encoding.RoleSender,
				Name:   "link name",
				Handle: serverSideHandle,
				Properties: map[encoding.Symbol]any{
					"server-side": true,
				},
			},
		})
	})

	// we'd send this attach frame to the service with _our_ channel and handle.
	sm.AddFrame(true, &frames.Frame{
		Header: frames.Header{Channel: clientSideChannel},
		Body: &frames.PerformAttach{
			Role:   encoding.RoleReceiver,
			Name:   "link name",
			Handle: clientSideHandle,
			Properties: map[encoding.Symbol]any{
				"client-side": true,
			},
		},
	})

	clientFrame := sm.LookupLocalAttachFrame(clientSideChannel, clientSideHandle)
	require.True(t, clientFrame.Body.Properties["client-side"].(bool))

	// the "server" hasn't yet sent their response to our ATTACH.
	require.Panics(t, func() {
		clientFrame := sm.LookupCorrespondingAttachFrame(true, clientSideChannel, clientSideHandle)
		require.True(t, clientFrame.Body.Properties["client-side"].(bool))
	})

	// and the service would reply with the link they created, and it's channel and handle.
	sm.AddFrame(false, &frames.Frame{
		Header: frames.Header{Channel: serverSideChannel},
		Body: &frames.PerformAttach{
			Role:   encoding.RoleSender,
			Name:   "link name",
			Handle: serverSideHandle,
			Properties: map[encoding.Symbol]any{
				"server-side": true,
			},
		},
	})

	remoteAttachFrame := sm.LookupRemoteAttachFrame(serverSideChannel, serverSideHandle)
	require.True(t, remoteAttachFrame.Body.Properties["server-side"].(bool))

	serverFrame := sm.LookupCorrespondingAttachFrame(true, clientSideChannel, clientSideHandle)
	require.True(t, serverFrame.Body.Properties["server-side"].(bool))

	clientAttachFrame := sm.LookupCorrespondingAttachFrame(false, serverSideChannel, serverSideHandle)
	require.True(t, clientAttachFrame.Body.Properties["client-side"].(bool))
}
