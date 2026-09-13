# 04 · 情节分析与扩展 / 一键重写域 — 源码级蒸馏与 gaea 落地规格

| 项 | 值 |
|---|---|
| 能力域 | 情节分析与扩展 / 一键重写 |
| 事实源 | `clones/MuMuAINovel` @ commit `600be7038539e4dd0568b63e6e534fdcfaf91687`（branch `main`, tag `v1.5.5`, `git status` clean） |
| 引用格式 | `clones/MuMuAINovel@600be703:<相对路径>:<起-L止>`（未带 commit 前缀的引用视为不可核验） |
| 行号来源 | 全部来自 `read` / `grep` 工具真实输出（未使用 `Measure-Object -Line`） |
| 提出者 / 任务 | researcher-plot / t4（mumu-distill） |
| 本域文件清单 | `plot_analyzer.py`、`plot_expansion_service.py`、`chapter_regenerator.py`、`api/chapters.py`（分析+重写段）、`api/polish.py`、`schemas/regeneration.py`、`schemas/chapter.py`、`models/analysis_task.py`、`models/regeneration_task.py`、`models/memory.py`、`services/json_helper.py`、`skills/story-deslop/**`、`prompt_service.py` 中 `PLOT_ANALYSIS` / `OUTLINE_EXPAND_*` / `CHAPTER_REGENERATION_SYSTEM` / `PARTIAL_REGENERATE` |
| 本地对照 | `internal/analysis/analysis.go`、`internal/chapter/chapter.go`、`internal/novelstyle/rewrite.go`、`internal/app/novel_{deslop,llm_rewrite,fingerprint}_handler.go`、`internal/app/plot_branch_handler.go`、`prompts/analysis-chapter.json`、`prompts/outline-expand.json` |
| 交付性质 | 只做蒸馏与规格，**本轮不写生产代码** |

---

## 1. 结论速览（TL;DR）

MuMu 本域的真实资产不是"分析一下章节"，而是**四套可复用的工程机制**：

| # | 机制 | 一句话本质 | MuMu 实现锚点 |
|---|---|---|---|
| M1 | **单次 LLM 调用产出"多维评分 + 可定位标注"的分析结果** | 一个 Prompt 同时产出 9 个分析维度，每个钩子/伏笔/情节点必须回填从原文**逐字复制 8-25 字**的 `keyword`，用于把分析结论钉回正文坐标 | `prompt_service.py` L1047-L1395；`plot_analyzer.py` L32-L194 |
| M2 | **异步任务化 + 三重超时保障 + 陈旧任务原子恢复** | 分析不再是请求-响应，而是 `AnalysisTask` 状态机（pending→running→completed/failed），带"硬超时 / 终态兜底 / 陈旧恢复"三层 | `api/chapters.py` L70-L71、L98-L136、L886-L922、L2449-L2487 |
| M3 | **分批展开 + 跨批差异化约束** | 大纲节点展开 N 章时按 `batch_size=5` 分批，后批注入"已生成章节摘要 + 已用关键事件清单 + 差异化强制要求" | `plot_expansion_service.py` L154-L289（尤其 L204-L237） |
| M4 | **重写三通道：整章重写 / 选段局部重写 / 去 AI 味** | 整章重写有版本快照与 diff；局部重写走"字符区间 + 长度模式 + 前后文衔接"；去 AI 味是独立的词表资产 + 6 道门禁 | `chapter_regenerator.py` + `api/chapters.py` L4449/L4922/L5178 + `api/polish.py` + `skills/story-deslop/**` |

**gaea 的关键缺口**（详见 §8）：gaea 已有确定性去 AI 味（`novelstyle.DeSlopRewrite`）与"打分定位 → 受限 LLM 句级重写"的安全闸闭环，**但完全没有**：① 分析结果的评分维度表与 `suggestions` 结构化建议；② 回填正文坐标的标注层；③ 异步任务模型；④ `expansion_plan` 级别的章节规划契约（`key_events/character_focus/emotional_tone/narrative_goal/conflict_type/ending_type`）；⑤ 选段局部重写；⑥ 驱动式重写（用建议索引 + 保留元素配置生成修改指令）与版本回滚。

---

## 2. 域内文件地图（真实体量）

| 文件 | 字节 | 角色 |
|---|---|---|
| `backend/app/services/plot_analyzer.py` | 28,428 | 章节分析核心（单次 LLM 调用 + 重试 + 记忆抽取 + 报告生成），602 行 |
| `backend/app/services/plot_expansion_service.py` | 28,743 | 大纲→多章规划（分批 + 差异化 + 章节落库 + 序号重排），691 行 |
| `backend/app/services/chapter_regenerator.py` | 10,690 | 整章重写（修改指令构建 + 流式生成 + diff 统计），248 行 |
| `backend/app/api/chapters.py` | 230,679 | 编排层：异步分析任务、一键分析、整章重写 SSE、局部重写 SSE、应用局部重写，5251 行 |
| `backend/app/api/polish.py` | 5,097 | AI 去味 API（单条 + 批量），141 行 |
| `backend/app/services/prompt_service.py` | 112,997 | 内联模板：`PLOT_ANALYSIS` L1047、`OUTLINE_EXPAND_SINGLE` L1398、`OUTLINE_EXPAND_MULTI` L1501、`CHAPTER_REGENERATION_SYSTEM` L1612、`PARTIAL_REGENERATE` L2414 |
| `backend/app/services/json_helper.py` | 21,794 | AI 返回 JSON 的层层清洗（606 行） |
| `backend/app/skills/story-deslop/**` | 3 文件 | 去 AI 味 Skill（SKILL.md 8,039B + anti-ai-writing.md 14,128B + banned-words.md 2,421B） |
| `backend/app/skills/story-long-write/references/hook-techniques.md` | 89,551 | 钩子技法大全（章尾 13 式 / 章首 7 式 / 段落级 11 型），2137 行 |

---

## 3. 情节分析域（PlotAnalyzer）

### 3.1 调用链与端到端时序

```
POST /api/chapters/{id}/analyze                    api/chapters.py L3477-L3563
  ├─ 鉴权 + 章节非空校验                            L3487-L3502
  ├─ 并发去重：查最新 AnalysisTask，pending/running 则直接复用  L3514-L3527
  ├─ 建 AnalysisTask(status=pending, progress=0)   L3530-L3538
  ├─ await asyncio.sleep(3)  ← 等 SQLite WAL 对其他会话可见  L3547
  └─ background_tasks.add_task(analyze_chapter_background)   L3550-L3556

analyze_chapter_background                          api/chapters.py L886-L922
  └─ asyncio.wait_for(_impl, timeout=600s)          L896-L906  → 超时写 failed
_analyze_chapter_background_impl                    api/chapters.py L925-L1433
  ├─ get_db_write_lock(user_id)  ← 每用户写锁        L947
  ├─ progress 10 → 20                             L986-L1010
  ├─ ai_service：按 usage="chapter_analysis" 取用户预设模型  L1013-L1017
  ├─ existing_foreshadows = foreshadow_service.get_planted_foreshadows_for_analysis(当前章号)  L1020-L1024
  ├─ 角色筛选：expansion_plan.character_focus → outline.structure.characters → 全量兜底  L1028-L1074
  ├─ characters_info = build_characters_info_with_careers(...)  L1076-L1081
  ├─ PlotAnalyzer(ai_service).analyze_chapter(...)  L1107-L1116
  ├─ progress 60 → 写 PlotAnalysis（有则更新、无则插入）→ progress 80   L1127-L1201
  ├─ foreshadow_service.clean_chapter_analysis_foreshadows(...)   L1203-L1214
  ├─ extract_memories_from_analysis → 删旧 StoryMemory → 逐条写 → batch_add_memories(向量库)  L1216-L1284
  ├─ CareerUpdateService.update_careers_from_analysis(...)        L1286-L1314
  ├─ CharacterStateUpdateService.update_from_analysis(...)        L1316-L1352
  ├─ CharacterStateUpdateService.update_organization_states(...)   L1354-L1380
  ├─ foreshadow_service.auto_update_from_analysis(...)            L1382-L1407
  └─ _set_analysis_task_terminal_state(completed)  L1409-L1417
```

**读侧**：`GET /api/chapters/{id}/analysis/status`（L2984-L3040）与 `POST /api/chapters/{project_id}/analysis/statuses`（L3043-L3100）；`GET /api/chapters/{id}/analysis`（L3265-L3328）；`GET /api/chapters/{id}/annotations`（L3331-L3474）。

> **重要发现**：本域存在**两套并行的分析编排**。`backend/app/api/memories.py:24` 的 `POST /memories/projects/{pid}/analyze-chapter/{cid}` 是同步版，重复了同一套 PlotAnalyzer + 记忆落库 + 伏笔/职业/关系更新的逻辑，且只有它返回 `entity_changes`（`memories.py` L203-L281）。而前端 `components/ChapterAnalysis.tsx` 从 `/chapters/{id}/analysis` 取数（L157）却读取 `entity_changes`（L385-L392）→ 该 "实体联动更新" 卡片在本流程下恒不渲染。见 F11。

### 3.2 输入装配算法

| 步骤 | 规则 | 证据 |
|---|---|---|
| 正文截断 | `analysis_content = content[:8000] if len(content) > 8000 else content` —— **硬截断前 8000 字** | `plot_analyzer.py` L66 |
| 温度 | `temperature=0.3`（"降低温度以获得更稳定的 JSON 输出"） | `plot_analyzer.py` L104 |
| 模板解析 | `user_id && db` → `PromptService.get_template("PLOT_ANALYSIS", user_id, db)`；异常降级到类属性默认 | L69-L77 |
| 伏笔三层注入 | 见 3.2.1 | L196-L268 |
| 角色清单 | 由上层筛选后传入 `characters_info`；空则填 `"（暂无角色信息）"` | L83-L91 |
| 模板参数 | `chapter_number / title / word_count / content / existing_foreshadows / characters_info` 共 6 个 | L83-L91 |

#### 3.2.1 伏笔注入的三层分级（本域最有复用价值的算法之一）

输入来自 `foreshadow_service.get_planted_foreshadows_for_analysis`（`foreshadow_service.py` L903-L971），只有 `status='planted'` 的伏笔参与，并按下述规则打标：

| 条件 | `resolve_status` | `resolve_hint` |
|---|---|---|
| `target_resolve_chapter_number == 当前章号` | `must_resolve_now` | 本章必须回收此伏笔 |
| `target_resolve_chapter_number < 当前章号` | `overdue` | 已超期 N 章，应尽快回收 |
| `target_resolve_chapter_number > 当前章号` | `not_yet` | 计划第 N 章回收，请勿提前回收 |
| 无 `target_resolve_chapter_number` | `no_plan` | 无明确回收计划，根据剧情自然回收 |

`PlotAnalyzer._format_existing_foreshadows`（L196-L268）再按层格式化：

- **第 1 层 `must_resolve_now`（无上限，最详细）**：`【ID: {id}】{title}` / 埋入章节 / 伏笔内容（截 200 字）/ 埋入暗示（截 100 字）/ `⚠️ 回收时 reference_foreshadow_id 填写: {id}`。
- **第 2 层 `overdue`（`[:5]`）**：`- 【ID: {id}】{title}（第{plant}章埋入）`。
- **第 3 层 其他（`[:10]`，超出部分写 `... 还有N个伏笔未列出`）**：同精简格式。
- 尾部固定操作指引：回收时在 `foreshadows` 数组加 `type='resolved'` 并填 `reference_foreshadow_id`。

> 设计要点：**只有"本章必须回收"的伏笔拿到完整内容**，其余降级为 ID+标题，用信息密度换上下文预算。这是让 LLM 在有限窗口里既"不漏回收"又"不越界提前回收"的关键。

#### 3.2.2 角色清单的筛选优先级（上层编排）

`_analyze_chapter_background_impl` L1028-L1074：

1. `chapter.expansion_plan` → `json.loads(...)['character_focus']`（1-N 模式）；
2. 否则 `chapter.outline_id` → `outline.structure` → `json.loads(...)['characters']`（元素可为 `dict` 或 `str`，取 `c['name']`）（1-1 模式）；
3. `Character.name.in_(filter_names)` 查询；**若筛选后为空则降级为全量角色**并置 `filter_character_names = None`（L1068-L1074）。

### 3.3 重试与退避策略

`analyze_chapter` 的 `for attempt in range(1, max_retries + 1)`（默认 `max_retries=3`），三条重试触发路径共用同一退避公式：

| 触发条件 | 证据 | 退避 |
|---|---|---|
| 响应为空或 `< 10` 字符 | L120-L136 | `wait_time = min(2 ** attempt, 10)` → 2s / 4s / 8s |
| JSON 解析失败（`_parse_analysis_response` 返回 None） | L153-L169 | 同上 |
| 其他异常（含流中断后的 `raise Exception("流式响应中断，内容不足")`） | L173-L190 | 同上 |

细节：

- `asyncio.CancelledError` 永远**直接 re-raise**，不参与重试（L107-L108、L171-L172）。
- `GeneratorExit`（流被中断）**不算致命**：若已累积 ≥100 字符则继续尝试解析（L109-L114）。
- 每次重试前 `await on_retry(attempt, max_retries, wait_time, reason)`，回调里把任务 `status` 保持 `running`、**重置 `started_at`** 并把 `progress = 25 + attempt * 5`（`api/chapters.py` L1085-L1104）；重置 `started_at` 是为了让"陈旧任务判定"不误杀正在重试的任务。
- 回调本身抛异常只 warning，不影响主流程（L127-L131、L160-L164、L181-L185）。

