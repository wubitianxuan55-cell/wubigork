# 能力域 T2 · 拆书 / 导入反推 —— MuMuAINovel 源码蒸馏与 gaea 落地规格

- **蒸馏对象**：`clones/MuMuAINovel`（GitHub `xiamuceer-j/MuMuAINovel`，Python FastAPI + React）
- **目标落地方**：gaea（Go + Wails + React）小说板块
- **本轮边界**：只读源码 + 产出规格；**不写生产代码**
- **唯一事实来源**：`clones/MuMuAINovel` 工作区克隆；本文所有结论均可由「文件:行号」回查，无臆测
- **任务**：t2（researcher-bookimport）

---

## 0. 十句结论（先读这段）

1. **MuMu 的「拆书」= 成品书导入 + 反向立项**：上传 TXT → 章节切分 → **AI 反推项目信息（简介/主题/类型/视角/目标字数）+ AI 反推逐章大纲** → 前端人工修订 → 落库 → **落库后自动补全世界观/职业/角色三件套**。它不做「拆文风/拆爽点/拆桥段」——那是 Skill 文本里的事，不是这条流水线。
2. **流水线被切成两个物理阶段**：阶段 A `create_task`（纯解析 + AI 反推，产出**预览**，只进内存不落库），阶段 B `apply_import_stream`（人工确认后落库 + 三次 AI 生成）。阶段之间**只有内存任务对象**连接——这是 MuMu 最大的工程短板，也是 gaea 必须反着做的第一件事。
3. **分章是规则优先 + 三级兜底**：强正则（`第X章/节/回/卷/集/部/篇`、`chapter N`、`chap. N`）→ 弱模式（≤25 字、无标点、前后空行）→ **固定窗口 3000–5000 字 + 标点边界**（`txt_parser_service.py:15-19, 116-168`）。编码按 `utf-8 → utf-8-sig → gb18030 → gbk → big5` 顺序试解，最后 `utf-8(ignore)` 兜底不抛错（`txt_parser_service.py:21-37`）。
4. **解析范围是「末 N 章」而不是整本**：`extract_mode=tail` 默认只取末 **10** 章，N 必须是 5 的倍数，>50 自动降级为整本（`api/book_import.py:53-59`，`book_import_service.py:104-109`）。产品意图：**用末尾几章反推「这本书接下来该怎么写」**，而不是全文入库。
5. **AI 反推大纲是「5 章一批 + 结构强对齐」**：`batch_size=5`，每批独立调用 `BOOK_IMPORT_REVERSE_OUTLINES`，要求数组长度严格 == 批内章数；**批内数量不齐就地用规则兜底结构补齐**（`book_import_service.py:1291-1341`、`1375-1444`）。这是全文最值得抄的工程手法。
6. **Prompt 工程范式 = `<system>/<task>/<input priority=P0>/<output priority=P0>/<constraints>` 五段式**，输入输出都标 P0，字段级约束写进 prompt，禁止 markdown 代码块，并要求"仅输出纯 JSON 数组"。两段拆书 Prompt 原文在 `prompt_service.py:2480-2531` 与 `2534-2607`。
7. **JSON 重试不是简单重试**：`call_with_json_retry` 失败时把**上次错误回复前 200 字**拼成提示语追加到 prompt 尾部再试（`ai_service.py:638-690`）——「负反馈注入式重试」，最多 3 次。gaea 应直接抄这条。
8. **落库七步链**（`apply_import_stream`）：建项目(0-5%) → 大纲(5-10%) → 章节(10-20%) → 世界观(20-40%) → 职业(40-65%) → 角色/组织(65-92%) → 提交(92-98%)。**后三步各自 try/except 隔离，单步失败不阻断后续**，失败步骤记入 `task.failed_steps` 并通过 SSE 的 `status="step_failures"` 特殊消息回传前端，由前端渲染「智能重试 / 跳过」两个按钮（`book_import_service.py:328-430`，`BookImport.tsx:1062-1118`）。
9. **唯一的「重试」是步骤级重试**，不是断点续传：`retry_failed_steps_stream`（`book_import_service.py:446-601`）校验请求的步骤名必须在 `failed_steps` 白名单里，然后**只重跑失败步骤**，成功步骤不重跑。但它依赖 `task.failed_steps`（内存）与 `task.imported_project_id`（内存）——**进程重启即彻底丢失，无法续传**。
10. **职业体系是这套流水线里最「非通用」的一环**：3 主 + 2 副的职业树、`max_stage`、`stages` JSON、`attribute_bonuses`，并在角色生成时把已生成的职业名**当作白名单塞进 prompt**，要求每个角色返回 `career_assignment`（`book_import_service.py:1879-1916`）。gaea 现有 `types` 里**没有 career 概念**，这一块要么新增数据模型，要么降级为「角色额外字段」。

---

## 1. 证据链：本文引用的全部文件

| 文件 | 规模 | 在流水线中的职责 |
|---|---|---|
| `backend/app/services/book_import_service.py` | 2285 行 / 100KB | **主服务**：任务生命周期、预览构建、反推编排、落库、三件套生成、步骤级重试 |
| `backend/app/services/txt_parser_service.py` | 171 行 | 编码识别、文本清洗、章节切分（强/弱/兜底三级） |
| `backend/app/api/book_import.py` | 287 行 | 6 个 HTTP 端点（含 2 个 SSE 流式端点），参数校验 |
| `backend/app/schemas/book_import.py` | 99 行 | 请求/响应 Pydantic 契约（含 `ProjectSuggestion` 等 8 个模型） |
| `backend/app/services/prompt_service.py` | 3158 行 | 提示词仓库：`BOOK_IMPORT_REVERSE_PROJECT_SUGGESTION:2480`、`BOOK_IMPORT_REVERSE_OUTLINES:2534`、`WORLD_BUILDING:82`、`CHARACTERS_BATCH_GENERATION:198`、`CAREER_SYSTEM_GENERATION:2302`；`get_template:2816`、`format_prompt:2610` |
| `backend/app/services/ai_service.py` | 762 行 | `call_with_json_retry:596`、`_add_json_hint:689` |
| `backend/app/utils/sse_response.py` | 463 行 | SSE 信封格式、`ProgressStage` 阶段常量、`create_sse_response:427` |
| `frontend/src/pages/BookImport.tsx` | 1159 行 | 4 步向导 UI、sessionStorage 缓存、SSE 消费、失败步骤重试面板 |
| `frontend/src/services/api.ts:476-522` | — | `bookImportApi` 6 个方法（含 2 个 SSE） |
| `frontend/src/utils/sseClient.ts` | 307 行 | `SSEPostClient`（fetch + ReadableStream 手写 SSE 解析） |
| `frontend/src/types/index.ts:1113-1205` | — | 前端 8 个拆书类型 |
| `backend/app/main.py:215` | — | `include_router(book_import.router, prefix="/api")` |

---

## 2. 完整链路（端到端）

```
[前端 Step0 上传]
  Dragger accept=".txt" → beforeUpload 返回 false（不自动上传）
  Select extract_mode ∈ {tail, full} / InputNumber tail_chapter_count ∈ [5,55] step5
  normalizedTail = max(5, ceil(n/5)*5)；normalizedTail>50 ⇒ effectiveExtractMode='full'
        │  POST /api/book-import/tasks  (multipart: file, extract_mode, tail_chapter_count)
        ▼
[API 校验] api/book_import.py:31-85
  .txt 后缀 / import_mode ∈ {append,overwrite} / extract_mode ∈ {tail,full}
  tail_chapter_count ≥ 5 且 %5==0；>50 ⇒ extract_mode='full'
  读全文，len>50MB ⇒ 413
  project_id 非空 ⇒ 400（本版固定新建项目）
        │  book_import_service.create_task  (task_id=uuid4)
        ▼
[阶段 A · _run_pipeline  book_import_service.py:603-650]
  5%   decode_bytes        → (text, encoding)
  10%  clean_text          → 归一化空白
  15%  split_chapters      → chapters_data[]
  18%  _build_preview
         ├ 截取末 N 章 _select_raw_chapters_for_preview
         ├ 逐章 _strip_chapter_prefix + _build_summary(120字) + 告警收集
         │   告警码：chapter_too_short(<300字) / chapter_too_long(>12000字) / duplicate_chapter_title / trimmed_for_extract_mode
         ├ 20→95% _generate_reverse_project_suggestion   ← AI #1
         │    fallback=_build_fallback_project_suggestion（关键词规则）
         │    ticker 协程每 2s 推进 +5%（35→85）模拟进度
         └ 95→99% _generate_reverse_outlines              ← AI #N（5章/批）
              每批 PromptService.format_prompt(BOOK_IMPORT_REVERSE_OUTLINES, ...)
              _normalize_reverse_outline_batch → 逐章 _normalize_single_reverse_outline
              批内缺项用 _build_fallback_outline_structure 补齐
  100% task.preview = preview ; status='completed'
        │
        │  GET /tasks/{id}  每 1500ms 轮询（BookImport.tsx:276-301）
        │  GET /tasks/{id}/preview  status=='completed' 后拉一次
        ▼
[前端 Step2 预览修正] BookImport.tsx:847-1015
  可改：title / genre / theme / description / narrative_perspective(3 选 1) / target_words
        + 逐章改 title / summary / content（Collapse 折叠）
  ⚠ 大纲不可改（页面上不渲染 outlines）
  ⚠ 前端把包内章节号重编为 1..N（updateChapter 只改字段不改号）
        │  POST /api/book-import/tasks/{id}/apply-stream  (SSE)
        ▼
[阶段 B · apply_import_stream  book_import_service.py:244-444]
  0-5%    _prepare_project（新建 Project + flush + 默认写作风格）
  5-10%   _import_outlines（title→id 映射，order_index 接续 append）
  10-20%  _import_chapters（chapter_number 接续，word_count=len(content)，outline_id 关联，status='draft'）
  20-40%  _generate_world_building_from_project        ← AI，失败可隔离
  40-65%  _generate_career_system_from_project         ← AI，失败可隔离
  65-92%  _generate_characters_and_organizations_from_project ← AI，失败可隔离（五阶段写入）
  92-98%  project.wizard_step=3 / wizard_status='completed' / status='writing' → db.commit()
        │
        │  SSE: progress×N → result → progress(100,'success') → done
        │  若 failed_steps 非空：额外 progress(status='step_failures', data={"failed_steps":[...]})
        ▼
[前端 Step3] 无失败 ⇒ message.success + 1s 后 navigate(/project/{id}/chapters)
             有失败 ⇒ 渲染失败步骤列表 + [智能重试全部失败步骤] / [跳过，直接进入项目]
                       │ POST /tasks/{id}/retry-stream (SSE)  steps=[step_name...]
                       ▼
              retry_failed_steps_stream：白名单校验 → 仅重跑失败步骤 → commit → 回传 still_failed
```

