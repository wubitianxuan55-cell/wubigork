# gaea 3.0 分域评审报告 06 — 办公引擎能力面与存储面（engine-capability）

> 调研方式：只读（glob/grep/read，未改任何代码）。范围：internal/gaea 下 tool/、command/、skill/、memory/、knowledge/、search/、semantic/、retrieval/、db/、archive/、backup/、billing/、cost/、costimport/、pricefeed/、factbase/、fileindex/、filewatch/、tasks/、pins/、largefile/、outputstyle/、specdata/、strutil/、nilutil/、i18n/、evidence/、frontmatter/、fileutil/、secure/、sandbox/、proc/、textsim/、wssearch/、vision/、plugin/ 及 boot/、agent/、control/ 中与本子域直接相关的装配点。
> 所有结论均标注 文件:行号；行号以本次 checkout 为准。未深入细读处已显式标注「未深入」。
> 与 DSH 的对照基于本会话已知的 DeepSeek Harness 机制（会话事件日志事实源、板块 Manifest、Provider Seam）。

## 1. 概览（子域文件清单 + 职责）

### 1.1 能力面（工具/命令/技能/插件）

| 子域 | 文件数 | 职责 |
|---|---|---|
| tool/ | 3（tool.go、envelope.go、registry_test.go）+ builtin/ 34 | Tool 抽象、process-global builtins、per-run Registry、结果信封 |
| tool/builtin/ | 34（含 5 测试 + compact.go + confine.go + workspace.go） | 22 个编译期内置工具，init() 自注册；compact 描述表；写入根/沙箱绑定 |
| command/ | 6 | Markdown slash 命令加载（frontmatter 模板），slash_command 工具 |
| skill/ | 11 | SKILL.md 技能系统：inline/subagent/pipeline 三种运行方式、allowed-tools 白名单 |
| plugin/ | 15 | MCP 客户端：Host/Spec/transport(stdio|http)、prompts→slash、resources→@引用、热增删 |
| agent/（工具面） | — | task 子代理工具、ask 工具、FilterRegistry/SubagentMetaTools 过滤、ToolDispatcher 预检 |
| memory/（工具面） | — | remember/forget/memory_get/promote_session_facts 工具 |

### 1.2 存储面

| 子域 | 文件数 | 职责 |
|---|---|---|
| db/ | 3 | 主脑 Hephaestus.db SQLite 网关（单例 per userDir，WAL，迁移链 V1–V8） |
| memory/ | 31 | 办公记忆：fileBackend(Markdown)/sqliteBackend(facts 表) 双后端 + 检索 + 生命周期 |
| knowledge/ | 7 | 领域知识库：SQLite knowledge 表（默认）/ Markdown 目录回退 |
| cost/ | 2 | 成本库 cost_entries + 分类树 + 语义召回 |
| costimport/ | 2 | xlsx/csv 报价单解析→预览候选（无确认不落库） |
| pricefeed/ | 4 | 价格源订阅：price_sources/price_fetch/cost_price_history |
| billing/ | 3 | 余额查询（DeepSeek /user/balance 形状） |
| factbase/ | 2 | 会话「事实底座」，<session>-facts.json 单文件整写 |
| pins/ | 2 | 工作区常用资料清单 .gaea/pinned.json 整写 |
| archive/ | 2 | 会话归档 JSONL（.gaea/archive/<session>.jsonl，append-only） |
| backup/ | 3 | 一键备份/恢复：zip + manifest，SQLite VACUUM INTO 快照 |
| tasks/ | 2 | 持久化任务队列（tasks 表，SchemaV8） |
| semantic/ | 2 | 本地语义向量索引（semantic_vectors 表，kind×id） |
| retrieval/ | 5 | 本地 Embedder（bge-m3）/ Reranker（cross-encoder）客户端 |
| search/ | 2 | TF-IDF + 中文 bigram 余弦检索（纯内存） |
| wssearch/ | 2 | 工作区全文搜索（TF-IDF，(路径,mtime,size) 缓存） |
| fileindex/ | 2 | 工作区文档扫描→提取正文→semantic.Store 向量化 |
| filewatch/ | 2 | fsnotify 目录监听→去抖批次→增量语义索引 |
| largefile/ | 3 | summarize_file 大文件 map-reduce 摘要工具 |
| outputstyle/ | 2 | 输出风格（persona 块） |
| specdata/ | 1 | 内置规范条文索引（土壤修复 GB/HJ 常量数据） |
| 小工具 | strutil/ nilutil/ i18n/ evidence/ frontmatter/ fileutil/ secure/ textsim/ | 零依赖工具、i18n 静态字典、complete_step 证据账本、frontmatter 解析、原子写、DPAPI 密钥保护、文本相似度 |

