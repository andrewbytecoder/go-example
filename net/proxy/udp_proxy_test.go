package proxy

import (
	"crypto/rand"
	"errors"
	"io"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUDPProxyServeUDP(t *testing.T) {
	backendAddr := ":18081"
	go newUDPServer(t, backendAddr, UDPHandlerFunc(func(conn *UDPConn) {
		for {
			buffer := make([]byte, 1024*1024)
			n, err := conn.Read(buffer)
			require.NoError(t, err)

			_, err = conn.Write(buffer[:n])
			require.NoError(t, err)
		}
	}))

	proxy, err := NewUDPProxy(backendAddr)
	require.NoError(t, err)

	proxyAddr := ":18080"
	go newUDPServer(t, proxyAddr, proxy)

	time.Sleep(500 * time.Millisecond)

	conn, err := net.Dial("udp", proxyAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, err = conn.Write([]byte("DATAWRITE"))
	require.NoError(t, err)

	buffer := make([]byte, 1024*1024)
	n, err := conn.Read(buffer)
	require.NoError(t, err)

	assert.Equal(t, "DATAWRITE", string(buffer[:n]))
}

func TestUDPProxyServeUDPMaxDataSize(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("skip on darwin due to lower UDP max datagram size")
	}

	dataSize := 65507
	backendAddr := ":18083"
	go newUDPServer(t, backendAddr, UDPHandlerFunc(func(conn *UDPConn) {
		buffer := make([]byte, dataSize)
		n, err := conn.Read(buffer)
		require.NoError(t, err)

		_, err = conn.Write(buffer[:n])
		require.NoError(t, err)
	}))

	proxy, err := NewUDPProxy(backendAddr)
	require.NoError(t, err)

	proxyAddr := ":18082"
	go newUDPServer(t, proxyAddr, proxy)

	time.Sleep(500 * time.Millisecond)

	conn, err := net.Dial("udp", proxyAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	want := make([]byte, dataSize)
	_, err = rand.Read(want)
	require.NoError(t, err)

	_, err = conn.Write(want)
	require.NoError(t, err)

	got := make([]byte, dataSize)
	_, err = conn.Read(got)
	require.NoError(t, err)

	assert.Equal(t, want, got)
}

func TestUDPListenerConsecutiveWrites(t *testing.T) {
	listener, err := ListenUDP(net.ListenConfig{}, "udp", ":0", 3*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if errors.Is(acceptErr, errClosedUDPListener) {
				return
			}
			require.NoError(t, acceptErr)

			go func(conn *UDPConn) {
				buffer1 := make([]byte, 2048)
				buffer2 := make([]byte, 2048)

				n1, err := conn.Read(buffer1)
				require.NoError(t, err)
				time.Sleep(10 * time.Millisecond)
				n2, err := conn.Read(buffer2)
				require.NoError(t, err)

				_, err = conn.Write(buffer1[:n1])
				require.NoError(t, err)
				_, err = conn.Write(buffer2[:n2])
				require.NoError(t, err)
			}(conn)
		}
	}()

	conn, err := net.Dial("udp", listener.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	buffer := make([]byte, 2048)
	_, err = conn.Write([]byte("TESTLONG0"))
	require.NoError(t, err)
	_, err = conn.Write([]byte("1TEST"))
	require.NoError(t, err)

	n, err := conn.Read(buffer)
	require.NoError(t, err)
	require.Equal(t, "TESTLONG0", string(buffer[:n]))

	n, err = conn.Read(buffer)
	require.NoError(t, err)
	require.Equal(t, "1TEST", string(buffer[:n]))
}

func newUDPServer(t *testing.T, addr string, handler UDPHandler) {
	t.Helper()

	listener, err := ListenUDP(net.ListenConfig{}, "udp", addr, 3*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	for {
		conn, err := listener.Accept()
		if errors.Is(err, errClosedUDPListener) {
			return
		}
		require.NoError(t, err)

		go handler.ServeUDP(conn)
	}
}

func requireUDPEcho(t *testing.T, data string, conn io.ReadWriter, timeout time.Duration) {
	t.Helper()

	_, err := conn.Write([]byte(data))
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)

		buffer := make([]byte, 1024*1024)
		n, err := conn.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, data, string(buffer[:n]))
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("timeout during echo for %s", data)
	}
}
