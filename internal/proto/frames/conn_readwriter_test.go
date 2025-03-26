package frames_test

import (
	"bytes"
	"errors"
	"io"
	"iter"
	"testing"

	"github.com/richardpark-msft/amqpfaultinjector/internal/mocks"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestParseStream2(t *testing.T) {
	buff := &bytes.Buffer{}

	_, err := buff.Write(attachFrameBytes)
	require.NoError(t, err)

	_, err = buff.Write(detachFrameBytes)
	require.NoError(t, err)

	connRW := frames.NewConnReadWriter(buff)

	next, _ := iter.Pull2(connRW.Iter())

	item, err, valid := next()
	require.True(t, valid)
	require.NoError(t, err)
	require.Equal(t, "name", item.(*frames.Frame).Body.(*frames.PerformAttach).Name)

	item, err, valid = next()
	require.True(t, valid)
	require.NoError(t, err)
	require.True(t, item.(*frames.Frame).Body.(*frames.PerformDetach).Closed)
}

func TestParseStreamWithSmallBlocks(t *testing.T) {
	combinedBytes := make([]byte, 0, len(attachFrameBytes)+len(detachFrameBytes))
	combinedBytes = append(combinedBytes, attachFrameBytes...)
	combinedBytes = append(combinedBytes, detachFrameBytes...)

	conn := mocks.NewMockReadWriter(gomock.NewController(t))

	conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
		if len(combinedBytes) > 0 {
			// use an arbitrarily small chunk, forcing the code to properly join across iterations.
			dest[0] = combinedBytes[0]
			combinedBytes = combinedBytes[1:]
			return 1, nil
		} else {
			return 0, io.EOF
		}
	}).AnyTimes()

	connRW := frames.NewConnReadWriter(conn)

	next, _ := iter.Pull2(connRW.Iter())

	item, err, valid := next()
	require.True(t, valid)
	require.NoError(t, err)
	require.Equal(t, "name", item.(*frames.Frame).Body.(*frames.PerformAttach).Name)

	item, err, valid = next()
	require.True(t, valid)
	require.NoError(t, err)
	require.True(t, item.(*frames.Frame).Body.(*frames.PerformDetach).Closed)
}

func TestParseStream_Frames(t *testing.T) {
	requireFrames := func(conn io.ReadWriter) {
		connRW := frames.NewConnReadWriter(conn)

		var allFrames []*frames.Frame

		for fr, err := range connRW.Iter() {
			require.NoError(t, err)

			fr, isFrame := fr.(*frames.Frame)
			require.True(t, isFrame)

			allFrames = append(allFrames, fr)
		}

		require.Equal(t, 2, len(allFrames))
		require.IsType(t, allFrames[0].Body, &frames.PerformAttach{})
		require.IsType(t, allFrames[1].Body, &frames.PerformDetach{})
	}

	t.Run("All bytes present", func(t *testing.T) {
		conn := mocks.NewMockReadWriter(gomock.NewController(t))

		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
			return copy(dest, attachFrameBytes), nil
		})

		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
			return copy(dest, detachFrameBytes), nil
		})

		conn.EXPECT().Read(gomock.Any()).Return(0, io.EOF).AnyTimes()
		requireFrames(conn)
	})

	t.Run("Two frames returned in EOF call", func(t *testing.T) {
		combinedBytes := make([]byte, 0, len(attachFrameBytes)+len(detachFrameBytes))
		combinedBytes = append(combinedBytes, attachFrameBytes...)
		combinedBytes = append(combinedBytes, detachFrameBytes...)

		conn := mocks.NewMockReadWriter(gomock.NewController(t))

		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
			return copy(dest, combinedBytes), io.EOF
		})

		requireFrames(conn)
	})

	t.Run("One frame normal, one frame at EOF", func(t *testing.T) {
		conn := mocks.NewMockReadWriter(gomock.NewController(t))

		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
			return copy(dest, attachFrameBytes), nil
		})

		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
			return copy(dest, detachFrameBytes), io.EOF
		})

		requireFrames(conn)
	})
}

func TestParseStream_Exits(t *testing.T) {
	t.Run("user cancels iteration", func(t *testing.T) {
		combinedBytes := make([]byte, 0, len(attachFrameBytes)+len(detachFrameBytes))
		combinedBytes = append(combinedBytes, attachFrameBytes...)
		combinedBytes = append(combinedBytes, detachFrameBytes...)

		conn := mocks.NewMockReadWriter(gomock.NewController(t))

		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
			return copy(dest, combinedBytes), nil
		})

		connRW := frames.NewConnReadWriter(conn)

		var allFrames []*frames.Frame

		// pull a single frame and stop iteration
		for fr, err := range connRW.Iter() {
			require.NoError(t, err)

			fr, isFrame := fr.(*frames.Frame)
			require.True(t, isFrame)

			allFrames = append(allFrames, fr)

			break
		}

		require.Equal(t, 1, len(allFrames))
		require.IsType(t, allFrames[0].Body, &frames.PerformAttach{})

		// start iterating again - picks up from where we left off
		for fr, err := range connRW.Iter() {
			require.NoError(t, err)

			fr, isFrame := fr.(*frames.Frame)
			require.True(t, isFrame)

			allFrames = append(allFrames, fr)

			break
		}

		require.Equal(t, 2, len(allFrames))
		require.IsType(t, allFrames[1].Body, &frames.PerformDetach{})
	})

	t.Run("error in network read", func(t *testing.T) {
		conn := mocks.NewMockReadWriter(gomock.NewController(t))

		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
			return 0, errors.New("whoa")
		})

		connRW := frames.NewConnReadWriter(conn)
		count := 0

		for fr, err := range connRW.Iter() {
			count++
			require.Nil(t, fr)
			require.EqualError(t, err, "whoa")
		}

		require.Equal(t, 1, count)
	})

	t.Run("error in the actual bytes we're parsing", func(t *testing.T) {
		conn := mocks.NewMockReadWriter(gomock.NewController(t))

		conn.EXPECT().Read(gomock.Any()).DoAndReturn(func(dest []byte) (int, error) {
			// invalid frame header (because the size represented here is smaller than 8 bytes)
			return copy(dest, []byte{0, 0, 0, 0, 0, 0, 0, 0}), nil
		})

		connRW := frames.NewConnReadWriter(conn)
		count := 0

		for fr, err := range connRW.Iter() {
			count++
			require.Nil(t, fr)
			require.EqualError(t, err, "failed parsing frame header at offset 0: received frame header with invalid size 0")
		}

		require.Equal(t, 1, count)
	})
}

var attachFrameBytes = frames.Frame{
	Header: frames.Header{},
	Body: &frames.PerformAttach{
		Name: "name",
	},
}.MustMarshalAMQP()

var detachFrameBytes = frames.Frame{
	Header: frames.Header{},
	Body: &frames.PerformDetach{
		Closed: true,
	},
}.MustMarshalAMQP()