### 3.4 结构化输出 Schema（逐字段）

顶层对象（`prompt_service.py` L1228-L1353）：

| 字段 | 类型 | 约束 / 取值 |
|---|---|---|
| `hooks[]` | array | `type`∈{悬念,情感,冲突,认知}（Prompt 文字枚举）/ `content` / `strength` 1-10 / `position`∈{开头,中段,结尾} / `keyword` **必填 8-25 字原文** |
| `foreshadows[]` | array | `title`(10-20字) / `content` / `type`∈{planted,resolved} / `strength` 1-10 / `subtlety` 1-10 / `reference_chapter`(回收时原埋入章号) / `reference_foreshadow_id`(回收时必填) / `keyword` **必填** / `category`∈{identity,mystery,item,relationship,event,ability,prophecy} / `is_long_term`(跨 10 章以上回收为 true) / `related_characters[]` / `estimated_resolve_chapter` **必填** |
| `conflict` | object | `types[]`∈{人与人,人与己,人与环境,人与社会} / `parties[]`（格式 `"主角-立场"`） / `level` 1-10 / `description` / `resolution_progress` 0.0-1.0 |
| `emotional_arc` | object | `primary_emotion`(≤10字) / `intensity` 1-10 / `curve`（如 `平静→紧张→高潮→释放`） / `secondary_emotions[]` |
| `character_states[]` | array | `character_name` / `survival_status`∈{null,active,deceased,missing,retired}（默认 null） / `state_before` / `state_after` / `psychological_change` / `key_event` / `relationship_changes`（map 名→变化） / 可选 `career_changes{main_career_stage_change,sub_career_changes[],new_careers[],career_breakthrough}` / 可选 `organization_changes[]{organization_name,change_type∈{joined,left,promoted,demoted,expelled,betrayed},new_position,loyalty_change,description}` |
| `plot_points[]` | array | `content` / `type`∈{revelation,conflict,resolution,transition} / `importance` 0.0-1.0 / `impact` / `keyword` **必填** |
| `scenes[]` | array | `location` / `atmosphere` / `duration` |
| `organization_states[]` | array | `organization_name` / `power_change`(整数，±N) / `new_location` / `new_purpose` / `status_description` / `key_event` / `is_destroyed`(bool，默认 false，仅"彻底覆灭"为 true) |
| `pacing` | string | `slow|moderate|fast|varied`（示例给 `varied`） |
| `dialogue_ratio` / `description_ratio` | float | 0.0-1.0（示例 0.4 / 0.3） |
| `scores` | object | 见 3.5 |
| `plot_stage` | string | 开端/发展/高潮/结局/过渡 |
| `suggestions[]` | array of **纯字符串** | 格式 `【问题类型】问题定位 + 为什么是问题 + 改进方向` |

`suggestions` 的强约束（Prompt L1367-L1370、L1393-L1394）：必须是纯字符串数组，**禁止** `{"suggestion": "..."}`、`{"content": "..."}` 等对象元素，无建议必须返回 `[]`（不能 null、不能省字段）。

### 3.5 评分体系与"建议数量 ↔ 总分"联动

分档标准（Prompt L1186-L1213），三维 + 一个加权总分：

| 维度 | 1.0-3.9 | 4.0-5.9 | 6.0-7.9 | 8.0-9.4 | 9.5-10.0 |
|---|---|---|---|---|---|
| `pacing` | 节奏混乱/切换生硬/拖沓 | 可读但明显问题 | 整体流畅偶有小问题 | 精准，高潮迭起 | 大师级 |
| `engagement` | 乏味缺钩子 | 有情节无亮点 | 有效但不够巧妙 | 引人入胜 | 每段都有阅读动力 |
| `coherence` | 逻辑混乱/前后矛盾 | 基本连贯有漏洞 | 偶有小瑕疵 | 逻辑严密 | 无懈可击 |
| `overall` | `(pacing + engagement + coherence) / 3`，保留一位小数，可 ±0.5 调整 | | | | |

**建议数量硬联动**（L1215-L1220，约束区 L1376-L1380 复述）：`overall < 4.0` → 4-5 条；`4.0-5.9` → 3-4 条；`6.0-7.9` → 1-2 条；`≥ 8.0` → 0-1 条。Prompt 明确禁止"所有章节都打 7-8 分的安全分"（L1391）并要求 `score_justification` 必填（L1346、L1375）。

`scores` 之外的每个维度（钩子/伏笔/冲突/情感/角色/情节点/场景）都进 Prompt 的 `<analysis_framework>`，共 9 个编号维度（L1086-L1226，其中含 5b 组织子项）。

### 3.6 JSON 解析容错栈

`PlotAnalyzer._parse_analysis_response`（L270-L303）→ `AIService._clean_json_response`（`ai_service.py` L692-L695）→ `json_helper.clean_json_response`（L365-L534）→ `loads_json`（L565-L606）。

`loads_json` 的降级顺序：
1. `json.loads(text)`；
2. 失败 → `_fix_all_invalid_escapes(text)` 后重试（L578-L585）；
3. 失败 → `json5.loads(text)`（L588-L593）；
4. 失败 → `json5.loads(fixed_text)`（L595-L602）；
5. 全失败 → 抛 `json.JSONDecodeError`。

`clean_json_response` 的预处理顺序（L375-L385）：`strip_think_tags` → `_fix_json_string_values`（**上下文感知**：区分字符串内外，结构位置的中文引号/逗号/冒号替换成 ASCII，字符串内保留或转义）→ 去 markdown 代码块 → 直接解析快速路径 → 从第一个 `{`/`[` 起做括号匹配（含严格字符串处理）→ 兜底 `_fix_all_invalid_escapes` → `_fix_multiple_objects_as_value`。

**容错后仍只做"缺字段补空"，不做语义校验**（`plot_analyzer.py` L288-L292）：

```python
required_fields = ['hooks', 'plot_points', 'scores']
for field in required_fields:
    if field not in result:
        result[field] = [] if field != 'scores' else {}
```

注意：`foreshadows` / `character_states` / `conflict` / `emotional_arc` / `suggestions` / `plot_stage` **不在校验清单**里，缺失时由下游 `.get(..., 默认值)` 兜。

### 3.7 记忆提取算法（分析结果 → 可检索记忆）

`extract_memories_from_analysis`（L305-L486）产出 `StoryMemory` 记录，**阈值表**如下：

| 来源 | 类型 | 准入阈值 | `importance_score` | `is_foreshadow` |
|---|---|---|---|---|
| 章节摘要（`summary` → 前 3 条 plot_point 拼接 → 正文前 300 字，三级回退） | `chapter_summary` | 有摘要即入 | 固定 **0.6** | 0 |
| 钩子 | `hook` | `strength >= 6` | `min(strength/10, 1.0)` | 0 |
| 伏笔 | `foreshadow` | 无阈值（全部） | `min(strength/10, 1.0)` | planted→**1** / 其他→**2** |
| 情节点 | `plot_point` | `importance >= 0.6` | `importance` | 0 |
| 角色状态 | `character_event` | 无阈值（全部） | 固定 **0.7** | 0 |
| 高强冲突（`conflict.level >= 7`） | `plot_point` | `level >= 7` | `min(level/10, 1.0)` | 0 |

摘要三级回退证据：L333-L341。冲突分支会把 `parties`/`types` 强制 `str()` 化以避免非字符串元素炸 JSON（L460-L466）。

#### 3.7.1 正文定位算法 `_find_text_position`（L488-L531）

四段式，返回 `(起始位置, 长度)`，失败 `(-1, 0)`：

1. **精确匹配**：`full_text.find(keyword)`，命中返回 `(pos, len(keyword))`；
2. **去标点后匹配**：对 keyword 与全文同时删除 `，。！？、；：""''（）《》【】` 再 `find`。**注意：命中后返回的是"净化文本中的位置"，即返回坐标与原文坐标可能偏移**（源码注释自称"简化处理"，L515）；
3. **模糊匹配**：仅当 `len(keyword) > 10`，取 `keyword[:15]` 再 `find`；
4. 未找到返回 `(-1, 0)`，并 debug 日志。

落库时：`StoryMemory.chapter_position = text_position`，`text_length = length`（`api/chapters.py` L1251-L1266），即以**正文字符偏移**作为标注锚点。

#### 3.7.2 记忆 ID 生成（非稳定）

`memory_id = f"{chapter_id}_{mem['type']}_{len(memory_records)}"`（`api/chapters.py` L1239）。是**序号式**而非内容哈希 —— 重跑分析时按"先删后建"重建（L1226-L1234），因此 ID 会变。与 gaea 的 `GenerateStableID`（`internal/analysis/analysis.go` L212-L215，`sha256(category+chapterFile+description)`）形成对照：**gaea 的稳定 ID 更适合增量合并，MuMu 的序号 ID 只能全量重建**。

### 3.8 分析结果落库映射

`analysis_result` → `PlotAnalysis`（`models/memory.py` L80-L166），关键是**降维**：

| Prompt 输出 | DB 列 | 转换 |
|---|---|---|
| `scores.overall/pacing/engagement/coherence` | `overall_quality_score`/`pacing_score`/`engagement_score`/`coherence_score` | 直取，默认 0 |
| `emotional_arc.intensity` (1-10) | `emotional_intensity` (Float) | **÷ 10.0** 归一到 0-1（L1146、L1175） |
| `emotional_arc.primary_emotion` | `emotional_tone` | 直取 |
| `conflict.level/types` | `conflict_level`/`conflict_types` | 直取 |
| `hooks` | `hooks` + `hooks_count` | `hooks_count = len(hooks)` |
| `foreshadows` | `foreshadows` + `foreshadows_planted`/`foreshadows_resolved` | 按 `type` 计数 |
| `plot_points` | `plot_points` + `plot_points_count` | `len()` |
| — | `analysis_report` | `analyzer.generate_analysis_summary(result)`（L533-L591 生成中文报告文本） |
| `suggestions` | `suggestions` | 直存 JSON 数组 |
| `emotional_arc.curve` | ❌ 无列 | `emotional_curve` 列存在但**从不写入** |
| — | `hooks_avg_strength` | 无代码写入 |
| — | `word_count` | 无代码写入 |

`to_dict()`（L171-L200）暴露给前端 21 个键，前端 `ChapterAnalysis.tsx` 用 `emotional_intensity * 10` 还原成 10 分制（L638）。

### 3.9 异步任务状态机与"三重超时"

`AnalysisTask`（`models/analysis_task.py` L8-L42）：`pending → running → completed/failed`；列含 `progress`(0-100)、`error_message`、`created_at`/`started_at`/`completed_at`/`archived_at`；三个索引：`(chapter_id, created_at)`、`(status)`、`(project_id, user_id, archived_at, created_at)`。

三层保障：

| 层 | 机制 | 常量 / 证据 |
|---|---|---|
| ① 硬超时 | `asyncio.wait_for(_impl, timeout=600)`，超时写 `failed` + 文案 `分析任务超时（超过10分钟）` | `api/chapters.py` L70 `ANALYSIS_TASK_TIMEOUT_SECONDS = 600`、L896-L911 |
| ② 取消兜底 | `except asyncio.CancelledError` 时用 `asyncio.shield(create_task(...))` 把任务写成 `failed`（"分析任务被取消或服务关闭"），再 re-raise | L912-L922 |
| ③ 独立终态事务 | `_set_analysis_task_terminal_state`：**独立 session** 写入终态，`WHERE id=? AND status IN ('pending','running')`（幂等 guard），失败重试 3 次、间隔 0.1s | L98-L136 |
| ④ 陈旧恢复 | `_recover_stale_analysis_tasks`：`status=='running'` 且 `(now - (started_at or created_at)) > 720s` → 原子 `UPDATE ... WHERE id=? AND status=?`，`rowcount` 决定是否命中，成功后 `commit` + `refresh` | L71 `ANALYSIS_TASK_STALE_SECONDS = 720`、L2449-L2487 |

去重策略：`POST /analyze` 若发现最新任务处于 `pending/running` 则**直接返回既有任务**，不新建（L3514-L3527）。

一键分析未分析章节（L3134-L3262）：按 `chapter_number` 顺序遍历 → 跳过无内容 / `pending|running` / `completed` → 其余建 `pending` 任务 → `flush` + `commit` → `_schedule_analysis_background(_run_batch_analysis_in_sequence(...))` **逐章串行**执行（L3103-L3131）；返回统计四元组 `total_candidates / total_started / total_skipped_no_content / total_skipped_running / total_already_completed`。

### 3.10 标注层（annotations）构建

`GET /{chapter_id}/annotations`（L3331-L3474）把 `StoryMemory` 列表转成前端可渲染的 `MemoryAnnotation`：

- `position` 优先取 `StoryMemory.chapter_position`；为 `-1` 时**按类型回查分析 JSON 里的 `keyword`** 并 `chapter.content.find(keyword)` 重算位置（L3385-L3423）；
- 同时补齐展示元数据：`hook` → `strength` + `position_desc`；`foreshadow` → `foreshadow_type` + `strength`（L3426-L3439）；
- 返回 `summary` 四计数：`hooks / foreshadows / plot_points / character_events`（L3467-L3473）。

前端可视化：`pages/ChapterAnalysis.tsx`（571 行）左侧章节列表 + 中间 `AnnotatedText`（正文内联高亮）+ 右侧 `MemorySidebar`（标注列表），双向点击联动（L177-L195），并订阅 `BACKGROUND_TASK_SETTLED` 事件在分析完成后自动刷新（L145-L156）。

