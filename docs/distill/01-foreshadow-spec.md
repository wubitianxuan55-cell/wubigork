# gaea 伏笔管理能力升级 · 可落地实施规格

- 来源：`clones/MuMuAINovel` v1.5.5 伏笔管理域源码蒸馏（分析报告见 `t1-foreshadow-source-analysis.md`）
- 目标工程：gaea（Go + Wails + React），Go module `github.com/gaea/gaea`
- 本轮**不写生产代码**，仅产出规格
- 约束：本规格的所有决策都可回溯到报告中的 `文件:行号` 证据

---

## 1. 定位与设计原则

### 1.1 gaea 现状（必须继承，不可回退）

| 现有资产 | 位置 | 处置 |
|---|---|---|
| `types.Foreshadow` / `ForeshadowStatus` / `ForeshadowFile` | `internal/types/types.go:180-212` | **扩展**，不是替换 |
| `analysis.GenerateStableID(type, chapterFile, content)` | `internal/analysis/analysis.go:211` | **保留**，作为唯一 ID 来源 |
| `syncForeshadows` 同步进 `foreshadows.json` | `internal/analysis/analysis.go:116-160` | **增强**匹配能力，保留「不重置既有状态」原则 |
| `lintForeshadowItems` 5 类确定性体检 | `internal/app/novel_foreshadow_lint_handler.go:64-121` | **保留并扩展**（MuMu 没有此能力） |
| `buildForeshadows` 场景圣经注入 | `internal/novelcontext/novelcontext.go:220-238` | **扩展为分层注入** |
| `InjectNovelContextSections` 生成前注入 | `internal/app/create_chapter_handler.go:605-660` | **扩展层数与约束文本** |
| `ForeshadowHandler.SaveForeshadows` 全量写回 | `internal/app/foreshadow_handler.go:13-50` | 保留，增加状态机校验 |

### 1.2 三条核心原则（从 MuMu 源码提炼）

1. **「只有埋入会创建记录，回收只更新」**（`foreshadow_service.py:1361-1362`）→ 回收动作必须能匹配到已存在的埋入条目，否则记录为**跳过原因**而非新建。
2. **「缺失值优于猜值」**（`foreshadow_service.py:1428-1431`）→ 计划回收章未知就不填，绝不给默认值，否则产生假超期。
3. **「精确 ID 失效时宁可不匹配」**（`foreshadow_service.py:1300-1302`）→ 有 `reference_foreshadow_id` 但查不到时，**禁止回落内容匹配**。

### 1.3 存储选型决策

MuMu 用关系表；gaea 现在用 JSON 文件。规格建议：

- **保持 `foreshadows.json` 为唯一事实来源**（与 gaea 全项目一致：`pm.ReadForeshadows()` / `pm.WriteForeshadows()`，无 DB 依赖、可 git 版本化、可 diff）。
- **仅把「章节号冗余」与「运行时视图」的内存结构升级**，不引入 SQLite 表作为主存储。
- 若后续确需按 `status`/`到期章` 做高频检索（>2000 条），再上 §2.3 的**只读索引表**（可重建缓存，非事实来源）。

> 决策依据：MuMu 之所以需要 3 个 `*_chapter_number` 冗余列（`models/foreshadow.py:39` `:43` `:47` 注释「冗余存储便于查询」），是因为章节 ID 与章号的映射需要 join `chapters` 表。gaea 的 `PlantedIn` 用的是**章节文件名**（`types.go:196`），文件名前导数字即章号（`chapterNumOf()`，`novel_foreshadow_lint_handler.go:45-56`），**无需冗余列**、无需 join、无需 DB。

---

## 2. 数据模型

### 2.1 Go 结构体（扩展 `internal/types/types.go`）

```go
// ── 伏笔（v2，扩展自 MuMuAINovel foreshadows 表 + 保留 gaea 原生能力）──

// ForeshadowStatus 伏笔状态（5 态，对齐 MuMu schemas/foreshadow.py:8-14）
type ForeshadowStatus string

const (
    ForeshadowPending    ForeshadowStatus = "pending"            // 已规划未埋入
    ForeshadowPlanted    ForeshadowStatus = "planted"            // 已埋入
    ForeshadowHinted     ForeshadowStatus = "hinted"             // 已暗示（gaea 原生，语义介于 pending/planted）
    ForeshadowResolved   ForeshadowStatus = "resolved"           // 已回收
    ForeshadowPartial    ForeshadowStatus = "partially_resolved" // 部分回收
    ForeshadowAbandoned  ForeshadowStatus = "abandoned"          // 已废弃
)
```

> 兼容说明：gaea 现役 3 态为 `planted/hinted/revealed`（`types.go:185-189`）。为**不破坏存量 `foreshadows.json`**，`ForeshadowRevealed = "revealed"` 必须**保留为别名**，并在读取时归一化 `revealed → resolved`；写入统一用新值。JSON 读取层需接受两套值。

```go
// ForeshadowCategory 伏笔分类（对齐 MuMu schemas/foreshadow.py:23-31，7 值）
type ForeshadowCategory string

const (
    ForeshadowIdentity     ForeshadowCategory = "identity"     // 身世
    ForeshadowMystery      ForeshadowCategory = "mystery"      // 悬念
    ForeshadowItem         ForeshadowCategory = "item"         // 物品
    ForeshadowRelationship ForeshadowCategory = "relationship" // 关系
    ForeshadowEvent        ForeshadowCategory = "event"        // 事件
    ForeshadowAbility      ForeshadowCategory = "ability"      // 能力
    ForeshadowProphecy     ForeshadowCategory = "prophecy"     // 预言
)

// ForeshadowSourceType 来源（对齐 models/foreshadow.py:32）
type ForeshadowSourceType string

const (
    ForeshadowSourceAnalysis ForeshadowSourceType = "analysis" // 分析提取（可再生副产物，可批量删）
    ForeshadowSourceManual   ForeshadowSourceType = "manual"   // 手动登记（作者资产，只重置不删）
)

// ResolveStatus 回收时机判定（对齐 foreshadow_service.py:955-967 的四值枚举）
type ResolveStatus string

const (
    ResolveMustNow ResolveStatus = "must_resolve_now" // 本章必须回收
    ResolveOverdue ResolveStatus = "overdue"          // 已超期
    ResolveNotYet  ResolveStatus = "not_yet"          // 计划在未来章，禁止提前回收
    ResolveNoPlan  ResolveStatus = "no_plan"          // 无明确回收计划
)

// Foreshadow 伏笔追踪条目 v2。
// 字段增删遵循：新增全部可选（omitempty），既有字段名不改。
type Foreshadow struct {
    // ── 标识（沿用 gaea 原生）──
    ID       string             `json:"id"`       // stable_id，由 analysis.GenerateStableID 生成
    Category ForeshadowCategory `json:"category"` // v2 扩为 7 值；读取时接受旧 4 值

    // ── 文本（MuMu: title / content / hint_text / resolution_text）──
    Title          string `json:"title,omitempty"`           // models/foreshadow.py:26，≤200 字
    Description    string `json:"description"`               // ★ 既有字段，= MuMu 的 content
    HintText       string `json:"hint_text,omitempty"`       // 埋入时的暗示文本（原文摘录）
    ResolutionText string `json:"resolution_text,omitempty"` // 回收时的揭示文本（原文摘录）

    // ── 来源 ──
    SourceType     ForeshadowSourceType `json:"source_type,omitempty"` // analysis / manual
    SourceMemoryID string               `json:"source_memory_id,omitempty"` // = 稳定 ID（幂等去重键）
    SourceAnalysis string               `json:"source_analysis_ref,omitempty"` // 分析记录引用（可选，避免 MuMu 的 D5 死字段）

    // ── 章节关联（用章节文件名，非 ID —— 与 gaea 一致）──
    PlantedIn       string `json:"planted_in"`                    // ★ 既有，埋入章文件名
    TargetResolveIn string `json:"target_resolve_in,omitempty"`   // 计划回收章文件名（新增，对应 MuMu target_resolve_chapter_number）
    RevealedIn      string `json:"revealed_in,omitempty"`         // ★ 既有，实际回收章文件名（= MuMu actual_resolve_chapter_id）

    // ── 状态 ──
    Status ForeshadowStatus `json:"status"`

    // ── 评分（新增，对齐 models/foreshadow.py:62-64）──
    Importance float64 `json:"importance,omitempty"` // 0.0-1.0，默认 0.5
    Strength   int     `json:"strength,omitempty"`   // 1-10，默认 5
    Subtlety   int     `json:"subtlety,omitempty"`   // 1-10，默认 5

    // ── 长线 ──
    IsLongTerm bool `json:"is_long_term"` // ★ 既有

    // ── 关联（新增，对齐 models/foreshadow.py:68-71）──
    RelatedCharacters   []string `json:"related_characters,omitempty"`
    RelatedForeshadows  []string `json:"related_foreshadow_ids,omitempty"` // ★ 伏笔链
    Tags                []string `json:"tags,omitempty"`

    // ── 备注（新增）──
    Notes           string `json:"notes,omitempty"`            // 创作备注（仅作者可见）
    ResolutionNotes string `json:"resolution_notes,omitempty"` // 回收方式说明

    // ── 注入控制（新增，对齐 models/foreshadow.py:78-80）──
    AutoRemind           *bool `json:"auto_remind,omitempty"`             // 默认 true
    RemindBeforeChapters int   `json:"remind_before_chapters,omitempty"`  // 默认 5，1..20
    IncludeInContext     *bool `json:"include_in_context,omitempty"`      // 默认 true

    // ── 时间戳（新增，ISO8601）──
    CreatedAt  string `json:"created_at,omitempty"`
    UpdatedAt  string `json:"updated_at,omitempty"`
    PlantedAt  string `json:"planted_at,omitempty"`
    ResolvedAt string `json:"resolved_at,omitempty"`
}

// ForeshadowFile foreshadows.json 完整结构（v1 兼容：Items 字段名不变）
type ForeshadowFile struct {
    SchemaVersion int          `json:"schema_version,omitempty"` // 新增，v2 = 2；缺省视为 1
    Items         []Foreshadow `json:"items"`
}
```

