# vocnet 第一阶段 API 设计文档

## 概述

根据路线图第一阶段需求，设计了完整的 gRPC API 体系，支持核心词汇管理功能。API 采用 Clean Architecture 原则，使用 Protocol Buffers 定义，通过 grpc-gateway 同时提供 gRPC 和 RESTful HTTP 接口。

## API 架构

### 服务模块划分

| 服务 | 职责 | proto 文件 |
|------|------|------------|
| WordService | 全局词汇管理 | `word/v1/word_service.proto` |
| UserWordService | 用户词汇学习管理 | `word/v1/user_word_service.proto` |
| WordRelationService | 词汇关系网络 | `word/v1/relation_service.proto` |
| SentenceService | 例句和语料管理 | `sentence/v1/sentence_service.proto` |
| StatsService | 学习统计和分析 | `stats/v1/stats_service.proto` |

### 通用类型系统

#### 枚举类型 (`common/v1/enums.proto`)

- **MasteryLevel**: 掌握程度 (0-5 星评级)
  - `MASTERY_LEVEL_SEEN` (1): 眼熟
  - `MASTERY_LEVEL_RECOGNIZE` (2): 理解  
  - `MASTERY_LEVEL_USABLE` (3): 可用
  - `MASTERY_LEVEL_PROFICIENT` (4): 熟练
  - `MASTERY_LEVEL_MASTERED` (5): 内化

- **RelationType**: 词汇关系类型
  - `RELATION_TYPE_SYNONYM`: 同义词
  - `RELATION_TYPE_ANTONYM`: 反义词
  - `RELATION_TYPE_DERIVATIVE`: 词形变化
  - `RELATION_TYPE_MNEMONIC`: 联想助记
  - `RELATION_TYPE_CUSTOM`: 自定义关系

- **Language**: 支持语言
- **WordStatus**: 学习状态
- **SourceType**: 来源类型

#### 公共消息类型 (`common/v1/types.proto`)

- `PaginationRequest/Response`: 分页参数
- `MasteryBreakdown`: 多维掌握度评分
- `ReviewTiming`: 复习时间安排
- `LearningStats`: 学习统计信息

## 核心 API 详解

### 1. UserWordService - 核心用户词汇管理

#### 关键 API 方法

**CollectWord** - 收藏词汇（路线图核心需求）
```protobuf
rpc CollectWord(CollectWordRequest) returns (CollectWordResponse)
```
- 支持引用已有全局词汇或创建新词汇
- 设置初始掌握程度和学习状态
- 自动处理词汇去重和关联

**UpdateUserWordMastery** - 更新掌握程度
```protobuf
rpc UpdateUserWordMastery(UpdateUserWordMasteryRequest) returns (UpdateUserWordMasteryResponse)
```
- 支持多维度掌握度评分 (听说读写用)
- 更新学习状态和复习计划
- 记录学习进度

**ListUserWords** - 词汇列表查询
```protobuf
rpc ListUserWords(ListUserWordsRequest) returns (ListUserWordsResponse)
```
- 支持按掌握程度、状态、关键词过滤
- 支持多种排序方式
- 支持复习优先级排序

**GetUserWordStats** - 学习统计
```protobuf
rpc GetUserWordStats(GetUserWordStatsRequest) returns (GetUserWordStatsResponse)
```
- 返回各掌握度分布统计
- 支持学习仪表板展示

### 2. WordRelationService - 词汇关系网络

**CreateWordRelation** - 建立词汇关系
```protobuf
rpc CreateWordRelation(CreateWordRelationRequest) returns (CreateWordRelationResponse)
```
- 支持多种关系类型和子类型
- 可设置关系权重和双向性
- 支持用户注释

**GetWordNetwork** - 词汇关系图谱
```protobuf
rpc GetWordNetwork(GetWordNetworkRequest) returns (GetWordNetworkResponse)
```
- 返回词汇关系网络图数据
- 支持最大深度和节点数限制
- 适用于可视化展示

### 3. SentenceService - 例句和语料管理

**AddSentence** - 添加例句
```protobuf
rpc AddSentence(AddSentenceRequest) returns (AddSentenceResponse)
```
- 支持来源信息记录
- 自动处理句子去重
- 支持用户个人标注

**AttachSentenceToWord** - 关联句子到词汇
```protobuf
rpc AttachSentenceToWord(AttachSentenceToWordRequest) returns (AttachSentenceToWordResponse)
```
- 记录词汇在句子中的具体用法
- 支持语法角色和用法类型标注
- 提供上下文学习支持

**ListWordSentences** - 查询词汇例句
```protobuf
rpc ListWordSentences(ListWordSentencesRequest) returns (ListWordSentencesResponse)
```
- 返回指定词汇的所有例句
- 包含词汇使用详情
- 支持公共和私有例句过滤

### 4. StatsService - 学习统计分析

**GetLearningStats** - 综合学习统计
```protobuf
rpc GetLearningStats(GetLearningStatsRequest) returns (GetLearningStatsResponse)
```
- 学习概览和掌握度分布
- 最近活动和进度趋势
- 支持仪表板和报表

**GetDifficultWords** - 困难词汇分析
```protobuf
rpc GetDifficultWords(GetDifficultWordsRequest) returns (GetDifficultWordsResponse)
```
- 基于复习失败率识别困难词汇
- 提供针对性复习建议
- 支持学习策略优化

## 数据模型映射

### 核心实体对应关系

