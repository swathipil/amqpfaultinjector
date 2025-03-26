package frames

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"

	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/buffer"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/encoding"
)

// PreambleOrFrame represents parseable items when extracting data from an AMQP
// stream. Includes [*Frame] and [Preamble].
type PreambleOrFrame interface {
	amqpItem()

	// Raw is the original raw bytes, and isn't required to change if the underlying
	// parsed entities change.
	Raw() []byte

	MustMarshalAMQP() []byte
	MarshalAMQP() ([]byte, error)
}

type BodyType string

const (
	// Pseudo-frames, tracking "special" types
	BodyTypeEmptyFrame = "Empty" // a frame without a body. This is commonly used for AMQP keep-alives.
	BodyTypeRawFrame   = "Raw"   // a frame that was fabricated. This is primarily useful when attempting to go outside of the AMQP, with full control over the encoded frame.

	// AMQP frame types
	BodyTypeAttach      = "Attach"
	BodyTypeBegin       = "Begin"
	BodyTypeClose       = "Close"
	BodyTypeDetach      = "Detach"
	BodyTypeDisposition = "Disposition"
	BodyTypeEnd         = "End"
	BodyTypeFlow        = "Flow"
	BodyTypeOpen        = "Open"
	BodyTypeTransfer    = "Transfer"

	// SASL frames
	BodyTypeSASLChallenge  = "SASLChallenge"
	BodyTypeSASLInit       = "SASLInit"
	BodyTypeSASLMechanisms = "SASLMechanisms"
	BodyTypeSASLOutcome    = "SASLOutcome"
	BodyTypeSASLResponse   = "SASLResponse"
)

type Frame struct {
	// Header is an AMQP frame header. This will be followed with a [ParsedItem] with a [Body].
	Header Header

	// Body is the AMQP frame body.
	Body Body

	// RawData is the original bytes of this frame. It does not get updated if you change the Body or
	// Header fields.
	raw []byte
}

// jsonFrame is the object we use to actually read/write the JSON representation
// of [Frame].
type jsonFrame struct {
	// should always match the exported fields in [Frame]
	Header Header
	Body   json.RawMessage

	// these fields are used only in the JSON serde
	BodyType BodyType

	// This only gets used when the user is off-roading and doing their
	// own encoding/decoding.
	Raw []byte `json:",omitempty"`
}

func NewRawFrame(rawFrame []byte) *Frame {
	return &Frame{raw: rawFrame, Body: &RawFrame{}}
}

func (fr *Frame) MarshalJSON() ([]byte, error) {
	var jf = jsonFrame{
		Header: fr.Header, BodyType: fr.Body.Type(),
	}

	bodyJSON, err := json.Marshal(fr.Body)

	if err != nil {
		return nil, err
	}

	jf.Body = bodyJSON

	// raw body type is used when you specifically are trying to go a bit off-road, and encode
	// a frame, arbitrarily. You might be attempting to break AMQP (by sending invalid data) or
	// encoding things in constraint violating ways. This means we don't want to attempt to parse
	// the payload, so we'll encode it as a raw set of bytes instead.
	if fr.Body.Type() == BodyTypeRawFrame {
		jf.Raw = fr.raw
	}

	return json.Marshal(jf)
}

func (fr *Frame) UnmarshalJSON(data []byte) error {
	// a mirror of the Frame type itself, but with 'Body' as a json.RawMessage
	// attribute instead.
	var tmpFrame *struct {
		Header   Header
		BodyType BodyType
		Body     json.RawMessage // we have to deserialize this bit ourselves
	}

	// first unmarshal the common attributes. Body, which is a polymorphic type,
	// will just be returned as bytes, which we will deserialize in the switch
	// statement below.
	if err := json.Unmarshal(data, &tmpFrame); err != nil {
		return err
	}

	fr.Header = tmpFrame.Header

	switch tmpFrame.BodyType {
	case BodyTypeEmptyFrame, BodyTypeRawFrame:
		if err := unmarshalJSONBody[*EmptyFrame](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeAttach:
		if err := unmarshalJSONBody[*PerformAttach](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeBegin:
		if err := unmarshalJSONBody[*PerformBegin](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeClose:
		if err := unmarshalJSONBody[*PerformClose](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeDetach:
		if err := unmarshalJSONBody[*PerformDetach](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeDisposition:
		if err := unmarshalJSONBody[*PerformDisposition](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeEnd:
		if err := unmarshalJSONBody[*PerformEnd](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeFlow:
		if err := unmarshalJSONBody[*PerformFlow](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeOpen:
		if err := unmarshalJSONBody[*PerformOpen](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeTransfer:
		if err := unmarshalJSONBody[*PerformTransfer](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLChallenge:
		if err := unmarshalJSONBody[*SASLChallenge](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLInit:
		if err := unmarshalJSONBody[*SASLInit](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLMechanisms:
		if err := unmarshalJSONBody[*SASLMechanisms](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLOutcome:
		if err := unmarshalJSONBody[*SASLOutcome](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLResponse:
		if err := unmarshalJSONBody[*SASLResponse](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	default:
		// TODO: allow this for now. We have an inconsistency when we serialize cbs frames vs
		// normal frames because of our censoring.
		// panic(fmt.Sprintf("unexpected frames.Body: %#v", tmpFrame.BodyType))
	}

	return nil
}

// UnmarshalJSON just makes it simpler to deserialize as a specific type.
func unmarshalJSONBody[BodyT Body](body []byte, dest *Body) error {
	var t *BodyT

	if err := json.Unmarshal(body, &t); err != nil {
		return err
	}

	*dest = *t
	return nil
}

// so we can conform to the AMQPItem interface.
func (a *Frame) Raw() []byte {
	return a.raw
}

func (a *Frame) amqpItem() {}

// MustMarshalAMQP marshals this frame into the proper format to send on an AMQP connection
// and panic()'s if an error occurs.
func (a Frame) MustMarshalAMQP() []byte {
	data, err := a.MarshalAMQP()

	if err != nil {
		panic(err)
	}

	return data
}

// MarshalAMQP marshals this frame into the proper format to send on an AMQP connection.
func (a Frame) MarshalAMQP() ([]byte, error) {
	buff := buffer.New(nil)

	// write header
	buff.Append([]byte{
		0, 0, 0, 0, // size, overwrite later
		2,                         // doff, see frameHeader.DataOffset comment
		uint8(a.Header.FrameType), // frame type
	})
	buff.AppendUint16(a.Header.Channel) // channel

	// write AMQP frame body
	err := encoding.Marshal(buff, a.Body)
	if err != nil {
		return nil, err
	}

	// validate size
	if uint(buff.Len()) > math.MaxUint32 {
		return nil, errors.New("frame too large")
	}

	// retrieve raw bytes
	bufBytes := buff.Bytes()

	// write correct size
	binary.BigEndian.PutUint32(bufBytes, uint32(len(bufBytes)))
	return buff.Bytes(), nil
}

type Raw []byte

func (a Raw) amqpItem()   {}
func (a Raw) Raw() []byte { return []byte(a) }

func (a Raw) MarshalAMQP() ([]byte, error) {
	return a.Raw(), nil
}
func (a Raw) MustMarshalAMQP() []byte {
	return a.Raw()
}

type Preamble = Raw
