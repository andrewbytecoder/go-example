package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"maps"
	"net"
	"sync"
	"time"

	ptypes "github.com/go-example/net/proxy/types"
	"github.com/pires/go-proxyproto"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
)

// TCPDialer dials backend TCP connections, optionally with PROXY protocol and termination delay support.
type TCPDialer interface {
	Dial(network, addr string, clientConn TCPClientConn) (net.Conn, error)
	DialContext(ctx context.Context, network, addr string, clientConn TCPClientConn) (net.Conn, error)
	TerminationDelay() time.Duration
}

// +k8s:deepcopy-gen=true

// ProxyProtocol holds the PROXY Protocol configuration.
// More info: https://doc.traefik.io/traefik/v3.7/routing/services/#proxy-protocol
type ProxyProtocol struct {
	// Version defines the PROXY Protocol version to use.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2
	Version int `description:"Defines the PROXY Protocol version to use." json:"version,omitempty" toml:"version,omitempty" yaml:"version,omitempty" export:"true"`
}

// SetDefaults Default values for a ProxyProtocol.
func (p *ProxyProtocol) SetDefaults() {
	p.Version = 2
}

type tcpDialer struct {
	dialer           *net.Dialer
	terminationDelay time.Duration
	proxyProtocol    *ProxyProtocol
}

func (d tcpDialer) TerminationDelay() time.Duration {
	return d.terminationDelay
}

func (d tcpDialer) Dial(network, addr string, clientConn TCPClientConn) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr, clientConn)
}

func (d tcpDialer) DialContext(ctx context.Context, network, addr string, clientConn TCPClientConn) (net.Conn, error) {
	conn, err := d.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	if d.proxyProtocol != nil && clientConn != nil && d.proxyProtocol.Version > 0 && d.proxyProtocol.Version < 3 {
		header := proxyproto.HeaderProxyFromAddrs(byte(d.proxyProtocol.Version), clientConn.RemoteAddr(), clientConn.LocalAddr())
		if _, err := header.WriteTo(conn); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("writing PROXY protocol header: %w", err)
		}
	}

	return conn, nil
}

type tcpTLSDialer struct {
	tcpDialer
	tlsConfig *tls.Config
}

func (d tcpTLSDialer) Dial(network, addr string, clientConn TCPClientConn) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr, clientConn)
}

func (d tcpTLSDialer) DialContext(ctx context.Context, network, addr string, clientConn TCPClientConn) (net.Conn, error) {
	conn, err := d.tcpDialer.DialContext(ctx, network, addr, clientConn)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, d.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	return tlsConn, nil
}

// TCPSpiffeX509Source allows retrieval of a x509 SVID and bundle.
type TCPSpiffeX509Source interface {
	x509svid.Source
	x509bundle.Source
}

// +k8s:deepcopy-gen=true

// TCPServersTransport options to configure communication between Traefik and the servers.
type TCPServersTransport struct {
	DialKeepAlive ptypes.Duration `description:"Defines the interval between keep-alive probes for an active network connection. If zero, keep-alive probes are sent with a default value (currently 15 seconds), if supported by the protocol and operating system. Network protocols or operating systems that do not support keep-alives ignore this field. If negative, keep-alive probes are disabled" json:"dialKeepAlive,omitempty" toml:"dialKeepAlive,omitempty" yaml:"dialKeepAlive,omitempty" export:"true"`
	DialTimeout   ptypes.Duration `description:"Defines the amount of time to wait until a connection to a backend server can be established. If zero, no timeout exists." json:"dialTimeout,omitempty" toml:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty" export:"true"`
	// ProxyProtocol holds the PROXY Protocol configuration.
	ProxyProtocol *ProxyProtocol `description:"Defines the PROXY Protocol configuration." json:"proxyProtocol,omitempty" toml:"proxyProtocol,omitempty" yaml:"proxyProtocol,omitempty" label:"allowEmpty" file:"allowEmpty" kv:"allowEmpty" export:"true"`
	// TerminationDelay, corresponds to the deadline that the proxy sets, after one
	// of its connected peers indicates it has closed the writing capability of its
	// connection, to close the reading capability as well, hence fully terminating the
	// connection. It is a duration in milliseconds, defaulting to 100. A negative value
	// means an infinite deadline (i.e. the reading capability is never closed).
	TerminationDelay ptypes.Duration  `description:"Defines the delay to wait before fully terminating the connection, after one connected peer has closed its writing capability." json:"terminationDelay,omitempty" toml:"terminationDelay,omitempty" yaml:"terminationDelay,omitempty" export:"true"`
	TLS              *TLSClientConfig `description:"Defines the TLS configuration." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" label:"allowEmpty" file:"allowEmpty" kv:"allowEmpty" export:"true"`
}

