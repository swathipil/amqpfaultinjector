package faultinjectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/encoding"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/richardpark-msft/amqpfaultinjector/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestMirroring(t *testing.T) {
	setup := func(t *testing.T) struct {
		Mirror *mirror
		Local  *testBuffer
		Remote *testBuffer
	} {
		localTestBuff, remoteTestBuff := newTestBuffer(), newTestBuffer()
		local := frames.NewConnReadWriter(localTestBuff)
		remote := frames.NewConnReadWriter(remoteTestBuff)

		m := newMirror(MirrorParams{
			Callback: func(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
				return nil, nil
			},
			FrameLogger: newFrameLoggerForTest(t, "mirrortest"),
			Local:       local,
			Remote:      remote,
		})

		return struct {
			Mirror *mirror
			Local  *testBuffer
			Remote *testBuffer
		}{m, localTestBuff, remoteTestBuff}
	}

	t.Run("io.EOF", func(t *testing.T) {
		td := setup(t)
		stop, err := td.Mirror.handleCallbackResult(true, nil, io.EOF)
		require.NoError(t, err)
		require.True(t, stop)
		require.Empty(t, td.Remote.Frames())
	})

	t.Run("error", func(t *testing.T) {
		td := setup(t)
		stop, err := td.Mirror.handleCallbackResult(true, nil, errors.New("random error"))
		require.False(t, stop)
		require.EqualError(t, err, "error from mirroring callback: random error")
		require.Empty(t, td.Remote.Frames())
	})

	// when you get both we flush out any remaining frames to the connection
	// and then 'stop'.
	t.Run("frames and io.EOF", func(t *testing.T) {
		td := setup(t)

		stop, err := td.Mirror.handleCallbackResult(true, []MetaFrame{
			{Action: MetaFrameActionAdded, Frame: &frames.Frame{Body: &frames.PerformAttach{Name: "added, attach frame", Role: encoding.RoleReceiver, Source: &frames.Source{Address: "source-address"}}}},
			{Action: MetaFrameActionDropped, Frame: &frames.Frame{Body: &frames.PerformAttach{Name: "dropped, attach frame", Role: encoding.RoleReceiver, Source: &frames.Source{Address: "source-address"}}}},
			{Action: MetaFrameActionModified, Frame: &frames.Frame{Body: &frames.PerformAttach{Name: "modified, attach frame", Role: encoding.RoleReceiver, Source: &frames.Source{Address: "source-address"}}}},
		}, io.EOF)
		require.NoError(t, err)
		require.True(t, stop)

		actualFrames := td.Remote.Frames()
		require.Equal(t, []string{
			"added, attach frame",
			// "dropped, attach frame"		<--- this gets dropped, never sent to the connection
			"modified, attach frame",
		}, oopsAllAttachFrameNames(actualFrames))
	})

	t.Run("frames", func(t *testing.T) {
		td := setup(t)

		stop, err := td.Mirror.handleCallbackResult(true, []MetaFrame{
			{Action: MetaFrameActionAdded, Frame: &frames.Frame{Body: &frames.PerformAttach{Name: "added, attach frame", Role: encoding.RoleReceiver, Source: &frames.Source{Address: "source-address"}}}},
			{Action: MetaFrameActionDropped, Frame: &frames.Frame{Body: &frames.PerformAttach{Name: "dropped, attach frame", Role: encoding.RoleReceiver, Source: &frames.Source{Address: "source-address"}}}},
			{Action: MetaFrameActionModified, Frame: &frames.Frame{Body: &frames.PerformAttach{Name: "modified, attach frame", Role: encoding.RoleReceiver, Source: &frames.Source{Address: "source-address"}}}},
		}, nil)
		require.NoError(t, err)
		require.False(t, stop)

		actualFrames := td.Remote.Frames()
		require.Equal(t, []string{
			"added, attach frame",
			// "dropped, attach frame"		<--- this gets dropped, never sent to the connection
			"modified, attach frame",
		}, oopsAllAttachFrameNames(actualFrames))
	})

	t.Run("delayed frames", func(t *testing.T) {
		td := setup(t)

		stop, err := td.Mirror.handleCallbackResult(true, []MetaFrame{
			{
				Action: MetaFrameActionModified,
				Frame:  &frames.Frame{Body: &frames.PerformAttach{Name: "delayed", Role: encoding.RoleReceiver, Source: &frames.Source{Address: "source-address"}}},
				Delay:  time.Second,
			},
		}, nil)
		require.NoError(t, err)
		require.False(t, stop)

		// the frame isn't going to be written right away, so there shouldn't be anything here yet
		actualFrames := td.Remote.Frames()
		require.Empty(t, actualFrames)

		time.Sleep(time.Second + 500*time.Millisecond)

		actualFrames = td.Remote.Frames()
		require.Equal(t, []string{"delayed"}, oopsAllAttachFrameNames(actualFrames))
	})

	t.Run("direction override", func(t *testing.T) {
		td := setup(t)

		out := true
		stop, err := td.Mirror.handleCallbackResult(out, []MetaFrame{
			{
				Action:      MetaFrameActionModified,
				Frame:       &frames.Frame{Body: &frames.PerformFlow{Drain: true}},
				OverrideOut: to.Ptr(false), // write this to the remote -> local stream
			},
		}, nil)
		require.NoError(t, err)
		require.False(t, stop)

		remoteFrames := td.Remote.Frames()
		require.Empty(t, remoteFrames)

		localFrames := td.Local.Frames()
		require.Equal(t, 1, len(localFrames))
		require.True(t, localFrames[0].Body.(*frames.PerformFlow).Drain)
	})
}

