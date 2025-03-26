package frames

import (
	"errors"
	"io"
	"iter"

	"github.com/richardpark-msft/amqpfaultinjector/internal/utils"
)

// ConnReadWriter makes it simple to write or read from a stream with frames or AMQP
// preambles.
type ConnReadWriter struct {
	conn      io.ReadWriter
	chunkSize int
	fb        *Buffer
}

type ConnReadWriterOption func(fi *ConnReadWriter) error

func NewConnReadWriter(conn io.ReadWriter, options ...ConnReadWriterOption) *ConnReadWriter {
	fi := &ConnReadWriter{
		chunkSize: 1024 * 1024,
		conn:      conn,
		fb:        &Buffer{},
	}

	for _, opt := range options {
		if err := opt(fi); err != nil {
			utils.Panicf("invalid option: %w", err)
		}
	}

	return fi
}

func (fc *ConnReadWriter) Iter() iter.Seq2[PreambleOrFrame, error] {
	return func(yield func(PreambleOrFrame, error) bool) {
		// do init to get the collection ready
		chunk := make([]byte, fc.chunkSize)
		eof := false

	IterationLoop:
		for {
			for {
				item, err := fc.fb.Extract()

				if err != nil {
					yield(nil, err)
					break IterationLoop
				}

				if item != nil {
					if !yield(item, err) {
						break IterationLoop
					}

					continue
				}

				break
			}

			if eof {
				break IterationLoop
			}

			n, err := fc.conn.Read(chunk)

			switch {
			case errors.Is(err, io.EOF):
				eof = true // process whatever data is left but after this there's no more.
			case err != nil:
				yield(nil, err)
				break IterationLoop
			}

			fc.fb.Add(chunk[0:n])
		}
	}
}

// TODO: remove
func (fc *ConnReadWriter) Write(data interface{ MarshalAMQP() ([]byte, error) }) error {
	buff, err := data.MarshalAMQP()

	if err != nil {
		return err
	}

	return fc.WriteBytes(buff)
}

func (fc *ConnReadWriter) WriteBytes(data []byte) error {
	_, err := fc.conn.Write(data)
	return err
}