**关键时序事实**：`task.preview` 是**落库阶段的唯一数据来源**（apply 用 `payload` 而非 `task.preview` 校验），`task.imported_project_id` 是**重试阶段定位项目的唯一途径**。两者都在内存里（`self._tasks: dict[str, _BookImportTask]`，`book_import_service.py:89`）。

---

## 3. 算法层：逐段拆解

### 3.1 编码识别与清洗（`txt_parser_service.py:21-45`）

```python
encodings = ["utf-8", "utf-8-sig", "gb18030", "gbk", "big5"]   # :28
for enc in encodings:
    try: return content.decode(enc), enc
    except UnicodeDecodeError: continue
return content.decode("utf-8", errors="ignore"), "utf-8(ignore)"  # :37 兜底不抛错
```

清洗（`:39-45`，顺序敏感）：
1. `\r\n` → `\n`，`\r` → `\n`
2. 去 `\ufeff`
3. **`\u3000`（全角空格）→ 两个半角空格**（注意不是删除）
4. `[ \t]+\n` → `\n`（去行尾空白）
5. `\n{4,}` → `\n\n\n`（压缩 ≥4 连续空行为 3）
6. `.strip()`

> **gaea 对照**：`internal/app/novel_import_handler.go:190-202 decodeText()` 只做 `utf8.Valid → GB18030 → GBK`，**缺 `utf-8-sig`（BOM）与 Big5**，且用 `utf8.Valid(DECODED)` 二次校验——GB18030 解码任何字节序列几乎都不会失败，这个二次校验会静默放过误判。建议补 5 级链 + 明文标注最终编码（MuMu 把 encoding 回传前端并显示在进度文案里，见 `:616`）。

### 3.2 章节切分（`txt_parser_service.py:47-168`）

**强模式**（`:15-19`，`re.match` 锚定行首）：

```python
r"^第[一二三四五六七八九十百千万零〇两\d]+[章节回卷集部篇].*$"
r"^chapter\s*\d+.*$"      # IGNORECASE
r"^chap\.\s*\d+.*$"       # IGNORECASE
```

**弱模式**（`:119-133`，四条件与）：
- `len(line) <= 25`
- 不含任何 `，。！？；：,.!?;:`
- 前一行是空行（或为第 0 行）
- 后一行是空行（或为末行）

**兜底窗口切分**（`:135-168`）：
```
min_window=3000, max_window=5000
boundary_punctuations = "。！？!?\n"
while start < n:
    ideal_end = min(start+max_window, n)
    if ideal_end >= n: end = n
    else:
        search_from = min(start+min_window, n)
        segment = text[search_from:ideal_end]
        offset = max(segment.rfind(p) for p in boundary_punctuations)
        end = search_from+offset+1 if offset>=0 else ideal_end
    emit(title=f"第{no}章", content=text[start:end].strip())
    start = end
```
> 语义：**优先在 3000–5000 窗口内找最靠后的句子边界**；找不到才硬切在 5000。标题统一伪造成 `第N章`。

**组装规则**（`:75-114`）：
- 首个标题前的正文 **≥200 字** 才成为「前言」章（否则丢弃）
- `title = lines[start].strip()[:200]`，空则回退 `第{no}章`
- 空正文且不是最后一章时，把下一行拿过来当正文（防丢章）
- 最后 `[c for c in chapters if c["title"] or c["content"]]` 过滤纯噪音

> **gaea 对照**：`novel_import_handler.go:131-137 isChapterHeading()` 已覆盖 `第X章/回/卷/节/篇/部/集` + `chapter N` + `序章/楔子/引子/前言/序言/尾声/后记/番外/外传/终章/大结局` + markdown `#` 标题（比 MuMu 更全）；但**完全没有弱模式与固定窗口兜底**——识别不到就退化成单章「全文」（`:183-185`）。这是 gaea 导入体验最直接的短板。

### 3.3 提取范围裁剪（两处，语义不同）

| 位置 | 函数 | 作用对象 | 行为 |
|---|---|---|---|
| 预览阶段 | `_select_raw_chapters_for_preview` `:894-911` | `chapters_data`（原始 dict） | `N>50 或 full` ⇒ 全量；否则取**末尾 N 章** |
| 落库阶段 | `_select_chapters_for_import` `:828-892` | `BookImportChapter` + `BookImportOutline` | 同上裁剪，然后**把章节号重编为 1..N**，大纲同样截取后重编号，并**补齐大纲到与章节等长** |

落库阶段还有两个「对齐」动作值得注意：
- 章节 `outline_title` 缺省取 `item.title`（`:857`）
- 大纲不足时用 `_build_fallback_outline_structure(chapter)` 生成占位结构（`:878-887`）
- 最后**反向回写**：`normalized_chapters[idx].outline_title = normalized_outlines[idx].title`（`:889-890`）——保证 `_import_chapters` 里的 `outline_id_map.get(outline_title)` 一定能命中

### 3.4 摘要生成（`_build_summary` `:2236-2242`）

`re.sub(r"\s+"," ",content).strip()`，≤120 字原样返回，否则 `[:120]+"…"`。**纯规则，零 AI 成本**——这是章节摘要的廉价占位，用于预览页与大纲 fallback。

### 3.5 世界设定规则推断（`_derive_world_settings` `:918-1003`）

四个维度各一个关键词表，**首次命中即返回**（不是打分）：

| 维度 | 关键词（节选） | 缺省值 |
|---|---|---|
| `time_period` `:946-960` | 民国/军阀/北洋/租界 → 近代民国时期；星际/宇宙/机甲/赛博/未来/人工智能 → 未来科技时代；古代/王朝/皇帝/后宫/朝堂/将军/宗门/修仙/江湖/武林 → 古代架空时代；校园/大学/高中/公司/都市/地铁 → 现代都市 | 现代都市（可在世界设定页调整） |
| `location` `:962-976` | 星际/宇宙/舰队/空间站/机甲 → 多星系宇宙与舰队文明；宗门/仙门/秘境/灵脉 → 宗门林立的江湖/仙侠世界；王朝/都城/皇宫/边关/朝堂；校园/大学/高中；都市/城市/街区/公司/医院 | 以人物活动区域为核心的现实场景 |
| `atmosphere` `:978-992` | 悬疑/谜/诡/凶案/惊悚/追查 → 紧张悬疑、危机渐进；热血/战斗/对决/复仇/战争 → 高压对抗、节奏强烈；治愈/日常/温馨/轻松/搞笑；权谋/宫斗/朝堂/家族斗争 | 人物驱动、冲突递进 |
| `rules` `:994-1003` | 修仙/玄幻/灵气/境界/宗门/飞升；星际/机甲/赛博/人工智能/基因；江湖/门派/武林/侠客；王朝/皇权/朝堂/礼法 | 以现实逻辑为基础… |

**采样文本构造**（`:925-935`）：`title + theme + genre + description` 拼接，再追加**前 3 章各 1200 字**，用 `\n` 连接。

推断顺序：**文本关键词优先，genre 关键词兜底**（例如 `time_period` 先看 text 四条，再看 `genre` 是否含 科幻/星际 或 仙侠/玄幻/…）。

> 价值判断：这套规则**只是为了让新建项目不出现空字段**（注释原话「确保新建项目有可用初始值」），随后会被 AI 世界观生成覆盖（`_generate_world_building_from_project:1727-1739`）。gaea 若要抄，抄的是**「先落一个非空基线，再让 AI 覆写」的抗空值策略**，而不是关键词表本身。

### 3.6 主题/类型/视角/目标字数推断（fallback 路径）

- `_detect_theme_from_text` `:1493-1504`：复仇/报仇/雪恨 → 复仇与救赎；成长/蜕变/逆袭 → 成长与逆袭；真相/谜团/秘密/调查 → 真相与抉择；权谋/争权/朝堂/家族 → 权力与人性；爱情/喜欢/恋爱/婚约 → 爱情与选择；**缺省「命运与选择」**
- `_detect_genre_from_text` `:1506-1519`：修仙/宗门/灵气/飞升/仙门 → 仙侠；玄幻/异界/魔法/斗气；星际/机甲/赛博/人工智能/宇宙 → 科幻；悬疑/凶案/推理/谜案/诡；总裁/职场/都市/豪门；恋爱/言情/心动/告白；**缺省「通用」**
- `_detect_narrative_perspective` `:1521-1528`：**取前 6000 字**，`len(re.findall(r"[我咱俺]\S{0,2}", snippet))` 与 `[他她它]\S{0,2}` 计数，**第一人称命中 ≥20 且 > 第三人称×1.2** ⇒ 第一人称，否则第三人称。注意正则 `\S{0,2}` 是**贪婪吞字**的近似计数，不是严格代词统计。
- `_normalize_target_words` `:1578-1588`：`int()` 失败回退 fallback；`<1000` 回退 fallback；`>3_000_000` 截到 300 万
- `_normalize_narrative_perspective` `:1553-1576`：先精确匹配三个中文枚举，再做小写化 + `-/_` 归一后匹配 11 个英文别名（`first_person`/`third_person`/`omniscient`/`god_view`…），再做中文子串匹配（`第一人称`/`第一视角`/`主角视角`/`我视角`…），最后回退

> **这是典型的「AI 输出容错层」**：枚举收窄 + 别名归一 + 边界夹取 + fallback。gaea 的 `util.ExtractJSON` 只解决 JSON 抽取，**没有字段级归一化层**——这是可以直接抄的收益点。

### 3.7 反推大纲的结构契约与归一化（最核心）

**AI 输出契约**（Prompt `prompt_service.py:2567-2592`）：
```json
[{"chapter_number":1,"title":"章节标题",
  "summary":"章节概要（200-600字）","scenes":["场景1","场景2"],
  "characters":[{"name":"角色名1","type":"character"},
                {"name":"组织/势力名1","type":"organization"}],
  "key_points":["要点1","要点2"],"emotion":"本章情感基调","goal":"本章叙事目标"}]
```
字段约束（prompt 内硬约束）：`chapter_number` 必须与输入一致；`title` 必须与输入一致；`scenes` 2-6 条；`characters` 可为空且 `type` 仅 `character|organization`；`key_points` 2-6 条；`emotion`/`goal` 各一句话。

**代码侧归一化**（`_normalize_single_reverse_outline` `:1397-1444`）——**逐字段 fallback，绝不信任 AI**：

| 字段 | 归一化规则 |
|---|---|
| `summary` | `raw.summary or raw.content or fallback.summary` → `.strip()`；仍空取 `fallback.summary`；最终 `[:2000]` |
| `scenes` | 非 list ⇒ `[]`；逐项 `str().strip()` 去空；`[:6]`；空则用 `fallback.scenes` |
| `characters` | 逐项必须是 dict、必须有 `name`；`type=='organization'` ⇒ `"organization"`，**其余一律 `"character"`**；`name[:80]`；空则用 `fallback.characters` |
| `key_points` | 同 scenes，`[:8]` |
| `emotion` | `raw.emotion or fallback.emotion or "剧情递进"`，`[:200]` |
| `goal` | `raw.goal or fallback.goal or "推进主线冲突"`，`[:300]` |
| `title` / `chapter_number` | **强制用输入的章节值**，AI 返回值被丢弃 |

