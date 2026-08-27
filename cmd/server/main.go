package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"kv-cache/internal/command"
	"kv-cache/internal/config"
	persist "kv-cache/internal/persist"
	storage "kv-cache/internal/storage"
	"kv-cache/internal/transport"
)

const defaultAddr = ":27926"

func main() {
	configPath := flag.String("config", "./config.yaml", "配置文件路径")
	dataDir := flag.String("data", "", "数据目录路径")
	noPersist := flag.Bool("no-persist", false, "禁用持久化")
	rewriteSize := flag.Int64("rewrite-size", 0, "AOF 自动 Rewrite 触发阈值（字节），0 表示禁用")
	appendOnlyPolicy := flag.String("append-only-policy", "", "AOF 同步策略: always, everysec, no")
	maxMemory := flag.Int64("max-memory", 0, "最大内存限制（字节），0 表示不限制")
	evictionPolicy := flag.String("eviction-policy", "", "淘汰策略: no-eviction, lru, random")
	flag.Parse()

	loader := config.NewLoader()
	loader.SetConfigFile(*configPath)

	cfg, err := loader.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *noPersist {
		cfg.NoPersist = true
	}
	if *rewriteSize > 0 {
		cfg.RewriteSize = *rewriteSize
	}
	if *appendOnlyPolicy != "" {
		cfg.AppendOnlyPolicy = *appendOnlyPolicy
	}
	if *maxMemory > 0 {
		cfg.MaxMemory = *maxMemory
	}
	if *evictionPolicy != "" {
		cfg.EvictionPolicy = *evictionPolicy
	}

	s := storage.NewMemoryStore()

	if cfg.MaxMemory > 0 {
		s.SetMaxMemory(cfg.MaxMemory)
		log.Printf("* MaxMemory set to %d bytes", cfg.MaxMemory)
	}

	s.SetEvictionPolicy(storage.ParseEvictionPolicy(cfg.EvictionPolicy))
	log.Printf("* Eviction policy: %s", s.GetEvictionPolicy())

	s.StartGC(time.Minute)

	var ps *persist.Persistence
	if !cfg.NoPersist {
		ps, err = persist.NewPersistence(cfg.DataDir)
		if err != nil {
			log.Fatalf("Failed to initialize persistence: %v", err)
		}
		defer ps.Close()

		ps.SetSyncPolicy(persist.ParseSyncPolicy(cfg.AppendOnlyPolicy))
		log.Printf("* Append only policy: %s", ps.GetSyncPolicy())
	}

	executor := command.NewExecutor(s, ps)
	loadData(executor, ps)

	if cfg.RewriteSize > 0 && ps != nil {
		ps.StartAutoRewrite(cfg.RewriteSize, time.Minute, func() []string {
			return executor.Export()
		})
	}

	server := transport.NewServer(executor, defaultAddr)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n* Saving data...")
		if ps != nil {
			ps.Close()
			ps.StopAutoRewrite()
		}
		s.StopGC()
		server.Close()
		log.Println("* Bye!")
		os.Exit(0)
	}()

	select {}
}

func loadData(executor *command.Executor, ps *persist.Persistence) {
	if ps == nil {
		return
	}
	executor.SetLoading(true)
	defer executor.SetLoading(false)
	if err := ps.Load(func(cmd string) error {
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return nil
		}
		return executor.ExecuteSilent(parts)
	}); err != nil {
		log.Printf("Failed to load data: %v", err)
	}
}