### 1.3 说明
- 用户简报中的「digitallife/」在 internal/gaea 下不存在：数字生命能力位于 internal/app/herdsman_digitallife.go（只读解析 Herdsman 的 life.sqlite3，internal/app/herdsman_digitallife.go:100-122），不属于引擎子域。
- 相邻面（cache/context/control/hook/jobs/permission）只作为装配点引用，不展开。

## 2. 工具注册机制与内置工具全清单（重点）

### 2.1 Tool 接口（internal/gaea/tool/tool.go）

| 能力 | 定义位置 | 说明 |
|---|---|---|
| Tool 基接口 | tool.go:17-31 | Name/Description/Schema() json.RawMessage/Execute(ctx,args)/ReadOnly() |
| ToolContext | tool.go:36-44 | SessionID/MessageID/AgentName/ToolCallID/Messages（借自 opencode） |
| ContextualTool | tool.go:50-53 | 可选：ExecuteWithContext(ctx, tc, args) 拿完整会话上下文 |
| CompactDescriptor | tool.go:59-62 | 可选：CompactDescription/CompactSchema 替换 provider 可见定义，压 token |
| PersistWriteTool | tool.go:72-74 | 可选标记：持久写共享存储（成本库/记忆/知识库/技能文件）；IsPersistWrite 判定 tool.go:77-80 |
| ToolEnvelope | tool/envelope.go:17-37 | 统一 JSON 结果信封 {ok,code,error,message,data}，code 机器可读（ok/timeout/denied/...） |

要点：
- ReadOnly 语义 = 「对宿主无可观察副作用」（tool.go:25-30）：agent 仅当整批全 ReadOnly 才并行派发（batch_executor.go:140-190）；bash 与 MCP 插件工具必须返回 false（效果无法静态推断）。
- 注册：process-global builtins map（tool.go:84）由 builtin 包 init() 经 RegisterBuiltin 填充（tool.go:88-94，重名 panic）；Builtins()/LookupBuiltin 只读访问（tool.go:97-114）。
- Registry（per-run 实例，tool.go:120-126）能力矩阵：
  - Add：插入/替换保首见顺序，schema 规范化一次入缓存 canon（tool.go:135-147）；对 suspended 前缀静默拒绝（:140-144）。
  - Hide / HideUnlessOnly：从模型可见 schema 隐藏但保留可调用（tool.go:151-172）。
  - Schemas() / FilteredSchemas(names)：导出 provider.ToolSchema；compact 描述优先；hidden 与白名单外剔除（tool.go:273-322）。
  - PersistWriteNames()：由 PersistWrite 标记自动导出禁写集合（tool.go:258-266）。
  - RemovePrefix / SuspendPrefix / ResumePrefix：MCP 命名空间热删与会话级挂起（tool.go:196-236）。
- MCP 命名空间：模型可见名 "mcp__<server>__<tool>"，SplitMCPName 解析（tool.go:176-191）。
- 结果信封：所有工具结果统一 ToolEnvelope JSON（envelope.go:17-37），Encode 时 SetEscapeHTML(false) + 确定性 map 序保缓存稳定（:6-7/:90-98）。

### 2.2 装配顺序（internal/gaea/boot/boot.go）

boot.Build 是唯一「配置→Controller」装配点（boot.go:82-418），工具注册表按固定 7 步组装：

1. addBuiltins（boot.go:155 → 500-533）：cfg.Tools.Enabled 空 = 全部 22 内置；否则按名取（未知名 warning :506-511）；随后受限实例覆盖同名默认——dir 非空（桌面）走 builtin.Workspace{...}.Tools() 绑定工作区（:518-525），否则 ConfineWriters(writeRoots)+ConfineBash(bashSpec)（:527-532）。
2. startPlugins（boot/plugins.go:23-55）：AutoStartPlugins + CONTEXT7_API_KEY 自动注入 context7 服务器；并行握手；失败进 Host.Failures 并出 Notice（:47-50）。
3. task 工具（boot.go:197-214，子代理继承 reg 减自身 + 可选独立 subagent provider :204-213）；remember/forget/promote_session_facts/memory_get（boot.go:219-222）；ask（boot.go:228）。
4. run_skill / install_skill（boot.go:272-273）+ BuiltinSubagentTools（format-convert/chart-builder/doc-assemble，boot.go:278-280）+ summarize_file（boot.go:283，需注入会话 provider）。
5. slash_command 工具（boot.go:329-353）：技能 + 自定义命令统一 SlashEntry 视图，命令优先于技能（:334-353）。
6. ExtraTools（boot.go:356-360）：桌面端注入 image_gen/diagram/routine_llm/translate_text/fact_add/fact_list/fact_clear（internal/app/gaea_handler.go:100-108）。
7. cfg.Tools.Compact 时 applyCompactToolset 隐藏冗余工具（boot.go:298-300 → 633-652）。

