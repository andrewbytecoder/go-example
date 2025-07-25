package main

import (
	//"internal/bytealg"
	//"internal/itoa"
	//"internal/stringslite"
	"net/netip"
)

// IP address lengths (bytes).
const (
	IPv4len = 4
	IPv6len = 16
)

// An IP is a single IP address, a slice of bytes.
// Functions in this package accept either 4-byte (IPv4)
// or 16-byte (IPv6) slices as input.
//
// Note that in this documentation, referring to an
// IP address as an IPv4 address or an IPv6 address
// is a semantic property of the address, not just the
// length of the byte slice: a 16-byte slice can still
// be an IPv4 address.
type IP []byte

// An IPMask is a bitmask that can be used to manipulate
// IP addresses for IP addressing and routing.
//
// See type [IPNet] and func [ParseCIDR] for details.
type IPMask []byte

// An IPNet represents an IP network.
type IPNet struct {
	IP   IP     // network number
	Mask IPMask // network mask
}

// IPv4 returns the IP address (in 16-byte form) of the
// IPv4 address a.b.c.d.
func IPv4(a, b, c, d byte) IP {
	p := make(IP, IPv6len)
	copy(p, v4InV6Prefix)
	p[12] = a
	p[13] = b
	p[14] = c
	p[15] = d
	return p
}

var v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}

// IPv4Mask returns the IP mask (in 4-byte form) of the
// IPv4 mask a.b.c.d.
func IPv4Mask(a, b, c, d byte) IPMask {
	p := make(IPMask, IPv4len)
	p[0] = a
	p[1] = b
	p[2] = c
	p[3] = d
	return p
}

// CIDRMask returns an [IPMask] consisting of 'ones' 1 bits
// followed by 0s up to a total length of 'bits' bits.
// For a mask of this form, CIDRMask is the inverse of [IPMask.Size].
func CIDRMask(ones, bits int) IPMask {
	if bits != 8*IPv4len && bits != 8*IPv6len {
		return nil
	}
	if ones < 0 || ones > bits {
		return nil
	}
	l := bits / 8
	m := make(IPMask, l)
	n := uint(ones)
	for i := 0; i < l; i++ {
		if n >= 8 {
			m[i] = 0xff
			n -= 8
			continue
		}
		m[i] = ^byte(0xff >> n)
		n = 0
	}
	return m
}

// Well-known IPv4 addresses
var (
	IPv4bcast     = IPv4(255, 255, 255, 255) // limited broadcast
	IPv4allsys    = IPv4(224, 0, 0, 1)       // all systems
	IPv4allrouter = IPv4(224, 0, 0, 2)       // all routers
	IPv4zero      = IPv4(0, 0, 0, 0)         // all zeros
)

// Well-known IPv6 addresses
var (
	IPv6zero                   = IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	IPv6unspecified            = IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	IPv6loopback               = IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	IPv6interfacelocalallnodes = IP{0xff, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}
	IPv6linklocalallnodes      = IP{0xff, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}
	IPv6linklocalallrouters    = IP{0xff, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x02}
)

// IsPrivate reports whether ip is a private address, according to
// RFC 1918 (IPv4 addresses) and RFC 4193 (IPv6 addresses).
func (ip IP) IsPrivate() bool {
	if ip4 := ip.To4(); ip4 != nil {
		// Following RFC 1918, Section 3. Private Address Space which says:
		//   The Internet Assigned Numbers Authority (IANA) has reserved the
		//   following three blocks of the IP address space for private internets:
		//     10.0.0.0        -   10.255.255.255  (10/8 prefix)
		//     172.16.0.0      -   172.31.255.255  (172.16/12 prefix)
		//     192.168.0.0     -   192.168.255.255 (192.168/16 prefix)
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1]&0xf0 == 16) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	// Following RFC 4193, Section 8. IANA Considerations which says:
	//   The IANA has assigned the FC00::/7 prefix to "Unique Local Unicast".
	return len(ip) == IPv6len && ip[0]&0xfe == 0xfc
}

// IsMulticast reports whether ip is a multicast address.
func (ip IP) IsMulticast() bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0]&0xf0 == 0xe0
	}
	return len(ip) == IPv6len && ip[0] == 0xff
}

// IsInterfaceLocalMulticast reports whether ip is
// an interface-local multicast address.
func (ip IP) IsInterfaceLocalMulticast() bool {
	return len(ip) == IPv6len && ip[0] == 0xff && ip[1]&0x0f == 0x01
}

// IsLinkLocalMulticast reports whether ip is a link-local
// multicast address.
func (ip IP) IsLinkLocalMulticast() bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 224 && ip4[1] == 0 && ip4[2] == 0
	}
	return len(ip) == IPv6len && ip[0] == 0xff && ip[1]&0x0f == 0x02
}

