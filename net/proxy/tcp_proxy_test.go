package proxy

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTCPProxyCloseWrite(t *testing.T) {
	backendListener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = backendListener.Close() })

	go fakeTCPServer(t, backendListener)

	_, port, err := net.SplitHostPort(backendListener.Addr().String())
	require.NoError(t, err)

	dialer := tcpDialer{dialer: &net.Dialer{}, terminationDelay: 10 * time.Millisecond}
	proxy, err := NewTCPProxy(":"+port, dialer)
	require.NoError(t, err)

	proxyListener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = proxyListener.Close() })

	go func() {
		for {
			conn, acceptErr := proxyListener.Accept()
			if acceptErr != nil {
				return
			}

			go proxy.ServeTCP(conn.(*net.TCPConn))
		}
	}()

	_, port, err = net.SplitHostPort(proxyListener.Addr().String())
	require.NoError(t, err)

	conn, err := net.Dial("tcp", ":"+port)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, err = conn.Write([]byte("ping\n"))
	require.NoError(t, err)

	err = conn.(*net.TCPConn).CloseWrite()
	require.NoError(t, err)

	buffer := bytes.NewBuffer(nil)
	n, err := io.Copy(buffer, conn)
	require.NoError(t, err)

	require.Equal(t, int64(4), n)
	require.Equal(t, "PONG", buffer.String())
}

func fakeTCPServer(t *testing.T, listener net.Listener) {
	t.Helper()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		go func(conn net.Conn) {
			defer conn.Close()

			for {
				withErr := false
				buf := make([]byte, 64)
				if _, err := conn.Read(buf); err != nil {
					withErr = true
				}

				if string(buf[:4]) == "ping" {
					time.Sleep(time.Millisecond)
					if _, err := conn.Write([]byte("PONG")); err != nil {
						return
					}
				}

				if withErr {
					return
				}
			}
		}(conn)
	}
}
