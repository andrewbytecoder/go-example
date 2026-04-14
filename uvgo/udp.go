package uvgo

import (
	"errors"
	"net"
	"sync"
	"syscall"
)

// UDPFlags mirrors libuv’s recv flags subset used here (extensible).
type UDPFlags uint

// UDPRecvCallback matches uv_udp_recv_cb: nread is byte count on success;
// on failure nread < 0 and buf/addr are nil (errno is passed as -nread when known).
type UDPRecvCallback func(u *UDP, nread int, buf []byte, addr *net.UDPAddr, flags UDPFlags)

// UDP is a small analogue of uv_udp_t: bound UDP socket with recv delivered on
// Loop.Run and send completion callbacks serialized there as well.
type UDP struct {
	mu     sync.Mutex
	loop   *Loop
	conn   *net.UDPConn
	recvCB UDPRecvCallback
	closed bool
}

// ListenUDP binds like uv_udp_init + uv_udp_bind, then starts recv like
// uv_udp_recv_start. Each datagram is copied; callbacks run on the Run goroutine.
func (l *Loop) ListenUDP(network string, laddr *net.UDPAddr, recv UDPRecvCallback) (*UDP, error) {
	if recv == nil {
		return nil, errors.New("uvgo: nil udp recv callback")
	}
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return nil, ErrStopped
	}
	l.mu.Unlock()

	pc, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}

	u := &UDP{
		loop:   l,
		conn:   pc,
		recvCB: recv,
	}

	l.mu.Lock()
	l.udps = append(l.udps, u)
	l.mu.Unlock()

	l.wg.Add(1)
	go u.recvLoop()

	return u, nil
}

func (u *UDP) recvLoop() {
	defer u.loop.wg.Done()
	buf := make([]byte, 65535)
	for {
		u.mu.Lock()
		conn := u.conn
		u.mu.Unlock()
		if conn == nil {
			return
		}

		n, addr, err := conn.ReadFrom(buf)
		var udpAddr *net.UDPAddr
		if addr != nil {
			udpAddr, _ = addr.(*net.UDPAddr)
		}
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			in := udpInbound{u: u, err: err}
			select {
			case u.loop.udpIn <- in:
			case <-u.loop.ctx.Done():
				return
			}
			return
		}

		p := make([]byte, n)
		copy(p, buf[:n])
		in := udpInbound{u: u, nread: n, buf: p, addr: udpAddr}
		select {
		case u.loop.udpIn <- in:
		case <-u.loop.ctx.Done():
			return
		}
	}
}

func errnoFromUDPRead(err error) int {
	var e syscall.Errno
	if errors.As(err, &e) {
		return int(e)
	}
	return 1
}

// Send queues a udp_send-like write: completion runs on Loop.Run (like
// uv_udp_send’s send_cb).
func (u *UDP) Send(addr *net.UDPAddr, data []byte, cb func(error)) {
	if cb == nil {
		cb = func(error) {}
	}
	if u == nil || u.loop == nil {
		cb(errors.New("uvgo: nil udp"))
		return
	}

	u.loop.wg.Add(1)
	go func() {
		defer u.loop.wg.Done()

		u.mu.Lock()
		conn := u.conn
		u.mu.Unlock()
		var err error
		if conn == nil {
			err = net.ErrClosed
		} else {
			_, err = conn.WriteTo(data, addr)
		}

		done := udpSendDone{cb: cb, err: err}
		select {
		case u.loop.udpSendDone <- done:
		case <-u.loop.ctx.Done():
		}
	}()
}

// Close stops recv and releases the socket (uv_close path for the handle).
func (u *UDP) Close() error {
	if u == nil {
		return nil
	}
	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		return nil
	}
	u.closed = true
	conn := u.conn
	u.conn = nil
	u.mu.Unlock()

	var err error
	if conn != nil {
		err = conn.Close()
	}

	u.loop.mu.Lock()
	out := u.loop.udps[:0]
	for _, x := range u.loop.udps {
		if x != u {
			out = append(out, x)
		}
	}
	u.loop.udps = out
	u.loop.mu.Unlock()

	return err
}

// LocalAddr returns the bound address, or nil if closed.
func (u *UDP) LocalAddr() net.Addr {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.conn == nil {
		return nil
	}
	return u.conn.LocalAddr()
}

type udpInbound struct {
	u     *UDP
	nread int
	buf   []byte
	addr  *net.UDPAddr
	flags UDPFlags
	err   error
}

type udpSendDone struct {
	cb  func(error)
	err error
}

func (l *Loop) dispatchUDPIn(in udpInbound) {
	if in.u == nil {
		return
	}
	cb := in.u.recvCB
	if cb == nil {
		return
	}
	if in.err != nil {
		e := errnoFromUDPRead(in.err)
		cb(in.u, -e, nil, nil, 0)
		return
	}
	cb(in.u, in.nread, in.buf, in.addr, in.flags)
}
