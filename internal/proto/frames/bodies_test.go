package frames_test

import (
	"testing"

	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/encoding"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/stretchr/testify/require"
)

func TestPerformAttach(t *testing.T) {
	t.Run("Basic ATTACH frames", func(t *testing.T) {
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
	})

	t.Run("ATTACH without rights", func(t *testing.T) {
		// it's also possible for both Source and Target to be nil, in cases where
		// you didn't actually have rights to ATTACH. This'll come back as an ATTACH frame
		// with those values 'nil', and a DETACH frame with the error immediately after.
		fr := frames.PerformAttach{
			Name:   "Y-dfQa6Qdr3yhJmmsE5boDHNMYLDQugm_MKf4ZzwwUGObvmKiSBa_g",
			Role:   encoding.RoleReceiver,
			Source: nil,
			Target: nil,
		}

		require.Equal(t, "", fr.Address(true))
		require.Equal(t, "", fr.Address(false))

		fr = frames.PerformAttach{
			Name:   "Y-dfQa6Qdr3yhJmmsE5boDHNMYLDQugm_MKf4ZzwwUGObvmKiSBa_g",
			Role:   encoding.RoleSender,
			Source: nil,
			Target: nil,
		}

		require.Equal(t, "", fr.Address(true))
		require.Equal(t, "", fr.Address(false))
	})
}
