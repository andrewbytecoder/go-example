//go:build windows

package proxy

import (
	"errors"
	"net"
	"syscall"
)

func isTCPReadConnResetError(err error) bool {
	opErr, ok := errors.AsType[*net.OpError](err)
	return ok && opErr.Op == "read" && errors.Is(err, syscall.WSAECONNRESET)
}
