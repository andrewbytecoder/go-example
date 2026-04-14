// Package uvgo provides a minimal Go analogue of libuv’s event loop,
// TCP listen/accept path (uv_tcp_t + uv_listen), and repeating/one-shot
// timers (uv_timer_t). It is not API-compatible with libuv; it mirrors the
// ideas with idiomatic Go and the standard library.
package uvgo

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// Loop is the rough equivalent of uv_loop_t: one goroutine runs Run and
// dispatches accepted connections and timer callbacks sequentially.
type Loop struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	stopped bool
	acceptQ chan net.Conn
	fire    chan *timerHandle

	listeners []net.Listener
	timers    []*timerHandle
	timerWake chan struct{}

	onTCPConn func(net.Conn)

	wg sync.WaitGroup
}

// NewLoop creates a loop bound to ctx. Cancel ctx to unblock Run.
func NewLoop(ctx context.Context) *Loop {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Loop{
		ctx:       ctx,
		cancel:    cancel,
		acceptQ:   make(chan net.Conn, 64),
		fire:      make(chan *timerHandle, 32),
		timerWake: make(chan struct{}, 1),
	}
}

// Stop closes listeners and cancels the loop context. Safe to call from any
// goroutine; Run will return after draining pending work.
func (l *Loop) Stop() {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return
	}
	l.stopped = true
	ls := l.listeners
	l.listeners = nil
	l.mu.Unlock()

	for _, ln := range ls {
		_ = ln.Close()
	}
	l.cancel()
}

// Run processes accepts and timers until Stop or ctx cancel. Callbacks run
// on the same goroutine as Run (like libuv’s default thread model).
func (l *Loop) Run() error {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		l.timerScheduler()
	}()

	for {
		select {
		case <-l.ctx.Done():
			l.drainAccepts()
			l.wg.Wait()
			if errors.Is(l.ctx.Err(), context.Canceled) {
				return nil
			}
			return l.ctx.Err()

		case c := <-l.acceptQ:
			if c == nil {
				continue
			}
			if l.onTCPConn != nil {
				l.onTCPConn(c)
			} else {
				_ = c.Close()
			}

		case th := <-l.fire:
			if th == nil || !th.active {
				continue
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
	}
}

func (l *Loop) drainAccepts() {
	for {
		select {
		case c := <-l.acceptQ:
			if c != nil {
				_ = c.Close()
			}
		default:
			return
		}
	}
}

func (l *Loop) rescheduleTimer(th *timerHandle) {
	l.mu.Lock()
	found := false
	for _, x := range l.timers {
		if x == th {
			found = true
			break
		}
	}
	if !found {
		l.timers = append(l.timers, th)
	}
	l.mu.Unlock()
	l.pingScheduler()
}

func (l *Loop) pingScheduler() {
	select {
	case l.timerWake <- struct{}{}:
	default:
	}
}

func (l *Loop) nextTimerDeadline() (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var min time.Time
	var ok bool
	now := time.Now()
	for _, th := range l.timers {
		if !th.active {
			continue
		}
		if !ok || th.next.Before(min) {
			min = th.next
			ok = true
		}
	}
	if !ok {
		return time.Time{}, false
	}
	if min.Before(now) {
		return now, true
	}
	return min, true
}

func (l *Loop) timerScheduler() {
	var cur *time.Timer
	defer func() {
		if cur != nil {
			cur.Stop()
		}
	}()

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-l.timerWake:
		}

		if cur != nil {
			if !cur.Stop() {
				select {
				case <-cur.C:
				default:
				}
			}
		}

		deadline, ok := l.nextTimerDeadline()
		if !ok {
			cur = nil
			continue
		}
		d := time.Until(deadline)
		if d < 0 {
			d = 0
		}
		cur = time.NewTimer(d)

		select {
		case <-l.ctx.Done():
			if !cur.Stop() {
				<-cur.C
			}
			return
		case <-l.timerWake:
			if !cur.Stop() {
				select {
				case <-cur.C:
				default:
				}
			}
			continue
		case <-cur.C:
			l.fireDueTimers()
		}
	}
}

func (l *Loop) fireDueTimers() {
	now := time.Now()
	l.mu.Lock()
	var due []*timerHandle
	remain := l.timers[:0]
	for _, th := range l.timers {
		if !th.active {
			continue
		}
		if !th.next.After(now) {
			due = append(due, th)
			continue
		}
		remain = append(remain, th)
	}
	l.timers = remain
	l.mu.Unlock()

	for _, th := range due {
		select {
		case l.fire <- th:
		case <-l.ctx.Done():
			return
		}
	}
}

var (
	// ErrStopped is returned when starting listen on a stopped loop.
	ErrStopped = errors.New("uvgo: loop stopped")
)
