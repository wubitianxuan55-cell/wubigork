# gaea 双空间（space_id: work/play）S1.1 落地设计

> 依据：只读勘察报告（2026，子代理产出）。本文件是 S1.1 的实现权威；Step 拆解见 §7。
> 前提：长期规划 `docs/gaea-nextgen-roadmap-2026.md`（§10 双空间 + §13.1 + §15 缺口清单）。

## 0. 勘察地图（关键架构事实）

| 域 | 位置 | 存储形态 | 空间现状 |
|---|---|---|---|
| 会话存储 | `internal/gaea/agent/session` + `internal/gaea/control` | `<workspace>/.gaea/sessions/` JSONL + 事件日志 + checkpoint + sidecar 家族 | 无 space，按工作区隔离 |
| 主脑记忆 facts | `internal/gaea/db`（Hephaestus.db）+ `internal/gaea/memory` | SQLite，最新 **SchemaV13** | 无 space，按 `project` 隔离 |
| 右脑（轻语）记忆 | `internal/whisper/db`（hermes.db） | SQLite SchemaV13 | **整库天然 play** |
| 任务 | `internal/gaea/tasks`（Hephaestus.db tasks 表，SchemaV8） | SQLite | 无 space |
| 角色库/聊天/小说 | characterlib.db / chat.db / 项目文件夹 | 各自独立 | **整域天然 play** |
| 产物路径 | `internal/app` 多处写死 `.gaea/exports` | 目录约定 | 无分区 |

**关键**：Hephaestus.db 只被 work 侧读写；hermes/chat/characterlib/小说只被 play 侧读写。
S1.1 空间维度聚焦**真正双态的存储**（会话 / facts / tasks），不给整库 play 的库加列。

## 1. 字段落点表

**加列**：
1. `facts`（Hephaestus.db）：`space_id TEXT NOT NULL DEFAULT 'work'` + `idx_facts_space ON facts(project, space_id)`（schema.go:16-32；写入端 memory/sqlite.go:58-70，隔离键 `project`）。
2. `tasks`：`space_id TEXT NOT NULL DEFAULT 'work'` + `idx_tasks_space ON tasks(space_id, status)`（schema.go:196-214；tasks.go:47-68 加 `Space string json:"spaceId"`）。
3. 会话目录族：**路径分区** `sessions/<space>/<id>.jsonl`；BranchMeta/Checkpoint/日志行加 space 字段做自描述。
4. 子代理 transcript meta（未接线，前瞻）：meta.json 加 `space`（接线时）。

**不加列**（及理由）：profile（全局画像共享层）；knowledge 系（当前仅 work 读写）；cost_* 全部（纯 work）；semantic_vectors（按 kind 谓词）；**hermes.db 全表**（整库 play）；chat.db/characterlib.db/小说项目（整库 play，且这两库**无版本迁移链**，想加列得先移植 schema_meta 框架）；`.gaea/reviews`（Go 零引用）。

## 2. 迁移方案

### SQLite：下一个 SchemaV = **V14**（internal/gaea/db/database.go:132 迁移链 + user_version）
```sql
ALTER TABLE facts ADD COLUMN space_id TEXT NOT NULL DEFAULT 'work';
CREATE INDEX IF NOT EXISTS idx_facts_space ON facts(project, space_id);
ALTER TABLE tasks ADD COLUMN space_id TEXT NOT NULL DEFAULT 'work';
CREATE INDEX IF NOT EXISTS idx_tasks_space ON tasks(space_id, status);
```
- 回填零成本（ADD COLUMN DEFAULT 对既有行即取默认值 = work）。
- **不动唯一键** `UNIQUE(project,name)`（SQLite 不能 ALTER 约束；S1.1 只做谓词过滤与隔离闸门；跨空间同名冲突留 S1.2 决策）。
- **不按空间分库**（连接池 per-userDir 单例 + 备份按固定路径枚举）。
- hermes.db 不迁移；characterlib/chat 无迁移链，绕开。