分析结果弹窗：`components/ChapterAnalysis.tsx`（901 行），6 个 Tab：概览 / 钩子 / 伏笔 / 情感曲线 / 角色 / 记忆（L398-L750）。轮询实现：`startPolling` 间隔 2s、总上限 `11 * 60 * 1000` ms（L171-L211）；用 `requestGenerationRef` 递增序号做**竞态防护**，过期响应直接丢弃（L45-L50、L88、L113 等）。

---

## 4. 大纲→章节展开域（PlotExpansionService）

### 4.1 分批算法

`analyze_outline_for_chapters`（L25-L84）分派：

```
if target_chapter_count <= batch_size:          # 默认 batch_size = 5
    → _generate_chapters_single_batch           # L86-L152，用 OUTLINE_EXPAND_SINGLE
else:
    → _generate_chapters_in_batches             # L154-L289，用 OUTLINE_EXPAND_MULTI
```

分批核心（L167-L289）：

```
total_batches = ceil(target_chapter_count / batch_size)          # L169
used_key_events = set()                                          # L189 ← 见 F3：死变量
for batch_num in range(total_batches):
    remaining = target_chapter_count - len(all_chapter_plans)     # L193
    current_batch_size = min(batch_size, remaining)               # L194
    current_start_index = len(all_chapter_plans) + 1              # L195
    await progress_callback(batch_num+1, total_batches, current_start_index, current_batch_size)  # L200-L201
    previous_context = <跨批差异化块，见 4.2>                       # L204-L237
    prompt = format(OUTLINE_EXPAND_MULTI, ..., previous_context, start_index, end_index, target_chapter_count=current_batch_size)
    batch_plans = _parse_expansion_response(ai_content, outline.id)   # L278
    for i, plan in enumerate(batch_plans): plan["sub_index"] = current_start_index + i   # L281-L282
    all_chapter_plans.extend(batch_plans)                         # L284
```

**关键不变量**：`sub_index` 是**批内相对序号被重写为全局序号**（L281-L282），而非依赖模型自报。这是分批展开能保持连续性的根因。

### 4.2 跨批差异化上下文（本域最有移植价值的 Prompt 工程）

后批注入的 `previous_context`（L204-L237）由三块拼成：

1. **全部已生成章节摘要**（不限数量）——`第N节《标题》:` + `剧情：plot_summary[:150]` + `关键事件：key_events[:3]` + `结尾方式：ending_type`（L209-L215）；
2. **已使用的关键事件**——把所有历史 `key_events` 摊平后取**最后 20 条**（L218-L221）；
3. **差异化强制要求**——显式声明当前批次区间，并要求每个新章在"开场场景 / 核心事件 / 结尾悬念"三个维度上完全不同，且 `key_events` 不得与清单中任何事件"相同或相似"（L230-L237）。

### 4.3 章节落库与序号重排

`create_chapters_from_plans`（L381-L482）：

- **起始章号自动计算**：`start_chapter_number = 前面所有 order_index 更小的大纲已有关联章节数之和 + 1`（L404-L440，逐大纲 `COUNT(Chapter.id)` 累加）；
- `expansion_plan` 序列化（L445-L453）：`{key_events, character_focus, emotional_tone, narrative_goal, conflict_type, estimated_words(default 3000), scenes(可为 null)}` —— **注意 `title`/`plot_summary`/`ending_type`/`sub_index` 不进 JSON**，其中 `title`→`Chapter.title`、`plot_summary`→`Chapter.summary`、`sub_index`→`Chapter.sub_index`；
- `Chapter(status="draft")`；
- 落库后调用 `_renumber_subsequent_chapters`（L600-L685）：从第 1 章开始，按大纲 `order_index` 顺序、组内按 `sub_index` 顺序，**重排该大纲及之后所有大纲下章节的 `chapter_number`**，只 `commit` 一次。

**双序设计**（值得照搬）：`sub_index` = 大纲内相对序（稳定，不随后续插入而变），`chapter_number` = 全书绝对序（插入/删除后由 `_renumber_subsequent_chapters` 全局重排）。

### 4.4 展开输出 JSON 契约（`OUTLINE_EXPAND_SINGLE`，L1444-L1456）

```json
[
  {
    "sub_index": 1,
    "title": "章节标题（体现核心冲突或情感）",
    "plot_summary": "剧情摘要（200-300字）：详细描述该章发生的事件，仅限当前大纲内容",
    "key_events": ["关键事件1", "关键事件2", "关键事件3"],
    "character_focus": ["角色A", "角色B"],
    "emotional_tone": "情感基调（如：紧张、温馨、悲伤）",
    "narrative_goal": "叙事目标（该章要达成的叙事效果）",
    "conflict_type": "冲突类型（如：内心挣扎、人际冲突）",
    "estimated_words": 3000
  }
]
```

模板尾部的 `{scene_field}` 在所有调用点都传空字符串（`plot_expansion_service.py` L131-L132、L260-L261），即**场景级字段的开关是预留但未接线**；`enable_scene_analysis` 参数全程只被透传、从未参与分支（grep 只命中形参与调用，见 F2b）。

### 4.5 解析容错与降级（`_parse_expansion_response`，L524-L597）

1. `_clean_json_response` → `loads_json`；
2. 非 list 则包成 `[obj]`（L538-L539）；
3. 逐项补 `outline_id`；缺 `ending_type` 时**按 `narrative_goal` 关键词推断**（含"悬念/疑问"→悬念；"冲突/对抗"→冲突升级；"转折"→情节转折；"情感/情绪"→情感收尾；否则 `自然过渡-{idx+1}`，L545-L558）；`key_events` 空则填 `[f"章节{idx+1}核心事件"]`（L561-L562）；
4. `JSONDecodeError` / 其他异常各返回**一条"占位规划"**（`title="AI解析失败的默认章节"`/`"解析异常的默认章节"`，`plot_summary` 为响应前 500 字或"系统错误"，`narrative_goal="需要重新生成"`，L567-L597）。

> 这是**静默降级**：解析失败不会抛错，调用方拿到"看似成功"的 1 章规划。下游若不做 `narrative_goal == "需要重新生成"` 的检查，会把占位章节当正常章节落库。

### 4.6 批量与流式入口

`plot_expansion_service` 被 `api/outlines.py` 多处调用（L2355、L2512、L2636、L3061、L3342）。对外入口：

- `POST /outlines/{outline_id}/expand-stream`（L2875-L2921，SSE，进度阶段 5/10/15/20/30/70/80/90/95/100 见文档串 L2896-L2906）；
- `POST /outlines/{outline_id}/expand-background`（L2833）；
- `POST /outlines/batch-expand-stream`（L3262-L3288）/ `batch-expand-background`（L3227）；
- `POST /outlines/{outline_id}/create-chapters-from-plans`（L3291）。
- `GET /outlines/{outline_id}/chapters`（L2924-L2994）回读：把 `expansion_plan` JSON 反序列化回 `key_events/character_focus/emotional_tone/narrative_goal/conflict_type/estimated_words/scenes` 七字段。

---

## 5. 一键重写域

### 5.1 三条重写路径总览

| 路径 | 入口 | 粒度 | 触发条件 | 有版本/回滚？ | 落盘时机 |
|---|---|---|---|---|---|
| **A 整章重写** | `POST /chapters/{id}/regenerate-stream`（L4449） | 整章 | `modification_source ∈ {custom, analysis_suggestions, mixed}`，至少一条建议或自定义指令 | ✅ `RegenerationTask.original_content` 快照 + `version_number` | 生成后**不自动落章**；前端确认后 `PUT /chapters/{id}` |
| **B 选段局部重写** | `POST /chapters/{id}/partial-regenerate-stream`（L4922） | 字符区间 | 必填 `user_instructions`（1-1000 字） | ❌ 无快照 | 前端"接受并应用"→ `POST /chapters/{id}/apply-partial-regenerate`（L5178）**立即拼接入库** |
| **C 去 AI 味** | `POST /polish`（`api/polish.py` L17）/ `POST /polish/batch`（L89） | 任意文本 | 无 | ❌ | 不落章；仅当传 `project_id` 时写 `GenerationHistory(generation_type="polish")` |
| **C' Skill 版去 AI 味** | `skills/story-deslop/SKILL.md` | 任意文本 | 触发词 `/story-deslop`、`/去AI味`、`去味`、`deslop`、`这篇太AI了`（`skill_loader.py` L285-L288） | ❌ | 纯报告 + 润色全文 |

### 5.2 路径 A：整章重写

**触发条件**（`schemas/regeneration.py` L15-L38）：

```python
modification_source: str = "custom"            # custom | analysis_suggestions | mixed
selected_suggestion_indices: Optional[List[int]]
custom_instructions: Optional[str]
preserve_elements: Optional[PreserveElementsConfig]
style_id: Optional[int]
target_word_count: int = 3000                  # ge=500, le=10000
focus_areas: List[str] = []                    # pacing/emotion/description/dialogue/conflict
save_as_version: bool = True
version_note: Optional[str]                   # max_length=500
auto_apply: bool = False
```

`PreserveElementsConfig`（L7-L12）：`preserve_structure=False` / `preserve_dialogues=[]` / `preserve_plot_points=[]` / `preserve_character_traits=True`。

前置校验（`api/chapters.py` L4489-L4500）：`modification_source ∈ {analysis_suggestions, mixed}` 时**必须存在 `PlotAnalysis`**，否则 404 `"该章节暂无分析结果"`。

**修改指令构建**（`chapter_regenerator.py` L110-L179），四段顺序固定：

```
# 章节修改指令
## 📋 需要改进的问题（来自AI分析）：
{idx+1}. {analysis.suggestions[idx]}            # 仅选中的索引，越界静默跳过（L129）
## ✍️ 用户自定义修改要求：
{custom_instructions}
## 🎯 重点优化方向：
- {focus_map[area]}                              # 五选映射，未知 key 静默丢弃
## 🔒 必须保留的元素：
- 保持原章节的整体结构和情节框架                   # preserve_structure
- 必须保留以下关键对话：  * {dialogue}            # preserve_dialogues
- 必须保留以下关键情节点：* {plot}                # preserve_plot_points
- 保持所有角色的性格特征和行为模式一致              # preserve_character_traits
```

`focus_map`（L143-L149）：`pacing`→"节奏把控 - 调整叙事速度，避免拖沓或过快"；`emotion`→"情感渲染 - 深化人物情感表达，增强感染力"；`description`→"场景描写 - 丰富环境细节，增强画面感"；`dialogue`→"对话质量 - 让对话更自然真实，推动剧情"；`conflict`→"冲突强度 - 强化矛盾冲突，提升戏剧张力"。

**Prompt 组装**（`PromptService.get_chapter_regeneration_prompt`，`prompt_service.py` L2628-L2753），拼接顺序：
`CHAPTER_REGENERATION_SYSTEM` → `## 📖 原始章节信息`(章节/标题/字数/全文) → 修改指令 → `## 🌍 项目背景信息`(标题/题材/主题/视角/时代/地理/氛围) → 可选 `## 👥 角色信息` → 可选 `## 📝 本章大纲` → 可选 `## 📚 前置章节上下文` → 可选 `## 🎨 写作风格要求` → `## ✨ 创作要求`（5 条 + 条件第 6 条"风格一致"）→ `## 🎬 开始创作` + 三条"不要输出标题/说明/元数据"的反向指令（L2744-L2750）。

**风格注入双通道**（L74-L83）：`style_content` 既进 User Prompt 的 `## 🎨 写作风格要求`，又包成最高优先级 System Prompt：

```
【🎨 写作风格要求 - 最高优先级】

{style_content}

⚠️ 请严格遵循上述写作风格要求进行重写，这是最重要的指令！
确保在整个章节重写过程中始终保持风格的一致性。
```

`style_id` 为空时回退到 `ProjectDefaultStyle`（L4583-L4592），且校验 `style.user_id is None or style.user_id == user_id`（L4602）。

**流式与进度**（`chapter_regenerator.py` L51-L104）：5% 构建指令 → 10% 构建提示词 → 15% 开始生成 → 生成阶段 `generation_progress = min(15 + (accumulated_length / target_word_count) * 80, 95)` → 100% 完成。温度 **0.7**（L92）——与分析阶段 0.3 形成对照。

**版本与回滚**（`api/chapters.py` L4647-L4668、L4732-L4759）：`RegenerationTask` 落库字段含 `original_content`（原文快照）、`original_word_count`、`regenerated_content`、`regenerated_word_count`、`version_number`、`version_note`、`selected_suggestion_indices`、`focus_areas`、`preserve_elements`、`status`、`started_at`/`completed_at`（`models/regeneration_task.py` L8-L51）。任务历史可查：`GET /chapters/{id}/regeneration/tasks`（L4801，`limit` 1-50）。

**diff 统计**（`chapter_regenerator.py` L207-L237）：`original_length / new_length / length_change / length_change_percent` + `difflib.SequenceMatcher(None, original, new).ratio()` → `similarity`(%) / `difference`(%) + 段落数（按 `\n\n` 切分）。结果**只在 SSE `result` 事件回传**（L4753-L4759），**不落库**（`RegenerationTask` 无 diff 列，见 F6）。

**回滚策略（真实行为）**：生成完成 → `tracker.result(...)` → 前端 `ChapterRegenerationModal.onResult` 累积内容并打开 `ChapterContentComparison`（`components/ChapterAnalysis.tsx` L842-L857）。对比页只有三个出口：

