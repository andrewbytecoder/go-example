//go:build linux || openbsd || darwin

package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-example/net/ip"
	"github.com/pires/go-proxyproto"
	"golang.org/x/sys/unix"
)

// ProxyProtocol contains Proxy-Protocol configuration.
type ProxyProtocol struct {
	Insecure   bool     `description:"Trust all." json:"insecure,omitempty" toml:"insecure,omitempty" yaml:"insecure,omitempty" export:"true"`
	TrustedIPs []string `description:"Trust only selected IPs." json:"trustedIPs,omitempty" toml:"trustedIPs,omitempty" yaml:"trustedIPs,omitempty"`
}
type EntryPoint struct {
	Address          string                `description:"Entry point address." json:"address,omitempty" toml:"address,omitempty" yaml:"address,omitempty"`
	AllowACMEByPass  bool                  `description:"Enables handling of ACME TLS and HTTP challenges with custom routers." json:"allowACMEByPass,omitempty" toml:"allowACMEByPass,omitempty" yaml:"allowACMEByPass,omitempty"`
	ReusePort        bool                  `description:"Enables EntryPoints from the same or different processes listening on the same TCP/UDP port." json:"reusePort,omitempty" toml:"reusePort,omitempty" yaml:"reusePort,omitempty"`
	AsDefault        bool                  `description:"Adds this EntryPoint to the list of default EntryPoints to be used on routers that don't have any Entrypoint defined." json:"asDefault,omitempty" toml:"asDefault,omitempty" yaml:"asDefault,omitempty"`
	Transport        *EntryPointsTransport `description:"Configures communication between clients and Traefik." json:"transport,omitempty" toml:"transport,omitempty" yaml:"transport,omitempty" export:"true"`
	ProxyProtocol    *ProxyProtocol        `description:"Proxy-Protocol configuration." json:"proxyProtocol,omitempty" toml:"proxyProtocol,omitempty" yaml:"proxyProtocol,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	ForwardedHeaders *ForwardedHeaders     `description:"Trust client forwarding headers." json:"forwardedHeaders,omitempty" toml:"forwardedHeaders,omitempty" yaml:"forwardedHeaders,omitempty" export:"true"`
	HTTP             HTTPConfig            `description:"HTTP configuration." json:"http,omitempty" toml:"http,omitempty" yaml:"http,omitempty" export:"true"`
	HTTP2            *HTTP2Config          `description:"HTTP/2 configuration." json:"http2,omitempty" toml:"http2,omitempty" yaml:"http2,omitempty" export:"true"`
	HTTP3            *HTTP3Config          `description:"HTTP/3 configuration." json:"http3,omitempty" toml:"http3,omitempty" yaml:"http3,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	UDP              *UDPConfig            `description:"UDP configuration." json:"udp,omitempty" toml:"udp,omitempty" yaml:"udp,omitempty"`
}

// GetAddress strips any potential protocol part of the address field of the
// entry point, in order to return the actual address.
func (ep *EntryPoint) GetAddress() string {
	splitN := strings.SplitN(ep.Address, "/", 2)
	return splitN[0]
}

// GetProtocol returns the protocol part of the address field of the entry point.
// If none is specified, it defaults to "tcp".
func (ep *EntryPoint) GetProtocol() (string, error) {
	splitN := strings.SplitN(ep.Address, "/", 2)
	if len(splitN) < 2 {
		return "tcp", nil
	}

	protocol := strings.ToLower(splitN[1])
	if protocol == "tcp" || protocol == "udp" {
		return protocol, nil
	}

	return "", fmt.Errorf("invalid protocol: %s", splitN[1])
}

