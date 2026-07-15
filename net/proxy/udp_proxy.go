package proxy

import (
	"io"
	"net"
)

// UDPProxy is a reverse proxy implementation for UDP traffic.
type UDPProxy struct {
	target string
}

// NewUDPProxy creates a new UDPProxy.
func NewUDPProxy(address string) (*UDPProxy, error) {
	return &UDPProxy{target: address}, nil
}

// ServeUDP proxies a UDP session to the configured backend.
func (p *UDPProxy) ServeUDP(conn *UDPConn) {

	defer conn.Close()

	connBackend, err := net.Dial("udp", p.target)
	if err != nil {
		return
	}
	defer connBackend.Close()

	errChan := make(chan error)
	go udpConnCopy(conn, connBackend, errChan)
	go udpConnCopy(connBackend, conn, errChan)

	err = <-errChan
	if err != nil {
	}

	<-errChan
}

func udpConnCopy(dst io.WriteCloser, src io.Reader, errCh chan error) {
	buffer := make([]byte, maxUDPDatagramSize)
	_, err := io.CopyBuffer(dst, src, buffer)
	errCh <- err

	if err := dst.Close(); err != nil {
	}
}