func TestMirrorParams_Address_AlreadyAnAttachFrame(t *testing.T) {
	td := []struct {
		Out             bool
		Role            encoding.Role
		ExpectedAddress string
	}{
		{Out: true, Role: encoding.RoleReceiver, ExpectedAddress: "source-address"},
		{Out: false, Role: encoding.RoleReceiver, ExpectedAddress: "target-address"},
		{Out: true, Role: encoding.RoleSender, ExpectedAddress: "target-address"},
		{Out: false, Role: encoding.RoleSender, ExpectedAddress: "source-address"},
	}

	for _, d := range td {
		t.Run(fmt.Sprintf("ATTACH frame: %v", d), func(t *testing.T) {
			body := &frames.PerformAttach{
				Name:   "name",
				Role:   d.Role,
				Source: &frames.Source{Address: "source-address"},
				Target: &frames.Target{Address: "target-address"},
			}

			p := MirrorCallbackParams{
				Out:   d.Out,
				Frame: &frames.Frame{Body: body},
			}

			require.Equal(t, d.ExpectedAddress, p.Address())
			require.NotNil(t, p.attachFrame)
			require.Equal(t, d.ExpectedAddress, p.Address())
		})
	}
}

func TestMirrorParams_Address_LinkFrame(t *testing.T) {
	t.Run("sender attach", func(t *testing.T) {
		sm := loadStateMap(t)

		p := MirrorCallbackParams{
			Out: true,
			Frame: &frames.Frame{
				Header: frames.Header{Channel: 300},
				Body: &frames.PerformFlow{
					Handle: utils.Ptr(uint32(300)),
				},
			},
			StateMap: sm,
		}

		require.Equal(t, "testQueue", p.Address())

		p = MirrorCallbackParams{
			Out: false,
			Frame: &frames.Frame{
				Header: frames.Header{Channel: 1001},
				Body: &frames.PerformFlow{
					Handle: utils.Ptr(uint32(1002)),
				},
			},
			StateMap: sm,
		}

		require.Equal(t, "testQueue", p.Address())
	})

	t.Run("receiver attach", func(t *testing.T) {
		sm := loadStateMap(t)

		p := MirrorCallbackParams{
			Out: true,
			Frame: &frames.Frame{
				Header: frames.Header{Channel: 200},
				Body: &frames.PerformFlow{
					Handle: utils.Ptr(uint32(200)),
				},
			},
			StateMap: sm,
		}

		require.Equal(t, "testQueue", p.Address())

		p = MirrorCallbackParams{
			Out: false,
			Frame: &frames.Frame{
				Header: frames.Header{Channel: 0},
				Body: &frames.PerformFlow{
					Handle: utils.Ptr(uint32(0)),
				},
			},
			StateMap: sm,
		}

		require.Equal(t, "testQueue", p.Address())
	})

	t.Run("$cbs receiver", func(t *testing.T) {
		sm := loadCBSStateMap(t)

		p := MirrorCallbackParams{
			Out: true,
			Frame: &frames.Frame{
				Header: frames.Header{Channel: 200},
				Body: &frames.PerformFlow{
					Handle: utils.Ptr(uint32(201)),
				},
			},
			StateMap: sm,
		}

		require.Equal(t, "$cbs", p.Address())

		p = MirrorCallbackParams{
			Out: false,
			Frame: &frames.Frame{
				Header: frames.Header{Channel: 0},
				Body: &frames.PerformFlow{
					Handle: utils.Ptr(uint32(1)),
				},
			},
			StateMap: sm,
		}

		require.Equal(t, "$cbs", p.Address())
	})
}

