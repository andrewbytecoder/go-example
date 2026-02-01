package config

import (
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type Config struct {
}

var (
	// DefaultConfig 这里配置默认值，可以确保只有配置之后的值才能被替换掉
	DefaultConfig = Config{}
)

// LoadFile parses and validates the given YAML file into a read-only Config
// @ref prometheus
func LoadFile(filename string, logger *zap.Logger) (*Config, error) {
	//	读取文件配置
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	// 进行严格序列化，只要配置了，但是在结构体里面没有定义直接保存
	err = yaml.UnmarshalStrict(content, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, err
}