// UDPConfig is the UDP configuration of an entry point.
type UDPConfig struct {
	Timeout Duration `description:"Timeout defines how long to wait on an idle session before releasing the related resources." json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// HTTP3Config is the HTTP3 configuration of an entry point.
type HTTP3Config struct {
	AdvertisedPort int `description:"UDP port to advertise, on which HTTP/3 is available." json:"advertisedPort,omitempty" toml:"advertisedPort,omitempty" yaml:"advertisedPort,omitempty" export:"true"`
}

// HTTP2Config is the HTTP2 configuration of an entry point.
type HTTP2Config struct {
	MaxConcurrentStreams      int32 `description:"Specifies the number of concurrent streams per connection that each client is allowed to initiate." json:"maxConcurrentStreams,omitempty" toml:"maxConcurrentStreams,omitempty" yaml:"maxConcurrentStreams,omitempty" export:"true"`
	MaxDecoderHeaderTableSize int32 `description:"Specifies the maximum size of the HTTP2 HPACK header table on the decoding (receiving from client) side." json:"maxDecoderHeaderTableSize,omitempty" toml:"maxDecoderHeaderTableSize,omitempty" yaml:"maxDecoderHeaderTableSize,omitempty" export:"true"`
	MaxEncoderHeaderTableSize int32 `description:"Specifies the maximum size of the HTTP2 HPACK header table on the encoding (sending to client) side." json:"maxEncoderHeaderTableSize,omitempty" toml:"maxEncoderHeaderTableSize,omitempty" yaml:"maxEncoderHeaderTableSize,omitempty" export:"true"`
}

// EntryPointsTransport configures communication between clients and Traefik.
type EntryPointsTransport struct {
	LifeCycle            *LifeCycle          `description:"Timeouts influencing the server life cycle." json:"lifeCycle,omitempty" toml:"lifeCycle,omitempty" yaml:"lifeCycle,omitempty" export:"true"`
	RespondingTimeouts   *RespondingTimeouts `description:"Timeouts for incoming requests to the Traefik instance." json:"respondingTimeouts,omitempty" toml:"respondingTimeouts,omitempty" yaml:"respondingTimeouts,omitempty" export:"true"`
	KeepAliveMaxTime     Duration            `description:"Maximum duration before closing a keep-alive connection." json:"keepAliveMaxTime,omitempty" toml:"keepAliveMaxTime,omitempty" yaml:"keepAliveMaxTime,omitempty" export:"true"`
	KeepAliveMaxRequests int                 `description:"Maximum number of requests before closing a keep-alive connection." json:"keepAliveMaxRequests,omitempty" toml:"keepAliveMaxRequests,omitempty" yaml:"keepAliveMaxRequests,omitempty" export:"true"`
}

// LifeCycle contains configurations relevant to the lifecycle (such as the shutdown phase) of Traefik.
type LifeCycle struct {
	RequestAcceptGraceTimeout Duration `description:"Duration to keep accepting requests before Traefik initiates the graceful shutdown procedure." json:"requestAcceptGraceTimeout,omitempty" toml:"requestAcceptGraceTimeout,omitempty" yaml:"requestAcceptGraceTimeout,omitempty" export:"true"`
	GraceTimeOut              Duration `description:"Duration to give active requests a chance to finish before Traefik stops." json:"graceTimeOut,omitempty" toml:"graceTimeOut,omitempty" yaml:"graceTimeOut,omitempty" export:"true"`
}

// RespondingTimeouts contains timeout configurations for incoming requests to the Traefik instance.
type RespondingTimeouts struct {
	ReadTimeout  Duration `description:"ReadTimeout is the maximum duration for reading the entire request, including the body. If zero, no timeout is set." json:"readTimeout,omitempty" toml:"readTimeout,omitempty" yaml:"readTimeout,omitempty" export:"true"`
	WriteTimeout Duration `description:"WriteTimeout is the maximum duration before timing out writes of the response. If zero, no timeout is set." json:"writeTimeout,omitempty" toml:"writeTimeout,omitempty" yaml:"writeTimeout,omitempty" export:"true"`
	IdleTimeout  Duration `description:"IdleTimeout is the maximum amount duration an idle (keep-alive) connection will remain idle before closing itself. If zero, no timeout is set." json:"idleTimeout,omitempty" toml:"idleTimeout,omitempty" yaml:"idleTimeout,omitempty" export:"true"`
}

// ForwardedHeaders Trust client forwarding headers.
type ForwardedHeaders struct {
	Insecure                   bool     `description:"Trust all forwarded headers." json:"insecure,omitempty" toml:"insecure,omitempty" yaml:"insecure,omitempty" export:"true"`
	TrustedIPs                 []string `description:"Trust only forwarded headers from selected IPs." json:"trustedIPs,omitempty" toml:"trustedIPs,omitempty" yaml:"trustedIPs,omitempty"`
	Connection                 []string `description:"List of Connection headers that are allowed to pass through the middleware chain before being removed." json:"connection,omitempty" toml:"connection,omitempty" yaml:"connection,omitempty"`
	NotAppendXForwardedFor     bool     `description:"Disable appending RemoteAddr to X-Forwarded-For header. Defaults to false (appending is enabled)." json:"notAppendXForwardedFor,omitempty" toml:"notAppendXForwardedFor,omitempty" yaml:"notAppendXForwardedFor,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	AddXForwardedSchemeHeaders bool     `description:"Add the X-Forwarded-Scheme and X-Scheme headers." json:"addXForwardedSchemeHeaders,omitempty" toml:"addXForwardedSchemeHeaders,omitempty" yaml:"addXForwardedSchemeHeaders,omitempty" export:"true"`
}

