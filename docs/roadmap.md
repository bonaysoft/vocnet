# vocnet 产品路线图 (Product Roadmap)

> 目标：打造一个现代化、可扩展、支持多维度记忆与复习的生词 / 语言素材管理后端，服务于 Web / App / 教学 / AI 辅助场景。
>
> 本路线图分阶段逐步演进：先稳固核心数据与 API，再扩展到复习算法、关联记忆、统计分析与智能辅助。

---
## 总览 (Overview)

| 阶段 | 主题 | 主要能力 | 是否包含复习 | 是否需较大数据模型变更 | 备注 |
|------|------|----------|--------------|--------------------------|------|
| Phase 1 | 核心词汇与素材管理 | 单词收藏、掌握程度、例句、用户与句子关联、词与词连接 | 否 | 中等（建立核心实体） | 奠基阶段 |
| Phase 2 | 闪卡式复习 MVP | 基于基础调度的复习（听 / 读 / 拼写 / 识别） | 是（简单调度） | 小 | 快速闭环 |
| Phase 3 | 记忆与关系深化 | 间隔重复算法优化、词网可视关联、统计报表 | 是（进阶算法） | 中 | 强化学习体验 |
| Phase 4 | 智能与个性化 | 语义推荐、自动例句挖掘、个性化计划 | 是（智能化） | 大 | 引入 AI 模型 |

---
## Phase 1：核心词汇管理 (Core Word Management)

> 目标：提供“可用且可靠”的生词体系，让用户可以收藏、标注掌握程度、挂接例句、建立简单关联。

### 1.1 数据模型 (Entities)

| 实体 | 描述 | 关键字段 (示例) | 说明 |
|------|------|----------------|------|
| `word` | 词汇主表 | id, text, phonetic(optional), language, created_at | 可后续加词性、音标、频次 |
| `user_word` | 用户与单词关系 | user_id, word_id, mastery_level, status, notes, created_at, updated_at | mastery_level 采用离散分级 |
| `sentence` | 例句/语块 | id, content, source_id, language, created_at | 允许跨用户共享（未来可区分私有/公共） |
| `source` | 句子来源 | id, type(book/web/audio), title, ref_url, meta(json) | 可选；未填则为临时来源 |
| `word_sentence` | 词与句子的关联 | word_id, sentence_id, user_id(optional), created_at | 支持公共与用户专属关联 |
| `word_relation` | 词与词连接 | word_id_a, word_id_b, relation_type, user_id(optional), created_at | relation_type: synonym / antonym / derivative / mnemonic / custom |
| `user_sentence_interaction` (可选一期末) | 用户与句子的动作 | user_id, sentence_id, action(read/add/copy), created_at | 为后续记忆统计做埋点 |

#### 掌握程度 (Mastery Level Proposal)
采用 0-5 星 + 语义：

| Level | 标签 | 描述 |
|-------|------|------|
| 0 | 未认识 (Unknown) | 初次遇到，不认识 |
| 1 | 眼熟 (Seen) | 看见大概有印象，但无法准确释义 |
| 2 | 理解 (Recognize) | 能读出/听懂大意，但拼写或造句困难 |
| 3 | 可用 (Usable) | 能正确理解+基本拼写，简单造句 |
| 4 | 熟练 (Proficient) | 听说读写都较为自信，语境使用准确 |
| 5 | 内化 (Mastered) | 多语境灵活使用，不需刻意复习 |

> 注意：Phase 2 的复习算法初版可以直接基于 mastery_level + 最近复习时间进行优先级调度。

### 1.2 API 范围 (gRPC + HTTP via grpc-gateway)

| 分类 | API | 描述 | 备注 |
|------|-----|------|------|
| Word | CreateWord | 创建新词（系统或管理员） | 普通用户一般通过收藏触发隐式创建 |
| Word | GetWord | 获取词详情 | |
| Word | ListWords | 分页过滤（text 前缀、语言） | |
| UserWord | CollectWord | 用户收藏一个词（若不存在词则同时创建） | 返回 user_word 记录 |
| UserWord | UpdateUserWordMastery | 更新掌握程度与备注 | |
| UserWord | ListUserWords | 按掌握程度 / 关键词 / 最近更新过滤 | 支持排序：mastery, updated_at |
| Relation | CreateWordRelation | 建立词与词连接 | relation_type 校验 |
| Relation | ListWordRelations | 查询某词相关词 | |
| Sentence | AddSentence | 添加句子（含来源可选） | 可复用已有 source |
| Sentence | AttachSentenceToWord | 将句子关联到词 | 若句子已存在则仅建关联 |
| Sentence | ListWordSentences | 查询词的例句列表 | 可过滤：用户私有 vs 公共 |
| Stats (可选) | CountUserWordsByMastery | 返回各 mastery 数量 | 支持仪表盘 |

