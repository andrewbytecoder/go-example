package regexp

import (
	"testing"
)

func TestPatRulePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "纯静态路径 - 简单",
			path:     "/etc/prometheus/rules.yml",
			expected: true,
		},
		{
			name:     "纯静态路径 - 多级目录",
			path:     "/etc/prometheus/rules/alpha.yml",
			expected: true,
		},
		{
			name:     "纯静态路径 - 不同扩展名",
			path:     "/etc/prometheus/rules.yaml",
			expected: true,
		},
		{
			name:     "纯静态路径 - 无扩展名",
			path:     "/etc/prometheus/rules",
			expected: true,
		},
		{
			name:     "纯静态路径 - 根目录文件",
			path:     "/rules.yml",
			expected: true,
		},
		{
			name:     "通配符路径 - 星号结尾",
			path:     "/etc/prometheus/*.yml",
			expected: true,
		},
		{
			name:     "通配符路径 - 星号后跟扩展名",
			path:     "/etc/prometheus/*.yaml",
			expected: true,
		},
		{
			name:     "通配符路径 - 仅星号",
			path:     "/etc/prometheus/*",
			expected: true,
		},
		{
			name:     "通配符路径 - 复杂路径",
			path:     "/etc/prometheus/rules/*.yml",
			expected: true,
		},
		{
			name:     "无效路径 - 星号在中间",
			path:     "/etc/*/prometheus/rules.yml",
			expected: false,
		},
		{
			name:     "无效路径 - 多个星号",
			path:     "/etc/prometheus/*.yml",
			expected: true,
		},
		{
			name:     "无效路径 - 星号后跟斜杠",
			path:     "/etc/prometheus/*/",
			expected: false,
		},
		{
			name:     "无效路径 - 星号在路径中间",
			path:     "/etc/prometheus/*/rules.yml",
			expected: false,
		},
		{
			name:     "有效路径 - 空字符串",
			path:     "",
			expected: true,
		},
		{
			name:     "有效路径 - 相对路径",
			path:     "etc/prometheus/rules.yml",
			expected: true,
		},
		{
			name:     "有效路径 - 当前目录",
			path:     "./etc/prometheus/rules.yml",
			expected: true,
		},
		{
			name:     "无效路径 - 星号后有子目录",
			path:     "/etc/prometheus/*/sub/rules.yml",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := patRulePath.MatchString(tt.path)
			if result != tt.expected {
				t.Errorf("path %q: expected %v, got %v", tt.path, tt.expected, result)
			}
		})
	}
}