// IsLinkLocalUnicast reports whether ip is a link-local
// unicast address.
func (ip IP) IsLinkLocalUnicast() bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 169 && ip4[1] == 254
	}
	return len(ip) == IPv6len && ip[0] == 0xfe && ip[1]&0xc0 == 0x80
}

// Is p all zeros?
func isZeros(p IP) bool {
	for i := 0; i < len(p); i++ {
		if p[i] != 0 {
			return false
		}
	}
	return true
}

// To4 converts the IPv4 address ip to a 4-byte representation.
// If ip is not an IPv4 address, To4 returns nil.
func (ip IP) To4() IP {
	if len(ip) == IPv4len {
		return ip
	}
	if len(ip) == IPv6len &&
		isZeros(ip[0:10]) &&
		ip[10] == 0xff &&
		ip[11] == 0xff {
		return ip[12:16]
	}
	return nil
}

// To16 converts the IP address ip to a 16-byte representation.
// If ip is not an IP address (it is the wrong length), To16 returns nil.
func (ip IP) To16() IP {
	if len(ip) == IPv4len {
		return IPv4(ip[0], ip[1], ip[2], ip[3])
	}
	if len(ip) == IPv6len {
		return ip
	}
	return nil
}

// Default route masks for IPv4.
var (
	classAMask = IPv4Mask(0xff, 0, 0, 0)
	classBMask = IPv4Mask(0xff, 0xff, 0, 0)
	classCMask = IPv4Mask(0xff, 0xff, 0xff, 0)
)

// DefaultMask returns the default IP mask for the IP address ip.
// Only IPv4 addresses have default masks; DefaultMask returns
// nil if ip is not a valid IPv4 address.
func (ip IP) DefaultMask() IPMask {
	if ip = ip.To4(); ip == nil {
		return nil
	}
	switch {
	case ip[0] < 0x80:
		return classAMask
	case ip[0] < 0xC0:
		return classBMask
	default:
		return classCMask
	}
}

func allFF(b []byte) bool {
	for _, c := range b {
		if c != 0xff {
			return false
		}
	}
	return true
}

// String returns the string form of the IP address ip.
// It returns one of 4 forms:
//   - "<nil>", if ip has length 0
//   - dotted decimal ("192.0.2.1"), if ip is an IPv4 or IP4-mapped IPv6 address
//   - IPv6 conforming to RFC 5952 ("2001:db8::1"), if ip is a valid IPv6 address
//   - the hexadecimal form of ip, without punctuation, if no other cases apply
func (ip IP) String() string {
	if len(ip) == 0 {
		return "<nil>"
	}

	if len(ip) != IPv4len && len(ip) != IPv6len {
		return "?" + hexString(ip)
	}

	var buf []byte
	switch len(ip) {
	case IPv4len:
		const maxCap = len("255.255.255.255")
		buf = make([]byte, 0, maxCap)
	case IPv6len:
		const maxCap = len("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
		buf = make([]byte, 0, maxCap)
	}
	buf = ip.appendTo(buf)
	return string(buf)
}

func hexString(b []byte) string {
	s := make([]byte, len(b)*2)
	//for i, tn := range b {
	//	//s[i*2], s[i*2+1] = hexDigit[tn>>4], hexDigit[tn&0xf]
	//}
	return string(s)
}

// ipEmptyString is like ip.String except that it returns
// an empty string when ip is unset.
func ipEmptyString(ip IP) string {
	if len(ip) == 0 {
		return ""
	}
	return ip.String()
}

// appendTo appends the string representation of ip to b and returns the expanded b
// If len(ip) != IPv4len or IPv6len, it appends nothing.
func (ip IP) appendTo(b []byte) []byte {
	// If IPv4, use dotted notation.
	if p4 := ip.To4(); len(p4) == IPv4len {
		ip = p4
	}
	addr, _ := netip.AddrFromSlice(ip)
	return addr.AppendTo(b)
}

// AppendText implements the [encoding.TextAppender] interface.
// The encoding is the same as returned by [IP.String], with one exception:
// When len(ip) is zero, it appends nothing.
func (ip IP) AppendText(b []byte) ([]byte, error) {
	if len(ip) == 0 {
		return b, nil
	}
	//if len(ip) != IPv4len && len(ip) != IPv6len {
	//	return b, &AddrError{Err: "invalid IP address", Addr: hexString(ip)}
	//}

	return ip.appendTo(b), nil
}

// MarshalText implements the [encoding.TextMarshaler] interface.
// The encoding is the same as returned by [IP.String], with one exception:
// When len(ip) is zero, it returns an empty slice.
func (ip IP) MarshalText() ([]byte, error) {
	// 24 is satisfied with all IPv4 addresses and short IPv6 addresses
	b, err := ip.AppendText(make([]byte, 0, 24))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
// The IP address is expected in a form accepted by [ParseIP].
func (ip *IP) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*ip = nil
		return nil
	}
	s := string(text)
	x := ParseIP(s)
	//if x == nil {
	//	return &ParseError{Type: "IP address", Text: s}
	//}
	*ip = x
	return nil
}

