package proxy

import (
	"io"
	"net"

	"github.com/rs/zerolog/log"
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
	log.Debug().Msgf("Handling UDP stream from %s to %s", conn.rAddr, p.target)

	defer conn.Close()

	connBackend, err := net.Dial("udp", p.target)
	if err != nil {
		log.Error().Err(err).Msg("Error while dialing backend")
		return
	}
	defer connBackend.Close()

	errChan := make(chan error)
	go udpConnCopy(conn, connBackend, errChan)
	go udpConnCopy(connBackend, conn, errChan)

	err = <-errChan
	if err != nil {
		log.Error().Err(err).Msg("Error while handling UDP stream")
	}

	<-errChan
}

func udpConnCopy(dst io.WriteCloser, src io.Reader, errCh chan error) {
	buffer := make([]byte, maxUDPDatagramSize)
	_, err := io.CopyBuffer(dst, src, buffer)
	errCh <- err

	if err := dst.Close(); err != nil {
		log.Debug().Err(err).Msg("Error while terminating UDP stream")
	}
}
