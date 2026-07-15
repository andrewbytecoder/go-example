package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net"
	stdhttp "net/http"
	stdhttputil "net/http/httputil"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http/httpguts"
)

type httpContextKey string

const (
	// StatusClientClosedRequest is the non-standard HTTP status code for client disconnection.
	StatusClientClosedRequest = 499

	// StatusClientClosedRequestText is the text form of StatusClientClosedRequest.
	StatusClientClosedRequestText = "Client Closed Request"

	notAppendXFFKey httpContextKey = "NotAppendXFF"
)

// SetNotAppendXFF indicates X-Forwarded-For should not be appended.
func SetNotAppendXFF(ctx context.Context) context.Context {
	return context.WithValue(ctx, notAppendXFFKey, true)
}

// ShouldNotAppendXFF reports whether X-Forwarded-For should not be appended.
func ShouldNotAppendXFF(ctx context.Context) bool {
	val := ctx.Value(notAppendXFFKey)
	if val == nil {
		return false
	}

	notAppendXFF, ok := val.(bool)
	if !ok {
		return false
	}

	return notAppendXFF
}

// NewSingleHostReverseProxy creates a single-host reverse proxy using Traefik's HTTP core behavior.
func NewSingleHostReverseProxy(target *url.URL, passHostHeader, preservePath bool, flushInterval time.Duration, roundTripper stdhttp.RoundTripper) (stdhttp.Handler, error) {
	if target == nil {
		return nil, errors.New("target URL is nil")
	}

	if roundTripper == nil {
		roundTripper = stdhttp.DefaultTransport
	}

	return buildSingleHostHTTPProxy(target, passHostHeader, preservePath, flushInterval, roundTripper, newHTTPBufferPool()), nil
}

// NewSingleHostHTTPProxy creates a single-host HTTP reverse proxy.
func NewSingleHostHTTPProxy(target *url.URL, passHostHeader, preservePath bool, flushInterval time.Duration, roundTripper stdhttp.RoundTripper) (stdhttp.Handler, error) {
	return NewSingleHostReverseProxy(target, passHostHeader, preservePath, flushInterval, roundTripper)
}

// NewSingleHostHTTPSProxy creates a single-host HTTPS reverse proxy.
func NewSingleHostHTTPSProxy(target *url.URL, passHostHeader, preservePath bool, flushInterval time.Duration, roundTripper stdhttp.RoundTripper) (stdhttp.Handler, error) {
	return NewSingleHostReverseProxy(target, passHostHeader, preservePath, flushInterval, roundTripper)
}

// 简单的日志适配器，满足 tsdb.Logger 接口
type logWriter struct{}

func (l *logWriter) Write(p []byte) (n int, err error) {
	fmt.Print(string(p))
	return len(p), nil
}
func buildSingleHostHTTPProxy(target *url.URL, passHostHeader bool, preservePath bool, flushInterval time.Duration, roundTripper stdhttp.RoundTripper, bufferPool stdhttputil.BufferPool) stdhttp.Handler {
	return &stdhttputil.ReverseProxy{
		Rewrite:       rewriteRequestBuilder(target, passHostHeader, preservePath),
		Transport:     roundTripper,
		FlushInterval: flushInterval,
		BufferPool:    bufferPool,
		ErrorLog:      stdlog.New(&logWriter{}, "", 0),
		ErrorHandler:  ErrorHandler,
	}
}

func rewriteRequestBuilder(target *url.URL, passHostHeader bool, preservePath bool) func(*stdhttputil.ProxyRequest) {
	return func(pr *stdhttputil.ProxyRequest) {
		copyForwardedHeader(pr.Out.Header, pr.In.Header)
		if !ShouldNotAppendXFF(pr.In.Context()) {
			if clientIP, _, err := net.SplitHostPort(pr.In.RemoteAddr); err == nil {
				prior, ok := pr.Out.Header["X-Forwarded-For"]
				omit := ok && prior == nil
				if len(prior) > 0 {
					clientIP = strings.Join(prior, ", ") + ", " + clientIP
				}
				if !omit {
					pr.Out.Header.Set("X-Forwarded-For", clientIP)
				}
			}
		}

		pr.Out.URL.Scheme = target.Scheme
		pr.Out.URL.Host = target.Host

		u := pr.Out.URL
		if pr.Out.RequestURI != "" {
			parsedURL, err := url.ParseRequestURI(pr.Out.RequestURI)
			if err == nil {
				u = parsedURL
			}
		}

		pr.Out.URL.Path = u.Path
		pr.Out.URL.RawPath = u.RawPath

		if preservePath {
			pr.Out.URL.Path, pr.Out.URL.RawPath = JoinURLPath(target, u)
		}

		pr.Out.URL.RawQuery = strings.ReplaceAll(u.RawQuery, ";", "&")
		pr.Out.RequestURI = ""

		pr.Out.Proto = "HTTP/1.1"
		pr.Out.ProtoMajor = 1
		pr.Out.ProtoMinor = 1

		if !passHostHeader {
			pr.Out.Host = pr.Out.URL.Host
		}

		if isWebSocketUpgrade(pr.Out) {
			cleanWebSocketHeaders(pr.Out)
		}
	}
}

