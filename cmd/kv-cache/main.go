package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kv-cache/internal/cli"
	"kv-cache/internal/config"
	persist "kv-cache/internal/persist"
	storage "kv-cache/internal/storage"
)

// DefaultConfigPath 默认配置文件路径
const DefaultConfigPath = "./config.yaml"

func main() {
	// 命令行参数（优先级高于配置文件）
	configPath := flag.String("config", DefaultConfigPath, "配置文件路径")
	dataDir := flag.String("data", "", "数据目录路径")
	noPersist := flag.Bool("no-persist", false, "禁用持久化")
	rewriteSize := flag.Int64("rewrite-size", 0, "AOF 自动 Rewrite 触发阈值（字节），0 表示禁用")
	maxMemory := flag.Int64("max-memory", 0, "最大内存限制（字节），0 表示不限制")
	evictionPolicy := flag.String("eviction-policy", "", "淘汰策略: noeviction, allkeys-lru, volatile-lru, allkeys-random, volatile-random")
	flag.Parse()

	// 创建配置加载器
	loader := config.NewLoader()
	loader.SetConfigFile(*configPath)

	// 加载配置
	cfg, err := loader.Load()
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
	if *evictionPolicy != "" {
		cfg.EvictionPolicy = *evictionPolicy
	}

	// 创建存储引擎
	s := storage.NewMemoryStore()

	// 设置内存限制和淘汰策略
	if cfg.MaxMemory > 0 {
		s.SetMaxMemory(cfg.MaxMemory)
		fmt.Printf("* MaxMemory set to %d bytes\n", cfg.MaxMemory)
	}

	// 解析淘汰策略
	switch cfg.EvictionPolicy {
	case "allkeys-lru":
		s.SetEvictionPolicy(storage.EvictAllKeysLRU)
	case "volatile-lru":
		s.SetEvictionPolicy(storage.EvictVolatileLRU)
	case "allkeys-random":
		s.SetEvictionPolicy(storage.EvictAllKeysRandom)
	case "volatile-random":
		s.SetEvictionPolicy(storage.EvictVolatileRandom)
	default:
		s.SetEvictionPolicy(storage.EvictNoeviction)
	}
	fmt.Printf("* Eviction policy: %s\n", s.GetEvictionPolicy())

	// 启动后台 GC（每分钟清理一次过期键）
	s.StartGC(time.Minute)

	// 创建 CLI
	c := cli.NewCLI(s, nil, os.Stdin, os.Stdout, true)

	// 创建持久化模块
	var ps *persist.Persistence
	if !cfg.NoPersist {
		ps, err = persist.NewPersistence(cfg.DataDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize persistence: %v\n", err)
			os.Exit(1)
		}
		defer ps.Close()

		// 更新 CLI 的 persist 引用
		c.UpdatePersist(ps)

		// 启动自动 AOF Rewrite（每分钟检查一次）
		if cfg.RewriteSize > 0 {
			ps.StartAutoRewrite(cfg.RewriteSize, time.Minute, func() []string {
				return c.Export()
			})
		}
	}

	// 加载已有数据
	if ps != nil {
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
		if ps != nil {
			ps.Close()
			ps.StopAutoRewrite()
		}
		s.StopGC()
		fmt.Println("* Bye!")
		os.Exit(0)
	}()

	// 运行 CLI
	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	if ps != nil {
		ps.Close()
		ps.StopAutoRewrite()
	}
	s.StopGC()
}