### 2.3 默认启用与开关

- 默认启用清单（cfg.Tools.Enabled 非空时）在 config.Default()：read_file/write_file/edit_file/edit_lines/move_file/ls/grep/bash/web_fetch/web_search/todo_write/complete_step/memory_search（internal/gaea/config/config.go:472-478）；空 = 全部 22 内置（config.go:324-331 + boot.go:501-504）。
- 桌面办公端强制置空 = 注册全部（internal/app/gaea_handler.go:64-65，注释称「47 个工程工具」，实际构成见 §2.4 表，见 §6.11 说明）。
- 其他开关：cfg.Tools.Compact（config.go:327-330）；cfg.Sandbox.Bash/WriteRoots/Network（config.go:467）；cfg.Permissions.{Mode,Allow,Ask,Deny}（config.go:333-343）；桌面端 Bash=off、WriteRoots 跟工作区（gaea_handler.go:66-72）。

### 2.4 办公引擎完整工具一览表（桌面构建路径实际注册，按注册来源分组）

（§2.4a）内置 22 个（init() 自注册；compact 描述表 tool/builtin/compact.go:7-30）：

| 工具名 | 文件:行（Name） | ReadOnly | PersistWrite | 沙箱关系 |
|---|---|---|---|---|
| read_file | readfile.go:37 | 是(:56) | — | 无 OS 沙箱；二进制→markitdown 回退（readfile.go:129-137） |
| write_file | writefile.go:26 | 否(:36) | — | 写入根 confine（writefile.go:53） |
| ls | ls.go:19 | 是(:29) | — | 无 |
| bash | bash.go:39 | 否(:67) | — | sandbox.Command 包裹（bash.go:93），Windows 仅 WSL2 |
| bash_output | bgjobs.go:36 | 是(:46) | — | 读 jobs 管理器缓冲 |
| kill_shell | bgjobs.go:116 | 否(:126) | — | 杀后台任务（Windows taskkill /T 回退） |
| wait | bgjobs.go:169 | 是(:179) | — | 阻塞等任务 |
| web_fetch | webfetch.go:39 | 是(:56) | — | SSRF 防护（webfetch.go:260 起）；代理 netclient.ProxySpec |
| web_search | websearch.go:46 | 是(:63) | — | 多引擎扇出（§5.1）；代理 searchProxy |
| todo_write | todo.go:32 | 是(:64) | — | 会话状态（冲突键 !ledger，batch_executor.go:171-172） |
| complete_step | completestep.go:41 | 是(:77) | — | 会话证据账本（evidence 包） |
| memory_search | memory_search.go:25 | 是(:42) | — | 内存倒排索引（boot 注入，sysprompt.go:68） |
| read_skill | readskill.go:20 | 是(:24) | — | 解析器 boot 注入（sysprompt.go:76-82） |
| format_convert | format_convert.go:19 | 是(:37) | — | 本地 docmd/markitdown 转换 |
| chart_gen | chart_gen.go:20 | 否(:42) | — | 本地 Python matplotlib 子进程 |
| diagram_gen | diagram_gen.go:23 | 否(:43) | — | 本地 Python matplotlib 子进程 |
| knowledge_search | knowledge_search.go:20 | 是(:34) | — | 读知识库（SQLite/文件） |
| knowledge_add | knowledge_add.go:21 | 否(:40) | 是(:41) | 写知识库 |
| cost_search | cost_tools.go:54 | 是(:69) | — | 读成本库 + 语义召回/精排 |
| cost_save | cost_tools.go:311 | 否(:334) | 是(:335) | 写成本库 |
| screen_capture | screenshot.go:25 | 否(:41) | — | 本机截图 |
| vision | vision.go:17 | 是(:34) | — | 本地视觉端点（§5.1） |

（§2.4b）boot 注册 13 个：