func copyForwardedHeader(dst, src stdhttp.Header) {
	prior, ok := src["X-Forwarded-For"]
	if ok {
		dst["X-Forwarded-For"] = prior
	}
	prior, ok = src["Forwarded"]
	if ok {
		dst["Forwarded"] = prior
	}
	prior, ok = src["X-Forwarded-Host"]
	if ok {
		dst["X-Forwarded-Host"] = prior
	}
	prior, ok = src["X-Forwarded-Proto"]
	if ok {
		dst["X-Forwarded-Proto"] = prior
	}
}

func cleanWebSocketHeaders(req *stdhttp.Request) {
	req.Header["Sec-WebSocket-Key"] = req.Header["Sec-Websocket-Key"]
	delete(req.Header, "Sec-Websocket-Key")

	req.Header["Sec-WebSocket-Extensions"] = req.Header["Sec-Websocket-Extensions"]
	delete(req.Header, "Sec-Websocket-Extensions")

	req.Header["Sec-WebSocket-Accept"] = req.Header["Sec-Websocket-Accept"]
	delete(req.Header, "Sec-Websocket-Accept")

	req.Header["Sec-WebSocket-Protocol"] = req.Header["Sec-Websocket-Protocol"]
	delete(req.Header, "Sec-Websocket-Protocol")

	req.Header["Sec-WebSocket-Version"] = req.Header["Sec-Websocket-Version"]
	delete(req.Header, "Sec-Websocket-Version")
}

func isWebSocketUpgrade(req *stdhttp.Request) bool {
	return httpguts.HeaderValuesContainsToken(req.Header["Connection"], "Upgrade") &&
		strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
}

// ErrorHandler is the handler called when something goes wrong while forwarding the request.
func ErrorHandler(w stdhttp.ResponseWriter, req *stdhttp.Request, err error) {
	ErrorHandlerWithContext(req.Context(), w, err)
}

// ErrorHandlerWithContext is the handler called when something goes wrong while forwarding the request.
func ErrorHandlerWithContext(ctx context.Context, w stdhttp.ResponseWriter, err error) {
	statusCode := ComputeStatusCode(err)

	if isTLSConfigError(err) {
	} else {
	}

	w.WriteHeader(statusCode)
	if _, werr := w.Write([]byte(statusText(statusCode))); werr != nil {
	}
}

func statusText(statusCode int) string {
	if statusCode == StatusClientClosedRequest {
		return StatusClientClosedRequestText
	}

	return stdhttp.StatusText(statusCode)
}

func isTLSConfigError(err error) bool {
	if _, ok := errors.AsType[tls.RecordHeaderError](err); ok {
		return true
	}

	_, ok := errors.AsType[*tls.CertificateVerificationError](err)
	return ok
}

// ComputeStatusCode computes the HTTP status code according to the given error.
func ComputeStatusCode(err error) int {
	switch {
	case errors.Is(err, io.EOF):
		return stdhttp.StatusBadGateway
	case errors.Is(err, context.Canceled):
		return StatusClientClosedRequest
	default:
		if netErr, ok := errors.AsType[net.Error](err); ok {
			if netErr.Timeout() {
				return stdhttp.StatusGatewayTimeout
			}

			return stdhttp.StatusBadGateway
		}
	}

	return stdhttp.StatusInternalServerError
}

// JoinURLPath computes the joined path and raw path of the given URLs.
func JoinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}

	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}

	return a.Path + b.Path, apath + bpath
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}

	return a + b
}