### 1.3 技术实现要点
- 使用 ent 生成 CRUD，事务封装 CollectWord（创建词 + user_word）
- `mastery_level` 添加 CHECK 约束（0-5）
- `relation_type` 使用 ENUM 或代码层枚举校验
- 请求校验：protoc-gen-validate (PGV)
- 错误模型：统一业务错误（如重复收藏、非法 level）
- 列表分页：limit + offset，后续可升级 keyset
- 添加基础索引：`word.text`, `user_word(user_id, mastery_level)`
- 预留审计字段扩展位（如 future soft delete）

### 1.4 接受标准 (Acceptance Criteria)
- 能通过 API 收藏新词并返回掌握程度初始值（默认 0 或客户端指定合法值）
- 能更新 mastery 并反映在 ListUserWords 中
- 能为词新增例句并列出
- 能将两个词建立关系并查询
- 统计接口能返回 mastery 维度数量分布
- OpenAPI 文档自动生成
- 基础单元测试 + repository 集成测试（覆盖核心事务 CollectWord 与关系创建）

### 1.5 迭代内不做 (Out of Scope for Phase 1)
- 高级复习算法 / SRS
- 语音 / TTS
- AI 释义 / 语义聚类
- 权限隔离（除 user_id 过滤外）
- 多租户 / 组织结构

---
## Phase 2：闪卡复习 MVP (Flashcard Review MVP)
> 在已有 `user_word` 与 mastery 数据基础上，提供最小可用的复习系统。

### 2.1 复习卡片类型 (Card Types)
| 类型 | 描述 | 数据依赖 | Phase 2 实现方式 |
|------|------|----------|------------------|
| 识别 (Read) | 看单词回忆释义/例句 | word + sentences | 客户端答后提交结果 |
| 听力 (Listen) | 听音辨词 | word.text | 使用占位 TTS 链接或未来生成 |
| 拼写 (Spell) | 给释义/音标拼写 | word.text | 校验用户输入 |
| 用法 (Usage) | 根据例句填空 | sentence + word | 从句子生成 cloze |

### 2.2 调度策略 (Scheduling Simplified)
初版优先级公式（示例，可调整）：
```
priority = f(mastery_level, days_since_last_review, error_streak)
- mastery_level 低 → 权重高
- 时间间隔 > 阈值 → 权重升高
- 最近错误多 → 提升权重
```
服务端只返回需要复习的 `n` 个 cards；客户端逐个上报结果。

### 2.3 新增字段
| 表 | 字段 | 描述 |
|----|------|------|
| user_word | last_review_at | 最近复习时间 |
| user_word | correct_streak | 连续正确次数 |
| user_word | wrong_streak | 连续错误次数（或合并 single difficulty_score） |
| (可选) review_log | user_id, word_id, card_type, result, ts | 行为日志，用于后续算法升级 |

### 2.4 API 扩展
| API | 描述 |
|-----|------|
| GetReviewBatch | 获取待复习卡片列表 (输入：limit, card_types[]) |
| SubmitReviewResult | 提交结果，更新 mastery 值或 streak & last_review_at |

### 2.5 接受标准
- 能获取基于规则排序后的复习批次
- 提交结果后字段更新正确
- 日志表写入成功（若启用）
- 单元测试覆盖调度优先级函数

### 2.6 Out of Scope
- 真正 SRS（如 SM-2 完整实现）
- 自适应个性化难度模型
- 语音识别（口语评估）

---
## Phase 3：深化与可视化 (Enhanced Memory & Analytics)
### 3.1 内容
- 升级调度：基于 review_log 计算记忆曲线 (类似 SM-2 / FSRS 简化)
- 词网可视化：返回某词的 n 度关系图 (graph API)
- 统计/仪表盘：复习频率、掌握度提升曲线、失败 TOP 列表
- 高级查询：按来源 (source) 回放遇见路径

### 3.2 扩展
- `word_relation` 增加权重/方向/标签
- Graph 查询缓存 (Redis / 内存 LRU)
- 聚合统计 materialized view / cron 刷新