### 文件型（会话目录族）：目录分区 + 读兼容
文件族：`<ts>-<model>.jsonl`（legacy 镜像）、`<id>.jsonl.meta`（分支）、`<id>.jsonl.state.json`、`<id>.gaea-log.jsonl`、`<id>.gaea-checkpoint.json`、`.pinned.json`、`archive/`、`subagents/`。
- 写：`WorkspaceSessionDir(cwd, space)` → `<cwd>/.gaea/sessions/work|play/`（config.go:638-647 扩参）。
- 读：**旧平铺文件视为 work 兼容读取，不搬文件**；`listDir`（save.go:176-212）与 `GaeaListProjectSessions`（gaea_ui.go:248-310）按两空间目录各列一次 + 平铺兜底。
- Fork 天然继承（`controller_rewind.go:150-154` dir=filepath.Dir(path)）；归档各空间目录下自己的 archive/。
- **硬校验点（漏改=静默拒绝，比崩溃更阴险）**：`sessionDirForPath`（gaea_ui.go:387-401）接受 `sessions/<space>/`；`GaeaArchiveSession`（gaea_ui.go:331）、`GaeaPinSession`（gaea_ui.go:362）的 `Base(dir)!="sessions"` 守卫放行。

### 回滚/只读兼容
配置键 `space.mode = "on"|"off"`（默认按产品定；建议默认 on 但 off 时所有读写路径忽略 space）——仿 `session.log_format` 三件套（cfg → ctrl.SetLogFormat → session 字段）。只读兼容：旧平铺会话恒按 work 可读；V14 列 DEFAULT 恒可查。

## 3. event log 格式版本（方案 A：行级 `space` 字段）
- `LogEntry` 加 `Space string json:"space,omitempty"`（log.go:87-92）；`OpenLog`/Writer 带 space 来源，`AppendRaw`/`formatLogLine` 统一写入（log.go:251-273,389-401）。
- 读取端（`parseLogBytes` log.go:357-373、`ProjectMessages`、`Restore` checkpoint.go:84-118）空 space 一律降级 `work`；恢复校验「日志 space ↔ 会话目录归属」不一致则拒绝（防穿越落点）。
- `Checkpoint` 加 `Space string json:"space,omitempty"`（checkpoint.go:20-28）对账。
- **不做日志头行**（会破坏「seq=已写行数」不变量：countLogLines/OpenLog 续 seq/Restore 游标/BalanceEntries 全要跳过 header，风险>收益）。
- `session.space` 独立键（非 log_format）注入路径仿 logFormat 三件套（config → controller NewSession/Resume/SetSessionPath/applyLogFormat 同点 → session.Session SetSpace/Space()，session.go:24-27,115-134）。
- 旧事件无字段 = 零值 + 读取端 `SpaceOr("work")`；golden 测试 fixture 需同步（gaea_history_golden_test.go/event_mode_test.go/session_log_test.go）。

## 4. 子代理空间继承（纯内存语义）
- 链路：TaskTool.Execute（task.go:181-252）→ prepareRun（:480-496）→ runSubSession（:322-385）→ RunSubAgent → NewSession（:514-516）。**父会话 id 不传子代理**；TaskTool 是 boot 期单例（boot.go:206-223），仅 ctx 有 jobs/CallContext。
- **SubagentStore 是全库死代码**（NewSubagentStore/WithTranscripts 零调用；`<sessionDir>/subagents/` 只有 app 读取约定）→ S1.1 无磁盘注入，纯 ctx 传 space。
- 注入：`session.Session` 带 space（SetSpace/Space()）；ctx 注入仿 `jobs.FromContext(ctx)`（task.go:230-238）加 `space{}`/`SpaceFromContext`（缺省 work）；`runSubAgentInternal`（task.go:527-551）继承——RunSubAgent 与 RunSubAgentWithSession（continue_from）两条路都覆盖；skillRunner（boot.go:245-280）同理。
- 防穿越：A）runSubAgentInternal 入口断言子会话 space==ctx space，fail-closed；B）文件层复用 dispatcher 门禁链 + `tool.IsPersistWrite`（task.go:276-283 现成先例），S1.2 时 WriteRoots 按 space 收敛；C）日志层：恢复校验（§3）。
- 前瞻：接通 SubagentStore 时 SubagentMeta 加 Space（subagent_store.go:36-43）+ PrepareContinue 校验一致。

