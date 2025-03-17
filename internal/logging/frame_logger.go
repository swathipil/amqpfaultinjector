package logging

import (
	"encoding/json"
	"time"

	"github.com/Azure/amqpfaultinjector/internal/proto"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
)

// NewFrameLogger creates a FrameLogger instance.
// file - the path to the file to write to.
func NewFrameLogger(file string) (*FrameLogger, error) {
	writer, err := NewSerializedWriter(file)

	if err != nil {
		return nil, err
	}

	logger := &FrameLogger{
		writer: writer,
		sm:     proto.NewStateMap(),
	}

	return logger, nil
}

// FrameLogger is basically a JSONLogger, but instead of accumulating bytes it assumes
// you'll hand it complete frames.Frames instead.
type FrameLogger struct {
	writer       *SerializedWriter
	sm           *proto.StateMap
	transformers transformers
}

func (l *FrameLogger) AddFrame(out bool, fr *frames.Frame, metadata any) error {
	direction := DirectionIn

	if out {
		direction = DirectionOut
	}

	jsonLine := &JSONLine{
		Time:      time.Now(),
		Direction: direction,
		Frame:     fr,
		FrameType: fr.Body.Type(),
		Metadata:  metadata,
	}

	l.sm.AddFrame(out, fr)

	if l.sm != nil {
		if openFrame := l.sm.GetOpenFrame(true); openFrame != nil {
			jsonLine.Connection = &openFrame.Body.ContainerID
		}
	}

	updateJSONLine(out, l.sm, fr, jsonLine)

	if err := l.transformers.Apply(fr, jsonLine); err != nil {
		return err
	}

	jsonBytes, err := json.Marshal(jsonLine)

	if err != nil {
		return err
	}

	return l.writer.Writeln(jsonBytes)
}

func (l *FrameLogger) Close() error {
	return l.writer.Close()
}

func updateJSONLine(out bool, sm *proto.StateMap, fr *frames.Frame, destLine *JSONLine) {
	var attachFrame *frames.PerformAttach

	if tmpAF, ok := fr.Body.(*frames.PerformAttach); ok {
		attachFrame = tmpAF
	} else {
		channel := fr.Header.Channel
		handle := fr.Body.GetHandle()

		if handle != nil {
			if tmpAF := sm.LookupAttachFrame(out, channel, *handle); tmpAF != nil {
				attachFrame = tmpAF.Body
			}
		}
	}

	if attachFrame != nil {
		destLine.LinkName = &attachFrame.Name

		// NOTE: this isn't a direct mirror of the 'Role' attribute - we want to map it to
		// what the user (ie, the client) thinks of this traffic, logically. So if it's
		// outgoing, we just use our role. But if it's _incoming_, then we reverse it because
		// we kind of think of those messages as "being replies".
		role := bool(attachFrame.Role)

		if !out {
			role = !role
		}

		destLine.Receiver = &role
		destLine.EntityPath = attachFrame.Address(out)
	}
}