**批量归一化**（`_normalize_reverse_outline_batch` `:1375-1395`）：按**输入章节顺序**逐位取 `ai_items[idx]`（不是按 AI 返回的 `chapter_number` 对齐），非 dict 的槽位直接以 fallback 填充——**位置对齐 + 缺位兜底**，因此 AI 少返/乱序都不会错章。

**全局兜底**（`:1334-1341`）：
```python
if len(all_structures) != len(chapters):
    logger.warning("反向大纲数量与章节数量不一致，回退校正")
    all_structures = [self._build_fallback_outline_structure(c) for c in chapters]
```
> 注意：批次内已各自补齐，所以这条只在**极端情况**（例如把 `all_structures` 中途追加出错）触发。**但它是「宁可全丢 AI 结果也不用错位结构」的取舍**——值得保留为 gaea 的断言式防线。

**规则兜底结构**（`_build_fallback_outline_structure` `:1446-1466`）：
```json
{"chapter_number":N,"title":"<章标题>","summary":"<摘要或『本章围绕主要人物与核心冲突推进剧情。』>",
 "scenes":["主角在当前处境中做出关键选择","冲突升级并形成新的悬念"],
 "characters":[],"key_points":["推进主线冲突","呈现角色动机与关系变化"],
 "emotion":"紧张递进","goal":"承接前章并推动后续剧情发展"}
```

### 3.8 AI 调用与重试（`ai_service.py:596-690`）

```
for attempt in 1..max_retries(默认3):
    prompt_i = prompt                        if attempt==1
             else _add_json_hint(prompt, last_response, attempt)   # :689-690
    result = generate_text(prompt_i, handle_tool_calls=True)
    try:
        data = parse_json(content)
        if expected_type=='object' and not dict: raise ValueError("期望对象")
        if expected_type=='array'  and not list: raise ValueError("期望数组")
        return data
    except:
        last_response = content
        if attempt == max_retries: raise ValueError(f"JSON 解析失败: {e}")
```

`_add_json_hint` 原文（`:689-690`）：
```python
return f"{prompt}\n\n⚠️ 第{attempt}次重试，请返回纯JSON，不要markdown包裹。上次错误: {failed[:200]}..."
```
> **两段可抄的设计**：① 期望类型显式传入（`expected_type="object"|"array"`），避免「AI 返回对象但代码按数组解包」的静默错；② 重试时把上次的**失败原文截断 200 字**注入 prompt，让模型自我纠偏，而不是无脑重放同一 prompt。

### 3.9 角色/组织落库（五阶段，`_generate_characters_and_organizations_from_project` `:1848-2234`）

先做**全量预加载去重集合**（`:1932-1965`）：

| 集合 | 来源 | 用途 |
|---|---|---|
| `existing_names: set[str]` | `select(Character).where(project_id)` | 名字去重（同名跳过） |
| `character_name_to_obj: dict[name, Character]` | 同上 | 关系目标解析 |
| `organization_name_to_obj: dict[name, Organization]` | `Organization JOIN Character` | 组织名解析（**组织以 Character 行承载，`is_organization=True`**） |
| `member_pairs: set[(org_id, char_id)]` | `OrganizationMember JOIN Organization` | 成员关系去重 |
| `relationship_pairs: set[(from_id, to_id)]` | `CharacterRelationship` | 有向关系去重（**只查 (from,to)，反向 (to,from) 视为另一条**） |
| `relationship_type_map: dict[name, id]` | `select(RelationshipType)` 全表 | 关系类型名 → 字典表 id（**未命中则存 NULL**，`:2144`） |

五阶段写入顺序（**必须按此序，后阶段依赖前阶段的 id 与 name→obj 映射**）：

1. **实体行**（`:1970-2018`）：`Character(project_id, name[:100], age/gender(组织置 None), is_organization, role_type[:50], personality, background, appearance, organization_type/organization_purpose(仅组织), organization_members(JSON), traits(JSON))` → `flush()` → 若是组织再插 `Organization(character_id, project_id, power_level=clamp(0..100), member_count=0, location, motto, color)` → `flush()`。同时更新 `existing_names` / `character_name_to_obj` / `organization_name_to_obj`。
2. **职业关联**（`:2020-2102`）：兼容 `career_assignment`（批量格式）与 `career_info`（单角色格式：`main_career_name`/`main_career_stage`/`sub_careers[].career_name`）。主职业名必须命中 `main_career_map`；`main_stage` 夹到 `[1, main_career.max_stage]`；写 `CharacterCareer(career_type='main'|'sub', current_stage, stage_progress=0)`，并回填冗余字段 `character.main_career_id/main_career_stage/sub_careers(JSON)`。**副职业最多取 2 个**（`sub_list[:2]`）。
3. **角色关系**（`:2104-2152`）：优先 `relationships_array`，回退 `relationships`（旧版兼容）。跳过组织、跳过自身、跳过反向重复对。`intimacy_level=clamp(-100..100)`，`status` 默认 `active`，`source="ai"`。
4. **组织成员（角色侧）**（`:2154-2192`）：`organization_memberships[].organization_name` 解析组织，写 `OrganizationMember(position 默认『成员』, rank=clamp(0..10), loyalty=clamp(0..100), status='active', source='ai')`，并 `org.member_count += 1`。
5. **组织成员（组织侧回填）**（`:2194-2231`）：`organization_members` 里的**成员名字**解析成角色，补 `OrganizationMember(position='成员', rank=0, loyalty=50, status='active', source='ai')`，同样 `member_count += 1`。

> **要点**：`organization_members` 字段在阶段 1 被**原样 JSON 存进 Character 行**，又在阶段 5 被当作**成员名字列表**解析——同一字段两用（存储 + 反查）。gaea 的 `types.Organization.Members []string` 存的直接是角色 ID，语义更干净，移植时应改写为「名字 → ID 两趟解析」。

### 3.10 职业体系生成（`_generate_career_system_from_project` `:1749-1846`）

- Prompt：`CAREER_SYSTEM_GENERATION`（`prompt_service.py:2302`），入参 `title/genre/theme/description/time_period/location/atmosphere/rules`
- **幂等清理**（`:1798-1803`）：先 `delete(CharacterCareer).where(career_id.in_(该项目的 career_ids))` 再 `delete(Career).where(project_id)`——**先删关联表再删主表**（外键顺序），然后重建
- `_to_career_model` 归一化（`:1807-1831`）：`stages` 非 list ⇒ `[]`；`max_stage` 非法或 ≤0 ⇒ `len(stages) or (10 if main else 6)`；`attribute_bonuses` 序列化为 JSON；名字缺省 `未命名{主|副}职业{i+1}`，`[:100]`
- 返回 `created` 计数（主 + 副）

### 3.11 世界观生成（`_generate_world_building_from_project` `:1680-1747`）

- Prompt：`WORLD_BUILDING`（`prompt_service.py:82`），四字段 `time_period/location/atmosphere/rules`
- **逐字段非空才覆盖**（`:1727-1739`）：四个 `if xxx:` 独立判断，`updated` 是「有几个字段被更新」的计数（**不是 1/0 布尔**）
- `raise_on_error=True` 仅流式路径传（`:340`）与重试路径传（`:504`）；非流式 `apply_import` 路径不传 ⇒ 内部吞异常返回 0

---

## 4. 数据模型与接口契约

### 4.1 后端 Pydantic（`schemas/book_import.py`）

```python
TaskStatus   = Literal["pending","running","completed","failed","cancelled"]      # :8
ImportMode   = Literal["append","overwrite"]                                       # :9
ExtractLevel = Literal["basic","standard","deep"]                                  # :10  ← 定义但全项目未使用（死类型）
WarningLevel = Literal["info","warning","error"]                                   # :11
BookImportExtractMode = Literal["tail","full"]                                     # :12

BookImportWarning{code, message, level='warning'}                                  # :15-19
ProjectSuggestion{title(1..200), description?, theme?, genre?,
                  narrative_perspective='第三人称', target_words=100000 (ge=1000)}  # :22-29
BookImportChapter{title(1..200), content='', summary?, chapter_number(ge=1), outline_title?}  # :32-38
BookImportOutline{title(1..200), content?, order_index(ge=1), structure?}          # :41-46
BookImportTaskCreateRequest{extract_mode='tail', tail_chapter_count=10 (5..9999)}  # :49-52
BookImportTaskCreateResponse{task_id, status}                                      # :55-59
BookImportTaskStatusResponse{task_id, status, progress(0..100), message?, error?, created_at, updated_at}  # :61-69
BookImportPreviewResponse{task_id, project_suggestion, chapters[], outlines[], warnings[]}  # :72-78
BookImportApplyRequest{project_suggestion, chapters[], outlines[]=[], import_mode='append'}  # :81-86
BookImportApplyResponse{success, project_id, statistics: dict[str,int], warnings[]=[]}  # :89-94
BookImportRetryRequest{steps: list[str] (min_length=1)}                            # :97-99
```

> **契约漏洞（值得 gaea 反着做）**：`statistics: dict[str,int]` 是无类型字典——前端只能靠 `?? 0` 猜键（`BookImport.tsx:438-439`）。gaea 应定成具名结构体。

### 4.2 五个 HTTP 端点 + 两个 SSE 端点

| 方法 | 路径 | 形态 | 语义 |
|---|---|---|---|
| POST | `/api/book-import/tasks` | multipart，response_model | 建任务，**立即返回 `{task_id, status:"pending"}`，解析在后台 task 里跑** |
| GET | `/api/book-import/tasks/{id}` | JSON | 轮询状态（前端 1500ms） |
| GET | `/api/book-import/tasks/{id}/preview` | JSON | 取预览；**非 `completed` 状态 ⇒ 400「任务尚未完成，无法获取预览」**；`preview` 为空 ⇒ 500 |
| DELETE | `/api/book-import/tasks/{id}` | JSON | 取消（设置 `cancelled=True` + 状态置 `cancelled`）；**已是终态 ⇒ 直接返回成功** |
| POST | `/api/book-import/tasks/{id}/apply` | JSON | 非流式落库（**前端从不调用**，见 4.4） |
| POST | `/api/book-import/tasks/{id}/apply-stream` | **SSE** | 流式落库 + 三件套生成 |
| POST | `/api/book-import/tasks/{id}/retry-stream` | **SSE** | 仅重跑失败步骤 |