- **应用新内容**：`PUT /api/chapters/{chapterId}` `{content: newContent}` → 成功后 `onApply()` → 延时 500ms **自动再触发一次 `POST /analyze`**（`ChapterContentComparison.tsx` L88-L138）；
- **放弃新内容**：二次确认后仅清前端 state（L140-L154）；
- **切换视图**：split / unified。

即：**"回滚"发生在前端内存里（放弃 = 不写库），不是后端版本回退**。`RegenerationTask` 的 `original_content` 只是审计记录，**没有任何"从版本 N 恢复"的 API**（grep `regeneration_task` 无 restore/revert 路由）。

### 5.3 路径 B：选段局部重写

**请求契约**（`schemas/chapter.py` L215-L251）：`selected_text`（必填）、`start_position`/`end_position`（字符索引，`ge=0`）、`user_instructions`（必填 1-1000 字）、`context_chars`（**默认值需 read 确认，前端固定传 500**，见 `PartialRegenerateModal.tsx` L91）、`style_id`、`length_mode`、`target_word_count`。

**位置校验与校正算法**（L4959-L4986）：

```
if start >= len(content)          → 400 "起始位置超出内容范围"
if end   >  len(content)          → 400 "结束位置超出内容范围"
if start >= end                   → 400 "起始位置必须小于结束位置"
actual = content[start:end]
if actual != selected_text:
    search_area = content[max(0,start-50) : min(len,end+50)]     # ±50 字符窗口
    if selected_text in search_area:
        offset = search_area.find(selected_text)
        start = (start-50) + offset ;  end = start + len(selected_text)
    else:
        400 "选中的文本与章节内容不匹配，请刷新页面后重试"
```

**上下文截取**（L5034-L5044）：`context_before = content[max(0,start-500):start]`、`context_after = content[end:min(len,end+500)]`；空则分别填 `（这是章节开头）` / `（这是章节结尾）`（L5081-L5084）。

**长度模式映射**（L5054-L5071，`length_mode = 原文长度的乘数区间`）：

| `length_mode` | 生成区间 | 提示文案 | `target_words`（用于 max_tokens） |
|---|---|---|---|
| `similar`（默认） | `0.8× ~ 1.2×` | 保持与原文相近的字数（约N字，允许 min-max 字浮动） | `1.5×` |
| `expand` | `1.2× ~ 2.0×` | 适当扩展内容（目标 min-max 字） | `2.0×` |
| `condense` | `0.5× ~ 0.8×` | 精简压缩内容（目标 min-max 字） | `1.5×` |
| `custom` | 用户值 ±20% | 目标字数：约N字（允许±20%浮动） | 用户值 |
| 其他 | 原长度 | 保持与原文相近的字数 | `1.5×` |

**max_tokens 公式**（L5100）：`calculated_max_tokens = max(500, min(int(target_words * 3), 8000))` —— 中文按"1 字 ≈ 3 token"估算，上限 8000。

**输出清理**（L5131-L5151）：`strip()` → 移除 8 个常见前缀（`重写后：`/`重写后:`/`改写后：`/`改写后:`/`以下是重写后的内容：`/`以下是重写后的内容:`/`重写内容：`/`重写内容:`，命中即 `break`）→ 成对去除 `"` `'` `「」` `『』`。

**应用**（L5178-L5250）：`new_content = content[:start] + new_text + content[end:]` → 更新 `chapter.content`、`chapter.word_count = len(new_content)` → **同步调整 `project.current_words = current_words - old_word_count + new_word_count`**（L5232-L5237）→ `commit`。

**前端**（`PartialRegenerateModal.tsx`，451 行）：必填校验 `user_instructions`（L70-L73）；`AbortController` 支持取消（L81、L128-L135）；生成中锁定弹窗关闭（`maskClosable/closable/keyboard = !isGenerating`，L190-L192）；"接受并应用"直接在弹窗内调 `applyPartialRegenerate`（L145-L149），再回调父组件用**同样的字符区间**改本地表单（`pages/Chapters.tsx` L1962-L1976）。

**无回滚**：`apply-partial-regenerate` 直接拼接落库，无 `RegenerationTask`、无快照、无 undo API。见 F5。

### 5.4 路径 C：AI 去味

**API 层**（`api/polish.py`）：`POST /polish`（L17-L86）取 `PromptService.get_template("AI_DENOISING", user_id, db)`（L40）→ `format_prompt(template, original_text=...)` → `generate_text(prompt, provider, model, temperature, max_tokens=len(original_text)*2)` → 返回 `PolishResponse{original_text, polished_text, word_count_before, word_count_after}`；`POST /polish/batch`（L89-L141）串行处理 `texts: list[str]`。

> ⚠️ **`AI_DENOISING` 模板在本仓库不存在**：全仓 grep 只命中 `api/polish.py` 的两处使用（L40、L114），`prompt_service.py` 内既无该模板常量、也未登记进 `get_all_system_templates()`（L2873-L3140 的定义表，`AI_DENOISING` 零命中）。`get_template` 的降级链（L2816-L2860）最终 `getattr(cls, key, None) → None`，随后 `format_prompt(None, ...)` 触发 `AttributeError` → 被 `except` 捕获 → **HTTP 500**。即：**除非用户自建同名自定义模板，`/polish` 必然 500**。见 F1。

**Skill 层（真正的资产）**：`skills/story-deslop/` 是目前唯一"能跑"的去 AI 味实现，且质量最高的部分全在文本资产里：

- `SKILL.md` 的**六道门禁**（L104-L170）：Gate A 禁用词替换 / B 句式去套路 / C 心理描写外化 / D 节奏打碎 / E 对话去腔调 / F 结尾去升华；
- **真人写作基准对照表**（L37-L46）：段落长度、对话标签、情绪表达、比喻、语气词、省略、排比、结尾共 8 维度"真人写法 vs AI 写法"；
- **高频表达替换表**（L49-L56）：`深吸一口气 → 胸口起伏了一下/删掉`、`眼中闪过一丝… → 他垂下眼`、`嘴角勾起一抹… → 笑了一下，没到眼底`、`仿佛… → 像…/白描`、`不禁… → 直接写动作`、`缓缓开口 → 说`；
- **AI 味分级 → 处理强度**（L88-L92）：轻度 = Gate A+B；中度 = A+B+C+D；重度 = 完整 6 Gate + 重点段落重写；
- `references/banned-words.md`：**一级禁用词**七类（情态/动作/表情/心理/判断/形容/过渡，如 `仿佛|好像|犹如|宛若|一丝|一抹|些许|几分|隐约`、`深吸一口气|缓缓|不禁|微微|轻轻|淡淡`、`眼中闪过|嘴角勾起|眉头微皱|瞳孔微缩`、`心中一动|心头一震|心下了然|不由`），**二级禁用词**三类（总结句式/排比句式/升华句式），**禁用句式模板** 10 条；
- `references/anti-ai-writing.md`：**7 种 AI 写作模式检测**（L179-L217，含"弱化副词：每 1000 字超过 3 个 = AI 签名"这类**可量化阈值**）、**系统性去 AI 三遍法**（L221-L259：Pass 1 去泛化 / Pass 2 去书面化 / Pass 3 回人味，附"轻度只做 Pass 1、中度 P1+P2、重度完整三遍"升级策略与自检清单）、**连续复用禁令**（L89-L100：不得连续使用原文 12 字以上原句，替换同义词不算改写）、**Show Don't Tell 对照表**（L104-L119）。

### 5.5 触发条件与回滚策略总表（可执行结论）

| 场景 | 能否触发 | 前置条件 | 失败模式 | 回滚 |
|---|---|---|---|---|
| 用分析建议重写整章 | ✅ | 必须有 `PlotAnalysis`；至少选 1 条建议 | 无分析 → 404 | 放弃 = 不写库（前端内存态） |
| 纯自定义指令重写整章 | ✅ | `custom_instructions` 非空（前端 L99-L102 校验） | — | 同上 |
| 混合模式 | ✅ | ≥1 建议 或 非空自定义指令（前端 L109-L114） | — | 同上 |
| 保留指定对话/情节点 | ⚠️ 半可用 | Schema/指令构建支持，但**前端表单未暴露** `preserve_dialogues` / `preserve_plot_points` 输入（`ChapterRegenerationModal.tsx` 只渲染 `preserve_structure` 与 `preserve_character_traits`，L376-L385） | 恒为空数组 → 该段指令不生成 | — |
| 局部重写 | ✅ | `user_instructions` 非空；选中区间与正文匹配（±50 字符容差） | 400 三类位置错 / 不匹配 | ❌ 无 |
| 去 AI 味（API） | ❌ | — | **500：`AI_DENOISING` 模板缺失** | — |
| 去 AI 味（Skill） | ✅ | 依赖 project_agent 技能加载链 | — | — |

---

## 6. Prompt 原文摘录（逐字）

### 6.1 `PLOT_ANALYSIS` 骨架（`prompt_service.py` L1047-L1395）

```
<system>
你是专业的小说编辑和剧情分析师，擅长深度剖析章节内容。
</system>

<task>
【分析任务】
全面分析第{chapter_number}章《{title}》的剧情要素、钩子、伏笔、冲突和角色发展。

【🔴 伏笔追踪任务（重要）】
系统已提供【已埋入伏笔列表】，当你识别到章节中有回收伏笔时：
1. 必须从列表中找出对应的伏笔ID
2. 在 foreshadows 数组中使用 reference_foreshadow_id 字段关联
3. 如果无法确定是哪个伏笔，reference_foreshadow_id 填 null
</task>
```

段落优先级标记（可直接照搬到 gaea 模板引擎）：`<chapter priority="P0">`、`<existing_foreshadows priority="P1">`、`<characters priority="P1">`、`<analysis_framework priority="P0">`、`<output priority="P0">`、`<constraints>`。

**keyword 强约束原文**（L1101、L1358-L1360）：

```
- **关键词**：【必填】从原文逐字复制8-25字的文本片段，用于精确定位
...
✅ keyword字段必填：钩子、伏笔、情节点的keyword不能为空
✅ 逐字复制：keyword必须从原文复制，长度8-25字
✅ 精确定位：keyword能在原文中精确找到
```

**禁止事项**（L1384-L1394，含最有价值的一条反向约束）：

```
❌ 所有章节都打7-8分的"安全分"
❌ 高分章节给大量建议，或低分章节不给建议
❌ suggestions 返回 {{"suggestion": "建议内容"}} 这类对象数组
```

### 6.2 `CHAPTER_REGENERATION_SYSTEM`（L1612-L1649）

```
<system>
你是经验丰富的专业小说编辑和作家，擅长根据反馈意见重新创作章节。
你的任务是根据修改指令，对原始章节进行深度改写和优化。
</system>

<guidelines>
【改写原则】
- **问题导向**：针对修改指令中指出的每个问题进行改进
- **保持精华**：保留原章节中优秀的描写、对话和情节设计
- **深化细节**：增强场景描写、情感渲染和人物刻画
- **节奏优化**：调整叙事节奏，避免拖沓或过快
- **风格一致**：如果提供了写作风格要求，必须严格遵循

【重点关注】
- 如果修改指令提到"节奏"问题，重点调整叙事速度和场景切换
- 如果修改指令提到"情感"问题，重点深化人物内心戏和情感表达
- 如果修改指令提到"描写"问题，重点丰富环境和动作细节
- 如果修改指令提到"对话"问题，重点让对话更自然、更有个性
- 如果修改指令提到"冲突"问题，重点强化矛盾和戏剧张力
</guidelines>

<output>
【输出规范】
直接输出重写后的章节正文内容。
- 不要包含章节标题、序号或其他元信息
- 不要输出任何解释、注释或创作说明
- 从故事内容直接开始，保持叙事的连贯性
</output>
```

### 6.3 `PARTIAL_REGENERATE`（L2414-L2477）

```
<system>
你是一位专业的小说改写助手，擅长根据用户的修改要求精准改写指定段落，同时确保与前后文无缝衔接。
</system>

<task>
【改写任务】
根据用户的修改要求，重写下面选中的文本段落。
【重要要求】
1. 只输出重写后的内容，不要包含任何解释、前缀或后缀
2. 保持与前后文的自然衔接和语气连贯
3. 严格遵循用户的修改要求
4. 保持整体叙事风格的一致性
</task>

<context priority="P0">
【前文参考】（用于衔接，勿重复）
{context_before}

【需要重写的原文】（共{original_word_count}字）
{selected_text}

【后文参考】（用于衔接，勿重复）
{context_after}
</context>

<user_requirements priority="P0">
【用户修改要求】
{user_instructions}

【字数要求】
{length_requirement}
</user_requirements>

<style priority="P1">
【写作风格】
{style_content}
</style>

<constraints>
【必须遵守】
✅ 前后衔接：输出内容必须与前文自然衔接，与后文平滑过渡
✅ 风格一致：保持与原文相同的叙事风格、语气和人称
✅ 要求优先：严格执行用户的修改要求
✅ 字数控制：遵循字数要求

【禁止事项】
❌ 重复前文内容
❌ 重复后文内容
❌ 添加任何元信息或说明
❌ 改变叙事人称或视角
❌ 偏离用户的修改要求
</constraints>
```

### 6.4 `OUTLINE_EXPAND_SINGLE` 的内容边界约束（L1463-L1497）

