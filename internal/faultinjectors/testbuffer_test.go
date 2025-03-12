package faultinjectors

import (
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
	"github.com/Azure/amqpfaultinjector/internal/utils"
)

type testBuffer struct {
	fb *frames.Buffer
}

func newTestBuffer() *testBuffer {
	return &testBuffer{
		fb: &frames.Buffer{},
	}
}

func (c *testBuffer) Frames() []*frames.Frame {
	var allFrames []*frames.Frame

	for {
		fr, err := c.fb.ExtractFrame() // let's just assume we're only dealing with frames

		if err != nil {
			utils.Panicf("buffer wasn't actually an AMQP frame: %w", err)
		}

		if fr == nil {
			break
		}

		allFrames = append(allFrames, fr)
	}

	return allFrames
}

func (c *testBuffer) Read(p []byte) (n int, err error) {
	utils.Panicf("read not implemented")
	return 0, nil
}

func (c *testBuffer) Write(p []byte) (n int, err error) {
	c.fb.Add(p)
	return len(p), nil
}