**字段映射对照表（MuMu → gaea）**：

| MuMu 字段 | gaea v2 | 说明 |
|---|---|---|
| `content` | `Description` | **不重命名** `description`，避免破坏既有 JSON 与前端 |
| `title` | `Title` | 新增 |
| `hint_text` | `HintText` | 新增 |
| `resolution_text` | `ResolutionText` | 新增 |
| `plant_chapter_id` + `plant_chapter_number` | `PlantedIn`（单一文件名） | 合并，章号由 `chapterNumOf()` 派生 |
| `target_resolve_chapter_id/number` | `TargetResolveIn` | 新增（**本域最大增量**） |
| `actual_resolve_chapter_id/number` | `RevealedIn` | 既有 |
| `status` | `Status` | 5 态 + 兼容别名 |
| `source_memory_id` | `SourceMemoryID` | 稳定 ID，幂等键 |
| `source_type` | `SourceType` | 驱动清理语义 |
| `related_characters/related_foreshadow_ids/tags` | 同名 | 新增 |
| `importance/strength/subtlety` | 同名 | 新增；**`urgency` 不落库**（运行时计算，见 §3.2） |
| `auto_remind/remind_before_chapters/include_in_context` | 同名（bool 用指针以区分未设置） | 新增 |
| `created_at/updated_at/planted_at/resolved_at` | 同名 ISO8601 | 新增 |
| `notes/resolution_notes` | 同名 | 新增 |
| `category` | `Category`（7 值） | 扩展值域 |
| `project_id` | — | gaea 项目 = 目录，无需字段 |

### 2.2 JSON 持久化（主存储）

路径沿用 gaea 现有约定（`pm.ReadForeshadows()` / `pm.WriteForeshadows()`）。

```json
{
  "schema_version": 2,
  "items": [
    {
      "id": "planted_001_a3f9c2b41d08",
      "category": "mystery",
      "title": "绿头发的视觉符号",
      "description": "主角初见时注意到对方一缕异常绿发，暗示其非人族血脉。",
      "hint_text": "她鬓角有一缕颜色深得不自然的绿。",
      "source_type": "analysis",
      "source_memory_id": "planted_001_a3f9c2b41d08",
      "planted_in": "001.md",
      "target_resolve_in": "015.md",
      "status": "planted",
      "importance": 0.8,
      "strength": 8,
      "subtlety": 7,
      "is_long_term": false,
      "related_characters": ["林雪"],
      "tags": ["身世", "血脉"],
      "resolution_notes": "由主角识破其血脉时揭晓",
      "auto_remind": true,
      "remind_before_chapters": 5,
      "include_in_context": true,
      "created_at": "2026-01-19T10:05:40Z",
      "updated_at": "2026-01-19T10:05:40Z",
      "planted_at": "2026-01-19T10:05:40Z"
    }
  ]
}
```

**幂等写入契约**：同一 `(PlantedIn, 归一化 Description)` → 同一 `SourceMemoryID`（见 §3.1）→ 写回时按 `ID` 原地更新，绝不追加重复条目。这与 gaea 现有 `syncForeshadows` 的「同 ID 已存在则保持既有条目不动」（`analysis.go:141`）一致，但**升级为「原地更新可选字段」**（MuMu 行为，`foreshadow_service.py:1406-1417`）。

### 2.3 只读索引表 DDL（可选，性能优化；非事实来源）

仅当条目数 >2000 或需要复杂筛选时启用。**可从 JSON 全量重建**，删除后不影响正确性。

**SQLite**：

```sql
-- gaea 主库（Hephaestus.db 或项目库）中的伏笔只读索引。纯缓存，可随时 DROP 重建。
CREATE TABLE IF NOT EXISTS foreshadow_index (
    id                TEXT PRIMARY KEY,           -- = Foreshadow.ID
    project_id        TEXT NOT NULL,
    title             TEXT NOT NULL,
    content           TEXT NOT NULL,              -- description
    category          TEXT,
    status            TEXT NOT NULL,              -- pending|planted|hinted|resolved|partially_resolved|abandoned
    source_type       TEXT,                       -- analysis|manual

    plant_chapter_num          INTEGER,           -- chapterNumOf(planted_in)，NULL=无
    target_resolve_chapter_num INTEGER,           -- chapterNumOf(target_resolve_in)，NULL=无计划（★ 关键：NULL 而非 0）
    actual_resolve_chapter_num INTEGER,

    is_long_term            INTEGER NOT NULL DEFAULT 0,
    importance              REAL    NOT NULL DEFAULT 0.5,
    strength                INTEGER NOT NULL DEFAULT 5,
    subtlety                INTEGER NOT NULL DEFAULT 5,
    auto_remind             INTEGER NOT NULL DEFAULT 1,
    remind_before_chapters  INTEGER NOT NULL DEFAULT 5,
    include_in_context      INTEGER NOT NULL DEFAULT 1,

    related_characters      TEXT,                 -- JSON 数组字符串
    related_foreshadow_ids  TEXT,                 -- JSON 数组字符串（伏笔链）
    tags                    TEXT,                 -- JSON 数组字符串
    notes                   TEXT,
    resolution_notes        TEXT,
    hint_text               TEXT,
    resolution_text         TEXT,

    created_at  TEXT,
    updated_at  TEXT,
    planted_at  TEXT,
    resolved_at TEXT
);

CREATE INDEX IF NOT EXISTS ix_foreshadow_index_project  ON foreshadow_index(project_id);
CREATE INDEX IF NOT EXISTS ix_foreshadow_index_status   ON foreshadow_index(project_id, status);
CREATE INDEX IF NOT EXISTS ix_foreshadow_index_due      ON foreshadow_index(project_id, status, target_resolve_chapter_num);
CREATE INDEX IF NOT EXISTS ix_foreshadow_index_planted  ON foreshadow_index(project_id, plant_chapter_num);
```

**Postgres**（若 gaea 未来接 PG）：

```sql
CREATE TABLE IF NOT EXISTS foreshadow_index (
    id                        TEXT PRIMARY KEY,
    project_id                TEXT NOT NULL,
    title                     TEXT NOT NULL,
    content                   TEXT NOT NULL,
    category                  TEXT,
    status                    TEXT NOT NULL
        CHECK (status IN ('pending','planted','hinted','resolved','partially_resolved','abandoned')),
    source_type               TEXT CHECK (source_type IN ('analysis','manual')),

    plant_chapter_num         INTEGER CHECK (plant_chapter_num IS NULL OR plant_chapter_num >= 1),
    target_resolve_chapter_num INTEGER CHECK (target_resolve_chapter_num IS NULL OR target_resolve_chapter_num >= 1),
    actual_resolve_chapter_num INTEGER CHECK (actual_resolve_chapter_num IS NULL OR actual_resolve_chapter_num >= 1),

    is_long_term            BOOLEAN NOT NULL DEFAULT FALSE,
    importance              REAL    NOT NULL DEFAULT 0.5 CHECK (importance >= 0 AND importance <= 1),
    strength                INTEGER NOT NULL DEFAULT 5 CHECK (strength BETWEEN 1 AND 10),
    subtlety                INTEGER NOT NULL DEFAULT 5 CHECK (subtlety BETWEEN 1 AND 10),
    auto_remind             BOOLEAN NOT NULL DEFAULT TRUE,
    remind_before_chapters  INTEGER NOT NULL DEFAULT 5 CHECK (remind_before_chapters BETWEEN 1 AND 20),
    include_in_context      BOOLEAN NOT NULL DEFAULT TRUE,

    related_characters      JSONB,
    related_foreshadow_ids  JSONB,
    tags                    JSONB,
    notes                   TEXT,
    resolution_notes        TEXT,
    hint_text               TEXT,
    resolution_text         TEXT,

    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ,
    planted_at  TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS ix_fi_project ON foreshadow_index(project_id);
CREATE INDEX IF NOT EXISTS ix_fi_status  ON foreshadow_index(project_id, status);
CREATE INDEX IF NOT EXISTS ix_fi_due     ON foreshadow_index(project_id, status, target_resolve_chapter_num);
CREATE INDEX IF NOT EXISTS ix_fi_planted ON foreshadow_index(project_id, plant_chapter_num);
```

