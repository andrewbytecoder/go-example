package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"maps"
	"net"
	stdhttp "net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"golang.org/x/net/http/httpguts"
)

// HTTPSpiffeX509Source allows retrieval of a x509 SVID and bundle.
type HTTPSpiffeX509Source interface {
	x509svid.Source
	x509bundle.Source
}

// HTTPTransportManager manages HTTP transports used for backend communication.
type HTTPTransportManager struct {
	rtLock        sync.RWMutex
	roundTrippers map[string]stdhttp.RoundTripper
	configs       map[string]*ServersTransport
	tlsConfigs    map[string]*tls.Config

	spiffeX509Source HTTPSpiffeX509Source
}

// NewHTTPTransportManager creates a new HTTPTransportManager.
func NewHTTPTransportManager(spiffeX509Source HTTPSpiffeX509Source) *HTTPTransportManager {
	manager := &HTTPTransportManager{
		roundTrippers:    make(map[string]stdhttp.RoundTripper),
		configs:          make(map[string]*ServersTransport),
		tlsConfigs:       make(map[string]*tls.Config),
		spiffeX509Source: spiffeX509Source,
	}

	manager.Update(map[string]*ServersTransport{"default@internal": {}})
	return manager
}

// Update updates the transport configurations.
func (t *HTTPTransportManager) Update(newConfigs map[string]*ServersTransport) {
	if newConfigs == nil {
		newConfigs = map[string]*ServersTransport{}
	}

	if _, ok := newConfigs["default@internal"]; !ok {
		newConfigs = maps.Clone(newConfigs)
		newConfigs["default@internal"] = &ServersTransport{}
	}

	t.rtLock.Lock()
	defer t.rtLock.Unlock()

	for configName, config := range t.configs {
		newConfig, ok := newConfigs[configName]
		if !ok {
			delete(t.configs, configName)
			delete(t.roundTrippers, configName)
			delete(t.tlsConfigs, configName)
			continue
		}

		if reflect.DeepEqual(newConfig, config) {
			continue
		}

		var (
			err       error
			tlsConfig *tls.Config
		)
		if tlsConfig, err = t.createTLSConfig(newConfig); err != nil {
		}
		t.tlsConfigs[configName] = tlsConfig

		t.roundTrippers[configName], err = t.createRoundTripper(newConfig, tlsConfig)
		if err != nil {
			t.roundTrippers[configName] = stdhttp.DefaultTransport
		}
	}

	for newConfigName, newConfig := range newConfigs {
		if _, ok := t.configs[newConfigName]; ok {
			continue
		}

		var (
			err       error
			tlsConfig *tls.Config
		)
		if tlsConfig, err = t.createTLSConfig(newConfig); err != nil {
		}
		t.tlsConfigs[newConfigName] = tlsConfig

		t.roundTrippers[newConfigName], err = t.createRoundTripper(newConfig, tlsConfig)
		if err != nil {
			t.roundTrippers[newConfigName] = stdhttp.DefaultTransport
		}
	}

	t.configs = newConfigs
}

// GetRoundTripper gets the round tripper corresponding to the given transport name.
func (t *HTTPTransportManager) GetRoundTripper(name string) (stdhttp.RoundTripper, error) {
	if name == "" {
		name = "default@internal"
	}

	t.rtLock.RLock()
	defer t.rtLock.RUnlock()

	if rt, ok := t.roundTrippers[name]; ok {
		return rt, nil
	}

	return nil, fmt.Errorf("servers transport not found %s", name)
}

// Get gets a transport by name.
func (t *HTTPTransportManager) Get(name string) (*ServersTransport, error) {
	if name == "" {
		name = "default@internal"
	}

	t.rtLock.RLock()
	defer t.rtLock.RUnlock()

	if rt, ok := t.configs[name]; ok {
		return rt, nil
	}

	return nil, fmt.Errorf("servers transport not found %s", name)
}

// GetTLSConfig gets the TLS config corresponding to the given transport name.
func (t *HTTPTransportManager) GetTLSConfig(name string) (*tls.Config, error) {
	if name == "" {
		name = "default@internal"
	}

	t.rtLock.RLock()
	defer t.rtLock.RUnlock()

	if rt, ok := t.tlsConfigs[name]; ok {
		return rt, nil
	}

	return nil, fmt.Errorf("tls config not found %s", name)
}

