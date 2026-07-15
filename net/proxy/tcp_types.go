package proxy

import "net"

// TCPHandler is the TCP counterpart of a request handler.
type TCPHandler interface {
	ServeTCP(conn TCPWriteCloser)
}

// TCPHandlerFunc adapts a function into a TCPHandler.
type TCPHandlerFunc func(conn TCPWriteCloser)

// ServeTCP serves TCP traffic.
func (f TCPHandlerFunc) ServeTCP(conn TCPWriteCloser) {
	f(conn)
}

// TCPWriteCloser describes a net.Conn with a CloseWrite method.
type TCPWriteCloser interface {
	net.Conn
	CloseWrite() error
}

// TCPClientConn exposes the client-side address information used by the dialer.
type TCPClientConn interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}
