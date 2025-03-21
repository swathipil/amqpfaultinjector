package faultinjectors

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Azure/amqpfaultinjector/internal/logging"
	"github.com/Azure/amqpfaultinjector/internal/proto/encoding"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
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