func (t *HTTPTransportManager) createTLSConfig(cfg *ServersTransport) (*tls.Config, error) {
	var config *tls.Config
	if cfg.Spiffe != nil {
		if t.spiffeX509Source == nil {
			return nil, errors.New("SPIFFE is enabled for this transport, but not configured")
		}

		spiffeAuthorizer, err := buildHTTPSpiffeAuthorizer(cfg.Spiffe)
		if err != nil {
			return nil, fmt.Errorf("unable to build SPIFFE authorizer: %w", err)
		}

		config = tlsconfig.MTLSClientConfig(t.spiffeX509Source, t.spiffeX509Source, spiffeAuthorizer)
	}

	if cfg.InsecureSkipVerify || len(cfg.RootCAs) > 0 || len(cfg.ServerName) > 0 || len(cfg.Certificates) > 0 || cfg.PeerCertURI != "" || len(cfg.PeerCertSANs) > 0 || len(cfg.CipherSuites) > 0 || cfg.MaxVersion != "" || cfg.MinVersion != "" {
		if config != nil {
			return nil, errors.New("TLS and SPIFFE configuration cannot be defined at the same time")
		}

		var cipherSuites []uint16
		for _, cipher := range cfg.CipherSuites {
			cipherID, exists := CipherSuites[cipher]
			if !exists {
				return nil, fmt.Errorf("invalid CipherSuite: %s", cipher)
			}
			cipherSuites = append(cipherSuites, cipherID)
		}

		var minVersion uint16
		if cfg.MinVersion != "" {
			value, exists := MinVersion[cfg.MinVersion]
			if !exists {
				return nil, fmt.Errorf("invalid TLS minimum version: %s", cfg.MinVersion)
			}
			minVersion = value
		}

		var maxVersion uint16
		if cfg.MaxVersion != "" {
			value, exists := MaxVersion[cfg.MaxVersion]
			if !exists {
				return nil, fmt.Errorf("invalid TLS maximum version: %s", cfg.MaxVersion)
			}
			maxVersion = value
		}

		if minVersion > maxVersion {
			return nil, fmt.Errorf("TLS minimum version %s is above the maximum version %s", cfg.MinVersion, cfg.MaxVersion)
		}

		config = &tls.Config{
			ServerName:         cfg.ServerName,
			InsecureSkipVerify: cfg.InsecureSkipVerify,
			RootCAs:            createHTTPRootCACertPool(cfg.RootCAs),
			Certificates:       cfg.Certificates.GetCertificates(),
			CipherSuites:       cipherSuites,
			MinVersion:         minVersion,
			MaxVersion:         maxVersion,
		}

		peerCertSANs := make([]SAN, len(cfg.PeerCertSANs))
		copy(peerCertSANs, cfg.PeerCertSANs)

		if cfg.PeerCertURI != "" {
			peerCertSANs = append(peerCertSANs, SAN{
				Type:  SANURIType,
				Value: cfg.PeerCertURI,
			})
		}

		if len(peerCertSANs) > 0 {
			config.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
				return VerifyPeerCertificate(peerCertSANs, config.RootCAs, rawCerts)
			}
		}
	}

	return config, nil
}

type httpConnWithTimeouts struct {
	net.Conn

	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (c httpConnWithTimeouts) Read(b []byte) (n int, err error) {
	if c.readTimeout > 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		defer c.Conn.SetReadDeadline(time.Time{}) //nolint:errcheck
	}

	return c.Conn.Read(b)
}

func (c httpConnWithTimeouts) Write(b []byte) (n int, err error) {
	if c.writeTimeout > 0 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
		defer c.Conn.SetWriteDeadline(time.Time{}) //nolint:errcheck
	}

	return c.Conn.Write(b)
}

func customHTTPDialContext(dialer *net.Dialer, cfg *ForwardingTimeouts) func(ctx context.Context, network string, address string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if cfg.ReadTimeout <= 0 && cfg.WriteTimeout <= 0 {
			return conn, err
		}

		return &httpConnWithTimeouts{
			Conn:         conn,
			readTimeout:  time.Duration(cfg.ReadTimeout),
			writeTimeout: time.Duration(cfg.WriteTimeout),
		}, err
	}
}

