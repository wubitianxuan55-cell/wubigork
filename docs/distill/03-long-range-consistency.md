# 03 · 长程一致性域（Memory + Embedding + 章节上下文）源码蒸馏

- 唯一事实来源：`clones/MuMuAINovel`（v1.5.5，git clone 于工作区）
- 本地对照对象：`internal/novelcontext/novelcontext.go`（876 行）+ `internal/memory/memory.go`（289 行）+ `internal/gaea/semantic/semantic.go`（288 行）
- 本轮不写生产代码。所有结论均附 `文件:行号`；无法从源码确认的一律显式标注「未能从源码确认」。
- 阅读方法：逐文件通读 + 跨文件 grep 交叉验证调用关系。「未被任何代码引用」的判断依据是全仓 `grep` 结果（见 §10）。

## 0. 阅读范围（真实文件清单）

| 文件 | 行数 / 大小 | 在本域中的角色 |
|---|---|---|
| `clones/MuMuAINovel/backend/app/services/chapter_context_service.py` | 1870 行 / 85 KB | 上下文组装核心（双模式构建器） |
| `clones/MuMuAINovel/backend/app/services/memory_service.py` | 795 行 / 29 KB | ChromaDB 向量记忆服务 |
| `clones/MuMuAINovel/backend/app/services/onnx_embedding.py` | 295 行 / 11 KB | ONNX 本地 embedding 推理 |
| `clones/MuMuAINovel/backend/app/services/plot_analyzer.py` | 602 行 / 28 KB | 记忆**生产者**（从未被蒸馏的另一半） |
| `clones/MuMuAINovel/backend/app/models/memory.py` | 200 行 | `StoryMemory` / `PlotAnalysis` 数据模型 |
| `clones/MuMuAINovel/backend/app/api/memories.py` | 514 行 | 记忆 API 契约 |
| `clones/MuMuAINovel/backend/app/api/chapters.py` | 5251 行（读相关段） | 记忆写入主路径 + 上下文集结处 |
| `clones/MuMuAINovel/backend/app/services/prompt_service.py` | 3158 行（读相关段） | 上下文注入 prompt 的最终位置 |
| `clones/MuMuAINovel/backend/app/services/import_export_service.py` | 1840 行（读相关段） | 记忆导出/导入 |
| `clones/MuMuAINovel/backend/requirements.txt` | 30 行 | 依赖与版本证据 |

依赖版本（`requirements.txt`）：`chromadb==1.3.2`、`onnxruntime==1.28.0`、`tokenizers==0.22.2`、`numpy==2.5.1`、`sqlalchemy==2.0.25`、`aiosqlite==0.22.1`、`asyncpg==0.31.0`。**无 tiktoken / transformers 运行时依赖**（`transformers` 只在构建期 `requirements-embedding-export.txt` 中，见该文件第 1 行注释）。

---

## 1. 数据模型（字段级）

### 1.1 `StoryMemory`（关系库侧事实源）
`backend/app/models/memory.py:8-77`，表名 `story_memories`。

| 字段 | 类型 | 行号 | 备注 |
|---|---|---|---|
| `id` | String(100) PK | :12 | 默认 `uuid4`，但实际写入路径会传入自定义 id（见 §2） |
| `project_id` | String(36) FK→projects CASCADE, index | :13 | |
| `chapter_id` | String(36) FK→chapters CASCADE, index, nullable | :14 | |
| `memory_type` | String(50) not null, index | :17 | 注释列出的枚举：`plot_point / character_event / world_detail / hook / foreshadow / dialogue / scene`（**:17-26**）——**注释漏了代码实际写入的 `chapter_summary`**，见 §10-D7 |
| `title` | String(200) | :29 | |
| `content` | Text not null | :30 | 注释「100-500字」，代码未强制 |
| `full_context` | Text | :31 | |
| `related_characters` | JSON | :34 | 注释为**角色 ID 列表**；`plot_analyzer.py:450` 写入的是**角色名**，语义不一致（§10-D13） |
| `related_locations` | JSON | :35 | |
| `tags` | JSON | :36 | |
| `importance_score` | Float default 0.5 | :39 | 0.0–1.0 |
| `story_timeline` | Integer not null, index | :42 | 写的是章节序号（`plot_analyzer.py` 经 `metadata.chapter_number`） |
| `chapter_position` | Integer default 0 | :43 | 章节内字符位置 |
| `text_length` | Integer default 0 | :44 | |
| `is_foreshadow` | Integer default 0 | :47 | **0=普通 / 1=已埋下 / 2=已回收** |
| `foreshadow_resolved_at` | String(100) FK→chapters SET NULL | :48 | |
| `foreshadow_strength` | Float | :49 | 只在校验导入时读取（`import_export_service.py:1248`），**写入路径从不赋值** |
| `vector_id` | String(100) unique | :52 | **`api/memories.py` 写入，`api/chapters.py` 不写 → 主路径恒 NULL**（§10-D5） |
| `embedding_model` | String(100) default `paraphrase-multilingual-MiniLM-L12-v2` | :53 | 硬编码默认值 |

### 1.2 `PlotAnalysis`（章节分析结果，上下文组装的二级事实源）
`backend/app/models/memory.py:80-200`，表名 `plot_analysis`，`chapter_id` **unique**（:86，每章至多一条）。

上下文集装实际只消费其中 3 个字段：
- `plot_points` JSON（:124-131）：`[{content, importance, type, impact}]`
- `character_states` JSON（:135-145）：`[{character_id, character_name, state_before, state_after, psychological_change, key_event, relationship_changes}]`
- `chapter_id`（关联键）

其余字段（`hooks`、`foreshadows`、`emotional_*`、`quality_*` 评分、`analysis_report`、`suggestions`、`pacing`、`dialogue_ratio`…）供前端分析页展示，**不进入章节生成上下文**（见 §3 表）。

---

## 2. 记忆的写入时机与两条互不一致的写入路径

### 2.1 写入时机：章节生成完成后**自动**触发
`api/chapters.py:1881-1907`：
1. 章节正文落库后创建 `AnalysisTask(status='pending')`（:1882-1891）；
2. `await asyncio.sleep(0.05)`（:1897，注释「短暂延迟确保SQLite WAL完成写入」）；
3. `background_tasks.add_task(analyze_chapter_background, ...)`（:1900-1907）；
4. 分析期间向前端推 `analysis_started` SSE 事件（:1921-1925）。

`analyze_chapter_background`（:886-922）用 `asyncio.wait_for(..., timeout=ANALYSIS_TASK_TIMEOUT_SECONDS)` 做硬超时兜底（`ANALYSIS_TASK_TIMEOUT_SECONDS = 600`，:70），并把中断/超时写成任务终态（:907-922）。

### 2.2 两个写入实现，字段落地规则不同

| 维度 | 路径 A：`api/chapters.py`（**主路径**，生成后自动分析） | 路径 B：`api/memories.py`（接口手动触发） |
|---|---|---|
| 入口 | `_analyze_chapter_background_impl` :925 | `analyze_chapter` :24 |
| 记忆 id | `f"{chapter_id}_{mem['type']}_{index}"` 确定性 :1239 | `str(uuid.uuid4())` :174 |
| `vector_id` | **不赋值** :1254-1269 | 赋 `vector_id=memory_id` :185 |
| 重分析前清理关系库 | 删除本章 `StoryMemory` :1226-1234 | 删除本章 `StoryMemory` :153-159 |
| 重分析前清理向量库 | **不调用** `delete_chapter_memories` | 调用 :161-169（异常降级为 warning） |
| 写向量库 | `batch_add_memories` :1279-1283 | 逐条 `add_memory` :191-198 |
| 单条失败影响 | 整批失败（`batch_add_memories` 的 `except` 吞异常返回 0，`memory_service.py:215-217`） | 单条失败仅该条丢失 |