**SSE 服务器实现范式**（`api/book_import.py:149-210`）——gaea 若无 SSE 可参考这个「Queue + 后台 task + 生成器」结构：
```python
progress_queue: asyncio.Queue[str | None] = asyncio.Queue()
async def _progress_callback(message, progress, status="processing"):
    await progress_queue.put(SSEResponse.format_sse({...}))
async def _run_import():
    try:
        result = await service.apply_import_stream(..., progress_callback=_progress_callback)
        await progress_queue.put(await SSEResponse.send_result({...}))
        await progress_queue.put(await SSEResponse.send_progress("导入完成！",100,"success"))
        await progress_queue.put(await SSEResponse.send_done())
    except HTTPException as exc: await progress_queue.put(await SSEResponse.send_error(exc.detail, exc.status_code))
    except Exception as exc:     await progress_queue.put(await SSEResponse.send_error(str(exc), 500))
    finally:                     await progress_queue.put(None)   # 终止哨兵
async def _streaming_generator():
    yield await SSEResponse.send_progress("开始导入拆书数据...",0,"processing")
    import_task = asyncio.create_task(_run_import())
    try:
        while True:
            msg = await progress_queue.get()
            if msg is None: break
            yield msg
    except GeneratorExit:
        import_task.cancel()          # ← 客户端断开即取消后台任务
```
注意 `yield await SSEResponse.send_progress(...)` 与 `await progress_queue.put(await SSEResponse.send_result(...))` 的**双重 await**：因为 `send_*` 全是 `async staticmethod`（`sse_response.py:262-339`），返回值才是字符串。

**SSE 信封格式**（`sse_response.py:233-259, 262-344`）：
```
data: {"type":"progress","message":"...","progress":42,"status":"processing"}\n\n
data: {"type":"chunk","content":"..."}\n\n
data: {"type":"result","data":{...}}\n\n
data: {"type":"error","error":"...","code":500}\n\n
data: {"type":"done"}\n\n
: heartbeat\n\n                                    # 心跳（注释行，客户端忽略）
```
响应头（`:452-462`）：`text/event-stream; charset=utf-8`、`Cache-Control: no-cache, no-transform`、`X-Accel-Buffering: no`、CORS 三件套；**故意不带 `Connection: keep-alive`**（HTTP/2 不兼容，代码注释已说明）。

**`status` 的取值集合**（前端 `sseClient.ts:7`）：`processing | success | error | warning`；**但拆书额外用了 `"step_failures"`**——一个**超出类型声明**的自定义 status，靠前端 `if (status === 'step_failures')` 特判（`BookImport.tsx:422, 496`）。这是 MuMu 的一处**类型谎言**（TS 联合类型没跟上）。

### 4.3 前端类型（`frontend/src/types/index.ts:1113-1205`）

与后端 Pydantic 一一对应，额外三个：
```ts
BookImportResult{success, project_id, statistics:{chapters,outlines,
                 generated_careers?, generated_entities?, generated_world_building?}, warnings[]}
BookImportStepFailure{step_name, step_label, error, retry_count?}
BookImportRetryResult{success, project_id, retry_results: Record<string,number>, still_failed: BookImportStepFailure[]}
```

### 4.4 前端状态机与缓存（`BookImport.tsx`）

**4 步向导**（`:188-194`）：
```ts
const stepItems = [{title:'上传文件'},{title:'解析中'},{title:'预览修改'},{title:'生成导入'}];
const currentStep = useMemo(() => {
  if (!taskId) return 0;
  if (taskStatus && ['pending','running'].includes(taskStatus.status)) return 1;
  if (applying || isApplyComplete) return 3;
  if (preview) return 2;
  return 1;
}, [taskId, taskStatus, preview, applying, isApplyComplete]);
```

**sessionStorage 缓存**（`:41-104`）——key `book_import_page_cache_v1`，字段含 `taskId/taskStatus/preview/applyProgress/applyMessage/applyError/isApplyComplete/extractMode/tailChapterCount/cachedAt`：

| 机制 | 实现 |
|---|---|
| 过期 | `cachedAt` 超 **6 小时** 直接清除（`:204`，注释：避免后端重启后继续用旧 taskId） |
| 写入 | **永远写 `preview: null`**（`:254`，注释：preview 含完整章节正文体积大，易触发配额） |
| 配额溢出降级 | 捕获 `QuotaExceededError` / `NS_ERROR_DOM_QUOTA_REACHED` → 写轻量版（preview=null）→ 再失败则 `removeItem`（`:75-95`） |
| 恢复 | 启动时读缓存 → `setPreview(cache.preview)`（必为 null）→ 靠 `useEffect` 用 `taskId+taskStatus` **重新拉 preview**（`:303-333`） |
| 清理 | `isApplyComplete` 或无任何数据时清除（`:230-247`） |
| 404 处理 | 三个入口（轮询/预览/刷新状态）都判 404 ⇒ 清缓存 + `setTaskId(null)` + warning「拆书任务已失效（可能因服务重启），请重新上传TXT」（`:286-296, 314-323, 372-382`） |

**autos 模式（SSE 消费）**（`:416-472`）：`onProgress` 里先判 `status === 'step_failures'`，是则 `JSON.parse(msg).failed_steps` 写 `failedSteps` 并 **return（不污染进度条）**；`onResult` 里记录 `importedProjectId.current`，置 `isApplyComplete`，然后 **`setTimeout(...,100)` 内用 `setFailedSteps(prev => ...)` 读取最新值**——用「读状态函数 + 延迟一帧」绕过 React 状态闭包陈旧问题（`:446-459`，注释明写「需要延迟一帧来等待 failedSteps 的更新」）。这是可抄的前端技巧，但**更干净的做法是把失败信息放进 `onResult` 的 payload**（服务端已经知道，`apply_import_stream` 里的 `result` 没带 failed_steps 是服务端的小疏漏）。

**过期校验**：`rangeLocked`（`:186`）在 `taskId||taskStatus||preview||creatingTask||applying||retrying` 任一为真时锁定解析范围选择器，并显示 Alert「当前任务的解析范围已锁定」（`:721-728`）。**这是正确的产品决策**：解析范围只在建任务时生效，改范围必须「重新开始」。

---

## 5. Prompt 原文摘录（逐字 + 行号）

> 以下四段为**逐字原文**（仅去掉 Python 的 `"""` 与 `{{`/`}}` 转义，`{xxx}` 为 `format_prompt` 的占位符）。审校可逐字比对。

### 5.1 `BOOK_IMPORT_REVERSE_PROJECT_SUGGESTION` — 反推项目立项信息（`prompt_service.py:2480-2531`）

```
<system>
你是资深网文策划编辑，擅长从小说正文中反向提炼项目立项信息。
</system>

<task>
【任务】
基于提供的前3章内容，提炼该小说的核心立项信息，用于创建新项目。

【目标】
在不偏离原文的前提下，输出可直接用于项目初始化的结构化信息。
</task>

<input priority="P0">
【输入信息】
书名：{title}
前3章内容：
{sampled_text}
</input>

<output priority="P0">
【输出格式】
仅输出一个纯JSON对象（不要markdown、不要代码块、不要解释）：

{
  "description": "小说简介",
  "theme": "核心主题",
  "genre": "小说类型",
  "narrative_perspective": "第一人称/第三人称/全知视角",
  "target_words": 100000
}

【字段要求】
1) description：120-260字，聚焦主角、核心冲突、主线目标与故事张力。
2) theme：120-260字，提炼作品想表达的核心命题。
3) genre：2-12字，如都市、玄幻、悬疑、科幻、言情等。
4) narrative_perspective：只能是“第一人称”或“第三人称”或“全知视角”。
5) target_words：整数。按网文体量合理预估；无法判断时返回100000。
</output>

<constraints>
【必须遵守】
✅ 严格基于已给正文内容，不凭空添加关键设定
✅ 保持信息自洽，避免互相矛盾
✅ 输出必须是可解析JSON对象
✅ 小说的genre可以由多个类型组成

【禁止事项】
❌ 输出JSON以外的任何文字
❌ 使用markdown标记或代码块包裹
❌ narrative_perspective输出枚举值之外的内容
❌ target_words输出非整数
</constraints>
```

**调用点**（`book_import_service.py:1158-1186`）——输入构造：
```python
sampled_chapters = chapters[:3]                                    # :1158 只取前 3 章
sampled_text = "\n\n".join(
    f"【第{idx + 1}章 {chapter.title}】\n{(chapter.content or '')[:2000]}"
    for idx, chapter in enumerate(sampled_chapters)
).strip()                                                          # :1159-1162 每章截 2000 字
template = await PromptService.get_template("BOOK_IMPORT_REVERSE_PROJECT_SUGGESTION", user_id, db)  # :1181
prompt = PromptService.format_prompt(template, title=suggestion.title or "拆书导入项目",
                                     sampled_text=sampled_text)    # :1182-1186
project_data = await ai_service.call_with_json_retry(prompt=prompt, max_retries=3,
                                                     expected_type="object")  # :1219-1223
```

**结果合并策略**（`:1231-1244`）——**AI 优先、空则 fallback、`title` 永不用 AI**：
```python
result = ProjectSuggestion(
    title=suggestion.title,                                            # 文件名 stem，不被 AI 覆盖
    description=(project_data.get("description") or fallback.description or "").strip(),
    theme=(project_data.get("theme") or fallback.theme or "").strip() or fallback.theme,
    genre=(project_data.get("genre") or fallback.genre or "").strip() or fallback.genre,
    narrative_perspective=self._extract_narrative_perspective(project_data, fallback.narrative_perspective),
    target_words=self._normalize_target_words(project_data.get("target_words"), fallback.target_words),
)
```

### 5.2 `BOOK_IMPORT_REVERSE_OUTLINES` — 反推章节大纲（`prompt_service.py:2534-2607`）

```
<system>
你是资深网文总编与剧情策划，擅长基于已完成章节反向提炼标准化章节大纲。
</system>

<task>
【任务】
基于给定的章节正文（每批最多5章），为每章反向生成对应大纲结构。

【核心目标】
输出结构必须与系统现有大纲生成结构严格一致（与 OUTLINE_CREATE 字段一致），用于直接入库。
</task>

<project priority="P0">
【项目信息】
书名：{title}
类型：{genre}
主题：{theme}
叙事视角：{narrative_perspective}
</project>

<input priority="P0">
【批次范围】
第{start_chapter}章 - 第{end_chapter}章（共{expected_count}章）

【章节内容】
{chapters_text}
</input>

<output priority="P0">
【输出格式】
仅输出纯JSON数组（不要markdown、不要代码块、不要解释）。
数组长度必须严格等于 {expected_count}。

每个对象字段必须严格为：
[
  {
    "chapter_number": 1,
    "title": "章节标题",
    "summary": "章节概要（200-600字）：主要情节、角色互动、关键事件、冲突与转折",
    "scenes": ["场景1描述", "场景2描述"],
    "characters": [
      {"name": "角色名1", "type": "character"},
      {"name": "组织/势力名1", "type": "organization"}
    ],
    "key_points": ["情节要点1", "情节要点2"],
    "emotion": "本章情感基调",
    "goal": "本章叙事目标"
  }
]

【字段约束】
- chapter_number：必须与输入章节号一致
- title：必须与输入章节标题一致
- summary：根据本章正文反向提炼，不得臆造未出现关键事件
- scenes：2-6条
- characters：可为空；type 仅允许 character 或 organization
- key_points：2-6条
- emotion：一句话
- goal：一句话
</output>

<constraints>
【必须遵守】
✅ 严格一章对应一个对象，数量与顺序完全一致
✅ 字段名、字段层级、字段类型严格一致
✅ 仅基于输入正文提炼，不擅自扩展设定
✅ 输出必须可被JSON直接解析

【禁止事项】
❌ 输出JSON之外任何文本
❌ 缺失字段或新增字段
❌ chapter_number/title 与输入不一致
❌ 使用 markdown 或代码块
</constraints>
```

