<div align="center">

# vocnet

面向词汇学习/语言素材管理的后端服务。提供统一的词汇、例句、使用场景等数据管理能力，支持 gRPC 与 HTTP/JSON 访问，便于集成到学习产品、语言分析工具或教学平台中。

</div>

## 核心功能 (What It Does)

- 词汇与用户词表管理（生词本、熟练度等扩展字段可拓展）
- 例句与使用场景存储与关联
- 词与词之间的关系（同义 / 反义 / 派生 等拓展空间）
- 用户与句子、词汇之间的交互记录模型（便于统计与记忆曲线拓展）
- 双协议访问：gRPC（高性能） + HTTP/JSON（易调试）
- 明确的分层架构，易于二次开发或裁剪

> 技术实现、架构细节请查看：`docs/technical-overview.md`

## 为什么使用 vocnet

| 需求场景 | vocnet 提供的价值 |
|----------|--------------------|
| 语言学习产品需要统一后端 | 现成的词汇 / 例句 / 关系 / 用户交互模型 |
| 需要高性能与多语言客户端 | gRPC 接口 + 自动生成的 HTTP 网关 |
| 想自定义业务逻辑 | Clean Architecture 便于替换/扩展 UseCase 与 Repository |
| 需要严格类型与数据库安全 | sqlc 生成类型安全访问代码 |

## 快速开始

### 前置要求

- Go 1.23+
- PostgreSQL 13+
- protoc (Protocol Buffers 编译器)
- 可选：Docker / Docker Compose

### 1. 获取代码
```bash
git clone https://github.com/eslsoft/vocnet.git
cd vocnet
make setup
```

### 2. 启动数据库并迁移
```bash
make db-up
make migrate-up
```

### 3. 生成代码（如需要）
```bash
make generate sqlc mocks
```

### 4. 启动服务
```bash
make run
# 或
make build && ./bin/rockd-server
```

默认端口：
- gRPC: 9090
- HTTP: 8080

### 5. 调用示例
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"John Doe","email":"john@example.com"}'

curl http://localhost:8080/api/v1/users/1
```

## 配置 (Environment)

在运行前可通过环境变量覆盖默认配置，详见示例：
```env
SERVER_HOST=localhost
GRPC_PORT=9090
HTTP_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_NAME=vocnet
DB_USER=postgres
DB_PASSWORD=postgres
LOG_LEVEL=info
```

## 开发常用命令
```bash
make help
make run            # 启动服务
make test           # 运行测试
make generate       # 生成 gRPC / Gateway / OpenAPI
make sqlc           # 生成数据库访问代码
make mocks          # 生成 gomock
make migrate-up     # 迁移上
make migrate-down   # 迁移回滚
```

## 相关文档

- 技术架构：`docs/technical-overview.md`
- 贡献指南：`CONTRIBUTING.md`
- OpenAPI 文档：`api/openapi/` (生成后)

## 测试
```bash
make test
make test-coverage
```

## 路线图 (Roadmap 摘要)

- [ ] 用户词汇熟练度算法
- [ ] 统计 / 报告 API
- [ ] 词汇关系扩展（同义/派生/音标）
- [ ] 鉴权与多用户隔离
- [ ] OpenTelemetry 集成

欢迎通过 Issue / PR 参与！

## 贡献

请阅读 `CONTRIBUTING.md` 获取分支、提交、测试及代码生成规范。

## 许可证

本项目基于 MIT License 发布，详见 `LICENSE`。

---

如果你在使用中发现改进点，欢迎提交 Issue 或 PR。🙌