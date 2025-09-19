# 技术概览 (Technical Overview)

本文件提供项目 `vocnet` 的技术与架构说明，原本位于 README 的开发与架构章节已迁移至此，README 现在主要面向功能与快速上手。

## 架构原则

项目采用 Clean Architecture 分层，依赖方向始终从外向内：

- **Entity (业务实体)**: 位于 `internal/entity/`，包含核心领域模型与业务规则
- **Use Case (用例层)**: 位于 `internal/usecase/`，编排业务流程，定义接口（Repository / Service 接口）
- **Interface Adapter (适配层)**: 位于 `internal/adapter/`，实现 gRPC 服务、HTTP 映射、数据访问
- **Frameworks & Drivers (基础设施层)**: 位于 `internal/infrastructure/`，数据库、配置、服务器、第三方集成

项目根目录结构示例：
```
├── cmd/                    # 应用入口 (main)
├── api/
│   ├── proto/             # Protocol Buffer 定义
│   ├── gen/               # 生成的 gRPC / Gateway 代码
│   └── openapi/           # 自动生成的 OpenAPI 文档
├── internal/
│   ├── entity/            # 领域实体
│   ├── usecase/           # 用例逻辑
│   ├── adapter/
│   │   ├── grpc/          # gRPC 服务实现
│   │   └── repository/    # 数据访问实现
│   ├── infrastructure/
│   │   ├── database/      # 数据库连接、事务
│   │   ├── config/        # 配置加载
│   │   └── server/        # gRPC 与 HTTP Server 启动
│   └── mocks/             # 生成的 Mock 文件
├── sql/
│   ├── schema/            # 数据库 Schema (迁移初始结构)
│   ├── queries/           # sqlc 查询定义
│   └── migrations/        # 数据库迁移脚本
├── docs/                  # 文档 (本文件等)
└── Makefile               # 开发辅助命令
```

## 技术栈

| 领域 | 技术 | 说明 |
|------|------|------|
| 语言 | Go (>=1.23) | 现代化并发、静态类型 |
| API | gRPC + grpc-gateway | gRPC 为主，自动映射 HTTP/JSON |
| 数据库 | PostgreSQL + sqlc | 类型安全 SQL 访问代码生成 |
| 配置 | Viper | 支持多源配置与热加载 |
| 日志 | logrus | 结构化日志 |
| 测试 | go test + gomock + testify | 单元与集成测试 |
| 构建 | Docker / Makefile | 标准化开发与部署 |
| 协议 | Protocol Buffers | 接口优先设计 |

## 配置管理

配置通过环境变量加载，可结合 `.env` 使用。核心变量示例：
```
SERVER_HOST=localhost
GRPC_PORT=9090
HTTP_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_NAME=vocnet
DB_USER=postgres
DB_PASSWORD=postgres
DB_SSLMODE=disable
LOG_LEVEL=info
LOG_FORMAT=json
```

## 数据访问与 sqlc

所有 SQL 查询位于 `sql/queries/`，通过 `sqlc generate` 生成类型安全代码至 `internal/infrastructure/database/db/`。

最佳实践：
- 查询按业务对象分文件：`user_words.sql`, `sentences.sql` 等
- 使用事务封装跨表操作
- 使用 `context.Context` 传递请求生命周期
- 利用 PostgreSQL 特性（CTE、索引、约束）保证一致性

## gRPC 与 HTTP 网关

- `.proto` 定义存放于 `api/proto`
- 使用 `--grpc-gateway_out` 生成 HTTP 端点
- OpenAPI 文档生成到 `api/openapi/`
- gRPC 服务在 `internal/adapter/grpc/` 实现

典型服务注册（示例）：
```go
grpcServer := grpc.NewServer()
// userv1.RegisterUserServiceServer(grpcServer, userService)
```
HTTP 网关通过 `RegisterXxxHandlerFromEndpoint` 绑定到同一端口或不同监听地址。

## 代码生成

统一通过 Makefile：
```
make generate   # 生成 protobuf / gateway / openapi
make sqlc       # 生成数据访问代码
make mocks      # 生成 gomock 接口实现
```

## 测试策略

| 测试层级 | 内容 | 依赖 |
|----------|------|------|
| 单元测试 | UseCase / Entity 逻辑 | Mock Repository |
| 适配层测试 | gRPC Service 行为 | Mock UseCase |
| 集成测试 | 数据库 + Repository | 本地 Postgres 或容器 |
| 端到端 (可选) | API 全路径 | 真实服务 + 临时数据库 |

建议：
- 使用表驱动测试
- 覆盖成功 + 失败分支
- Mock 外部系统，真实数据库访问仅限集成层

## 日志与错误

- logrus 统一结构化输出
- 错误需加语义上下文（`fmt.Errorf("load user: %w", err)`）
- 业务可定义领域级错误并向上转换为 gRPC Status / HTTP Code

## 目录角色速查

| 目录 | 作用 |
|------|------|
| `cmd/server` | 主程序入口 |
| `api/proto` | 接口契约定义 |
| `api/gen` | 生成的协议及网关代码 |
| `internal/entity` | 领域模型与核心规则 |
| `internal/usecase` | 应用用例 orchestrator |
| `internal/adapter/grpc` | gRPC 服务实现 |
| `internal/adapter/repository` | 数据持久化实现 |
| `internal/infrastructure/database` | 连接、事务、sqlc 生成代码 |
| `sql/schema` | 初始 schema 定义 |
| `sql/queries` | sqlc 查询文件 |
| `docs` | 技术与项目文档 |

## 未来可扩展方向

- 引入 OpenTelemetry 进行分布式追踪与指标采集
- 增加缓存层 (Redis) 优化热点查询
- 增加鉴权 / 认证模块 (JWT / OAuth2)
- 增加多语言 i18n 支持
- 增加 CI/CD Pipeline (GitHub Actions)

## 相关链接

- 主 README：`../README.md`
- 贡献指南：`../CONTRIBUTING.md`

如需进一步补充内容（性能调优、部署策略、运维指南等），可在 `docs/` 下新增文件并从此处添加索引。