**结论**：主路径在「同一章被重新分析」时，关系库旧行被删、向量库旧向量仍在（`delete_chapter_memories` 未在该路径调用；全仓 grep 该函数的调用点只有 `api/chapters.py:389`（清空章节内容 API）与 `:478`（删除章节 API），以及 `api/memories.py:163`）。由于路径 A 的 id 是确定性的，第二次分析会产生**同 id 的 `collection.add`**。`memory_service.batch_add_memories` 未做去重/upsert（`memory_service.py:205-210`），异常被吞并返回 0，结果是**该章向量整体不更新**（§10-D1）。具体抛出的异常类型随 ChromaDB 版本变化，**本机未安装 chromadb（`python -c "import chromadb"` → ModuleNotFoundError），未能实测**。

### 2.3 记忆提取规则（谁决定 importance）
`plot_analyzer.py:305-486` `extract_memories_from_analysis`：

| 记忆类型 | 触发条件 | `importance_score` | 行号 |
|---|---|---|---|
| `chapter_summary` | 必有：`analysis['summary']` → 前 3 条 `plot_points` 拼接 → `chapter_content[:300]` | 固定 **0.6** | :329-359 |
| `hook` | `hook.strength >= 6` | `min(strength/10, 1.0)` | :362-385 |
| `foreshadow` | 全部 | `min(strength/10, 1.0)`；`is_foreshadow = 1 if type=='planted' else 2` | :388-412 |
| `plot_point` | `importance >= 0.6` | 原样 | :415-436 |
| `character_event` | 每个 `character_states` 项 | 固定 **0.7** | :439-453 |
| `plot_point`（冲突） | `conflict.level >= 7` | `min(level/10, 1.0)` | :456-479 |

**没有任何记忆去重/合并/衰减**；`text_position` 由 `_find_text_position` 三级匹配（精确 → 去标点 → 关键词前半段，:488-518）计算。

---

## 3. 章节上下文组装算法

### 3.1 双模式：`one-to-one` 与 `one-to-many`
模式来自项目设置 `outline_mode`，在 `api/chapters.py:1639`（1-1）与 `:1680`（1-N）分流；对应构建器 `OneToOneContextBuilder`（`chapter_context_service.py:1113`）与 `OneToManyContextBuilder`（:215）。

### 3.2 组装顺序（**顺序即优先级**）

**1-N 模式** `OneToManyContextBuilder.build`（:248-366）：

| 序 | 区段 | 数据来源 | 行号 |
|---|---|---|---|
| 1 | `chapter_outline`（P0） | `chapter.expansion_plan` JSON（`plot_summary/key_events/character_focus/emotional_tone/narrative_goal/conflict_type`）→ 回退 `outline.content` → `chapter.summary` → `'暂无大纲'` | :299 / :368-393 |
| 2 | `recent_chapters_context`（P0） | 最近 **10** 章（`RECENT_CHAPTERS_COUNT=10`，:235）的 `StoryMemory(memory_type='chapter_summary')` 优先，回退 `Chapter.expansion_plan.plot_summary`，再回退 `Chapter.summary`；每章 180 字符 + 最多 3 个真实情节点 | :302-306 / :668-750 |
| 3 | `continuation_point` + `previous_chapter_summary` + `previous_chapter_events`（P0） | 上一章**完整正文**（`ENDING_LENGTH=None`，:229 → :854-860 走「不截断」分支）+ 章节摘要（`StoryMemory` → `chapter.summary` → `expansion_plan.plot_summary`，截 300）+ 关键事件最多 5 条 | :308-321 / :825-893 |
| 4 | `chapter_characters`（P1） | 本项目**全部**角色 → 按 `expansion_plan.character_focus` 过滤 → **截断至 10 个**；批量查关系/组织成员/职业关联；`appearance[:100]`/`personality[:100]`/`background[:150]`/`current_state[:150]`；关系网络全量拼接；组织归属最多 2 个；组织成员最多 5 个 | :323-332 / :395-666 |
| 5 | `chapter_careers`（P1/P2） | 上述职业的**完整阶段体系**（逐阶 `level/name/description`，不截断阶数） | :637-666 |
| 6 | `emotional_tone` | `expansion_plan.emotional_tone` → `outline.structure.emotion` → `'未设定'` | :330 / :895-919 |
| 7 | `relevant_memories`（P2） | 向量检索（见 §5） | :335-340 / :752-798 |
| 8 | `foreshadow_reminders`（P2） | `foreshadow_service` 三档：本章必回收 / 超期（最多 3） / 未来 3 章内到期（最多 3） | :342-348 / :985-1061 |

**1-1 模式** `OneToOneContextBuilder.build`（:1148-1360）：

| 序 | 区段 | 来源 | 行号 |
|---|---|---|---|
| 1 | `chapter_outline`（P0） | `outline.structure` 的 `summary/scenes/key_points/emotion/goal` 五段拼接 | :1188 / :1473-1508 |
| 2 | `recent_chapters_context`（P1） | 最近 10 章摘要（180 字符/章）+ 最多 3 个情节点 | :1193-1197 / :1362-1427 |
| 3 | `continuation_point`（P1） | 上一章**完整正文**（**无条件不截断**，:1211-1214） | :1200-1238 |
| 4 | `previous_chapter_summary`（P1） | `StoryMemory(chapter_summary)` → `chapter.summary`，截 300 | :1226-1234 |
| 5 | `chapter_characters` + `chapter_careers`（P1/P2） | 从 `outline.structure.characters` 取名字 → 查 `Character`；名字列表为空则「暂无角色信息」；最多 10 个角色，**每个角色单独发一次关系查询（N+1）**，`:1672-1681` | :1244-1287 / :1510-1791 |
| 6 | `foreshadow_reminders`（P2） | 同 1-N，共用同一实现 | :1291-1298 / :1793-1869 |
| 7 | `relevant_memories`（P2） | 向量检索，阈值硬编码 **0.6**（:1317） | :1301-1340 |

### 3.3 检索 query 的构造（**这是本域最可借鉴的一段**）
不是把大纲原文整段拿去 embed，而是**结构化拆字段、拼高信号短语**：

- 1-N：`_build_memory_query_from_expansion_plan`（:800-823）
  段式拼接 `人物：…；关键事件：…；叙事目标：…；冲突：…；情绪：…`，字段限额 `character_focus[:8] / key_events[:6] / narrative_goal[:1] / conflict_type[:1] / emotional_tone[:1]`；整体 **`[:800]`**；无 expansion_plan 时回退 `chapter_outline[:500]` 并把换行替换为空格。
- 1-1：`_build_memory_query_from_outline_structure`（:1429-1471）
  段式拼接 `人物/组织/关键事件/叙事目标/场景/情绪`，按 `type == "organization"` 把组织从人物里分出来（:1447-1450）；限额 `characters[:8] / organization[:5] / key_points[:6] / goal[:1] / scenes[:2] / emotion[:1]`；整体 **`[:900]`**；空则用 `structure.summary[:200]`；再空回退 `chapter_outline[:500]`。
- 辅助函数：`_stringify_list_items`（:48-62，dict 取 `name/content/title`，默认 limit=5）、`_format_memory_query_section`（:65-70，用 `；` 连接）。