**约束设计说明**：
- `target_resolve_chapter_num` 允许 NULL 且**不复用 0 表示「无计划」**——直接对应 MuMu「不设默认值避免误报超期」原则（`foreshadow_service.py:1428-1431`）。SQLite 的 `NULL` 在 `ORDER BY ... ASC` 中排最前，而 MuMu 用 `.nulls_last()`（`:104`）——**gaea 必须显式写 `ORDER BY (target_resolve_chapter_num IS NULL), target_resolve_chapter_num ASC`**（SQLite）或 `ORDER BY target_resolve_chapter_num ASC NULLS LAST`（PG）。
- 数据库层 CHECK 约束**复刻 MuMu Pydantic 的边界**（`schemas/foreshadow.py:49-64`）：`importance 0..1`、`strength/subtlety 1..10`、`remind_before_chapters 1..20`。
- **不建** `Foreshadow` 的 FK 到章节表：gaea 章节是文件，无表。

---

## 3. 核心算法规格（Go 落地）

### 3.1 稳定 ID（沿用 gaea 现有实现）

```go
// 已存在：internal/analysis/analysis.go:211 GenerateStableID(type, chapterFile, content)
// 保持签名不变。MuMu 参考实现（foreshadow_service.py:24-48）：
//   content_norm = strings.ToLower(strings.TrimSpace(content))
//   content_hash = md5(content_norm)[:12]
//   chapter_hash = md5(chapter_id)[:8]
//   return f"{type}_{chapter_hash}_{content_hash}"
```
要求：
- `content` 必须先 `TrimSpace` + `ToLower`（`foreshadow_service.py:42`）。
- 哈希截断长度保持一致（内容 12、章节 8），保证 ID 短且可读。
- **`type` 参与 ID** → 同一内容作为 `planted` 与 `resolved` 得到不同 ID（`foreshadow_service.py:48`）。gaea 若要「按 ID 关联回收」，回收匹配必须用**埋入时的 ID**，而非重新生成（见 §3.3 策略 1）。

### 3.2 运行时紧急度（不落库）

```go
// UrgencyLevel 对齐 models/foreshadow.py:156-178
// 返回 0=不紧急 1=需关注 2=急需回收 3=已超期
func UrgencyLevel(f Foreshadow, currentChapterNum int) int {
    if f.Status != ForeshadowPlanted || f.TargetResolveIn == "" {
        return 0
    }
    target := chapterNumOf(f.TargetResolveIn)
    if target <= 0 {
        return 0
    }
    remaining := target - currentChapterNum
    switch {
    case remaining < 0:
        return 3 // 已超期
    case remaining <= 2:
        return 2 // 急需回收
    case remaining <= f.RemindBeforeChapters: // 默认 5
        return 1 // 需关注
    default:
        return 0
    }
}
```

**规格决策（修正 MuMu 缺陷 D15）**：把前置判断从 `status != 'planted'` 放宽为 `status ∉ {resolved, abandoned}`，使 `partially_resolved` 也能算出回收压力（MuMu 的 `:166` 会把它判成 0）。

```go
if f.Status == ForeshadowResolved || f.Status == ForeshadowAbandoned {
    return 0
}
```

**统一阈值**：后端 `<=2` 为准（`models/foreshadow.py:173`），前端只渲染后端结果，不再自算（修正缺陷 D7 的两处不一致）。

**修正缺陷 D8**：`AutoRemind` 的过滤必须**统一应用**到所有「提醒类」查询上（MuMu 只在 `get_pending_resolve_foreshadows` 应用，`:586`）。规格：`AutoRemind == false` 的条目不出现在「近期参考」段，但**仍出现在「必须回收」与「超期」段**（这两段是硬约束，不因提醒开关而消隐）——这是对 MuMu 行为的**明确化**，而非简单照抄。

### 3.3 分析结果 → 伏笔登记表同步

**入口**：`analysis.SyncForeshadows(projectDir, chapterFile, chapterNum, changes []ForeshadowChange, existing []Foreshadow) (SyncResult, error)`

```go
type SyncResult struct {
    PlantedCount    int      `json:"planted_count"`
    ResolvedCount   int      `json:"resolved_count"`
    CreatedCount    int      `json:"created_count"`
    UpdatedIDs      []string `json:"updated_ids"`
    CreatedIDs      []string `json:"created_ids"`
    MatchedByContent int     `json:"matched_by_content"`
    SkippedResolveCount int  `json:"skipped_resolve_count"`
    SkippedReasons  []SkipReason `json:"skipped_reasons"` // ★ 修正 MuMu 缺陷 D3
    Errors          []string `json:"errors"`
}

type SkipReason struct {
    Kind    string `json:"kind"`     // invalid_reference|already_resolved|not_planted|no_match|limit_reached|empty_content
    RefID   string `json:"ref_id,omitempty"`
    Title   string `json:"title,omitempty"`
    Message string `json:"message"`
}
```

**硬上限**：`const maxNewForeshadowsPerChapter = 5`（`foreshadow_service.py:1287`）。**只约束新建**，更新已有条目不占额度（`:1424-1426`）。

**`type == "resolved"` 处理流程**（严格按序，早退即记录 `SkippedReasons`）：

```
1. refID = change.ReferenceForeshadowID
2. if refID != "":
       existing = lookup(refID)
       if existing == nil:                        // 对应 :1303-1312
           SKIP(kind=invalid_reference, ref_id=refID,
                message="reference_foreshadow_id 无效或已删除")
           continue                               // ★ 绝不回落内容匹配（:1300-1302）
3. if refID == "" and existing == nil:            // 对应 :1314-1323
       matched, score = MatchByContent(change, plantedItems, 0.5)
       if matched != nil: existing = matched; matchedByContent = true
4. if existing != nil:                            // 对应 :1326-1332
       if existing.Status == resolved:
            if existing.RevealedIn == chapterFile: SKIP(already_resolved, "已在本章回收过")
            else: SKIP(already_resolved, "已在第N章回收")
            continue
5. if existing != nil && existing.Status == planted:   // 对应 :1335-1356
       写 status=resolved, RevealedIn=chapterFile, ResolvedAt=now,
          ResolutionText=change.Description（非空时）
       resolvedCount++; updatedIDs += id
       从内存 planted 列表移除（防止同批重复匹配，:1356）
6. else if existing != nil:                       // :1357-1358
       SKIP(not_planted, "状态不是 planted")
7. else:                                          // :1359-1367 ★ 核心原则
       SKIP(no_match, "未找到匹配的已埋入伏笔，跳过回收（不创建新记录）")
       continue
```

**`type == "planted"` 处理流程**（对应 `:1369-1464`）：

```
1. if change.Description == "": SKIP(empty_content, "伏笔内容为空"); continue       // :1370-1373
2. title = change.Title; if title == "": title = truncate(description, 50) + "..."  // :1375-1377
3. stableID = GenerateStableID("planted", chapterFile, description)                 // :1380-1382
4. existing = lookupBy(stableID == SourceMemoryID) 
                OR (Title == title AND PlantedIn == chapterFile AND SourceType == "analysis")  // :1385-1402
5. if existing != nil:                                                              // :1406-1417
       更新：Title, Description,
             Strength/Subtlety（仅当 change 提供了非零值 —— MuMu 用 fs_data.get(k, 现有值)）
             HintText ← change.Keyword
             Category, IsLongTerm, RelatedCharacters
             TargetResolveIn ← change.EstimatedResolveChapter（仅当非空）
             SourceMemoryID ← stableID
       不入新条目、不占额度
6. else:                                                                            // :1421-1464
       if newCount >= 5: SKIP(limit_reached, "已达每章新伏笔上限(5个)"); continue
       item := Foreshadow{
           ID: stableID, Category: change.Category, Title: title,
           Description: description, HintText: change.Keyword,
           SourceType: analysis, SourceMemoryID: stableID,
           PlantedIn: chapterFile, Status: planted,
           PlantedAt: now, CreatedAt: now, UpdatedAt: now,
           TargetResolveIn: change.EstimatedResolveChapter,   // 可为空 → 不设默认值 ★ :1428-1431
           IsLongTerm: change.IsLongTerm,
           Importance: min(float64(change.Strength)/10.0, 1.0),  // ★ :1447 派生关系
           Strength: change.Strength（默认 5）, Subtlety: change.Subtlety（默认 5）,
           RelatedCharacters: change.RelatedCharacters,
           AutoRemind: true, RemindBeforeChapters: 5, IncludeInContext: true,
       }
       newCount++; plantedCount++; createdCount++
```

**评分派生规则（唯一）**：`Importance = min(Strength / 10.0, 1.0)`（`foreshadow_service.py:1447`）。**不从 `subtlety` 派生 `importance`**（subtlety 是隐藏度，与重要性正交）。

**单条容错**：每条 `change` 用独立错误边界包裹（`:1466-1469`），失败只 append `Errors`，**不中断整批**。批次末尾一次性落盘（`:1471`），保证原子性。

### 3.4 内容匹配兜底（六策略加权评分）

**签名**：`MatchByContent(resolved ForeshadowChange, planted []Foreshadow, minSimilarity float64) (*Foreshadow, float64)`

**第 0 步：标题后缀剥离**（`foreshadow_service.py:1584-1589`）——对 `["回收","揭示","解答","兑现"]` 顺序检查 `strings.HasSuffix`，命中即剥（只剥一次，`break`）：

