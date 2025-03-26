package testhelpers

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/stretchr/testify/require"
)

func ValidateLog(t *testing.T, jsonlFile string) {
	// check that the files meet all of our expectations:
	// EntityPath should be filled out on all relevant frames
	// $cbs put-token calls should show up
	var lines = MustReadJSON(t, jsonlFile)

	foundCBS := false

	for _, line := range lines {
		if line.EntityPath == "$cbs" && line.Direction == logging.DirectionOut && line.FrameType == frames.BodyTypeTransfer {
			foundCBS = true

			require.Equal(t, "put-token", line.MessageData.CBSData.ApplicationProperties["operation"])
			require.Empty(t, line.MessageData.Message)
			require.Empty(t, line.Frame.Body)
		} else {
			// sanity check that the other attributes are getting properly written
			require.NotEmpty(t, line.Frame.Body)
			require.NotEmpty(t, line.Direction)
		}

		switch line.FrameType {
		case frames.BodyTypeAttach:
		case frames.BodyTypeDetach:
		case frames.BodyTypeFlow:
		case frames.BodyTypeTransfer:
			require.NotEmpty(t, line.EntityPath)
			require.NotContains(t, line.EntityPath, "reply") // ie, RPC links use a particular pattern, but we don't want those to be used for EntityPath.

		case
			// session frames don't have an entity path
			frames.BodyTypeBegin,
			frames.BodyTypeEnd,
			frames.BodyTypeDisposition,
			// connection level frames don't have an entity path
			frames.BodyTypeOpen,
			frames.BodyTypeClose,
			frames.BodyTypeSASLChallenge,
			frames.BodyTypeSASLInit,
			frames.BodyTypeSASLMechanisms,
			frames.BodyTypeSASLOutcome,
			frames.BodyTypeSASLResponse,
			frames.BodyTypeEmptyFrame:
			require.Empty(t, line.EntityPath)
		}

		require.NotEmpty(t, line.FrameType)
		require.Empty(t, line.RawBody(), "Only filled out for RawFrames, not expected for this test")
	}

	require.True(t, foundCBS)
}

// This type is exactly the same as JSONLine except for Frame and Extra, which are left as raw messages
// which can be deserialized into a more specific type.
type logLine struct {
	Time       time.Time
	Direction  logging.Direction
	EntityPath string  `json:",omitempty"`
	Connection *string `json:",omitempty"`
	Receiver   *bool   `json:",omitempty"`
	LinkName   *string `json:",omitempty"`

	Frame struct {
		Header frames.Header
		Body   json.RawMessage
		Raw    []byte
	}

	FrameType frames.BodyType `json:",omitempty"`

	MessageData struct {
		logging.JSONMessageData
		Message json.RawMessage
	}
}

func (ll logLine) RawBody() []byte {
	return ll.Frame.Raw
}

func MustReadJSON(t *testing.T, path string) []logLine {
	file, err := os.Open(path)
	require.NoError(t, err)

	scn := bufio.NewScanner(file)

	var lines []logLine

	for scn.Scan() {
		var line *logLine

		err := json.Unmarshal(scn.Bytes(), &line)
		require.NoError(t, err)

		lines = append(lines, *line)
	}

	return lines
}