```
<constraints>
【⚠️ 内容边界约束 - 必须严格遵守】
✅ 只能展开当前大纲节点的内容
✅ 深化当前大纲，而非跨越到后续
✅ 放慢叙事节奏，充分体验当前阶段

❌ 绝对不能推进到后续大纲内容
❌ 不要让剧情快速推进
❌ 不要提前展开【后一节】的内容

【展开原则】
✅ 将单一事件拆解为多个细节丰富的章节
✅ 深入挖掘情感、心理、环境、对话
✅ 每章是当前大纲内容的不同侧面或阶段

【🔴 相邻章节差异化约束（防止重复）】
✅ 每章有独特的开场方式（不同场景、时间点、角色状态）
✅ 每章有独特的结束方式（不同悬念、转折、情感收尾）
✅ key_events在相邻章节间绝不重叠
✅ plot_summary描述该章独特内容，不与其他章雷同
✅ 同一事件的不同阶段要明确区分"前、中、后"
...
```

### 6.5 模板注册表（`get_all_system_templates`，L2863-L3140）中本域条目

| key | name | category | parameters |
|---|---|---|---|
| `CHAPTER_REGENERATION_SYSTEM` | 章节重写系统提示 | 章节重写 | L2965-L2971：`chapter_number, title, word_count, content, modification_instructions, project_context, style_content, target_word_count` |
| `PARTIAL_REGENERATE` | 局部重写 | 章节重写 | L2972-L2978：`context_before, original_word_count, selected_text, context_after, user_instructions, length_requirement, style_content` |
| `PLOT_ANALYSIS` | 情节分析 | 情节分析 | L2979-L2984：`chapter_number, title, content, word_count`（⚠️ **漏登记** `existing_foreshadows`、`characters_info` 两个实际使用参数） |
| `OUTLINE_EXPAND_SINGLE` | 大纲单批次展开 | 情节展开 | L2985-L2993（16 参数） |
| `OUTLINE_EXPAND_MULTI` | 大纲分批展开 | 情节展开 | L2994-L3003（19 参数，多 `previous_context, start_index, end_index`，且 `target_chapter_count` 语义变为"本批章数"） |

> `PLOT_ANALYSIS` 的 `parameters` 与真实调用（6 个参数）不一致：`parameters` 少列 `existing_foreshadows`、`characters_info`。这套 `parameters` 被用于"提示词工坊/模板编辑"的前端提示与预览校验，故属**真实可用性缺陷**。

---

## 7. 源码级发现清单

> 每条含：现象 → 证据 → 对 gaea 的影响。严重度按"是否会导致本轮移植决策跑偏"排序。

### F1 【高】`AI_DENOISING` 模板缺失 → `/polish` 恒 500
- 证据：`api/polish.py:40`、`api/polish.py:114` 使用 `get_template("AI_DENOISING", ...)`；全仓 grep `AI_DENOISING` **仅这两处**；`prompt_service.py` 无该常量，`get_all_system_templates()`（L2863-L3140）也未登记。
- 降级链（L2816-L2860）：DB 无自定义 → `getattr(cls, "AI_DENOISING", None)` → `None` → `format_prompt(None, ...)` → `None.format(...)` → `AttributeError` → `except` → `HTTPException(500)`。
- 额外排除项：`get_all_system_templates()` 尾部会把 Skill 模板并入清单（`prompt_service.py:3130-L3138`，`templates.extend(skill_templates)`），但 Skill 模板键由 `skill_loader.py:69` 的 `_template_key(name) -> "SKILL_{name}"` 生成（L220 写入 `"template_key"`），即 `SKILL_STORY_DESLOP` 等，**仍不会产生 `AI_DENOISING`**。
- **对 gaea 的影响**：不要把 `/polish` 当作可用参考实现；去 AI 味能力的真实参考是 `skills/story-deslop/**` 的文本资产 + 本仓 `novelstyle.DeSlopRewrite`。

### F2 【高】`expansion_strategy` 是空转参数
- 证据：`plot_expansion_service.py:129` 与 `:256` 都写 `strategy_instruction=expansion_strategy`（**直接传原始枚举字符串**）；`api/outlines.py` 三处取 `data.get("expansion_strategy", "balanced")`（L2324、L2489、L2605）后原样下传，无任何映射（grep `climax` 在 `backend/**/*.py` 零命中）；Prompt 只在 L1406-L1408 打印 `【展开策略】{strategy_instruction}`；前端仅把 `balanced/climax/detail` 显示为中文按钮（`Outline.tsx:1407-L1409`）。
- 后果：`balanced` / `climax` / `detail` 三档对模型而言只是三个无解释的英文 token，**几乎没有引导力**。
- **gaea 对策**：`outline-expand` 模板必须把策略写成**自然语言指令块**（每档一段完整中文说明 + 该档的分配规则），而不是透传枚举。

### F3 【高】`used_key_events` 是死变量，跨批去重完全依赖 Prompt 自觉
- 证据：`plot_expansion_service.py:189` 定义 `used_key_events = set()`；全文件 grep `used_key_events` **仅此 1 处命中**，之后从未写入或读取。真实去重是 L218-L221 从 `all_chapter_plans` 现算 `all_used_events[-20:]` 拼进 Prompt 文字。
- 后果：① 超过 20 条历史事件即丢失约束；② 无任何程序化校验，重复 `key_events` 只能靠模型自律。
- **gaea 对策**：去重必须做成**程序侧硬校验**（如 `key_events` 归一化后与历史集合求交，命中即重试该批或标记告警）。

### F4 【高】`enable_scene_analysis` / `scene_instruction` / `scene_field` 三条线全部未接线
- 证据：`enable_scene_analysis` 的 grep 命中全部是形参与透传（L32、L47、L66、L79、L93、L161）；`scene_instruction=""`、`scene_field=""` 在所有 `format_prompt` 调用点都是空串（L131-L132、L260-L261）；模板 L1454/L1566 的 `"estimated_words": 3000{scene_field}` 因此恒为 `3000`。
- **gaea 对策**：要么把场景级字段真正接上（`scenes: [{location, characters, purpose}]`，注意 MuMu 的 `schemas/chapter.py:171-L176` 已有 `SceneData` 模型但 `_parse_expansion_response` 从不填充），要么在 gaea 侧明确删除该参数，不保留空档。

### F5 【中】局部重写无版本、无回滚
- 证据：`apply-partial-regenerate`（`api/chapters.py:5178-L5250`）直接 `content[:start] + new_text + content[end:]` 落库（L5224），无 `RegenerationTask`、无快照；`RegenerationTask` 仅服务于整章重写路径。
- **gaea 对策**：局部重写必须与整章重写共用同一版本快照机制（见 §8.3 `RewriteVersion`），否则"接受并应用"是不可逆操作。

### F6 【中】整章重写的 diff 统计不落库
- 证据：`regenerator.calculate_content_diff(...)` 结果只在 SSE `result` 事件返回（`api/chapters.py:4742-L4759`）；`models/regeneration_task.py` 无 `diff_stats`/`similarity` 列。
- **gaea 对策**：`similarity`（0-100）是"重写是否真的改动了"的廉价质量信号，应落库并纳入 UI 展示（可参考 gaea 已有的 `RewriteReport.BeforeScore/AfterScore` 模式）。

### F7 【中】`_recover_stale_analysis_tasks` 的文档与实现两处不一致
- 证据：`GET /{chapter_id}/analysis/status` 的 docstring 声明两条规则——"running 且超过 **1 分钟**未更新 → failed"、"**pending** 且超过 2 分钟未启动 → failed"（L2993-L2995）。而实现 L2449-L2487：
  1. 阈值实际是 `ANALYSIS_TASK_STALE_SECONDS = 720`（**12 分钟**，L71），非 1 分钟；
  2. `error_message` 赋值只有 `if task.status == "running":` 一个分支（L2459-L2462），**没有 pending 分支**。
- 后果：`pending` 任务若协程从未启动（进程重启等），会永久停在 `pending`，且被"一键分析"当成"正在执行"跳过（L3204-L3206 只跳过 `pending|running`）—— 项目会永久卡住这些章节。
- **gaea 对策**：任务模型必须有"孤儿 pending"清理（按 `created_at` 超时判失败），且**文档阈值必须以常量为唯一来源**，禁止在 docstring 里硬写数字。

### F8 【中】前端轮询窗口与后端陈旧恢复窗口错配 → `auto_recovered` 对 UI 不可见
- 证据：前端 `deadline = Date.now() + 11 * 60 * 1000`（`components/ChapterAnalysis.tsx:172`）；后端硬超时 600s（`api/chapters.py:70`）、陈旧判定 720s（L71）。若协程静默死亡（不写终态），该任务会一直显示 `running`：
  - 前端在 660s 触发 `setError('查询分析状态超时，请关闭后重新打开')`（L176-L179）—— 但 DB 里任务**仍是 running**；
  - 后端要到 720s 才在**下一次状态查询时**把它改为 `failed` + `auto_recovered=true`（L2449-L2487）。
  即：用户先看到"超时"错误，`auto_recovered` 永远不在本次会话被前端看到；要等用户重开弹窗（再查一次状态）才自愈。
- **gaea 对策**：前端轮询上限必须 ≥ 后端陈旧判定窗口，或改为后端主动推事件；错误文案应区分"任务仍在运行（可继续等）"与"任务已失败"。

### F9 【中】分析截断 8000 字与"keyword 逐字复制"存在结构性冲突
- 证据：`plot_analyzer.py:66` 硬截断前 8000 字送入模型；Prompt 要求 `keyword` 从**原文**逐字复制（L1359）。因此 8000 字之后的钩子/伏笔无法产出合法 keyword。而 `_find_text_position` 又在**全文**上搜索（L488-L531），一旦模型凭"印象"编造 keyword，就可能匹配到全文的别处，产生错位标注。
- **gaea 对策**：① 截断与 keyword 使用同一份文本（一致口径）；② keyword 命中失败时**丢弃该条标注**而不是留 `-1`（当前 `-1` 会被下游当作"无位置"并触发回查重算，见 L3385）。

### F10 【中】分析结果字段校验清单不完整
- 证据：`_parse_analysis_response` 只补 `hooks / plot_points / scores` 三个字段（L288-L292）。`foreshadows`、`character_states`、`conflict`、`emotional_arc`、`suggestions`、`plot_stage`、`scenes` 缺失时不补，全依赖下游 `.get(k, default)`。
- 后果：`character_states` 缺失 → 职业/状态/关系/组织四条联动链全部跳过（L1287、L1317、L1355 的 `if analysis_result.get(...)` 守卫），**静默不更新任何实体**。
- **gaea 对策**：分析结果落库前做**显式 Schema 校验 + 缺字段告警**（区分"确实没有"与"模型没按格式回"）。

### F11 【中】前后端契约不一致：`entity_changes` 永不到达分析弹窗
- 证据：前端 `types/index.ts:872-L882` 的 `ChapterAnalysisResponse` 含 `entity_changes`，`components/ChapterAnalysis.tsx:385-L392` 据此渲染"实体联动更新"卡片；但该组件取数自 `GET /api/chapters/{id}/analysis`（L157），而后端该接口返回体（`api/chapters.py:3310-L3328`）**只有** `chapter_id / analysis / memories / created_at`。`entity_changes` 只由另一套接口返回（`api/memories.py:275-L282`，路由定义在 L24）。
- **gaea 对策**：坚持"一个能力一个入口"，不要复制 MuMu 的双编排。

### F12 【低】`PLOT_ANALYSIS` 模板元数据的 `parameters` 漏项
- 证据：`prompt_service.py:2979-L2984` 登记 4 个参数，真实调用传 6 个（`plot_analyzer.py:83-L91`）。
- 影响：模板编辑 UI 的参数提示不完整，用户在"提示词工坊"里改模板时看不到 `existing_foreshadows`/`characters_info`。

### F13 【低】局部重写位置校正窗口仅 ±50 字符
- 证据：`api/chapters.py:4972-L4973` `search_start = max(0, start-50)`、`search_end = min(len, end+50)`。
- 后果：若正文在选段之前发生过 ≥50 字的编辑（例如上一段落被改），必报 400"选中的文本与章节内容不匹配"。
- **gaea 对策**：改用"全文查找 + 唯一性校验"（找到 1 处则校正，找到多处或 0 处才拒绝），而不是窗口查找。

### F14 【低】前端把"前 3 条建议"硬编码为 high 优先级
- 证据：`components/ChapterAnalysis.tsx:372-L379` `priority: index < 3 ? 'high' : 'medium'`。而后端的建议数量本身由 `overall` 分数决定（Prompt L1217-L1220：`overall≥8.0` 时只给 0-1 条）。
- 后果：高分章节的建议被无差别标红。
- **gaea 对策**：优先级应由**建议自身的类型**（`【节奏问题】【描写不足】…` 已在建议文本里，Prompt L1350-L1351）解析得出，或由分析结果额外返回结构化 `category`。

---

## 8. gaea 落地实施规格

### 8.1 现状盘点（gaea 已有 vs MuMu 可借）

