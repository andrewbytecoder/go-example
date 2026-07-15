package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	maxUDPDatagramSize = 65535
	udpCloseRetryDelay = 500 * time.Millisecond
)

var errClosedUDPListener = errors.New("udp: listener closed")

// UDPListener augments a session-oriented listener over a UDP PacketConn.
type UDPListener struct {
	pConn *net.UDPConn

	mu    sync.RWMutex
	conns map[string]*UDPConn

	accepting bool
	acceptCh  chan *UDPConn
	timeout   time.Duration
}

// ListenUDPPacketConn creates a new listener from a PacketConn.
func ListenUDPPacketConn(packetConn net.PacketConn, timeout time.Duration) (*UDPListener, error) {
	if timeout <= 0 {
		return nil, errors.New("timeout should be greater than zero")
	}

	pConn, ok := packetConn.(*net.UDPConn)
	if !ok {
		return nil, errors.New("packet conn is not a UDPConn")
	}

	listener := &UDPListener{
		pConn:     pConn,
		conns:     make(map[string]*UDPConn),
		accepting: true,
		acceptCh:  make(chan *UDPConn),
		timeout:   timeout,
	}

	go listener.readLoop()
	return listener, nil
}

// ListenUDP creates a new UDP session listener.
func ListenUDP(listenConfig net.ListenConfig, network, address string, timeout time.Duration) (*UDPListener, error) {
	if timeout <= 0 {
		return nil, errors.New("timeout should be greater than zero")
	}

	packetConn, err := listenConfig.ListenPacket(context.Background(), network, address)
	if err != nil {
		return nil, fmt.Errorf("listen packet: %w", err)
	}

	listener, err := ListenUDPPacketConn(packetConn, timeout)
	if err != nil {
		return nil, fmt.Errorf("listen packet conn: %w", err)
	}

	return listener, nil
}

// Accept waits for and returns the next UDP session.
func (l *UDPListener) Accept() (*UDPConn, error) {
	conn := <-l.acceptCh
	if conn == nil {
		return nil, errClosedUDPListener
	}

	return conn, nil
}

// Addr returns the listener network address.
func (l *UDPListener) Addr() net.Addr {
	return l.pConn.LocalAddr()
}

// Close closes the listener immediately.
func (l *UDPListener) Close() error {
	return l.Shutdown(0)
}

// Shutdown closes the listener after waiting for existing sessions up to graceTimeout.
func (l *UDPListener) Shutdown(graceTimeout time.Duration) error {
	l.mu.Lock()
	if !l.accepting {
		l.mu.Unlock()
		return nil
	}
	l.accepting = false
	l.mu.Unlock()

	retryInterval := min(udpCloseRetryDelay, graceTimeout)
	end := time.Now().Add(graceTimeout)
	for !time.Now().After(end) {
		l.mu.RLock()
		if len(l.conns) == 0 {
			l.mu.RUnlock()
			break
		}
		l.mu.RUnlock()

		time.Sleep(retryInterval)
	}

	return l.close()
}

func (l *UDPListener) close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	err := l.pConn.Close()
	for key, conn := range l.conns {
		conn.close()
		delete(l.conns, key)
	}
	close(l.acceptCh)
	return err
}

func (l *UDPListener) readLoop() {
	for {
		buf := make([]byte, maxUDPDatagramSize)

		n, raddr, err := l.pConn.ReadFrom(buf)
		if err != nil {
			return
		}

		conn, err := l.getConn(raddr)
		if err != nil {
			continue
		}

		select {
		case conn.receiveCh <- buf[:n]:
		case <-conn.doneCh:
		}
	}
}

func (l *UDPListener) getConn(raddr net.Addr) (*UDPConn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	conn, ok := l.conns[raddr.String()]
	if ok {
		return conn, nil
	}

	if !l.accepting {
		return nil, errClosedUDPListener
	}

	conn = l.newConn(raddr)
	l.conns[raddr.String()] = conn
	l.acceptCh <- conn
	go conn.readLoop()

	return conn, nil
}

func (l *UDPListener) newConn(rAddr net.Addr) *UDPConn {
	return &UDPConn{
		listener:     l,
		rAddr:        rAddr,
		receiveCh:    make(chan []byte),
		readCh:       make(chan []byte),
		sizeCh:       make(chan int),
		doneCh:       make(chan struct{}),
		timeout:      l.timeout,
		lastActivity: time.Now(),
	}
}

// UDPConn represents an ongoing session with a client over UDP packets.
type UDPConn struct {
	listener *UDPListener
	rAddr    net.Addr

	receiveCh chan []byte
	readCh    chan []byte
	sizeCh    chan int
	msgs      [][]byte

	muActivity   sync.RWMutex
	lastActivity time.Time

	timeout  time.Duration
	doneOnce sync.Once
	doneCh   chan struct{}
}

// Read reads at most one UDP datagram into p.
func (c *UDPConn) Read(p []byte) (int, error) {
	select {
	case c.readCh <- p:
		n := <-c.sizeCh
		c.muActivity.Lock()
		c.lastActivity = time.Now()
		c.muActivity.Unlock()
		return n, nil
	case <-c.doneCh:
		return 0, io.EOF
	}
}

// Write writes one UDP datagram to the client.
func (c *UDPConn) Write(p []byte) (int, error) {
	c.muActivity.Lock()
	c.lastActivity = time.Now()
	c.muActivity.Unlock()

	return c.listener.pConn.WriteTo(p, c.rAddr)
}

// Close releases resources related to the UDP session.
func (c *UDPConn) Close() error {
	c.close()

	c.listener.mu.Lock()
	defer c.listener.mu.Unlock()
	delete(c.listener.conns, c.rAddr.String())
	return nil
}

func (c *UDPConn) readLoop() {
	ticker := time.NewTicker(c.timeout / 10)
	defer ticker.Stop()

	for {
		if len(c.msgs) == 0 {
			select {
			case msg := <-c.receiveCh:
				c.msgs = append(c.msgs, msg)
			case <-ticker.C:
				c.muActivity.RLock()
				deadline := c.lastActivity.Add(c.timeout)
				c.muActivity.RUnlock()
				if time.Now().After(deadline) {
					c.Close()
					return
				}
				continue
			}
		}

		select {
		case cBuf := <-c.readCh:
			msg := c.msgs[0]
			c.msgs = c.msgs[1:]
			n := copy(cBuf, msg)
			c.sizeCh <- n
		case msg := <-c.receiveCh:
			c.msgs = append(c.msgs, msg)
		case <-ticker.C:
			c.muActivity.RLock()
			deadline := c.lastActivity.Add(c.timeout)
			c.muActivity.RUnlock()
			if time.Now().After(deadline) {
				c.Close()
				return
			}
		}
	}
}

func (c *UDPConn) close() {
	c.doneOnce.Do(func() {
		close(c.doneCh)
	})
}