```go
cleanTitle := resolvedTitle
for _, suffix := range []string{"回收", "揭示", "解答", "兑现"} {
    if strings.HasSuffix(cleanTitle, suffix) {
        cleanTitle = strings.TrimSuffix(cleanTitle, suffix)
        break
    }
}
```

**六策略评分**（逐候选，注意 `max` 与 `+=` 的语义差异）：

```go
for _, fs := range planted {
    score := 0.0
    fsTitle := strings.TrimSpace(fs.Title)
    rt := strings.TrimSpace(resolved.Title)

    // 策略1 标题族（取最大，不累加）
    if rt != "" && fsTitle != "" {
        switch {
        case rt == fsTitle:                       score = 1.0                       // :1604-1606
        case cleanTitle != "" && cleanTitle == fsTitle: score = 0.95               // :1607-1609
        case strings.Contains(rt, fsTitle) || strings.Contains(fsTitle, rt):
                                                   score = max(score, 0.8)         // :1610-1612
        case cleanTitle != "" && (strings.Contains(cleanTitle, fsTitle) || strings.Contains(fsTitle, cleanTitle)):
                                                   score = max(score, 0.75)        // :1613-1615
        default:
            ov := WordOverlap(rt, fsTitle)
            score = max(score, ov*0.7)                                             // :1616-1620
        }
    }
    // 策略2 关键词命中（取最大）
    if resolved.Keyword != "" && fs.Description != "" && strings.Contains(fs.Description, resolved.Keyword) {
        score = max(score, 0.75)                                                   // :1623-1625
    }
    // 策略3 内容 n-gram 相似（取最大）
    if resolved.Description != "" && fs.Description != "" {
        score = max(score, WordOverlap(resolved.Description, fs.Description)*0.6)  // :1628-1630
    }
    // 策略4 引用章号一致（累加）
    if resolved.ReferenceChapter > 0 && fs.PlantedIn != "" &&
       chapterNumOf(fs.PlantedIn) == resolved.ReferenceChapter {
        score += 0.15                                                              // :1633-1635
    }
    // 策略5 分类一致（累加）
    if resolved.Category != "" && fs.Category == resolved.Category {
        score += 0.1                                                               // :1638-1640
    }
    // 策略6 关联角色 Jaccard（累加）
    if len(resolved.RelatedCharacters) > 0 && len(fs.RelatedCharacters) > 0 {
        inter := intersectSize(resolved.RelatedCharacters, fs.RelatedCharacters)
        union := unionSize(resolved.RelatedCharacters, fs.RelatedCharacters)
        if union > 0 { score += float64(inter) / float64(union) * 0.1 }            // :1643-1645
    }
    // 采纳：严格大于 + 过阈值（:1648-1650）
    if score > bestScore && score >= minSimilarity {
        bestScore, best = score, &fs
    }
}
```

**采纳规则**：`score > bestScore && score >= minSimilarity`，默认 `minSimilarity = 0.5`（`:1552`）。用 `>` 而非 `>=` → 同分取**先出现者**（`planted` 列表按 `planted_in` asc 排序，即取最早埋入的）。

### 3.5 字符级 n-gram 相似度（Go 实现，中文安全）

```go
// WordOverlap 对齐 foreshadow_service.py:1657-1691。
// ★ Go 必须按 rune 切分，按 byte 切会让中文 n-gram 全部错位。
func WordOverlap(a, b string) float64 {
    norm := func(s string) string {
        s = strings.ToLower(s)
        s = strings.ReplaceAll(s, " ", "")
        s = strings.ReplaceAll(s, "\n", "")
        return s
    }
    ngrams := func(s string, n int) map[string]struct{} {
        r := []rune(norm(s))
        if len(r) < n {
            return map[string]struct{}{string(r): {}}
        }
        out := make(map[string]struct{}, len(r)-n+1)
        for i := 0; i+n <= len(r); i++ {
            out[string(r[i:i+n])] = struct{}{}
        }
        return out
    }
    jaccard := func(a, b map[string]struct{}) float64 {
        inter := 0
        for k := range a {
            if _, ok := b[k]; ok { inter++ }
        }
        union := len(a) + len(b) - inter
        if union == 0 { return 0 }
        return float64(inter) / float64(union)
    }
    o2 := jaccard(ngrams(a, 2), ngrams(b, 2))
    o3 := jaccard(ngrams(a, 3), ngrams(b, 3))
    return o2*0.4 + o3*0.6    // ★ 3-gram 权重更高（更精确，:1690-1691）
}
```

---

## 4. 分层注入规格（生成侧）

### 4.1 完整分层模型（合并 MuMu 两处实现，消除缺陷 D9）

| 层 | 名称 | 数据来源 | 排序 | 条数上限 | 单条截断 | 禁止事项 |
|---|---|---|---|---|---|---|
| L1 | 本章必须回收 | `Status=planted AND chapterNum(TargetResolveIn)==cur` | `importance` desc | **不限** | 内容 100 字 | — |
| L2 | 超期待回收 | `Status=planted AND chapterNum(TargetResolveIn)<cur` | `TargetResolveIn` asc | **3** | 内容 80 字 | — |
| L3 | 近期待回收（仅参考） | `Status=planted AND chapterNum(TargetResolveIn) <= cur+lookahead` 且 **> cur** | `TargetResolveIn` asc | **5** | — | **明确禁止本章提前回收** |
| L4 | 本章计划埋入 | `Status=pending AND chapterNum(PlantedIn)==cur` | `importance` desc | 不限 | 内容 80 字 | — |

**参数值**：`lookahead` 默认 5（`foreshadow_service.py:562`），上限 20（`schemas/foreshadow.py:187`）。章节生成路径与 API 路径使用**同一份参数**（消除 D9 的 `lookahead=3` vs `=5` 分歧）。

**远期伏笔不注入**（docstring 第 4 条，`:729`「远期伏笔 → 不发送，防止干扰」）——即 `TargetResolveIn > cur+lookahead` 的条目不出现。

### 4.2 分层文本模板（逐字对齐 `foreshadow_service.py:752-800`）

```
【🎯 本章必须回收的伏笔 - 请务必在本章完成回收】
- ID:{id8} | {title}
  埋入章节：第{plantNum}章
  伏笔内容：{content[:100]}{"..." if len>100}
  回收提示：{resolution_notes}     ← 仅当非空

【⚠️ 超期待回收伏笔 - 请尽快回收】
- ID:{id8} | {title} [已超期{cur-targetNum}章]
  埋入章节：第{plantNum}章，原计划第{targetNum}章回收
  伏笔内容：{content[:80]}

【📋 近期待回收伏笔（仅供参考，请勿在本章回收）】
⚠️ 以下伏笔尚未到回收时机，本章请勿提前回收，仅作为剧情背景了解
- {title}（计划第{targetNum}章回收，还有{remaining}章）

【✨ 本章计划埋入伏笔】
- {title}
  伏笔内容：{content[:80]}
  埋入提示：{hint_text}            ← 仅当非空
```

**实现修正**：超期段**不要**无条件加 `...`（MuMu 缺陷 D14，`:770`），应与「计划埋入」段一致按长度判断；ID 展示用前 8 字符（`:754` `:768`）。

### 4.3 与 gaea 现有注入的合并方式

`internal/app/create_chapter_handler.go:605-660` 现有单层注入：
```
## 未回收伏笔（创作约束）
以下伏笔已埋设尚未回收，写作时不得与之矛盾；可自然推进，不要强行提前揭穿：
<body>
```
**改为**：`## 伏笔调度（分层约束）`，内部按 §4.2 四段渲染；保留原有的**负向约束语**（「不得与之矛盾」「不要强行提前揭穿」）作为 L3 段落的定型语（与 MuMu `:783` 语义一致）。

**预算体系（沿用 gaea 现有常量，不照搬 MuMu 无预算的做法）**：

| 常量 | 值 | 位置 |
|---|---|---|
| `ctxForeshadowBudget` | 1600 rune | `create_chapter_handler.go:570` |
| `ctxForeshadowLineLen` | 100 rune | `:572` |
| `ctxForeshadowMaxItems` | 15 条 | `:573` |
| 场景圣经内预算 | `foreshadowBudget = 280` rune / `foreshadowLineMax = 90` rune | `novelcontext.go:36` `:40` |

**条数按层级优先级消耗预算**：L1（不限，但受总预算约束）→ L2（≤3）→ L3（≤5）→ L4（不限，受总预算），累计达 1600 rune 或 15 条即停。**L1 与 L4 享有优先权**（硬约束优先于参考信息）。

---

## 5. API 契约（Wails 绑定 / HTTP 双形态）

gaea 现有伏笔接口均为 Wails 绑定方法（`internal/app/foreshadow_handler.go`）。下表同时给出 HTTP 等价路径，便于 React 侧统一调用。