### 3.4 候选 → 筛选 → 兜底 → 注入 的四段流水线
`_select_memories_with_fallback`（:73-101）是唯一筛选器：

```
search_memories(limit=50, min_importance=0.0, chapter_range=(1, current-1))
  → 打印前 5 条候选的 similarity/distance/chapter/content[:60]   (:83-90)
  → filtered = [m for m if m.similarity > 0.6]                    (:92)
  → 若 filtered 非空 → 用它；否则 → memories[:5]（降级兜底）        (:93-101)
  → 最终注入 [:10]，每行 f"- (相关度:{similarity:.2f}) {content[:100]}" (:789-793 / :1324-1329)
```

---

## 4. token 预算与压缩 / 截断策略

### 4.1 结论：**MuMuAINovel 没有 token 预算，只有字符（rune）切片**
证据：全后端 grep `truncat|max_context|context_limit|token_limit|count_tokens|tiktoken` 仅命中
- 日志预览截断 `logger.py:27/31/38/76/146`、
- 错误消息截断 `ai_metrics.py:124/183`、
- `project_agent_tools.py:582` 的 `content_truncated = len(content) > 50000`、
- **唯一的 tokenizer 截断** `onnx_embedding.py:249`（embedding 输入，非 prompt 上下文）。

`OneToManyContext.get_total_context_length()`（:147-157）与 1-1 版本（:201-210）只做 `len(str)` 求和，写入 `context_stats`（:351-362 / :1343-1356）并打日志，**没有任何「超预算则裁剪」的分支**。

### 4.2 全部截断点（字符/rune 计数）

| 位置 | 上限 | 行号 |
|---|---|---|
| 最近章节摘要正文 | 180 | `:716` / `:1408` |
| 上一章摘要 | 300 | `:873` / `:875` / `:879` / `:1227` / `:1230` |
| 上一章关键事件条数 | 5 | `:889` |
| 最近章节关键事件条数 | 3 | `:719-726` / `:1414-1421` |
| 上一章正文（衔接锚点） | **无上限** | `:229` `ENDING_LENGTH=None` → `:854-858` |
| 角色 `appearance` / `personality` | 100 / 100 | `:552` / `:555`（1-N）、`:1612` / `:1615`（1-1） |
| 角色 `background` | 150 | `:558` / `:1618` |
| 角色 `current_state` | 150 | `:41` |
| 角色数量 | 10 | `:435` / `:1589` |
| 组织成员数 | 5（1-N 组织块）/ 无（1-1 组织块整串 100） | `:629` / `:1723` |
| 组织归属条数 | 2 | `:617` |
| 职业阶段体系 | **不截断**（逐阶全列） | `:646-655` / `:1747-1755` |
| 检索 query | 800（1-N）/ 900（1-1）/ 500（回退） | `:821` / `:1469` / `:823`、`:1471` |
| 注入记忆条数 × 单条 | 10 × 100 | `:789-792` / `:1324-1327` |
| 伏笔内容 | 100 / 80 | `:1017` / `:1035` |
| 超期伏笔条数 / 即将到期条数 | 3 / 3 | `:1031` / `:1052` |
| 章节摘要记忆正文 | 由 `extract_memories_from_analysis` 决定（`analysis.summary` 原样或 `content[:300]`） | `plot_analyzer.py:334-341` |

### 4.3 压缩策略
只有两种，且都不是「摘要式压缩」：
1. **切片**：上述固定长度截断；
2. **选段**：`recent_chapters_context` 优先用已有 `chapter_summary` 记忆，缺失时才回退 expansion_plan，再回退 `chapter.summary`（`chapter_context_service.py:713-742`）——即「**优先复用别处已产出的摘要，避免重复归纳**」。
   该方法 `_summarize_style`（:921-929，按 `STYLE_MAX_LENGTH` 压缩风格）与 `_build_story_skeleton`（:1063-1108，每 `SKELETON_SAMPLE_INTERVAL` 章采样 + 每章摘要 100 字符）已实现但**常量未定义且无调用方**（§10-D3）。

---

## 5. Embedding 方案（模型 / 维度 / 存储 / 索引 / 检索参数）

### 5.1 模型与运行
`backend/app/services/onnx_embedding.py`：

| 项 | 值 | 行号 |
|---|---|---|
| 模型 | `paraphrase-multilingual-MiniLM-L12-v2`（ONNX 导出，FP32） | :24 / `memory_service.py:45` |
| 分发源 | ModelScope `mumujie/paraphrase-multilingual-MiniLM-L12-v2-ONNX`，固定 revision `60750e200f336606cdd1ecbda9bb33fbf4d5b2a1` | :27-32 |
| `model.onnx` | 470,236,255 bytes；sha256 `e7515ed8…69cda5` | :34-37 |
| `tokenizer.json` | 9,081,518 bytes；sha256 `2c3387be…729076b8` | :38-41 |
| `embedding_config.json` | 182 bytes；sha256 `d9cfbb22…be01973c2` | :42-45 |
| 训练侧 | `tokenizers==0.22.2` 的 Rust `Tokenizer.from_file` | :244 |
| **输入上限** | `self.max_seq_length = config["max_seq_length"]` | :239（**值来自 config 文件，仓库内不存在该文件**） |
| **输出维度** | `self.embedding_dimension = config["embedding_dimension"]`，仅用于形状校验（:291-294） | :240 |
| 池化 | **强制 mean pooling 且禁止归一化**：`if config.get("pooling") != "mean" or config.get("normalize", False): raise ValueError("当前运行时仅支持不归一化的 Mean Pooling 模型")` | :241-242 |
| 推理 | `onnxruntime.InferenceSession(..., providers=["CPUExecutionProvider"])`，`ORT_ENABLE_ALL` | :252-258 |
| 池化实现 | `(token_embeddings * mask).sum(1) / clip(mask.sum(1), 1e-9, None)` | :285-288 |
| 模型查找/下载 | `resolve_model_dir()` 多候选 → 缺则下载（3 次重试、指数退避、`Range` 断点续传、`.part` + `os.replace` 原子落盘、大小 + sha256 双校验） | :82-220 |

**维度数值**：`embedding_dimension` 由未入库的 `embedding_config.json` 决定 → **未能从源码确认**。外部模型卡一致为 384 维（`paraphrase-multilingual-MiniLM-L12-v2`），但这是外部知识，不作为源码证据（§14）。

### 5.2 向量存储与索引
`backend/app/services/memory_service.py`：

| 项 | 值 | 行号 |
|---|---|---|
| 客户端 | `chromadb.PersistentClient(path="data/chroma_db")`（**相对路径**） | :32 / :36 |
| 隔离粒度 | 每 `(user_id, project_id)` 一个 collection | :51-88 |
| collection 命名 | `u_{sha256(user_id)[:8]}_p_{sha256(project_id)[:8]}`（≈30 字符，为满足 3–63 字符限制，注释见 :64-72） | :73-75 |
| collection metadata | `user_id / project_id / created_at` | :78-85 |
| **索引参数** | **未指定 `hnsw:space`**（全仓 grep `hnsw` 无命中）→ 使用 ChromaDB 默认距离 | — |
| 写入 | `collection.add(ids=[id], embeddings=[[...]], documents=[content], metadatas=[...])` | :139-144 |
| 相似度换算 | `similarity = 1 - distances[i]` | :285 |
| 查询 | `collection.query(query_embeddings=[q], n_results=limit, where=...)` | :271-275 |
| 过滤构造 | 0/1/多条件分别直传/单条/`{"$and": [...]}` | :250-268 |

