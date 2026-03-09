package yaml

import "gopkg.in/yaml.v3"

// ---------------------------------------------------------
// 场景演示：根据 "kind" 字段动态解析不同的配置结构
// ---------------------------------------------------------

// BaseConfig 是所有配置的基座，包含公共字段
type BaseConfig struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}

// ServerConfig 具体配置结构 A
type ServerConfig struct {
	ExternalLabels Labels `yaml:"external_labels"`
	Port           int    `yaml:"port"`
	Host           string `yaml:"host"`
}

// DatabaseConfig 具体配置结构 B
type DatabaseConfig struct {
	URL      string `yaml:"url"`
	MaxConns int    `yaml:"max_conns"`
}

type Config struct {
	BaseConfig     BaseConfig     `yaml:"base_config"`
	ServerConfig   ServerConfig   `yaml:"server_config"`
	DatabaseConfig DatabaseConfig `yaml:"database_config"`
}

// UnmarshalYAML 接口
// 任意类型实现了 UnmarshalYAML 接口，那么 YAML 解析器会自动调用这个方法，而不是使用默认的解析方法进行解析
func (c *BaseConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {

	// 单独解析BaseConfig
	type plain BaseConfig
	bc := &BaseConfig{}
	if err := unmarshal((*plain)(bc)); err != nil {
		return err
	}

	// 这里进行自己的操作，在进行全局解析的时候会自动调用这个方法
	if bc.Kind == "" {
		bc.Kind = "ServerConfig"
	}

	if bc.Name == "" {
		bc.Name = "default"
	}

	*c = *bc
	return nil
}

func LoadConfig(content []byte) (*Config, error) {
	c := &Config{}

	if err := yaml.Unmarshal(content, c); err != nil {
		return nil, err
	}

	return c, nil
}
