package yaml

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		wantErr        bool
		wantBaseKind   string
		wantBaseName   string
		wantServerPort int
		wantDBMaxConns int
	}{
		{
			name: "成功加载完整配置",
			yamlContent: `
base_config:
  kind: Application
  name: my-app
server_config:
  kind: ServerConfig
  name: web-server
  port: 8080
  host: 0.0.0.0
database_config:
  kind: DatabaseConfig
  name: main-db
  url: postgres://localhost:5432/mydb?sslmode=disable
  max_conns: 100
`,
			wantErr:        false,
			wantBaseKind:   "Application",
			wantBaseName:   "my-app",
			wantServerPort: 8080,
			wantDBMaxConns: 100,
		},
		{
			name: "成功加载部分配置 - 使用默认值",
			yamlContent: `
base_config:
  kind: ""
  name: ""
server_config:
  port: 3000
  host: localhost
database_config:
  url: mysql://root@localhost/testdb
  max_conns: 50
`,
			wantErr:        false,
			wantBaseKind:   "ServerConfig",
			wantBaseName:   "default",
			wantServerPort: 3000,
			wantDBMaxConns: 50,
		},
		{
			name: "成功加载最小配置",
			yamlContent: `
base_config: {}
server_config: {}
database_config: {}
`,
			wantErr:        false,
			wantBaseKind:   "ServerConfig",
			wantBaseName:   "default",
			wantServerPort: 0,
			wantDBMaxConns: 0,
		},
		{
			name: "成功加载仅服务器配置",
			yamlContent: `
base_config:
  kind: ServerOnly
  name: server-instance
server_config:
  port: 9000
  host: 127.0.0.1
`,
			wantErr:        false,
			wantBaseKind:   "ServerOnly",
			wantBaseName:   "server-instance",
			wantServerPort: 9000,
			wantDBMaxConns: 0,
		},
		{
			name: "成功加载仅数据库配置",
			yamlContent: `
base_config:
  kind: DBOnly
  name: db-instance
database_config:
  url: postgres://user:pass@localhost/prod
  max_conns: 200
`,
			wantErr:        false,
			wantBaseKind:   "DBOnly",
			wantBaseName:   "db-instance",
			wantServerPort: 0,
			wantDBMaxConns: 200,
		},
		{
			name:        "失败 - 无效的 YAML 格式",
			yamlContent: `invalid: yaml: content: here`,
			wantErr:     true,
		},
		{
			name:        "空内容",
			yamlContent: "",
			wantErr:     false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig([]byte(tt.yamlContent))

			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && cfg == nil {
				t.Fatal("LoadConfig() returned nil config with no error")
			}

			if cfg != nil {
				if cfg.BaseConfig.Kind != tt.wantBaseKind {
					t.Errorf("BaseConfig.Kind = %q, want %q", cfg.BaseConfig.Kind, tt.wantBaseKind)
				}

				if cfg.BaseConfig.Name != tt.wantBaseName {
					t.Errorf("BaseConfig.Name = %q, want %q", cfg.BaseConfig.Name, tt.wantBaseName)
				}

				if cfg.ServerConfig.Port != tt.wantServerPort {
					t.Errorf("ServerConfig.Port = %d, want %d", cfg.ServerConfig.Port, tt.wantServerPort)
				}

				if cfg.DatabaseConfig.MaxConns != tt.wantDBMaxConns {
					t.Errorf("DatabaseConfig.MaxConns = %d, want %d", cfg.DatabaseConfig.MaxConns, tt.wantDBMaxConns)
				}
			}
		})
	}
}