// TLSClientConfig options to configure TLS communication between Traefik and the servers.
type TLSClientConfig struct {
	ServerName         string          `description:"Defines the serverName used to contact the server." json:"serverName,omitempty" toml:"serverName,omitempty" yaml:"serverName,omitempty"`
	InsecureSkipVerify bool            `description:"Disables SSL certificate verification." json:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" export:"true"`
	RootCAs            []FileOrContent `description:"Defines a list of CA certificates used to validate server certificates." json:"rootCAs,omitempty" toml:"rootCAs,omitempty" yaml:"rootCAs,omitempty"`
	Certificates       Certificates    `description:"Defines a list of client certificates for mTLS." json:"certificates,omitempty" toml:"certificates,omitempty" yaml:"certificates,omitempty" export:"true"`
	// Deprecated: PeerCertURI is deprecated, please use the PeerCertSANs option instead.
	PeerCertURI  string  `description:"Defines the URI used to match against SAN URI during the peer certificate verification." json:"peerCertURI,omitempty" toml:"peerCertURI,omitempty" yaml:"peerCertURI,omitempty"`
	PeerCertSANs []SAN   `description:"Defines the SANs (Subject Alternative Names) used to match against SANs during the peer certificate verification." json:"peerCertSANs,omitempty" toml:"peerCertSANs,omitempty" yaml:"peerCertSANs,omitempty"`
	Spiffe       *Spiffe `description:"Defines the SPIFFE TLS configuration." json:"spiffe,omitempty" toml:"spiffe,omitempty" yaml:"spiffe,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// TCPDialerManager handles dialers for the reverse proxy.
type TCPDialerManager struct {
	serversTransportsMu sync.RWMutex
	serversTransports   map[string]*TCPServersTransport
	spiffeX509Source    TCPSpiffeX509Source
}

// NewTCPDialerManager creates a new TCPDialerManager.
func NewTCPDialerManager(spiffeX509Source TCPSpiffeX509Source) *TCPDialerManager {
	manager := &TCPDialerManager{
		serversTransports: make(map[string]*TCPServersTransport),
		spiffeX509Source:  spiffeX509Source,
	}

	manager.Update(map[string]*TCPServersTransport{"default@internal": {}})
	return manager
}

// Update updates the TCP servers transport configurations.
func (d *TCPDialerManager) Update(configs map[string]*TCPServersTransport) {
	if configs == nil {
		configs = map[string]*TCPServersTransport{}
	}

	if _, ok := configs["default@internal"]; !ok {
		configs = maps.Clone(configs)
		configs["default@internal"] = &TCPServersTransport{}
	}

	d.serversTransportsMu.Lock()
	defer d.serversTransportsMu.Unlock()

	d.serversTransports = configs
}

// +k8s:deepcopy-gen=true

// TCPServer holds a TCP Server configuration.
type TCPServer struct {
	Address string `json:"address,omitempty" toml:"address,omitempty" yaml:"address,omitempty" label:"-"`
	Port    string `json:"-" toml:"-" yaml:"-"`
	TLS     bool   `json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty"`
}

// +k8s:deepcopy-gen=true

// TCPServersLoadBalancer holds the LoadBalancerService configuration.
type TCPServersLoadBalancer struct {
	Servers          []TCPServer `json:"servers,omitempty" toml:"servers,omitempty" yaml:"servers,omitempty" label-slice-as-struct:"server" export:"true"`
	ServersTransport string      `json:"serversTransport,omitempty" toml:"serversTransport,omitempty" yaml:"serversTransport,omitempty" export:"true"`
	// ProxyProtocol holds the PROXY Protocol configuration.
	//
	// Deprecated: use ServersTransport to configure ProxyProtocol instead.
	ProxyProtocol *ProxyProtocol `json:"proxyProtocol,omitempty" toml:"proxyProtocol,omitempty" yaml:"proxyProtocol,omitempty" label:"allowEmpty" file:"allowEmpty" kv:"allowEmpty" export:"true"`
	// TerminationDelay, corresponds to the deadline that the proxy sets, after one
	// of its connected peers indicates it has closed the writing capability of its
	// connection, to close the reading capability as well, hence fully terminating the
	// connection. It is a duration in milliseconds, defaulting to 100. A negative value
	// means an infinite deadline (i.e. the reading capability is never closed).
	//
	// Deprecated: use ServersTransport to configure the TerminationDelay instead.
	TerminationDelay *int                  `json:"terminationDelay,omitempty" toml:"terminationDelay,omitempty" yaml:"terminationDelay,omitempty" export:"true"`
	HealthCheck      *TCPServerHealthCheck `json:"healthCheck,omitempty" toml:"healthCheck,omitempty" yaml:"healthCheck,omitempty" label:"allowEmpty" file:"allowEmpty" kv:"allowEmpty" export:"true"`
}

// +k8s:deepcopy-gen=true

// TCPServerHealthCheck holds the HealthCheck configuration.
type TCPServerHealthCheck struct {
	Port              int              `json:"port,omitempty" toml:"port,omitempty,omitzero" yaml:"port,omitempty" export:"true"`
	Send              string           `json:"send,omitempty" toml:"send,omitempty" yaml:"send,omitempty" export:"true"`
	Expect            string           `json:"expect,omitempty" toml:"expect,omitempty" yaml:"expect,omitempty" export:"true"`
	Interval          ptypes.Duration  `json:"interval,omitempty" toml:"interval,omitempty" yaml:"interval,omitempty" export:"true"`
	UnhealthyInterval *ptypes.Duration `json:"unhealthyInterval,omitempty" toml:"unhealthyInterval,omitempty" yaml:"unhealthyInterval,omitempty" export:"true"`
	Timeout           ptypes.Duration  `json:"timeout,omitempty" toml:"timeout,omitempty" yaml:"timeout,omitempty" export:"true"`
}