入库向量是 §5.1 的**未归一化** mean-pooling 输出；collection 未声明空间 → 距离度量为 ChromaDB 默认（L2）。因此 `similarity = 1 - distance` 在数学上是 `1 - ‖a-b‖²`（平方 L2）。对未归一化向量，只要 `‖a-b‖² > 0.4`，`similarity < 0.6`，阈值即不命中 → 实际生效的几乎总是 `_select_memories_with_fallback` 的 **fallback 分支（top 5）**。默认距离度量非源码显式声明，且本机未安装 chromadb，**此数值行为的最终确认需在装有 chromadb==1.3.2 的环境实测**（§10-D2 / §14）。

### 5.3 检索参数总表

| 参数 | 值 | 位置 |
|---|---|---|
| 候选池 topK | **50** | `chapter_context_service.py:230` / `:1133`（`MEMORY_CANDIDATE_LIMIT`），作为 `limit` 传给 Chroma `n_results`（`memory_service.py:273`） |
| 相似度阈值（1-N） | **0.6** | `:234` `MEMORY_SIMILARITY_THRESHOLD`；经 `_select_memories_with_fallback` 用 `>` 比较（:92） |
| 相似度阈值（1-1） | **0.6**（硬编码字面量，不复用常量） | `:1317` |
| 无高分命中兜底 | **5 条** | `:232` / `:1135` `MEMORY_FALLBACK_COUNT` |
| 最终注入条数 | **10** | `:231` / `:1134` `MEMORY_CONTEXT_LIMIT` |
| 重要性下限（上下文路径） | **0.0**（即不过滤） | `:774` / `:1311` |
| 章节范围过滤 | `chapter_number ∈ [1, 当前章-1]` | `:775` / `:1312` |
| 重排（rerank） | **无** | — |
| 查询改写 / HyDE / 多查询 | **无** | — |
| 去重 / MMR | **无** | — |
| 归一化 | **禁止**（模型侧强制） | `onnx_embedding.py:241-242` |
| 向量库批量写入 | 逐条 `encode`（无 batch encode 调用，`memory_service.py:187`） | `batch_add_memories` 名为批量但 embedding 仍逐条算 |
| 陈旧过滤 | **无**（不存在「模型版本变化则重嵌」逻辑） | — |

### 5.4 写入 Chroma 的元数据（可被 `where` 过滤的字段）
`memory_service.py:120-137`：`memory_type`、`chapter_id`、`chapter_number`、`importance`（来自 `metadata.importance_score`）、`tags`(JSON 字符串)、`title[:200]`、`is_foreshadow`、`created_at`，条件性 `related_characters`(JSON 字符串)。

**被丢弃的元数据**：`plot_analyzer.py` 写入的 `keyword`、`text_position`、`text_length`、`strength`、`position_desc`、`reference_chapter`、`foreshadow_type` —— 只进关系库，不进向量库。因此**向量召回结果无法定位回正文位置**。

### 5.5 删除 / 更新
- `delete_chapter_memories`（:600-636）：按 `chapter_id` 过滤后 `collection.delete(ids=...)`。
- `delete_project_memories`（:638-674）：`client.delete_collection`，`"does not exist"` 视为成功。
- `update_memory`（:676-731）：内容变则重算 embedding 并 `collection.update`。
- `delete_foreshadow_memories`（:547-598）：按关键词在 `title+document` 上做**子串匹配**后删除 `memory_type='foreshadow'` 的条目；docstring 明确承认「当前记忆系统未持久化 `reference_foreshadow_id`/`foreshadow_id` 映射」（:556-558）——**伏笔与记忆之间没有外键**。
- `get_memory_stats`（:733-791）：`collection.get()` 全量拉取后 Python 侧聚合（`:752`），无分页。

---

## 6. 记忆分层模型与生命周期

### 6.1 实际分层（按写入规则，非声明式）
| 层 | 类型 | 粒度 | 生成时机 |
|---|---|---|---|
| 章节层 | `chapter_summary`（0.6 固定） | 1 章 1 条 | 每次章节分析 |
| 情节层 | `plot_point`（LLM 重要性 ≥0.6）、`plot_point`（冲突 level ≥7） | 1 章多条 | 每次章节分析 |
| 钩子层 | `hook`（strength ≥6） | 1 章 ≤N 条 | 每次章节分析 |
| 角色层 | `character_event`（0.7 固定） | 1 状态变化 1 条 | 每次章节分析 |
| 伏笔层 | `foreshadow`（`is_foreshadow` 1/2） | 1 伏笔 1 条 | 每次章节分析 |
| 章节正文 | 不入向量库，仅在 1-1/1-N 里作为 `continuation_point` **整篇**注入 | 1 章 1 份 | 生成时读 |

### 6.2 生命周期：**只有「删除 + 重写」，无衰减/遗忘/晋升/固化**
- 重分析 = 删本章全部记忆 + 重新生成（`api/chapters.py:1226-1234`、`api/memories.py:153-159`）。
- 清空章节内容 = 级联删除（`api/chapters.py:375-409`）。
- 删除项目 = 删 collection（`memory_service.py:638-674` + `import_export_service` 相关路径）。
- **不存在**：importance 衰减、访问频率提升、LRU 淘汰、短期→长期晋升、冲突记忆合并、过期归档。

### 6.3 检索时无任何分层权重
`search_memories` 只按 HNSW 距离排序；`importance` 只在 `where` 里做「下限过滤」（上下文路径传 0.0 即禁用，`chapter_context_service.py:774`）。同一 `importance` 不参与打分。

对照：gaea 已有完整生命周期设施（`internal/gaea/memory/decay.go`、`forget.go`、`promote.go`、`distill.go`、`eventlog.go`、`mem_graph_meta`、`SchemaV19 facts.pinned 固化态`，`internal/gaea/db/schema.go:463-464`）。

---

## 7. Prompt 原文摘录（上下文注入位置）

### 7.1 1-N 模式（第 2 章及以后）`CHAPTER_GENERATION_ONE_TO_MANY_NEXT`
`prompt_service.py:769-863`，关键区块原文：

```
<outline priority="P0">
【本章大纲 - 必须遵循】
{chapter_outline}
</outline>

<recent_context priority="P1">
【最近章节规划 - 故事脉络参考】
{recent_chapters_context}
</recent_context>

<continuation priority="P0">
【衔接锚点 - 必须承接】
上一章完整正文：
「{continuation_point}」

【🔴 上一章已完成剧情（禁止重复！）】
{previous_chapter_summary}

⚠️ 严重警告：
1. 上述"已完成剧情"和"衔接锚点"是**已经写过的**内容
2. 本章必须推进到**新的情节点**，绝对不能重新叙述已经发生的事件
3. 本章应承接上一章最后的情境继续推进，不要复述上一章完整正文
4. 如果上一章以对话或场景结束，请从结束后的动作、反应或场景转换开始
</continuation>

<foreshadow_reminders priority="P1">
【🎯 伏笔提醒 - 需关注】
{foreshadow_reminders}
</foreshadow_reminders>

<memory priority="P2">
【相关记忆 - 参考】
{relevant_memories}
</memory>
```
（行号：`outline` :782-785、`recent_context` :787-790、`continuation` :792-805、`characters` :807-815、`careers` :817-820、`foreshadow_reminders` :822-825、`memory` :827-830、`constraints` :832-855，其中「反重复特别指令」:842-845、禁止事项含「重复叙述上一章已发生的事件」:852。）