| 工具 | 注册点 | 说明 |
|---|---|---|
| task | boot.go:197-214 | 子代理；TaskTool（agent/task.go:97-117） |
| remember | boot.go:219 | 记忆落库（PersistWrite，memory/remember.go:132） |
| forget | boot.go:220 | 记忆删除（PersistWrite，memory/forget.go:57） |
| promote_session_facts | boot.go:221 | 会话事实转正（PersistWrite，memory/promote.go:70） |
| memory_get | boot.go:222 | 读记忆全文（memory/get.go:20） |
| ask | boot.go:228 | 结构化提问（无 Approver 时自行决策） |
| run_skill | boot.go:272 | 技能调用（skill/tools.go:38） |
| install_skill | boot.go:273 | 技能安装（PersistWrite，skill/tools.go:263-265） |
| format-convert | boot.go:278 | 子代理技能包装（skill/tools.go:224-233，3 个之一） |
| chart-builder | boot.go:278 | 同上 |
| doc-assemble | boot.go:278 | 同上 |
| summarize_file | boot.go:283 | 大文件 map-reduce 摘要（largefile/tool.go:27） |
| slash_command | boot.go:353 | 斜杠命令/技能统一入口（command/slashtool.go:56） |

（§2.4c）桌面 ExtraTools 注入 7 个（internal/app/gaea_handler.go:100-108）：

| 工具 | 定义文件 | 说明 |
|---|---|---|
| image_gen | app/gaea_tools.go:24 | 生图（复用模型中心 xAI/Herdsman/Ollama/ComfyUI） |
| diagram | app/gaea_diagram.go:52 | 业务图生成 |
| routine_llm | app/gaea_routine_llm.go:23 | 例程 LLM 调用 |
| translate_text | app/gaea_translate.go:186 | 翻译 |
| fact_add | app/gaea_factbase.go:78 | 事实底座写入 |
| fact_list | app/gaea_factbase.go:134 | 事实底座读取 |
| fact_clear | app/gaea_factbase.go:172 | 事实底座清空 |

（§2.4d）已定义但未注册 2 个（死代码风险，见 §6.3）：ocr（app/gaea_specialist_tools.go:18）、semantic_search（app/gaea_specialist_tools.go:75）。

合计：22 + 13 + 7 = 42 个实际注册工具（+2 未注册 = 44）；gaea_handler.go:64 注释的「47 个」与实现不符（遗留计数，见 §6.11）。

### 2.5 四个核心工具精读（编程板块 / Step 1 试点）

- read_file（readfile.go）：路径经 resolveIn(workDir) 绑定工作区（:74）；目录预检给可操作报错（:94-96）；编码探测 BOM/UTF-16/GB18030 流式解码（:111-163）；二进制 NUL→markitdown 转 md（:129-137，60s 超时 :253）；默认 2000 行、上限 10000（:32/:81-84）；行号右对齐 + [more lines...] 提示（:207-231）；取消检查（:172-177）。
- write_file（writefile.go）：confine(roots) 限制写入根（:53，confine 实现 confine.go:49-60）；自动建父目录（:56-60）；保留原文件权限与编码（:61-70）；返回字节数（:74）。
- bash（bash.go）：5 分钟超时（:23/:147）；PowerShell 拒绝 &&/||（:86-90）；sandbox.Command 包裹（:93）；run_in_background→jobs 管理器（:95-143，跨 turn 存活，Windows Job Object 树杀 :113-135）；前台 8 秒后识别长期运行命令提前返回（:195-258）；输出截断 plain 48KB / json 各 24KB（:282-312）。
- workspace.go（装配器，非工具）：Workspace{Tools(enabled...)} 生成绑定工作目录的工具集（:32-60），WriteRoots 空且 Dir 非空时 Dir 自身为唯一写入根（:33-36）——桌面多标签并发隔离基础。

### 2.6 执行面（agent/，未深入实现细节，仅列行为分类）

- ToolDispatcher 预检（权限 gate + hooks PreToolUse，boot.go:295 → agent/tool_precheck.go:19）；execute_one 串行执行 + 读缓存/写失效（execute_one.go:163-184）+ audit 钩子（:186-195）+ PostToolUse（:213-217）+ 循环守卫（:234）；batch_executor 按冲突键分组并行（batch_executor.go:140-190，getConflictKey :166-190，maxParallel 8 :195）。


### 2.7 权限门与 hooks（工具调用的前置闸，与过滤可行性相关）
- permission/（internal/gaea/permission/）：Policy{Mode, Allow, Ask, Deny}（permission.go:90-97），Decide 优先级 deny>ask>allow>fallback（:114-121，读工具 fallback=Allow，写工具 fallback=Mode）；Rule 支持 "ToolName(glob)"（:52-76）。
- Gate 两态：headlessGate（无 Approver，ask→allow，boot.go:167-174，gaea run 与子代理共用）与交互 Gate（EnableInteractiveApproval，gaea_handler.go:113-115，桌面端工具放行/拒绝弹窗 + ask 结构化提问）。
- hooks/（internal/gaea/hook/）：PreToolUse（可阻断）/ PostToolUse（只观察）/ UserPromptSubmit / Stop / PermissionRequest；项目 hooks 需 trust（hook.IsTrusted，boot.go:181-190），克隆仓库默认不信任。