func newFrameLoggerForTest(t *testing.T, prefix string) *logging.FrameLogger {
	dir, err := os.MkdirTemp("", prefix+"*")
	require.NoError(t, err)

	t.Cleanup(func() { os.RemoveAll(dir) })

	jsonlFile := filepath.Join(dir, "mirror-traffic.json")

	fl, err := logging.NewFrameLogger(jsonlFile)
	require.NoError(t, err)

	return fl
}

func oopsAllAttachFrameNames(allFrames []*frames.Frame) []string {
	var names []string

	for _, fr := range allFrames {
		names = append(names, fr.Body.(*frames.PerformAttach).Name)
	}

	return names
}

// LoadStateMap loads up some pre-recorded ATTACH frames int our statemap:
// - a receiver, with local channel/handle: 200/200, remote 0/0
// - a sender, with local channel/handle 300/300, remote 1001/1002
func loadStateMap(t *testing.T) *proto.StateMap {
	sm := proto.NewStateMap()

	// handle/channel are 200 for these example files.
	reader, err := os.Open("testdata/receiver_attach_frames.json")
	require.NoError(t, err)
	defer reader.Close()
	decoder := json.NewDecoder(reader)

	var line *logging.JSONLine
	require.NoError(t, decoder.Decode(&line))
	require.NotEmpty(t, line)

	sm.AddFrame(line.Direction == logging.DirectionOut, line.Frame)

	line = nil
	require.NoError(t, decoder.Decode(&line))
	require.NotEmpty(t, line)

	sm.AddFrame(line.Direction == logging.DirectionOut, line.Frame)

	// handle/channel are 300 for these example files.
	reader, err = os.Open("testdata/sender_attach_frames.json")
	require.NoError(t, err)
	defer reader.Close()
	decoder = json.NewDecoder(reader)

	line = nil
	require.NoError(t, decoder.Decode(&line))
	require.NotEmpty(t, line)

	sm.AddFrame(line.Direction == logging.DirectionOut, line.Frame)

	line = nil
	require.NoError(t, decoder.Decode(&line))
	require.NotEmpty(t, line)

	sm.AddFrame(line.Direction == logging.DirectionOut, line.Frame)

	// sanity check, all the frames are loaded and with the proper direction.
	out := true
	localSenderAF := sm.LookupAttachFrame(out, 300, 300)
	require.Equal(t, "testQueue", localSenderAF.Body.Address(out))

	out = false
	remoteSenderAF := sm.LookupAttachFrame(out, 1001, 1002)
	require.Equal(t, "testQueue", remoteSenderAF.Body.Address(out))

	out = true
	localReceiveAF := sm.LookupAttachFrame(out, 200, 200)
	require.Equal(t, "testQueue", localReceiveAF.Body.Address(out))

	out = false
	remoteReceiverAF := sm.LookupAttachFrame(out, 0, 0)
	require.Equal(t, "testQueue", remoteReceiverAF.Body.Address(out))

	return sm
}

func loadCBSStateMap(t *testing.T) *proto.StateMap {
	sm := proto.NewStateMap()

	// handle/channel are 200 for these example files.
	reader, err := os.Open("testdata/cbs_receiver_frames.json")
	require.NoError(t, err)
	defer reader.Close()
	decoder := json.NewDecoder(reader)

	var line *logging.JSONLine
	require.NoError(t, decoder.Decode(&line))
	require.NotEmpty(t, line)

	sm.AddFrame(line.Direction == logging.DirectionOut, line.Frame)

	line = nil
	require.NoError(t, decoder.Decode(&line))
	require.NotEmpty(t, line)

	sm.AddFrame(line.Direction == logging.DirectionOut, line.Frame)

	return sm
}