### 7.2 1-1 模式（第 2 章及以后）`CHAPTER_GENERATION_ONE_TO_ONE_NEXT`
`prompt_service.py:690-766`：`<previous_chapter_summary priority="P1">`（:708-711）、`<recent_context priority="P1">`（:713-716）、`<previous_chapter priority="P1">【上一章完整正文】`（:718-721）、`<memory priority="P2">`（:738-741）。

### 7.3 模板参数登记表（上下文相关的键）
`prompt_service.py:2942-2964`：
- `CHAPTER_GENERATION_ONE_TO_MANY_NEXT.parameters` = `[project_title, genre, chapter_number, chapter_title, chapter_outline, target_word_count, narrative_perspective, characters_info, continuation_point, foreshadow_reminders, relevant_memories, story_skeleton, previous_chapter_summary]`（:2946-2948）——**`story_skeleton` 在模板正文中无对应占位符**（§10-D9）。
- `CHAPTER_GENERATION_ONE_TO_ONE_NEXT.parameters` 含 `previous_chapter_content / chapter_careers / foreshadow_reminders / relevant_memories`（:2961-2963）。

### 7.4 上下文 → 模板的实际绑定
`api/chapters.py:1639-1728`：1-1 走 `CHAPTER_GENERATION_ONE_TO_ONE[_NEXT]`（:1643 / :1664），1-N 走 `CHAPTER_GENERATION_ONE_TO_MANY[_NEXT]`（:1691 / :1713）；空值用中文兜底串替换：`'暂无角色信息'`、`'暂无职业信息'`、`'暂无需要关注的伏笔'`、`'暂无相关记忆'`（:1656-1659 等）。随后按 `WritingStyleManager.apply_style_to_prompt` 追加写作风格（:1730-1734）。

---

## 8. 后端 API 契约清单（`backend/app/api/memories.py`，prefix `/api/memories`，:21）

| 方法 | 路径 | 查询/请求参数 | 返回 | 行号 |
|---|---|---|---|---|
| POST | `/projects/{project_id}/analyze-chapter/{chapter_id}` | 无 body | `{success, message, analysis, memories_count, foreshadow_stats, entity_changes{carrers,character_states,organization_states}}` | :24-289 |
| GET | `/projects/{project_id}/memories` | `memory_type?`, `chapter_id?`, `limit=50` | `{success, memories[to_dict], total}` | :292-329 |
| GET | `/projects/{project_id}/analysis/{chapter_id}` | — | `{success, analysis}` / 404 `该章节还未进行分析` | :332-368 |
| POST | `/projects/{project_id}/search` | `query`(必填), `memory_types?`, `limit=10`, `min_importance=0.0` | `{success, query, memories[含 similarity/distance], total}` | :371-406 |
| GET | `/projects/{project_id}/foreshadows` | `current_chapter`(必填) | `{success, foreshadows, total}` | :409-438 |
| GET | `/projects/{project_id}/stats` | — | `{success, stats{total_count, by_type, by_chapter, foreshadow_count, foreshadow_resolved}}` | :441-466 |
| DELETE | `/projects/{project_id}/chapters/{chapter_id}/memories` | — | `{success, message}` | :469-514 |

全部接口前置 `verify_project_access(project_id, user_id, db)`（:40 等），`user_id` 取自 `request.state.user_id`（:37 等）——**依赖 middleware 注入的会话用户**（多用户服务端假设，见 §13.2）。

另：`api/chapters.py` 提供批次分析 `POST /chapters/project/{projectId}/analysis/analyze-unanalyzed` 与 `POST /chapters/project/{projectId}/analysis/statuses`（前端 `services/api.ts:785` / `:789-790` 调用）。

---

## 9. 前端交互清单

- **不存在独立的记忆/向量管理页面**：`frontend/src/pages/*.tsx` 共 28 个页面，无 `Memory*`（`ls frontend/src/pages`）。
- 记忆只出现在两处：
  1. `frontend/src/components/ChapterAnalysis.tsx:708-716`：分析结果页的「记忆 (N)」Tab，用 `dataSource={memories}` 表格展示**当前章**提取到的记忆（数据来自 `analysis.memories`，`types/index.ts:875` 的 `StoryMemory[]`）。
  2. `frontend/src/pages/ProjectList.tsx:355/368`：项目导出选项 `include_memories`（`services/api.ts:409-410` 定义 `include_memories?` / `include_plot_analysis?`）；`ProjectList.tsx:1002` 展示导入校验统计里的「故事记忆」条数。
- **前端从不调用 `/api/memories/*`**：`services/api.ts` 中 grep `memor` 无 `/memories` 路径命中 → §8 的 7 个接口（含语义搜索 `POST /search`、统计 `/stats`、伏笔列表 `/foreshadows`）**在 UI 上没有入口**。

---

## 10. 源码级缺陷 / 风险清单（可被逐条复核）

| ID | 严重度 | 问题 | 证据 | 确定性 |
|---|---|---|---|---|
| D1 | high | 重新分析章节时**只清关系库、不清向量库**；路径 A 的记忆 id 是确定性的 → 重复 `add` 同 id，`batch_add_memories` 吞异常返回 0 → 该章向量整体不更新 | `api/chapters.py:1225-1234`（只删 `StoryMemory`）对比 `api/memories.py:161-169`（调 `delete_chapter_memories`）；id 规则 `chapters.py:1239`；`memory_service.py:205-217` | 代码路径确定；异常类型未实测 |
| D2 | high | `similarity = 1 - distance` 用于**未归一化**向量的 L2 空间 → 阈值 0.6 在数学上几乎不可达，`MEMORY_SIMILARITY_THRESHOLD` 形同虚设，实际总是走 fallback top-5 | `memory_service.py:285` + `onnx_embedding.py:241-242` + 无 `hnsw:space` | 公式与配置确定；未实测数值 |
| D3 | medium | 3 个方法引用了**从未定义**的类常量，一旦调用即 `AttributeError`；且当前无任何调用方（死代码） | `chapter_context_service.py:921-929`（`STYLE_MAX_LENGTH`）、`:931-961`（`MEMORY_IMPORTANCE_THRESHOLD`）、`:1063-1108`（`SKELETON_SAMPLE_INTERVAL`）；全仓 grep 这 3 个常量仅有定义处无赋值 | 确定 |
| D4 | medium | `build_context_for_generation`（「核心功能：结合多种检索策略」）**无任何调用方** → 其描述的 recent+relevant+foreshadow+character+plot_point 五路融合并未在生产生效；连带 `get_recent_memories` 仅被它调用 | `memory_service.py:413-513`；grep `build_context_for_generation` 仅定义处 | 确定 |
| D5 | medium | 主写入路径不写 `vector_id`（模型定义为 unique），`api/memories.py` 路径写 → 同一表两种数据形态 | `chapters.py:1254-1269` vs `memories.py:185`；`models/memory.py:52` | 确定 |
| D6 | medium | 导入项目时记忆**只重建关系库、不重建向量库** → 导入后语义检索在导入项目上永为空 | `import_export_service.py:1207-1253`（无 `memory_service` 调用） | 确定 |
| D7 | low | `memory_type` 注释枚举缺 `chapter_summary`，而该类型是每章必有、且被 `recent_chapters_context` 直接查询的类型 | `models/memory.py:17-26` vs `plot_analyzer.py:346`、`chapter_context_service.py:698/867/1091/1221/1391` | 确定 |
| D8 | low | ChromaDB 数据目录 `"data/chroma_db"` 是**相对路径** → 依赖进程 CWD（桌面/容器化部署易漂移） | `memory_service.py:32-36` | 确定 |
| D9 | low | 模板参数表声明 `story_skeleton`，但 1-N 模板正文无该占位符，且其构建器方法因 D3 不可用 | `prompt_service.py:2948`；模板 `:769-863` 无 `{story_skeleton}` | 确定 |
| D10 | low | 优先级标注自相矛盾：`OneToManyContext` docstring 与字段注释说伏笔是 P2，模板里是 P1 | `chapter_context_service.py:112/142` vs `prompt_service.py:822` | 确定 |
| D11 | high | **整章上章正文无限量注入**（`ENDING_LENGTH=None`），叠加无 token 预算 → 长章节（如 8000 字/章）时 1-N 与 1-1 的 prompt 体积不可控 | `chapter_context_service.py:229/854-860`（1-N）、`:1211-1214`（1-1，连兜底参数都没有）；§4.1 无预算证据 | 确定 |
| D12 | low | `previous_chapter_events`（最多 5 条关键事件）被组装但**从不进入任何模板**（`prompt_service.py` 全文无该字样，连参数登记表都没有） | `chapter_context_service.py:312/320/884-889`；prompt_service 无 `previous_chapter_events` | 确定 |
| D13 | medium | `StoryMemory.related_characters` 注释为角色 **ID** 列表，生产者写入的是角色 **名**（`char_state.get('character_name')`） | `models/memory.py:34` vs `plot_analyzer.py:440/450` | 确定 |
| D14 | low | 「伏笔记忆」有两套互不一致的识别方式：检索用 `is_foreshadow==1`，删除用 `memory_type=='foreshadow'` | `memory_service.py:380-388` vs `:574` | 确定 |
| D15 | medium | 无任何 rerank；候选池 50 与注入 10 之间的排序完全依赖 ANN 距离，且 D2 使阈值失效 → 实际是「最近 5 条」而非「最相关 5 条」 | `chapter_context_service.py:73-101`、`memory_service.py:271-287` | 确定（依赖 D2） |
| D16 | low | 1-1 模式角色关系查询是 N+1（每角色一次 `select`），1-N 已做批量（`char_rels_map`） | `chapter_context_service.py:1670-1681` vs `:438-448` | 确定 |
| D17 | low | `_find_text_position` 的「去标点后反向映射」注释为「简化处理」，返回的 `position` 在原文中可能不对齐 | `plot_analyzer.py:508-516` | 确定 |

