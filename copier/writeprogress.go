package copier

import (
	"time"
)

// Inspired by: https://golangcode.com/download-a-file-with-progress/

// WriteProgress counts the number of bytes written to it.
// It implements to the io.Writer (and thus goes well with io.TeeReader).
// If freq time has passed, then it calls the given func.
type WriteProgress struct {
	soFar    uint64
	goal     uint64
	freq     time.Duration
	lastTime time.Time
	fn       func(uint64, uint64)
}

func NewWriteProgress(freq time.Duration, fn func(uint64, uint64)) WriteProgress {
	return WriteProgress{
		freq: freq,
		fn:   fn,
	}
}

func (wp *WriteProgress) SetGoal(n uint64) {
	wp.goal = n
}

func (wp *WriteProgress) BytesWritten() uint64 {
	return wp.soFar
}

func (wp *WriteProgress) Write(p []byte) (int, error) {
	// update count
	n := len(p)
	wp.soFar += uint64(n)

	// has enough time passed to log an update?
	if wp.lastTime.IsZero() {
		wp.lastTime = time.Now()
	} else {
		now := time.Now()
		if now.Sub(wp.lastTime) < wp.freq {
			// wait a bit longer before sending an update
			return n, nil
		}
		wp.lastTime = now
	}

	wp.fn(wp.soFar, wp.goal)
	return n, nil
}
