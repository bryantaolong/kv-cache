# kv-cache

An in-memory key-value store supporting multiple data types and persistence.

## Features

- **Multiple Data Types**: String, Hash, List, Set, ZSet
- **Expiration**: TTL support for keys
- **Persistence**: AOF (Append Only File) persistence
- **Type Safety**: Complete type checking to prevent operations on wrong types
- **Quote Support**: Supports string values with spaces

## Quick Start

```bash
# Build
go build -o kv-cache.exe ./cmd/kv-cache

# Run
./kv-cache.exe

# Specify data directory
./kv-cache.exe --data ./mydata

# Disable persistence
./kv-cache.exe --no-persist

# Use configuration file
./kv-cache.exe --config ./config.yaml
```

## Supported Commands

### Generic Key Commands

| Command | Description | Example |
|---------|-------------|---------|
| `SET key value [ttl]` | Set key value, optional TTL (seconds) | `SET name alice` |
| `GET key` | Get key value | `GET name` |
| `DEL key [key...]` | Delete one or more keys | `DEL name age` |
| `KEYS [pattern]` | List all keys, supports `*` wildcard | `KEYS *` `KEYS user:*` |
| `FLUSHDB` | Clear current database | `FLUSHDB` |
| `EXPIRE key ttl` | Set expiration time (seconds) | `EXPIRE name 60` |
| `TTL key` | Check remaining expiration time | `TTL name` |

### Hash Commands

| Command | Description | Example |
|---------|-------------|---------|
| `HSET key field value` | Set field value | `HSET user:1 name bob` |
| `HGET key field` | Get field value | `HGET user:1 name` |
| `HGETALL key` | Get all fields | `HGETALL user:1` |
| `HDEL key field [field...]` | Delete fields | `HDEL user:1 age` |

### List Commands

| Command | Description | Example |
|---------|-------------|---------|
| `LPUSH key value [value...]` | Insert from left | `LPUSH mylist a b c` |
| `RPUSH key value [value...]` | Insert from right | `RPUSH mylist x y z` |
| `LPOP key` | Pop from left | `LPOP mylist` |
| `RPOP key` | Pop from right | `RPOP mylist` |
| `LRANGE key start stop` | Get range of elements | `LRANGE mylist 0 -1` |
| `LLEN key` | Get list length | `LLEN mylist` |

### Set Commands

| Command | Description | Example |
|---------|-------------|---------|
| `SADD key member [member...]` | Add members | `SADD myset a b c` |
| `SMEMBERS key` | Get all members | `SMEMBERS myset` |
| `SCARD key` | Get member count | `SCARD myset` |

### ZSet Commands

| Command | Description | Example |
|---------|-------------|---------|
| `ZADD key score member` | Add member | `ZADD myzset 10 alice` |
| `ZRANGE key start stop` | Get by rank range | `ZRANGE myzset 0 9` |
| `ZCARD key` | Get member count | `ZCARD myzset` |

### System Commands

| Command | Description | Example |
|---------|-------------|---------|
| `CLEAR` | Clear screen | `CLEAR` |
| `HELP` | Display help information | `HELP` |
| `EXIT` / `QUIT` | Exit program | `EXIT` |

## Project Structure

```
kv-cache/
├── cmd/kv-cache/           # Main program entry
│   └── main.go
├── internal/
│   ├── cli/                # Command line interaction
│   │   ├── cli.go          # CLI core logic
│   │   ├── clear.go        # CLEAR command
│   │   ├── hash.go         # Hash command handlers
│   │   ├── keyspace.go     # Keyspace command handlers
│   │   ├── list.go         # List command handlers
│   │   ├── set.go          # Set command handlers
│   │   ├── string.go       # String command handlers
│   │   └── zset.go         # ZSet command handlers
│   ├── config/             # Configuration management (viper-based)
│   │   ├── config.go
│   │   └── config_test.go
│   ├── persist/            # AOF persistence
│   │   ├── aof.go          # AOF file operations
│   │   └── sync.go         # Sync strategies
│   └── storage/            # Storage engine
│       ├── store.go        # MemoryStore core implementation
│       ├── store_test.go   # Storage engine tests
│       ├── string.go       # String command implementation
│       ├── hash.go         # Hash command implementation
│       ├── list.go         # List command implementation
│       ├── set.go          # Set command implementation
│       ├── zset.go         # ZSet command implementation
│       ├── evictor.go      # Memory eviction policies
│       ├── gc.go           # Expired key cleanup
│       └── types/          # Data type definitions
│           ├── value.go    # Value and DataType
│           ├── hash.go     # Hash type
│           ├── list.go     # List type
│           ├── set.go      # Set type
│           ├── zset.go     # ZSet type
│           └── string.go   # String utilities
├── config.yaml             # Configuration file example
└── data/                   # Default data directory
    └── appendonly.aof      # AOF persistence file
```

## Usage Examples

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
Supported commands:

Generic Key Commands:
  SET key value [ttl]         - Set key value (ttl in seconds)
  ...
kv-cache> CLEAR

kv-cache> EXIT
Bye!
```

## Quoting Guidelines

Supports double quotes for strings containing spaces:

```
# Correct
SET msg "hello world"
HSET user:1 name "John Doe"

# Also supports no quotes (when no spaces)
SET name alice
```

## Configuration

Supports three configuration methods (priority from high to low):

1. **Command Line Arguments** - Highest priority
2. **Environment Variables** - Prefix `KVCACHE_`
3. **Configuration File** - YAML/JSON/TOML format
4. **Default Values**

### Command Line Arguments

```bash
./kv-cache.exe --help

# Common parameters
./kv-cache.exe --config ./config.yaml           # Specify configuration file
./kv-cache.exe --data ./mydata                  # Data directory
./kv-cache.exe --no-persist                     # Disable persistence
./kv-cache.exe --rewrite-size 134217728         # AOF rewrite threshold (bytes), 0 to disable
./kv-cache.exe --append-only-policy everysec    # AOF sync policy: always, everysec, no
./kv-cache.exe --max-memory 104857600           # Max memory limit (bytes), 0 for unlimited
./kv-cache.exe --eviction-policy lru    # Eviction policy: no-eviction, lru (currently simplified), random
```

### Environment Variables

```bash
# Windows
set KVCACHE_DATA_DIR=./mydata
set KVCACHE_MAX_MEMORY=104857600
set KVCACHE_EVICTION_POLICY=lru

# Linux/macOS
export KVCACHE_DATA_DIR=./mydata
export KVCACHE_MAX_MEMORY=104857600
```

### Configuration File

Create `config.yaml`:

```yaml
# Server configuration
# address: ":6379"  # Listen address (network service not yet implemented)

# Data directory
data-dir: "./data"

# Persistence configuration
no-persist: false              # Whether to disable persistence
rewrite-size: 67108864         # AOF auto-rewrite threshold (bytes), default 64MB
append-only-policy: "everysec" # AOF sync policy: always, everysec, no

# Memory configuration
max-memory: 0                  # Max memory limit (bytes), 0 for unlimited
eviction-policy: "lru" # Eviction policy: no-eviction, lru (currently simplified), random
```

Refer to the `config.yaml` file for details.

## Persistence

- Data is saved to `./data/appendonly.aof` by default
- Automatically loads AOF file to restore data on startup
- Each write command is automatically appended to the AOF file
- Supports `FLUSHDB` to clear the file as well

## Exiting the Program

- Type `EXIT` or `QUIT`
- Press `Ctrl+C` (signal handler will save data)

## Development

```bash
# Run tests
go test ./...

# Build
go build -o kv-cache.exe ./cmd/kv-cache
```
