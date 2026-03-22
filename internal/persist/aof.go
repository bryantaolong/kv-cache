package persist

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Persistence 处理 AOF 持久化
type Persistence struct {
	filepath    string
	file        *os.File
	writer      *bufio.Writer
	mutex       sync.Mutex
	running     bool
	stopChan    chan struct{}
	rewriteSize int64   // 触发自动 Rewrite 的文件大小阈值（字节）
	syncer      *Syncer // 同步策略管理器
}

// NewPersistence 创建持久化实例
func NewPersistence(dataDir string) (*Persistence, error) {
	// 确保目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filepath := filepath.Join(dataDir, "appendonly.aof")

	// 以追加模式打开文件
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open AOF file: %w", err)
	}

	p := &Persistence{
		filepath: filepath,
		file:     file,
		writer:   bufio.NewWriter(file),
		running:  false,
		stopChan: make(chan struct{}),
	}

	// 初始化同步管理器
	p.syncer = NewSyncer(func() error {
		return p.file.Sync()
	})
	p.syncer.Start()

	return p, nil
}

// Append 追加命令到 AOF 文件
func (p *Persistence) Append(command string) error {
	if p == nil {
		return nil
	}

	p.mutex.Lock()

	// 写入命令（带换行符）
	if _, err := p.writer.WriteString(command + "\n"); err != nil {
		p.mutex.Unlock()
		return fmt.Errorf("failed to write to AOF: %w", err)
	}

	// 根据策略决定是否 flush 到 OS 缓冲区
	if p.syncer.NeedFlush() {
		if err := p.writer.Flush(); err != nil {
			p.mutex.Unlock()
			return fmt.Errorf("failed to flush AOF: %w", err)
		}
	}

	// 根据策略决定是否立即 sync
	needSync := p.syncer.AfterWrite()
	if needSync {
		if err := p.file.Sync(); err != nil {
			p.mutex.Unlock()
			return fmt.Errorf("failed to sync AOF: %w", err)
		}
	}

	p.mutex.Unlock()
	return nil
}

// SetSyncPolicy 设置 AOF 同步策略
func (p *Persistence) SetSyncPolicy(policy SyncPolicy) {
	if p == nil || p.syncer == nil {
		return
	}
	p.syncer.SetPolicy(policy)
}

// GetSyncPolicy 获取当前 AOF 同步策略
func (p *Persistence) GetSyncPolicy() string {
	if p == nil || p.syncer == nil {
		return "unknown"
	}
	return p.syncer.GetPolicy().String()
}

// Load 加载 AOF 文件并执行
func (p *Persistence) Load(executor func(string) error) error {
	if p == nil {
		return nil
	}

	// 检查文件是否存在
	if _, err := os.Stat(p.filepath); os.IsNotExist(err) {
		// 文件不存在，这是正常的（首次启动）
		return nil
	}

	// 启动动画 goroutine - 每秒输出一个句号，6个后清空
	stopAnimation := make(chan bool)
	go func() {
		count := 0
		for {
			select {
			case <-stopAnimation:
				return
			case <-time.After(time.Second):
				fmt.Print(".")
				count++
				if count >= 6 {
					fmt.Print("\r      \r") // 清行并回到行首
					count = 0
				}
			}
		}
	}()

	// 打开文件只读
	file, err := os.Open(p.filepath)
	if err != nil {
		stopAnimation <- true
		return fmt.Errorf("failed to open AOF for loading: %w", err)
	}
	defer file.Close()

	// 逐行读取并执行
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		command := scanner.Text()
		if command == "" {
			continue
		}

		if err := executor(command); err != nil {
			// 加载时遇到错误，打印警告但继续
			fmt.Fprintf(os.Stderr, "Warning: failed to execute command [%s]: %v\n", command, err)
		}
	}

	// 停止动画
	stopAnimation <- true
	// 清除残留的句号
	fmt.Print("\r      \r")

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read AOF: %w", err)
	}

	return nil
}