// Build builds a TCP dialer by configuration.
func (d *TCPDialerManager) Build(config *TCPServersLoadBalancer, isTLS bool) (TCPDialer, error) {
	name := "default@internal"
	if config.ServersTransport != "" {
		name = config.ServersTransport
	}

	d.serversTransportsMu.RLock()
	st, ok := d.serversTransports[name]
	d.serversTransportsMu.RUnlock()
	if !ok || st == nil {
		return nil, fmt.Errorf("no transport configuration found for %q", name)
	}

	var terminationDelay ptypes.Duration
	if config.TerminationDelay != nil {
		terminationDelay = ptypes.Duration(*config.TerminationDelay)
	}
	proxyProtocol := config.ProxyProtocol

	if config.ServersTransport != "" {
		terminationDelay = st.TerminationDelay
		proxyProtocol = st.ProxyProtocol
	}

	if proxyProtocol != nil && (proxyProtocol.Version < 1 || proxyProtocol.Version > 2) {
		return nil, fmt.Errorf("unknown proxyProtocol version: %d", proxyProtocol.Version)
	}

	var tlsConfig *tls.Config
	if st.TLS != nil {
		if st.TLS.Spiffe != nil {
			if d.spiffeX509Source == nil {
				return nil, errors.New("SPIFFE is enabled for this transport, but not configured")
			}

			authorizer, err := buildTCPSpiffeAuthorizer(st.TLS.Spiffe)
			if err != nil {
				return nil, fmt.Errorf("unable to build SPIFFE authorizer: %w", err)
			}

			tlsConfig = tlsconfig.MTLSClientConfig(d.spiffeX509Source, d.spiffeX509Source, authorizer)
		}

		if st.TLS.InsecureSkipVerify || len(st.TLS.RootCAs) > 0 || len(st.TLS.ServerName) > 0 || len(st.TLS.Certificates) > 0 || st.TLS.PeerCertURI != "" || len(st.TLS.PeerCertSANs) > 0 {
			if tlsConfig != nil {
				return nil, errors.New("TLS and SPIFFE configuration cannot be defined at the same time")
			}

			tlsConfig = &tls.Config{
				ServerName:         st.TLS.ServerName,
				InsecureSkipVerify: st.TLS.InsecureSkipVerify,
				RootCAs:            createTCPRootCACertPool(st.TLS.RootCAs),
				Certificates:       st.TLS.Certificates.GetCertificates(),
			}

			peerCertSANs := make([]SAN, len(st.TLS.PeerCertSANs))
			copy(peerCertSANs, st.TLS.PeerCertSANs)

			if st.TLS.PeerCertURI != "" {
				peerCertSANs = append(peerCertSANs, SAN{
					Type:  SANURIType,
					Value: st.TLS.PeerCertURI,
				})
			}

			if len(peerCertSANs) > 0 {
				tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
					return VerifyPeerCertificate(peerCertSANs, tlsConfig.RootCAs, rawCerts)
				}
			}
		}
	}

	dialer := tcpDialer{
		dialer: &net.Dialer{
			Timeout:   time.Duration(st.DialTimeout),
			KeepAlive: time.Duration(st.DialKeepAlive),
		},
		terminationDelay: time.Duration(terminationDelay),
		proxyProtocol:    proxyProtocol,
	}

	if !isTLS {
		return dialer, nil
	}

	return tcpTLSDialer{tcpDialer: dialer, tlsConfig: tlsConfig}, nil
}

func createTCPRootCACertPool(rootCAs []FileOrContent) *x509.CertPool {
	if len(rootCAs) == 0 {
		return nil
	}

	roots := x509.NewCertPool()
	for _, cert := range rootCAs {
		certContent, err := cert.Read()
		if err != nil {
			continue
		}

		roots.AppendCertsFromPEM(certContent)
	}

	return roots
}

func buildTCPSpiffeAuthorizer(cfg *Spiffe) (tlsconfig.Authorizer, error) {
	switch {
	case len(cfg.IDs) > 0:
		spiffeIDs := make([]spiffeid.ID, 0, len(cfg.IDs))
		for _, rawID := range cfg.IDs {
			id, err := spiffeid.FromString(rawID)
			if err != nil {
				return nil, fmt.Errorf("invalid SPIFFE ID: %w", err)
			}

			spiffeIDs = append(spiffeIDs, id)
		}

		return tlsconfig.AuthorizeOneOf(spiffeIDs...), nil
	case cfg.TrustDomain != "":
		trustDomain, err := spiffeid.TrustDomainFromString(cfg.TrustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid SPIFFE trust domain: %w", err)
		}

		return tlsconfig.AuthorizeMemberOf(trustDomain), nil
	default:
		return tlsconfig.AuthorizeAny(), nil
	}
}