| Proto 消息 | 数据库表 | 说明 |
|------------|----------|------|
| `Word` | `words` | 全局词汇表 |
| `UserWord` | `user_words` | 用户词汇学习记录 |
| `WordRelation` | `word_relations` | 词汇关系 |
| `Sentence` | `sentences` | 全局例句表 |
| `UserSentence` | `user_sentences` | 用户例句关系 |
| `WordUsage` | `word_usages` | 词汇在句子中的用法 |
| `Source` | `sources` | 例句来源信息 |

### SQL 查询支持

为所有 API 操作设计了对应的 SQL 查询：

- **事务操作**: `CollectWord` 支持原子性创建词汇和用户关系
- **复杂查询**: 词汇关系网络的递归查询
- **统计聚合**: 多维度掌握度分布统计
- **过滤排序**: 支持复杂的词汇列表过滤和排序

## HTTP RESTful 映射

通过 grpc-gateway 自动生成 RESTful API：

| gRPC 方法 | HTTP 端点 | 方法 |
|-----------|-----------|------|
| `CollectWord` | `POST /api/v1/user-words` | POST |
| `ListUserWords` | `GET /api/v1/user-words` | GET |
| `UpdateUserWordMastery` | `PATCH /api/v1/user-words/{id}/mastery` | PATCH |
| `CreateWordRelation` | `POST /api/v1/word-relations` | POST |
| `AddSentence` | `POST /api/v1/sentences` | POST |
| `GetLearningStats` | `GET /api/v1/users/{user_id}/stats` | GET |

## 开发工具链

### 代码生成 (使用 Buf)

```bash
make buf-deps  # 更新 buf 依赖
make buf-lint  # 检查 protobuf 代码规范
make generate  # 使用 buf 生成所有 protobuf 代码
make ent-generate  # 生成 ent schema 代码
make mocks     # 生成测试模拟代码
```

### Buf 工具链配置

- **buf.gen.yaml**: 代码生成配置，定义所有插件和输出选项
- **buf.work.yaml**: 工作空间配置，管理 protobuf 模块
- **api/proto/buf.yaml**: 模块配置文件，定义依赖和规则
- **api/proto/buf.lock**: 依赖锁定文件，确保构建可重现性 (自动生成)

### 支持的生成物

- **Go 代码**: gRPC 服务和客户端代码 (via buf.build/protocolbuffers/go)
- **gRPC 服务**: gRPC 服务端代码 (via buf.build/grpc/go)
- **grpc-gateway**: HTTP/JSON 反向代理 (via buf.build/grpc-ecosystem/gateway)
- **OpenAPI**: 自动生成的 API 文档 (via buf.build/grpc-ecosystem/openapiv2)
- **验证代码**: 基于 buf validate 的输入验证 (via buf.build/bufbuild/validate-go)
- **SQLC**: 类型安全的数据库查询代码

### Buf 工具优势

- **依赖管理**: 通过 BSR (Buf Schema Registry) 管理 protobuf 依赖
- **代码规范**: 内置 linting 和格式化规则
- **Breaking Changes**: 自动检测 API 变更影响
- **远程插件**: 使用远程插件，无需本地安装 protoc 和各种插件
- **可重现构建**: 通过 buf.lock 确保依赖版本一致性

## 路线图第一阶段接受标准检查

✅ **能通过 API 收藏新词并返回掌握程度初始值**
- `CollectWord` API 支持设置初始掌握度

✅ **能更新 mastery 并反映在 ListUserWords 中**  
- `UpdateUserWordMastery` 和 `ListUserWords` API 完整实现

✅ **能为词新增例句并列出**
- `AddSentence` 和 `AttachSentenceToWord` API 支持

✅ **能将两个词建立关系并查询**
- `CreateWordRelation` 和 `ListWordRelations` API 支持

✅ **统计接口能返回 mastery 维度数量分布**
- `GetUserWordStats` 和 `GetMasteryDistribution` API 支持

✅ **OpenAPI 文档自动生成**
- Makefile 配置自动生成 OpenAPI 文档

✅ **基础单元测试 + repository 集成测试**
- 通过 mock 生成支持完整测试覆盖

## 后续扩展点

设计中预留了第二阶段复习功能的扩展点：

- `ReviewTiming` 消息类型为复习调度做准备
- `MasteryBreakdown` 支持多维度技能评估
- 统计 API 为复习算法提供数据基础
- 错误追踪字段为学习分析做准备

## 使用建议

### 开发工作流

1. **环境设置**: 运行 `make install-tools` 安装 buf 和其他必要工具
2. **依赖管理**: 使用 `make buf-deps` 更新 protobuf 依赖
3. **代码生成**: 使用 `make generate` 生成所有代码
4. **代码检查**: 使用 `make buf-lint` 检查 protobuf 代码规范

### 开发建议

1. **开发顺序**: 建议按 UserWordService → WordRelationService → SentenceService → StatsService 的顺序实现
2. **测试策略**: 每个服务都有对应的 mock 接口，支持单元测试和集成测试
3. **性能优化**: 关键查询已预留索引设计，复杂统计查询可以考虑缓存
4. **API 版本**: 使用 `/v1/` 版本控制，为未来升级做准备
5. **Protobuf 最佳实践**: 
   - 在添加新字段前运行 `make buf-lint` 检查代码规范
   - 使用 `make buf-breaking` 检查 API 变更的兼容性
   - 定期更新 buf 依赖以获取最新的 Google APIs 和验证规则
