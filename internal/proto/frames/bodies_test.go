package frames_test

import (
	"testing"

	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/encoding"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/stretchr/testify/require"
)

func TestPerformAttach(t *testing.T) {
	fr := frames.PerformAttach{
		Role:   encoding.RoleSender,
		Source: &frames.Source{Address: "source-address"},
		Target: &frames.Target{Address: "target-address"},
	}

	require.Equal(t, "target-address", fr.Address(true))
	require.Equal(t, "source-address", fr.Address(false))

	fr = frames.PerformAttach{
		Role:   encoding.RoleReceiver,
		Source: &frames.Source{Address: "source-address"},
		Target: &frames.Target{Address: "target-address"},
	}
	require.Equal(t, "source-address", fr.Address(true))
	require.Equal(t, "target-address", fr.Address(false))
}