### 2.8 compact 工具集与「模型可见性」控制（V6.0 P8，过滤先例）
- applyCompactToolset（boot.go:633-652）按 HideUnlessOnly 隐藏 8 组冗余工具：delete_range/delete_symbol、multi_edit、kill_shell/wait、explore/research/review/security_review、notebook_edit、glob——每组隐藏都要求替代工具存在，保证模型永远有可执行路径。
- 语义：hidden 工具仍在注册表、仍可被模型按名调用（tool.go:149-153），只是不进 Schemas() 列表（tool.go:295-297）——「按板块收窄模型可见面」的现成实现，与 DSH manifest 化思路同构。
- 开关：cfg.Tools.Compact（config.go:327-330），默认 false。
## 3. 存储面盘点（要点清单，细节见 06 报告早期草稿已并入 §3.2 表）

### 3.1 db 网关与 schema（internal/gaea/db/）
- 单例 SQLite per userDir：database.go:16-19/:28-74；WAL + synchronous=NORMAL + busy_timeout=5000（:45/:53-64）；MaxOpenConns(1)（:51）；迁移链 runMigrations（:132-160，schema_meta.user_version 递增）。
- 库名 Hephaestus.db（database.go:21-24）。12 张表（schema.go）：
  - schema_meta（:10-13）；facts 办公记忆（:16-33，UNIQUE(project,name)，archived 软删）；profile 画像（:35-42）；knowledge（:44-62）；cost_entries + cost_categories（:67-83/:178-190）；price_sources/price_fetch/cost_price_history（:98-133）；semantic_vectors（:139-147，(kind,id)）；knowledge_history（:152-170，无写入方，见 §6.2）；tasks（:197-214）。

