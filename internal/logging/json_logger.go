package logging

import (
	"encoding/json"
	"time"

	"github.com/richardpark-msft/amqpfaultinjector/internal/proto"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/models"
	"github.com/richardpark-msft/amqpfaultinjector/internal/utils"
)

// NewJSONLogger creates a JSONLogger instance.
// file - the path to the file to write to.
func NewJSONLogger(file string, enableStateTracing bool) (*JSONLogger, error) {
	writer, err := NewSerializedWriter(file)

	if err != nil {
		return nil, err
	}

	var sm *proto.StateMap

	if enableStateTracing {
		sm = proto.NewStateMap()
	}

	logger := &JSONLogger{
		fbout:  &frames.Buffer{},
		fbin:   &frames.Buffer{},
		writer: writer,
		sm:     sm,
	}

	return logger, nil
}

type JSONLogger struct {
	writer *SerializedWriter

	fbout *frames.Buffer

	fbin *frames.Buffer
	sm   *proto.StateMap

	transformers transformers
}

// AddPacket adds in raw bytes, typically received from an AMQP connection. It will
// assemble and write out any frames that it can find after the bytes are added.
// out - whether these are packet bytes being sent to the remote service (true), or being received _from_ the remote service (false).
// packet - the raw bytes, as they are written on the TCP connection.
//
// NOTE: this call cannot be used concurrently.
func (l *JSONLogger) AddPacket(out bool, packet []byte) error {
	if out {
		l.fbout.Add(packet)
	} else {
		l.fbin.Add(packet)
	}

	return l.flush(out)
}

type JSONMessageData struct {
	CBSData *FilteredCBSData `json:",omitempty"`
	Message *models.Message  `json:",omitempty"`
}

type JSONLine struct {
	Time       time.Time
	Direction  Direction
	EntityPath string  `json:",omitempty"`
	Connection *string `json:",omitempty"`
	Receiver   *bool   `json:",omitempty"`
	LinkName   *string `json:",omitempty"`

	FrameType frames.BodyType `json:",omitempty"`

	// Metadata are extra properties that can be added by logging implementations.
	Metadata any `json:",omitempty"`

	// Frame is the actual AMQP Frame object, including the Header and the Body.
	Frame *frames.Frame

	MessageData JSONMessageData `json:",omitempty"`
}

// flush writes out any complete frames it finds within its buffer.
func (l *JSONLogger) flush(out bool) error {
	direction := DirectionIn
	fb := l.fbin

	if out {
		direction = DirectionOut
		fb = l.fbout
	}

	for {
		pi, err := fb.Extract()

		switch {
		case err != nil:
			return err
		case pi == nil:
			return nil
		}

		switch frame := pi.(type) {
		case *frames.Frame:
			jsonLine := &JSONLine{
				Time:      time.Now(),
				Direction: direction,
				Frame:     frame,
				FrameType: frame.Body.Type(),
			}

			if l.sm != nil {
				l.sm.AddFrame(out, frame)

				if openFrame := l.sm.GetOpenFrame(true); openFrame != nil {
					jsonLine.Connection = &openFrame.Body.ContainerID
				}
			}

			updateJSONLine(out, l.sm, frame, jsonLine)

			if err := l.transformers.Apply(frame, jsonLine); err != nil {
				return err
			}

			jsonBytes, err := json.Marshal(jsonLine)

			if err != nil {
				return err
			}

			if err := l.writer.Writeln(jsonBytes); err != nil {
				return err
			}
		case frames.Preamble:
			// just ignore this - it's a constant and there's no real value in it for diagnostics.
		default:
			utils.Panicf("Unhandled type %T", frame)
		}
	}
}

const EntityPathManagement = "$management"
const EntityPathCBS = "$cbs"

const EventHubPropertySecurityToken = "security_token"

func (l *JSONLogger) Close() error {
	return l.writer.Close()
}
