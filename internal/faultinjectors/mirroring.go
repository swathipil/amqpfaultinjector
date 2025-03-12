package faultinjectors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/Azure/amqpfaultinjector/internal/logging"
	"github.com/Azure/amqpfaultinjector/internal/proto"
	"github.com/Azure/amqpfaultinjector/internal/proto/frames"
)

type MirrorParams struct {
	Callback    MirrorCallback
	FrameLogger *logging.FrameLogger

	Local  *frames.ConnReadWriter
	Remote *frames.ConnReadWriter
}

// Mirror mirrors frames, bidirectionally, between local <-> remote.
// See [MirrorCallback] for the expected contract, including how to terminate mirroring.
func Mirror(ctx context.Context, params MirrorParams) error {
	m := newMirror(params)
	return m.Serve(ctx)
}

type mirror struct {
	local, remote *frames.ConnReadWriter
	frameLogger   *logging.FrameLogger
	sm            *proto.StateMap
	callback      MirrorCallback
}

func newMirror(params MirrorParams) *mirror {
	return &mirror{
		callback:    params.Callback,
		frameLogger: params.FrameLogger,
		local:       params.Local,
		remote:      params.Remote,
		sm:          proto.NewStateMap(),
	}
}

// Serve starts the bidirectional mirroring between source <-> dest.
func (m *mirror) Serve(ctx context.Context) error {
	wg := sync.WaitGroup{}

	var localErr, remoteErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		localErr = m.uniMirror(ctx, true)
		slog.Info("Done mirroring local -> remote", "error", localErr)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		remoteErr = m.uniMirror(ctx, false)
		slog.Info("Done mirroring remote -> local", "error", remoteErr)
	}()

	wg.Wait()

	if localErr != nil {
		return localErr
	}

	if remoteErr != nil {
		return remoteErr
	}

	return nil
}

// processMetaFrame takes cares of logging and sending the frame to the appropriate destination.
func (m *mirror) processMetaFrame(out bool, metaFrame *MetaFrame) error {
	if m.frameLogger != nil {
		// TODO: make this better too - I just want to log all the metadata attributes, apart
		// from the frame itself.
		md := *metaFrame
		md.Frame = nil

		if err := m.frameLogger.AddFrame(out, metaFrame.Frame, md); err != nil {
			slog.Warn("Failed adding packet to logger", "error", err)
		}
	}

	switch metaFrame.Action {
	case MetaFrameActionDropped:
		// logged only, does not get sent.
		return nil
	case MetaFrameActionPassthrough:
		m.sm.AddFrame(out, metaFrame.Frame)

		if out {
			// write the resulting frame to the connection
			if err := m.remote.Write(frames.Raw(metaFrame.Frame.Raw())); err != nil {
				return err
			}
		} else {
			// write the resulting frame to the connection
			if err := m.local.Write(frames.Raw(metaFrame.Frame.Raw())); err != nil {
				return err
			}
		}
	case MetaFrameActionAdded, MetaFrameActionModified:
		// write out the packet now
		finalOut := out

		if metaFrame.OverrideOut != nil {
			// user wants to choose the stream to write to
			finalOut = *metaFrame.OverrideOut
		}

		m.sm.AddFrame(finalOut, metaFrame.Frame)

		if finalOut {
			// write the resulting frame to the connection
			if err := m.remote.Write(metaFrame.Frame); err != nil {
				return err
			}
		} else {
			// write the resulting frame to the connection
			if err := m.local.Write(metaFrame.Frame); err != nil {
				return err
			}
		}
	default:
		panic(fmt.Sprintf("unexpected FaultInjectorAction: %#v", metaFrame.Action))
	}

	return nil
}

// uniMirror mirrors from one connection to another, in a single direction.
func (m *mirror) uniMirror(ctx context.Context, out bool) error {
	ctx, _ = logging.ContextWithSloggerAndValues(ctx, "out", out)

	source, dest := m.remote, m.local

	if out {
		source, dest = m.local, m.remote
	}

	for item, err := range source.Iter() {
		if err != nil {
			return fmt.Errorf("failed to read next item in mirroring loop: %w", err)
		}

		fr, isFrame := item.(*frames.Frame)

		if !isFrame { // ie, an AMQP preamble
			if err := dest.Write(item); err != nil { // just write it, no mirror-er cares to inspect it.
				return fmt.Errorf("failed to write preamble in mirroring loop: %w", err)
			}

			continue
		}

		metaFrames, err := m.callback(ctx, MirrorCallbackParams{
			Out:      out,
			Frame:    fr,
			StateMap: m.sm,
		})

		stop, err := m.handleCallbackResult(out, metaFrames, err)

		if err != nil {
			return err
		}

		if stop {
			return nil
		}
	}

	return nil
}

func (m *mirror) handleCallbackResult(out bool, metaFrames []MetaFrame, err error) (bool, error) {
	stop := false

	switch {
	case errors.Is(err, io.EOF):
		slog.Debug("callback has returned io.EOF")

		// even if they pass io.EOF they still might have some work for us to do. We'll
		// bail after all frames have been processed.
		stop = true
	case err != nil:
		return false, fmt.Errorf("error from mirroring callback: %w", err)
	}

	for _, metaFrame := range metaFrames {
		if metaFrame.Delay == 0 {
			if err := m.processMetaFrame(out, &metaFrame); err != nil {
				return false, fmt.Errorf("failed to processMetaFrame: %w", err)
			}
		} else {
			time.AfterFunc(metaFrame.Delay, func() {
				if err := m.processMetaFrame(out, &metaFrame); err != nil {
					slog.Error("failed to processMetaFrame", "error", err)
				}
			})
		}
	}

	return stop, nil
}
