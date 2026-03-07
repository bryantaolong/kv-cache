package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"kv-cache/internal/cli"
	"kv-cache/internal/persistence"
	storage "kv-cache/internal/storage"
)

func main() {
	// 命令行参数
	dataDir := flag.String("data", "./data", "数据目录路径")
	noPersist := flag.Bool("no-persist", false, "禁用持久化")
	flag.Parse()

	// 创建存储引擎
	s := storage.NewMemoryStore()

	// 创建持久化模块
	var persist *persistence.Persistence
	if !*noPersist {
		var err error
		persist, err = persistence.NewPersistence(*dataDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize persistence: %v\n", err)
			os.Exit(1)
		}
		defer persist.Close()
	}

	// 创建 CLI
	c := cli.NewCLI(s, persist, os.Stdin, os.Stdout, true)

	// 加载已有数据
	if persist != nil {
		if err := c.LoadData(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load data: %v\n", err)
			// 不退出，继续运行
		}
	}

	// 设置信号处理（优雅关闭）
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n* Saving data...")
		if persist != nil {
			if err := persist.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing persistence: %v\n", err)
			}
		}
		fmt.Println("* Bye!")
		os.Exit(0)
	}()

	// 运行 CLI
	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 正常退出时关闭持久化
	if persist != nil {
		fmt.Println("持久化数据中...")
		persist.Close()
	}
}