### 3.2 各域持久化方式一览（事件式/重写式判定）
| 域 | 存储位置 | 追加式/重写式 |
|---|---|---|
| 办公记忆 | Hephaestus.db facts 表（memory/sqlite.go:11-18）或 Markdown+MEMORY.md（memory/file_backend.go:15-21） | 行级 UPSERT（重写式）；file 后端每事实 .md + 索引重写 |
| 知识库 | knowledge 表（knowledge/service.go:62-80）或 ~/.gaea/knowledge Markdown | 行级 UPSERT；knowledge_history 未写 |
| 成本库 | cost_entries 表（cost/cost.go:72-101） | 行级 UPSERT；价格历史追加（pricefeed/store.go:207） |
| 语义向量 | semantic_vectors（semantic/semantic.go:39-45） | 行级 UPSERT（内容感知增量 Ensure :74-120） |
| 事实底座 | <session>-facts.json（factbase/factbase.go:98-119） | 整库重写（JSON 原子写） |
| 常用资料 | .gaea/pinned.json（pins/pins.go:26-57） | 整文件重写（限 20 条） |
| 会话归档 | .gaea/archive/<session>.jsonl（archive/archive.go:37-76） | 纯追加 JSONL（事件式） |
| 会话本体 | .gaea/sessions/*.jsonl（config/config.go:566-588） | 纯追加 JSONL（事件式） |
| 审计 trail | audit.jsonl（control/audit.go:19-71） | 追加式，生产未接线（§6.4） |
| 决策日志 | decisions.jsonl（control/decisions.go:16-60） | 追加式，未接线 |
| Dream 审计 | <userDir>/dream-audit.jsonl（controller_memory.go:19-29） | 追加式，已接线 |
| 任务队列 | tasks 表（tasks/tasks.go:1-59） | 行 UPDATE（状态机） |
| 价格抓取/历史 | price_fetch / cost_price_history（pricefeed/store.go:143/:207） | 追加式（INSERT + status） |
| 备份 | zip+manifest（backup/backup.go:1-130） | 快照式（VACUUM INTO） |
| 文件索引 | semantic_vectors kind=file + wssearch 缓存（fileindex、wssearch） | 增量 UPSERT |
| 记忆画像 | profile 表（memory/profile.go） | 行 UPSERT |

结论：已有事件式先例 = session/archive JSONL + audit/decisions（未接线）+ SQLite 内 price_fetch/cost_price_history；整库/整文件重写 = factbase、pins、MEMORY.md/INDEX.md、所有 UPSERT 型事实表。

### 3.3 检索链与文件索引（未深入实现细节，仅列结构）
- search：纯内存 TF-IDF + CJK bigram 余弦（search.go:28-139）；semantic：semantic_vectors 增量重嵌（semantic.go:71-120）；retrieval：本地 Embedder（bge-m3）/Reranker（bge-reranker-v2-m3）（embed.go:19-43、rerank.go:26-50）；wssearch：工作区全文搜索 + (路径,mtime,size) 缓存（wssearch.go:1-60）；fileindex：白名单扩展名扫描 + 300 文件/2MB 上限（fileindex.go:19-48）；filewatch：fsnotify 去抖批次（filewatch.go:27-57）。

## 4. 沙箱 / 进程 / MCP 插件面（要点清单）

### 4.1 sandbox/（internal/gaea/sandbox/）
- Spec{Mode, WriteRoots, Network}（sandbox.go:15-27），Mode=="enforce" 才包一层（:30）。
- macOS：Seatbelt sandbox-exec（seatbelt_darwin.go）；Windows：仅 WSL2 时包成 wsl.exe -d <distro> -- bash -c ...（seatbelt_windows.go:9-31、wsl.go:56-67），否则裸跑；其他平台裸跑。
- 文件写入隔离是进程内路径校验而非 OS 沙箱：confine(roots)（builtin/confine.go:49-60）只约束内置文件工具。
- 与 DSH 对照：DSH 是「文件操作按 mode 在管道层强制」；gaea 是「bash OS jail（Windows 退化）+ 内置工具进程内 roots 校验」。桌面端还把 Sandbox.Bash 置 off（gaea_handler.go:67）→ 办公 bash 无 OS 隔离，唯一边界是 permission 策略 + 写入根校验。

### 4.2 proc/（进程管理，internal/gaea/proc/）
- 隐藏窗口：hide_windows.go:12-18（CREATE_NO_WINDOW）；杀进程树：kill_windows.go:24-32（taskkill /T）/ kill_other.go:15-22（负 pid SIGKILL）；Job Object 树回收：kill_windows.go:44-108（CREATE_SUSPENDED + KILL_ON_JOB_CLOSE）；低优先级：priority_windows.go:14-19（BELOW_NORMAL）。
- 与 DSH 对照：DSH 进程面是 CodeRuntime worker（受信执行单元 + 结果契约 + 两轮零状态）；gaea proc 只是「桌面子进程管理原语」，无 run-code 等价物。

### 4.3 plugin/（MCP 客户端，internal/gaea/plugin/）
- Spec（plugin.go:30-47）：stdio|http（sse 未实现，newTransport :479-494 显式报错）；protocolVersion 2024-11-05（:25）。
- 工具适配：mcpTool 读 readOnlyHint annotation（plugin.go:529-540）→ remoteTool.ReadOnly()（:649，默认 false 不透明）；名称 mcp__<server>__<tool>（:575-597）；remoteTool 实现 CompactDescriptor（:661-664）。
- prompts→slash：Prompt{Name:"mcp__<server>__<prompt>"}（prompts.go:13-20/:54）；resources→@引用：Host.ReadResource("@<server>:<uri>")（plugin.go:102-116、resources.go:52-81）。
- 热增删：Host.Add（plugin.go:373-378）/Host.Remove（plugin.go:421-458，返回前缀供 Registry.RemovePrefix）；会话级禁用 SuspendPrefix（tool.go:216-231）。
- 启动装配：boot/plugins.go:23-55（AutoStartPlugins + CONTEXT7_API_KEY 自动注入）。
- transport 内部（stdio/http 消息循环）未深入。

## 5. 与 3.0 目标相关的关键发现（重点）

### 5.1 Provider Seam（Step 3）要动的硬编码 switch 清单
| # | 能力 | 现状 | 证据 |
|---|---|---|---|
| 1 | Web 搜索后端 | 6 引擎硬编码优先级扇出：local SearXNG→Tavily→Brave→public SearXNG→Bing→DDG；不可配置换序 | websearch.go:169-198（buildEngines），引擎接口 :34-40 |
| 2 | 本地 embedding | HERDSMAN_BASE_URL 环境变量（默认 localhost:8080）+ bge-m3；无 config 路由 | cost_tools.go:190-204；retrieval/embed.go:19-43 |
| 3 | 本地 rerank | 同上（bge-reranker-v2-m3） | cost_tools.go:237-251；retrieval/rerank.go:26-50 |
| 4 | 视觉识别 | 默认端点/模型硬编码（Qwen3.6 长名），仅 GAEA_VISION_BASE_URL/MODEL 两环境变量 | vision/vision.go:21-35 |
| 5 | 生图 | app 层 cfg.ImageBackend 字符串 switch（comfyui 才传 size） | app/gaea_tools.go:94-97 |
| 6 | OCR | 绑死本地 OvisOCR2 常驻服务 | app/gaea_specialist_tools.go:12-14（未注册，§6.3） |
| 7 | 文档转换 | markitdown CLI→python -m markitdown 两级回退 | tool/builtin/readfile.go:246-282 |
| 8 | 余额查询 | 只认 DeepSeek GET /user/balance 形状 | billing/balance.go:34-43 |
| 9 | LLM provider | 已是注册表模式（provider.go:326-361，Register/New by kind）——正向先例 | provider/provider.go:326-361；boot.go:479-494 |
| 10 | 工具名硬编码分类（agent 面） | write/read/ledger/spawn 判定散布 5 处字符串 case：execute_one.go:176/336/442、batch_executor.go:173、evidence.go:303、tool_precheck.go:19、compact_summary.go:45 | 工具「行为分类」无独立 Seam |
| 11 | i18n | 静态 Messages 结构体逐语言文件 | i18n/i18n.go:1-14 |

### 5.2 事件日志（Step 1 事实源）—— 已存在的消费者与缺口
- 已接线：会话 .jsonl（config.go:566-588，dream 消费 control/dream.go:60-80）；archive .jsonl（boot.go:316-323 executor.SetArchive，agent 每轮 RecordMessage agent/agent_run.go:199）；dream-audit.jsonl（controller_memory.go:19-29）；event.Sink 类型化事件流（event/event.go:188-233）。
- 代码存在但未接线：AuditLogger（control/audit.go:19-71）与 DecisionLogger（control/decisions.go:16-60）——agent.auditFunc 钩子已通（agent/agent.go:288-290、execute_one.go:186-195）但 boot 未传 AuditFunc（boot.go:303-313），app 无 NewAuditLogger 调用（全局 grep 仅定义与测试）→ 审计 trail 是「半个现成的事件日志消费者」，Step 1 接线成本最低。
- SQLite 内追加表：price_fetch（pricefeed/store.go:143）、cost_price_history（:207）已是事实源形态；tasks 记录状态变迁。
- 证据账本（evidence.Ledger）是进程内内存（evidence/evidence.go:34-45），complete_step 依赖它但跨会话不可见。
- DSH 对照：DSH 以会话事件日志为唯一事实源、UI 从日志投影；gaea 目前「session .jsonl + 内存状态」双轨，audit/decisions 两轨离线。

### 5.3 工具注册表能否支撑「按板块过滤工具集」（编程板块试点）
**能，机制齐备，缺 Manifest 化声明：**
- 按名白名单：Registry.FilteredSchemas(names)（tool.go:281-322，只影响模型可见 schema）；agent.FilterRegistry（agent/task.go:256-285，构造子代理完整注册表，自动剔除 PersistWrite 与 SubagentMetaTools）——task/子代理技能已在用。
- 隐藏不减注册：Hide/HideUnlessOnly（tool.go:151-172）+ applyCompactToolset（boot.go:633-652）——「可见 ~25 / 可调 ~40」现成先例。
- 会话级挂起 MCP 前缀：SuspendPrefix（tool.go:216-236）；技能级白名单：Skill.AllowedTools frontmatter（skill/skill.go:61-65 → boot.go:245）。
- 缺口：过滤全部是「工具名字符串列表」，工具上无「板块/category/tag」属性，无声明式 Manifest（enabled 子集 + hidden 子集 + extra 注入 + 系统提示片段）；配置 ToolsConfig.Enabled 与 Compact 是仅有两个全局开关（config.go:324-331）。编程板块试点可直接复用 FilterRegistry + FilteredSchemas + HideUnlessOnly 三件套 + 一个 manifest 文件（如 .gaea/boards/coding.toml）。

## 6. 缺陷与风险
1. Windows 沙箱名存实亡：seatbelt_windows.go:9-31 无 WSL2 即裸跑；桌面端 Sandbox.Bash=off（gaea_handler.go:67）→ 办公 bash 无 OS 隔离。
2. 知识库版本历史空转：knowledge_history 表（schema.go:152-170）无写入方（全局 grep 仅 schema 与注释）；knowledge.sqlite.Save 是 ON CONFLICT 覆盖（sqlite.go:38-42）。
3. 两个已定义工具未注册：ocr / semantic_search（app/gaea_specialist_tools.go:14/:71）不在 ExtraTools 列表（gaea_handler.go:100-108）——死代码或清单漂移。
4. 审计/决策日志未接线：全链路代码就绪，boot/app 无接线点（§5.2）。
5. 工具名硬编码分类：5 处字符串 case（§5.1 #10）；ReadOnly/PersistWrite 两标记覆盖不全（todo_write 标 ReadOnly:true 但改状态，todo.go:64）。
6. 双后端漂移：memory fileBackend 不持久化 UpdatedAt/LastUsedAt/SourceSession（store.go:68-73）；knowledge 文件/SQLite 双实现（store.go:14-23）。
7. 整文件重写点：pins.json（pins.go:47-57）、factbase JSON（factbase.go:98-119）、MEMORY.md/INDEX.md——并发写丢失窗口、不可重放。
8. MCP 面：sse 未实现（plugin.go:487-490）；remoteTool 只读性默认 false（plugin.go:649）→ 插件工具永远串行且被当写者；Registry.Add 对 suspended 前缀静默拒绝（tool.go:140-144）。
9. 包级全局注入：builtin 用包级变量注入 searchCfg/searchProxy/memory index/read_skill resolver（websearch_config.go:9-21、sysprompt.go:68-82）——进程级单例，与 Workspace{}.Tools() per-run 隔离冲突（桌面多工作区）。
10. 桌面 ExtraTools 硬编码：7 个工具写死在 app 包（gaea_handler.go:100-108），不在 config/Manifest。
11. 「47 个工程工具」注释与实际不符：实际注册 42 个（22 内置 + 13 boot + 7 ExtraTools），+2 未注册 = 44；explore/research/review/security_review、edit_file 等名字出现在 agent/compact/config 中但无实现（§2.3、§2.4）。
12. evidence 账本不持久化（evidence.go:34-45）：complete_step 证据跨会话不可审计。

## 7. 改造建议（按 3.0 阶段对齐）
1. Step 1（事件日志事实源）：接线 AuditLogger/DecisionLogger（boot.Options 加 AuditFunc，app NewAuditLogger 到 <SessionDir>/audit.jsonl），激活 execute_one.go:186-195 钩子；统一 session/archive/audit 三份 JSONL 的 {session_id, turn, tool_call_id} schema 前缀。
2. Step 2（板块 Manifest）：Tool 增加可选 BoardTags()/Category()；boot 装配改 manifest 驱动（enabled + hidden + extra + 系统提示片段）；编程板块试点组合 FilterRegistry（task.go:256）+ FilteredSchemas（tool.go:281）+ HideUnlessOnly（tool.go:158）+ .gaea/boards/coding.toml。
3. Step 3（Provider Seam）：按 §5.1 把 1-8 号硬编码后端收敛为注册表/接口（searchEngine 接口 websearch.go:34-40 是样板）；embed/rerank/vision/ocr/生图抽 Provider 接口 + config 注册；billing 按 kind 注册。
4. run-code 类能力（编程板块试点核心）：新增 run_code 工具 = 固定 Node/TS runner 脚本 + 结构化结果信封（复用 ToolEnvelope）+ 超时/树杀（复用 bash.go Job Object 逻辑）；permission 加 run_code subject 解析。
5. 清理硬编码工具名分类（§6.5）：字符串 case 收敛为工具接口分类能力（WriteCategory 或 ReadOnlyHint 细化），与 Manifest 同源生成。
6. Windows 沙箱：无 WSL2 时 fail-closed（可配置）而非静默降级；中期评估 Windows Sandbox API / Job Object 资源限制；confine.go 保留第二层。
7. 补知识库版本历史：knowledge.sqlite.Save 在 ON CONFLICT 前 INSERT knowledge_history 快照（表结构现成）。
8. 消除包级全局注入（§6.9）：改为注入工具实例（与 Workspace{}.Tools() 一致）。
9. 注册或删除 ocr/semantic_search（§6.3），对齐「47」计数注释。
10. 工具面 Manifest 落点：per-run Registry（tool.go:120）已是正确粒度，只需把 boot 7 步装配（§2.2）改 manifest 驱动。
11. evidence 持久化：Receipt 按 turn 追加进 audit.jsonl，使 complete_step 证据跨会话可审计。

## 附 A. 调研方法与「未深入」标注
- 工具清单：对 tool/builtin/ 全量 grep RegisterBuiltin 与 Name()/ReadOnly()/PersistWrite() 逐文件核对；boot 与 ExtraTools 从 boot.go、gaea_handler.go 直接读取；skill/tools.go:220-249 核对 BuiltinSubagentTools。
- 存储面：db/schema.go 全表登记；各域 backend 读写接口与追加/重写语义由 SQL 语句形态判定。
- 未接线判定：AuditLogger/DecisionLogger/knowledge_history 全局 grep 仅命中定义与测试。
- 未深入（未细读实现，仅确认存在与接口）：transport_stdio.go / transport_http.go 消息循环；backup 打包/恢复完整实现；pricefeed Confirm 流程；tasks Manager 内部调度；skill pipeline DAG 执行；cache/context 包。

---
*（本报告约 430 行；所有行号来自 C:\AI\wubigrok 当前 checkout，只读调研未改动代码。收尾版。）*