| # | 方法名 / HTTP | 入参 | 出参 | 对应 MuMu 端点 |
|---|---|---|---|---|
| 1 | `ListForeshadows(projectID, filter)` / `GET /api/foreshadows/projects/{projectId}` | `status?` `category?` `sourceType?` `isLongTerm?` `page=1` `limit=50(1..100)` | `ForeshadowListResponse{total, items[], stats}` | `api/foreshadows.py:28` |
| 2 | `GetForeshadowStats(projectID, currentChapter?)` / `GET .../stats` | `currentChapter?` | `ForeshadowStats{total,pending,planted,hinted,resolved,partiallyResolved,abandoned,longTermCount,overdueCount}` | `:69` |
| 3 | `GetChapterForeshadowContext(projectID, chapterNum, opts)` / `GET .../context/{chapterNumber}` | `includePending=true` `includeOverdue=true` `lookahead=5(1..20)` | `ForeshadowContextResponse`（**含 `mustResolve`**，修正 D1） | `:91` |
| 4 | `GetPendingResolveForeshadows(projectID, currentChapter, lookahead=5)` / `GET .../pending-resolve` | `currentChapter` **必填** | `{total, items[]}` | `:128` |
| 5 | `GetForeshadow(id)` / `GET /api/foreshadows/{id}` | — | `Foreshadow` | `:160` |
| 6 | `CreateForeshadow(req)` / `POST /api/foreshadows` | `ForeshadowCreate` | `Foreshadow` | `:186` |
| 7 | `UpdateForeshadow(id, req)` / `PUT /api/foreshadows/{id}` | `ForeshadowUpdate`（**拒绝 `status`**，修正 D10） | `Foreshadow` | `:211` |
| 8 | `DeleteForeshadow(id)` / `DELETE /api/foreshadows/{id}` | — | `{message, id}` | `:238` |
| 9 | `PlantForeshadow(id, req)` / `POST /api/foreshadows/{id}/plant` | `{chapterFile, chapterNumber, hintText?}` | `Foreshadow` | `:265` |
| 10 | `ResolveForeshadow(id, req)` / `POST /api/foreshadows/{id}/resolve` | `{chapterFile, chapterNumber, resolutionText?, isPartial=false}` | `Foreshadow` | `:296` |
| 11 | `AbandonForeshadow(id, reason?)` / `POST /api/foreshadows/{id}/abandon` | `reason?`（query） | `Foreshadow` | `:327` |
| 12 | `SyncForeshadowsFromAnalysis(projectID, req)` / `POST .../sync-from-analysis` | `{chapterFiles?: [], overwriteExisting=false, autoSetPlanted=true}` | `SyncFromAnalysisResponse`（**填充 `newForeshadows`/`skippedReasons`**，修正 D3） | `:358` |
| 13 | `LintForeshadows()`（**gaea 独有，保留**） | — | `ForeshadowLintReport` | `novel_foreshadow_lint_handler.go:154` |

### 5.1 请求/响应结构（Pydantic → Go tag 对照）

```go
type ForeshadowCreate struct {
    Title                    string   `json:"title"`                                 // required, 1..200
    Content                  string   `json:"content"`                               // required, min 1
    HintText                 string   `json:"hint_text,omitempty"`
    ResolutionText           string   `json:"resolution_text,omitempty"`
    PlantChapterNumber       int      `json:"plant_chapter_number,omitempty"`        // >=1
    TargetResolveChapterNumber int    `json:"target_resolve_chapter_number,omitempty"` // >=1
    IsLongTerm               bool     `json:"is_long_term"`
    Importance               float64  `json:"importance"`                            // 0..1, 默认 0.5
    Strength                 int      `json:"strength"`                              // 1..10, 默认 5
    Subtlety                 int      `json:"subtlety"`                              // 1..10, 默认 5
    RelatedCharacters        []string `json:"related_characters,omitempty"`
    Tags                     []string `json:"tags,omitempty"`
    Category                 string   `json:"category,omitempty"`
    Notes                    string   `json:"notes,omitempty"`
    ResolutionNotes          string   `json:"resolution_notes,omitempty"`
    AutoRemind               bool     `json:"auto_remind"`                           // 默认 true
    RemindBeforeChapters     int      `json:"remind_before_chapters"`                // 1..20, 默认 5
    IncludeInContext         bool     `json:"include_in_context"`                    // 默认 true
}
```
**创建不变量**（服务层强制，`foreshadow_service.py:162/165/170`）：`source_type = "manual"`、`status = "pending"`、`urgency` 不存在（运行时算）。

```go
type ForeshadowUpdate struct {
    Title, Content, HintText, ResolutionText *string   `json:"...,omitempty"`
    PlantChapterNumber, TargetResolveChapterNumber *int `json:"...,omitempty"`
    IsLongTerm *bool    `json:"is_long_term,omitempty"`
    Importance *float64 `json:"importance,omitempty"`
    Strength, Subtlety *int `json:"...,omitempty"`
    RelatedCharacters, RelatedForeshadowIDs, Tags []string `json:"...,omitempty"`
    Category, Notes, ResolutionNotes *string `json:"...,omitempty"`
    AutoRemind, IncludeInContext *bool `json:"...,omitempty"`
    RemindBeforeChapters *int `json:"...,omitempty"`
    // ★ 故意不含 Status：状态只能经 plant/resolve/abandon 三个专用端点迁移（修正 D10）
}
```
用**指针**区分「未提供」与「显式置零」——对应 MuMu 的 `model_dump(exclude_unset=True)`（`:216`）。

```go
type ForeshadowContextResponse struct {
    ChapterNumber   int          `json:"chapter_number"`
    ContextText     string       `json:"context_text"`
    PendingPlant    []Foreshadow `json:"pending_plant"`
    MustResolve     []Foreshadow `json:"must_resolve"`     // ★ 修正 D1
    PendingResolve  []Foreshadow `json:"pending_resolve"`
    Overdue         []Foreshadow `json:"overdue"`
    RecentlyPlanted []Foreshadow `json:"recently_planted"` // 实现之，或删除该字段（不保留空占位）
}
```
**规格决策**：`recently_planted` 要么**真实实现**（取 `PlantedIn` 在 `cur-3..cur-1` 的 `planted` 条目，供模型铺垫），要么从契约中**删除**。**不允许保留永空字段**（修正 D11）。

```go
type SyncForeshadowsFromAnalysisRequest struct {
    ChapterFiles      []string `json:"chapter_files,omitempty"`       // 空=全部章节的分析记录
    OverwriteExisting bool     `json:"overwrite_existing"`            // ★ 必须实现或删除（修正 D2）
    AutoSetPlanted    bool     `json:"auto_set_planted"`              // ★ 同上
}
```
**规格决策**：
- `AutoSetPlanted`（默认 true）→ 真正控制「分析出的 planted 是否直接落 `status=planted`」；为 false 时落 `pending`（供作者先复核再埋入）。
- `OverwriteExisting`（默认 false）→ false 时对已存在条目只**补空缺字段**（不覆盖非空的 title/description/hint_text）；true 时**全字段覆盖**。MuMu 是「总是覆盖一批字段」（`:1406-1417`），等价于 `true`；本规格默认 `false` 更安全。

```go
type ForeshadowStats struct {
    Total             int `json:"total"`
    Pending           int `json:"pending"`
    Planted           int `json:"planted"`
    Hinted            int `json:"hinted"`            // gaea 原生
    Resolved          int `json:"resolved"`
    PartiallyResolved int `json:"partially_resolved"`
    Abandoned         int `json:"abandoned"`
    LongTermCount     int `json:"long_term_count"`
    OverdueCount      int `json:"overdue_count"`     // 仅在 currentChapter > 0 时计算
}
```
`Total` 取分状态计数之和（`foreshadow_service.py:858`），不额外扫描。异常时返回**全 0 结构而非错误**（`:892-901`）——但 gaea 是强类型 Go，建议**返回 `(stats, error)`**，由调用方决定降级（比 MuMu 的静默全 0 更可诊断）。

### 5.2 错误语义

| 场景 | MuMu 行为 | gaea 规格 |
|---|---|---|
| 伏笔不存在 | `404 伏笔不存在`（`api/foreshadows.py:171`） | `ErrForeshadowNotFound`，Wails 层返回结构化错误 |
| 无项目访问权 | `verify_project_access` 抛 `HTTPException` | gaea 是本地单用户，**跳过该检查** |
| 未知内部错误 | `HTTPException(500, f"...: {e}")`（把异常字符串回传） | 不回传内部错误串；返回 `ErrInternal` + 日志 |
| 检索类失败 | `return []` / 全 0 字典 | 返回 `err`，调用方决定降级（章节生成路径**必须降级**，不 fail 生成） |

---

## 6. Prompt 规格

### 6.1 分析侧 Prompt（新增到 gaea 分析模板）

**槽位注入**（对齐 `prompt_service.py:1072-1077`）：

```xml
<existing_foreshadows priority="P1">
【已埋入伏笔列表 - 用于回收匹配】
以下是本项目中已埋入但尚未回收的伏笔，分析时如发现章节内容回收了某个伏笔，请使用对应的ID：

{existing_foreshadows}
</existing_foreshadows>
```

**任务指令**（对齐 `prompt_service.py:1055-1059`）：

```
【🔴 伏笔追踪任务（重要）】
系统已提供【已埋入伏笔列表】，当你识别到章节中有回收伏笔时：
1. 必须从列表中找出对应的伏笔ID
2. 在 foreshadows 数组中使用 reference_foreshadow_id 字段关联
3. 如果无法确定是哪个伏笔，reference_foreshadow_id 填 null
```