---

## 11. 与 `internal/novelcontext/novelcontext.go` 的逐点对比

### 11.1 定位差异（先对齐概念）
- **MuMuAINovel**：章节级上下文 = `OneToManyContext` / `OneToOneContext` 两个 dataclass（`chapter_context_service.py:104-210`），字段扁平，注入到 `<outline>/<continuation>/<memory>/<foreshadow_reminders>` 等 XML 标签（`prompt_service.py:769-863`）。
- **gaea**：场景级上下文 = `SceneBible`（`novelcontext.go:48-58`），**按 POV 角色做视角掩码**，渲染为 markdown 区段（`Render`，:119-170），注入上限 2200 rune（`create_chapter_handler.go:575`），由 `BuildSceneBibleFromChapter`（`novelcontext.go:111-114`）在 `CreateChapter` 中调用（`create_chapter_handler.go:110-114`）。

### 11.2 差距清单（逐点）

| # | 能力点 | MuMuAINovel（文件:行号） | gaea `novelcontext.go`（行号） | 差距判定 |
|---|---|---|---|---|
| 1 | 检索方式 | **语义向量检索**（ChromaDB HNSW，模型 ONNX 本地推理） | **纯规则/子串/ID 命中**（`collectSceneEntities`:284-338 用 `containsFold` 扫正文；`db.GetByName`） | gaea 缺语义召回；MuMu 缺确定性 |
| 2 | 候选池 / topK | 50 候选 → 阈值 0.6 → 10 注入 → 兜底 5（`:230-234`） | 无候选池概念；`charItemsMax=4`、`charKnownByMax=4` 等为渲染截断（`:43-44`） | gaea 无可调检索参数 |
| 3 | 时间维度 | **最近 10 章摘要窗口**（`:1375`）+ 上一章完整正文 + 300 字摘要 | `readPrevSummary` 只取**上一章**摘要（`:844-861`），`timeAnchorBudget=120` 截断 | gaea 缺「最近 N 章」窗口 |
| 4 | 上一章正文 | 整篇注入（无上限，1-N `:854-860`；1-1 `:1211-1214`） | 完全不注入正文，只注入摘要（`:255`，`timeAnchorBudget/2` 截断） | **gaea 更优**（受控） |
| 5 | 预算模型 | **无 token/字符总预算**，只有逐字段切片（§4.2） | 各区段独立预算 + `Render(maxRunes)` 全局兜底 + `JoinWithBudget(ctxBudgetTotal=4000)`（`:27-45`、`:119-170`、`create_chapter_handler.go:568-576/619-635`） | **gaea 更优**，可直接反哺 MuMu |
| 6 | 确定性 | 依赖 ANN 排序（同查询同库大体稳定但非保证） | **显式排序保证确定性**：`collectSceneEntities` 按 ID 排序（`:336`）、`buildSceneChars` 按名排序（`:414`）、`sortedKeys`（`:834-841`） | gaea 更利于测试 |
| 7 | POV 视角掩码 | **无此概念**（角色块全量给出所有关系/组织/状态，`chapter_context_service.py:599-630`） | `buildPOVMask`（`:496-533`）+ `povKnowsFact` 四规则（`:547-569`）+ `isObservableKey`（`:573-581`）+ `HiddenFacts` 生成约束区段（`:146-149`） | **gaea 独有优势**，MuMu 无 |
| 8 | 角色块渲染 | 10 角色 × 12 类字段，关系全量拼接（`:534-635`） | `SceneChar` 6 字段 + `characterLineMax=140` 行长截断（`:60-68`、`:782-813`） | 各有取舍；gaea 更省 token |
| 9 | 职业/等级体系 | 独立 `chapter_careers` 区段，**完整阶段体系逐阶展开**（`:637-666`、`:1732-1782`） | 无职业概念（`role_type` 只在 `SceneChar`） | MuMu 有该能力可借鉴（与角色域 t5 重叠） |
| 10 | 伏笔注入 | 三档：必回收 / 超期 / 未来 3 章（`:985-1061`），但伏笔↔记忆无外键（D14） | `buildForeshadows` 取 `planted/hinted` 一行一条（`:221-241`，`foreshadowLineMax=90`）；**无「必须本章回收」档** | 各有所长：MuMu 有到期推理，gaea 有确定性 ID（`types.Foreshadow.ID = {type}_{chapter}_{content_hash}`，`types/types.go:193`） |
| 11 | 文风/风格 | 不进入上下文对象，生成时另路追加（`api/chapters.py:1730-1734`） | `buildStyle` 作为圣经区段（`:199-209`，`styleBudget=240`） | gaea 更集中 |
| 12 | 世界观 | 不在上下文对象内（大纲/角色自带设定） | `buildSetting`（`:175-196`，`settingBudget=240`） | gaea 更集中 |
| 13 | 故事主线 | 无 | `buildThread`（`:212-218`，`threadBudget=100`） | gaea 独有 |
| 14 | 实体子图 | 无「实体」抽象，直接 SQL join 角色/关系/组织/职业表 | `graph.EntityDB` + `collectSceneEntities` 四源合并（`:284-338`） | 架构级差异（gaea 有图底座） |
| 15 | 记忆生命周期 | 无衰减/遗忘/晋升（§6.2） | `internal/gaea/memory/{decay,forget,promote,distill,eventlog}.go` + `SchemaV19 facts.pinned`（`schema.go:463-464`） | **gaea 领先**，MuMu 无可借鉴 |
| 16 | 记忆检索实现 | 外部向量库（ChromaDB 进程内） | 已有两套本地实现：① `internal/memory/memory.go` BM25（k1=1.5, b=0.75，`:41-43`）+ **token 预算注入** `InjectIntoContext(maxMemories=5, maxTokens=3000)`（`:248-271`）；② `internal/gaea/semantic/semantic.go` 稠密向量 + `Cosine` + `MinCosine=0.1`（`:22`） | gaea 已有原语，缺「小说记忆」这个 kind |
| 17 | 兜底策略 | 每区段 try/except → 返回 None（如 `:748-750`、`:796-798`） | 每读取失败静默降级为空、绝不中断（`:73` 契约注释） | 同等 |
| 18 | 预算与截断的单位 | 字符（Python `len`，等价 rune） | rune（`util.Truncate`，`util.go:100-106`） | 一致，可直接换算 |
| 19 | 摘要复用 | **优先复用 `chapter_summary` 记忆**，缺失才回退大纲/章节字段（`:713-742`） | 只用 `chapter-summary.json`（`readPrevSummary`） | 一致思路，MuMu 回退链更长 |
| 20 | 历史断档容忍 | SQL `chapter_number < N ORDER BY DESC`（`:676-681`）→ 天然容忍断档 | `readPrevSummary` 先试 `chapterNum-1`，失败回退「最后一个非空摘要」（`:844-861`） | 同等 |

