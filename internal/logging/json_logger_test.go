package logging

import (
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/models"
	"github.com/richardpark-msft/amqpfaultinjector/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestJSONLoggerTransformPutToken(t *testing.T) {
	receivedFrame := &frames.Frame{Body: &frames.PerformTransfer{
		Payload: mustMarshalBinary(t, &models.Message{
			ApplicationProperties: map[string]any{
				"status-code":        int64(202),
				"status-description": "Accepted",
			},
		}),
	}}

	peekMessage := &models.Message{
		Properties: &models.MessageProperties{
			ReplyTo: to.Ptr("sb-te9c6b9643d59d62d-queue/$management"),
		},
		ApplicationProperties: map[string]any{
			"associated-link-name": "acc35691-6dd5-4cc1-bb9a-2a9a6a998522",
			"operation":            "com.microsoft:peek-message",
			"type":                 "entity-mgmt",
		},
		Value: map[string]any{
			"from-sequence-number": int64(1),
			"message-count":        int64(1),
		},
	}

	peekSentFrame := &frames.Frame{Body: &frames.PerformTransfer{
		Payload: mustMarshalBinary(t, peekMessage),
	}}

	tt := []struct {
		Title          string
		InputFrame     *frames.Frame
		InputJSONFrame *JSONLine
		ExpectedFrame  *frames.Frame
		ExpectedExtra  JSONMessageData
	}{
		{
			"AttachSend",
			&frames.Frame{Body: &frames.PerformAttach{}},
			nil,
			&frames.Frame{Body: &frames.PerformAttach{}},
			JSONMessageData{},
		},
		{
			"TransferSendAuth",
			&frames.Frame{Body: &frames.PerformTransfer{
				Payload: mustMarshalBinary(t, &models.Message{
					ApplicationProperties: map[string]any{
						"expiration": "2025-01-17T19:56:03Z",
						"name":       "sb://sb-testing.servicebus.windows.net/sb-testing-queue",
						"operation":  "put-token",
						"type":       "jwt",
					},
				}),
			}},
			&JSONLine{EntityPath: "$cbs", Direction: "out"},
			nil,
			JSONMessageData{
				CBSData: &FilteredCBSData{
					ApplicationProperties: map[string]any{
						"expiration": "2025-01-17T19:56:03Z",
						"name":       "sb://sb-testing.servicebus.windows.net/sb-testing-queue",
						"operation":  "put-token",
						"type":       "jwt",
					},
				},
			},
		},
		{
			"TransferReceiveAuth",
			receivedFrame,
			&JSONLine{EntityPath: "$cbs", Direction: "in"},
			receivedFrame,
			JSONMessageData{
				Message: &models.Message{
					ApplicationProperties: map[string]any{
						"status-code":        int64(202),
						"status-description": "Accepted",
					},
				},
			},
		},
		{
			"PeekMgmtTransfer",
			peekSentFrame,
			&JSONLine{EntityPath: "$management", Direction: "in"},
			peekSentFrame,
			JSONMessageData{
				Message: peekMessage,
			},
		},
	}

	for _, testData := range tt {
		t.Run(testData.Title, func(t *testing.T) {
			tm := transformers{}
			payload, extra, err := tm.transformPutToken(testData.InputFrame, testData.InputJSONFrame)
			require.NoError(t, err)

			require.Equal(t, payload, testData.ExpectedFrame)
			require.Equal(t, extra, testData.ExpectedExtra)
		})
	}

}

func mustMarshalBinary(t *testing.T, msg *models.Message) []byte {
	data, err := msg.MarshalBinary()
	require.NoError(t, err)
	return data
}

func TestJSONLoggerWithRustReplay(t *testing.T) {
	t.Skip("Manually uncomment when doing troubleshooting")

	file, err := os.CreateTemp("", "jsonwriter*")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	jsonLogger, err := NewJSONLogger(file.Name(), false)
	require.NoError(t, err)

	numOut := 0
	numIn := 0

	for binLine, err := range utils.ParseBinFile("<bin file path goes here>") {
		require.NoError(t, err)

		t.Logf("%#v", binLine)

		out := binLine.Label == "out"

		if out {
			numOut++
		} else {
			numIn++
		}

		err = jsonLogger.AddPacket(out, binLine.Packet)
		require.NoErrorf(t, err, "%s, out: %d, in: %d", binLine.Label, numOut, numIn)
	}
}