**批次输入构造**（`_build_reverse_outline_chapters_text` `:1363-1373`）：
```
【第{N}章 {title}】
章节摘要：{summary or '无'}
正文节选：
{content[:2200] or '无'}
```
（章与章之间 `\n\n` 连接。**注意：既给摘要又给 2200 字正文节选**——摘要来自 `_build_summary` 的规则截断，是廉价先验。）

**批次循环**（`:1291-1332`）：
```python
batch_size = 5                                                       # :1291
total_batches = ceil(len(chapters)/batch_size)                       # :1292
for batch_idx, start in enumerate(range(0, len(chapters), batch_size), start=1):
    batch = chapters[start:start+batch_size]
    start_chapter, end_chapter = batch[0].chapter_number, batch[-1].chapter_number
    prompt = format_prompt(template, title=..., genre=..., theme=..., narrative_perspective=...,
                           start_chapter=..., end_chapter=..., expected_count=len(batch),
                           chapters_text=self._build_reverse_outline_chapters_text(batch))
    ai_data = await ai_service.call_with_json_retry(prompt=prompt, max_retries=3, expected_type="array")
    all_structures.extend(self._normalize_reverse_outline_batch(ai_data, batch))
```

### 5.3 `WORLD_BUILDING` — 世界观（`prompt_service.py:82-195`，被拆书流水线复用）

入参 4 个：`title / genre / theme / description`（`book_import_service.py:1704-1710`）。
输出四字段 JSON，**每字段 300-500 字**：`time_period`（时间背景与社会状态）、`location`（空间环境与地理特征）、`atmosphere`（感官体验与情感基调）、`rules`（世界规则与社会结构）。

`<guidelines>` 段（`:105-134`）按题材给「时间尺度」指导，核心是**反过度设计**：
- 现代都市/言情/青春：时间=当代（2020年代）或近未来（2030-2050）；**避免大崩解/纪元/末日等宏大概念**
- 历史/古代：明确朝代或虚构古代
- 玄幻/仙侠/修真：修炼文明的特定时期
- 科幻：未来明确时期（如 2150 年、星际时代初期）
- 奇幻/魔法：魔法文明的特定阶段
- **设定尺度控制**：现代都市→聚焦某城市/行业/阶层；校园青春→学校环境；职场言情→公司文化；**史诗题材才需要宏大世界观**

`<constraints>`（`:182-195`）：必须「简介契合 / 类型适配 / 主题贴合 / 具象化 / 逻辑自洽」；禁止「与类型不匹配 / 为小规模题材用宏大世界观 / 模板化空泛表达 / markdown」。

### 5.4 `CHARACTERS_BATCH_GENERATION` — 批量角色/组织（`prompt_service.py:198-321`）

入参 8 个：`count / time_period / location / atmosphere / rules / theme / genre / requirements`（`book_import_service.py:1906-1916`）。

**数量硬约束**（`:206-207`）：
```
【数量要求 - 严格遵守】
数组中必须精确包含{count}个对象，不多不少。
```
**实体类型分配**（`:209-213`）：至少 1 主角（protagonist）、多个配角（supporting）、可含反派（antagonist）、**可含 1-2 个高影响力组织（`power_level: 70-95`）**。

**角色对象字段**：`name / age / gender / is_organization=false / role_type / personality(100-200字) / background(100-200字) / appearance(50-100字) / traits[] / relationships_array[{target_character_name, relationship_type, intimacy_level, description}] / organization_memberships[{organization_name, position, rank, loyalty}]`

**组织对象字段**：`name / is_organization=true / role_type / personality(100-200字) / background(100-200字) / appearance(50-100字) / organization_type / organization_purpose / organization_members[成员名] / power_level / location / motto / color / traits[]`

**关系类型参考**（`:283-287`）：家族（父亲/母亲/兄弟/姐妹/子女/配偶/恋人）、社交（师父/徒弟/朋友/同学/同事/邻居/知己）、职业（上司/下属/合作伙伴）、敌对（敌人/仇人/竞争对手/宿敌）
**数值范围**（`:289-293`）：`intimacy_level ∈ [-100,100]`、`loyalty ∈ [0,100]`、`rank ∈ [0,10]`、`power_level ∈ [70,95]`

**拆书场景的 `requirements` 拼装**（`book_import_service.py:1888-1904`）：
```python
requirements = ("请生成能够支撑前期剧情推进的关键角色与组织，"
                "角色和组织都要与世界观、职业体系一致。"
                "如果包含组织，数量不超过2个。"
                "请尽量为非组织角色补充 organization_memberships。")

if main_careers or sub_careers:
    careers_context = "\n\n【职业分配要求】\n"
    careers_context += "请为每个非组织角色返回 career_assignment 字段："
    careers_context += '{"main_career":"主职业名称","main_stage":2,"sub_careers":[{"career":"副职业名称","stage":1}]}'
    careers_context += "\n职业名称必须从以下列表中选择：\n"
    if main_careers: careers_context += "- 可用主职业：" + "、".join([c.name for c in main_careers]) + "\n"
    if sub_careers:  careers_context += "- 可用副职业：" + "、".join([c.name for c in sub_careers]) + "\n"
    requirements += careers_context
```
> **这是「上下文注入式枚举约束」的范式**：把刚生成的职业名以自然语言白名单形式塞进 prompt，让模型在**闭集**里选，而不是自由发挥后靠字典查表失败。落库端仍有 `main_name in main_career_map` 兜底（`:2051`）。

### 5.5 `CAREER_SYSTEM_GENERATION`（`prompt_service.py:2302` 起）

入参 8 个（`book_import_service.py:1770-1781`）：`title / genre / theme / description / time_period / location / atmosphere / rules`。
输出 `{"main_careers":[...], "sub_careers":[...]}`，每个 career 含 `name / description / category / stages / max_stage / requirements / special_abilities / worldview_rules / attribute_bonuses`。

### 5.6 提示词模板的解析与降级（`prompt_service.py:2816-2860, 2610-2624`）

```python
@classmethod
async def get_template(cls, template_key, user_id, db) -> str:
    result = await db.execute(select(PromptTemplate).where(
        PromptTemplate.user_id == user_id,
        PromptTemplate.template_key == template_key,
        PromptTemplate.is_active == True))                       # :2838-2844
    custom_template = result.scalar_one_or_none()
    if custom_template: return custom_template.template_content  # :2847-2849 用户自定义优先
    template_content = getattr(cls, template_key, None)          # :2855 降级到类属性（系统默认）
    if template_content is None: logger.warning("⚠️ 未找到系统默认模板: " + template_key)
    return template_content

@staticmethod
def format_prompt(template: str, **kwargs) -> str:                # :2610-2624
    try: return template.format(**kwargs)
    except KeyError as e: raise ValueError(f"缺少必需的参数: {e}")
```
**系统模板元信息**（`:2886-2900`）：两个拆书模板的注册项，含 `name/category:'拆书导入'/description/parameters` 四字段——**这是 gaea「提示词可在 UI 里改」的等价物**：模板键 → 元信息 → 参数列表。

> **gaea 对照**：`internal/prompt/prompt.go` 是 **RTCO JSON 模板**（`name/system/task/input_sections/output/constraints`），由 `BuildSystemPrompt` + `BuildUserPrompt` 拼装，**没有用户自定义覆盖层**、**没有参数声明**、**没有 `{占位符}` 替换**。MuMu 的「DB 自定义 > 类属性默认」双层 + `parameters` 元信息，是 gaea 提示词中心可加分的地方（但属于 T6 提示词域的范畴，本文只标注接口形状）。

---

## 6. 进度与失败语义（精确数值）

### 6.1 阶段 A（解析）进度表（`_run_pipeline` `:603-650`）

| 进度 | 状态 | 文案 |
|---|---|---|
| 5 | running | `正在识别编码并读取文本...` |
| 10 | running | `文本清洗完成（编码：{encoding}）` |
| 15 | running | `已识别 {n} 个章节，正在构建预览结构...` |
| 18 | running | `正在按解析配置筛选章节并构建预览...` |
| 18→20 | running | `已处理{末N章\|整本} {i}/{total} 个章节结构...`（`idx % max(1,total//5)==0` 或 `idx==total` 时才推送，`:1084`） |
| 20 | running | `正在调用AI反向生成项目信息（标题/简介/主题/类型）...` |
| 25 | running | `正在初始化AI服务...` |
| 30 | running | `正在准备AI提示词...` |
| 35 | running | `AI正在分析文本内容...` |
| 35→85 | running | ticker 每 2s `+5`，文案循环：`AI正在分析文本内容... → AI正在识别故事主题与类型... → AI正在推断叙事角度... → AI正在生成项目简介... → AI正在整理生成结果...`（`:1199-1205`） |
| 90 / 95 | running | `AI生成完成，正在整理项目信息...` / `项目信息生成完毕，准备预览...` |
| 95 | running | `正在反向生成章节大纲（分批5章）...` |
| 95→99 | running | `正在生成大纲批次 {i}/{total}（第{start}-{end}章）...`（公式 `95 + int(3*(i-1)/total)`，`:1306`） |
| 99 | running | `大纲反向生成完成，正在整理预览...` |
| 100 | completed | `解析完成，可预览并确认导入` |

**异常路径**：`asyncio.CancelledError` ⇒ `status='cancelled'`、`message='任务已取消'`、progress 保持（`:640-641`）；其他异常 ⇒ `status='failed'`、`message='解析失败'`、`error=str(exc)`（`:642-650`）。

### 6.2 阶段 B（落库）进度表（`apply_import_stream` `:244-444`）

| 进度 | 文案 | 失败隔离 |
|---|---|---|
| 2 / 5 | `正在创建项目...` / `项目创建完成` | 无（失败 ⇒ 整体 500 + rollback） |
| 6 / 10 | `正在导入大纲...` / `已导入 {n} 个大纲` | 无 |
| 12 / 20 | `正在导入 {n} 个章节...` / `已导入 {n} 个章节（{m}字）` | 无 |
| 22 / 40 | `🌍 正在生成世界观...` / `🌍 世界观生成完成` | **有**：失败 ⇒ `_StepFailure('world_building','世界观生成')` + warning 进度 + **继续** |
| 42 / 65 | `💼 正在生成职业体系...` / `💼 职业体系生成完成（{n}个）` | **有**：`_StepFailure('career_system','职业体系生成')` |
| 67 / 92 | `👥 正在生成角色与组织...` / `👥 角色/组织生成完成（{n}个）` | **有**：`_StepFailure('characters','角色与组织生成')` |
| 95 / 98 | `正在保存到数据库...` / `数据保存完成` | 无 |
| 98 | `⚠️ 导入完成，但有 {n} 个生成步骤失败，可点击重试`（status=`warning`） | — |
| 98 | **额外消息** `status="step_failures"`，`message=json.dumps({"failed_steps":[{step_name,step_label,error}]})` | :425-430 |
| 100 | `导入完成！`（status=`success`），随后 `done` | — |

