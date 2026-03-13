package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"kv-cache/internal/cli"
	"kv-cache/internal/persistence"
	storage "kv-cache/internal/storage"
)

// Config 配置结构
type Config struct {
	// 服务器配置
	Address string `yaml:"address"` // 监听地址

	// 数据目录
	DataDir string `yaml:"data-dir"` // 数据目录路径

	// 持久化配置
	NoPersist   bool  `yaml:"no-persist"`    // 是否禁用持久化
	RewriteSize int64 `yaml:"rewrite-size"` // AOF 自动 Rewrite 触发阈值（字节）

	// 内存配置
	MaxMemory    int64  `yaml:"maxmemory"`     // 最大内存限制（字节）
	EvictPolicy  string `yaml:"eviction-policy"` // 淘汰策略
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	// 默认配置
	cfg := &Config{
		Address:     ":6379",
		DataDir:     "./data",
		NoPersist:   false,
		RewriteSize: 64 * 1024 * 1024, // 64MB
		MaxMemory:   0,
		EvictPolicy: "noeviction",
	}

	// 如果配置文件不存在，使用默认配置
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析 YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// DefaultConfigPath 默认配置文件路径
const DefaultConfigPath = "./config.yaml"

func main() {
	// 命令行参数（优先级高于配置文件）
	configPath := flag.String("config", DefaultConfigPath, "配置文件路径")
	dataDir := flag.String("data", "", "数据目录路径")
	noPersist := flag.Bool("no-persist", false, "禁用持久化")
	rewriteSize := flag.Int64("rewrite-size", 0, "AOF 自动 Rewrite 触发阈值（字节），0 表示禁用")
	maxMemory := flag.Int64("maxmemory", 0, "最大内存限制（字节），0 表示不限制")
	evictPolicy := flag.String("maxmemory-policy", "", "淘汰策略: noeviction, allkeys-lru, volatile-lru, allkeys-random, volatile-random")
	flag.Parse()

	// 加载配置文件
	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 命令行参数覆盖配置文件
	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *noPersist {
		cfg.NoPersist = true
	}
	if *rewriteSize > 0 {
		cfg.RewriteSize = *rewriteSize
	}
	if *maxMemory > 0 {
		cfg.MaxMemory = *maxMemory
	}
	if *evictPolicy != "" {
		cfg.EvictPolicy = *evictPolicy
	}

	// 创建存储引擎
	s := storage.NewMemoryStore()

	// 设置内存限制和淘汰策略
	if cfg.MaxMemory > 0 {
		s.SetMaxMemory(cfg.MaxMemory)
		fmt.Printf("* Maxmemory set to %d bytes\n", cfg.MaxMemory)
	}

	// 解析淘汰策略
	switch cfg.EvictPolicy {
	case "allkeys-lru":
		s.SetEvictPolicy(storage.EvictAllKeysLRU)
	case "volatile-lru":
		s.SetEvictPolicy(storage.EvictVolatileLRU)
	case "allkeys-random":
		s.SetEvictPolicy(storage.EvictAllKeysRandom)
	case "volatile-random":
		s.SetEvictPolicy(storage.EvictVolatileRandom)
	default:
		s.SetEvictPolicy(storage.EvictNoeviction)
	}
	fmt.Printf("* Eviction policy: %s\n", s.GetEvictPolicy())

	// 启动后台 GC（每分钟清理一次过期键）
	s.StartGC(time.Minute)

	// 创建 CLI
	c := cli.NewCLI(s, nil, os.Stdin, os.Stdout, true)

	// 创建持久化模块
	var persist *persistence.Persistence
	if !cfg.NoPersist {
		persist, err = persistence.NewPersistence(cfg.DataDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize persistence: %v\n", err)
			os.Exit(1)
		}
		defer persist.Close()

		// 更新 CLI 的 persist 引用
		c.UpdatePersist(persist)

		// 启动自动 AOF Rewrite（每分钟检查一次）
		if cfg.RewriteSize > 0 {
			persist.StartAutoRewrite(cfg.RewriteSize, time.Minute, func() []string {
				return c.Export()
			})
		}
	}

	// 加载已有数据
	if persist != nil {
		if err := c.LoadData(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load data: %v\n", err)
		}
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n* Saving data...")
		if persist != nil {
			persist.Close()
			persist.StopAutoRewrite()
		}
		s.StopGC()
		fmt.Println("* Bye!")
		os.Exit(0)
	}()

	// 运行 CLI
	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	if persist != nil {
		persist.Close()
		persist.StopAutoRewrite()
	}
	s.StopGC()
}