**字段规范**（逐字照搬 `prompt_service.py:1103-1126`，见分析报告 §5.3）。其中必须保留的三条强约束：
1. `⚠️ 回收伏笔时，标题应与原伏笔标题保持一致，不要添加"回收"等后缀` + 具体示例（`:1111-1112`）
2. `reference_foreshadow_id：【回收时必填】... 如果列表中有标注【ID: xxx】的伏笔，回收时必须使用该ID`（`:1118-1120`）
3. `estimated_resolve_chapter：【必填】预估回收章节号（埋下时必须预估，回收时为当前章节）`（`:1126`）

**自检条款**（`prompt_service.py:1366`）：
```
✅ 【伏笔ID追踪】回收伏笔时，必须从【已埋入伏笔列表】中查找匹配的ID填入 reference_foreshadow_id
```

**动态候选清单三层渲染**（`plot_analyzer.py:223-266`，逐字见分析报告 §5.6）。**第 1 层的紧邻指令是关键**：
```
   ⚠️ 回收时 reference_foreshadow_id 填写: {fs_id}
```

### 6.2 生成侧 Prompt

**priority 随信息完备度升级**（`prompt_service.py:593` vs `:822`）：

| 场景 | priority | 标题 |
|---|---|---|
| 首章（无前文） | `P2` | `【🎯 伏笔提醒】` |
| 续写（有前文） | `P1` | `【🎯 伏笔提醒 - 需关注】` |

渲染形态：
```xml
<foreshadow_reminders priority="P2">
【🎯 伏笔提醒】
{foreshadow_reminders}
</foreshadow_reminders>
```

**约束条款**（`prompt_service.py:610` 与 `:840` 逐字相同，gaea 只需一份）：
```
✅ 如有伏笔提醒，请在本章中适当埋入或回收相应伏笔
```

**gaea 增强（把分层写进约束）**：
```
【伏笔调度规则】
✅ 标注「本章必须回收」的伏笔，务必在本章内完成回收
✅ 标注「已超期」的伏笔，若本章情节允许应优先安排回收
❌ 标注「仅供参考，请勿在本章回收」的伏笔，本章不得提前揭晓或回收
✅ 标注「本章计划埋入」的伏笔，应在不突兀的前提下自然埋入（暗示而非说明）
✅ 如需回收伏笔，请在分析输出中回填该伏笔的 ID
```

---

## 7. 前端交互规格（React）

### 7.1 常量表（对齐 `Foreshadows.tsx:26-43`）

```ts
const STATUS_CONFIG = {
  pending:            { label: '待埋入',   color: 'default',       icon: 'clock' },
  planted:            { label: '已埋入',   color: 'green',         icon: 'bulb' },
  hinted:             { label: '已暗示',   color: 'cyan',          icon: 'eye' },   // gaea 原生
  resolved:           { label: '已回收',   color: 'blue',          icon: 'check' },
  partially_resolved: { label: '部分回收', color: 'orange',        icon: 'exclamation' },
  abandoned:          { label: '已废弃',   color: 'default',       icon: 'close' },
} as const;

const CATEGORY_CONFIG = {
  identity:     { label: '身世', color: 'purple'  },
  mystery:      { label: '悬念', color: 'magenta' },
  item:         { label: '物品', color: 'gold'    },
  relationship: { label: '关系', color: 'cyan'    },
  event:        { label: '事件', color: 'blue'    },
  ability:      { label: '能力', color: 'green'   },
  prophecy:     { label: '预言', color: 'volcano' },
} as const;

// 默认排序口径（Foreshadows.tsx:358-364）
const STATUS_ORDER = { planted: 1, pending: 2, hinted: 3, partially_resolved: 4, resolved: 5, abandoned: 6 };
```

### 7.2 表格（7 列 + 操作列）

| 列 | 字段 | 宽 | 渲染 |
|---|---|---|---|
| 状态 | `status` | 100 | Tag(color+icon)，`sorter` 用 `STATUS_ORDER` |
| 标题 | `title` | — | 可点链接（→详情）；`is_long_term` 追加紫色「长线」Tag；下方叠加紧急度 Badge |
| 分类 | `category` | 80 | 无值 `-` |
| 埋入章节 | `planted_in` 派生章号 | 120 | `第N章`，`defaultSortOrder: 'ascend'`，空值排最后 |
| 计划回收 | `target_resolve_in` 派生章号 | 120 | `第N章`，空值排最后 |
| 重要性 | `importance` | 100 | `★×round(imp*5) + ☆×(5-...)` |
| 来源 | `source_type` | 80 | `analysis`→蓝「分析」，否则绿「手动」 |

**操作列按状态条件渲染**（`Foreshadows.tsx:473-510`）：
| 按钮 | 显示条件 |
|---|---|
| 查看详情 | 恒显 |
| 编辑 | 恒显 |
| 标记埋入 | `status === 'pending'` |
| 标记回收 | `status === 'planted'` |
| 废弃（Popconfirm「确定要废弃这个伏笔吗？」） | `status !== 'abandoned' && status !== 'resolved'` |
| 删除（Popconfirm「确定要删除这个伏笔吗？」） | 恒显 |

### 7.3 紧急度 Badge（前端不自算，修正 D7）

```tsx
// ★ 直接用后端算好的 urgency（0..3），不再本地比较章号
if (f.urgency === 3) return <Badge status="error"   text={`已超期${f.overdueChapters}章`} />;
if (f.urgency === 2) return <Badge status="warning" text={`还剩${f.remainingChapters}章`} />;
if (f.urgency === 1) return <Badge status="default" text={`还剩${f.remainingChapters}章`} />;
return null;
```
（`overdueChapters` / `remainingChapters` / `urgency` 由 `ListForeshadows` 计算并返回；若不想扩契约，前端必须**复用后端同一常量 `<=2`**。）

### 7.4 统计卡片（7 张，`Col span=3`）

总计 / 待埋入（灰）/ 已埋入（绿）/ 已回收（蓝）/ 长线伏笔（青）/ 超期未回收（红高亮）。**超期基准 `currentChapter` 由「有内容的章节最大章号」决定**（`Foreshadows.tsx:128-141`），不要用章节目录长度（可能含未写章节）。

### 7.5 五个 Modal

| Modal | 标题 | 字段 |
|---|---|---|
| ① 添加/编辑 | `编辑伏笔` / `添加伏笔` | `title`(≤200,required) · `category`(Select) · `content`(TextArea 3行,required) · `plant_chapter_number`(≥1) · `target_resolve_chapter_number`(≥1) · `related_characters`(Select 多选) · `importance`(0-1) · `strength`(1-10) · `subtlety`(1-10) · `is_long_term`(Switch) · `hint_text`(TextArea 2行) · `notes`(TextArea 2行) · `auto_remind`(Switch) · `include_in_context`(Switch) · `remind_before_chapters`(1-20) |
| ② 详情 | `伏笔详情`（宽 600） | 正文 `pre-wrap`；title + 状态/长线/分类 Tag；内容 · 暗示文本 · 揭示文本 · 埋入章节（缺值「未设定」）· 计划回收（缺值「未设定」）· 实际回收（仅非空）· 重要性(★) · 强度 N/10 · 隐藏度 N/10 · 关联角色 Tag 列表 · 备注 · 来源文案；footer = 关闭 + 编辑 |
| ③ 标记埋入 | `标记伏笔埋入` | `chapter_id`(required) + `hint_text`(可选) |
| ④ 标记回收 | `标记伏笔回收` | `chapter_id`(required) + `resolution_text`(可选) + `is_partial`(Switch) |
| ⑤ 手动同步 | `手动同步分析伏笔` | 只传 `auto_set_planted`；成功文案 `同步完成: 新增N个伏笔, 跳过M个` |

**编辑态数组归一**（对应 `Foreshadows.tsx:305-309`）：`tags: f.tags ?? []`、`related_characters: f.related_characters ?? []`——避免 Antd Select 收到 `undefined`。

**③④ 的章号反查**：前端把选中的章节文件名对应的章号一并发给后端（`Foreshadows.tsx:227-235` `:250-258`）。

### 7.6 事件联动（必须实现）

监听后台分析任务完成事件；当 `payload.resources` 包含 `'foreshadows'` 且项目匹配时，**自动刷新列表与统计**（`Foreshadows.tsx:149-160`）。这是「分析 → 伏笔自动更新 → UI 可见」闭环的关键触点。gaea 对应事件：分析任务 `TaskSettled`（需在 payload 中带 `resources: ['foreshadows']`）。

### 7.7 工具栏

3 个筛选 Select（状态/分类/来源，`allowClear`）+ 刷新按钮。筛选变化时 **page 重置为 1**。分页默认 `pageSize = 20`（MuMu 前端默认 20，后端默认 50、上限 100）。

---

## 8. 生命周期清理与状态机校验

### 8.1 三个清理入口（语义严格区分）

