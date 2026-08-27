package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config 对应 application.yml 顶层结构
type Config struct {
	Server     ServerConfig    `mapstructure:"server"`
	AgentModel BaseModelConfig `mapstructure:"agent-model"`
	PlanModel  BaseModelConfig `mapstructure:"plan-model"`
	File       FileConfig      `mapstructure:"file"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// BaseModelConfig 对应 Java ModelConfig.BaseModelConfig（@Data）
type BaseModelConfig struct {
	BaseURL         string `mapstructure:"base-url"`
	APIKey          string `mapstructure:"api-key"`
	ModelName       string `mapstructure:"model-name"`
	CompletionsPath string `mapstructure:"completions-path"`
	Stream          bool   `mapstructure:"stream"`
}

type FileConfig struct {
	Base   string `mapstructure:"base"`
	Domain string `mapstructure:"domain"`
}

// Load 读取并解析 YAML；将 ${VAR} 展开为环境变量（复刻 Spring 的占位符行为）。
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件: %w", err)
	}
	expanded := os.ExpandEnv(string(raw))

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(expanded)); err != nil {
		return nil, fmt.Errorf("解析配置文件: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("映射配置: %w", err)
	}
	return &cfg, nil
}