## 5. 产物路径分区（`.gaea/exports` 写死点清单）
| 文件:行 | 内容 | 改法 |
|---|---|---|
| internal/app/gaea_handler.go:169-175 | 启动 mkdir .gaea/{work,exports} | 按空间 mkdir |
| internal/app/gaea_export.go:71 | 导出交付物 | 换 helper |
| internal/app/gaea_crosslink.go:72 | 交叉嵌入 | 换 helper |
| internal/app/gaea_deliverable_zip.go:57 | 交付 zip | 换 helper |
| internal/app/gaea_pdf.go:93 | PDF | 换 helper |
| internal/app/gaea_knowledge_meta.go:146 | 知识导出 | 换 helper（knowledge 属 work 固定） |
| internal/app/gaea_preview.go:58 | 预览白名单 | 加 `.gaea/play/exports` |
| internal/gaea/agent/single_prompt.go:49-58 | prompt 文本写死 | 参数化 |
| internal/app/gaea_templates.go:24-52 | 8 个任务模板 | 参数化 |
| internal/app/chat_service.go:331 | 聊天导出 whisper_data/exports/chat | **不改**（play 天然） |

新增 `internal/gaea/spaces` 包：`Space` 常量（work/play）、`Validate`、`ExportsDir(cwd,space)`、`WorkDir(cwd,space)`。**work 返回现状路径**（不挪目录不破坏既有产物链接），play 返回 `.gaea/play/exports`。

## 6. 绑定面与前端影响
- 新增 `GaeaSpaceList()/GaeaSpaceActive()/GaeaSpaceActivate(space)` 挂 **CoreB**（绑定面 499 → 501/502；重新 `gen_bindings -names` 同步 bindingNames.ts）。
- `SessionMeta` 加 `spaceId` 字段（gaea_ui.go:224 前后）——**改返回体不加方法**，前端按字段过滤。
- 会话统计/恢复不变（路径参数含空间目录，守卫放行后自动工作）。

## 7. Step 拆解（4 个独立可提交/可回退；依赖：S1→S2→S3；S4 依赖 S2 可与 S3 并行）
| Step | 内容 | 验证 | 回退点 |
|---|---|---|---|
| **S1 列落库** | SchemaV14（facts/tasks 列+索引）；tasks.Task 加 Space、Enqueue 入参带 space（缺省 work）、GaeaTaskList 谓词（mode=off 恒真）；memory sqliteBackend 读写带 space | db 迁移测试（旧库升级+新库全链）、tasks/memory 回归、行为不变断言 | 纯 ADD COLUMN，代码回退即恢复 |
| **S2 会话空间（核心）** | spaces 包；WorkspaceSessionDir；目录分区写新读兼容（平铺=work）；sessionDirForPath/archive/pin 守卫放行；LogEntry/Checkpoint/BranchMeta 加 space（读端降级 work）；Controller 三点传播；space.mode 开关 | log_test/event_mode_test/checkpoint_test/rewind_test/gaea_projects_test + 旧平铺恢复兼容 | 开关 off + 常量恒 work → 路径退回平铺 |
| **S3 子代理继承+校验** | ctx space 注入；SpaceFromContext；runSubAgentInternal/skillRunner 继承；一致性断言 + PersistWrite 排除复用 | task_test/subagent_store_test 继承断言 | 缺省 work = 行为退回现状 |
| **S4 产物分区+绑定面** | spaces.ExportsDir/WorkDir 替换 5 处 Go 写死 + 文本参数化；preview 白名单；GaeaSpace* 挂 CoreB；gen_bindings 重新生成 + 前端同步 | export/zip/pdf/preview 测试；bindings drift CI | helper 对 work 返回原路径；新绑定删除即回退 |

## 8. 风险与坑（实测）
1. SubagentStore 死代码 → continue_from 桌面端实际不可用，勿按落盘假设设计。
2. 会话路径守卫是**静默拒绝**（gaea_ui.go:331,362,387-401）——S2 验证清单逐条覆盖三守卫。
3. 恢复链（resumeLastSession→ResumeFromDisk→Restore）不读 meta → space 只能从**路径**推导（目录分区方案的决定性优势）。
4. `NewEventLogSink(dir, sink)` 的 dir 被丢弃（sink.go:29-31），路径由 pathSrc 懒解析——别在此处加空间逻辑。
5. facts `UNIQUE(project,name)` 不可 ALTER → 同 slug 同名跨空间冲突留 S1.2 决策（建议 `(project,space_id,name)` 重建）。
6. characterlib.db/chat.db 无迁移机制 → 靠整库归属绕开。
7. tasks 表是用户级全局（跨工作区可见）→ 旧任务回填 work，UI 文案说明。
8. 日志 golden 测试 fixture 需同步（宽松解码不破坏运行）。
9. `.gaea/reviews` 非代码路径，分区只需改 prompt 约定。
10. `space.mode` 走配置文件（勿走环境变量，避免测试矩阵翻倍；GAEA_DATA_ROOT 已承担测试隔离）。
