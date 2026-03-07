在 Go 项目中，**标准做法是把测试文件放在与被测代码同目录下**，命名为 `xx_test.go`。这是 Go 官方推荐且社区广泛遵循的惯例。

## 两种方式的对比

| 方式 | 命名/位置 | 适用场景 | 优缺点 |
|------|----------|----------|--------|
| **同目录测试** (标准做法) | `foo.go` 旁边放 `foo_test.go` | **绝大多数情况** | ✅ 导入路径简单<br>✅ 可测试包内未导出函数 (`package foo`)<br>✅ 黑盒测试 (`package foo_test`)<br>✅ 一眼看到哪些文件有测试 |
| **集中 test/ 目录** | `test/integration_test.go` | 集成测试、E2E 测试、复杂测试数据 | ⚠️ 需要处理导入路径<br>⚠️ 无法直接测试内部实现<br>✅ 测试代码与业务代码物理隔离 |

## 标准做法详解

```
project/
├── internal/
│   ├── user/
│   │   ├── user.go          # 业务代码
│   │   ├── user_test.go     # 单元测试 (同目录)
│   │   └── repository.go
│   │   └── repository_test.go
│   └── post/
│       ├── post.go
│       └── post_test.go
├── pkg/
│   └── utils/
│       ├── crypto.go
│       └── crypto_test.go
└── test/                    # 可选：放集成测试、fixtures
    ├── integration/
    │   └── api_test.go
    └── fixtures/
        └── data.json
```

### 包声明的两种模式

```go
// user_test.go - 黑盒测试（推荐用于测试公开 API）
package user_test

import "your-project/internal/user"

func TestUser_Create(t *testing.T) {
    u, err := user.New("bryan")  // 只能访问导出符号
    // ...
}

// user_internal_test.go - 白盒测试（需要访问内部实现时）
package user  // 注意：没有 _test 后缀

func TestUser_hashPassword(t *testing.T) {
    u := &User{}
    hash := u.hashPassword("secret")  // 可访问未导出方法
    // ...
}
```

## 什么时候用 `test/` 目录？

1. **集成测试**：需要启动完整服务、连接真实数据库
2. **E2E 测试**：模拟用户完整操作流程
3. **测试数据过大**：fixtures、样本文件太多，污染业务目录
4. **跨包复杂测试**：测试涉及多个模块协作，放在任何一个包都不合适

```
test/
├── integration/
│   ├── user_flow_test.go    # 用户注册->登录->发帖流程
│   └── setup.go             # 测试环境初始化
├── e2e/
│   └── api_test.go          # HTTP 端到端测试
└── fixtures/
    ├── large-dataset.json   # 大测试数据
    └── images/
```

## 给你的建议

既然你在做 **user-system** 和探索 Go 后端项目结构，建议：

- **单元测试**：严格遵循 `xx_test.go` 同目录模式
- **集成测试**：在项目根目录或 `internal/` 下建 `test/integration/`
- **测试工具函数**：放在 `pkg/testutil/` 或 `internal/testhelper/`

这样既能保持 Go 的简洁哲学，又能在需要时灵活处理复杂测试场景。