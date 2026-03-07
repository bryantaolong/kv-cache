package persistence

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
	filepath string
	file     *os.File
	writer   *bufio.Writer
	mutex    sync.Mutex
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

	return &Persistence{
		filepath: filepath,
		file:     file,
		writer:   bufio.NewWriter(file),
	}, nil
}

// Append 追加命令到 AOF 文件
func (p *Persistence) Append(command string) error {
	if p == nil {
		return nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 写入命令（带换行符）
	if _, err := p.writer.WriteString(command + "\n"); err != nil {
		return fmt.Errorf("failed to write to AOF: %w", err)
	}

	// 立即刷新（可选：可以改为定期刷新以提高性能）
	if err := p.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush AOF: %w", err)
	}

	// 同步到磁盘（可选：可以改为每秒同步或让 OS 决定）
	// 目前为了数据安全，每次写入都 sync
	// 性能要求高时可以改为 p.file.Sync() 每秒调用一次
	if err := p.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync AOF: %w", err)
	}

	return nil
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

	// 刷新缓冲区
	if err := p.writer.Flush(); err != nil {
		return err
	}

	// 关闭文件
	return p.file.Close()
}

// Rewrite 重写 AOF 文件（压缩）
// 通过导出当前内存状态，生成最小命令集
func (p *Persistence) Rewrite(exportFunc func() []string) error {
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
func (p *Persistence) Reader() (io.ReadCloser, error) {
	if p == nil {
		return nil, fmt.Errorf("persistence not initialized")
	}
	return os.Open(p.filepath)
}
