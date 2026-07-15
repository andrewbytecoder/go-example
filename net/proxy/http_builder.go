package proxy

import (
	"crypto/tls"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"time"

	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

// HTTPTransportProvider manages transports used for backend communication.
type HTTPTransportProvider interface {
	Get(name string) (*dynamic.ServersTransport, error)
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
func (b *HTTPProxyBuilder) Update(_ map[string]*dynamic.ServersTransport) {}

// Build builds a new reverse proxy with the given configuration.
func (b *HTTPProxyBuilder) Build(cfgName string, targetURL *url.URL, passHostHeader, preservePath bool, flushInterval time.Duration) (stdhttp.Handler, error) {
	roundTripper, err := b.transportManager.GetRoundTripper(cfgName)
	if err != nil {
		return nil, fmt.Errorf("getting round tripper: %w", err)
	}

	return buildSingleHostHTTPProxy(targetURL, passHostHeader, preservePath, flushInterval, roundTripper, b.bufferPool), nil
}
