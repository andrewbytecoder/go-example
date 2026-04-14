// Package uvgo provides a minimal Go analogue of libuv’s event loop,
// TCP listen/accept (uv_tcp_t + uv_listen), UDP (uv_udp_t), repeating/one-shot
// timers (uv_timer_t), stat-based file polling (uv_fs_poll_t), uv_run phases,
// and loop metadata (uv_loop_init / uv_now / uv_stop). It is not API-compatible
// with libuv.
package uvgo

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Loop is the rough equivalent of uv_loop_t: one goroutine runs Run / RunMode and
// dispatches work in uv_run order (pending → idle → prepare → poll → check).
type Loop struct {
	ctx    context.Context
	cancel context.CancelFunc

	createdAt time.Time

	mu       sync.Mutex
	stopped  bool
	stopFlag atomic.Bool

	timerOnce sync.Once

	cachedNow atomic.Uint64

	pendingQ chan func()

	prepareHooks []*phaseHook
	idleHooks    []*phaseHook
	checkHooks   []*phaseHook

	acceptQ chan net.Conn
	fire    chan *timerHandle

	listeners []net.Listener
	timers    []*timerHandle
	timerWake chan struct{}

	onTCPConn func(net.Conn)

	fspollQ chan *fspollJob

	udpIn       chan udpInbound
	udpSendDone chan udpSendDone
	udps        []*UDP

	wg sync.WaitGroup
}

// NewLoop creates a loop bound to ctx. Cancel ctx to unblock Run.
func NewLoop(ctx context.Context) *Loop {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	l := &Loop{
		createdAt:   time.Now(),
		ctx:         ctx,
		cancel:      cancel,
		pendingQ:    make(chan func(), 1024),
		acceptQ:     make(chan net.Conn, 64),
		fire:        make(chan *timerHandle, 32),
		timerWake:   make(chan struct{}, 1),
		fspollQ:     make(chan *fspollJob, 64),
		udpIn:       make(chan udpInbound, 256),
		udpSendDone: make(chan udpSendDone, 64),
	}
	l.UpdateTime()
	return l
}

func (l *Loop) startTimerScheduler() {
	l.timerOnce.Do(func() {
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			l.timerScheduler()
		}()
	})
}

// shutdownSockets closes TCP listeners and UDP handles (unix loop teardown +
// open handles). Safe to call multiple times.
func (l *Loop) shutdownSockets() {
	l.mu.Lock()
	ls := l.listeners
	l.listeners = nil
	udps := l.udps
	l.udps = nil
	l.mu.Unlock()

	for _, ln := range ls {
		_ = ln.Close()
	}
	for _, u := range udps {
		_ = u.Close()
	}
}

// Stop closes listeners and UDP handles, cancels the loop context, and matches
// a full uv_loop_close preparation step for this subset.
func (l *Loop) Stop() {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return
	}
	l.stopped = true
	l.mu.Unlock()

	l.shutdownSockets()
	l.cancel()
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

func (l *Loop) drainUDPQueuesNoop() {
	for {
		select {
		case <-l.udpIn:
		default:
			goto send
		}
	}
send:
	for {
		select {
		case d := <-l.udpSendDone:
			if d.cb != nil {
				d.cb(d.err)
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