### 11.3 gaea 现存一致性实现里的两处自身缺口（对照后才显形）
| 缺口 | 证据 | 影响 |
|---|---|---|
| `prevSummary` 无总预算 | `create_chapter_handler.go:49-61`：遍历**所有** `chapterNum < limitChapter` 的大纲节点，每章 `util.Truncate(n.Summary, 200)`，**无累计上限** | 200 章即 ~40k rune 的 prompt 前缀；`joinWithBudget(ctxBudgetTotal)` 只约束伏笔+世界观区段（`:616`），不覆盖 `prev_summary` |
| `mem_graph_nodes.embedding` 恒 NULL | `internal/gaea/db/schema.go:415-416` 注释明示「本刀恒 NULL——5.2 上下文编译的语义检索从这里起步」 | 小说记忆的向量落位尚未启用；`semantic_vectors` 表（`schema.go:138-147`）现成可用但未被 novel 域使用（grep novel 域无 `semantic.` 调用） |

---

## 12. gaea 侧改造点（规格级，不写生产代码）

### 12.1 建议新增（Go 包与文件级）
| 目标 | 建议落位 | 依据 |
|---|---|---|
| 小说记忆实体 | `internal/types`：新增 `StoryMemory`（对齐 `models/memory.py:8-77` 的字段子集：`ID/ChapterNum/ChapterID/Type/Title/Content/Importance/IsForeshadow/Tags/RelatedCharacters/Position/TextLen`） | 字段级对齐 §1.1 |
| 记忆落盘 | `internal/project`：`memories/MMM-<n>-memory.json` 或复用 `chapters/NNN-summary.json` 扩展（**建议独立文件，避免与章节摘要耦合**） | gaea 现有 `chapters/NNN-summary.json`（`types/types.go:169-178`）只承载摘要 |
| 语义召回 | `internal/novelcontext`：新增 `RecallStoryMemories(ctx, pm, query, topK, threshold) []StoryMemoryHit`，内部走 `internal/gaea/semantic.Store`，`kind = "story_memory"` | `semantic.go:149-217` 现成；`schema.go:138-147` 现成表 |
| 上下文预算统一 | `internal/novelcontext`：把 `prevSummary` 的**200×N 无界拼接**改为「预算内最近 N 章」（建议 N=10，每章 180 rune，对齐 MuMu `:1375/:1408`），并把 `sceneBudget` 体系扩展出 `memoryBudget` / `recentBudget` | §11.3 缺口 1 + `novelcontext.go:27-45` |
| 记忆注入区段 | `SceneBible` 增字段 `Memories []string`，`Render` 增「## 相关记忆（按相关度）」区段，`memoryBudget` 建议 300–500 rune | `novelcontext.go:48-58` / `:119-170` 渲染框架 |
| 检索参数 | 常量：`storyMemoryTopK = 8`、`storyMemoryThreshold = 0.35`（**比 MuMu 的 0.6 低，因为 gaea 用归一化余弦**）、`storyMemoryFallback = 3` | MuMu 参数 §5.3；gaea 用 `retrieval.Cosine`（已归一化语义），阈值语义与 L2 不同，**不可照抄 0.6** |
| 索引维护 | 复用 `semantic.Store.Ensure`（增量 + 正文快照比对，`semantic.go:97-146`）与 `Stale`（`:254-288`）；章节被重写时先 `Remove(kind, id)` 再 `Ensure` | 直接规避 MuMu 的 D1/D6 |

### 12.2 需要新增的索引 / 键
- `story_memory` 主键：**确定性** `id = fmt.Sprintf("%03d-%s-%d", chapterNum, mtype, ordinal)`（对齐 MuMu `chapters.py:1239` 的确定性思路，但避免其重写不清理的缺陷）。
- 检索过滤维度（对齐 MuMu `where` 能力，`memory_service.py:250-268`）：`chapter_num <= current-1`、`importance >= min`、`type in [...]`。gaea 侧 `semantic_vectors` 只有 `(kind,id,vec,doc,updated_at)`，**过滤条件需在 Go 侧内存过滤**（`SearchReady` 全表拉取后打分，`semantic.go:178-198`）→ 若要支持过滤，建议扩展：`semantic_vectors` 增加 `meta TEXT`（JSON）列，或把可过滤字段编码进 `kind`（如 `story_memory|type=plot_point`）。

### 12.3 不建议照搬（单机桌面不适配）
| MuMu 设计 | 不适配原因 | 替代 |
|---|---|---|
| ChromaDB 1.3.2 进程内向量库（`memory_service.py:36`） | 新增 Python 侧重依赖（onnxruntime + chromadb）；gaea 已是 Go 单机 + `Hephaestus.db`；且其 collection 命名/相对路径为多用户服务端设计 | 复用 `semantic_vectors`（`schema.go:138-147`）+ `retrieval.Cosine` |
| `u_{user_hash}_p_{project_hash}` 双哈希 collection 隔离（`:73-75`） | gaea 是单机桌面，无 user 维度；项目隔离已由 `project.Manager.Dir` 承担 | 直接以 `project` 目录/`space` 列隔离 |
| `verify_project_access` + `request.state.user_id`（`api/memories.py:37/40`） | 依赖多用户 middleware（OAuth/JWT） | gaea 无 auth，直接走 project |
| `data/chroma_db` 相对路径（`:32`） | 桌面应用 CWD 不稳定 | 绝对路径：项目目录 / 用户数据目录 |
| `transformer/` 训练期依赖 | 只在构建期 | 若走 `semantic_vectors`，embedding 由 Herdsman（bge-m3）提供，零 Python 依赖 |
| 470 MB ONNX FP32 模型随包分发（`onnx_embedding.py:34-37`） | 桌面安装包体积代价大 | 复用 Herdsman 本地 `bge-m3`（`internal/gaea/retrieval/embed.go:84` 注释「维度 1024 for bge-m3」）；或 `qwen3-embedding`（`internal/app/gaea_cost_rerank.go:66`） |

