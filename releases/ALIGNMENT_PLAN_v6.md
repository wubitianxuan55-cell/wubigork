# 轻语模块对齐规划 v6.x

> 基准: ackem 695文件 | wubigrok 138文件 | 剩余 ~555文件
> 高优先级: 45文件 ~280K行 | 中优先级: 80文件 ~380K行
> 核心可移植代码 ~660K行 (排除Minecraft模板/英文i18n/extensions等)

---

## 阶段 5: 基础设施补齐 (v6.0–v6.2)

### v6.0 — 数据库核心 (8文件)
| 文件 | 行数 | 说明 |
|------|------|------|
| db/database.ts | 5,405 | 数据库主模块 |
| db/schemaV1-10.ts | 7,353 | Schema迁移 |
| db/repos/memoryFacts.ts | ~3,500 | 记忆事实仓库 |
| db/repos/chatHistory.ts | ~2,500 | 聊天历史仓库 |
| db/repos/companionState.ts | ~2,000 | 伴侣状态仓库 |
| db/repos/diary.ts | ~2,000 | 日记仓库 |
| db/repos/episodes.ts | ~2,000 | 情节仓库 |
| db/repos/proceduralHabits.ts | ~1,500 | 程序习惯仓库 |

### v6.1 — IPC + Settings (10文件)
| 文件 | 行数 | 说明 |
|------|------|------|
| ipc/chat.ts | ~10,000 | 聊天IPC |
| ipc/memory.ts | ~8,000 | 记忆IPC |
| ipc/session.ts | ~6,000 | 会话IPC |
| ipc/shared.ts | ~5,000 | 共享IPC |
| settings.ts | 8,231 | 全局设置 |
| llmClient.ts | 2,997 | LLM客户端 |
| llmRetry.ts | 3,214 | LLM重试 |
| llmToolChoice.ts | 3,124 | 工具选择 |
| openAiSseStream.ts | 4,831 | SSE流 |
| chat.ts (核心部分) | ~10,000 | 主聊天逻辑 |

### v6.2 — canon收官 (2文件)
| 文件 | 行数 | 说明 |
|------|------|------|
| canon/creatorMemory.ts | 18,690 | 创造者记忆完整实现 |
| canon/originEscalationGuard.ts | 5,561 | 起源升级守卫 |

---

## 阶段 6: channels/weixin 全部 (v6.3–v6.5)

### v6.3 — 微信核心 (8文件)
api/auth/bridge/store/types/index/proactiveGate/monitor

### v6.4 — 微信消息 (7文件)
deliveryPlanner/profiles/documentDelivery/outboundSequence/proactiveMessage/scheduler/queue

### v6.5 — 微信格式+UI (6文件)
markdownToWeixinPlain/emojiContext/activity/stickerRegistry/qrImage/structuredTurn

---

## 阶段 7: memory深度补完 (v6.6–v6.7)

### v6.6 — 核心存储 (5文件)
factStore补完 / episodicStore补完 / knowledgeGraph补完 / habitsStore补完 / retriever补完

### v6.7 — 镜像+文档导入 (8文件)
mirrorCheckRunner / documentImport/* / memoryAudit/*

---

## 🔴 高优先级合计 (~45文件, ~280K行)

| 模块 | 文件数 | 预估行数 |
|------|--------|----------|
| db核心 | 20 | ~50,000 |
| ipc | 8 | ~45,000 |
| channels/weixin | 26 | ~70,000 |
| canon剩余 | 2 | ~24,000 |
| companion | 1 | ~6,000 |
| settings | 1 | ~8,000 |
| LLM层 | 5 | ~20,000 |
| chat核心 | 1 | ~10,000 |
| memory镜像 | 1 | ~3,500 |
| prompt核心 | 4 | ~30,000 |
| context | 2 | ~3,000 |
