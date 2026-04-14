package uvgo

import (
	"context"
	"errors"
	"net"
	"time"
)

// RunMode mirrors uv_run_mode (see src/unix/core.c uv_run).
type RunMode int

const (
	RunDefault RunMode = iota // UV_RUN_DEFAULT: run until no work / stop / cancel
	RunOnce                   // UV_RUN_ONCE: one iteration (phases + at most one blocking poll)
	RunNoWait                 // UV_RUN_NOWAIT: phases + non-blocking poll only
)

// phaseHook is a registered prepare/idle/check callback (like uv_prepare_t).
type phaseHook struct {
	fn func()
}

// RunMode runs an analogue of uv_run (src/unix/core.c): each iteration runs
// pending → idle → prepare → poll (blocking, one event, or non-blocking drain)
// → check. RunDefault loops until context cancel, RequestStop, or Stop. libuv
// also stops when uv__loop_alive is false; this port does not auto-exit on
// empty Alive() to avoid surprising context lifetime.
func (l *Loop) RunMode(mode RunMode) error {
	l.stopFlag.Store(false)
	l.UpdateTime()

	l.startTimerScheduler()

	for {
		if err := l.ctx.Err(); err != nil {
			l.teardownAfterRun(true)
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		if l.stopFlag.Load() {
			l.stopFlag.Store(false)
			l.teardownAfterRun(true)
			return nil
		}

		l.UpdateTime()

		l.runPendingPhase()
		l.runIdlePhase()
		l.runPreparePhase()

		switch mode {
		case RunNoWait:
			l.pollNoWait()
		case RunOnce, RunDefault:
			if !l.pollOneBlocking() {
				err := l.ctx.Err()
				l.teardownAfterRun(true)
				if err != nil && errors.Is(err, context.Canceled) {
					return nil
				}
				if err != nil {
					return err
				}
				return nil
			}
		}

		l.runCheckPhase()

		if mode == RunOnce || mode == RunNoWait {
			return nil
		}
	}
}

// Run is equivalent to RunMode(RunDefault).
func (l *Loop) Run() error {
	return l.RunMode(RunDefault)
}

// Post schedules fn on the next event-loop iteration (pending queue, uv__run_pending
// analogue). Safe from any goroutine; drops if the queue is full.
func (l *Loop) Post(fn func()) {
	if fn == nil {
		return
	}
	select {
	case l.pendingQ <- fn:
	default:
	}
}

// Prepare registers a callback invoked after pending and idle, before each poll
// (uv_prepare_t). Returns unregister to remove it.
func (l *Loop) Prepare(fn func()) (unregister func()) {
	if fn == nil {
		return func() {}
	}
	h := &phaseHook{fn: fn}
	l.mu.Lock()
	l.prepareHooks = append(l.prepareHooks, h)
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		out := make([]*phaseHook, 0, len(l.prepareHooks))
		for _, x := range l.prepareHooks {
			if x != h {
				out = append(out, x)
			}
		}
		l.prepareHooks = out
		l.mu.Unlock()
	}
}

// Idle registers uv_idle_t-style callbacks (run after pending, before prepare in
// libuv order is pending→idle→prepare — matching core.c).
func (l *Loop) Idle(fn func()) (unregister func()) {
	if fn == nil {
		return func() {}
	}
	h := &phaseHook{fn: fn}
	l.mu.Lock()
	l.idleHooks = append(l.idleHooks, h)
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		out := make([]*phaseHook, 0, len(l.idleHooks))
		for _, x := range l.idleHooks {
			if x != h {
				out = append(out, x)
			}
		}
		l.idleHooks = out
		l.mu.Unlock()
	}
}

// Check registers uv_check_t-style callbacks (after poll, before next iteration).
func (l *Loop) Check(fn func()) (unregister func()) {
	if fn == nil {
		return func() {}
	}
	h := &phaseHook{fn: fn}
	l.mu.Lock()
	l.checkHooks = append(l.checkHooks, h)
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		out := make([]*phaseHook, 0, len(l.checkHooks))
		for _, x := range l.checkHooks {
			if x != h {
				out = append(out, x)
			}
		}
		l.checkHooks = out
		l.mu.Unlock()
	}
}

// Alive approximates uv_loop_alive: listeners, UDP sockets, or active timers.
func (l *Loop) Alive() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.listeners) > 0 || len(l.udps) > 0 {
		return true
	}
	for _, th := range l.timers {
		if th != nil && th.active {
			return true
		}
	}
	return false
}

// --- internal ---

func (l *Loop) runPendingPhase() {
	for {
		select {
		case fn := <-l.pendingQ:
			if fn != nil {
				fn()
			}
		default:
			return
		}
	}
}

func (l *Loop) runIdlePhase() {
	l.runHooks(&l.idleHooks)
}

func (l *Loop) runPreparePhase() {
	l.runHooks(&l.prepareHooks)
}

func (l *Loop) runCheckPhase() {
	l.runHooks(&l.checkHooks)
}

func (l *Loop) runHooks(hooks *[]*phaseHook) {
	l.mu.Lock()
	hs := append([]*phaseHook(nil), *hooks...)
	l.mu.Unlock()
	for _, h := range hs {
		if h != nil && h.fn != nil {
			h.fn()
		}
	}
}

func (l *Loop) pollNoWait() {
	for {
		select {
		case <-l.ctx.Done():
			return
		case c := <-l.acceptQ:
			l.dispatchAccept(c)
		case job := <-l.fspollQ:
			if job != nil {
				l.fspollDispatch(job)
			}
		case in := <-l.udpIn:
			l.dispatchUDPIn(in)
		case sd := <-l.udpSendDone:
			l.dispatchSendDone(sd)
		case th := <-l.fire:
			l.dispatchTimer(th)
		default:
			return
		}
	}
}

// pollOneBlocking waits for one event. Returns false if the loop should re-check ctx (ctx done).
func (l *Loop) pollOneBlocking() bool {
	select {
	case <-l.ctx.Done():
		return false
	case c := <-l.acceptQ:
		l.dispatchAccept(c)
	case job := <-l.fspollQ:
		if job != nil {
			l.fspollDispatch(job)
		}
	case in := <-l.udpIn:
		l.dispatchUDPIn(in)
	case sd := <-l.udpSendDone:
		l.dispatchSendDone(sd)
	case th := <-l.fire:
		l.dispatchTimer(th)
	}
	return true
}

func (l *Loop) dispatchAccept(c net.Conn) {
	if c == nil {
		return
	}
	if l.onTCPConn != nil {
		l.onTCPConn(c)
	} else {
		_ = c.Close()
	}
}

func (l *Loop) dispatchSendDone(sd udpSendDone) {
	if sd.cb != nil {
		sd.cb(sd.err)
	}
}

func (l *Loop) dispatchTimer(th *timerHandle) {
	if th == nil || !th.active {
		return
	}
	if th.cb != nil {
		th.cb()
	}
	if th.repeat > 0 {
		th.next = time.Now().Add(th.repeat)
		l.rescheduleTimer(th)
	} else {
		th.active = false
	}
}

func (l *Loop) teardownAfterRun(full bool) {
	if !full {
		return
	}
	l.shutdownSockets()
	l.drainAccepts()
	l.drainUDPQueuesNoop()
	l.wg.Wait()
}
