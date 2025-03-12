package logging

import (
	"fmt"
	"log/slog"

	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/Azure/amqpfaultinjector/internal/proto/models"
)

// transformers applies some basic sanitizers and transformers to make
// the resulting output more useful, while still redacting sensitive information
// like tokens.
type transformers struct {
	multipartsOut []byte
	multipartsIn  []byte
}

func (tf *transformers) Apply(fr *frames.Frame, jsonFrame *JSONLine) error {
	if payload, extra, err := tf.transformPutToken(fr, jsonFrame); err != nil {
		return err
	} else {
		jsonFrame.Frame = payload
		jsonFrame.MessageData = extra
		return nil
	}
}

func (tf *transformers) getMultipart(direction Direction) *[]byte {
	switch direction {
	case DirectionIn:
		return &tf.multipartsIn
	case DirectionOut:
		return &tf.multipartsOut
	default:
		panic(fmt.Sprintf("unexpected logging.Direction: %#v", direction))
	}
}

func (tf *transformers) transformPutToken(fr *frames.Frame, jsonFrame *JSONLine) (frame *frames.Frame, messageData JSONMessageData, err error) {
	switch body := fr.Body.(type) {
	case *frames.PerformTransfer:
		if body.More {
			*tf.getMultipart(jsonFrame.Direction) = append(*tf.getMultipart(jsonFrame.Direction), body.Payload...)
			frame = fr // we still want to log the frame, we just can't decode the payload.
		} else {
			// if we can decode the bytes of the TRANSFER frame include it in the Extra field.
			payloadBytes := body.Payload

			mp := tf.getMultipart(jsonFrame.Direction)

			if len(*mp) > 0 {
				*mp = append(*mp, body.Payload...)
				payloadBytes = *mp
				*mp = nil
			}

			payloadMsg := &models.Message{}

			if err := payloadMsg.UnmarshalBinary(payloadBytes); err != nil {
				// we're going to mark this as a non-fatal error. This happens if we're parsing _batch_ messages, which
				// we don't normally have to do as a client-side SDK, but do if we're parsing messages that are being
				// sent to the service. Fixing this is tracked by https://github.com/richardpark-msft/priv-amqpfaultinjector/issues/82
				slog.Warn("Failed to unmarshal TRANSFER frame payload", "error", err)
			}

			// here's where we'll omit the message for put-token.
			switch {
			case jsonFrame.EntityPath == EntityPathCBS && jsonFrame.Direction == "out":
				// these are the put-token calls to $cbs, which do contain a token, as the .Value of the AMQP message payload.
				// The frame has to be removed too, since it's encoded payload contains the security_token as well.
				frame = nil
				messageData = JSONMessageData{
					CBSData: &FilteredCBSData{
						ApplicationProperties: payloadMsg.ApplicationProperties,
					},
				}
			case jsonFrame.EntityPath == EntityPathManagement && jsonFrame.Direction == "out" && payloadMsg.ApplicationProperties[EventHubPropertySecurityToken] != "":
				// In Event Hubs $management operations include a security token. We'll remove that single value.
				// The frame has to be removed too, since it's encoded payload contains the security_token as well.
				payloadMsg.ApplicationProperties[EventHubPropertySecurityToken] = "<redacted>"
				frame = nil
				messageData = JSONMessageData{Message: payloadMsg}
			default:
				// include the entire message as the MessageData, we don't need to censor it.
				frame = fr
				messageData = JSONMessageData{Message: payloadMsg}
			}
		}
		return
	default:
		return fr, JSONMessageData{}, nil
	}
}

// FilteredCBSData extracts the important properties from a put-token message
// while not revealing the token, making it safe for storing in source control
// or distributing as examples/documentation.
type FilteredCBSData struct {
	ApplicationProperties map[string]any
}
