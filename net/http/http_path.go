package http

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// ComputeExternalURL computes a sanitized external URL from a raw input. It infers unset
// URL parts from the OS and the given listen address.
func ComputeExternalURL(u, listenAddr string) (*url.URL, error) {
	if u == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		_, port, err := net.SplitHostPort(listenAddr)
		if err != nil {
			return nil, err
		}
		u = fmt.Sprintf("http://%s:%s/", hostname, port)
	}

	// u 里面不能是 " ' 开头的字符串，否则会报错
	if startsOrEndsWithQuote(u) {
		return nil, errors.New("URL must not begin or end with quotes")
	}

	// 将url 解析成url.URL对象
	eu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	// url 以 http:// 开头, 经过url.Parse() 解析 url.Path为 "/data" 开头的uri路径
	// 这里保证路径以 / 开头
	ppref := strings.TrimRight(eu.Path, "/")
	if ppref != "" && !strings.HasPrefix(ppref, "/") {
		ppref = "/" + ppref
	}
	eu.Path = ppref

	return eu, nil
}

func startsOrEndsWithQuote(s string) bool {
	return strings.HasPrefix(s, "\"") || strings.HasPrefix(s, "'") ||
		strings.HasSuffix(s, "\"") || strings.HasSuffix(s, "'")
}
