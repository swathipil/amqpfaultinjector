package encoding

import (
	"testing"

	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/buffer"
	"github.com/stretchr/testify/require"
)

func TestEmptyVsNilForBinaryEncoding(t *testing.T) {
	// this is a fun one - so the go-amqp package, when deserializing messages, can't handle
	// a Null delivery-tag but _it_ encodes it as nil when you send it an empty slice. So I've
	// changed our formatter to not do that and respect what it was passed in.

	// I'm doc'ing this here because I think we might need to propagate a fix but I'm not 100% clear.
	t.Run("nil", func(t *testing.T) {
		buff := buffer.New(nil)
		require.NoError(t, WriteBinary(buff, nil))
		require.Equal(t, TypeCodeNull, AMQPType(buff.Bytes()[0]))
	})

	t.Run("[]byte{}", func(t *testing.T) {
		buff := buffer.New(nil)
		require.NoError(t, WriteBinary(buff, []byte{}))
		// NOTE: there are a few options of what we _could_ do here, all AMQP-legal AFAICT
		// we could do:
		// - vbin8, 0 size   <- this is what we're going with
		// - vbin32, 0 size  <- also valid, but wasteful since we burn 4 bytes for the size of the list!
		// - TypeCodeNull    <- no list
		// - list0 <- 0-sized list (haven't seen anyone use this)
		require.Equal(t, TypeCodeVbin8, AMQPType(buff.Bytes()[0]))
		require.Equal(t, uint8(0), buff.Bytes()[1])
	})

	t.Run("[]byte{1,2}", func(t *testing.T) {
		buff := buffer.New(nil)
		require.NoError(t, WriteBinary(buff, []byte{1, 2}))
		// NOTE: there are a few options of what we _could_ do here, all AMQP-legal AFAICT
		// we could do:
		// - vbin8, 0 size   <- this is what we're going with
		// - vbin32, 0 size  <- also valid, but wasteful since we burn 4 bytes for the size of the list!
		// - TypeCodeNull    <- no list
		// - list0 <- 0-sized list (haven't seen anyone use this)
		require.Equal(t, TypeCodeVbin8, AMQPType(buff.Bytes()[0]))
		require.Equal(t, uint8(2), buff.Bytes()[1])
		require.Equal(t, uint8(1), buff.Bytes()[2])
		require.Equal(t, uint8(2), buff.Bytes()[3])
	})
}