func (t *HTTPTransportManager) createRoundTripper(cfg *ServersTransport, tlsConfig *tls.Config) (stdhttp.RoundTripper, error) {
	if cfg == nil {
		return nil, errors.New("no transport configuration given")
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	if cfg.ForwardingTimeouts != nil {
		dialer.Timeout = time.Duration(cfg.ForwardingTimeouts.DialTimeout)
	}

	transport := &stdhttp.Transport{
		Proxy:                 stdhttp.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ReadBufferSize:        64 * 1024,
		WriteBufferSize:       64 * 1024,
		TLSClientConfig:       tlsConfig,
	}

	if cfg.ForwardingTimeouts != nil {
		transport.ResponseHeaderTimeout = time.Duration(cfg.ForwardingTimeouts.ResponseHeaderTimeout)
		transport.IdleConnTimeout = time.Duration(cfg.ForwardingTimeouts.IdleConnTimeout)
		transport.DialContext = customHTTPDialContext(dialer, cfg.ForwardingTimeouts)
		transport.HTTP2 = &stdhttp.HTTP2Config{
			SendPingTimeout: time.Duration(cfg.ForwardingTimeouts.ReadIdleTimeout),
			PingTimeout:     time.Duration(cfg.ForwardingTimeouts.PingTimeout),
		}
	}

	if cfg.DisableHTTP2 {
		return &httpKerberosRoundTripper{
			OriginalRoundTripper: transport,
			new: func() stdhttp.RoundTripper {
				return transport.Clone()
			},
		}, nil
	}

	rt := newHTTPSmartRoundTripper(transport)
	return &httpKerberosRoundTripper{
		OriginalRoundTripper: rt,
		new: func() stdhttp.RoundTripper {
			return rt.Clone()
		},
	}, nil
}

type stickyHTTPRoundTripper struct {
	RoundTripper stdhttp.RoundTripper
}

type httpTransportContextKey string

var httpTransportKey httpTransportContextKey = "transport"

// AddHTTPTransportOnContext stores the sticky transport holder in the request context.
func AddHTTPTransportOnContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, httpTransportKey, &stickyHTTPRoundTripper{})
}

type httpKerberosRoundTripper struct {
	new                  func() stdhttp.RoundTripper
	OriginalRoundTripper stdhttp.RoundTripper
}

func (k *httpKerberosRoundTripper) RoundTrip(request *stdhttp.Request) (*stdhttp.Response, error) {
	value, ok := request.Context().Value(httpTransportKey).(*stickyHTTPRoundTripper)
	if !ok {
		return k.OriginalRoundTripper.RoundTrip(request)
	}

	if value.RoundTripper != nil {
		return value.RoundTripper.RoundTrip(request)
	}

	resp, err := k.OriginalRoundTripper.RoundTrip(request)
	if err == nil && containsNTLMorNegotiate(resp.Header.Values("WWW-Authenticate")) {
		value.RoundTripper = k.new()
	}

	return resp, err
}

func containsNTLMorNegotiate(values []string) bool {
	return slices.ContainsFunc(values, func(value string) bool {
		return strings.HasPrefix(value, "NTLM") || strings.HasPrefix(value, "Negotiate")
	})
}

type httpSmartRoundTripper struct {
	http2 *stdhttp.Transport
	http  *stdhttp.Transport
	h2c   *stdhttp.Transport
}

func newHTTPSmartRoundTripper(transport *stdhttp.Transport) *httpSmartRoundTripper {
	transportHTTP1 := transport.Clone()
	transportHTTP1.Protocols = new(stdhttp.Protocols)
	transportHTTP1.Protocols.SetHTTP1(true)

	transportHTTP2 := transport.Clone()
	transportHTTP2.Protocols = new(stdhttp.Protocols)
	transportHTTP2.Protocols.SetHTTP1(true)
	transportHTTP2.Protocols.SetHTTP2(true)

	transportH2C := transport.Clone()
	transportH2C.Protocols = new(stdhttp.Protocols)
	transportH2C.Protocols.SetUnencryptedHTTP2(true)

	return &httpSmartRoundTripper{
		http2: transportHTTP2,
		http:  transportHTTP1,
		h2c:   transportH2C,
	}
}

func (m *httpSmartRoundTripper) Clone() stdhttp.RoundTripper {
	return &httpSmartRoundTripper{
		http2: m.http2.Clone(),
		http:  m.http.Clone(),
		h2c:   m.h2c.Clone(),
	}
}

func (m *httpSmartRoundTripper) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	h2c := req.URL.Scheme == "h2c"
	if h2c {
		req.URL.Scheme = "http"
	}

	if httpguts.HeaderValuesContainsToken(req.Header["Connection"], "Upgrade") {
		return m.http.RoundTrip(req)
	}

	if h2c {
		return m.h2c.RoundTrip(req)
	}

	return m.http2.RoundTrip(req)
}

func createHTTPRootCACertPool(rootCAs []FileOrContent) *x509.CertPool {
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

func buildHTTPSpiffeAuthorizer(cfg *Spiffe) (tlsconfig.Authorizer, error) {
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