| 能力点 | gaea 现状 | 证据 | MuMu 对应 | 结论 |
|---|---|---|---|---|
| 章节多维分析 | `analysis.AnalysisResult` 9 字段（hook 字符串 / foreshadows / conflict 字符串 / emotion_curve 字符串 / character_states / key_events / scene_rhythm / quality_score int / improvement_tips ≤3） | `internal/analysis/analysis.go:35-L45` | 9 维度全结构化 + 三维评分 + 分档建议 | 🔧 **升级**：字段类型要结构化（评分拆维度、建议带上限联动、加 keyword 锚点） |
| 分析 Prompt | `prompts/analysis-chapter.json`（system/task/input_sections/output/constraints 五段，单行 JSON） | `prompts/analysis-chapter.json:1` | `PLOT_ANALYSIS` 带 priority 标签 + 分档评分标准 | 🔧 **升级**：补评分 rubric 与建议数量联动 |
| JSON 容错 | `util.RetryJSON(ctx, caller, sys, usr, 2)`（重试 2 次） | `internal/analysis/analysis.go:97` | 3 层重试 + 指数退避 + 6 层 JSON 清洗 | 🔧 **升级**：重试时把失败原因回注 Prompt（MuMu 缺失，可对 gaea 加分） |
| 伏笔同步 | 稳定 ID `sha256(category+chapterFile+description)[:8]`，增量化合并，去重保留首条 | `internal/analysis/analysis.go:120-L183`、`GenerateStableID` L212-L215 | 序号式 memory id + 先删后建 | ✅ **gaea 更优**，保留；把 MuMu 的 `resolve_status` 四态（must_resolve_now/overdue/not_yet/no_plan）**并入** gaea 的伏笔清单注入 |
| 角色状态同步 | `syncCharacterStates` 按名覆盖 `Status` | `internal/analysis/analysis.go:186-L209` | `state_before → state_after` + `psychological_change` + 关系/职业/组织四链 | 🔧 **升级**：加"前态→后态"与关系/组织变化 |
| 全书审稿 | `BookReviewResult`（letter/total_score/scores map/peaks/valleys/arc_completions） | `internal/analysis/analysis.go:220-L367` | 无对应（MuMu 无全书审稿） | ✅ **gaea 独有** |
| 章节审查→修订方案 | `ChapterReviewResult` 含 `revise_plan`（"可直接作为 prompt 注入到重写流程"） | `internal/chapter/chapter.go:154-L181` | `analysis.suggestions[]` + `selected_suggestion_indices` | 🔧 **接续**：`revise_plan` 已是现成的"驱动式重写"输入，把它接到重写通道即可 |
| 异步任务化 | ❌ 无（同步 `TimeoutMinutes: 10`） | `internal/analysis/analysis.go:94` | `AnalysisTask` 四态 + 三重超时 + 一键顺序分析 | 🔧 **新增（P1）** |
| 分析标注（正文坐标） | ❌ 无 | — | `keyword` → `chapter_position`/`text_length` + `AnnotatedText` | 🔧 **新增（P1）** |
| 章节规划契约 | `types.ChapterSummary`（Title/Summary/QualityEstimate/EmotionTone/KeyEvents/CharactersAppeared） | `internal/analysis/analysis.go:249-L279` 使用 | `expansion_plan` 七字段（含 `character_focus/narrative_goal/conflict_type/ending_type`） | 🔧 **升级**：补齐 `narrative_goal/conflict_type/ending_type` 三字段 |
| 大纲→多章展开 | `prompts/outline-expand.json` 生成 `children[]`（title/summary 30-50字/order_index/status/key_points） | `prompts/outline-expand.json:1` | 分批 + 跨批差异化 + 双序（sub_index/chapter_number） | 🔧 **升级**：加分批与差异化，`summary` 从 30-50 字提到 200-300 字 |
| 剧情分支 | `PlotBranch`（Title/Summary/CharactersInvolved/CoreConflict/ForeshadowImpact/Tone），`BrainstormBranches` / `ApplyBranch` | `internal/app/plot_branch_handler.go:23-L31`、L34、L155 | 无对应 | ✅ **gaea 独有**，可作为 `FitBranches` 的补充 |
| 去 AI 味（确定性） | `DeSlopRewrite`：词表替换（按词长降序避免截断）+ 标点归一（`…{2,}|\.{6,}|！{2,}|!{2,}`）+ **复测打分** | `internal/novelstyle/rewrite.go:42-L95` | `skills/story-deslop/references/banned-words.md` 的词表 | ✅ **gaea 更工程化**；把 MuMu 的**一级/二级禁用词与禁用句式模板**作为 gaea `words.json` 的扩充输入，并新增"二级禁用词/句式"维度 |
| 去 AI 味（LLM 句级） | `RewriteChapterAiTaste`：打分定位命中句（≤10 句）→ LLM 批量重写 → **篇幅漂移 > 50% 放弃该句** → 复测分数不降则不落盘 | `internal/app/novel_llm_rewrite_handler.go:26-L204`（安全闸 L125-L139） | `PARTIAL_REGENERATE`（选段 + 长度模式）+ 前缀清理 | ✅ **gaea 安全闸更强**；补 MuMu 的**输出前缀清理**（L5135-L5151）与**长度模式四档**（L5054-L5071） |
| 选段局部重写 | ❌ 无 | — | `partial-regenerate-stream` + `apply-partial-regenerate` | 🔧 **新增（P0）** |
| 驱动式整章重写 | ❌ 无（只有整章去味重写） | — | `regenerate-stream` + `selected_suggestion_indices` + `focus_areas` + `preserve_elements` | 🔧 **新增（P0）** |
| 版本与回滚 | ❌ 无 | — | `RegenerationTask` 快照 + `version_number`（但无 restore API） | 🔧 **新增（P0）**，且**要补 MuMu 缺的 restore** |
| 提示词资产 | `prompts/*.json`（16 个） | `prompts/` | `prompt_service.py` 内联 + DB 覆盖 + 工坊 | ✅ gaea 的文件式资产已足够；补 2 个本域模板 |

### 8.2 目标能力模型

本域在 gaea 落成 **4 个能力**，全部挂在既有 `writingState`（Wails 绑定）上，复用 `internal/analysis` 与 `internal/novelstyle`：

| 代号 | 能力 | 新增/改造 | 依赖 |
|---|---|---|---|
| **C1** | `AnalyzeChapter` 升级：结构化 9 维度 + 三维评分 + 建议联动 + keyword 锚点 | 改造 `internal/analysis/analysis.go` + `prompts/analysis-chapter.json` | 无 |
| **C2** | 章节规划契约 `ChapterPlan`（`expansion_plan` 等价物） + 分批展开 + 双序重排 | 新增 `internal/outline/expand.go` + `prompts/outline-expand.json` 升级 | 无 |
| **C3** | 驱动式重写：建议/自定义指令 + 保留元素 + 长度模式 + **版本快照与恢复** | 新增 `internal/rewrite/` + 复用 `internal/novelstyle` 安全闸 | C1（要读建议） |
| **C4** | 分析标注层：`keyword → 正文偏移` + 前端内联高亮 | 新增 `internal/analysis/annotate.go` + `frontend` 组件 | C1 |

### 8.3 数据模型

**C1 · 分析结果（替换 `analysis.AnalysisResult`，向后兼容读取旧 `analysis.json`）**

```go
// internal/analysis/types.go（新增文件；analysis.go 保留改造后的 Agent）

type Keyword struct {
    Text   string `json:"text"`             // 从原文逐字复制，8-25 字
    Pos    int    `json:"pos"`              // 正文字符偏移；-1 = 未命中
    Length int    `json:"length"`
}

type Hook struct {
    Type     string  `json:"type"`          // 悬念|情感|冲突|认知
    Content  string  `json:"content"`
    Strength int     `json:"strength"`      // 1-10
    Position string  `json:"position"`      // 开头|中段|结尾
    Keyword  Keyword `json:"keyword"`
}

type ForeshadowHit struct {
    Title         string   `json:"title"`
    Content       string   `json:"content"`
    Type          string   `json:"type"`     // planted|resolved
    Strength      int      `json:"strength"`
    Subtlety      int      `json:"subtlety"`
    Category      string   `json:"category"` // identity|mystery|item|relationship|event|ability|prophecy
    IsLongTerm    bool     `json:"is_long_term"`
    RelatedChars  []string `json:"related_characters"`
    EstimateResolve int    `json:"estimated_resolve_chapter"`
    ReferenceChapter int   `json:"reference_chapter,omitempty"`
    ReferenceStableID string `json:"reference_stable_id,omitempty"` // 复用 gaea 已有 GenerateStableID
    Keyword       Keyword  `json:"keyword"`
}

type Conflict struct {
    Types             []string `json:"types"`   // 人与人|人与己|人与环境|人与社会
    Parties           []string `json:"parties"`
    Level             int      `json:"level"`   // 1-10
    Description       string   `json:"description"`
    ResolutionProgress float64 `json:"resolution_progress"` // 0.0-1.0
}

type EmotionalArc struct {
    PrimaryEmotion    string   `json:"primary_emotion"`
    Intensity         int      `json:"intensity"`      // 1-10
    Curve             string   `json:"curve"`
    SecondaryEmotions []string `json:"secondary_emotions"`
}

type CharacterStateChangeV2 struct {           // 与既有 CharacterStateChange 并存
    Name                string            `json:"name"`
    OldState            string            `json:"old_state"`
    NewState            string            `json:"new_state"`
    PsychologicalChange string            `json:"psychological_change"`
    KeyEvent            string            `json:"key_event"`
    SurvivalStatus      string            `json:"survival_status,omitempty"` // active|deceased|missing|retired
    RelationshipChanges map[string]string `json:"relationship_changes,omitempty"`
}

type PlotPoint struct {
    Content    string   `json:"content"`
    Type       string   `json:"type"`        // revelation|conflict|resolution|transition
    Importance float64  `json:"importance"`  // 0.0-1.0
    Impact     string   `json:"impact"`
    Keyword    Keyword  `json:"keyword"`
}

type Scene struct {
    Location  string `json:"location"`
    Atmosphere string `json:"atmosphere"`
    Duration  string `json:"duration"`
}

type Scores struct {
    Pacing       float64 `json:"pacing"`       // 1.0-10.0 一位小数
    Engagement   float64 `json:"engagement"`
    Coherence    float64 `json:"coherence"`
    Overall      float64 `json:"overall"`      // (P+E+C)/3 后 ±0.5
    Justification string `json:"score_justification"`
}

type AnalysisV2 struct {
    Hooks            []Hook              `json:"hooks"`
    Foreshadows      []ForeshadowHit     `json:"foreshadows"`
    Conflict         Conflict            `json:"conflict"`
    EmotionalArc     EmotionalArc        `json:"emotional_arc"`
    CharacterStates  []CharacterStateChangeV2 `json:"character_states"`
    PlotPoints       []PlotPoint         `json:"plot_points"`
    Scenes           []Scene             `json:"scenes"`
    Pacing           string              `json:"pacing"`   // slow|moderate|fast|varied
    DialogueRatio    float64             `json:"dialogue_ratio"`
    DescriptionRatio float64             `json:"description_ratio"`
    Scores           Scores              `json:"scores"`
    PlotStage        string              `json:"plot_stage"`
    Suggestions      []string            `json:"suggestions"` // 纯字符串，数量与 Overall 联动
    Summary          string              `json:"summary"`     // 供记忆/检索用
}
```

**C2 · 章节规划（写入章节 front-matter 或独立 `plans.json`，与 gaea 现有 `ChapterSummary` 并存）**

```go
type ChapterPlan struct {
    SubIndex      int      `json:"sub_index"`       // 大纲内相对序（稳定）
    Title         string   `json:"title"`
    PlotSummary   string   `json:"plot_summary"`    // 200-300 字
    KeyEvents     []string `json:"key_events"`      // 2-4 条，跨章不得重复
    CharacterFocus []string `json:"character_focus"`
    EmotionalTone string   `json:"emotional_tone"`
    NarrativeGoal string   `json:"narrative_goal"`
    ConflictType  string   `json:"conflict_type"`
    EndingType    string   `json:"ending_type"`     // 悬念|冲突升级|情节转折|情感收尾|自然过渡
    EstimatedWords int     `json:"estimated_words"`
}
```

**C3 · 重写版本（新增 `internal/rewrite/version.go`）**

```go
type RewriteVersion struct {
    ID             string   `json:"id"`               // uuid
    ChapterNum     int      `json:"chapter_num"`
    Mode           string   `json:"mode"`             // whole|partial|deslop
    Status         string   `json:"status"`           // pending|running|completed|failed|applied|discarded
    Source         string   `json:"source"`           // custom|analysis_suggestions|mixed
    SuggestionIdxs []int    `json:"suggestion_idxs,omitempty"`
    CustomInstr    string   `json:"custom_instructions,omitempty"`
    FocusAreas     []string `json:"focus_areas,omitempty"`
    Preserve       *PreserveConfig `json:"preserve_elements,omitempty"`
    // partial 专用
    StartPos       int      `json:"start_pos,omitempty"`
    EndPos         int      `json:"end_pos,omitempty"`
    LengthMode     string   `json:"length_mode,omitempty"` // similar|expand|condense|custom
    TargetWords    int      `json:"target_word_count"`
    // 内容与度量
    OriginalContent   string `json:"original_content"`      // 快照（partial 时为全文快照，保证可回滚）
    OriginalWordCount int    `json:"original_word_count"`
    NewContent        string `json:"new_content"`
    NewWordCount      int    `json:"new_word_count"`
    Similarity        float64 `json:"similarity"`           // 0-100，difflib 等价物
    BeforeAIScore     int    `json:"before_ai_score,omitempty"` // 复用 novelstyle.TasteScore.Score
    AfterAIScore      int    `json:"after_ai_score,omitempty"`
    CreatedAt         time.Time `json:"created_at"`
    AppliedAt         *time.Time `json:"applied_at,omitempty"`
}

type PreserveConfig struct {
    Structure      bool     `json:"preserve_structure"`
    Dialogues      []string `json:"preserve_dialogues"`
    PlotPoints     []string `json:"preserve_plot_points"`
    CharacterTraits bool    `json:"preserve_character_traits"`
}
```

落盘布局（契合 gaea 的 `.gaea/` 文件式存储）：

