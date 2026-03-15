package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"kv-cache/internal/persistence"
	storage "kv-cache/internal/storage"
	"kv-cache/internal/storage/types"
)

// CLI 命令行交互结构
type CLI struct {
	store       *storage.MemoryStore
	persist     *persistence.Persistence
	reader      *bufio.Reader
	writer      io.Writer
	interactive bool
	loading     bool
}

// NewCLI 创建 CLI 实例
func NewCLI(s *storage.MemoryStore, p *persistence.Persistence, reader io.Reader, writer io.Writer, interactive bool) *CLI {
	return &CLI{
		store:       s,
		persist:     p,
		reader:      bufio.NewReader(reader),
		writer:      writer,
		interactive: interactive,
	}
}

// UpdatePersist 更新 persist 引用
func (c *CLI) UpdatePersist(p *persistence.Persistence) {
	c.persist = p
}

// LoadData 从 AOF 加载数据
func (c *CLI) LoadData() error {
	if c.persist == nil {
		return nil
	}
	c.loading = true
	defer func() { c.loading = false }()
	return c.persist.Load(func(cmd string) error {
		// 加载时不记录到 AOF（避免重复记录）
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return nil
		}
		return c.executeSilent(strings.ToUpper(parts[0]), parts)
	})
}

// Run 启动命令行循环
func (c *CLI) Run() error {
	for {
		if c.interactive {
			fmt.Fprint(c.writer, "kv-cache> ")
		}

		line, err := c.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := parseArgs(line)
		if len(parts) == 0 {
			continue
		}
		cmd := strings.ToUpper(parts[0])

		// 记录到 AOF（如果是修改命令）
		if c.persist != nil && isWriteCommand(cmd) {
			c.persist.Append(line)
		}

		if err := c.execute(cmd, parts); err != nil {
			if err == ErrQuit {
				return nil
			}
			fmt.Fprintf(c.writer, "(error) ERR %v\n", err)
		}
	}
}

var ErrQuit = fmt.Errorf("quit")

