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

	"github.com/rs/zerolog/log"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
	traefiktls "github.com/traefik/traefik/v3/pkg/tls"
	"github.com/traefik/traefik/v3/pkg/types"
)

// TCPDialer dials backend TCP connections, optionally with PROXY protocol and termination delay support.
type TCPDialer interface {
	Dial(network, addr string, clientConn TCPClientConn) (net.Conn, error)
	DialContext(ctx context.Context, network, addr string, clientConn TCPClientConn) (net.Conn, error)
	TerminationDelay() time.Duration
}

type tcpDialer struct {
	dialer           *net.Dialer
	terminationDelay time.Duration
	proxyProtocol    *dynamic.ProxyProtocol
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

// TCPDialerManager handles dialers for the reverse proxy.
type TCPDialerManager struct {
	serversTransportsMu sync.RWMutex
	serversTransports   map[string]*dynamic.TCPServersTransport
	spiffeX509Source    TCPSpiffeX509Source
}

// NewTCPDialerManager creates a new TCPDialerManager.
func NewTCPDialerManager(spiffeX509Source TCPSpiffeX509Source) *TCPDialerManager {
	manager := &TCPDialerManager{
		serversTransports: make(map[string]*dynamic.TCPServersTransport),
		spiffeX509Source:  spiffeX509Source,
	}

	manager.Update(map[string]*dynamic.TCPServersTransport{"default@internal": {}})
	return manager
}

// Update updates the TCP servers transport configurations.
func (d *TCPDialerManager) Update(configs map[string]*dynamic.TCPServersTransport) {
	if configs == nil {
		configs = map[string]*dynamic.TCPServersTransport{}
	}

	if _, ok := configs["default@internal"]; !ok {
		configs = maps.Clone(configs)
		configs["default@internal"] = &dynamic.TCPServersTransport{}
	}

	d.serversTransportsMu.Lock()
	defer d.serversTransportsMu.Unlock()

	d.serversTransports = configs
}

// Build builds a TCP dialer by configuration.
func (d *TCPDialerManager) Build(config *dynamic.TCPServersLoadBalancer, isTLS bool) (TCPDialer, error) {
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

			peerCertSANs := make([]traefiktls.SAN, len(st.TLS.PeerCertSANs))
			copy(peerCertSANs, st.TLS.PeerCertSANs)

			if st.TLS.PeerCertURI != "" {
				log.Warn().Msg("PeerCertURI option is deprecated, please use PeerCertSANs instead")
				peerCertSANs = append(peerCertSANs, traefiktls.SAN{
					Type:  traefiktls.SANURIType,
					Value: st.TLS.PeerCertURI,
				})
			}

			if len(peerCertSANs) > 0 {
				tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
					return traefiktls.VerifyPeerCertificate(peerCertSANs, tlsConfig.RootCAs, rawCerts)
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

func createTCPRootCACertPool(rootCAs []types.FileOrContent) *x509.CertPool {
	if len(rootCAs) == 0 {
		return nil
	}

	roots := x509.NewCertPool()
	for _, cert := range rootCAs {
		certContent, err := cert.Read()
		if err != nil {
			log.Error().Err(err).Msg("Error while reading RootCAs")
			continue
		}

		roots.AppendCertsFromPEM(certContent)
	}

	return roots
}

func buildTCPSpiffeAuthorizer(cfg *dynamic.Spiffe) (tlsconfig.Authorizer, error) {
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