---
## Phase 4：智能与个性化 (Intelligent & Personalized)
### 4.1 AI / NLP 增强
- 自动推荐关联词（基于词向量 / 语义相似度）
- 自动生成或抓取高质量例句 (开放语料 + 过滤)
- 个性化复习计划：预测遗忘概率，动态调度
- 语音输入/朗读评估（集成外部 ASR/TTS 服务）

### 4.2 安全 & 多租户
- 用户组 / 课堂模式
- 访问控制策略 (RBAC)

---
## 时间与优先级 (Indicative Timeline)
> 仅为参考，会根据反馈调整。

| 阶段 | 预计周期 | 关键里程碑 |
|------|----------|------------|
| Phase 1 | 2-3 周 | 数据表 & 基础 API 可用；例句关联；关系创建；统计接口 |
| Phase 2 | 2 周 | 复习批次 + 结果回传闭环；基础调度函数测试 |
| Phase 3 | 3-4 周 | 高级调度 + 图谱 API + 统计仪表服务 |
| Phase 4 | 4+ 周 | 语义推荐 + AI 例句 + 个性化调度 |

---
## 风险与缓解 (Risks & Mitigations)
| 风险 | 描述 | 缓解策略 |
|------|------|----------|
| 数据模型过度提前设计 | 过于复杂导致一期延迟 | Phase 1 严格控制字段，只留扩展点 |
| 复习算法失衡 | 用户体验差 | 先上线简化规则 + 日志收集迭代 |
| 例句质量参差 | 用户不信任数据 | 引入来源字段 + 用户自有优先 |
| 关系滥用 | 噪声连接过多 | relation_type + 限制自定义 & 举报机制（后期） |
| 性能瓶颈 | 关系/统计查询慢 | 早期添加必要索引 + 后期缓存/物化视图 |

---
## 与当前架构的对齐 (Architecture Alignment)
- Clean Architecture：新增 UseCase 层实现调度算法与业务规则
- 数据访问继续通过 ent；新增仓储实现支持 `user_words`、`review_log`、`word_relations` 查询
- Proto 目录：分包 `word.v1`, `review.v1`
- 枚举定义：`MasteryLevel`, `RelationType`, `CardType`, `ReviewResult`
- 日志/统计可作为独立 usecase，避免直接耦合在 service 层

---
## 下一步建议 (Immediate Next Steps)
1. 设计并迁移 Phase 1 所需数据库结构（确认字段 + 约束 + 索引）
2. 定义 proto：word_service.proto / relation_service.proto / sentence_service.proto / stats_service.proto（可合并为一个初版）
3. 生成代码 + 编写 repository & usecase + service
4. 编写单元测试（收藏事务、掌握度更新、关系去重）
5. 补充 README 中 Roadmap 链接

---
## 附录：后续枚举初稿 (Enums Draft)
```protobuf
enum MasteryLevel {
  MASTERY_LEVEL_UNSPECIFIED = 0; // default 0
  MASTERY_LEVEL_SEEN = 1;
  MASTERY_LEVEL_RECOGNIZE = 2;
  MASTERY_LEVEL_USABLE = 3;
  MASTERY_LEVEL_PROFICIENT = 4;
  MASTERY_LEVEL_MASTERED = 5;
}

enum RelationType {
  RELATION_TYPE_UNSPECIFIED = 0;
  RELATION_TYPE_SYNONYM = 1;
  RELATION_TYPE_ANTONYM = 2;
  RELATION_TYPE_DERIVATIVE = 3;
  RELATION_TYPE_MNEMONIC = 4; // 联想/助记
  RELATION_TYPE_CUSTOM = 10;  // 保留扩展
}

enum CardType {
  CARD_TYPE_UNSPECIFIED = 0;
  CARD_TYPE_RECOGNITION = 1;
  CARD_TYPE_LISTENING = 2;
  CARD_TYPE_SPELLING = 3;
  CARD_TYPE_USAGE = 4;
}

enum ReviewResult {
  REVIEW_RESULT_UNSPECIFIED = 0;
  REVIEW_RESULT_PASS = 1;  // 正确
  REVIEW_RESULT_FAIL = 2;  // 错误
  REVIEW_RESULT_HARD = 3;  // 勉强记起（可选）
}
```

---
如需对路线图做出修改或补充，请在 PR 中更新本文件。🚀