### 12.4 可直接借鉴（与单机架构无冲突）
1. **结构化查询构造**（`chapter_context_service.py:800-823` / `:1429-1471`）：把大纲/扩展计划拆成「人物/组织/关键事件/叙事目标/场景/情绪」六段短语拼接，并使用**每种字段的独立条数限额**。可直接移植为 Go 的 `buildMemoryQuery(outlineNode, scene) string`。
2. **候选→阈值→兜底→topN 四段流水线**（`:73-101`）：尤其是「无高分命中时保留少量 top 结果」这一兜底，正好弥补 gaea 关键词检索「一个都不命中就空手」的问题。
3. **阈值+兜底的双参数化**（阈值 0.6 / 兜底 5 / 注入 10）：把它改成 gaea 的 `threshold/fallback/topN` 三常量即可。
4. **记忆类型与重要性赋值的显式规则表**（`plot_analyzer.py:329-479`）：`hook.strength >= 6`、`plot_point.importance >= 0.6`、`conflict.level >= 7` 三个门限 + 每类固定重要性（`chapter_summary=0.6`、`character_event=0.7`）——是一份可直接落成 Go 常量表的**入库门槛规格**。
5. **章节摘要复用回退链**（`:713-742`）：`StoryMemory(chapter_summary)` → `expansion_plan.plot_summary` → `chapter.summary` → 无。gaea 目前只读 `chapter-summary.json` 一个来源，可补两级回退。
6. **反重复的 prompt 约束写法**（`prompt_service.py:792-805` + `:842-845`）：把「衔接锚点」与「上一章已完成剧情」并列、附禁止重复清单。gaea `novelcontext` 目前只给 `时间锚点`，可在渲染时补同款约束段。
7. **模型分发工程细节**（`onnx_embedding.py:82-182`）：`.part` + `Range` 断点续传 + 大小/sha256 双校验 + `os.replace` 原子提交 + `FileLock`。若 gaea 未来要分发本地 embedding/rerank 模型，这套下载器值得照搬（纯 Python 逻辑，Go 易平移）。
8. **记忆写入的确定性 id**（`chapters.py:1239`）：值得保留思路（幂等重写的前提），但必须配 `Remove` 先删（规避 D1）。

---

## 13. gaea 现状（已读证据）

- `internal/novelcontext/novelcontext.go` 876 行：`SceneBible` 编译器，**零 LLM、零网络、纯本地**（:8 契约），预算常量 :27-45，`Render` :119-170。
- 调用点：`internal/app/create_chapter_handler.go:110-114`（`CreateChapter` 主链路）、`internal/app/scene_gen_handler.go:152-153`（场景生成）。预算 `ctxSceneBibleBudget = 2200`（`create_chapter_handler.go:575`）。
- 章节生成上下文拼装：`create_chapter_handler.go:44-61`（`prevSummary`，**无总预算**）、`:73-84`（模板 + `buildChapterContextSections`），`buildChapterContextSections` :607-617，`joinWithBudget` :619-635，预算常量 :568-576。
- 记忆检索原语：`internal/memory/memory.go`（BM25，`NewIndex` :38，`BuildFromProject` :83，`Search` :179，`InjectIntoContext` :248-271，`Memory.Tokens` 由 `util.EstimateTokens` :155-168 估算）；调用点 `internal/app/context_handler.go:24`、`:55`、`:210`（`BuildRichContext(systemPrompt, userText)` 用 `maxMemories=5, maxTokens=3000`，:212）。
- 向量原语：`internal/gaea/retrieval/embed.go`（`Embedder` :17、`NewEmbedder` :30、`Embed` :84、`Cosine` :126-140）；`internal/gaea/semantic/semantic.go`（`EmbedBatch=64` :20、`MinCosine=0.1` :22、`Ensure` :97、`SearchReady` :174、`SearchMany` :157）。
- 表结构：`semantic_vectors(kind,id,vec,doc,updated_at)` PK`(kind,id)`（`internal/gaea/db/schema.go:138-147`）；`mem_graph_nodes.embedding BLOB` 占位恒 NULL（:437-450，注释 :415-416）。
- 生命周期设施：`internal/gaea/memory/{decay,forget,promote,distill,eventlog}.go`；注入预算 `profileBudget=600` / `recallBudget=800`（`recall.go:13-16`）；陈旧提示阈值 `staleCitedDays=90`（`:205`）。
- 无关域对照：`internal/whisper/vector_store.go`（TF-IDF + 可选稠密）、`internal/gaea/search/search.go`（TF-IDF + bigram 余弦）——与本域无引用关系。

---

## 14. 未能从源码确认的清单（禁止臆断，显式列出）

1. **embedding 维度与 `max_seq_length` 的具体数值**：由未入库的 `embedding_config.json` 决定（`onnx_embedding.py:236-240`，文件在 `MODEL_FILES` 中声明但仓库内不存在，需运行期从 ModelScope 下载）。外部模型卡普遍为 384 维 / 128 token，**非源码证据**。
2. **ChromaDB collection 的实际距离度量**：代码未设 `hnsw:space`（全仓 grep `hnsw` 无命中），默认值属 ChromaDB 实现细节；本机未安装 `chromadb`（`import chromadb` 失败），**未能实测** D2 的数值后果。
3. **重复 id `add` 的具体异常类型**：`batch_add_memories` 用宽 `except` 吞掉（`memory_service.py:215-217`），无法从源码判断 ChromaDB 1.3.2 抛 `DuplicateIDError` 还是 `UniqueConstraintError`；**未能实测**。
4. **`RTCO` 缩写的定义**：仓库内仅出现命名引用（`chapter_context_service.py:1/109/165`、`prompt_service.py` 多处注释），**无定义处**；P0/P1/P2 的实际语义只能由代码推断。
5. **`foreshadow_service` 内部实现**：本报告只引用其被调用签名（`get_must_resolve_foreshadows` / `get_overdue_foreshadows` / `get_pending_resolve_foreshadows`，`chapter_context_service.py:1006/1023/1039`），未展开该文件（属伏笔域 t1）。
6. **`ANALYSIS_TASK_TIMEOUT_SECONDS`**：已定位为 `api/chapters.py:70`（=600 秒）。
7. **`StoryMemory.content` 注释声明的「100-500字」是否有校验**：未在写入路径发现长度校验代码。
8. **生产环境是否真的启用 1-1 模式**：`outline_mode` 的来源与默认值未追（属大纲/设置域）。
9. **`previous_chapter_events` 是否曾被某个历史模板消费**：当前 `prompt_service.py` 全文无 `previous_chapter_events` 字样（grep 仅命中 `story_skeleton`，:2948），**当前版本确定未被消费**。

---

## 15. 一句话结论

MuMuAINovel 的长程一致性是「**章节生成后自动分析 → 结构化抽取记忆 → 双库落盘 → 下次生成时向量召回 + 最近 10 章摘要 + 上一章整篇正文 + 三档伏笔提醒**」的闭环，工程完整度高于 gaea 现有 `novelcontext`（多出语义召回、最近 N 章窗口、伏笔到期推理、结构化查询构造）；但其**没有 token/字符总预算**（上章整篇无界注入）、**相似度阈值因未归一化向量与 L2 距离而失效**、**重分析不清向量库**、**导库不重建向量**。gaea 侧的正确路径不是照搬 ChromaDB，而是把 MuMu 的「召回参数化 + 结构化 query + 入库规则表 + 章节摘要回退链」移植到已有 `internal/gaea/semantic` + `internal/memory` 原语上，并保留 gaea 更强的一侧：**全局预算截断（`Render(maxRunes)`）与 POV 视角掩码**。
