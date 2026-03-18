# kv-cache

一个基于内存的键值存储系统，支持多种数据类型和持久化功能。

## 特性

- **多种数据类型**：String、Hash、List、Set、ZSet
- **过期时间**：支持键的 TTL 设置
- **持久化**：AOF (Append Only File) 持久化
- **类型安全**：完整的类型检查，防止操作错误类型
- **引号支持**：支持带空格的字符串值

## 快速开始

```bash
# 编译
go build -o kv-cache.exe ./cmd/kv-cache

# 运行
./kv-cache.exe

# 指定数据目录
./kv-cache.exe --data ./mydata

# 禁用持久化
./kv-cache.exe --no-persist

# 使用配置文件
./kv-cache.exe --config ./config.yaml
```

## 支持的命令

### 通用键命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `SET key value [ttl]` | 设置键值，可选过期时间(秒) | `SET name alice` |
| `GET key` | 获取键值 | `GET name` |
| `DEL key [key...]` | 删除一个或多个键 | `DEL name age` |
| `KEYS [pattern]` | 列出所有键，支持 `*` 通配符 | `KEYS *` `KEYS user:*` |
| `FLUSHDB` | 清空当前数据库 | `FLUSHDB` |
| `EXPIRE key ttl` | 设置过期时间(秒) | `EXPIRE name 60` |
| `TTL key` | 查看剩余过期时间 | `TTL name` |

### Hash 命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `HSET key field value` | 设置字段值 | `HSET user:1 name bob` |
| `HGET key field` | 获取字段值 | `HGET user:1 name` |
| `HGETALL key` | 获取所有字段 | `HGETALL user:1` |
| `HDEL key field [field...]` | 删除字段 | `HDEL user:1 age` |

### List 命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `LPUSH key value [value...]` | 从左侧插入 | `LPUSH mylist a b c` |
| `RPUSH key value [value...]` | 从右侧插入 | `RPUSH mylist x y z` |
| `LPOP key` | 从左侧弹出 | `LPOP mylist` |
| `RPOP key` | 从右侧弹出 | `RPOP mylist` |
| `LRANGE key start stop` | 获取范围元素 | `LRANGE mylist 0 -1` |
| `LLEN key` | 获取列表长度 | `LLEN mylist` |

### Set 命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `SADD key member [member...]` | 添加成员 | `SADD myset a b c` |
| `SREM key member [member...]` | 移除成员 | `SREM myset a` |
| `SMEMBERS key` | 获取所有成员 | `SMEMBERS myset` |
| `SCARD key` | 获取成员数量 | `SCARD myset` |

### ZSet 命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `ZADD key score member` | 添加成员 | `ZADD myzset 10 alice` |
| `ZREM key member [member...]` | 移除成员 | `ZREM myzset alice` |
| `ZRANGE key start stop` | 按排名范围获取 | `ZRANGE myzset 0 9` |
| `ZCARD key` | 获取成员数量 | `ZCARD myzset` |

### 系统命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `CLEAR` | 清空屏幕 | `CLEAR` |
| `HELP` | 显示帮助信息 | `HELP` |
| `EXIT` / `QUIT` | 退出程序 | `EXIT` |

## 项目结构

```
kv-cache/
├── cmd/kv-cache/           # 主程序入口
│   └── main.go
├── internal/
│   ├── cli/                # 命令行交互
│   │   └── cli.go
│   ├── config/             # 配置管理（基于 viper）
│   │   └── config.go
│   ├── persist/            # AOF 持久化
│   │   └── aof.go
│   └── storage/            # 存储引擎
│       ├── store.go        # MemoryStore 实现
│       ├── store_*.go      # 各数据类型命令
│       └── types/          # 数据类型定义
│           ├── value.go    # Value 和 DataType
│           ├── hash.go     # Hash 类型
│           ├── list.go     # List 类型
│           ├── set.go      # Set 类型
│           ├── zset.go     # ZSet 类型
│           └── string.go   # 字符串工具
├── config.yaml             # 配置文件示例
└── data/                   # 默认数据目录
    └── appendonly.aof      # AOF 持久化文件
```

## 使用示例

```bash
$ ./kv-cache.exe

kv-cache> SET name "alice jones"
OK

kv-cache> GET name
"alice jones"

kv-cache> HSET user:1 name bob age 20 city "new york"
(integer) 3

kv-cache> HGETALL user:1
1) "name"
2) "bob"
3) "age"
4) "20"
5) "city"
6) "new york"

kv-cache> LPUSH mylist "hello world" foo bar
(integer) 3

kv-cache> LRANGE mylist 0 -1
1) "bar"
2) "foo"
3) "hello world"

kv-cache> KEYS *
1) "name"
2) "user:1"
3) "mylist"

kv-cache> FLUSHDB
OK

kv-cache> KEYS *
(empty array)

kv-cache> HELP
支持的命令:

通用键命令:
  SET key value [ttl]         - 设置键值 (ttl单位为秒)
  ...
kv-cache> CLEAR

kv-cache> EXIT
Bye!
```

## 引号使用说明

支持双引号包裹包含空格的字符串：

```
# 正确
SET msg "hello world"
HSET user:1 name "John Doe"

# 也支持无引号（不包含空格时）
SET name alice
```

## 配置

支持三种配置方式（优先级从高到低）：

1. **命令行参数** - 最高优先级
2. **环境变量** - 前缀 `KVCACHE_`
3. **配置文件** - YAML/JSON/TOML 格式
4. **默认值**

### 命令行参数

```bash
./kv-cache.exe --help

# 常用参数
./kv-cache.exe --config ./config.yaml      # 指定配置文件
./kv-cache.exe --data ./mydata             # 数据目录
./kv-cache.exe --no-persist                # 禁用持久化
./kv-cache.exe --rewrite-size 134217728    # AOF 重写阈值（字节）
./kv-cache.exe --maxmemory 104857600       # 最大内存限制（字节）
./kv-cache.exe --maxmemory-policy allkeys-lru  # 内存淘汰策略
```

### 环境变量

```bash
# Windows
set KVCACHE_DATA_DIR=./mydata
set KVCACHE_MAXMEMORY=104857600
set KVCACHE_EVICTION_POLICY=allkeys-lru

# Linux/macOS
export KVCACHE_DATA_DIR=./mydata
export KVCACHE_MAXMEMORY=104857600
```

### 配置文件

创建 `config.yaml`：

```yaml
# 服务器配置
address: ":6379"

# 数据目录
data-dir: "./data"

# 持久化配置
no-persist: false           # 是否禁用持久化
rewrite-size: 67108864      # AOF 自动重写阈值（字节），默认 64MB

# 内存配置
maxmemory: 0                # 最大内存限制（字节），0 表示不限制
eviction-policy: "noeviction"  # 淘汰策略: noeviction, allkeys-lru, volatile-lru, allkeys-random, volatile-random
```

参考 `config.example.yaml` 文件。

## 持久化

- 数据默认保存在 `./data/appendonly.aof`
- 启动时自动加载 AOF 文件恢复数据
- 每次写命令自动追加到 AOF 文件
- 支持 `FLUSHDB` 清空后文件也会清空

## 退出程序

- 输入 `EXIT` 或 `QUIT`
- 按 `Ctrl+C`（信号处理会保存数据）

## 开发

```bash
# 运行测试
go test ./...

# 构建
go build -o kv-cache.exe ./cmd/kv-cache
```