**子步骤内的相对进度**（各生成函数用 `progress_range` 线性映射，`:1693-1696, 1761-1764, 1861-1864`）：
`p = start + int((end-start) * sub)`，`sub` 序列例如世界观 `0.1→0.2→0.3→0.8→1.0`（`:1699,1702,1712,1721,1741`）。

**重试阶段进度**（`retry_failed_steps_stream` `:486-488`）：
```python
step_start_pct = int(5 + (step_idx / total_steps) * 85)
step_end_pct   = int(5 + ((step_idx+1) / total_steps) * 85)
```
即把 5%–90% 按待重试步骤数均分。终态：`93 正在保存到数据库...` → `96 数据保存完成` → 若有 `still_failed` 再推 `98 step_failures` 消息。

### 6.3 步骤重试的白名单契约（`:464-472`）

```python
failed_step_names = {f.step_name for f in task.failed_steps}
invalid_steps = [s for s in steps_to_retry if s not in failed_step_names]
if invalid_steps:
    raise HTTPException(400, f"以下步骤不在失败列表中，无法重试: {', '.join(invalid_steps)}")
```
合法 `step_name` 仅三个字面量：`world_building` / `career_system` / `characters`。

**重试计数语义**（`:491-492`）：
```python
original_failure = next((f for f in task.failed_steps if f.step_name == step_name), None)
retry_count = (original_failure.retry_count if original_failure else 0) + 1
```
`retry_count` 只在**仍然失败**时写回 `_StepFailure`（`:514`）；成功则不保留计数。前端展示为 `已重试 {n} 次` 标签（`BookImport.tsx:1086-1088`）。**无次数上限**——用户可以无限点重试。

---

## 7. 缺陷与风险清单（源码级证据）

> 这一节是给 gaea 的「反着做」清单。每条都给出证据行号。

| # | 缺陷 | 证据 | 影响 | gaea 对策 |
|---|---|---|---|---|
| D1 | **任务态纯内存**：`self._tasks: dict[str,_BookImportTask]`，无持久化、无 TTL | `book_import_service.py:89, 122-125` | 进程重启 ⇒ 所有 task_id 失效；预览与失败步骤全丢；前端只能靠 404 检测并清缓存重来 | 落到 gaea 已有 `tasks` 表（Hephaestus.db）：`KindBookImport` + `Payload` 存 task 参数 + `Result` 存预览/步骤态 |
| D2 | **取消不生效于最贵阶段**：`_run_pipeline` 在预览构建完成后、AI 反推期间**不再检查取消标志** | `book_import_service.py:637-639`（`_check_cancelled` 只在 611/617/627/637 调用，都在 `_build_preview` 之前或紧接其后） | 用户点「取消」后两次 AI 调用（项目信息 + N/5 批大纲）仍会跑完并计费 | 每批 AI 调用前后各查一次 `ctx.Done()` |
| D3 | **非流式 `apply_import` 是死代码**：前端永远调 `apply-import-stream`；且它**不记录 `failed_steps`、不设置 `task.imported_project_id`、不置 wizard 状态**（其实是靠 `_run_post_import_wizard_generation` 内部设的） | `api/book_import.py:106-122`；`BookImport.tsx:416` 只调 stream；`book_import_service.py:149-239` vs `244-444` | 双份落库逻辑，行为不一致（角色数 `max(...,8)` vs `max(...,5)`：`:219` vs `:376`） | 只保留一条实现，SSE 只是「带回调的同一条流水线」 |
| D4 | **`get_preview` 对中间态返回 400**：「任务尚未完成，无法获取预览」 | `book_import_service.py:132-138` | 前端必须在 `completed` 后才拉；`completed` 与 `preview!=None` 之间那条 `if not task.preview: raise 500` 是理论不可能的防御 | 设计成「预览可增量返回」或「阶段 A 结束即完成」 |
| D5 | **`apply` 的章节范围以客户端 payload 为准**，服务端只做一次裁剪再信任 | `book_import_service.py:167-180, 268-281` | 恶意/异常前端可绕过 extract_mode 直接灌全书 | 服务端持有权威裁剪结果，客户端只能改内容不能改范围 |
| D6 | **`retry_stream` 缺 `import_mode`**：重试路径不知道原任务是 append 还是 overwrite | `retry_failed_steps_stream:446-455` 无 `import_mode` 参数 | 幂等清理依赖项目新建的前提（注释 :1798 自认「拆书导入走新建项目，但这里保持幂等」） | 重试必须携带原任务的完整模式快照 |
| D7 | **`status="step_failures"` 是超出类型声明的自定义值** | `SSEMessage.status` 联合类型（`sseClient.ts:7`）不含它；比对 `BookImport.tsx:422, 496` | 类型系统失效，后续重构易删掉该分支 | 用独立 SSE event 名（`event: step_failures`）或 `type` 字段 |
| D8 | **`result` 里不带 `failed_steps`**，前端只能靠**之前**收到的 `step_failures` 进度消息 + `setTimeout(100ms)` 读状态 | `book_import_service.py:414-430`（失败信息只走 progress）vs `:432-437`（result 无该字段）；`BookImport.tsx:446-459` | 竞态：若进度消息与 result 到达顺序异常，失败列表可能丢失 | `result` payload 内联 `failed_steps`，前端零竞态 |
| D9 | **`ExtractLevel = Literal["basic","standard","deep"]` 定义后全项目未使用** | `schemas/book_import.py:10`（grep `ExtractLevel` 仅此一处定义） | 死类型，误导后续维护 | — |
| D10 | **`statistics` 是无类型 dict** | `schemas/book_import.py:93`；前端 `result.statistics?.generated_careers ?? 0`（`BookImport.tsx:438`） | 键名变更无编译期保护 | 具名 struct |
| D11 | **职业/组织表清理顺序不能错**：`_clear_project_data` 必须 `OrganizationMember → Organization → CharacterCareer → Career → Character` | `book_import_service.py:713-727` | 顺序错则外键报错 | 用事务 + 明确顺序，或 ON DELETE CASCADE |
| D12 | **`api` 与 `service` 对 tail 归一化不一致**：api 用原始值判 `>50`（`:58`），service 是先向上取整到 5 的倍数再判（`:106-109`） | `api/book_import.py:53-59` vs `book_import_service.py:104-109` | 入参 51 ⇒ api 判 full，service 判 55 ⇒ full（结果相同）；入参 48 ⇒ 都 tail；**入参 47 ⇒ api tail(47<50)，service 归一化 50 ⇒ tail(50 不 >50)**，无差异但逻辑脆弱 | 一处归一化，入参即规范化输出 |
| D13 | **`asyncio.create_task` 无引用持有**：`create_task(self._run_pipeline(...))` 返回值被丢弃 | `book_import_service.py:125` | GC 可能提前回收任务（CPython 已有强引用保护，但语义不严谨） | Go 侧用 `go func()` 或显式任务句柄 |
| D14 | **组织以 `Character` 行承载**（`is_organization=True`），组织另存 `Organization.character_id` 外键 | `book_import_service.py:1980-2013`；`organization_name_to_obj` 靠 `Organization JOIN Character`（`:1938-1945`） | 同一实体两行，名字唯一性只在 Character 侧保证 | gaea 的 `types.Organization` 独立表更干净，移植时需改写 |

---

## 8. gaea 落地规格（分阶段、可验收）

### 8.0 现状盘点（先明确差异）

| 能力 | MuMu | gaea 现状 | 差距 |
|---|---|---|---|
| 文件格式 | 仅 `.txt`（API 硬校验） | **TXT / Markdown / EPUB**（`novel_import_handler.go:39`） | gaea 领先 ✅ |
| 编码 | 5 级链 + ignore 兜底 | 3 级（utf8/gb18030/gbk），缺 BOM/Big5 | 需补 |
| 分章 | 强 + 弱 + 固定窗口兜底 | 仅强正则 + markdown；失败退化成单章「全文」 | **需补弱模式 + 兜底窗口** |
| 提取范围 | tail/full + 5 的倍数 + >50 降级 | 无（整本导入） | 需新增 |
| 预览修订 | 项目信息 6 字段 + 逐章 title/summary/content | 无 | 需新增 |
| AI 反推项目信息 | ✅ | ❌ | 需新增 |
| AI 反推大纲 | ✅（5 章批 + 结构契约） | ❌（导入只写 `OutlineNode{Status:Done}`） | 需新增 |
| 落库后三件套 | 世界观 + 职业 + 角色/组织 | ❌ | 需新增（职业需新建模型，见 8.5） |
| 步骤级失败隔离 + 重试 | ✅（5 个落库步骤中后 3 个可隔离） | ❌ | 需新增 |
| 任务持久化 | ❌（内存） | ✅ `tasks` 表（Hephaestus.db SchemaV8+，重启续跑） | **gaea 领先** ✅ |

### 8.1 阶段 P0 · 解析引擎升级（纯规则，零 AI，可独立验收）

**目标**：把 MuMu 的三级分章算法与 5 级编码链移植进 gaea，替换现有「识别不到就单章」的退化。

**落点**：`internal/app/novel_import_handler.go`（改造 `parseTextChapters` / `decodeText`），或新建 `internal/bookimport/parse.go`。

**数据结构**（新建，建议放 `internal/bookimport`）：

```go
// ParsedChapter 解析出的原始章节（未编号、未裁剪）
type ParsedChapter struct {
    Title   string
    Content string
}

// ParseOptions 解析选项（对齐 MuMu 的 extract_mode / tail_chapter_count）
type ParseOptions struct {
    ExtractMode      string // "tail" | "full"
    TailChapterCount int    // 必须为 5 的倍数；>50 自动降级 full
}

// ParseReport 解析报告（承载 MuMu 的 warnings 机制）
type ParseReport struct {
    Encoding        string   `json:"encoding"`         // 最终采用的编码名
    TotalChapters   int      `json:"totalChapters"`    // 识别到的总章数
    SelectedChapters int     `json:"selectedChapters"` // 裁剪后章数
    SplitStrategy   string   `json:"splitStrategy"`    // strong | weak | window | single
    Warnings        []Warning `json:"warnings"`
}

type Warning struct {
    Code    string `json:"code"`    // chapter_too_short | chapter_too_long | duplicate_chapter_title | trimmed_for_extract_mode
    Message string `json:"message"`
    Level   string `json:"level"`   // info | warning | error
}
```

**算法规格**（逐条对齐 MuMu，行号见 §3）：

