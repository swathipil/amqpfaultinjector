package frames

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuffer(t *testing.T) {
	fb := &Buffer{}

	fb.Add(attachFrameBytes)

	item, err := fb.Extract()
	require.NoError(t, err)
	require.Equal(t, "name", item.(*Frame).Body.(*PerformAttach).Name)

	requireEmpty(t, fb)

	fb.Add(detachFrameBytes)

	item, err = fb.Extract()
	require.NoError(t, err)
	require.True(t, item.(*Frame).Body.(*PerformDetach).Closed)
}

func TestBuffer_PartialFrames(t *testing.T) {
	t.Run("ExactBoundaries", func(t *testing.T) {
		fb := &Buffer{}

		fb.Add(attachFrameBytes[0:8])
		fb.Add(attachFrameBytes[8:])

		item, err := fb.Extract()
		require.NoError(t, err)
		require.Equal(t, "name", item.(*Frame).Body.(*PerformAttach).Name)
	})

	t.Run("SplitAcrossBoundaries", func(t *testing.T) {
		fb := &Buffer{}

		fb.Add(attachFrameBytes[0:10])
		requireEmpty(t, fb)

		fb.Add(attachFrameBytes[10:])
		item, err := fb.Extract()
		require.NoError(t, err)
		require.Equal(t, "name", item.(*Frame).Body.(*PerformAttach).Name)
	})

	t.Run("HeaderSplitAcrossBoundaries", func(t *testing.T) {
		fb := &Buffer{}

		fb.Add(attachFrameBytes[0:3])
		requireEmpty(t, fb)

		fb.Add(attachFrameBytes[3:])
		item, err := fb.Extract()
		require.NoError(t, err)
		require.Equal(t, "name", item.(*Frame).Body.(*PerformAttach).Name)
	})
}

func TestBuffer_ExtractFrame_Preamble(t *testing.T) {
	fb := &Buffer{}
	fb.Add(amqpPreamble)
	fr, err := fb.ExtractFrame()
	require.Nil(t, fr)
	require.EqualError(t, err, "underlying type was frames.Raw, not an *AMQPItem. If the stream can also contain an AMQP preamble you must use Extract() instead.")
}

func requireEmpty(t *testing.T, fb *Buffer) {
	item, err := fb.Extract()
	require.NoError(t, err)
	require.Nil(t, item, "No items should be available")

	fr, err := fb.ExtractFrame()
	require.NoError(t, err)
	require.Nil(t, fr, "No items should be available")
}

var attachFrameBytes = Frame{
	Header: Header{},
	Body: &PerformAttach{
		Name: "name",
	},
}.MustMarshalAMQP()

var detachFrameBytes = Frame{
	Header: Header{},
	Body: &PerformDetach{
		Closed: true,
	},
}.MustMarshalAMQP()

var amqpPreamble = []byte{'A', 'M', 'Q', 'P', 0, 1, 0, 0}