// RedirectEntryPoint is the definition of an entry point redirection.
type RedirectEntryPoint struct {
	To        string `description:"Targeted entry point of the redirection." json:"to,omitempty" toml:"to,omitempty" yaml:"to,omitempty" export:"true"`
	Scheme    string `description:"Scheme used for the redirection." json:"scheme,omitempty" toml:"scheme,omitempty" yaml:"scheme,omitempty" export:"true"`
	Permanent bool   `description:"Applies a permanent redirection." json:"permanent,omitempty" toml:"permanent,omitempty" yaml:"permanent,omitempty" export:"true"`
	Priority  int    `description:"Priority of the generated router." json:"priority,omitempty" toml:"priority,omitempty" yaml:"priority,omitempty" export:"true"`
}

// Redirections is a set of redirection for an entry point.
type Redirections struct {
	EntryPoint *RedirectEntryPoint `description:"Set of redirection for an entry point." json:"entryPoint,omitempty" toml:"entryPoint,omitempty" yaml:"entryPoint,omitempty" export:"true"`
}

// HTTPConfig is the HTTP configuration of an entry point.
type HTTPConfig struct {
	Redirections              *Redirections      `description:"Set of redirection" json:"redirections,omitempty" toml:"redirections,omitempty" yaml:"redirections,omitempty" export:"true"`
	Middlewares               []string           `description:"Default middlewares for the routers linked to the entry point." json:"middlewares,omitempty" toml:"middlewares,omitempty" yaml:"middlewares,omitempty" export:"true"`
	TLS                       *TLSConfig         `description:"Default TLS configuration for the routers linked to the entry point." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	EncodedCharacters         *EncodedCharacters `description:"Defines which encoded characters are allowed in the request path." json:"encodedCharacters,omitempty" toml:"encodedCharacters,omitempty" yaml:"encodedCharacters,omitempty" export:"true"`
	EncodeQuerySemicolons     bool               `description:"Defines whether request query semicolons should be URLEncoded." json:"encodeQuerySemicolons,omitempty" toml:"encodeQuerySemicolons,omitempty" yaml:"encodeQuerySemicolons,omitempty" export:"true"`
	SanitizePath              *bool              `description:"Defines whether to enable request path sanitization (removal of /./, /../ and multiple slash sequences)." json:"sanitizePath,omitempty" toml:"sanitizePath,omitempty" yaml:"sanitizePath,omitempty" export:"true"`
	MaxHeaderBytes            int                `description:"Maximum size of request headers in bytes." json:"maxHeaderBytes,omitempty" toml:"maxHeaderBytes,omitempty" yaml:"maxHeaderBytes,omitempty" export:"true"`
	UnderscoreHeadersStrategy string             `description:"Defines the strategy to handle requests with headers with underscores (keep, delete, and reject)." json:"underscoreHeadersStrategy,omitempty" toml:"underscoreHeadersStrategy,omitempty" yaml:"underscoreHeadersStrategy,omitempty" export:"true"`
}

// EncodedCharacters configures which encoded characters are allowed in the request path.
type EncodedCharacters struct {
	AllowEncodedSlash         bool `description:"Defines whether requests with encoded slash characters in the path are allowed." json:"allowEncodedSlash,omitempty" toml:"allowEncodedSlash,omitempty" yaml:"allowEncodedSlash,omitempty" export:"true"`
	AllowEncodedBackSlash     bool `description:"Defines whether requests with encoded back slash characters in the path are allowed." json:"allowEncodedBackSlash,omitempty" toml:"allowEncodedBackSlash,omitempty" yaml:"allowEncodedBackSlash,omitempty" export:"true"`
	AllowEncodedNullCharacter bool `description:"Defines whether requests with encoded null characters in the path are allowed." json:"allowEncodedNullCharacter,omitempty" toml:"allowEncodedNullCharacter,omitempty" yaml:"allowEncodedNullCharacter,omitempty" export:"true"`
	AllowEncodedSemicolon     bool `description:"Defines whether requests with encoded semicolon characters in the path are allowed." json:"allowEncodedSemicolon,omitempty" toml:"allowEncodedSemicolon,omitempty" yaml:"allowEncodedSemicolon,omitempty" export:"true"`
	AllowEncodedPercent       bool `description:"Defines whether requests with encoded percent characters in the path are allowed." json:"allowEncodedPercent,omitempty" toml:"allowEncodedPercent,omitempty" yaml:"allowEncodedPercent,omitempty" export:"true"`
	AllowEncodedQuestionMark  bool `description:"Defines whether requests with encoded question mark characters in the path are allowed." json:"allowEncodedQuestionMark,omitempty" toml:"allowEncodedQuestionMark,omitempty" yaml:"allowEncodedQuestionMark,omitempty" export:"true"`
	AllowEncodedHash          bool `description:"Defines whether requests with encoded hash characters in the path are allowed." json:"allowEncodedHash,omitempty" toml:"allowEncodedHash,omitempty" yaml:"allowEncodedHash,omitempty" export:"true"`
}

// TLSConfig is the default TLS configuration for all the routers associated to the concerned entry point.
type TLSConfig struct {
	Options      string   `description:"Default TLS options for the routers linked to the entry point." json:"options,omitempty" toml:"options,omitempty" yaml:"options,omitempty" export:"true"`
	CertResolver string   `description:"Default certificate resolver for the routers linked to the entry point." json:"certResolver,omitempty" toml:"certResolver,omitempty" yaml:"certResolver,omitempty" export:"true"`
	Domains      []Domain `description:"Default TLS domains for the routers linked to the entry point." json:"domains,omitempty" toml:"domains,omitempty" yaml:"domains,omitempty" export:"true"`
}

// newListenConfig creates a new net.ListenConfig for the given configuration of
// the entry point.
func newListenConfig(configuration *EntryPoint) (lc net.ListenConfig) {
	if configuration != nil && configuration.ReusePort {
		lc.Control = controlReusePort
	}
	return
}

// controlReusePort is a net.ListenConfig.Control function that enables SO_REUSEPORT
// on the socket.
func controlReusePort(network, address string, c syscall.RawConn) error {
	var setSockOptErr error
	// 设置端口重用
	err := c.Control(func(fd uintptr) {
		// Note that net.ListenConfig enables unix.SO_REUSEADDR by default,
		// as seen in https://go.dev/src/net/sockopt_linux.go. Therefore, no
		// additional action is required to enable it here.

		setSockOptErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if setSockOptErr != nil {
			return
		}
	})
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	if setSockOptErr != nil {
		return fmt.Errorf("setsockopt: %w", setSockOptErr)
	}
	return nil
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}

	if err := tc.SetKeepAlive(true); err != nil {
		return nil, err
	}

	if err := tc.SetKeepAlivePeriod(3 * time.Minute); err != nil {
		// Some systems, such as OpenBSD, have no user-settable per-socket TCP keepalive options.
		if !errors.Is(err, syscall.ENOPROTOOPT) {
			return nil, err
		}
	}

	return tc, nil
}

type onceCloseListener struct {
	net.Listener

	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(func() { oc.closeErr = oc.Listener.Close() })
	return oc.closeErr
}

func buildProxyProtocolListener(ctx context.Context, entryPoint *EntryPoint, listener net.Listener) (net.Listener, error) {
	timeout := entryPoint.Transport.RespondingTimeouts.ReadTimeout
	// proxyproto use 200ms if ReadHeaderTimeout is set to 0 and not no timeout
	if timeout == 0 {
		timeout = -1
	}
	proxyListener := &proxyproto.Listener{Listener: listener, ReadHeaderTimeout: time.Duration(timeout)}

	if entryPoint.ProxyProtocol.Insecure {
		return proxyListener, nil
	}

	checker, err := ip.NewChecker(entryPoint.ProxyProtocol.TrustedIPs)
	if err != nil {
		return nil, err
	}

	proxyListener.Policy = func(upstream net.Addr) (proxyproto.Policy, error) {
		ipAddr, ok := upstream.(*net.TCPAddr)
		if !ok {
			return proxyproto.REJECT, fmt.Errorf("type error %v", upstream)
		}

		if !checker.ContainsIP(ipAddr.IP) {
			return proxyproto.IGNORE, nil
		}
		return proxyproto.USE, nil
	}

	return proxyListener, nil
}

func buildListener(ctx context.Context, name string, config *EntryPoint) (net.Listener, error) {

	// 创建一个新的 net.ListenConfig 实例
	lc := newListenConfig(config)

	if !strings.Contains(os.Getenv("GODEBUG"), "multipathtcp") {
		// 禁用多路径TCP，Multipath TCP is not supported on macOS.
		// MultipathTCP is not supported on all platforms, and is notably unsupported in combination with TCP keep-alive.
		lc.SetMultipathTCP(false)
	}

	listener, err := lc.Listen(ctx, "tcp", config.GetAddress())
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", config.GetAddress(), err)
	}

	listener = tcpKeepAliveListener{listener.(*net.TCPListener)}

	// 如果配置了 ProxyProtocol，则创建一个 ProxyProtocol 监听器
	if config.ProxyProtocol != nil {
		listener, err = buildProxyProtocolListener(ctx, config, listener)
		if err != nil {
			return nil, fmt.Errorf("error creating proxy protocol listener: %w", err)
		}
	}

	return &onceCloseListener{Listener: listener}, nil
}
