package frames

import (
	"encoding/binary"
	"fmt"

	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/buffer"
)

type Buffer struct {
	currentHeader      *Header
	currentHeaderBytes []byte
	buff               []byte
	offset             int
}

// Add adds connection data. Use [Extract] to extract frames from the added data.
// NOTE: this call cannot be used concurrently.
func (fb *Buffer) Add(buff []byte) {
	fb.buff = append(fb.buff, buff...)
}

// ExtractFrame extracts an AMQPFrame, if available.
// NOTE: this function should only be used when you're certain the underlying stream will not contain
// an AMQP preamble. It's only safe to use when SASL negotiation has completed.
// It returns (nil, nil) if there are no longer any items to parse with the buffered data it has remaining.
func (fb *Buffer) ExtractFrame() (*Frame, error) {
	amqpItem, err := fb.Extract()

	if err != nil {
		return nil, err
	}

	if amqpItem == nil {
		return nil, nil
	}

	v, ok := amqpItem.(*Frame)

	if !ok {
		return nil, fmt.Errorf("underlying type was %T, not an *AMQPItem. If the stream can also contain an AMQP preamble you must use Extract() instead.", amqpItem)
	}

	return v, nil
}

// Extract will return any parsed items (AMQP preamble, frame headers or frames)
// It returns (nil, nil) if there are no longer any items to parse with the buffered data it has remaining.
func (fb *Buffer) Extract() (PreambleOrFrame, error) {
	extractFrame := func() (PreambleOrFrame, error) {
		// we're attempting to parse a frame (we already have a header)
		body, n, err := parseFrameBody(fb.currentHeader, fb.buff)

		if err != nil {
			return nil, fmt.Errorf("failed parsing frame body at offset %d: %w", fb.offset, err)
		}

		if body == nil {
			// not enough bytes, and we're only looking for a body
			return nil, nil
		}

		// combine the header bytes and the frame bytes for the entire "raw" frame.
		rawBuff := fb.currentHeaderBytes
		rawBuff = append(rawBuff, fb.buff[0:n]...)

		amqpItem := &Frame{
			Header: *fb.currentHeader,
			Body:   body,
			raw:    rawBuff,
		}

		fb.buff = fb.buff[n:]
		fb.offset += n
		fb.currentHeader = nil

		return amqpItem, nil
	}

	if fb.currentHeader != nil {
		return extractFrame()
	}

	// we're in between frames - we're looking for either the start of a frame
	// or the AMQP preamble, which can show up in between frames when we're doing SASL
	// negotiation.
	sigBytes := parseAMQPSig(fb.buff)

	if sigBytes != nil {
		fb.buff = fb.buff[len(sigBytes):]
		fb.offset += len(sigBytes)

		return Preamble(sigBytes), nil
	}

	header, n, err := parseHeader(fb.buff)

	if err != nil {
		return nil, fmt.Errorf("failed parsing frame header at offset %d: %w", fb.offset, err)
	}

	if header == nil {
		return nil, nil
	}

	fb.currentHeaderBytes = fb.buff[0:n]
	fb.currentHeader = header
	fb.buff = fb.buff[n:]
	fb.offset += len(fb.currentHeaderBytes)

	return extractFrame()
}

func parseAMQPSig(buff []byte) []byte {
	if len(buff) < 8 {
		return nil // ie: not enough data to grab the AMQP preamble
	}

	if buff[0] == 'A' && buff[1] == 'M' && buff[2] == 'Q' && buff[3] == 'P' {
		return buff[0:8]
	}

	return nil
}

func parseHeader(buf []byte) (*Header, int, error) {
	if len(buf) < 8 {
		return nil, 0, nil // ie: not enough data to grab a header
	}

	fh := Header{
		Size:       binary.BigEndian.Uint32(buf[0:4]),
		DataOffset: buf[4],
		FrameType:  buf[5],
		Channel:    binary.BigEndian.Uint16(buf[6:8]),
		Raw:        buf[0:8],
	}

	// NOTE: we're assuming nobody is sending extended header attributes, but if that should
	// change we only need to modify that assumption here (and in what we return for the size
	// of the header, overall))
	if fh.Size < HeaderSize {
		return nil, 0, fmt.Errorf("received frame header with invalid size %d", fh.Size)
	}

	if fh.DataOffset < 2 {
		return nil, 0, fmt.Errorf("received frame header with invalid data offset %d", fh.DataOffset)
	}

	return &fh, HeaderSize, nil
}

func parseFrameBody(header *Header, buf []byte) (Body, int, error) {
	frameBodySize := int(header.Size) - len(header.Raw)

	if len(buf) < frameBodySize {
		return nil, 0, nil
	}

	if frameBodySize == 0 {
		// this is an empty frame!
		return &EmptyFrame{}, frameBodySize, nil
	}

	buff := buf[0:frameBodySize]

	tmp := buffer.New(buff)
	frameBody, err := parseBody(tmp)

	if err != nil {
		return nil, 0, err
	}

	return frameBody, frameBodySize, nil
}