```
.gaea/projects/<project>/
  rewrites/
    <chapterNum>/
      <versionID>.json        # RewriteVersion（含 original_content 快照）
      index.json              # [{id, mode, status, similarity, created_at}] 时间倒序
  analysis/
    analysis.json             # 兼容既有
    analysis-v2.json          # AnalysisV2
    annotations.json          # [{type,title,content,importance,pos,length,tags}]
  chapters/
    001.md                    # 既有正文
    plans.json                # map[chapterNum]ChapterPlan
```

### 8.4 算法规格（伪码级）

**A1 · 分析结果校验与锚点回填（C1）**

```
func (a *Agent) AnalyzeV2(ctx, chapterNum, content) (*AnalysisV2, error):
    // 1. 截断口径与 keyword 口径必须一致
    analysisText := content                 // 不截断；若必须截断，keyword 只在 analysisText 内搜
    // 2. 伏笔注入：四态 resolve_status（借 MuMu 算法）
    planted := pm.ReadForeshadows() 过滤 Status == planted
    for f in planted:
        switch:
          f.ResolveTarget == chapterNum        -> "must_resolve_now"
          f.ResolveTarget != 0 && < chapterNum -> "overdue"
          f.ResolveTarget > chapterNum         -> "not_yet"
          default                              -> "no_plan"
    // 3. 三层注入：must_resolve_now 带全文；overdue 取前 5 带标题；其余取前 10 仅 ID+标题
    // 4. LLM（temperature 0.3）+ RetryJSON(2)，重试时把上次错误注入 prompt
    // 5. 硬校验（MuMu 缺失，必须补）
    require: len(Hooks)>=1, len(PlotPoints)>=3, Scores in [1,10], Suggestions 全为 string
    suggestCount := 按 Scores.Overall 分档校验数量；不符 -> 修正或标记 warning
    // 6. 锚点回填（四段式，见 A2）
    for each hook/foreshadow/plotpoint: k := locate(analysisText, k.Text)
        if k.Pos < 0: 丢弃该条锚点（保留内容，annotate=false）   // MuMu 留 -1 会导致错位
    // 7. 分档建议数量联动
    expected := { <4.0: [4,5], 4.0-5.9: [3,4], 6.0-7.9: [1,2], >=8.0: [0,1] }
```

**A2 · 正文定位 `locate`（借 MuMu `_find_text_position`，修正其第 2 段的坐标缺陷）**

```
func locate(full, kw string) Keyword:
    if kw == "" || full == "" : return {-1,0}
    if i := strings.Index(full, kw); i >= 0: return {kw, i, len(kw)}   // 精确，坐标正确
    // 去标点匹配必须做"坐标反投影"：记录被删字符的原始索引
    clean, indexMap := stripPunctWithMap(full)                          // MuMu 缺 indexMap → 坐标偏
    if j := strings.Index(clean, stripPunct(kw)); j >= 0:
        start := indexMap[j]; end := indexMap[min(j+len(kw), len(indexMap)-1)]
        return {kw, start, end-start}
    if len([]rune(kw)) > 10:
        if i := strings.Index(full, string([]rune(kw)[:15])); i >= 0: return {kw, i, ...}
    return {-1, 0}
```

**A3 · 分批展开与跨批硬去重（C2）**

```
func ExpandOutline(node, target, batchSize=5) ([]ChapterPlan, error):
    plans := []ChapterPlan{}
    used := map[string]bool{}                       // 归一化后的 key_events
    for batchStart := 1; batchStart <= target; batchStart += batchSize:
        n := min(batchSize, target-len(plans))
        prev := buildPrevContext(plans, used)       // 全量摘要 + 全量 used（不截 20，走 token 预算裁剪）
        batch, err := callExpandMulti(node, prev, batchStart, batchStart+n-1, n)
        if err != nil: return nil, err
        if len(batch) != n: return nil, fmt.Errorf("批次数不符: got %d want %d", len(batch), n)  // 硬校验
        for i := range batch:
            batch[i].SubIndex = batchStart + i                    // 全局重写，不信模型
            for _, e := range batch[i].KeyEvents:
                k := normalize(e)
                if used[k]: return nil, fmt.Errorf("key_events 跨章重复: %s", e)   // MuMu 只提示，这里硬失败
                used[k] = true
        plans = append(plans, batch...)
```

**A4 · 重写指令构建（C3，逐字复用 MuMu 四段结构）**

```
func buildInstructions(v *RewriteVersion, analysis *AnalysisV2) string:
    out := []string{"# 章节修改指令", ""}
    if v.Source in {analysis_suggestions, mixed} && len(v.SuggestionIdxs) > 0:
        out += "## 📋 需要改进的问题（来自AI分析）：", ""
        for idx in v.SuggestionIdxs:
            if 0 <= idx && idx < len(analysis.Suggestions):
                out += fmt.Sprintf("%d. %s", idx+1, analysis.Suggestions[idx])
    if v.CustomInstr != "": out += "## ✍️ 用户自定义修改要求：", v.CustomInstr, ""
    if len(v.FocusAreas) > 0: out += "## 🎯 重点优化方向：", focusMapLines(v.FocusAreas), ""
    if v.Preserve != nil: out += "## 🔒 必须保留的元素：", preserveLines(v.Preserve), ""
    return strings.Join(out, "\n")
```

`focusMap`（逐字照搬 `chapter_regenerator.py:144-L149`）：`pacing/emotion/description/dialogue/conflict` → 五条中文指令。未知 key **必须报错**（MuMu 静默丢弃，见 L152-L153）。

**A5 · 重写安全闸（合并 gaea 已有 + MuMu 缺失项）**

```
func applyRewrite(v *RewriteVersion, newText string) (string, error):
    cleaned := stripPrefixes(newText, MU_PREFIXES_8)   // 借 MuMu L5135-L5151
    cleaned = trimPairedQuotes(cleaned)                // " ' 「」 『』
    before, _ := novelstyle.ScoreTextNoRef(v.OriginalContent)
    after,  _ := novelstyle.ScoreTextNoRef(cleaned)
    sim := similarity(v.OriginalContent, cleaned)      // 0-100
    // gaea 已有闸（novel_llm_rewrite_handler.go:125-L139）推广到整章
    if v.Mode == "deslop" && after.Score >= before.Score: return reject("未改善，不落盘")
    if v.Mode == "whole" && len([]rune(cleaned)) < v.TargetWords/2: return reject("篇幅严重不足")
    v.NewContent, v.NewWordCount, v.Similarity = cleaned, len([]rune(cleaned)), sim
    // 落盘：快照 original_content → 写 version 文件 → status=completed
    // 只有显式 ApplyRewriteVersion 才把 cleaned 写入 章节正文
```

**A6 · 版本恢复（MuMu 完全缺失，gaea 必须补）**

```
func RestoreRewriteVersion(versionID string) error:
    v := loadVersion(versionID)
    if v.Status != "applied": return fmt.Errorf("该版本未应用，无需恢复")
    // 恢复 = 把 OriginalContent 写回章节；并生成一条 mode=restore 的新版本（审计链不断）
```

### 8.5 接口规格（Wails 绑定，沿用 `writingState` 方法命名）

| 方法 | 签名 | 说明 | 对应 MuMu |
|---|---|---|---|
| `AnalyzeChapterV2` | `(chapterNum int) (map[string]interface{}, error)` | 同步分析，返回 `AnalysisV2` + 落盘 | `POST /chapters/{id}/analyze`（异步版见 P1） |
| `GetChapterAnalysis` | `(chapterNum int) (map[string]interface{}, error)` | 读已存分析结果 + 标注 | `GET /chapters/{id}/analysis` |
| `GetChapterAnnotations` | `(chapterNum int) ([]Annotation, error)` | keyword 锚点列表 | `GET /chapters/{id}/annotations` |
| `ExpandOutlineNode` | `(nodeID string, target, batchSize int, strategy string) ([]ChapterPlan, error)` | 分批展开 + 跨批硬去重 | `analyze_outline_for_chapters` |
| `ApplyChapterPlans` | `(nodeID string, plans []ChapterPlan) error` | 落库 + 双序重排 | `create_chapters_from_plans` + `_renumber_subsequent_chapters` |
| `RewriteChapter` | `(chapterNum int, req RewriteRequest) (*RewriteVersion, error)` | 整章重写（含安全闸），只产生版本不落章 | `regenerate-stream` |
| `RewriteSelection` | `(chapterNum int, start, end int, instr, lengthMode string) (*RewriteVersion, error)` | 选段重写 | `partial-regenerate-stream` |
| `ApplyRewriteVersion` | `(versionID string) error` | 应用版本到正文 | `PUT /chapters/{id}` + `apply-partial-regenerate` |
| `DiscardRewriteVersion` | `(versionID string) error` | 放弃 | 前端"放弃新内容" |
| `RestoreRewriteVersion` | `(versionID string) error` | **回滚**（MuMu 无） | — |
| `ListRewriteVersions` | `(chapterNum int) ([]RewriteVersionSummary, error)` | 版本历史 | `GET /chapters/{id}/regeneration/tasks` |

`RewriteRequest`：

```go
type RewriteRequest struct {
    Source             string          `json:"source"`              // custom|analysis_suggestions|mixed
    SuggestionIdxs     []int           `json:"suggestion_indices"`
    CustomInstructions string          `json:"custom_instructions"`
    FocusAreas         []string        `json:"focus_areas"`
    Preserve           *PreserveConfig `json:"preserve_elements"`
    StyleID            int             `json:"style_id"`
    TargetWordCount    int             `json:"target_word_count"`   // 500..10000，默认 3000
    AutoApply          bool            `json:"auto_apply"`          // 默认 false
}
```

### 8.6 Prompt 资产改造

| 文件 | 动作 | 要点 |
|---|---|---|
| `prompts/analysis-chapter.json` | **重写** | ① `output` 换成 §3.4 的完整 Schema；② 新增 `scoring_rubric` 段（三维 5 档 + overall 公式 + 禁止安全分）；③ 新增 `suggestions_policy`（数量与 Overall 联动 + 必须纯字符串 + 每条须含 `【问题类型】`）；④ `keyword` 必须逐字复制 8-25 字；⑤ 新增 `existing_foreshadows` 输入段（四态标记） |
| `prompts/outline-expand.json` | **重写** | ① 输出字段扩到 §8.3 `ChapterPlan` 七字段；② `summary` 从 30-50 字提到 200-300 字；③ 新增 `strategy_block`：把 `balanced/climax/detail` 展开为**三整段中文指令**（不是枚举透传）；④ 新增 `prev_context` 输入段（全量摘要 + 已用 key_events + 三条差异化强制要求，逐字借 MuMu L230-L237）；⑤ `constraints` 加"不越界到后续大纲""key_events 相邻章不重叠" |
| `prompts/chapter-rewrite.json` | **新增** | 由 `CHAPTER_REGENERATION_SYSTEM`（L1612-L1649）+ `get_chapter_regeneration_prompt` 的六段拼接（L2659-L2751）合成：`original_chapter / modification_instructions / project_context / characters / chapter_outline / style / requirements / start_writing` |
| `prompts/partial-rewrite.json` | **新增** | 逐字移植 `PARTIAL_REGENERATE`（L2414-L2477），保留 `<context priority="P0">` / `<user_requirements priority="P0">` / `<style priority="P1">` / `<constraints>` 四段结构 |
| `prompts/deslop.json` | **新增** | 由 `story-deslop/SKILL.md` Gate A-F（L104-L170）+ `banned-words.md` + `anti-ai-writing.md` 的"7 种模式 + 三遍法 + 12 字复用禁令"压缩成一个可注入模板 |
| `.gaea/skills/novel-deslop/words.json` | **扩充** | 并入 `banned-words.md` 的七类一级禁用词 + 三类二级禁用词（二级词以"密度阈值"形式建模，见 §8.7） |

**策略块示例（替代 MuMu 的空转枚举）**：

```
【展开策略 · 均衡分配 (balanced)】
- 将本节点内容按"起—承—转—合"四段均衡切分到 N 章
- 每章篇幅目标差异不超过 20%；关键事件按时间顺序自然分布
- 高潮点安排在第 ceil(N*0.75) 章附近，前后各留 1 章铺垫与余韵

【展开策略 · 高潮重点 (climax)】
- 前 ceil(N*0.5) 章为压缩铺垫，单章只推进 1 个关键事件
- 第 ceil(N*0.8) 章起进入高潮，允许 2-3 章连续高强度冲突
- 每章结尾必须使用不同类型的钩子（禁止连续同型）

【展开策略 · 细节丰富 (detail)】
- 把每个关键事件拆成"前/中/后"三个阶段，各占 1 章
- 强化环境、心理、对话细节；单章可推进 0 个新事件，只深化既有事件
- 允许放慢节奏，但每章必须有独立的叙事价值
```

### 8.7 UI 规格

**U1 章节分析面板**（等价 `components/ChapterAnalysis.tsx`，901 行 → gaea 版建议拆成 3 个子组件）

