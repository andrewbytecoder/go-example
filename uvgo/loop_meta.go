package uvgo

import (
	"time"
)

// Loop holds minimal uv_loop_t–style bookkeeping that unix loop.c wires up
// during uv_loop_init: a cached millisecond clock (uv_now via UpdateTime) and
// a stop flag (uv_stop / loop->stop_flag).

// UpdateTime corresponds to uv_update_time: refreshes the cached timestamp used
// by Now. The event loop calls this each iteration; you may call it manually
// when implementing custom backends.
func (l *Loop) UpdateTime() {
	if l == nil {
		return
	}
	l.cachedNow.Store(uint64(time.Since(l.createdAt).Milliseconds()))
}

// Now returns milliseconds since the loop was allocated (uv_now). The value is
// updated when UpdateTime runs (including once per RunMode iteration).
func (l *Loop) Now() uint64 {
	if l == nil {
		return 0
	}
	v := l.cachedNow.Load()
	if v == 0 {
		return uint64(time.Since(l.createdAt).Milliseconds())
	}
	return v
}

// StopFlag returns loop->stop_flag. It is set by RequestStop (uv_stop).
func (l *Loop) StopFlag() bool {
	if l == nil {
		return false
	}
	return l.stopFlag.Load()
}

// RequestStop corresponds to uv_stop: sets stop_flag and cancels the loop
// context so Run returns. The next Run clears stop_flag. Pending listeners and
// UDP sockets are closed in Run when tearing down so blocked I/O can finish
// (same path as parent-context cancellation).
func (l *Loop) RequestStop() {
	if l == nil {
		return
	}
	l.stopFlag.Store(true)
	l.cancel()
}
