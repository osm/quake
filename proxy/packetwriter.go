package proxy

import (
	"errors"
	"sync"
	"time"
)

type packetWriter struct {
	mu        sync.Mutex
	queue     [][]byte
	holdUntil time.Time
	closed    bool

	wake chan struct{}
	done chan struct{}

	write   func([]byte) error
	onError func(error)
	onIdle  func(*packetWriter) bool
}

func newPacketWriter(
	write func([]byte) error,
	onError func(error),
	onIdle func(*packetWriter) bool,
) *packetWriter {
	w := &packetWriter{
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		write:   write,
		onError: onError,
		onIdle:  onIdle,
	}
	go w.run()
	return w
}

// DelayCLCFor holds the current and subsequently received sequenced
// client-to-server packets until delay has elapsed. Repeated calls extend an
// existing hold but never shorten it.
//
// A CLC handler can call this before Serve forwards the packet that invoked the
// handler.
func (c *Client) DelayCLCFor(delay time.Duration) {
	if delay <= 0 {
		return
	}

	c.clcWriterMu.Lock()
	defer c.clcWriterMu.Unlock()

	if c.conn == nil {
		return
	}
	if c.clcWriter == nil {
		c.clcWriter = newPacketWriter(
			func(data []byte) error {
				c.clcWriterMu.Lock()
				defer c.clcWriterMu.Unlock()
				return c.writeCLCUnlocked(data)
			},
			func(err error) {
				if c.logger != nil {
					c.logger.Printf("unable to write command, %v\n", err)
				}
			},
			c.closeIdleCLCWriter,
		)
	}
	c.clcWriter.holdFor(delay)
}

func (c *Client) writeCLC(data []byte) error {
	c.clcWriterMu.Lock()
	defer c.clcWriterMu.Unlock()

	if c.clcWriter != nil {
		c.clcWriter.enqueue(data)
		return nil
	}

	return c.writeCLCUnlocked(data)
}

func (c *Client) writeCLCUnlocked(data []byte) error {
	if c.conn == nil {
		return errors.New("client CLC writer unavailable")
	}

	_, err := c.conn.Write(data)
	return err
}

func (c *Client) closeCLCWriter() {
	c.clcWriterMu.Lock()
	defer c.clcWriterMu.Unlock()

	if c.clcWriter != nil {
		c.clcWriter.close()
		c.clcWriter = nil
	}
}

func (c *Client) closeIdleCLCWriter(writer *packetWriter) bool {
	c.clcWriterMu.Lock()
	defer c.clcWriterMu.Unlock()

	if c.clcWriter != writer {
		writer.close()
		return true
	}
	if !writer.closeIfIdle() {
		return false
	}

	c.clcWriter = nil
	return true
}

func (w *packetWriter) holdFor(delay time.Duration) {
	if delay <= 0 {
		return
	}

	w.mu.Lock()
	if !w.closed {
		until := time.Now().Add(delay)
		if until.After(w.holdUntil) {
			w.holdUntil = until
		}
	}
	w.mu.Unlock()
	w.signal()
}

func (w *packetWriter) enqueue(data []byte) {
	packet := append([]byte(nil), data...)

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.queue = append(w.queue, packet)
	w.mu.Unlock()
	w.signal()
}

func (w *packetWriter) close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.queue = nil
	close(w.done)
	w.mu.Unlock()
}

func (w *packetWriter) closeIfIdle() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return true
	}
	if len(w.queue) > 0 || time.Until(w.holdUntil) > 0 {
		return false
	}

	w.closed = true
	w.queue = nil
	close(w.done)
	return true
}

func (w *packetWriter) signal() {
	select {
	case w.wake <- struct{}{}:
	case <-w.done:
	default:
	}
}

func (w *packetWriter) run() {
	for {
		packet, wait, ok, closed := w.next()
		if closed {
			return
		}
		if !ok {
			if w.onIdle != nil && w.onIdle(w) {
				return
			}

			select {
			case <-w.wake:
				continue
			case <-w.done:
				return
			}
		}

		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-w.wake:
				stopTimer(timer)
			case <-w.done:
				stopTimer(timer)
				return
			}
			continue
		}

		if err := w.write(packet); err != nil && w.onError != nil {
			w.onError(err)
		}
	}
}

func (w *packetWriter) next() ([]byte, time.Duration, bool, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil, 0, false, true
	}

	if wait := time.Until(w.holdUntil); wait > 0 {
		return nil, wait, true, false
	}

	if len(w.queue) == 0 {
		return nil, 0, false, false
	}

	packet := w.queue[0]
	w.queue[0] = nil
	w.queue = w.queue[1:]
	return packet, 0, true, false
}

func stopTimer(timer *time.Timer) {
	if timer.Stop() {
		return
	}

	select {
	case <-timer.C:
	default:
	}
}
