package testhelpers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/stretchr/testify/require"
)

func TestLogLinesUnmarshalling(t *testing.T) {
	t.Run("RawFrame", func(t *testing.T) {
		now := time.Now().UTC()

		// deserialization and serialization aren't quite matched up right now
		data, err := json.Marshal(logging.JSONLine{
			Time:       now,
			Direction:  logging.DirectionOut,
			EntityPath: "entity path",
			// raw frames, when serialized, will output an extra field called 'Raw', which contains
			// the actual bytes. This intentionally bypasses any sort of interpretation or serialization
			// since the data might not actually be valid AMQP frame data.
			FrameType: frames.BodyTypeRawFrame,
			Frame:     frames.NewRawFrame([]byte("arbitrary bytes")),
		})
		require.NoError(t, err)
		require.NotEmpty(t, data)

		var actualLogLine *logLine
		err = json.Unmarshal(data, &actualLogLine)
		require.NoError(t, err)

		require.Equal(t, &logLine{
			Time:       now,
			Direction:  logging.DirectionOut,
			EntityPath: "entity path",
			Frame: struct {
				Header frames.Header
				Body   json.RawMessage
				Raw    []byte
			}{
				Body: json.RawMessage([]byte("{}")),
				Raw:  []byte("arbitrary bytes"),
			},
			FrameType: frames.BodyTypeRawFrame,
		}, actualLogLine)
	})

	t.Run("AttachFrame", func(t *testing.T) {
		now := time.Now().UTC()

		// deserialization and serialization aren't quite matched up right now
		data, err := json.Marshal(logging.JSONLine{
			Time:       now,
			Direction:  logging.DirectionOut,
			EntityPath: "entity path",
			// raw frames, when serialized, will output an extra field called 'Raw', which contains
			// the actual bytes. This intentionally bypasses any sort of interpretation or serialization
			// since the data might not actually be valid AMQP frame data.
			FrameType: frames.BodyTypeAttach,
			Frame: &frames.Frame{
				Body: &frames.PerformAttach{
					Name: "attach-name",
				},
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, data)

		var actualLogLine *logLine
		err = json.Unmarshal(data, &actualLogLine)
		require.NoError(t, err)

		require.Equal(t, now, actualLogLine.Time)
		require.Equal(t, logging.DirectionOut, actualLogLine.Direction)
		require.Equal(t, "entity path", actualLogLine.EntityPath)

		var attachFrame *frames.PerformAttach
		err = json.Unmarshal(actualLogLine.Frame.Body, &attachFrame)
		require.NoError(t, err)

		require.Equal(t, &frames.PerformAttach{
			Name: "attach-name",
		}, attachFrame)

		require.Nil(t, actualLogLine.RawBody())
	})
}
