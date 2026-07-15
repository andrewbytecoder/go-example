package proxy

import (
	"crypto/tls"
	"errors"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	stdhttputil "net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsed
}

func TestRewriteRequestBuilder(t *testing.T) {
	tests := []struct {
		name            string
		target          *url.URL
		passHostHeader  bool
		preservePath    bool
		incomingURL     string
		expectedScheme  string
		expectedHost    string
		expectedPath    string
		expectedRawPath string
		expectedQuery   string
		notAppendXFF    bool
	}{
		{
			name:           "basic proxy",
			target:         mustParseURL(t, "http://example.com"),
			passHostHeader: false,
			preservePath:   false,
			incomingURL:    "http://localhost/test?param=value",
			expectedScheme: "http",
			expectedHost:   "example.com",
			expectedPath:   "/test",
			expectedQuery:  "param=value",
		},
		{
			name:           "https target",
			target:         mustParseURL(t, "https://secure.example.com"),
			passHostHeader: false,
			preservePath:   false,
			incomingURL:    "http://localhost/secure",
			expectedScheme: "https",
			expectedHost:   "secure.example.com",
			expectedPath:   "/secure",
		},
		{
			name:            "preserve path",
			target:          mustParseURL(t, "http://example.com/base"),
			passHostHeader:  false,
			preservePath:    true,
			incomingURL:     "http://localhost/foo%2Fbar",
			expectedScheme:  "http",
			expectedHost:    "example.com",
			expectedPath:    "/base/foo/bar",
			expectedRawPath: "/base/foo%2Fbar",
		},
		{
			name:           "do not append xff",
			target:         mustParseURL(t, "http://example.com"),
			passHostHeader: false,
			preservePath:   false,
			incomingURL:    "http://localhost/test?param=value",
			expectedScheme: "http",
			expectedHost:   "example.com",
			expectedPath:   "/test",
			expectedQuery:  "param=value",
			notAppendXFF:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rewriteRequest := rewriteRequestBuilder(test.target, test.passHostHeader, test.preservePath)

			ctx := t.Context()
			if test.notAppendXFF {
				ctx = SetNotAppendXFF(ctx)
			}

			reqIn := httptest.NewRequest(stdhttp.MethodGet, test.incomingURL, stdhttp.NoBody)
			reqIn = reqIn.WithContext(ctx)
			reqIn.Header.Add("X-Forwarded-For", "1.2.3.4")
			reqIn.RemoteAddr = "127.0.0.1:1234"

			reqOut := httptest.NewRequest(stdhttp.MethodGet, test.incomingURL, stdhttp.NoBody)
			pr := &stdhttputil.ProxyRequest{In: reqIn, Out: reqOut}
			rewriteRequest(pr)

			if test.notAppendXFF {
				assert.Equal(t, "1.2.3.4", reqOut.Header.Get("X-Forwarded-For"))
			} else {
				assert.Equal(t, "1.2.3.4, 127.0.0.1", reqOut.Header.Get("X-Forwarded-For"))
			}

			assert.Equal(t, test.expectedScheme, reqOut.URL.Scheme)
			assert.Equal(t, test.expectedHost, reqOut.URL.Host)
			assert.Equal(t, test.expectedPath, reqOut.URL.Path)
			assert.Equal(t, test.expectedRawPath, reqOut.URL.RawPath)
			assert.Equal(t, test.expectedQuery, reqOut.URL.RawQuery)
			assert.Empty(t, reqOut.RequestURI)
		})
	}
}

func TestHTTPProxyWithHTTPSUpstream(t *testing.T) {
	backend := httptest.NewTLSServer(stdhttp.HandlerFunc(func(rw stdhttp.ResponseWriter, req *stdhttp.Request) {
		_, _ = rw.Write([]byte(req.URL.Path))
	}))
	t.Cleanup(backend.Close)

	manager := NewHTTPTransportManager(nil)
	manager.Update(map[string]*dynamic.ServersTransport{
		"default@internal": {InsecureSkipVerify: true},
	})

	builder := NewHTTPProxyBuilder(manager)
	proxyHandler, err := builder.Build("default@internal", mustParseURL(t, backend.URL), true, false, 0)
	require.NoError(t, err)

	proxy := httptest.NewServer(stdhttp.HandlerFunc(func(rw stdhttp.ResponseWriter, req *stdhttp.Request) {
		proxyHandler.ServeHTTP(rw, req)
	}))
	t.Cleanup(proxy.Close)

	resp, err := stdhttp.Get(proxy.URL + "/hello")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "/hello", string(body))
}

func TestIsTLSConfigError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "nil"},
		{name: "random", err: errors.New("random")},
		{name: "record header", err: tls.RecordHeaderError{}, expected: true},
		{name: "certificate verification", err: &tls.CertificateVerificationError{}, expected: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, isTLSConfigError(testCase.err))
		})
	}
}

func TestNewSingleHostHTTPProxy(t *testing.T) {
	backend := httptest.NewServer(stdhttp.HandlerFunc(func(rw stdhttp.ResponseWriter, req *stdhttp.Request) {
		_, _ = rw.Write([]byte("ok"))
	}))
	t.Cleanup(backend.Close)

	handler, err := NewSingleHostHTTPProxy(mustParseURL(t, backend.URL), true, false, time.Millisecond, stdhttp.DefaultTransport)
	require.NoError(t, err)

	proxy := httptest.NewServer(handler)
	t.Cleanup(proxy.Close)

	resp, err := stdhttp.Get(proxy.URL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))
}
