package loganalyzer_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/amqpfaultinjector/internal/logging"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/Azure/amqpfaultinjector/internal/utils"
	"github.com/stretchr/testify/require"
)

type logLine[MessageDataT any] struct {
	Time        time.Time
	Direction   logging.Direction
	FrameType   frames.BodyType
	EntityPath  string
	Connection  *string
	Receiver    *bool
	LinkName    *string
	MessageData logging.JSONMessageData
	Frame       json.RawMessage
}

type MessageData struct {
	Properties struct {
		MessageID     any
		CorrelationID any
	}
	ApplicationProperties map[string]any
	Value                 struct {
		LockTokens [][]byte `json:"lock-tokens"`
	}
}

func TestLogAnalysisMgmtOpReusesMessageIDs(t *testing.T) {
	t.Skip("Need to regen these files")
	err := analyzeLogForDuplicateRPCIDs("testdata/amqpproxy-traffic-alr-bad.json")
	require.EqualError(t, err, "message ID 0 has multiple operations active")
}

func TestLogAnalysisMgmtOpUsesUniqueMessageIDs(t *testing.T) {
	t.Skip("Need to regen these files")
	err := analyzeLogForDuplicateRPCIDs("testdata/amqpproxy-traffic-alr-good.json")
	require.NoError(t, err)
}

func analyzeLogForDuplicateRPCIDs(path string) error {
	reader, err := os.Open(path)

	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(reader)

	messageIDs := map[string]bool{}

	for scanner.Scan() {
		var ll *logLine[MessageData]
		err = json.Unmarshal(scanner.Bytes(), &ll)

		if err != nil {
			return err
		}

		// renew lock requests
		if ll.Direction == logging.DirectionOut &&
			ll.FrameType == frames.BodyTypeTransfer &&
			strings.HasSuffix(ll.EntityPath, "$management") &&
			ll.MessageData.Message != nil &&
			ll.MessageData.Message.ApplicationProperties != nil &&
			ll.MessageData.Message.ApplicationProperties["operation"] == "com.microsoft:renew-lock" {

			key := stringizeMessageID(ll.MessageData.Message.Properties.MessageID)

			if messageIDs[key] {
				return fmt.Errorf("message ID %s has multiple operations active", key)
			}

			messageIDs[key] = true
		}

		if ll.Direction == logging.DirectionIn &&
			ll.FrameType == frames.BodyTypeTransfer &&
			strings.HasSuffix(ll.EntityPath, "$management") {
			key := stringizeMessageID(ll.MessageData.Message.Properties.CorrelationID)
			delete(messageIDs, key)
		}
	}

	return scanner.Err()
}

func stringizeMessageID(v any) string {
	switch id := v.(type) {
	case string:
		return id
	case []any:
		var buff []byte

		for _, x := range id {
			asInt := x.(float64)
			buff = append(buff, byte(asInt))
		}

		return fmt.Sprintf("%X", buff)
	case []byte:
		return fmt.Sprintf("%X", id)
	case float64:
		return fmt.Sprintf("%d", byte(id))
	default:
		utils.Panicf("Can't stringize %T, %#v", v, v)
		return ""
	}
}
