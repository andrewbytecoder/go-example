package uvgo

import (
	"net"
)

// ListenTCP binds like uv_tcp_bind + uv_listen. Accepted connections are
// delivered to onConn on the Loop.Run goroutine (serialized like libuv’s
// default thread).
//
// Only one onConn handler is kept per Loop; the last non-nil ListenTCP wins.
func (l *Loop) ListenTCP(network, address string, onConn func(net.Conn)) error {
	if onConn == nil {
		onConn = func(c net.Conn) { _ = c.Close() }
	}

	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return ErrStopped
	}
	l.onTCPConn = onConn
	l.mu.Unlock()

	ln, err := net.Listen(network, address)
	if err != nil {
		return err
	}

	l.mu.Lock()
	l.listeners = append(l.listeners, ln)
	l.mu.Unlock()

	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			select {
			case l.acceptQ <- c:
			case <-l.ctx.Done():
				_ = c.Close()
				return
			}
		}
	}()
	return nil
}
