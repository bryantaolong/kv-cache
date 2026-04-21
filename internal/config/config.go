package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	// 服务器配置
	Address string `mapstructure:"address"`

	// 数据目录
	DataDir string `mapstructure:"data-dir"`

	// 持久化配置
	NoPersist        bool   `mapstructure:"no-persist"`
	RewriteSize      int64  `mapstructure:"rewrite-size"`
	AppendOnlyPolicy string `mapstructure:"append-only-policy"` // AOF 同步策略: always, everysec, no

	// 内存配置
	MaxMemory      int64  `mapstructure:"max-memory"`
	EvictionPolicy string `mapstructure:"eviction-policy"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Address:          ":6379",
		DataDir:          "./data",
		NoPersist:        false,
		RewriteSize:      64 * 1024 * 1024, // 64MB
		AppendOnlyPolicy: "everysec",       // 默认每秒同步，平衡性能与安全
		MaxMemory:        0,
		EvictionPolicy:   "lru",
	}
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	// 验证淘汰策略
	validPolicies := []string{"no-eviction", "lru", "random"}
	found := false
	for _, policy := range validPolicies {
		if strings.ToLower(c.EvictionPolicy) == policy {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid eviction policy: %s", c.EvictionPolicy)
	}

	// 验证 AOF 同步策略
	validSyncPolicies := []string{"always", "everysec", "no"}
	found = false
	for _, policy := range validSyncPolicies {
		if strings.ToLower(c.AppendOnlyPolicy) == policy {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid append-only policy: %s", c.AppendOnlyPolicy)
	}

	return nil
}

// Loader 配置加载器
type Loader struct {
	v *viper.Viper
}

// NewLoader 创建新的配置加载器
func NewLoader() *Loader {
	v := viper.New()

	// 从 DefaultConfig 同步默认值到 viper
	defaults := DefaultConfig()
	v.SetDefault("address", defaults.Address)
	v.SetDefault("data-dir", defaults.DataDir)
	v.SetDefault("no-persist", defaults.NoPersist)
	v.SetDefault("rewrite-size", defaults.RewriteSize)
	v.SetDefault("append-only-policy", defaults.AppendOnlyPolicy)
	v.SetDefault("max-memory", defaults.MaxMemory)
	v.SetDefault("eviction-policy", defaults.EvictionPolicy)

	// 支持环境变量
	v.SetEnvPrefix("KVCACHE")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	return &Loader{v: v}
}

// SetConfigFile 设置配置文件路径
func (l *Loader) SetConfigFile(path string) {
	l.v.SetConfigFile(path)
}

// SetConfigType 设置配置文件类型
func (l *Loader) SetConfigType(configType string) {
	l.v.SetConfigType(configType)
}

// Load 加载配置
func (l *Loader) Load() (*Config, error) {
	// 创建空结构体，viper 会填充默认值（通过 SetDefault 设置的）
	cfg := &Config{}

	// 尝试读取配置文件（如果设置了）
	if l.v.ConfigFileUsed() != "" {
		if err := l.v.ReadInConfig(); err != nil {
			// 配置文件不存在不是错误，使用默认值
			// 检查是否是文件不存在的错误（包括 viper.ConfigFileNotFoundError 和系统错误）
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				// 检查是否是系统文件不存在的错误
				if !os.IsNotExist(err) {
					return nil, fmt.Errorf("failed to read config file: %w", err)
				}
			}
		}
	}

	// 将配置解析到结构体（会自动应用 SetDefault 设置的默认值）
	if err := l.v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// BindFlag 绑定命令行参数
func (l *Loader) BindFlag(key string, value interface{}) {
	l.v.Set(key, value)
}

// GetViper 获取 viper 实例（用于高级用法）
func (l *Loader) GetViper() *viper.Viper {
	return l.v
}

// LoadFromFile 从指定文件加载配置的便捷函数
// 目前仅在测试中使用
func LoadFromFile(path string) (*Config, error) {
	loader := NewLoader()
	loader.SetConfigFile(path)
	return loader.Load()
}
