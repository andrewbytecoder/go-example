package proxy

import (
	"crypto/tls"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"time"

	ptypes "github.com/go-example/net/proxy/types"
)

// ForwardingTimeouts contains timeout configurations for forwarding requests to the backend servers.
type ForwardingTimeouts struct {
	DialTimeout           ptypes.Duration `description:"The amount of time to wait until a connection to a backend server can be established. If zero, no timeout exists." json:"dialTimeout,omitempty" toml:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty" export:"true"`
	ResponseHeaderTimeout ptypes.Duration `description:"The amount of time to wait for a server's response headers after fully writing the request (including its body, if any). If zero, no timeout exists." json:"responseHeaderTimeout,omitempty" toml:"responseHeaderTimeout,omitempty" yaml:"responseHeaderTimeout,omitempty" export:"true"`
	IdleConnTimeout       ptypes.Duration `description:"The maximum period for which an idle HTTP keep-alive connection will remain open before closing itself." json:"idleConnTimeout,omitempty" toml:"idleConnTimeout,omitempty" yaml:"idleConnTimeout,omitempty" export:"true"`
	ReadIdleTimeout       ptypes.Duration `description:"The timeout after which a health check using ping frame will be carried out if no frame is received on the HTTP/2 connection. If zero, no health check is performed." json:"readIdleTimeout,omitempty" toml:"readIdleTimeout,omitempty" yaml:"readIdleTimeout,omitempty" export:"true"`
	PingTimeout           ptypes.Duration `description:"The timeout after which the HTTP/2 connection will be closed if a response to ping is not received." json:"pingTimeout,omitempty" toml:"pingTimeout,omitempty" yaml:"pingTimeout,omitempty" export:"true"`

	// related to NGINX provider
	ReadTimeout  ptypes.Duration `description:"Defines a timeout for reading a response from the proxied server. The timeout between two successive read operations. The connection is closed if nothing is transmitted within this time." json:"-" toml:"-" yaml:"-" export:"true"`
	WriteTimeout ptypes.Duration `description:"Defines a timeout for transmitting a request to the proxied server. The timeout between two successive write operations. The connection is closed if nothing is transmitted within this time." json:"-" toml:"-" yaml:"-" export:"true"`
}

// SetDefaults sets the default values.
func (f *ForwardingTimeouts) SetDefaults() {
	f.DialTimeout = ptypes.Duration(30 * time.Second)
	f.IdleConnTimeout = ptypes.Duration(90 * time.Second)
	f.PingTimeout = ptypes.Duration(15 * time.Second)
}