- 6 个 Tab：概览 / 钩子 / 伏笔 / 情感曲线 / 角色 / 记忆。
- 概览区：4 个 `Statistic`（整体质量 / 节奏 / 吸引力 / 连贯性，均 `x / 10`）+ `analysis_report` 预格式文本（`white-space: pre-wrap`）+ 建议列表（纯字符串按序渲染）+ **"根据建议重新生成"主按钮**（仅当 `suggestions.length > 0`）。
- 钩子区：`type` / `position` / `强度 x/10` 三个标签 + 描述。
- 伏笔区：`已埋下|已回收` + `强度` + `隐藏度` + `呼应第N章` + 描述。
- 情感曲线区：主导情绪、情感强度（`emotional_intensity * 10`）、剧情阶段、冲突等级、冲突类型标签组。
- 角色区：`状态变化：A → B` / 心理变化 / 关键事件 / 关系变化标签。
- **新增（MuMu 缺）**：把 `keyword` 锚点渲染为可点击跳转，点击后正文滚动到 `pos` 并高亮 `length` 字。
- **优先级来源修正**：建议的红/橙/蓝标签由建议文本前缀 `【…】` 解析，不用 `index < 3` 硬编码（见 F14）。

**U2 标注阅读器**（等价 `pages/ChapterAnalysis.tsx`，571 行）

- 三栏：章节列表（280px）/ 正文（`AnnotatedText` 内联高亮）/ 标注侧栏（400px）。
- 双向点击联动 + 高亮 Switcher（`显示标注`）。
- 顶部统计条：`共有 N 个标注：🎣x个钩子 🌟y个伏笔 💎z个情节点 👤w个角色事件`。
- 移动端（`< 768px`）：章节列表与标注侧栏改 Drawer，工具栏折成两行。
- **竞态防护照搬**：用递增 `generation` 序号丢弃过期响应（`ChapterAnalysis.tsx:45-L50`、L88、L113）。

**U3 重写工作台**（等价 `ChapterRegenerationModal` + `ChapterContentComparison` + `PartialRegenerateModal`，三件套）

- 表单：修改来源三选一（仅自定义 / 仅分析建议 / 混合）→ 建议多选（带建议文本）→ 自定义指令 TextArea（`showCount`，`maxLength=1000`）→ 高级折叠：重点优化方向 5 复选、保留元素（**MuMu 只有 2 项，gaea 必须补 `preserve_dialogues` / `preserve_plot_points` 的输入控件**，见 §5.5）、目标字数（500-10000，步进 500）。
- 进度：复用 gaea 既有 SSE/事件流；展示 `已生成 N 字`。
- 对比：**split / unified 双视图**（MuMu 用 `react-diff-viewer-continued`）+ 4 个统计（原文字数 / 新文字数 / 字数变化 / 变化比例 %）+ **新增 AI 味分数前后对比**（复用 `novelstyle.TasteScore`，MuMu 完全没有这个信息）。
- 出口 4 个（MuMu 3 个）：应用 / 放弃 / 切换视图 / **回滚到该版本**。
- 局部重写弹窗：选中文本展示（`maxHeight: 150`）+ 重写要求必填 + 长度控制 4 档（保持长度 / 扩展内容 / 精简内容 / 自定义，自定义显示 `InputNumber` 10-10000 步进 50）+ 生成区（流式追加 + 闪烁光标 + 字数对比 Alert）+ 接受并应用 / 重新生成。

### 8.8 分阶段路线与验收

| 阶段 | 内容 | 验收标准（可执行） |
|---|---|---|
| **P0-1** | C1：`AnalysisV2` 类型 + `prompts/analysis-chapter.json` 重写 + 锚点回填 + 硬校验 | ① 若已有测试项目，对新章节跑 `AnalyzeChapterV2`，产物 `analysis-v2.json` 能被 `json.Unmarshal` 零错误；② 所有 `keyword.Pos >= 0` 的锚点，`content["pos":"pos+length"] == keyword.Text` 逐字成立；③ `len(Suggestions)` 落在 Overall 对应的区间内 |
| **P0-2** | C3：`internal/rewrite/`（版本模型 + 指令构建 + 安全闸 + 应用/放弃/回滚） + `prompts/chapter-rewrite.json`、`prompts/partial-rewrite.json` | ① 整章重写产出版本文件且**章节正文未被修改**；② `ApplyRewriteVersion` 后正文变化，`RestoreRewriteVersion` 后**逐字节回到原文**；③ `DiscardRewriteVersion` 后正文不变；④ 局部重写的位置校正：把选段前插入 100 字后仍能正确定位（MuMu ±50 窗口会失败，见 F13） |
| **P1-1** | C2：`internal/outline/expand.go` 分批 + 跨批硬去重 + 双序重排 + `prompts/outline-expand.json` 重写 | ① 目标 12 章（>batchSize 5）时产出恰好 12 个 `ChapterPlan`，`SubIndex` 为 1..12 连续；② 注入一条历史 `key_events` 使其在第二批重现，确认**硬失败并给出事件名**（而不是静默通过）；③ 三档策略各跑一次，产出在"章数分布"上可观察差异（证明策略不再空转） |
| **P1-2** | C4：标注层 + U1/U2 前端；异步任务模型（`pending/running/completed/failed` + 硬超时 + 孤儿 pending 清理 + 陈旧 running 恢复） | ① 前端断开再连可以看到 running 任务的状态与进度；② 手动 kill 协程后，任务在 720s 内被标 failed（含 **pending 分支**，修 F7）；③ 轮询上限 ≥ 陈旧判定窗口（修 F8） |
| **P2** | 去 AI 味资产合并：把 `banned-words.md` 七类词并入 `words.json`；二级词以**密度阈值**建模（"每 1000 字 > 3 个"）；新增句式检测维度；`prompts/deslop.json` | ① 对同一批章节，合并词表后 `DeSlopRewrite` 的 `AfterScore < BeforeScore` 命中率对比基线提升；② 密度类规则（微微/淡淡/缓缓/轻轻）能被单独报告计数 |

**明确不做**（避免范围蔓延）：

- 不移植 MuMu 的 SQLAlchemy/异步 DB 层（gaea 是文件式 + Wails，任务状态落 JSON 即可）。
- 不移植 `api/memories.py` 的重复分析编排（F11）。
- 不移植 MuMu 的 `_find_text_position` 第 2 段"去标点后返回净化坐标"的错误实现（改为带索引映射的反投影，见 A2）。
- 不引入 Python 依赖（`difflib`/`json5`）：相似度用 gaea 已有分词 + 集合交并实现；JSON 容错复用 `util.ExtractJSON`，不引入 json5。

---

## 9. 未能从源码确认 / 存疑项

1. **`schemas/chapter.py` L223-L228 的 `context_chars` 默认值**：`read` 输出显示 `Field(` 在 L223 起始、L229 是 `style_id`，中间 L224-L228 的精确定义文本在本轮未逐字读取（前端固定传 500）。**未能从源码确认**默认数值。
2. **`length_mode` / `target_word_count` 的 Schema 校验边界**：`schemas/chapter.py` L230-L234 起始，具体 `ge/le` 值未逐字读取。**未能从源码确认**。
3. **`AnalysisData` 前端接口（`types/index.ts`）的完整字段集**：只读了 L860-L882 附近；`AnalysisData` 定义本身未定位。**未能从源码确认**其是否与 `to_dict()` 的 21 键一一对应。
4. **`_schedule_analysis_background` 的实现**：`api/chapters.py` 中被调用（L3246、L3550 使用 `background_tasks`），但其定义体未读取，因此"一键分析是否与 FastAPI BackgroundTasks 共用同一执行器"**未能从源码确认**。
5. **`story-deslop` Skill 的加载与执行链**：`skill_loader.py` L285-L288 确认了触发词到 `SKILL_STORY_DESLOP` 的映射，但 Skill 如何被 project_agent 调用成一次实际改写（是否注入 System Prompt、是否强制加载 references）**未能从源码确认**。
6. **`hooks_avg_strength` / `emotional_curve` / `PlotAnalysis.word_count`** 三列：全仓 grep 未发现写入点，但未做穷尽搜索，**可能**存在其他写入路径（如 `import_export_service.py`）。标注为存疑。
7. **`AI_DENOISING` 是否存在于某个迁移的种子数据**：未检索 `alembic/` 与 `backend/app/constants/`。若存在种子模板，F1 的严重度降为"默认部署下 500"。

---

## 10. 附录：关键引用索引

```
# 分析核心
clones/MuMuAINovel@600be703:backend/app/services/plot_analyzer.py:19-30          # PlotAnalyzer
clones/MuMuAINovel@600be703:backend/app/services/plot_analyzer.py:32-194         # analyze_chapter + 重试
clones/MuMuAINovel@600be703:backend/app/services/plot_analyzer.py:66             # 8000 字截断
clones/MuMuAINovel@600be703:backend/app/services/plot_analyzer.py:196-268        # 伏笔三层注入
clones/MuMuAINovel@600be703:backend/app/services/plot_analyzer.py:270-303        # 解析 + 缺字段补空
clones/MuMuAINovel@600be703:backend/app/services/plot_analyzer.py:305-486        # 记忆提取阈值
clones/MuMuAINovel@600be703:backend/app/services/plot_analyzer.py:488-531        # 四段式定位
clones/MuMuAINovel@600be703:backend/app/services/plot_analyzer.py:533-591        # 报告生成

# 分析 Prompt
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1047-1395     # PLOT_ANALYSIS 全文
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1186-1213     # 三维评分 rubric
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1215-1225     # 建议数量 ↔ 分数
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1228-1353     # 输出 Schema
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1356-1395     # 约束区
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:2979-2984     # PLOT_ANALYSIS 元数据

# 异步任务与编排
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:70-71                    # 600s / 720s
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:98-136                   # 终态独立事务
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:886-922                  # 硬超时 + 取消兜底
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:925-1433                 # 分析主体
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:2449-2487                # 陈旧恢复（仅 running）
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:2984-3100                # 状态查询（单/批）
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:3134-3262                # 一键分析未分析章节
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:3265-3474                # 结果 / 标注
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:3477-3563                # 触发分析
clones/MuMuAINovel@600be703:backend/app/models/analysis_task.py:8-42

# 展开
clones/MuMuAINovel@600be703:backend/app/services/plot_expansion_service.py:25-84  # 分派
clones/MuMuAINovel@600be703:backend/app/services/plot_expansion_service.py:154-289 # 分批 + 差异化
clones/MuMuAINovel@600be703:backend/app/services/plot_expansion_service.py:189   # 死变量 used_key_events
clones/MuMuAINovel@600be703:backend/app/services/plot_expansion_service.py:381-482 # 落库 + 双序
clones/MuMuAINovel@600be703:backend/app/services/plot_expansion_service.py:524-597 # 解析 + 静默降级
clones/MuMuAINovel@600be703:backend/app/services/plot_expansion_service.py:600-685 # 序号重排
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1398-1498     # OUTLINE_EXPAND_SINGLE
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1501-1609     # OUTLINE_EXPAND_MULTI
clones/MuMuAINovel@600be703:backend/app/schemas/chapter.py:171-212               # SceneData / ExpansionPlan

# 重写
clones/MuMuAINovel@600be703:backend/app/services/chapter_regenerator.py:22-108    # 主流程 + 进度
clones/MuMuAINovel@600be703:backend/app/services/chapter_regenerator.py:110-179  # 指令四段 + focus_map
clones/MuMuAINovel@600be703:backend/app/services/chapter_regenerator.py:181-205  # Prompt 组装
clones/MuMuAINovel@600be703:backend/app/services/chapter_regenerator.py:207-237  # difflib diff
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1612-1649     # CHAPTER_REGENERATION_SYSTEM
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:2414-2477     # PARTIAL_REGENERATE
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:2628-2753     # 整章重写 Prompt 组装
clones/MuMuAINovel@600be703:backend/app/schemas/regeneration.py:7-38             # 请求契约
clones/MuMuAINovel@600be703:backend/app/models/regeneration_task.py:8-51         # 版本快照
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:4449-4798                # 整章重写 SSE
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:4922-5175                # 局部重写 SSE
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:5178-5250                # 应用局部重写（无回滚）
clones/MuMuAINovel@600be703:backend/app/api/polish.py:17-141                     # AI 去味（F1 缺陷）
clones/MuMuAINovel@600be703:backend/app/skills/story-deslop/SKILL.md:104-170     # 六道门禁
clones/MuMuAINovel@600be703:backend/app/skills/story-deslop/references/banned-words.md:3-59
clones/MuMuAINovel@600be703:backend/app/skills/story-deslop/references/anti-ai-writing.md:179-259 # 7 模式 + 三遍法

# JSON 容错
clones/MuMuAINovel@600be703:backend/app/services/json_helper.py:365-534          # clean_json_response
clones/MuMuAINovel@600be703:backend/app/services/json_helper.py:565-606          # loads_json 四级降级

# 前端
clones/MuMuAINovel@600be703:frontend/src/components/ChapterAnalysis.tsx:29-211   # 取数 + 2s 轮询 + 11min 上限
clones/MuMuAINovel@600be703:frontend/src/components/ChapterAnalysis.tsx:372-379  # index<3 → high（F14）
clones/MuMuAINovel@600be703:frontend/src/components/ChapterAnalysis.tsx:382-753  # 6 Tab 渲染
clones/MuMuAINovel@600be703:frontend/src/components/ChapterRegenerationModal.tsx:94-210  # 表单 + SSE
clones/MuMuAINovel@600be703:frontend/src/components/ChapterContentComparison.tsx:88-154 # 应用/放弃
clones/MuMuAINovel@600be703:frontend/src/components/PartialRegenerateModal.tsx:69-166    # 局部重写
clones/MuMuAINovel@600be703:frontend/src/pages/ChapterAnalysis.tsx:65-571        # 标注阅读器

# Prompt 元数据 / 工坊
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:2793-2860     # get_template 降级链
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:2863-3140     # get_all_system_templates
clones/MuMuAINovel@600be703:backend/app/services/skill_loader.py:285-288         # 去味触发词
```