1. **编码链**（对齐 `txt_parser_service.py:28`）：`utf-8 → utf-8-sig → gb18030 → gbk → big5`，全失败则 `utf-8 + 忽略非法字节`（Go 用 `strings.ToValidUTF8(s, "")`），并在 `ParseReport.Encoding` 标注实际采用项。
2. **清洗**（对齐 `:41-45`，顺序严格）：
   - `\r\n`→`\n`，`\r`→`\n`；去 `\uFEFF`
   - `\u3000` → 两个半角空格
   - 行尾 `[ \t]+` 删除
   - `\n{4,}` → `\n\n\n`
   - 最终 `TrimSpace`
3. **强标题**（对齐 `:15-19`）：行首锚定 `^第[一二三四五六七八九十百千万零〇两0-9]+[章节回卷集部篇]` / `(?i)^chapter\s*\d+` / `(?i)^chap\.\s*\d+`。
   > **保留 gaea 现有更全的集合**（`序章/楔子/引子/前言/序言/尾声/后记/番外/外传/终章/大结局` + markdown `#`）——这是 gaea 的既有优势，不要为了对齐而回退。
4. **弱标题**（对齐 `:119-133`，四条件与）：`runeLen(line) <= 25` ∧ 不含 `，。！？；：,.!?;:` ∧ 前行为空（或 idx==0）∧ 后行为空（或 idx==len-1）。
5. **兜底窗口**（对齐 `:135-168`）：`minWindow=3000 maxWindow=5000`，边界字符 `。！？!?\n`，从 `start+minWindow` 到 `start+maxWindow` 的切片里取**最靠后的边界位置**；找不到就在 `start+maxWindow` 硬切。标题伪造成 `第N章`。
6. **组装**（对齐 `:75-114`）：首标题前正文 ≥200 字符（**rune 计数，非 byte**——MuMu 的 `len()` 是字节数，中文会偏大，gaea 应改为 `utf8.RuneCountInString`）⇒ 生成「前言」章；标题 `[:200]` 截断；空正文取下一行补；最后过滤 title 与 content 皆空的项。
7. **裁剪 + 重编号**（对齐 `_select_chapters_for_import:828-892`）：`tail` 取末尾 N 章（`N=clamp(5 的倍数)`, `N>50 ⇒ full`），随后**章节号重编为 1..N**，大纲同步重编与补齐。
8. **告警**（对齐 `:1065-1109`）：`content` rune 数 `<300` ⇒ `chapter_too_short`(warning)；`>12000` ⇒ `chapter_too_long`(info)；标题计数 `>1` ⇒ `duplicate_chapter_title`(warning)；发生裁剪 ⇒ `trimmed_for_extract_mode`(info)。

**验收命令**：
```
go test ./internal/bookimport/... ./internal/app/...
go build ./...
```
**必须覆盖的用例**（建议先放测试 fixture 到 `.tmp/` 下不入库）：
- UTF-8 BOM 文件、GB18030 文件、纯 ASCII 文件
- 强标题 3 章 / 弱标题（短行 + 前后空行）/ 无标题 8000 字（走窗口切分成 2 章）
- 首标题前 250 字前言 ⇒ 生成「前言」章；150 字 ⇒ 丢弃
- `tail=10` 且总章 30 ⇒ 取 21..30 并重编为 1..10
- `tail=55` ⇒ 自动 full

### 8.2 阶段 P1 · AI 反推与预览（含 Prompt 移植）

**落点**：新建 `internal/bookimport/` 包（解析 + 反推 + 落库编排），`internal/app/` 加绑定 `NovelImportReconstruct*`。

**Prompt 模板落地**：gaea 是 RTCO JSON 模板（`prompts/*.json`），需把 MuMu 的两段五段式 Prompt **等价改写**为 RTCO 结构：

| MuMu 段 | gaea 模板字段 |
|---|---|
| `<system>` | `system` |
| `<task>【任务】【目标】` | `task` |
| `<input priority="P0">` | `input_sections` 中最关键的键标 `"P0"` |
| `<output priority="P0">` | `output.{format, description}`（JSON 结构写进 `description`） |
| `<constraints>` ✅/❌ | `constraints.{must, forbidden}` |

新增两个模板文件：
- `prompts/book-import-project.json`（对应 `BOOK_IMPORT_REVERSE_PROJECT_SUGGESTION`）
  - `input_sections`: `title`(P0)、`sampled_text`(P0)
  - `output.format: "json"`，`output.description` 保留原文的 5 字段与字数要求
  - `constraints.must`: 严格基于已给正文、信息自洽、可解析 JSON、genre 可多类型
  - `constraints.forbidden`: 输出 JSON 以外文字、markdown 包裹、`narrative_perspective` 越枚举、`target_words` 非整数
- `prompts/book-import-outline.json`（对应 `BOOK_IMPORT_REVERSE_OUTLINES`）
  - `input_sections`: `project_title`(P0)、`genre`(P0)、`theme`(P0)、`narrative_perspective`(P0)、`batch_range`(P0)、`expected_count`(P0)、`chapters_text`(P0)
  - `output.format: "json"`，`description` 逐字保留 7 字段契约 + 字段约束（scenes 2-6、key_points 2-6、characters.type 仅两值）
  - `constraints.must`: 一章一对象、数量顺序完全一致、字段名/层级/类型严格一致、仅基于输入正文、可 JSON 解析
  - `constraints.forbidden`: 任何非 JSON 文本、缺失或新增字段、`chapter_number`/`title` 与输入不一致、markdown

> ⚠️ **RTCO 引擎的一个真实约束**：`BuildUserPrompt` 用 `for key, def := range t.Inputs` 遍历 **Go map**（`internal/prompt/prompt.go:194`）——**输出顺序随机**，而 `expected_count`、`batch_range`、`chapters_text` 这类字段对上下文位置敏感（长文本放最后更稳）。若要在 gaea 保持 MuMu「P0 输入按固定顺序拼装」的语义，需要**给 `InputDef` 增加 `order int` 字段并按序排序**（改 `prompt.go` 属 T6 提示词域，本文只提出接口需求）。

**反推编排规格**：

```
ReconstructStage1(ctx, chapters []ParsedChapter, aiClient) (ProjectSuggestion, error)
  采样：前 3 章，每章正文截 2000 rune（对齐 book_import_service.py:1158-1162）
  渲染模板 → 调 LLM（期望 object）
  失败 ⇒ 规则 fallback（_build_fallback_project_suggestion 等价物）
  合并：title 永不用 AI；其余字段 AI 优先、空取 fallback
  归一化：narrative_perspective（中文枚举 + 11 个英文别名）、target_words（<1000 回退、>3e6 夹取）

ReconstructOutlines(ctx, chapters, suggestion, aiClient, progress func(stage string, i, n int)) ([]OutlineNode, error)
  batchSize = 5
  for each batch:
      渲染模板(start_chapter, end_chapter, expected_count=len(batch), chapters_text)
      调 LLM（期望 array）
      位置对齐归一化：第 i 个槽位取 ai[i]（非 map 则 fallback）
      逐字段 fallback（summary/scenes[:6]/characters(type 二元)/key_points[:8]/emotion[:200]/goal[:300]）
      强制 title/chapter_number 用输入值
  if len(all) != len(chapters): 全部回退规则结构（断言式防线）
```

**Go 侧关键接口**（对齐 MuMu 的字段语义，但用 gaea 的具名类型）：

```go
// ProjectSuggestion 反推出的项目立项信息（对齐 schemas/book_import.py:22-29）
type ProjectSuggestion struct {
    Title                string `json:"title"`
    Description          string `json:"description,omitempty"`
    Theme                string `json:"theme,omitempty"`
    Genre                string `json:"genre,omitempty"`
    NarrativePerspective string `json:"narrative_perspective"` // 第一人称|第三人称|全知视角
    TargetWords          int    `json:"target_words"`
}

// ReconstructPreview 预览载荷（对齐 BookImportPreviewResponse）
type ReconstructPreview struct {
    TaskID            string            `json:"task_id"`
    ProjectSuggestion ProjectSuggestion `json:"project_suggestion"`
    Chapters          []PreviewChapter  `json:"chapters"`   // title/content/summary/chapter_number/outline_title
    Outlines          []PreviewOutline  `json:"outlines"`   // title/content/order_index/structure
    Warnings          []Warning         `json:"warnings"`
}
```

### 8.3 阶段 P2 · 落库与三件套生成（gaea 数据模型映射）

**gaea 落库映射表**（MuMu 表 → gaea 结构）：

| MuMu | gaea | 转换规则 |
|---|---|---|
| `Project` | `types.ProjectMeta` + `project.Create` | `title/genre/description` 直映；`target_words` 落 `ProjectMeta`（**需新增字段** `TargetWords int`）；`wizard_*` **不映射**（gaea 无向导概念） |
| `Project.world_*` 4 字段 | `types.WorldviewFile.Sections[]` | MuMu 是 4 个平铺字符串，gaea 是 6 维 section 列表。映射：`time_period → era(时代背景)`、`location → geography(地理风貌)`、`atmosphere → culture(文化习俗)` 或新增 section、`rules → rules(规则体系)`；**建议保留 gaea 6 维，把 AI 输出的 4 字段拼进对应 section 而不是硬套 4 段** |
| `Outline` | `types.OutlineNode` | `title → Title`、`structure.summary → Summary`、`structure.scenes[] → SceneIdeas`、`structure.characters[].name → Characters`（**需先解析成角色 ID**）、`structure.key_points[] → KeyPoints`、`structure.emotion → Emotion`、`order_index → OrderIndex`、`status = OutlineDone`（导入的章已有正文，故为 done） |
| `Chapter` | `pm.WriteChapter(n, content)` + `nnn-summary.json` | `chapter_number → n`；`content` 直写；`summary` 落 `types.ChapterSummary`（`Title`/`Summary`），其余字段留空待后续 `GenerateSummary` 补 |
| `Character` | `types.Character` | 字段名一一对应（`role_type/personality/background/appearance`）；`age` MuMu 是 `str`，gaea 是 `string` ✅；`gender` 同；**MuMu 的 `is_organization` 行不落 gaea 的 Character**，转 `types.Organization` |
| `Organization` | `types.Organization` | `power_level` MuMu 是 `int 0..100`，gaea 是 `string` ⇒ 转字符串或改类型；`members` MuMu 存名字、gaea 存 ID ⇒ **两趟解析**（先建全部角色，再把名字换 ID） |
| `CharacterRelationship` | `types.Relationship` | `character_from_id → FromID`、`character_to_id → ToID`、`relationship_name → RelationType`、`intimacy_level → Intimacy`(clamp -100..100)、`description → Description` |
| `Career` / `CharacterCareer` | **无对应** | 见 8.5 |

**幂等与断点续传设计（gaea 必须比 MuMu 强）**：

