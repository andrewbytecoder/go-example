package uvgo

import (
	"time"
)

type timerHandle struct {
	next   time.Time
	repeat time.Duration
	cb     func()
	active bool
}

// Timer is the analogue of uv_timer_t (opaque handle).
type Timer struct {
	h *timerHandle
	l *Loop
}

// TimerStart registers a callback similar to uv_timer_start:
//   - first is the delay before the first invocation;
//   - repeat is the interval between later invocations; 0 means one-shot.
//
// The callback always runs on the goroutine that called Loop.Run.
//
// Safe to call before or after Run starts.
func (l *Loop) TimerStart(first, repeat time.Duration, cb func()) *Timer {
	th := &timerHandle{
		next:   time.Now().Add(first),
		repeat: repeat,
		cb:     cb,
		active: true,
	}
	l.mu.Lock()
	l.timers = append(l.timers, th)
	l.mu.Unlock()
	l.pingScheduler()
	return &Timer{h: th, l: l}
}

// Stop corresponds to uv_timer_stop: cancel further callbacks.
func (t *Timer) Stop() {
	if t == nil || t.h == nil || t.l == nil {
		return
	}
	l := t.l
	l.mu.Lock()
	if t.h != nil {
		t.h.active = false
	}
	l.mu.Unlock()
	l.pingScheduler()
}