| 入口 | 触发时机 | 动作 | 保留 |
|---|---|---|---|
| `DeleteChapterForeshadows(chapterFile, onlyAnalysisSource=true)` | 章节内容清空 / 重新生成 | 删除 `PlantedIn==ch` ∨ `RevealedIn==ch`（∨ 分析来源引用）的条目；默认仅 `source_type==analysis` | 手动条目（默认） |
| `CleanChapterAnalysisForeshadows(chapterFile)` | **重新分析前** | ① 删 `source_type==analysis ∧ PlantedIn==ch`；② 回退本章回收的条目（`RevealedIn==ch ∧ status∈{resolved,partially_resolved}` → `planted`，清空 `RevealedIn`/`ResolvedAt`/`ResolutionText`） | 手动条目 |
| `ClearProjectForeshadowsForReset()` | 全新生成 / 项目重置 | ① 删全部 `source_type==analysis`；② 手动条目重置为 `pending`，**清空 `PlantedIn`/`RevealedIn`/`TargetResolveIn`/`PlantedAt`/`ResolvedAt`** | 手动条目记录本身 |

**关键不变量**：`source_type` 是这两类清理的唯一判据。手动条目永不被批量删除。

**修正缺陷 D5**：MuMu 的「按分析来源引用匹配」分支依赖一个**从不写入**的字段，是死代码。gaea 规格：**不引入 `source_analysis_ref` 的匹配分支**，或写入时真正填充它。选择：写入时填充 `SourceAnalysis`（指向分析记录），并在 `DeleteChapterForeshadows` 中使用——这样「重新分析」不会误删其他分析产物。

### 8.2 状态机（强制）

允许的迁移（其余一律拒绝，返回结构化错误）：

| from | to | 入口 |
|---|---|---|
| （新建） | `pending` | `CreateForeshadow`（手动） |
| （新建） | `planted` / `pending` | `SyncForeshadows`（分析；受 `auto_set_planted` 控制） |
| `pending` | `planted` | `PlantForeshadow` / 自动埋入 |
| `pending` | `abandoned` | `AbandonForeshadow` |
| `hinted` | `planted` | `PlantForeshadow` |
| `hinted` | `abandoned` | `AbandonForeshadow` |
| `planted` | `resolved` | `ResolveForeshadow(is_partial=false)` / 分析自动回收 |
| `planted` | `partially_resolved` | `ResolveForeshadow(is_partial=true)` |
| `planted` | `abandoned` | `AbandonForeshadow` |
| `partially_resolved` | `resolved` | `ResolveForeshadow(is_partial=false)` |
| `resolved` | `partially_resolved` | `ResolveForeshadow(is_partial=true)` |
| `resolved` / `partially_resolved` | `planted` | `CleanChapterAnalysisForeshadows`（**仅系统内部回退**，不暴露 API） |
| 任意 | `pending` | `ClearProjectForeshadowsForReset`（**仅系统内部**，且仅手动条目） |

**拒绝的迁移**：`abandoned → *`（需先删除重建）；`resolved → planted` 经 API（仅系统回退）。

**修正缺陷 D6**：自动回收的候选集从 `status == planted` 扩展为 `status ∈ {planted, partially_resolved}`，使部分回收的伏笔可被二次回收完成。对应 `get_planted_foreshadows_for_analysis` 的 WHERE 放宽。

### 8.3 删除联动（对齐 `delete_foreshadow` 的四路清理）

gaea 无向量库依赖，但需保留语义：

1. **章节正文关键句索引**（若 gaea 有）：按 `title` + `description[:50]` 关键词清理。
2. **分析记录中的引用**：遍历该项目的分析产物，移除 `reference_foreshadow_id == id` 的条目（`foreshadow_service.py:306`）以及可确定为同一伏笔的 `planted` 记录（按稳定 ID 或 title+content 前缀匹配，`:316-332`），并重算该分析的 `foreshadows_planted` / `foreshadows_resolved` 计数（`:342-349`）。
3. 返回清理计数（关系记录数 / 引用数 / 受影响分析数），写入日志（`:360-364`）。

---

## 9. 分阶段实施方案

### P0 — 数据模型与状态机（无 Prompt 变更，纯结构性）

| 项 | 内容 | 验收 |
|---|---|---|
| P0.1 | 扩展 `types.Foreshadow` / `ForeshadowStatus` / `ForeshadowCategory`（§2.1） | `revealed` 旧值可读并归一化为 `resolved`；存量 `foreshadows.json` 无需手工迁移 |
| P0.2 | `ForeshadowFile.SchemaVersion`；读写兼容 v1/v2 | 读 v1 文件 → 写回变 v2 且字段不丢；`go test` 覆盖 |
| P0.3 | `UrgencyLevel()` + `ResolveStatus()` 纯函数（§3.2 / §3.4 四值判定） | 表驱动单测覆盖 0/1/2/3 与 4 个 `ResolveStatus` |
| P0.4 | 状态机校验器（§8.2） | 非法迁移返回结构化错误；合法迁移全通过 |
| P0.5 | 常量集中声明（`lookaheadDefault=5` `lookaheadMax=20` `maxNewPerChapter=5` `remindBeforeDefault=5` `urgencyUrgent=2`） | 全仓无散落魔数 |

**风险**：字段命名。**必须保留 `description`**（不改名为 `content`），否则前端与既有 JSON 全线断裂。

### P1 — 分层注入（生成侧收益最大）

| 项 | 内容 | 验收 |
|---|---|---|
| P1.1 | `BuildForeshadowContext(projectID, chapterNum, opts)` 实现四层（§4.1） | 四层条数与排序符合表；`lookahead` 生效；远期不注入 |
| P1.2 | 渲染 §4.2 模板文本 | 逐字符对齐模板；空层不输出标题 |
| P1.3 | 接入 `create_chapter_handler.go`，替换现有单层注入（§4.3） | 预算 1600 rune / 15 条生效；L1/L4 优先 |
| P1.4 | 接入 `novelcontext.go` 场景圣经（`foreshadowBudget = 280`） | 场景圣经仍 ≤280 rune，且按 L1>L2>L3 优先级取样 |
| P1.5 | 生成侧 Prompt 槽位与 priority（P2 首章 / P1 续写）+ 约束条款（§6.2） | 首章模板含 `priority="P2"`，续写含 `P1` |
| P1.6 | **消除 D9**：删除 `chapter_context_service.py` 式的重复实现，全仓唯一入口 | `grep` 确认只有一处分层逻辑 |

### P2 — 分析驱动自动回收（闭环核心）

| 项 | 内容 | 验收 |
|---|---|---|
| P2.1 | `MatchByContent()` + `WordOverlap()`（§3.4 / §3.5） | 表驱动单测：完全相同=1.0、后缀剥离命中、`min_similarity=0.5` 边界、同分取先出现者 |
| P2.2 | `SyncForeshadows()` 三级匹配 + 两道防重 + 上限（§3.3） | `refID` 无效时不回落内容匹配（**关键回归测试**）；`resolved` 无匹配不建记录；每章新建 ≤5 |
| P2.3 | 分析侧候选清单三层渲染（§6.1） | L1 条目带 `⚠️ 回收时 reference_foreshadow_id 填写: {id}` |
| P2.4 | 分析侧 Prompt 槽位 + 字段规范 + 自检条款（§6.1） | 模板含 `<existing_foreshadows priority="P1">` 与三条强约束 |
| P2.5 | `SyncResult` 完整填充（含 `SkippedReasons`，修正 D3） | 各 `SkipReason.Kind` 均有测试覆盖 |

### P3 — 清理、统计与 Lint 扩展

| 项 | 内容 | 验收 |
|---|---|---|
| P3.1 | 三个清理入口（§8.1） | 手动条目在 `onlyAnalysisSource=true` 下永不删除 |
| P3.2 | 删除联动（§8.3） | 分析记录中的引用被清理且计数重算 |
| P3.3 | `GetForeshadowStats` 含 `OverdueCount` / `LongTermCount` / 分状态（§5.1） | 与 `ListForeshadows` 单测断言一致 |
| P3.4 | **Lint 扩展**：在现有 5 类检查上新增 `overdue`（超期）与 `unplanned`（无计划回收章且已埋入 >10 章） | 新增 2 个 finding code；既有 5 类不回归 |
| P3.5 | `overdue_count` 不复用全量行扫描（修正 D13） | 单次遍历计数 |

### P4 — 前端

| 项 | 内容 | 验收 |
|---|---|---|
| P4.1 | 常量表 + 6 态（含 `hinted`）（§7.1） | 状态 Tag 全覆盖 |
| P4.2 | 表格 7 列 + 条件操作列（§7.2） | 按钮显隐规则逐条对齐 |
| P4.3 | 紧急度 Badge 用后端 `urgency`（§7.3，修正 D7） | 前端无章号比较逻辑 |
| P4.4 | 7 张统计卡（§7.4） | `currentChapter` = 有内容章节最大章号 |
| P4.5 | 5 个 Modal（§7.5） | 数组字段归一；章号反查 |
| P4.6 | 事件联动自动刷新（§7.6） | 分析任务完成后列表与统计刷新 |

### P5 — 可选：只读索引表（§2.3）

仅当条目 >2000 或筛选性能不达标时启用。**必须先有 P0–P4**。

---

## 10. 测试规格

### 10.1 纯函数单测（无 LLM、无网、可 `go test`）

