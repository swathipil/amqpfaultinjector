package frames

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"

	"github.com/Azure/amqpfaultinjector/internal/proto/buffer"
	"github.com/Azure/amqpfaultinjector/internal/proto/encoding"
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
	BodyTypeEmptyFrame     = "EmptyFrame"
	BodyTypeAttach         = "Attach"
	BodyTypeBegin          = "Begin"
	BodyTypeClose          = "Close"
	BodyTypeDetach         = "Detach"
	BodyTypeDisposition    = "Disposition"
	BodyTypeEnd            = "End"
	BodyTypeFlow           = "Flow"
	BodyTypeOpen           = "Open"
	BodyTypeTransfer       = "Transfer"
	BodyTypeSASLChallenge  = "SASLChallenge"
	BodyTypeSASLInit       = "SASLInit"
	BodyTypeSASLMechanisms = "SASLMechanisms"
	BodyTypeSASLOutcome    = "SASLOutcome"
	BodyTypeSASLResponse   = "SASLResponse"
)

type Frame struct {
	BodyType BodyType `json:"-"`

	// Header is an AMQP frame header. This will be followed with a [ParsedItem] with a [Body].
	Header Header

	// Body is the AMQP frame body.
	Body Body

	// RawData is the original bytes of this frame. It does not get updated if you change the Body or
	// Header fields.
	raw []byte
}

// UnmarshalJSON just makes it simpler to deserialize as a specific type.
func UnmarshalJSONBody[BodyT Body](body []byte, dest *Body) error {
	var t *BodyT

	if err := json.Unmarshal(body, &t); err != nil {
		return err
	}

	*dest = *t
	return nil
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

	switch tmpFrame.BodyType {
	case BodyTypeEmptyFrame:
		if err := UnmarshalJSONBody[*EmptyFrame](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeAttach:
		if err := UnmarshalJSONBody[*PerformAttach](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeBegin:
		if err := UnmarshalJSONBody[*PerformBegin](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeClose:
		if err := UnmarshalJSONBody[*PerformClose](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeDetach:
		if err := UnmarshalJSONBody[*PerformDetach](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeDisposition:
		if err := UnmarshalJSONBody[*PerformDisposition](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeEnd:
		if err := UnmarshalJSONBody[*PerformEnd](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeFlow:
		if err := UnmarshalJSONBody[*PerformFlow](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeOpen:
		if err := UnmarshalJSONBody[*PerformOpen](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeTransfer:
		if err := UnmarshalJSONBody[*PerformTransfer](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLChallenge:
		if err := UnmarshalJSONBody[*SASLChallenge](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLInit:
		if err := UnmarshalJSONBody[*SASLInit](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLMechanisms:
		if err := UnmarshalJSONBody[*SASLMechanisms](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLOutcome:
		if err := UnmarshalJSONBody[*SASLOutcome](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	case BodyTypeSASLResponse:
		if err := UnmarshalJSONBody[*SASLResponse](tmpFrame.Body, &fr.Body); err != nil {
			return err
		}
	default:
		// TODO: allow this for now. We have an inconsistency when we serialize cbs frames vs
		// normal frames because of our censoring.
		// panic(fmt.Sprintf("unexpected frames.Body: %#v", tmpFrame.BodyType))
	}

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