// parseArgs 解析命令行参数，支持引号包裹的字符串
func parseArgs(line string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if ch == '"' {
			if inQuotes {
				// 结束引号
				args = append(args, current.String())
				current.Reset()
				inQuotes = false
			} else {
				// 开始引号
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
				inQuotes = true
			}
		} else if ch == ' ' && !inQuotes {
			// 空格分隔（不在引号内）
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}

	// 处理最后一个参数
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// isWriteCommand 判断是否为写命令（需要持久化）
func isWriteCommand(cmd string) bool {
	writeCmds := map[string]bool{
		"SET": true, "DEL": true, "EXPIRE": true,
		"HSET": true, "HDEL": true,
		"LPUSH": true, "RPUSH": true, "LPOP": true, "RPOP": true,
		"SADD": true, "SREM": true,
		"ZADD": true, "ZREM": true,
	}
	return writeCmds[cmd]
}

// execute 执行单个命令
func (c *CLI) execute(cmd string, parts []string) error {
	switch cmd {
	// 系统命令
	case "QUIT", "EXIT":
		fmt.Fprintln(c.writer, "Bye!")
		return ErrQuit
	case "CLEAR":
		return c.handleClear(parts)
	case "HELP":
		c.printHelp()

	// String 命令
	case "SET":
		return c.handleSet(parts)
	case "GET":
		return c.handleGet(parts)

	// 通用键命令
	case "DEL":
		return c.handleDel(parts)
	case "KEYS":
		return c.handleKeys(parts)
	case "FLUSHDB":
		return c.handleFlushDB()
	case "EXPIRE":
		return c.handleExpire(parts)
	case "TTL":
		return c.handleTTL(parts)

	// Hash 命令
	case "HSET":
		return c.handleHSet(parts)
	case "HGET":
		return c.handleHGet(parts)
	case "HGETALL":
		return c.handleHGetAll(parts)
	case "HDEL":
		return c.handleHDel(parts)

	// List 命令
	case "LPUSH":
		return c.handleLPush(parts)
	case "RPUSH":
		return c.handleRPush(parts)
	case "LPOP":
		return c.handleLPop(parts)
	case "RPOP":
		return c.handleRPop(parts)
	case "LRANGE":
		return c.handleLRange(parts)
	case "LLEN":
		return c.handleLLen(parts)

	// Set 命令
	case "SADD":
		return c.handleSAdd(parts)
	case "SMEMBERS":
		return c.handleSMembers(parts)
	case "SCARD":
		return c.handleSCard(parts)

	// ZSet 命令
	case "ZADD":
		return c.handleZAdd(parts)
	case "ZRANGE":
		return c.handleZRange(parts)
	case "ZCARD":
		return c.handleZCard(parts)

	default:
		return fmt.Errorf("unknown command '%s'", cmd)
	}
	return nil
}

// executeSilent 静默执行命令（不输出到终端，不记录到 AOF）
func (c *CLI) executeSilent(cmd string, parts []string) error {
	switch cmd {
	case "SET":
		return c.handleSet(parts)
	case "HSET":
		return c.handleHSet(parts)
	case "RPUSH":
		return c.handleRPush(parts)
	case "LPUSH":
		return c.handleLPush(parts)
	case "SADD":
		return c.handleSAdd(parts)
	case "ZADD":
		return c.handleZAdd(parts)
	case "EXPIRE":
		return c.handleExpire(parts)
	}
	return nil
}

func (c *CLI) printHelp() {
	help := `
支持的命令:

通用键命令:
  SET key value [ttl]         - 设置键值 (ttl单位为秒)
  GET key                     - 获取键值
  DEL key [key...]            - 删除键
  KEYS [pattern]              - 列出所有键 (pattern支持*通配)
  FLUSHDB                     - 清空当前数据库
  EXPIRE key ttl              - 设置过期时间(秒)
  TTL key                     - 查看剩余过期时间

Hash 命令:
  HSET key field value        - 设置字段
  HGET key field              - 获取字段
  HGETALL key                 - 获取所有字段
  HDEL key field [field...]   - 删除字段

List 命令:
  LPUSH key value [value...]  - 从左侧插入
  RPUSH key value [value...]  - 从右侧插入
  LPOP key                    - 从左侧弹出
  RPOP key                    - 从右侧弹出
  LRANGE key start stop       - 获取范围元素
  LLEN key                    - 获取列表长度

Set 命令:
  SADD key member [member...] - 添加成员
  SMEMBERS key                - 获取所有成员
  SCARD key                   - 获取成员数量

ZSet 命令:
  ZADD key score member       - 添加成员
  ZRANGE key start stop       - 获取排名范围成员
  ZCARD key                   - 获取成员数量

其他:
  clear                       - 清屏
  help                        - 显示帮助
  quit / exit                 - 退出程序
`
	fmt.Fprint(c.writer, help)
}

// Export 导出所有数据为命令列表（用于 AOF Rewrite）
func (c *CLI) Export() []string {
	var commands []string

	keys := c.store.Keys()
	for _, key := range keys {
		val, exists := c.store.Get(key)
		if !exists {
			continue
		}

		switch v := val.Data.(type) {
		case string:
			cmd := fmt.Sprintf("SET %s %s", key, v)
			if val.ExpireAt != nil {
				ttl := int(time.Until(*val.ExpireAt).Seconds())
				if ttl > 0 {
					cmd += fmt.Sprintf(" %d", ttl)
				}
			}
			commands = append(commands, cmd)

		case map[string]string: // Hash
			for field, value := range v {
				commands = append(commands, fmt.Sprintf("HSET %s %s %s", key, field, value))
			}

		case []string: // List
			if len(v) > 0 {
				commands = append(commands, fmt.Sprintf("RPUSH %s %s", key, strings.Join(v, " ")))
			}

		case types.Set: // Set
			members := v.SMembers()
			if len(members) > 0 {
				commands = append(commands, fmt.Sprintf("SADD %s %s", key, strings.Join(members, " ")))
			}

		case *types.ZSet: // ZSet
			members := v.ZRange(0, -1)
			for _, m := range members {
				commands = append(commands, fmt.Sprintf("ZADD %s %f %s", key, m.Score, m.Member))
			}
		}
	}

	return commands
}