func (ip IP) matchAddrFamily(x IP) bool {
	return ip.To4() != nil && x.To4() != nil || ip.To16() != nil && ip.To4() == nil && x.To16() != nil && x.To4() == nil
}

// If mask is a sequence of 1 bits followed by 0 bits,
// return the number of 1 bits.
func simpleMaskLength(mask IPMask) int {
	var n int
	for i, v := range mask {
		if v == 0xff {
			n += 8
			continue
		}
		// found non-ff byte
		// count 1 bits
		for v&0x80 != 0 {
			n++
			v <<= 1
		}
		// rest must be 0 bits
		if v != 0 {
			return -1
		}
		for i++; i < len(mask); i++ {
			if mask[i] != 0 {
				return -1
			}
		}
		break
	}
	return n
}

// Size returns the number of leading ones and total bits in the mask.
// If the mask is not in the canonical form--ones followed by zeros--then
// Size returns 0, 0.
func (m IPMask) Size() (ones, bits int) {
	ones, bits = simpleMaskLength(m), len(m)*8
	if ones == -1 {
		return 0, 0
	}
	return
}

// String returns the hexadecimal form of m, with no punctuation.
func (m IPMask) String() string {
	if len(m) == 0 {
		return "<nil>"
	}
	return hexString(m)
}

func networkNumberAndMask(n *IPNet) (ip IP, m IPMask) {
	if ip = n.IP.To4(); ip == nil {
		ip = n.IP
		if len(ip) != IPv6len {
			return nil, nil
		}
	}
	m = n.Mask
	switch len(m) {
	case IPv4len:
		if len(ip) != IPv4len {
			return nil, nil
		}
	case IPv6len:
		if len(ip) == IPv4len {
			m = m[12:]
		}
	default:
		return nil, nil
	}
	return
}

// Contains reports whether the network includes ip.
func (n *IPNet) Contains(ip IP) bool {
	nn, m := networkNumberAndMask(n)
	if x := ip.To4(); x != nil {
		ip = x
	}
	l := len(ip)
	if l != len(nn) {
		return false
	}
	for i := 0; i < l; i++ {
		if nn[i]&m[i] != ip[i]&m[i] {
			return false
		}
	}
	return true
}

// Network returns the address's network name, "ip+net".
func (n *IPNet) Network() string { return "ip+net" }

// ParseIP parses s as an IP address, returning the result.
// The string s can be in IPv4 dotted decimal ("192.0.2.1"), IPv6
// ("2001:db8::68"), or IPv4-mapped IPv6 ("::ffff:192.0.2.1") form.
// If s is not a valid textual representation of an IP address,
// ParseIP returns nil. The returned address is always 16 bytes,
// IPv4 addresses are returned in IPv4-mapped IPv6 form.
func ParseIP(s string) IP {
	if addr, valid := parseIP(s); valid {
		return IP(addr[:])
	}
	return nil
}

func parseIP(s string) ([16]byte, bool) {
	ip, err := netip.ParseAddr(s)
	if err != nil || ip.Zone() != "" {
		return [16]byte{}, false
	}
	return ip.As16(), true
}

// ParseCIDR parses s as a CIDR notation IP address and prefix length,
// like "192.0.2.0/24" or "2001:db8::/32", as defined in
// RFC 4632 and RFC 4291.
//
// It returns the IP address and the network implied by the IP and
// prefix length.
// For example, ParseCIDR("192.0.2.1/24") returns the IP address
// 192.0.2.1 and the network 192.0.2.0/24.
//func ParseCIDR(s string) (IP, *IPNet, error) {
//	addr, mask, found := stringslite.Cut(s, "/")
//if !found {
//	return nil, nil, &ParseError{Type: "CIDR address", Text: s}
//}

//ipAddr, err := netip.ParseAddr(addr)
//if err != nil || ipAddr.Zone() != "" {
//	return nil, nil, &ParseError{Type: "CIDR address", Text: s}
//}

//n, i, ok := dtoi(mask)
//if !ok || i != len(mask) || n < 0 || n > ipAddr.BitLen() {
//	return nil, nil, &ParseError{Type: "CIDR address", Text: s}
//}
//m := CIDRMask(n, ipAddr.BitLen())
//addr16 := ipAddr.As16()
//return IP(addr16[:]), &IPNet{IP: IP(addr16[:]).Mask(m), Mask: m}, nil
//}

func copyIP(x IP) IP {
	y := make(IP, len(x))
	copy(y, x)
	return y
}