| 用例 | 断言 |
|---|---|
| `TestUrgencyLevel` | `target-cur = -1,0,1,2,3,5` → `3,2,2,2,1,1`（`remind=5`）；`target=0`/无计划 → `0`；`resolved`/`abandoned` → `0`；`partially_resolved` → **非 0**（D15 修正） |
| `TestResolveStatus` | 四值判定全覆盖，含 `TargetResolveIn == ""` → `no_plan` |
| `TestWordOverlap` | 相同串=1.0；完全无关=0.0；**中文 2/3-gram 按 rune 切分**（用「绿头发的视觉符号」对「绿头发的视觉符号回收」验证，期望 >0.7） |
| `TestMatchByContent` | 精确标题=1.0；剥离「回收」后缀后命中=0.95；包含=0.8；`keyword` 命中=0.75；仅内容相似 0.6×ov；阈值 0.5 边界（0.499 不采纳 / 0.5 采纳）；同分取先出现者 |
| `TestGenerateStableID` | 同章同内容两次调用同 ID；**大小写/首尾空白归一**（`" A "` == `"a"`）；不同 type 不同 ID |
| `TestStateMachine` | §8.2 全部合法迁移通过；非法迁移（`abandoned→planted`、`pending→resolved`）被拒 |
| `TestSchemaCompat` | 读 v1 `foreshadows.json`（只有 `id/category/description/planted_in/revealed_in/status/is_long_term`）→ 结构兼容、`revealed` 归一化为 `resolved`、写回不丢字段 |

### 10.2 同步集成测试（对齐 gaea 现有 `analysis/foreshadow_sync_test.go` 风格）

| 用例 | 断言 |
|---|---|
| `TestSync_ResolvedNoMatchDoesNotCreate` | 分析给出 `type=resolved` 但无匹配 → 条目数不变、`SkippedResolveCount=1`、`SkippedReasons[0].Kind=no_match` |
| `TestSync_InvalidRefIDNoContentFallback` | 提供不存在的 `reference_foreshadow_id`，且存在一个标题高度相似的 planted 条目 → **不得回收该条目**（关键回归，对应 `foreshadow_service.py:1300-1302`） |
| `TestSync_Idempotent` | 同一章分析两次 → 条目数不变、`UpdatedIDs` 非空、`CreatedCount=0` |
| `TestSync_MaxNewPerChapter` | 单章给 8 条 `planted` → 新建 5 条、`SkippedReasons` 有 3 条 `limit_reached` |
| `TestSync_NoEstimatedResolveStaysNull` | `estimated_resolve_chapter` 缺失 → `TargetResolveIn == ""`，**不得填默认值**，且不被判超期 |
| `TestSync_PartialResolvedCanBeResolved` | `partially_resolved` 条目可被后续分析回收为 `resolved`（D6 修正） |
| `TestSync_CorruptFileAborts` | 伏笔文件损坏（读取失败）→ 放弃同步、不覆盖文件（沿用既有测试 `foreshadow_sync_test.go:134` 的语义） |
| `TestSync_OverwriteFlag` | `overwrite_existing=false` 时非空 `title/hint_text` 不被覆盖；`true` 时被覆盖 |

### 10.3 注入测试（对齐 `create_chapter_context_test.go` 风格）

| 用例 | 断言 |
|---|---|
| `TestContext_FourLayers` | 构造 planted/target 各异的条目，断言四段标题、条数上限（L2≤3、L3≤5）、L3 含「请勿在本章回收」 |
| `TestContext_NoFarFuture` | `target > cur+lookahead` 的条目不出现在任何层 |
| `TestContext_Budget` | 总长 ≤ `ctxForeshadowBudget`、总条数 ≤ `ctxForeshadowMaxItems`；L1/L4 优先于 L3 |
| `TestContext_EmptyLayersOmitted` | 某层无数据时不输出其标题（无空标题） |
| `TestContext_MustResolveInResponse` | `must_resolve` 字段**非空且有值**（D1 修正） |
| `TestContext_NoDuplicateAcrossLayers` | 同一伏笔不同时出现在 L2 与 L3（`target == cur` 只在 L1，`target < cur` 只在 L2） |

### 10.4 Lint 回归

| 用例 | 断言 |
|---|---|
| `TestLint_Existing5Codes` | `ordering`/`status-mismatch`/`dangling`/`stale`/`duplicate` 全部不回归 |
| `TestLint_NewOverdueCode` | 超期条目产生 `overdue` finding |
| `TestLint_NewUnplannedCode` | 无 `TargetResolveIn` 且埋入 >10 章 → `unplanned` finding |
| `TestLint_LongTermExempt` | `IsLongTerm` 条目不产生 `stale`/`unplanned`（沿用 `:102` 豁免） |

### 10.5 手工验收清单（Prompt 效果）

1. 单章分析含明确回收 → 对应条目 `status=resolved`、`RevealedIn` 正确、`ResolutionText` 被填充。
2. 单章分析含新伏笔 → 新条目 `Status=planted`、`Importance == min(Strength/10, 1)`。
3. 重新分析同一章 → 无重复条目。
4. 生成含「必须回收」伏笔的章节 → 正文确实回收了该伏笔（人读判定）。
5. 生成含「近期参考」伏笔的章节 → 正文**未**提前揭晓。
6. 后台分析任务完成 → 伏笔页自动刷新。

---

## 11. 决策记录（ADR 摘要）

| # | 决策 | 依据 | 替代方案与否决原因 |
|---|---|---|---|
| A1 | 主存储保持 `foreshadows.json`，不引 SQLite 表 | gaea 全项目文件化；章节号可从文件名派生，无需 join 与冗余列 | 否决「照搬 MuMu 关系表」——引入 DB 依赖与两处事实源 |
| A2 | `description` 字段名不改 | 存量 JSON / 前端 / 既有 handler | 否决「对齐 MuMu 的 `content`」——破坏性 rename |
| A3 | 5 态（新增 `pending`/`partially_resolved`/`abandoned`），保留 `hinted`，`revealed` 作读取别名 | MuMu 5 态 + gaea 原生 3 态取并 | 否决「只保留 3 态」——丢失「先规划后埋入」与「部分回收」 |
| A4 | 新增 `target_resolve_in`（计划回收章） | 超期预测的唯一前提；MuMu 最大价值点 | 否决「用 lint 的 stale 阈值替代」——无法预测，只能事后发现 |
| A5 | `urgency` 不落库，运行时计算 | `models/foreshadow.py:166-178` 的既有取舍 | 落库需在每次章节推进后全量重写，脆 |
| A6 | 计划回收章缺失时不填默认值 | `foreshadow_service.py:1428-1431` | 否决「填 planted+10」——制造假超期 |
| A7 | `refID` 无效时禁止回落内容匹配 | `foreshadow_service.py:1300-1302` | 否决「总是兜底匹配」——错绑同名伏笔，污染登记表 |
| A8 | `UpdateForeshadow` 不含 `status` | 修正 D10 | 否决「白名单过滤」——契约层排除比运行时过滤更早失败 |
| A9 | 内容匹配保留 `min_similarity = 0.5` 与六策略权重 | `foreshadow_service.py:1548-1651` 的标定值 | 无更好标定数据，保持原值 |
| A10 | L1/L2 不受 `auto_remind` 抑制 | 硬约束语义（对 MuMu 行为的明确化，修正 D8） | 否决「完全静音」——会让本章必须回收的伏笔静默丢失 |
| A11 | 保留并扩展 gaea 原生 Lint | gaea 独有资产，MuMu 无 | 否决「用 MuMu 的分层替代 lint」——两者职责不同（注入 vs 体检） |
| A12 | `recently_planted` 要么真实实现要么删字段 | 修正 D11 | 否决「保留空占位」——契约承诺即负债 |

---

## 12. 文件清单（后续实现阶段预期改动路径）

| 路径 | 改动性质 |
|---|---|
| `internal/types/types.go` | 扩展（`:180-212` 伏笔段） |
| `internal/types/foreshadow_ext.go`（新） | 纯函数：`UrgencyLevel` / `ResolveStatus` / `WordOverlap` / `MatchByContent` / 状态机校验 |
| `internal/types/foreshadow_ext_test.go`（新） | §10.1 单测 |
| `internal/analysis/analysis.go` | 扩展 `syncForeshadows` → `SyncForeshadows`（§3.3） |
| `internal/analysis/foreshadow_sync_test.go` | 扩展 §10.2 |
| `internal/app/foreshadow_handler.go` | 扩展 12 个绑定方法（§5） |
| `internal/app/foreshadow_context.go`（新） | `BuildForeshadowContext` 四层（§4） |
| `internal/app/create_chapter_handler.go` | 替换注入逻辑（`:605-660`）；新增 Prompt 槽位（§4.3 / §6.2） |
| `internal/app/novel_foreshadow_lint_handler.go` | 新增 `overdue` / `unplanned` 检查（P3.4） |
| `internal/novelcontext/novelcontext.go` | `buildForeshadows` 改按层优先级取样（`:220-238`） |
| `internal/project/project.go` | `ReadForeshadows`/`WriteForeshadows` 加 schema 版本兼容 |
| `internal/stats/stats.go` | 扩展统计口径（`:64-67`） |
| `frontend/src/pages/Foreshadows.tsx`（或 gaea 对应页面） | §7 全部 |
| `frontend/src/services/api.ts` | 伏笔 API 客户端（对齐 `api.ts:1290-1370`） |

---

*规格完成。所有决策可回溯至 `clones/MuMuAINovel` v1.5.5 的具体行号，对照 gaea 现役实现的差距已在分析报告 §7 列表化说明。*
