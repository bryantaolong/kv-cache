package command

import (
	"fmt"
	"strings"
	"time"

	ps "kv-cache/internal/persist"
	storage "kv-cache/internal/storage"
	"kv-cache/internal/storage/types"
)

// Executor 命令执行引擎，不涉及任何 I/O。
type Executor struct {
	store   *storage.MemoryStore
	persist *ps.Persistence
	loading bool
}

// NewExecutor 创建命令执行引擎。
func NewExecutor(s *storage.MemoryStore, p *ps.Persistence) *Executor {
	return &Executor{
		store:   s,
		persist: p,
	}
}

// SetPersist 设置持久化引用。
func (e *Executor) SetPersist(p *ps.Persistence) {
	e.persist = p
}

// SetLoading 设置是否为加载模式（加载时不记录 AOF）。
func (e *Executor) SetLoading(v bool) { e.loading = v }

// Result 命令执行结果。
type Result struct {
	Lines []string
	Quit  bool
}

// Execute 执行一条命令。
func (e *Executor) Execute(parts []string) (*Result, error) {
	if len(parts) == 0 {
		return &Result{}, nil
	}
	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "QUIT", "EXIT":
		return &Result{Quit: true}, nil
	case "CLEAR":
		return &Result{}, nil
	case "HELP":
		return &Result{Lines: []string{helpText}}, nil

	case "SET":
		return e.handleSet(parts)
	case "GET":
		return e.handleGet(parts)
	case "DEL":
		return e.handleDel(parts)
	case "KEYS":
		return e.handleKeys(parts)
	case "FLUSHDB":
		return e.handleFlushDB()
	case "EXPIRE":
		return e.handleExpire(parts)
	case "TTL":
		return e.handleTTL(parts)
	case "HSET":
		return e.handleHSet(parts)
	case "HGET":
		return e.handleHGet(parts)
	case "HGETALL":
		return e.handleHGetAll(parts)
	case "HDEL":
		return e.handleHDel(parts)
	case "LPUSH":
		return e.handleLPush(parts)
	case "RPUSH":
		return e.handleRPush(parts)
	case "LPOP":
		return e.handleLPop(parts)
	case "RPOP":
		return e.handleRPop(parts)
	case "LRANGE":
		return e.handleLRange(parts)
	case "LLEN":
		return e.handleLLen(parts)
	case "SADD":
		return e.handleSAdd(parts)
	case "SMEMBERS":
		return e.handleSMembers(parts)
	case "SCARD":
		return e.handleSCard(parts)
	case "ZADD":
		return e.handleZAdd(parts)
	case "ZRANGE":
		return e.handleZRange(parts)
	case "ZCARD":
		return e.handleZCard(parts)
	default:
		return nil, fmt.Errorf("unknown command '%s'", cmd)
	}
}

// ExecuteSilent 静默执行命令（不记录 AOF）。
func (e *Executor) ExecuteSilent(parts []string) error {
	if len(parts) == 0 {
		return nil
	}
	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "SET":
		_, err := e.handleSet(parts)
		return err
	case "HSET":
		_, err := e.handleHSet(parts)
		return err
	case "RPUSH":
		_, err := e.handleRPush(parts)
		return err
	case "LPUSH":
		_, err := e.handleLPush(parts)
		return err
	case "SADD":
		_, err := e.handleSAdd(parts)
		return err
	case "ZADD":
		_, err := e.handleZAdd(parts)
		return err
	case "EXPIRE":
		_, err := e.handleExpire(parts)
		return err
	}
	return nil
}

// Export 导出所有数据为命令列表（用于 AOF Rewrite）。
func (e *Executor) Export() []string {
	var commands []string

	keys := e.store.Keys()
	for _, key := range keys {
		val, exists := e.store.Get(key)
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

func (e *Executor) appendPersist(cmd string) error {
	if e.persist != nil && !e.loading {
		return e.persist.Append(cmd)
	}
	return nil
}

const helpText = `
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
