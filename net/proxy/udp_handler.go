package proxy

// UDPHandler is the UDP counterpart of a request handler.
type UDPHandler interface {
	ServeUDP(conn *UDPConn)
}

// UDPHandlerFunc adapts a function into a UDPHandler.
type UDPHandlerFunc func(conn *UDPConn)

// ServeUDP serves UDP traffic.
func (f UDPHandlerFunc) ServeUDP(conn *UDPConn) {
	f(conn)
}
