package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// handleClear 清屏命令，功能类似 Windows 的 cls 和 Unix 的 clear
func (c *CLI) handleClear(parts []string) error {
	if len(parts) != 1 {
		return fmt.Errorf("wrong number of arguments for 'clear' command")
	}

	return clearScreen()
}

// clearScreen 根据操作系统执行清屏操作
func clearScreen() error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		return cmd.Run()
	default:
		// Unix/Linux/Mac 使用 ANSI 转义序列清屏
		fmt.Print("\033[H\033[2J")
		return nil
	}
}