// ServersTransport options to configure communication between Traefik and the servers.
type ServersTransport struct {
	ServerName          string              `description:"Defines the serverName used to contact the server." json:"serverName,omitempty" toml:"serverName,omitempty" yaml:"serverName,omitempty"`
	InsecureSkipVerify  bool                `description:"Disables SSL certificate verification." json:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" export:"true"`
	RootCAs             []FileOrContent     `description:"Defines a list of CA certificates used to validate server certificates." json:"rootCAs,omitempty" toml:"rootCAs,omitempty" yaml:"rootCAs,omitempty"`
	Certificates        Certificates        `description:"Defines a list of client certificates for mTLS." json:"certificates,omitempty" toml:"certificates,omitempty" yaml:"certificates,omitempty" export:"true"`
	CipherSuites        []string            `description:"Defines the cipher suites to use when contacting backend servers." json:"cipherSuites,omitempty" toml:"cipherSuites,omitempty" yaml:"cipherSuites,omitempty" export:"true"`
	MinVersion          string              `description:"Defines the minimum TLS version to use when contacting backend servers." json:"minVersion,omitempty" toml:"minVersion,omitempty" yaml:"minVersion,omitempty" export:"true"`
	MaxVersion          string              `description:"Defines the maximum TLS version to use when contacting backend servers." json:"maxVersion,omitempty" toml:"maxVersion,omitempty" yaml:"maxVersion,omitempty" export:"true"`
	MaxIdleConnsPerHost int                 `description:"If non-zero, controls the maximum idle (keep-alive) to keep per-host. If zero, DefaultMaxIdleConnsPerHost is used. If negative, disables connection reuse." json:"maxIdleConnsPerHost,omitempty" toml:"maxIdleConnsPerHost,omitempty" yaml:"maxIdleConnsPerHost,omitempty" export:"true"`
	ForwardingTimeouts  *ForwardingTimeouts `description:"Defines the timeouts for requests forwarded to the backend servers." json:"forwardingTimeouts,omitempty" toml:"forwardingTimeouts,omitempty" yaml:"forwardingTimeouts,omitempty" export:"true"`
	DisableHTTP2        bool                `description:"Disables HTTP/2 for connections with backend servers." json:"disableHTTP2,omitempty" toml:"disableHTTP2,omitempty" yaml:"disableHTTP2,omitempty" export:"true"`
	// Deprecated: PeerCertURI is deprecated, please use the PeerCertSANs option instead.
	PeerCertURI  string  `description:"Defines the URI used to match against SAN URI during the peer certificate verification." json:"peerCertURI,omitempty" toml:"peerCertURI,omitempty" yaml:"peerCertURI,omitempty"`
	PeerCertSANs []SAN   `description:"Defines the SANs (Subject Alternative Names) used to match against SANs during the peer certificate verification." json:"peerCertSANs,omitempty" toml:"peerCertSANs,omitempty" yaml:"peerCertSANs,omitempty"`
	Spiffe       *Spiffe `description:"Defines the SPIFFE configuration." json:"spiffe,omitempty" toml:"spiffe,omitempty" yaml:"spiffe,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// +k8s:deepcopy-gen=true

// Spiffe holds the SPIFFE configuration.
type Spiffe struct {
	// IDs defines the allowed SPIFFE IDs (takes precedence over the SPIFFE TrustDomain).
	IDs []string `description:"Defines the allowed SPIFFE IDs (takes precedence over the SPIFFE TrustDomain)." json:"ids,omitempty" toml:"ids,omitempty" yaml:"ids,omitempty"`
	// TrustDomain defines the allowed SPIFFE trust domain.
	TrustDomain string `description:"Defines the allowed SPIFFE trust domain." json:"trustDomain,omitempty" toml:"trustDomain,omitempty" yaml:"trustDomain,omitempty"`
}

// HTTPTransportProvider manages transports used for backend communication.
type HTTPTransportProvider interface {
	Get(name string) (*ServersTransport, error)
	GetRoundTripper(name string) (stdhttp.RoundTripper, error)
	GetTLSConfig(name string) (*tls.Config, error)
}

// HTTPProxyBuilder builds HTTP and HTTPS reverse proxies.
type HTTPProxyBuilder struct {
	bufferPool       *httpBufferPool
	transportManager HTTPTransportProvider
}

// NewHTTPProxyBuilder creates a new HTTPProxyBuilder.
func NewHTTPProxyBuilder(transportManager HTTPTransportProvider) *HTTPProxyBuilder {
	return &HTTPProxyBuilder{
		bufferPool:       newHTTPBufferPool(),
		transportManager: transportManager,
	}
}

// Update does nothing and is kept to mirror Traefik's builder shape.
func (b *HTTPProxyBuilder) Update(_ map[string]*ServersTransport) {}

// Build builds a new reverse proxy with the given configuration.
func (b *HTTPProxyBuilder) Build(cfgName string, targetURL *url.URL, passHostHeader, preservePath bool, flushInterval time.Duration) (stdhttp.Handler, error) {
	roundTripper, err := b.transportManager.GetRoundTripper(cfgName)
	if err != nil {
		return nil, fmt.Errorf("getting round tripper: %w", err)
	}

	return buildSingleHostHTTPProxy(targetURL, passHostHeader, preservePath, flushInterval, roundTripper, b.bufferPool), nil
}