// Close 关闭持久化
func (p *Persistence) Close() error {
	if p == nil || p.file == nil {
		return nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 停止后台 sync 协程
	if p.syncer != nil {
		p.syncer.Stop()
	}

	// 刷新缓冲区
	if err := p.writer.Flush(); err != nil {
		return err
	}

	// 关闭文件
	return p.file.Close()
}

// Rewrite 重写 AOF 文件（压缩）
// 通过导出当前内存状态，生成最小命令集
// 暂时私有化，不对外暴露
func (p *Persistence) rewrite(exportFunc func() []string) error {
	if p == nil {
		return nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 获取当前数据导出
	commands := exportFunc()
	if len(commands) == 0 {
		// 没有数据，清空 AOF
		p.file.Close()
		os.Remove(p.filepath)
		file, err := os.OpenFile(p.filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		p.file = file
		p.writer = bufio.NewWriter(file)
		return nil
	}

	// 创建临时文件
	tmpPath := p.filepath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// 写入所有命令
	writer := bufio.NewWriter(tmpFile)
	for _, cmd := range commands {
		if _, err := writer.WriteString(cmd + "\n"); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}

	tmpFile.Close()

	// 关闭原文件
	p.file.Close()

	// 原子替换
	if err := os.Rename(tmpPath, p.filepath); err != nil {
		// 恢复失败，尝试重新打开原文件
		p.file, _ = os.OpenFile(p.filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		return fmt.Errorf("failed to replace AOF file: %w", err)
	}

	// 重新打开新文件
	file, err := os.OpenFile(p.filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to reopen AOF file: %w", err)
	}
	p.file = file
	p.writer = bufio.NewWriter(file)

	return nil
}

// GetSize 获取 AOF 文件大小
func (p *Persistence) GetSize() int64 {
	if p == nil {
		return 0
	}
	info, err := os.Stat(p.filepath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// Reader 返回 AOF 文件读取器（用于外部读取）
// 暂时不使用
func (p *Persistence) reader() (io.ReadCloser, error) {
	if p == nil {
		return nil, fmt.Errorf("persistence not initialized")
	}
	return os.Open(p.filepath)
}

// StartAutoRewrite 启动自动 Rewrite 后台 Goroutine
// rewriteSize: 触发 Rewrite 的文件大小阈值（字节），0 表示不自动触发
// interval: 检查间隔
// exportFunc: 导出数据为命令列表的函数
func (p *Persistence) StartAutoRewrite(rewriteSize int64, interval time.Duration, exportFunc func() []string) {
	if p == nil || rewriteSize <= 0 {
		return
	}

	p.mutex.Lock()
	if p.running {
		p.mutex.Unlock()
		return
	}
	p.running = true
	p.rewriteSize = rewriteSize
	p.mutex.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-p.stopChan:
				p.mutex.Lock()
				p.running = false
				p.mutex.Unlock()
				return
			case <-ticker.C:
				p.checkAndRewrite(exportFunc)
			}
		}
	}()
}

// StopAutoRewrite 停止自动 Rewrite
func (p *Persistence) StopAutoRewrite() {
	if p == nil {
		return
	}
	select {
	case p.stopChan <- struct{}{}:
	default:
	}
}

// checkAndRewrite 检查是否需要触发 Rewrite
func (p *Persistence) checkAndRewrite(exportFunc func() []string) {
	size := p.GetSize()
	if size >= p.rewriteSize {
		fmt.Printf("* AOF file size (%d bytes) >= threshold (%d bytes), triggering rewrite...\n", size, p.rewriteSize)
		if err := p.rewrite(exportFunc); err != nil {
			fmt.Fprintf(os.Stderr, "Auto rewrite failed: %v\n", err)
		} else {
			fmt.Printf("* Rewrite completed, new file size: %d bytes\n", p.GetSize())
		}
	}
}