1. **任务持久化**：注册 `tasks.KindBookImport Kind = "book_import"`，`Payload` 存 `{filePath, extractMode, tailChapterCount, importMode, targetProjectDir}`，`Result` 存 `{preview, stepStates, failedSteps, projectID}`。
2. **阶段 A（解析 + 反推）幂等**：产物落 `workspace/.gaea/work/bookimport/{taskID}/preview.json`（沿用 gaea `work/` 目录约定，见 `.gaea/work`）。重跑前若文件存在且 `chapters` 与源文件的 mtime/size 一致 ⇒ 直接复用（**这就是 MuMu 缺的断点**）。
3. **阶段 B（落库）幂等**：每一步写一个 `stepState` 标记文件 `{taskID}/steps.json`：`{"project": "done", "outlines": "done", "chapters": "done", "worldview": "failed", "careers": "done", "entities": "pending"}`。重启后调度器续跑（gaea `tasks.Manager.Start()` 已支持 running→queued 复原，见 `gaea_tasks.go:8` 注释）。
4. **单步幂等键**（照抄 MuMu 的去重集合，但落成可查询的规则）：
   - 项目：`uniqueProjectDir(novelsDir, title)`（gaea 已有，同名追加序号）
   - 大纲：`{ProjectDir}/outline.json` 覆盖写（导入语义 = 全量重写）
   - 章节：`WriteChapter(n, content)` 幂等覆盖；`chapter_number` 用**服务端权威编号**而非客户端传入
   - 世界观：`Sections[]` 按 `ID` 合并覆盖（对齐 gaea `SaveSection`）
   - 职业：先删本项目全部 Career 再重建（对齐 `book_import_service.py:1798-1803`）
   - 角色：`existingNames` 名字去重（**同名跳过，不覆盖**）
   - 关系：`(fromID, toID)` 有向对去重
   - 组织成员：`(orgID, charID)` 对去重
   - `member_count`：**不要像 MuMu 那样 `+= 1` 累加**（重跑会翻倍，`book_import_service.py:2192, 2231`），改为最后 `len(members)` 一次性赋值
5. **取消语义**：每个 AI 调用（每次批）前后各查一次 `ctx.Err()`——修补 D2。

### 8.4 阶段 P3 · 前端（Wails 绑定 + React）

**绑定方法**（建议名，遵循 gaea `NovelB` 命名风格；样式见 `internal/app/bindings_novel.go`）：

```go
// 阶段 A
func (a *App) NovelImportAnalyze(filePath, extractMode string, tailChapterCount int) (ImportTaskView, error)
func (a *App) NovelImportPreview(taskID string) (ReconstructPreview, error)

// 阶段 B（复用 gaea tasks 调度器，异步 + 事件推送）
func (a *App) NovelImportApply(taskID string, payload ImportApplyPayload) (ImportTaskView, error)
func (a *App) NovelImportRetryFailed(taskID string, steps []string) (ImportTaskView, error)

// 通用
func (a *App) NovelImportTasks() []tasks.Task        // 复用 GaeaTaskList
func (a *App) NovelImportCancel(taskID string) error // 复用 GaeaTaskCancel
```

**进度通道**：不要做 SSE（Wails 桌面端无必要）。复用 gaea 既有 `a.emit("gaea-task", ...)`（`gaea_tasks.go:93-103`）——`tasks.Task` 已带 `Progress`/`Message`/`OutputTail`/`Error`，前端订阅同一通道即可，**与任务中心零新增协议**。

**前端状态机（照抄 MuMu 的 4 步，但去掉 sessionStorage hack）**：
```
Step0 上传/选文件 + 解析范围（Select tail|full + InputNumber step=5 min=5 max=55）
Step1 解析中（进度条 + 当前 message + 取消）
Step2 预览修订（项目信息 6 字段 + 逐章 Collapse 改 title/summary/content）+ [确认导入]
Step3 生成导入（进度条 + 失败步骤列表 + [重试全部失败步骤] / [跳过，直接进入项目]）
```
> **不要抄 sessionStorage 缓存**：gaea 任务已持久化在 DB，页面刷新后 `NovelImportTasks()` 直接查到 `running/completed` 的任务并续上。MuMu 那套 6 小时过期 + 配额降级 + 404 清缓存，是「后端任务不持久」逼出来的补偿复杂度，**gaea 抄了就是负资产**。

**解析范围锁定**（值得抄的产品语义）：任务创建后 `rangeLocked=true`，范围选择器 disabled + Alert 提示「拆书任务会按创建任务时的解析范围执行。若需修改范围，请点击『重新开始』后重新上传并解析。」（`BookImport.tsx:721-728`）。

### 8.5 职业体系的取舍（决策点，需 captain 定调）

MuMu 的 `Career` / `CharacterCareer` 是一套**独立数据模型**（主/副职业、阶段树、属性加成、世界观规则），gaea 的 `types` 里**完全没有对应概念**。三个选项：

| 方案 | 做法 | 成本 | 收益 |
|---|---|---|---|
| A · 完整移植 | 新增 `types.Career` / `CareerStage` / `CharacterCareer`，落 `careers.json`；拆书流水线生成之 | 高（新数据模型 + 新 UI + 新 prompt） | 与 MuMu 能力对等，支撑「职业成长」类玩法 |
| B · 降级为角色字段 | 不建 Career 表；把 AI 返回的职业信息压进 `Character.Notes` 或 `Motivation` | 低 | 保留信息不丢，但无结构化查询/成长 |
| C · 只搬运 Prompt，不落库 | 生成职业体系 → 写进 `WorldviewFile` 的 `rules`/`factions` section，作为设定文本存在 | 中低 | **推荐**：职业体系本质是「世界观的一部分」，gaea 世界观已是 6 维 section，加一个 `careers(职业体系)` section 即可，零新数据模型，AI 生成的角色也仍能引用职业名做个性化 |

> 本文建议 **C**：把 MuMu 的「3 主 + 2 副职业树」生成为 `worldview.json` 的一个新 section（`id: "careers"`, `title: "职业体系"`），角色生成时把该 section 内容作为 P1 上下文注入 prompt，并在 `Character.Notes` 里记录其职业名。这样**用 gaea 已有的数据模型承载 MuMu 的能力**，避免为一次性导入需求引入长期维护负担。若后续要做「职业成长系统」，再升级到 A 方案。

### 8.6 分阶段执行计划（每刀独立提交、可回退、可验收）

| 刀 | 范围 | 新增/改动 | 验收 |
|---|---|---|---|
| **T2-1** | 解析引擎 | `internal/bookimport/parse.go` + `parse_test.go`；`novel_import_handler.go` 改为委托 | `go test ./internal/bookimport/...`；UTF8/GB18030/BOM/无标题三组 fixture 分章数正确 |
| **T2-2** | 反推模板 | `prompts/book-import-project.json`、`prompts/book-import-outline.json` | `go test ./internal/prompt/...`（模板可解析）；模板 `name` 与代码 `eng.Get(...)` 键一致 |
| **T2-3** | 反推编排 | `internal/bookimport/reconstruct.go`（采样、批处理、字段归一化、fallback） | 表驱动单测：AI 返回乱序/缺字段/非 JSON ⇒ 全部走 fallback 且章数对齐 |
| **T2-4** | 落库编排 | `internal/bookimport/apply.go`（幂等键 + stepState） | 同一 payload 连跑两次 ⇒ 章节数/角色数不变（幂等断言） |
| **T2-5** | 任务与绑定 | `tasks.KindBookImport` 注册 + `bindings` 五方法 + gen_bindings 重生成 | `go test ./internal/app/...`（`TestBindingsCompleteness` 方法集对账通过） |
| **T2-6** | 前端 4 步向导 | 复用 `gaea-task` 事件通道 | 手动走查：上传 → 解析 → 修订 → 导入 → 失败步骤重试 |

---

## 9. 给下游（t7 审校 / t8 总报告）的引用锚点

| 主题 | 文件:行号 |
|---|---|
| 编码链 5 级 + ignore 兜底 | `backend/app/services/txt_parser_service.py:21-37` |
| 清洗顺序（\u3000 替换 / \n{4,} 压缩） | `txt_parser_service.py:39-45` |
| 强标题正则三条 | `txt_parser_service.py:15-19` |
| 弱标题四条件 | `txt_parser_service.py:119-133` |
| 固定窗口兜底切分 | `txt_parser_service.py:135-168` |
| 前言章 ≥200 字阈值 | `txt_parser_service.py:79-89` |
| tail/full 归一化（API 侧） | `backend/app/api/book_import.py:53-59` |
| tail/full 归一化（Service 侧） | `book_import_service.py:104-109` |
| 预览裁剪 | `book_import_service.py:894-911` |
| 落库裁剪 + 重编号 + 大纲补齐 | `book_import_service.py:828-892` |
| 素材采样（前 3 章 × 2000 字） | `book_import_service.py:1158-1162` |
| 大纲批次输入构造 | `book_import_service.py:1363-1373` |
| 5 章批循环 | `book_import_service.py:1291-1332` |
| 单条大纲字段级归一化 | `book_import_service.py:1397-1444` |
| 规则兜底大纲结构 | `book_import_service.py:1446-1466` |
| 世界观逐字段覆盖 + updated 计数 | `book_import_service.py:1727-1742` |
| 职业幂等清理（先关联后主表） | `book_import_service.py:1798-1803` |
| 职业名白名单注入 prompt | `book_import_service.py:1895-1904` |
| 角色去重集合预加载 | `book_import_service.py:1932-1965` |
| 五阶段落库写入 | `book_import_service.py:1970-2233` |
| 步骤失败隔离模式 | `book_import_service.py:328-397` |
| step_failures SSE 消息 | `book_import_service.py:414-430` |
| 步骤级重试白名单 | `book_import_service.py:464-472` |
| JSON 重试负反馈提示 | `backend/app/services/ai_service.py:638-690` |
| SSE 双重 await 范式 | `backend/app/api/book_import.py:149-210` |
| SSE 信封格式 | `backend/app/utils/sse_response.py:233-344` |
| Prompt 反推项目信息全文 | `backend/app/services/prompt_service.py:2480-2531` |
| Prompt 反推大纲全文 | `prompt_service.py:2534-2607` |
| Prompt 世界观全文 | `prompt_service.py:82-195` |
| Prompt 批量角色全文 | `prompt_service.py:198-321` |
| 模板双层降级（用户自定义 > 类属性） | `prompt_service.py:2816-2860` |
| RTCO map 遍历顺序不定 | `internal/prompt/prompt.go:192-205` |
| gaea 任务调度器（持久化 + 重启续跑） | `internal/gaea/tasks/tasks.go:1-140`；`internal/app/gaea_tasks.go:39-103` |
| gaea 现有导入（三格式 + 无兜底分章） | `internal/app/novel_import_handler.go:37-202` |
| gaea types 数据模型 | `internal/types/types.go:24-165` |

---

## 10. 一句话交接

**MuMu 的拆书能力 = 「尾章采样 + 五段式 Prompt 反推项目与大纲 + 5 章批量 + 字段级 fallback 归一化 + 步骤级失败隔离重试」这套工程手法；gaea 的增量 = 「任务持久化 + 断点续传 + 幂等落库」这套它完全没有的底盘。两者拼起来才是完整方案。**
