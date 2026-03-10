package path

import (
	"os"
	"path/filepath"
	"testing"
)

// 使用通配符匹配路径中符合的所有文件
func TestFilepathGlob(t *testing.T) {
	// 创建临时测试目录结构
	tmpDir := t.TempDir()

	// 创建测试文件和目录
	testFiles := []string{
		"test1.yml",
		"test2.yaml",
		"test3.txt",
		"config.yml",
		"subdir/test4.yml",
		"subdir/test6.yml",
		"deep/nested/test7.yml",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tmpDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	tests := []struct {
		name         string
		pattern      string
		wantCount    int
		wantContains []string
		wantErr      bool
	}{
		{
			name:      "匹配所有 yml 文件 - 单层目录",
			pattern:   filepath.Join(tmpDir, "*.yml"),
			wantCount: 2,
			wantContains: []string{
				filepath.Join(tmpDir, "test1.yml"),
				filepath.Join(tmpDir, "config.yml"),
			},
			wantErr: false,
		},
		{
			name:      "匹配所有 yaml 文件",
			pattern:   filepath.Join(tmpDir, "*.yaml"),
			wantCount: 1,
			wantContains: []string{
				filepath.Join(tmpDir, "test2.yaml"),
			},
			wantErr: false,
		},
		{
			name:      "匹配所有 txt 文件",
			pattern:   filepath.Join(tmpDir, "*.txt"),
			wantCount: 1,
			wantContains: []string{
				filepath.Join(tmpDir, "test3.txt"),
			},
			wantErr: false,
		},
		{
			name:      "匹配子目录中的 yml 文件",
			pattern:   filepath.Join(tmpDir, "subdir", "*.yml"),
			wantCount: 2,
			wantContains: []string{
				filepath.Join(tmpDir, "subdir", "test4.yml"),
				filepath.Join(tmpDir, "subdir", "test6.yml"),
			},
			wantErr: false,
		},
		{
			name:      "不存在的模式 - 无匹配",
			pattern:   filepath.Join(tmpDir, "*.json"),
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "无效模式 - 语法错误",
			pattern:   filepath.Join(tmpDir, "[unclosed"),
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "绝对路径匹配",
			pattern:   filepath.Join(tmpDir, "deep", "nested", "*.yml"),
			wantCount: 1,
			wantContains: []string{
				filepath.Join(tmpDir, "deep", "nested", "test7.yml"),
			},
			wantErr: false,
		},
		{
			name:      "匹配多个扩展名模式",
			pattern:   filepath.Join(tmpDir, "*.{yml,yaml}"),
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := filepath.Glob(tt.pattern)

			if (err != nil) != tt.wantErr {
				t.Errorf("filepath.Glob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(matches) != tt.wantCount {
				t.Errorf("filepath.Glob() got %d matches, want %d. Matches: %v", len(matches), tt.wantCount, matches)
			}

			if !tt.wantErr && tt.wantContains != nil {
				matchSet := make(map[string]bool)
				for _, m := range matches {
					matchSet[m] = true
				}

				for _, want := range tt.wantContains {
					if !matchSet[want] {
						t.Errorf("Expected match %q not found in results: %v", want, matches)
					}
				}
			}
		})
	}
}

func TestFilepathGlobPatternExamples(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{
		"a.go", "b.go", "c.txt",
		"test_1.go", "test_2.go", "test_10.go",
		"main.go",
	}

	for _, file := range files {
		os.WriteFile(filepath.Join(tmpDir, file), []byte(""), 0644)
	}

	tests := []struct {
		name      string
		pattern   string
		wantCount int
		desc      string
	}{
		{
			name:      "星号匹配任意字符",
			pattern:   filepath.Join(tmpDir, "*.go"),
			wantCount: 6,
			desc:      "* 匹配零个或多个字符",
		},
		{
			name:      "数字前缀匹配",
			pattern:   filepath.Join(tmpDir, "test_[0-9].go"),
			wantCount: 2,
			desc:      "匹配 test_1.go 和 test_2.go，但不匹配 test_10.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := filepath.Glob(tt.pattern)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(matches) != tt.wantCount {
				t.Errorf("%s: got %d matches, want %d. Pattern: %s, Matches: %v",
					tt.desc, len(matches), tt.wantCount, tt.pattern, matches)
			}
		})
	}
}
