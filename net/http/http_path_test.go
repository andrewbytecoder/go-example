package http

import (
	"os"
	"testing"
)

// 给一个 http://www.test.com/prometheus/api/v1/query?query=up 路径的正确解析方式，不是进行字符串的分割
// 而是使用 url.Parse() 方法进行解析，然后再对各个部分分开进行解析

func TestComputeExternalURL(t *testing.T) {
	tests := []struct {
		name       string
		u          string
		listenAddr string
		wantURL    string
		wantErr    bool
	}{
		{
			name:       "完整 URL - 标准格式",
			u:          "http://localhost:9090/prometheus",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "http://localhost:9090/prometheus",
			wantErr:    false,
		},
		{
			name:       "完整 URL - 带斜杠结尾",
			u:          "http://localhost:9090/prometheus/",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "http://localhost:9090/prometheus",
			wantErr:    false,
		},
		{
			name:       "完整 URL - 多个斜杠结尾",
			u:          "http://localhost:9090/prometheus///",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "http://localhost:9090/prometheus",
			wantErr:    false,
		},
		{
			name:       "完整 URL - 无路径",
			u:          "http://localhost:9090",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "http://localhost:9090",
			wantErr:    false,
		},
		{
			name:       "完整 URL - 根路径",
			u:          "http://localhost:9090/",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "http://localhost:9090",
			wantErr:    false,
		},
		{
			name:       "完整 URL - HTTPS 协议",
			u:          "https://example.com:443/api",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "https://example.com:443/api",
			wantErr:    false,
		},
		{
			name:       "完整 URL - 复杂路径",
			u:          "http://prometheus.example.com:9090/api/v1/query",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "http://prometheus.example.com:9090/api/v1/query",
			wantErr:    false,
		},
		{
			name:       "空 URL - 使用监听地址推断",
			u:          "",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "",
			wantErr:    false,
		},
		{
			name:       "无效 URL - 缺少协议",
			u:          "localhost:9090/prometheus",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "",
			wantErr:    true,
		},
		{
			name:       "无效 URL - 以双引号开头",
			u:          "\"http://localhost:9090/prometheus\"",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "",
			wantErr:    true,
		},
		{
			name:       "无效 URL - 以单引号开头",
			u:          "'http://localhost:9090/prometheus'",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "",
			wantErr:    true,
		},
		{
			name:       "无效 URL - 以双引号结尾",
			u:          "http://localhost:9090/prometheus\"",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "",
			wantErr:    true,
		},
		{
			name:       "无效 URL - 以单引号结尾",
			u:          "http://localhost:9090/prometheus'",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "",
			wantErr:    true,
		},
		{
			name:       "无效 URL - 语法错误",
			u:          "http://localhost:invalid/path",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "",
			wantErr:    true,
		},
		{
			name:       "相对路径 - 自动添加斜杠",
			u:          "http://localhost:9090/test",
			listenAddr: "0.0.0.0:9090",
			wantURL:    "http://localhost:9090/test",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ComputeExternalURL(tt.u, tt.listenAddr)

			if (err != nil) != tt.wantErr {
				t.Errorf("ComputeExternalURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != nil {
				if tt.wantURL != "" {
					gotStr := got.String()
					if gotStr != tt.wantURL {
						t.Errorf("ComputeExternalURL() = %v, want %v", gotStr, tt.wantURL)
					}
				} else {
					// 对于空 URL 的情况，只验证没有错误且返回了对象
					if got == nil {
						t.Errorf("ComputeExternalURL() returned nil, expected non-nil URL")
					}
				}
			}
		})
	}
}

func TestStartsOrEndsWithQuote(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		expected bool
	}{
		{
			name:     "双引号开头",
			s:        "\"hello",
			expected: true,
		},
		{
			name:     "单引号开头",
			s:        "'hello",
			expected: true,
		},
		{
			name:     "双引号结尾",
			s:        "hello\"",
			expected: true,
		},
		{
			name:     "单引号结尾",
			s:        "hello'",
			expected: true,
		},
		{
			name:     "双引号开头和结尾",
			s:        "\"hello\"",
			expected: true,
		},
		{
			name:     "单引号开头和结尾",
			s:        "'hello'",
			expected: true,
		},
		{
			name:     "无引号",
			s:        "hello",
			expected: false,
		},
		{
			name:     "空字符串",
			s:        "",
			expected: false,
		},
		{
			name:     "中间有引号",
			s:        "hel\"lo",
			expected: false,
		},
		{
			name:     "URL 带引号开头",
			s:        "\"http://example.com\"",
			expected: true,
		},
		{
			name:     "URL 带引号结尾",
			s:        "http://example.com\"",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := startsOrEndsWithQuote(tt.s)
			if result != tt.expected {
				t.Errorf("startsOrEndsWithQuote(%q) = %v, want %v", tt.s, result, tt.expected)
			}
		})
	}
}

func TestComputeExternalURL_EmptyUsesHostname(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Skipf("Cannot get hostname: %v", err)
	}

	got, err := ComputeExternalURL("", "0.0.0.0:9090")
	if err != nil {
		t.Fatalf("ComputeExternalURL() unexpected error = %v", err)
	}

	if got.Host == "" {
		t.Error("Expected hostname to be set when URL is empty")
	}

	expectedHost := hostname + ":9090"
	if got.Host != expectedHost {
		t.Errorf("Expected host %q, got %q", expectedHost, got.Host)
	}
}
