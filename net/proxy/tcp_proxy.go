package proxy

import (
	"errors"
	"io"
	"net"
	"syscall"
	"time"
)

// TCPProxy forwards a TCP connection to a TCP service.
type TCPProxy struct {
	address string
	dialer  TCPDialer
}

// NewTCPProxy creates a new TCPProxy.
func NewTCPProxy(address string, dialer TCPDialer) (*TCPProxy, error) {
	return &TCPProxy{
		address: address,
		dialer:  dialer,
	}, nil
}

// ServeTCP forwards the connection to a backend service.
func (p *TCPProxy) ServeTCP(conn TCPWriteCloser) {

	defer conn.Close()

	connBackend, err := p.dialBackend(conn)
	if err != nil {
		return
	}
	defer connBackend.Close()

	errChan := make(chan error)
	go p.connCopy(conn, connBackend, errChan)
	go p.connCopy(connBackend, conn, errChan)

	err = <-errChan
	if err != nil {
		if isTCPReadConnResetError(err) {
		} else {
		}
	}

	<-errChan
}

func (p *TCPProxy) dialBackend(clientConn net.Conn) (TCPWriteCloser, error) {
	conn, err := p.dialer.Dial("tcp", p.address, clientConn)
	if err != nil {
		return nil, err
	}

	return conn.(TCPWriteCloser), nil
}

func (p *TCPProxy) connCopy(dst, src TCPWriteCloser, errCh chan error) {
	_, err := io.Copy(dst, src)
	errCh <- err

	errClose := dst.CloseWrite()
	if errClose != nil {
		if !isTCPSocketNotConnectedError(errClose) {
		}
		return
	}

	if p.dialer.TerminationDelay() >= 0 {
		if err := dst.SetReadDeadline(time.Now().Add(p.dialer.TerminationDelay())); err != nil {
		}
	}
}

func isTCPSocketNotConnectedError(err error) bool {
	_, ok := errors.AsType[*net.OpError](err)
	return ok && errors.Is(err, syscall.ENOTCONN)
}
