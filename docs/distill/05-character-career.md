# 05 · 角色状态机与职业/境界体系 —— MuMuAINovel 源码蒸馏 + gaea 落地规格

> 事实基线：`clones/MuMuAINovel` @ commit **600be7038539e4dd0568b63e6e534fdcfaf91687**（main, tag v1.5.5）
> 引用格式：`clones/MuMuAINovel@600be703:<相对路径>:<起-L止>`
> 行号来源：全部来自 grep / read 工具真实输出。
> 纪律：凡无法从源码确认的内容，显式标注「未能从源码确认」。

---

## 0. 一句话结论

MuMu 的角色状态机本质是 **「LLM 章节分析产出稀疏差分 → 后端按章节号单调守卫写回」** 的两段式结构；
职业体系则是 **「项目级职业模板（10 阶/5 阶 + 阶段名表）→ 角色-职业关联（current_stage）→ 分析驱动 ±1 阶推进」** 的三层结构。

gaea 当前对应能力仅有 `internal/analysis/analysis.go:186-209` 的 `syncCharacterStates`（把 `NewState` 直接写进 `Character.Status`），
**既没有独立心理状态字段，也没有章节号单调守卫、没有级联、没有职业/境界体系**。
本报告给出可直接落地的 Go 数据结构与差分更新算法规格。

---

## 1. 目标文件清单（已逐文件读过的真实源码）

| 层 | 文件 | 字节 | 本报告用途 |
|---|---|---|---|
| 差分更新核心 | `backend/app/services/character_state_update_service.py` | 38016 | 状态机主算法（829 行全文读完） |
| 职业模型 | `backend/app/services/career_service.py` | 8834 | 职业体系生成 + 解析落库 |
| 职业差分 | `backend/app/services/career_update_service.py` | 15687 | ±N 阶推进、新职业获得 |
| 自动补角 | `backend/app/services/auto_character_service.py` | 25309 | 大纲后缺失角色补全 + 职业名称→ID 匹配 |
| 自动补组织 | `backend/app/services/auto_organization_service.py` | 19933 | 组织补全 |
| 分析 Prompt | `backend/app/services/prompt_service.py` | 112997 | `PLOT_ANALYSIS` 第 5/5b 节字段契约 |
| 调用编排 | `backend/app/api/chapters.py` | 230679 | 分析→职业→状态→组织 四级调用顺序；角色上下文构建 |
| 备用编排 | `backend/app/api/memories.py` | 19770 | 手动分析入口的同一套编排 |
| 上下文注入 | `backend/app/services/chapter_context_service.py` | 85135 | `_append_current_state_lines` 状态回灌生成侧 |
| 数据模型 | `backend/app/models/{character,career,relationship}.py` | — | 表结构 |
| Schemas | `backend/app/schemas/career.py`, `relationship.py`, `character.py` | — | API 契约 |
| API | `backend/app/api/{careers,characters,relationships,organizations}.py` | — | REST 面 |
| 关系类型种子 | `backend/app/init_relationship_types.py` | — | 21 条预置关系类型 |
| 前端 | `frontend/src/pages/{Careers,RelationshipGraph,Characters}.tsx` | 20638 / 56806 / 62274 | 职业管理页、关系图谱（含职业节点） |

---

## 2. 数据模型对照：MuMu 表 → gaea 现状

### 2.1 MuMu 的「角色 = 角色 ∪ 组织」单表双态设计

`clones/MuMuAINovel@600be703:backend/app/models/character.py:8-53` —— **组织不是独立实体，而是一条 `is_organization=True` 的 Character 行**，
组织专有属性（类型/目的/成员）在同一个 `characters` 表里用可空列表达：

```python
is_organization = Column(Boolean, default=False)          # :19
role_type       = Column(String(50))                      # :22  protagonist/supporting/antagonist
personality     = Column(Text)                            # :25  角色=性格 / 组织=组织特性
organization_type     = Column(String(100))               # :31
organization_purpose  = Column(String(500))               # :32
organization_members  = Column(Text)                      # :33

status                 = Column(String(20), default="active")  # :36 active/deceased/missing/retired/destroyed
status_changed_chapter = Column(Integer)                       # :37
current_state          = Column(Text)                          # :40 心理状态（分析自动写）
state_updated_chapter  = Column(Integer)                       # :41
main_career_id    = Column(String(36), ForeignKey("careers.id", ondelete="SET NULL"))  # :44
main_career_stage = Column(Integer)                                                     # :45
sub_careers       = Column(Text)  # [{"career_id":"xxx","stage":3}, ...]                # :46
```

**要点 1**：`status`（存活生命周期，离散枚举）与 `current_state`（心理/处境自由文本）**是两个正交字段**，各自有独立的章节号水位。
**要点 2**：`main_career_id/stage` 与 `sub_careers(JSON)` 是**冗余快照**，权威在 `character_careers` 关联表（见 §4.2）。

### 2.2 关系与组织的三张表

`clones/MuMuAINovel@600be703:backend/app/models/relationship.py`

- `RelationshipType`（:8-22）：`name / category(family|social|hostile|professional) / reverse_name / intimacy_range / icon`，自增 int 主键，**21 条种子**（`backend/app/init_relationship_types.py:16-45`：父亲/母亲/兄弟/姐妹/子女/配偶/恋人 | 师父/徒弟/朋友/同学/邻居/知己 | 上司/下属/同事/合作伙伴 | 敌人/仇人/竞争对手/宿敌）。
- `CharacterRelationship`（:25-56）：`character_from_id` / `character_to_id` / `relationship_type_id`（可空）/ `relationship_name`（自由文本）/ `intimacy_level`（**默认 50，域 -100..100**，:41）/ `status`（active/broken/past/complicated，:42）/ `started_at` / `ended_at`（**字符串，存"第N章"**，:46-47）/ `source`（ai/manual/imported，:50）。
- `Organization`（:59-84）：`character_id`（**unique**，一对一挂到 Character 行）/ `parent_org_id`（自引用，支持组织树）/ `level` / `power_level`（默认 50，域 0..100）/ `member_count` / `location` / `motto` / `color`。
- `OrganizationMember`（:87-116）：`position` / `rank` / `status`（active/retired/expelled/deceased）/ `joined_at` / `left_at`（故事时间字符串）/ `loyalty`（默认 50）/ `contribution` / `notes`。

### 2.3 职业模型（三层）

`clones/MuMuAINovel@600be703:backend/app/models/career.py`

**第 1 层 · 职业模板 `Career`（:8-44）**
```python
type      = Column(String(20))  # "main" / "sub"                     # :17
stages    = Column(Text)        # JSON [{"level":1,"name":"","description":""}]  # :22
max_stage = Column(Integer, default=10)                              # :23
requirements / special_abilities / worldview_rules                  # :26-28
attribute_bonuses = Column(Text)  # {"strength":"+10%"}              # :31
source    = Column(String(20), default='ai')                         # :34
```
索引：`idx_project_id`、`idx_type`（:39-40）。

**第 2 层 · 角色-职业关联 `CharacterCareer`（:47-77）**
```python
career_type      = Column(String(20))                # main/sub          # :54
current_stage    = Column(Integer, default=1)        # 对应 Career.stages[].level  # :57
stage_progress   = Column(Integer, default=0)        # 0-100，**全仓库无写入方**  # :58
started_at / reached_current_stage_at  # 故事时间线字符串               # :61-62
notes
__table_args__ = (..., Index('idx_character_career', 'character_id', 'career_id', unique=True))  # :73
```
> **关键约束**：`(character_id, career_id)` **唯一**（:73）。语义 = 一个角色在一个职业上只有一行进度，阶段是标量而非历史。

**关于 `stage_progress`**：全部由后端**恒置 0**——共 16 处 `stage_progress=0` 构造点，分布在
`characters.py:425,503,696,730,1114,1164`、`careers.py:745,828`、`wizard_stream.py:944,975`、
`auto_character_service.py:234,247`、`book_import_service.py:2062,2095`、`import_export_service.py:1619,1662`；
`career_update_service.py` **根本不写该字段**（其新建 `CharacterCareer` 只给 `current_stage=1`，`:341`、`:370`）。
唯一能写入非零值的路径是 `careers.py:883`（`PUT .../careers/{career_id}/stage` 接口的入参）。

消费方只有前端组件 `frontend/src/components/CharacterCareerCard.tsx`（:20 类型、:183 提交、:223 进度条、
:385「阶段进度（0-100）」表单项），而该组件**全仓库只有自身第 41 行定义、无任何 import 引用**
（grep `CharacterCareerCard` 在 `frontend/src` 仅命中 `CharacterCareerCard.tsx:41` 与 `:407`）。
→ 结论：**该字段在分析流程中恒为 0，其消费方是未接线的前端组件**。

**第 3 层 · 角色表冗余快照**（`character.py:44-46`，见 §2.1）。

### 2.4 `CharacterCareer` stage 与 `Career.stages[].level` 的解析契约

唯一「把数字阶变成可读阶段名」的地方在 `clones/MuMuAINovel@600be703:backend/app/api/careers.py:642-669`：
```python
stages = json.loads(career.stages) if career.stages else []      # :642
stage_name = "未知阶段"                                            # :645
for stage in stages:                                              # :647
    if stage.get("level") == char_career.current_stage:           # :648
        stage_name = stage.get("name", f"第{char_career.current_stage}阶段")  # :649
        break
```
→ **契约：`current_stage` 必须等于某个 `stages[].level` 才能显示阶段名，否则只有"第N阶"**。
`chapter_context_service.py:570-592` 用同一套 fallback 逻辑把阶段名注入章节生成 Prompt。

### 2.5 gaea 现状对照

`internal/types/types.go:65-80`：
```go
type Character struct {
    ...
    Arc    string `json:"arc,omitempty"`    // 角色弧光轨迹（自由文本）
    Status string `json:"status"`           // Alive / Dead / Missing / Transformed
    ...
}
type Organization struct {               // :104-113
    Members []string `json:"members,omitempty"`  // 成员角色 ID
}
type Relationship struct {               // :116-122
    FromID, ToID string
    RelationType string    // friend/enemy/family/mentor/rival/lover/member/leader
    Intimacy     int       // -100 ~ 100
}
```

| MuMu 能力 | gaea 现状 | 判定 |
|---|---|---|
| `current_state` 心理状态 + 水位章节号 | **无字段** | ❌ 缺失 |
| `status` 离散枚举 + 水位 | `Status` 有字段，**无水位、无级联** | ⚠️ 半成品 |
| `intimacy_level` -100..100 | `Intimacy` 有，域一致 | ✅ 对齐 |
| 关系 status(active/past/broken) + ended_at | **无** | ❌ 缺失 |
| 关系变更历史（description 追加 `[第N章]`） | **无** | ❌ 缺失 |
| 职业/境界体系 | **完全无** | ❌ 缺失 |
| 组织 `PowerLevel string` | 是字符串，源码注释为「实力等级」（`types.go:109`），无固定值域；MuMu 是 `int 0..100` | ⚠️ 类型不同（见 §8 坑 7） |
| 组织成员 `[]string` | 只有 ID 列表，**无 position/rank/loyalty/status** | ❌ 缺失 |
| 关系类型 21 条种子 + category | `RelationType` 是 8 值硬编码枚举 | ⚠️ 需扩 |

---

## 3. 角色状态机：差分更新算法全解

### 3.1 触发点与调用顺序（关键！）

MuMu 有**两条**入口，编排顺序一致但锁粒度不同：

**入口 A · 章节分析后台任务** `clones/MuMuAINovel@600be703:backend/app/api/chapters.py:1286-1385`
```
1. CareerUpdateService.update_careers_from_analysis(...)          # :1287-1314  💼 职业
2. CharacterStateUpdateService.update_from_analysis(...)          # :1317-1352  👤 心理+关系+组织成员
3. CharacterStateUpdateService.update_organization_states(...)    # :1355-1385  🏛️ 组织自身
```
**入口 B · 手动分析** `clones/MuMuAINovel@600be703:backend/app/api/memories.py:215-254`（同样顺序，`entity_changes` 聚合返回）。

> **顺序语义**：职业先写（因为 `_update_sub_career_stage` 要读 `character.sub_careers` JSON 快照），
> 然后心理/关系，最后组织自身。三级相互之间**不共享事务边界**（各自 `db.commit()`），
> 任一级失败都被 `try/except` 吞掉并记日志（`chapters.py:1310-1312`、`:1350-1352`、`:1378-1385`）——**失败不阻断分析**。

### 3.2 输入契约：`character_states` 数组

来自 `PLOT_ANALYSIS` Prompt 的输出（`clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1285-1310`）：

```jsonc
"character_states": [
  {
    "character_name": "张三",
    "survival_status": null,               // null|active|deceased|missing|retired
    "state_before": "犹豫",
    "state_after": "坚定",                  // ← 唯一必填才生效的字段（见 :288-290）
    "psychological_change": "心理变化描述",
    "key_event": "触发事件",
    "relationship_changes": {"李四": "关系改善"},
    "career_changes": {
      "main_career_stage_change": 1,        // 整数 +1/-1/0
      "sub_career_changes": [{"career_name": "炼丹", "stage_change": 1}],
      "new_careers": [],
      "career_breakthrough": "突破描述"
    },
    "organization_changes": [
      {"organization_name": "某门派", "change_type": "promoted",
       "new_position": "长老", "loyalty_change": "忠诚度提升",
       "description": "因立下大功被提拔为长老"}
    ]
  }
]
```
`change_type` 枚举：`joined | left | expelled | betrayed | promoted | demoted`（Prompt 声明 `prompt_service.py:1159`，实现分支 `character_state_update_service.py:528/563/583/604`）。

**顶层** `organization_states`（`prompt_service.py:1327-1337`）：
```jsonc
[{"organization_name":"某门派","power_change":-10,"new_location":null,"new_purpose":null,
  "status_description":"因内乱势力受损","key_event":"长老叛变","is_destroyed":false}]
```

**Prompt 层的关键「稀疏」纪律**（`prompt_service.py:1361-1365` 的 `<constraints>`）：
- `✅ 职业变化可选：仅当章节明确描述时填写`
- `✅ 存活状态谨慎：survival_status 仅当章节有明确死亡/失踪/退场描写时填写，默认 null`
- `✅ 组织覆灭谨慎：is_destroyed 仅当组织被彻底消灭时设 true，组织受损不算覆灭`
- `❌ 无根据地添加职业变化` / `❌ 无根据地添加组织变化`

→ **设计精髓：让 LLM「默认什么都不填」**。分析结果天然稀疏，后端只需处理差分，无需 diff 全量状态。

### 3.3 主循环算法（逐步）

`clones/MuMuAINovel@600be703:backend/app/services/character_state_update_service.py:32-185`

```
update_from_analysis(db, project_id, character_states, chapter_id, chapter_number):

  if not character_states: return 零计数结果                        # :53-61  空数组短路

  # ── 预加载（避免 N+1）────────────────────────────────────────
  all_characters = SELECT Character WHERE project_id = ?            # :74-77
  characters_by_name = {c.name: c for c in all_characters
                        if not c.is_organization}                   # :80-82   ★ 组织不进角色名索引
  all_orgs = SELECT Organization WHERE project_id = ?               # :85-88
  char_id_to_name = {c.id: c.name for c in all_characters}          # :91
  org_by_name = {char_id_to_name[org.character_id]: org}            # :94-98   ★ 组织按 Character.name 索引

  for char_state in character_states:                               # :100
      name = char_state.get('character_name')
      if not name: continue                                         # :101-103
      character = characters_by_name.get(name)
      if not character: WARN "角色不存在，跳过"; continue             # :105-108  ★ 未知角色静默丢弃

      # ── 0. 存活状态（短路优先）────────────────────────────────
      survival = char_state.get('survival_status')
      if survival in ('deceased','missing','retired'):              # :112
          await _update_survival_status(...)                        # :113-121  级联（见 §3.4）
          state_updated_count += 1
          continue                       # ★★ 死亡/失踪后不再更新心理状态、关系、组织  # :124

      # ── 1. 心理状态 ──────────────────────────────────────────
      if await _update_psychological_state(...): state_updated_count += 1   # :127-134

      # ── 2. 关系 ─────────────────────────────────────────────
      rc = char_state.get('relationship_changes', {})
      if rc and isinstance(rc, dict):                               # :137-138
          created, updated = await _update_relationships(...)       # :139-148
          relationship_created_count += created; relationship_updated_count += updated

      # ── 3. 组织成员 ──────────────────────────────────────────
      oc = char_state.get('organization_changes', [])
      if oc and isinstance(oc, list):                               # :153-154
          org_updated_count += await _update_organization_memberships(...)  # :155-164

  total = state + rel_created + rel_updated + org                   # :167-172
  if total > 0: await db.commit()                                   # :173-174  ★ 单次提交
```

**算法特征小结**

| 特征 | 实现 | 行 |
|---|---|---|
| 稀疏差分 | 只处理 AI 显式给出的字段，未提及即不动 | 全循环 |
| 空输入短路 | `if not character_states` 直接返回，不查库 | :53-61 |
| 名称精确匹配 | `characters_by_name` 精确 dict 查找（**非模糊/别名**） | :80-82, :105 |
| 未知名容错 | WARN + `continue`，不抛异常 | :107-108 |
| 存活状态短路 | `deceased/missing/retired` 一旦命中直接 `continue` | :124 |
| 逐条 try/except | 关系项、组织项各自包 try，单条失败不影响其余 | :451-454, :643-646 |
| 单事务提交 | 循环结束后一次性 commit | :174 |

### 3.4 存活状态级联（最值得蒸馏的部分）

`clones/MuMuAINovel@600be703:backend/app/services/character_state_update_service.py:187-267`

```
_update_survival_status(character, new_status, chapter_number, key_event):

  STATUS_DESC = {deceased:'死亡', missing:'失踪', retired:'退场'}            # :205-209

  # ★ 单调守卫（防低章节覆盖高章节）
  if character.status_changed_chapter is not None
     and chapter_number < character.status_changed_chapter:
       INFO "状态已在第N章变更，跳过"; return                                # :214-217

  old = character.status or 'active'                                        # :219
  character.status = new_status                                             # :220
  character.status_changed_chapter = chapter_number                         # :221
  character.current_state = f"{status_desc}（第{chapter_number}章）"          # :222  ★ 状态文本化
  character.state_updated_chapter = chapter_number                          # :223  ★ 同步推高心理水位

  changes.append(f"💀 {name} {desc}：{key_event[:50]}")                       # :225-226

  # ── 级联 1：所有 active 关系 → past，ended_at = "第N章" ──────────────────
  active_rels = SELECT CharacterRelationship WHERE project_id = ?
                AND status = 'active'
                AND (from_id = char.id OR to_id = char.id)                  # :230-241
  for rel in active_rels:
      rel.status = 'past'; rel.ended_at = f"第{chapter_number}章"             # :243-245

  # ── 级联 2：所有 active 组织成员身份 → deceased/retired ────────────────────
  member_status = 'deceased' if new_status == 'deceased' else 'retired'     # :250
  active_members = SELECT OrganizationMember WHERE character_id = char.id
                   AND status = 'active'                                    # :251-258
  for m in active_members:
      m.status = member_status
      m.left_at = f"第{chapter_number}章"                                     # :261-262
      m.notes = f"{m.notes or ''}\n[第{chapter_number}章] 角色{status_desc}".strip()  # :263-265
```

**级联三件套（可复用的通用模式）**：
1. **水位单调守卫**：`chapter_number < 已记录水位 → 跳过`（幂等，重复分析低章节不破坏数据）
2. **关系终态化**：`active → past` + `ended_at` 落章号（**保留行，不删除**，历史可回溯）
3. **成员身份终态化**：`active → deceased/retired` + `left_at` + **notes 追加时间线注释**（`\n[第N章] 原因`）

> ⚠️ 级联 2 的查询**没有 `project_id` 过滤**（`:251-258` 只有 `character_id` + `status`）。
> 因 `character_id` 是全局 UUID 主键，实际不会跨项目污染，但这是与级联 1 不一致的写法。记为 §8 坑 6。

### 3.5 心理状态更新

`clones/MuMuAINovel@600be703:backend/app/services/character_state_update_service.py:269-314`

```
_update_psychological_state(character, char_state, chapter_number):
  state_after = char_state.get('state_after')
  if not state_after: return False                              # :288-290  ★ 无 state_after 即不更新

  # ★ 单调守卫（与存活状态独立水位！）
  if character.state_updated_chapter is not None
     and chapter_number < character.state_updated_chapter:
       INFO "心理状态已被第N章更新，跳过"; return False            # :293-299

  character.current_state = state_after                         # :302  ★ 覆盖式，无历史栈
  character.state_updated_chapter = chapter_number              # :303

  changes.append(f"👤 {name} 心理状态: {state_before} → {state_after} ({psych_change[:50]})")  # :308-311
```

**要点**：`current_state` 是 **last-write-wins 覆盖**，`state_before` 只进日志、**不入库**。
→ 状态迁移历史**只能从 `changes` 日志与 `StoryMemory` 间接重建**，DB 里没有可直接查询的状态时间线。
这是本能力域最重要的**结构性缺口**（见 §8 坑 1 与 §7 的 gaea 改进设计）。

### 3.6 关系更新 + 亲密度关键词算法

`clones/MuMuAINovel@600be703:backend/app/services/character_state_update_service.py:316-456`

**输入形态兼容**（`:349-357`）：`{角色名: "变化描述"}` 或 `{角色名: {"change": "...", ...}}`。

**已存在判定是双向的**（`:373-390`）：
```python
or_( and_(from==A, to==B), and_(from==B, to==A) )   # :377-386
existing_rel = result.scalar_one_or_none()          # :390  ★ 无向语义
```
→ **A→B 与 B→A 视为同一条关系**，不创建反向边。图谱层是「无向边 + 箭头仅表方向」。

**分支 1 · 已存在关系（:395-421）**：
```python
existing_rel.relationship_name = change_desc          # :398  ★ 关系名 = 最新变化描述（AI 覆盖）
chapter_note = f"[第{chapter_number}章] {change_desc}" # :401
existing_rel.description = (desc + "\n" + note) if desc else note  # :402-405  ★ 变更历史追加
if intimacy_delta != 0:
    new = max(-100, min(100, (existing_rel.intimacy_level or 0) + delta))  # :410
```
**分支 2 · 新建关系（:423-449）**：
```python
initial_intimacy = max(-100, min(100, 50 + intimacy_delta))            # :426  ★ 基线 50
CharacterRelationship(
    relationship_type_id=None,           # :433  ★ 不强制关联预定义类型
    relationship_name=change_desc,       # :434  ★ 直接用 AI 描述当关系名
    intimacy_level=initial_intimacy, status="active",
    description=f"[第{chapter_number}章] {change_desc}",  source="analysis")  # :435-438
```

**亲密度算法**（`:807-829` + 词典 `:13-26`）：
```python
INTIMACY_ADJUSTMENTS = {
  "改善":+10,"加深":+15,"信任":+10,"亲近":+15,"友好":+10,"认可":+10,"合作":+5,"和解":+20,
  "喜欢":+15,"爱":+20,"尊敬":+10,"感激":+10,"好转":+10,"增进":+10,"亲密":+15,"忠诚":+10,
  "恶化":-10,"疏远":-15,"背叛":-30,"敌对":-25,"矛盾":-10,"冲突":-15,"怀疑":-10,"不信任":-15,
  "厌恶":-20,"仇恨":-25,"决裂":-30,"猜忌":-10,"紧张":-5,"破裂":-25,"反目":-25,"嫉妒":-10,
  "初识":0,"相遇":0,"结盟":+10,"分离":-5,
}                                                                      # :13-26

def _calculate_intimacy_delta(change_desc):
    delta = 0; matched = False
    for kw, adj in INTIMACY_ADJUSTMENTS.items():
        if kw in change_desc: delta += adj; matched = True                # :820-823  ★ 子串包含，全部累加
    if matched: delta = max(-30, min(30, delta))                          # :826-827  ★ 单次钳制 ±30
    return delta
```

> ⚠️ **子串叠加缺陷（实测确认）**：`"不信任"` 同时包含 `"信任"(+10)` 与 `"不信任"(-15)`，净 **-5** 而非 -15；
> `"不信任"` 还包含 `"信"`？——不，词典无单字 `"信"`。但 `"不信任"` ⊃ `"信任"` 成立。
> 同理 `"背叛"` 含 `"叛"`？词典无单字项，安全；`"猜忌"` 含 `"怀疑"`？——否（字符不同）。
> 已验证的确定性叠加：**`不信任` → 命中 `信任`(+10) 与 `不信任`(-15) → -5**。
> 语义明显失真（"不信任"比"怀疑"(-10) 还轻）。§7 给出 gaea 的正确算法。

**关系类型匹配**：`character_state_update_service` **完全不匹配** `relationship_type_id`（恒 `None`，:433）。
匹配只发生在**自动补角**路径（`auto_character_service.py:324-334`）：按 `relationship_type` 名字查 `RelationshipType` 表，命中则回填 `relationship_type_id`。

### 3.7 组织成员变动（6 种 change_type × 忠诚度算法）

`clones/MuMuAINovel@600be703:backend/app/services/character_state_update_service.py:458-648`

**忠诚度词典**（:486-490，定义在**函数体内**，每次调用重建）：
```python
LOYALTY_ADJUSTMENTS = {"提升":+10,"增强":+10,"坚定":+15,"忠心":+15,
                       "动摇":-15,"怀疑":-10,"不满":-10,"降低":-10,
                       "背叛":-50,"叛变":-50,"反感":-20,"失望":-15}
```
累加后 `max(-50, min(50, delta))`（:526）。

分支表：

| `change_type` | 前置条件 | 效果 | 行 |
|---|---|---|---|
| `joined` | 已有非 active 成员行 | `status='active'`、`left_at=None`、可选更新 `position`、notes 追加「重新加入」 | :530-542 |
| `joined` | 无成员行 | 新建 `OrganizationMember(position=new_position or '成员', rank=0, loyalty=clamp(50+Δ,0,100), status='active', joined_at=f"第N章", source='analysis')` **且 `organization.member_count += 1`** | :543-561 |
| `left` / `expelled` / `betrayed` | 已 active | `status = {'left':'retired','expelled':'expelled','betrayed':'expelled'}`；`left_at=f"第N章"`；loyalty += Δ；notes 追加 | :563-581 |
| `promoted` | 已是成员 | 可选改 position；`rank += 1`；**Δ==0 时默认 `loyalty += 5`**；notes 记「晋升: old → new」 | :583-600 |
| `promoted` | 非成员 | **WARN「不是成员，无法晋升」，不创建成员行** | :601-602 |
| `demoted` | 已是成员 | 可选改 position；`rank = max(0, rank-1)`；**Δ==0 时默认 `loyalty -= 5`** | :604-621 |
| 其他 | 有成员行且 Δ≠0 | 仅调 loyalty + notes | :625-641 |

> **重要不变量**：`member_count` **只在「新建成员行」时 +1**（:558），在 `left/expelled/betrayed` 时**不减**。
> 且 `_update_survival_status` 把成员置 `retired` 时（:260-265）也不改 `member_count`。
> → `member_count` 会**单调虚高**。MuMu 用两个补丁 API 兜底：`chapters.py` 外的 `POST /projects/{id}/fix-member-counts`（`backend/app/api/projects.py:543`）与 `fix-organizations`（`:494`）。这是**已知技术债**，gaea 不应复制（见 §7）。

### 3.8 组织自身状态更新

`clones/MuMuAINovel@600be703:backend/app/services/character_state_update_service.py:650-805`

```
update_organization_states(db, project_id, organization_states, chapter_number):
  # 预加载：is_organization=True 的 Character + 其 Organization
  org_chars = SELECT Character WHERE project_id=? AND is_organization=True   # :677-683
  org_char_by_name = {c.name: c}                                            # :684
  org_by_char_id = {org.character_id: org}                                  # :692-696

  for org_state in organization_states:
      org_name → org_char → organization        （任一缺失则 WARN + continue）  # :704-712

      if org_state.get('is_destroyed'):                                   # :718-719
          org_char.status = 'destroyed'                                   # :721
          org_char.status_changed_chapter = chapter_number                 # :722
          org_char.current_state = f"覆灭（第{chapter_number}章）"          # :723
          org_char.state_updated_chapter = chapter_number                  # :724
          organization.power_level = 0                                     # :725
          all active OrganizationMember → status='retired', left_at, notes  # :727-742
          continue                                                         # :750  ★ 覆灭后不再更新其他属性

      power_change = org_state.get('power_change', 0)                       # :753
      if isinstance(power_change,(int,float)):
          new_power = max(0, min(100, (organization.power_level or 50) + int(power_change)))  # :755-756
      new_location → organization.location                                  # :763-768
      new_purpose  → org_char.organization_purpose                          # :771-776
      status_description → org_char.current_state + state_updated_chapter   # :779-785
  if updated_count > 0: commit                                              # :801-802
```

> ⚠️ `update_organization_states` **没有章节号单调守卫**（对比 `_update_survival_status` :214 与 `_update_psychological_state` :293 都有）。
> 重跑第 3 章分析会把第 20 章覆灭的组织"复活"成非 destroyed 状态（因为 `is_destroyed=False` 时走 :752 起的属性分支，不重置 status，但 `status_description` 会覆盖 `current_state`）。
> 记为 §8 坑 2。

### 3.9 状态回灌：让状态机闭环

状态机的一半价值在**写回生成侧**。MuMu 有两条回灌通道：

**通道 1 · 章节分析 Prompt 的角色上下文**（`clones/MuMuAINovel@600be703:backend/app/api/chapters.py:577-838`）
`build_characters_info_with_careers()` 拼一行式角色摘要，字段顺序固定（:835）：
```
- 名字(角色/组织, role_type)[💀已死亡]
  | 类型:.., 宗旨:.., 势力等级:.., 据点:.., 口号:.., 成员数:.. | 成员: A(长老), B(弟子)[retired]
  | 主职业: 剑修(3/10阶) | 副职业: 炼丹师(2/5阶)
  | 当前状态: 坚定(第12章)
  | 所属组织: 青云宗(内门弟子)[忠诚度:72]
  | 关系: 李四(关系改善)[亲密度:65], 王五(宿敌)[亲密度:20]
  : 性格前100字
```
关键点：
- 存活状态用 emoji 标记（:718-724）：`deceased→💀已死亡, missing→❓已失踪, retired→📤已退场, destroyed→💀已覆灭`
- **只显示非默认值**：`intimacy_level != 50` 才显示（:821）、`loyalty != 50` 才显示（:794）——**降噪**
- 截断策略：current_state 50 字（:782）、personality 100 字（:831）、purpose 60 字（:735）、关系取前 5（:807）、组织取前 3（:791）

**通道 2 · 章节正文生成 Prompt**（`clones/MuMuAINovel@600be703:backend/app/services/chapter_context_service.py:31-45, 533-666`）
多行结构化格式，与通道 1 不同风格：
```
【张三】(角色, 主角)
  年龄: 25
  性别: 男
  外貌: ...
  性格: ...
  背景: ...
  当前状态: 存活（第3章变更）          # ← _append_current_state_lines :33-38
  当前心理/处境: 坚定（第12章更新）      # ← :40-45，截断 150 字
  主职业: 剑修 (3/10阶 - 金丹期)        # ← :578
  副职业: 炼丹师 (2/5阶 - 中品)         # ← :592
  关系网络: 与李四：关系改善；与王五：宿敌   # ← :611
  组织归属: 青云宗（内门弟子）、丹盟（客卿）  # ← :618
```
并且 `chapter_context_service.py:637-664` 额外拼一个**独立职业详情块**（阶段名+描述全展开），
通过 `context.chapter_careers` 注入 `chapter_careers` 模板变量（`chapters.py:1657` 等 20+ 处）。

> **两条通道格式不一致是历史债**（一行式 vs 多行式）。gaea 应统一为一种（§7 选多行式，信息密度高且易截断）。

---

## 4. 职业/境界体系：完整建模与更新路径

### 4.1 职业体系生成（项目级）

**Prompt 构造** `clones/MuMuAINovel@600be703:backend/app/services/career_service.py:18-107` 与增量版 `backend/app/api/careers.py:159-456`。

初始版输出契约（`career_service.py:60-92`）：
```jsonc
{"main_careers":[{"name","description","category","stages":[{"level":1,"name","description"}],
                  "max_stage":10,"requirements","special_abilities","worldview_rules",
                  "attribute_bonuses":{"strength":"+10%"}}],
 "sub_careers":[{"...","stages":"5-8个","max_stage":5}]}
```
**类型自适应指引**（`career_service.py:96-101`）——蒸馏到 gaea 时可直接复用：
```
修仙类：剑修、体修、法修、符修，阶段如 炼气、筑基、金丹、元婴...
玄幻类：战士、法师、刺客，阶段如 见习、初级、中级、高级...
都市异能：异能者分类，阶段如 觉醒、初阶、中阶、高阶...
科幻未来：基因战士、机甲师，阶段如 E级、D级、C级、B级...
```

**增量生成（关键设计）** `backend/app/api/careers.py:185-252`：
- 先查已有职业，拼 `已有主职业（N个）/已有副职业（M个）` 摘要（:196-212）
- 若为空则注入 `当前还没有任何职业，这是第一次创建职业体系。`（:211-212）
- 生成要求显式写：`⚠️ 重要：请生成与已有职业**不重复**的新职业，形成互补体系`（:247）
- 支持 `user_requirements` 用户额外要求，并给出**冲突降级策略**（:236-238）：
  `如果用户要求与项目世界观冲突，请在不违背世界观的前提下进行合理改写和本地化适配`
- 默认数量：初始 主2/副6（`career_service.py:21-22`）；增量 API 默认 主5/副8（`schemas/career.py:79-80`），前端默认 主3/副5（`Careers.tsx:459,462`）

**落库** `backend/app/services/career_service.py:110-191`：`stages` 与 `attribute_bonuses` 各自 `json.dumps(..., ensure_ascii=False)` 存 TEXT（:134-136），逐条 try/except（:156-158），最后一次性 commit（:189）。

> ⚠️ **两条生成路径代码重复**：`career_service.parse_and_save_careers`（:110-191）与 `api/careers.py:367-429` 内联实现逻辑几乎逐行相同。
> 蒸馏时**只保留一份实现**。

### 4.2 角色获得职业（三条路径）

**路径 1 · AI 生成角色时一起给职业**（`clones/MuMuAINovel@600be703:backend/app/services/auto_character_service.py:145-250`）

```
career_info = character_data.get("career_info", {})               # :146
raw_main = career_info.get("main_career_name")                    # :147
stage    = career_info.get("main_career_stage", 1)                # :148
raw_subs = career_info.get("sub_careers", [])                     # :149

# ★ 名称 → ID 匹配（AI 只输出名称，不输出 ID）
if raw_main and not is_organization:                              # :157
    matched = SELECT Career WHERE name=raw_main AND project_id=? AND type='main'   # :158-164
    if matched:
        main_career_id = matched.id                               # :167
        if stage > matched.max_stage:                             # :169  ★★ 阶段上限钳制
            WARN "AI返回的主职业阶段(N)超过最高阶段(M)，自动修正为最高阶段"
            stage = matched.max_stage                             # :171
for sub_data in raw_subs[:2]:                                     # :178  ★★ 硬截断 2 个副职业
    matched = SELECT Career WHERE name=? AND project_id=? AND type='sub'  # :182-188
    if sub_stage > matched.max_stage: sub_stage = matched.max_stage       # :193-195
    sub_careers_data.append({'career_id': matched.id, 'stage': sub_stage})# :197-200

Character(..., main_career_id=..., main_career_stage=...,
          sub_careers=json.dumps(sub_careers_data, ensure_ascii=False))   # :219-221
# 同时建 CharacterCareer 行（main current_stage=stage / sub current_stage=stage, stage_progress=0） # :228-250
```
`characters.py:1017-1140` 有一个**结构完全相同、行数更多的内联副本**（带 6 条 `logger.info` 调试输出），实现同一个 `generate-stream` 流程。

**路径 2 · 手填角色表单**（`clones/MuMuAINovel@600be703:backend/app/api/characters.py:379-509`）
- `main_career_id` 变更 → 校验 `Career.type=='main'` 且 `stage <= max_stage`（:387-401），删旧建新（:404-432）
- `main_career_id = None` → 清空（:446-447）
- 仅改 `main_career_stage` → 同步 CharacterCareer（:448-460）
- `sub_careers` **硬截断 `[:2]`**（:481）并在 :508-509 回写冗余字段

**路径 3 · 手动 API**（`clones/MuMuAINovel@600be703:backend/app/api/careers.py`）
- `POST /careers/character/{id}/careers/main`（:682-754）：换主职业 = 删旧 + 建新（:734-749），`reached_current_stage_at = started_at`
- `POST /careers/character/{id}/careers/sub`（:757-837）：**上限 5**（:819-820 `if sub_count >= 5`）
- `PUT /careers/character/{id}/careers/{career_id}/stage`（:840-897）：更新阶段，**降级只 WARN 不阻止**（:878-879）
- `DELETE /careers/character/{id}/careers/{career_id}`（:900-937）：**主职业不允许删**（:929-930）
- 读取时通过 `careers.py:642-669` 把 stage 数字解析成阶段名

> ⚠️ **副职业上限三处不一致**：AI 路径 `[:2]`（`auto_character_service.py:178`、`characters.py:481`、`characters.py:708`、`characters.py:1050`、`characters.py:1132`）、
> `career_update_service._add_new_career` 上限 **2**（`career_update_service.py:359-361`）、
> 手动 API 上限 **5**（`careers.py:819`）。
> → 手动能给 5 个，AI 只认前 2 个，分析新加的第 3+ 个会被上限拦住。**gaea 必须统一为单一常量**。记为 §8 坑 3。

### 4.3 职业随章节推进（±N 阶）

`clones/MuMuAINovel@600be703:backend/app/services/career_update_service.py`

**主循环**（:15-127）：
```
for char_state in character_states:                              # :45
    career_changes = char_state.get('career_changes', {})        # :47
    if not dict: continue                                        # :50
    main_delta      = career_changes.get('main_career_stage_change', 0)   # :54
    sub_changes     = career_changes.get('sub_career_changes', [])        # :55
    new_careers     = career_changes.get('new_careers', [])               # :56
    if main_delta == 0 and not sub_changes and not new_careers: continue  # :58-59  ★ 三个都是 0 才跳过
    character = SELECT Character WHERE name=? AND project_id=?            # :64-70
    if not character: WARN; continue                                      # :72-74

    if main_delta != 0 and character.main_career_id:                      # :77  ★★ 双条件
        if await _update_main_career_stage(...): updated_count += 1       # :78-87
    for sub_change in sub_changes:                                        # :90-101
        if await _update_sub_career_stage(...): updated_count += 1
    for name in new_careers:                                              # :104-115
        if await _add_new_career(...): updated_count += 1
if updated_count > 0: commit                                              # :118-119
```

**主职业阶段推进**（:129-204）——**本能力域最核心的钳制逻辑**：
```python
char_career = SELECT CharacterCareer WHERE character_id=? AND career_type='main'   # :141-147
career      = SELECT Career WHERE id = char_career.career_id                       # :154-157
old_stage = char_career.current_stage
new_stage = min(max(1, old_stage + stage_change), career.max_stage)                 # :165  ★★ 双端钳制
if new_stage == old_stage:
    INFO "已达到边界，无法变更"; return False                                        # :168-170
char_career.current_stage = new_stage                                              # :173
character.main_career_stage = new_stage        # ★ 同步冗余快照                    # :176
changes_log.append({'character','career','career_type':'main',
                    'old_stage','new_stage','change':stage_change,
                    'chapter','description':career_changes.get('career_breakthrough','')})  # :182-191
```

**副职业阶段推进**（:206-290）：
```python
career = SELECT Career WHERE name=? AND project_id=? AND type='sub'      # :224-230  ★ 按名字+项目+类型
char_career = SELECT CharacterCareer WHERE character_id=? AND career_id=? AND career_type='sub'  # :238-245
new_stage = min(max(1, old_stage + stage_change), career.max_stage)      # :253
char_career.current_stage = new_stage                                    # :259
# ★ 同步 Character.sub_careers JSON（遍历找 career_id 匹配项改 stage）      # :262-268
sub_careers = json.loads(character.sub_careers) if character.sub_careers else []
for sc in sub_careers:
    if sc.get('career_id') == career.id: sc['stage'] = new_stage; break
character.sub_careers = json.dumps(sub_careers, ensure_ascii=False)
```

**新职业获得**（:292-398）：
```python
career = SELECT Career WHERE name=? AND project_id=?                     # :304-310  ★ 无 type 限制
if career.type == 'main':                                                # :328
    if character.main_career_id: WARN "已有主职业，无法添加"; return False  # :330-332
    CharacterCareer(career_type='main', current_stage=1)                  # :336-343
    character.main_career_id = career.id; character.main_career_stage = 1 # :346-347
else:  # sub
    if len(SELECT CharacterCareer WHERE character_id=? AND career_type='sub') >= 2:  # :353-359
        WARN "副职业已达上限(2个)"; return False
    CharacterCareer(career_type='sub', current_stage=1)                   # :365-372
    sub_careers.append({'career_id': career.id, 'stage': 1})              # :377-380
```

**核心不变量（蒸馏重点）**：

| 不变量 | 实现 | 行 |
|---|---|---|
| 阶段 ∈ [1, max_stage] | `min(max(1, old+Δ), max_stage)` | `career_update_service.py:165`, `:253` |
| 主职业唯一 | `character.main_career_id` 非空即拒绝 | `:330-332` |
| 副职业上限 | 计数 ≥ 2 拒绝 | `:359-361` |
| 新职业必须已存在 | 名字查不到 → WARN + False | `:312-314` |
| 无变化不写库 | `new_stage == old_stage` → return False | `:168-170`, `:255-256` |
| 冗余字段同步 | CharacterCareer 与 Character.{main_career_stage,sub_careers} 双写 | `:176`, `:268` |
| **无章节号水位** | — | ⚠️ 见下 |

> ⚠️ **职业推进没有章节号单调守卫**。重跑第 3 章分析，`main_career_stage_change=+1` 会在当前（可能已是第 50 章的）阶段上**再 +1**。
> 这是本能力域**最严重的正确性缺陷**（对比 `character_state_update_service.py:214/:293` 都有守卫）。§7 必须修。

### 4.4 自动补角/补组织（大纲驱动的前置补齐）

`clones/MuMuAINovel@600be703:backend/app/services/auto_character_service.py:351-546`

```
check_and_create_missing_characters(project_id, outline_data_list, ...):
  # 1. 从所有大纲的 structure.characters 提取名字（兼容两种格式）      # :381-411
  #    新格式 {"name":"x","type":"character|organization"} → 跳过 organization
  #    旧格式 "纯字符串"
  #    同时收集上下文 character_context[name] += 《标题》: summary[:200]   # :409-411
  # 2. 查项目现有角色                                                   # :424-428
  # 3. missing_names = all_names - existing_names                       # :431
  # 4. 对每个缺失角色：spec = {name, role_description: 出现场景, suggested_role_type:'supporting',
  #                          importance:'medium'}                       # :470-475
  #    character_data = _generate_character_details(spec, ...)          # :480-487
  #    character_data['name'] = char_name   # ★ 强制用大纲里的名字       # :490
  #    character = _create_character_record(...)                        # :498-502
  #    _create_relationships(...)                                       # :515-521
  # 6. db.flush()（由调用方 commit）                                     # :537-538
```

组织补全 `auto_organization_service.py:114-155`：组织 = `Character(is_organization=True)` + `Organization` 两行同建（:123-151），
默认 `power_level=50, member_count=0`（:143-144）。

**Prompt：`AUTO_CHARACTER_GENERATION`**（`clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1926-2057`）——蒸馏要点：
- `<requirements>` 显式要求「**必须分析新角色与已有角色的关系**，至少建立 1-3 个有意义的关系」（:1967）
- 职业选择纪律「如果【已有角色】部分包含"可用主职业列表"或"可用副职业列表"…**必须填写职业的名称而非 ID**」（:1981-1987）
- 输出 schema 含 `relationships[]` / `organization_memberships[]` / `career_info{}` 三段（:2005-2032）
- 数值域显式声明（:2038-2041）：`intimacy_level：-100到100`、`loyalty：0到100`、`rank：0到10`
- 禁止项「❌ 引用不存在的角色或组织」「❌ 使用职业ID而非职业名称」（:2055-2056）

**可用职业列表注入**（`auto_character_service.py:53-84`）：
```python
careers = SELECT Career WHERE project_id=? ORDER BY type, name              # :55-59
careers_info += "\n\n可用主职业列表（请在career_info中填写职业名称和阶段）：\n"   # :69
for career in main_careers:
    careers_info += f"- 名称: {career.name}, 最高阶段: {career.max_stage}阶"  # :71
    if career.description: careers_info += f", 描述: {career.description[:50]}"  # :73
for career in sub_careers[:5]:                                              # :78  ★ 副职业只列前5个
careers_info += "\n⚠️ 重要提示：生成角色时，职业阶段不能超过该职业的最高阶段！\n"  # :84
```
这段 `careers_info` 被**拼接进 `existing_characters` 槽位**（:104 `existing_characters=existing_chars_summary + careers_info`）——
一个可复用的省钱技巧：不给新模板变量，复用已有槽位。

---

## 5. Prompt 工程蒸馏（可直接搬进 gaea 的 prompts/*.json）

### 5.1 `PLOT_ANALYSIS` 角色维度原文要点

`clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py`

**输入槽位**（:1079-1084）：
```
<characters priority="P1">
【项目角色信息 - 用于角色状态分析】
以下是项目中已有的角色列表，分析 character_states 和 relationship_changes 时请使用这些角色的准确名称：
{characters_info}
</characters>
```
→ **"请使用这些角色的准确名称"** 是本设计能跑通的前提：后端用精确 dict 匹配，靠 Prompt 保证名称不漂移。

**角色维度指令原文**（:1139-1159）：
```
**5. 角色状态追踪 (Character Development)**
对每个出场角色分析：
- 心理状态变化(前→后)
- 关系变化
- 关键行动和决策
- 成长或退步
- **💀 存活状态（重要）**：
  - survival_status: 角色当前存活状态
  - 可选值：active(正常)/deceased(死亡)/missing(失踪)/retired(退场)
  - 默认为null（表示无变化），仅当章节中角色明确死亡、失踪或永久退场时才填写
  - 死亡/失踪需要有明确的剧情依据，不可臆测
- ** 职业变化（可选）**：
  - 仅当章节明确描述职业进展时填写
  - main_career_stage_change: 整数(+1晋升/-1退步/0无变化)
  - sub_career_changes: 副职业变化数组
  - new_careers: 新获得职业
  - career_breakthrough: 突破过程描述
- **🏛️ 组织变化（可选）**：
  - 仅当章节明确描述角色与组织关系变化时填写
  - organization_changes: 组织变动数组
  - 每项包含：organization_name、change_type(加入joined/离开left/晋升promoted/降级demoted/开除expelled/叛变betrayed)、
    new_position(新职位，可选)、loyalty_change(忠诚度变化描述，可选)、description(变化描述)
```

**组织维度指令**（:1161-1171 + :1363-1365）：`5b. 组织状态追踪 - 可选`，字段 `power_change(R)`/`new_location`/`new_purpose`/`status_description`/`key_event`/`is_destroyed`。

**枚举值必须显式列举**——这是 MuMu Prompt 的一致风格（`change_type` 6 值、`survival_status` 4 值都在 Prompt 里写死），
因为后端是硬编码分支匹配，LLM 一旦自由发挥就会落到 `else` 分支被静默忽略（`character_state_update_service.py:625-641`）。

### 5.2 蒸馏出的 gaea Prompt 片段（建议直接使用）

`prompts/analysis-chapter.json` 的 `character_states` 输出契约应升级为：

```jsonc
"character_states": [{
  "name": "角色名（必须与现有角色列表逐字一致）",
  "survival_status": null,            // null|deceased|missing|retired，默认 null
  "old_state": "旧心理状态",           // 可留空
  "new_state": "新心理状态",           // 留空表示本章无心理变化
  "state_reason": "触发事件（≤50字）",
  "relationships": [{"target": "对方名", "change": "关系变化描述", "delta_hint": -30..30}],
  "career": {"main_delta": 0, "sub_deltas": [{"name":"职业","delta":1}],
             "new_careers": [], "breakthrough": ""},
  "organizations": [{"name":"组织","change_type":"joined|left|expelled|betrayed|promoted|demoted",
                     "new_position":"", "loyalty_hint": 0, "description":""}]
}],
"organization_states": [{"name":"","power_delta":0,"new_location":null,"new_purpose":null,
                          "status_description":"","key_event":"","is_destroyed":false}]
```
（`delta_hint`/`loyalty_hint` 为 gaea 新增：让 LLM 直接给数值，不再靠关键词词典猜。见 §7）

---

## 6. 关系图谱与前端建模

### 6.1 后端图谱 API

`clones/MuMuAINovel@600be703:backend/app/api/relationships.py:77-157`

```
GET /relationships/graph/{project_id} →
  nodes = [RelationshipGraphNode(id, name, type='organization'|'character', role_type, avatar)
           for c in SELECT Character WHERE project_id=?]                    # :100-109
  links = [RelationshipGraphLink(source=from_id, target=to_id,
                                 relationship=relationship_name or "未知关系",
                                 intimacy=intimacy_level, status=status)
           for r in CharacterRelationship WHERE project_id=?]               # :119-128
  # ★ 追加组织成员边：source 用 org.character_id（保证与节点 ID 一致）
  member_links = [RelationshipGraphLink(source=org.character_id, target=member.character_id,
                                        relationship=f"组织成员·{member.position}",
                                        intimacy=member.loyalty, status=member.status)
                  for (member, org) in SELECT OrganizationMember JOIN Organization ...]  # :132-149
  links.extend(member_links)                                                # :151
```

**设计要点**：
1. **节点 = 全部 Character（含组织）**，`type` 由 `is_organization` 决定（:104）
2. **成员边的 `intimacy` 复用 `loyalty`**（:145）——一个字段承载两种语义，前端靠 relationship 前缀区分
3. **关系边无向**（source/target 只是 A→B 原始方向），前端做 pair 去重

### 6.2 前端图谱：把职业建成节点

`clones/MuMuAINovel@600be703:frontend/src/pages/RelationshipGraph.tsx`

**边分类元数据**（:105-115）——可蒸馏的分类体系：
```ts
EDGE_CATEGORY_META = {
  organization: {label:'组织成员',  order:1},
  career_main:  {label:'主职业关联',order:2},
  career_sub:   {label:'副职业关联',order:3},
  career_group: {label:'职业分类',  order:4},
  family:       {label:'亲属关系',  order:5},
  hostile:      {label:'敌对关系',  order:6},
  professional: {label:'职业关系',  order:7},
  social:       {label:'社交关系',  order:8},
  default:      {label:'其他关系',  order:99},
}
```

**三类节点 + 两个虚拟分组节点**（:100-101, :809-867）：
```
真实节点：Character（角色/组织）
职业节点：career-main-{careerId} / career-sub-{careerId}
虚拟分组：__career_group_main__ / __career_group_sub__
```
**四层边**（:869-969）：
```
orgMemberEdges:        组织 → 成员        "组织成员·{position}"  layoutWeight=8  （最优先，先稳定层级）
careerGroupEdges:      分组 → 职业        "职业分类·主职业/副职业" dashed, weight=4
careerToCharacterEdges:职业 → 角色        "主职业·{name}"/"副职业·{name}"  weight=3/2
memberRelationEdges:   角色 ↔ 角色        relationship_name       weight=1  （最后）
```
**布局权重决定层次**（:971-974）：`layoutEdges = org + careerGroup + careerToCharacter`，
用 dagre 对这些高权边做力导向定向，**人际关系边只参与渲染不参与布局**。
兜底：若无组织/职业边，则退回用关系边布局（:972）。

**职业边样式**（:678-723）：
```ts
isCareerMainLink → opacity 0.6, strokeWidth 2, labelBg 实底, category 'career_main'
isCareerSubLink  → 虚线 '6 3', opacity 0.6, category 'career_sub'
isCareerClassLink→ strokeWidth 1.5, opacity 0.5, category 'career_group'
isOrgMemberLink  → 虚线, category 'organization'
```
**职业属性读取**（:917-956）：遍历**非组织**角色，`character.main_career_id` → `career-main-{id}` 边；
`safeParseSubCareers(character.sub_careers)` → `career-sub-{career_id}` 边（:389-408 有容错解析）。
**详情面板**（:1093）：`第{nodeDetail.main_career_stage}阶`。

### 6.3 职业管理页（`Careers.tsx`）

- 主/副职业双 Tab（:301-320），卡片展示「阶段体系（共N个）」前 5 个阶段（:281-289）
- 阶段编辑用**文本行格式**：`1. 炼气期 - 初窥门径`（:415-420 提示词），提交时正则解析（:113-129）：
  `/^(\d+)\.\s*([^-]+)(?:\s*-\s*(.*))?$/`，`max_stage = stages.length`（:134）
- AI 增量生成走 SSE fetch + `data: ` 行解析（:190-253），进度条 `SSEProgressModal`
- 事件总线：`BACKGROUND_TASK_SETTLED` + `resources.includes('careers')` → 自动刷新（:71-82）
- 表单字段：name/type/description/category/stages/requirements/special_abilities/worldview_rules（:390-443）

### 6.4 ⚠️ 前端关键空档：角色-职业绑定 UI 未接线

后端 `careers.py:608-937` 提供了完整的「角色-职业」接口族：
`GET/POST /character/{id}/careers/main`、`POST .../sub`、`PUT .../{career_id}/stage`、`DELETE .../{career_id}`。
唯一消费方是 `clones/MuMuAINovel@600be703:frontend/src/components/CharacterCareerCard.tsx`
（`CharacterCareerCard.tsx:67` 读取、`:104` 设主职业、`:122` 加副职业、`:142` 调阶、`:165` 删副职业，
含 :385「阶段进度（0-100）」表单与 :223 `Progress` 进度条）。

但 grep `CharacterCareerCard` 在 `frontend/src` 内**只命中该文件自身**（`:41` 定义、`:407` 导出），
**没有任何页面 import 它**；`frontend/src/pages/Careers.tsx` 只调用项目级 `/careers` 增删改查（:53,138,141,165,190）与
`/careers/generate-system`（:190），**完全没有角色-职业绑定入口**。
`Characters.tsx:1054,1333` 虽有「当前阶段 `main_career_stage`」表单项，但只走 `PUT /characters/{id}` 的冗余字段路径。

→ **结论：MuMu 的角色-职业绑定能力后端完备、前端断线**。gaea 实施时这不是"照抄现有页面"，
而是**把断掉的 UI 补起来**（本报告 §7.6 已按此调整）。

### 6.5 gaea 现状对照

`C:\AI\wubigrok\internal\graph\`（EntityDB）已经把角色当实体，`internal/novelcontext/novelcontext.go` 把
`status` 从实体属性里读出（:438 `"status": c.Status`、:453 `sc.Status = e.Properties["status"]`、:463-464 fallback 到 `types.Character.Status`），
并在 `Render()`（:136-138）与 `formatSceneChar`（:782-790 `if c.Status != "" { bits = append(bits, "状态: "+c.Status) }`）注入生成 Prompt。

→ **gaea 已有「实体属性 → Prompt」的现成通道**，只需把 `current_state`/`career` 也写进 `Entity.Properties` 即可零成本回灌。
`frontend/src/components/RelationGraph.tsx:19-20, 121-134` 是简易关系图（无职业节点、无组织成员边、无分类筛选）；
`frontend/src/pages/CharacterPage.tsx:84,407,437,703-733` 有角色/组织/关系三 Tab + `RelationGraph`。

---

## 7. gaea 落地实施规格

### 7.1 数据模型（`internal/types/types.go` 扩展）

```go
// Character 新增两态水位（与 MuMu characters 表对齐）
type Character struct {
    ... // 既有字段不动
    Status              string `json:"status"`                         // 既有: Alive/Dead/Missing/Transformed
    StatusChangedChapter int   `json:"status_changed_chapter,omitempty"` // 新增
    CurrentState         string `json:"current_state,omitempty"`         // 新增：心理/处境
    StateUpdatedChapter  int    `json:"state_updated_chapter,omitempty"` // 新增
    MainCareerID         string `json:"main_career_id,omitempty"`        // 新增
    MainCareerStage      int    `json:"main_career_stage,omitempty"`     // 新增
    SubCareers           []CharacterCareerRef `json:"sub_careers,omitempty"` // 新增
}

type CharacterCareerRef struct {
    CareerID string `json:"career_id"`
    Stage    int    `json:"stage"`
}
```

```go
// Organization 扩展（保留 PowerLevel string 做兼容，新增数值域）
type Organization struct {
    ID          string
    Name        string
    Type        string   `json:"type,omitempty"`
    Description string   `json:"description,omitempty"`
    PowerLevel  string   `json:"power_level,omitempty"` // 既有，保留只读兼容
    PowerValue  int      `json:"power_value,omitempty"` // 新增：0..100，分析驱动
    Location    string
    Motto       string
    Members     []string `json:"members,omitempty"`     // 既有 ID 列表，保留只读兼容
    MemberList  []OrgMember `json:"member_list,omitempty"` // 新增：完整成员关系
    Destroyed   bool     `json:"destroyed,omitempty"`
    DestroyedChapter int `json:"destroyed_chapter,omitempty"`
}

type OrgMember struct {
    CharacterID string `json:"character_id"`
    Position    string `json:"position"`
    Rank        int    `json:"rank"`
    Status      string `json:"status"`      // active/retired/expelled/deceased
    Loyalty     int    `json:"loyalty"`     // 0..100 默认 50
    JoinedAt    string `json:"joined_at,omitempty"`   // "第N章"
    LeftAt      string `json:"left_at,omitempty"`
    Notes       string `json:"notes,omitempty"`
}
```

```go
// Relationship 新增状态与时间线
type Relationship struct {
    FromID       string
    ToID         string
    RelationType string
    Description  string
    Intimacy     int
    Status       string `json:"status,omitempty"`   // 新增: active/broken/past/complicated
    StartedAt    string `json:"started_at,omitempty"`
    EndedAt      string `json:"ended_at,omitempty"`
    History      []RelChange `json:"history,omitempty"` // 新增：变更时间线
}
type RelChange struct {
    Chapter     int    `json:"chapter"`
    Description string `json:"description"`
}
```

**职业体系独立文件**（建议 `careers.json`，与 characters.json 解耦）：
```go
type Career struct {
    ID           string        `json:"id"`
    Name         string        `json:"name"`
    Type         string        `json:"type"`        // main / sub
    Category     string        `json:"category,omitempty"`
    Description  string        `json:"description,omitempty"`
    Stages       []CareerStage `json:"stages"`      // 阶段名表
    MaxStage     int           `json:"max_stage"`
    Requirements string        `json:"requirements,omitempty"`
    SpecialAbilities string    `json:"special_abilities,omitempty"`
    WorldviewRules   string    `json:"worldview_rules,omitempty"`
    Source       string        `json:"source"`      // ai / manual
}
type CareerStage struct {
    Level       int    `json:"level"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}
```
> `Career` **项目级存一份**（不随角色复制），角色只存 `{career_id, stage}` 引用——这正是 MuMu 三层模型的关键，
> 也天然匹配 gaea「角色是独立资产 + 项目关联」的 `characterlib.ProjectCharacter`（`internal/characterlib/model.go:61-68`，已有 `ArcState` 字段可承载 `current_state` 的镜像）。

### 7.2 状态差分更新器（新包 `internal/characterstate`）

```go
// Outcome 一次章节分析对角色域的净变更（用于 UI toast / 日志 / 幂等校验）
type Outcome struct {
    StateUpdated      int
    RelCreated        int
    RelUpdated        int
    OrgMemberUpdated  int
    OrgStateUpdated   int
    CareerUpdated     int
    Changes           []string
}

// ApplyChapterDiff 按 MuMu 语义把分析差分应用到角色文件。
// 三阶段严格顺序：career → character/relationship/org-member → org-self。
func ApplyChapterDiff(pm *project.Manager, chapterNum int,
                      charStates []CharStateDiff, orgStates []OrgStateDiff) (Outcome, error)
```

**必须实现的 7 条不变量**（逐条对应 MuMu 源码）：

| # | 不变量 | MuMu 出处 | gaea 要求 |
|---|---|---|---|
| 1 | 心理状态水位单调：`chapterNum < StateUpdatedChapter` → 跳过 | `character_state_update_service.py:293-299` | **实现**，返回 `SkippedStale` 计数 |
| 2 | 存活状态水位单调 | 同上 `:214-217` | **实现** |
| 3 | 存活短路：`deceased/missing/retired` 后不再改心理/关系/组织成员 | 同上 `:112-124` | **实现**，且顺序必须是 survival 优先 |
| 4 | 存活级联三件套（关系→past+EndedAt、成员→终态+LeftAt、Notes 追加） | 同上 `:229-267` | **实现** |
| 5 | 职业阶段 ∈ [1, MaxStage] 双端钳制 | `career_update_service.py:165,253` | **实现** |
| 6 | 主职业唯一 / 副职业上限（**单一常量**） | `career_update_service.py:330,359` vs `careers.py:819` **不一致** | **修正**：统一 `MaxSubCareers = 2`，前后端与 AI 路径共用同一常量 |
| 7 | **职业推进也要有水位守卫** | MuMu **缺失** | **新增**：`CharacterCareerRef.UpdatedChapter`，防重跑改阶 |

**坑 3 修复**：`career_update_service.py:359-361`（上限 2）与 `careers.py:819`（上限 5）与
`auto_character_service.py:178` / `characters.py:481,708,1050,1132`（`[:2]` 截断）四处不一致。
gaea 定 `const MaxSubCareers = 2`，**所有路径**（Prompt 声明、API 校验、AI 合并、分析新增）共用。

### 7.3 亲密度算法（替换关键词包含法）

MuMu 的 `_calculate_intimacy_delta`（`character_state_update_service.py:807-829`）有子串叠加缺陷
（`"不信任"` 命中 `"信任"+10` 与 `"不信任"-15` → 净 -5）。gaea 用**最长匹配 + 否定前缀**：

```go
// 词典：按关键词长度降序排列，最长匹配优先，命中即停（不再累加同族词）
var intimacyDict = []struct{ kw string; delta int }{
    {"不信任", -15}, {"信任", +10},          // 长词在前
    {"和解", +20}, {"背叛", -30}, {"决裂", -30}, {"反目", -25}, {"破裂", -25}, {"仇恨", -25},
    {"疏远", -15}, {"冲突", -15}, {"改善", +10}, {"加深", +15}, {"亲近", +15}, {"亲近感", +15},
    ... // 其余同 MuMu :13-26
}

// CalcIntimacyDelta 返回单次调整值，已钳制到 [-30, 30]。
// 语义：取第一个（最长的）匹配关键词的 delta，不再跨词累加。
func CalcIntimacyDelta(desc string) int {
    best := 0
    bestLen := 0
    for _, e := range intimacyDict {
        if len([]rune(e.kw)) > bestLen && strings.Contains(desc, e.kw) {
            best, bestLen = e.delta, len([]rune(e.kw))
        }
    }
    return clamp(best, -30, 30)
}
```
**更优解（推荐 A/B 并用）**：§5.2 的 Prompt 已让 LLM 直接输出 `delta_hint`，
当 `delta_hint != 0` 时**直接用 LLM 值**（同样钳制 ±30），仅在缺失时退回 `CalcIntimacyDelta`。
理由：LLM 已读完本章正文，语义判断远强于关键词表；关键词表退化为兜底。

### 7.4 状态回灌（复用 gaea 现成通道）

1. **图谱实体属性**：`internal/novelcontext/novelcontext.go:435-447` `entityFromCharacter` 里追加
   `"current_state": c.CurrentState, "career_main": ..., "career_stage": ...`，
   `sceneCharFromEntity`（:449-471）与 `formatSceneChar`（:782-790）按需渲染。
   → 现有 `status` 已走这条路（:438, :453, :463-464），**零新增管道**。
2. **Prompt 注入**：`prompts/chapter-generate.json` 的 `input_sections` 增加 `character_states`（P1），
   内容由后端按 MuMu `chapter_context_service.py:31-45` 的格式生成：
   ```
   当前状态: 存活（第3章变更）
   当前心理/处境: 坚定（第12章更新）
   主职业: 剑修 (3/10阶 - 金丹期)
   所属组织: 青云宗（内门弟子）[忠诚度:72]
   关系网络: 与李四：关系改善[65]；与王五：宿敌[20]
   ```
3. **降噪规则**（照搬 MuMu `chapters.py:794,821`）：`intimacy == 50` 与 `loyalty == 50` 的默认值**不注入**，
   减少 Prompt 噪声。

### 7.5 关系图谱升级（`frontend/src/components/RelationGraph.tsx`）

按 MuMu `RelationshipGraph.tsx` 蒸馏：
1. **节点三类**：角色 / 组织 / 职业（`career-main-{id}`、`career-sub-{id}`）+ 2 个虚拟分组节点
2. **边四层 + 分类筛选**（`EDGE_CATEGORY_META` 9 类，:105-115）
3. **布局权重**：组织成员边 = 8、职业分类 = 4、职业关联 = 3/2、人际 = 1；
   仅前 3 类参与布局，人际边只渲染（:971-974）
4. **虚拟分组节点常量**：`__career_group_main__` / `__career_group_sub__`（:100-101）
5. **职业详情浮层**：显示 `current_stage / max_stage / 阶段名 / 阶段描述`（照 `careers.py:642-669` 的 level 匹配逻辑）

### 7.6 职业管理页（`frontend/src/pages/CharacterPage.tsx` 增 Tab 或新页）

照 `Careers.tsx`：
- 主/副双 Tab、卡片显示阶段列表
- 阶段编辑文本行格式 `1. 阶段名 - 描述` + 正则解析（`Careers.tsx:113-129`）
- 「AI 生成新职业（增量式）」按钮：传已有职业摘要 + `user_requirements`，**绝不替换已有职业**
- 事件总线 `BACKGROUND_TASK_SETTLED` + `resources: ['careers']` 自动刷新（`Careers.tsx:71-82`）
- **补齐 MuMu 缺失的角色-职业绑定 UI**（见 §6.4）：角色详情内嵌「主职业：选择 / 当前阶段：1..max_stage（显示阶段名）/ 阶段进度」
  与「副职业：最多 2 个」列表，直连 §7.7 #9 的 `SetCharacterCareer` / `UpdateCareerStage` 绑定方法

### 7.7 落点与改动清单（供实施阶段使用）

| # | 落点 | 动作 |
|---|---|---|
| 1 | `internal/types/types.go:65-129` | Character/Organization/Relationship 扩展字段（§7.1） |
| 2 | `internal/types/types.go`（新增） | `Career` / `CareerStage` / `CharacterCareerRef` / `OrgMember` / `RelChange` |
| 3 | `internal/characterstate/`（新包） | `ApplyChapterDiff` + 7 条不变量 + `CalcIntimacyDelta` |
| 4 | `internal/characterstate/career.go` | 职业阶段钳制、名称→ID 匹配、主/副职业约束 |
| 5 | `internal/analysis/analysis.go:56-60` | `CharacterStateChange` 扩展为完整差分（survival/relationships/career/org） |
| 6 | `internal/analysis/analysis.go:186-209` | `syncCharacterStates` → 委托给 `characterstate.ApplyChapterDiff` |
| 7 | `internal/project/` | `careers.json` 读写（`ReadCareers`/`WriteCareers`） |
| 8 | `internal/novelcontext/novelcontext.go:435-471,782-790` | 实体属性与渲染追加 `current_state` / 职业 |
| 9 | `internal/app/bindings_novel.go` | 新增 `SaveCareer` / `DeleteCareer` / `SetCharacterCareer` / `UpdateCareerStage` |
| 10 | `prompts/analysis-chapter.json` | `character_states` 契约升级（§5.2） |
| 11 | `prompts/chapter-generate.json` | 新增 `character_states` P1 槽位 |
| 12 | `frontend/src/components/RelationGraph.tsx` | 职业节点 + 四层边 + 分类筛选（§7.5） |
| 13 | `frontend/src/pages/CharacterPage.tsx` | 职业 Tab / 角色详情显示当前状态与职业 |
| 14 | `prompts/character.json`（`character-agent`） | 输出契约追加 `current_state` 与职业字段 |

---

## 8. MuMu 侧缺陷清单（蒸馏时**不要**复制的部分）

| # | 位置 | 缺陷 | gaea 处置 |
|---|---|---|---|
| 1 | `character_state_update_service.py:301-303` | `current_state` 覆盖式写入，`state_before` 只进日志 **不入库** → 无状态时间线可查 | 新增 `StateHistory []StateChange{Chapter, From, To, Reason}` |
| 2 | `character_state_update_service.py:650-805` | `update_organization_states` **无章节号水位守卫**（对比 `:214`/`:293` 都有） | 加 `OrgStateUpdatedChapter` 守卫 |
| 3 | `career_update_service.py:359-361` vs `careers.py:819` vs `auto_character_service.py:178` | 副职业上限 **2 / 5 / [:2] 截断** 三处不一致 | 统一 `const MaxSubCareers = 2` |
| 4 | `career_update_service.py:129-290` | 职业阶段推进**无水位守卫**，重跑低章节会重复 +1 | `CharacterCareerRef.UpdatedChapter` |
| 5 | `character_state_update_service.py:820-823` | 关键词**子串全量累加**：`"不信任"` 命中 `"信任"+10` 与 `"不信任"-15` → 净 -5（语义失真） | §7.3 最长匹配 + LLM `delta_hint` 优先 |
| 6 | `character_state_update_service.py:251-258` | 存活级联的成员查询**缺 `project_id` 过滤**（级联 1 有，:230-241） | 统一带项目过滤 |
| 7 | `types.go:109` vs `relationship.py:72` | MuMu `power_level` 是 `int 0..100`；gaea 是 `string "实力等级"` | 并存 `PowerLevel`(旧) + `PowerValue int`(新)，逐步迁移 |
| 8 | `character_state_update_service.py:558` | `member_count` 只在新建成员时 +1，离开**不减** → 单调虚高；靠 `projects.py:494/543` 的 fix API 兜底 | **改为派生值**（`len(MemberList where status==active)`），不存字段 |
| 9 | `career_update_service.py:262-268` / `characters.py:380-509` | 职业状态**双写**（CharacterCareer 表 + Character.sub_careers JSON），任一路径漏写即不一致 | **单一事实源**：只存 `[]CharacterCareerRef`，不设冗余标量 |
| 10 | `career_service.py:110-191` vs `careers.py:367-429` | 职业落库逻辑**两条重复实现** | 只保留一份 |
| 11 | `auto_character_service.py:145-250` vs `characters.py:1017-1140` | 职业名称→ID 匹配**两处重复实现**（后者带 6 条调试日志） | 只保留一份 |
| 12 | `career.py:58` | `stage_progress` 在**全部 16 处后端构造点恒被置 0**（`career_update_service.py` 根本不写它），唯一消费方 `CharacterCareerCard.tsx` 是**未被任何文件 import 的死组件** | 要么实现（LLM 输出阶段内进度），要么不引入 |
| 13 | `character_state_update_service.py:625-641` | `change_type` 落到 `else` 分支时**静默只调忠诚度**，未知类型不报错 | 未知 `change_type` 记 warning 并丢弃 |
| 14 | `chapters.py:577-838` vs `chapter_context_service.py:533-666` | 角色上下文**两种格式**（一行式 / 多行式）并存 | 统一为多行式（信息密度高、易分段截断） |

---

## 9. 未能从源码确认的事项

- MuMu 是否在别处（如 `project_agent_extended_tools.py` 的 LLM 工具调用路径）也会写 `current_state`：
  已 grep 到 `project_agent_extended_tools.py:186-187` 与 `project_agent_tools.py:116-117,237-238,439,558` 引用这些字段，
  但本报告未逐行审阅该路径的写入语义（其为 Agent 工具层的通用字段白名单，非状态机本体）。**未能从源码确认其是否与章节分析竞争写。**
- `stage_progress` 是否有前端消费方：**已确认**——仅 `frontend/src/components/CharacterCareerCard.tsx` 消费，
  而该组件全仓库无 import 引用（grep `CharacterCareerCard` 仅命中其自身 `:41` 定义与 `:407` 默认导出）。
  其调用的 `/api/careers/character/{id}/careers[/main|/sub|/{career_id}/stage]` 系列接口（`CharacterCareerCard.tsx:67,104,122,142,165`）
  在 `Careers.tsx` 中**完全未被使用** → 角色-职业绑定与调阶的前端入口是**未接线的空档**。
  这直接影响 §7.6 的 UI 规格：gaea 不只是"照抄 Careers.tsx"，而要把这段缺失的绑定 UI 一并补上。
- MuMu 是否有 DB 迁移为 `career_update_service` 补水位字段：已列出 `backend/alembic/{sqlite,postgres}/versions/`，
  其中 `20260212_1244_d4d253e3f4c6_添加角色心理状态追踪字段.py:24-26` 与
  `20260212_0922_642c76fc69d4_添加角色心理状态追踪字段.py:24-25` 只加 `status_changed_chapter/current_state/state_updated_chapter`，
  **未见职业水位字段** → 与 §8 坑 4 一致。

---

## 10. 参考引用索引（全部锚定 600be703）

```
clones/MuMuAINovel@600be703:backend/app/services/character_state_update_service.py:1-829      # 全文
clones/MuMuAINovel@600be703:backend/app/services/career_update_service.py:1-398               # 全文
clones/MuMuAINovel@600be703:backend/app/services/career_service.py:1-234                      # 全文
clones/MuMuAINovel@600be703:backend/app/services/auto_character_service.py:1-551              # 全文
clones/MuMuAINovel@600be703:backend/app/services/auto_organization_service.py:1-200
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1050-1390（PLOT_ANALYSIS 角色/组织契约）
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:1926-2057（AUTO_CHARACTER_GENERATION）
clones/MuMuAINovel@600be703:backend/app/services/prompt_service.py:2189-2195（AUTO_ORGANIZATION_GENERATION 头）
clones/MuMuAINovel@600be703:backend/app/services/chapter_context_service.py:25-46,300-340,525-666
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:575-838（build_characters_info_with_careers）
clones/MuMuAINovel@600be703:backend/app/api/chapters.py:1000-1385（分析编排四级调用）
clones/MuMuAINovel@600be703:backend/app/api/memories.py:95-274（手动分析同套编排）
clones/MuMuAINovel@600be703:backend/app/api/careers.py:37-937
clones/MuMuAINovel@600be703:backend/app/api/characters.py:341-509,1017-1140
clones/MuMuAINovel@600be703:backend/app/api/relationships.py:32-262
clones/MuMuAINovel@600be703:backend/app/models/character.py:8-57
clones/MuMuAINovel@600be703:backend/app/models/career.py:8-77
clones/MuMuAINovel@600be703:backend/app/models/relationship.py:8-116
clones/MuMuAINovel@600be703:backend/app/schemas/career.py:1-155
clones/MuMuAINovel@600be703:backend/app/schemas/relationship.py:1-200
clones/MuMuAINovel@600be703:backend/app/init_relationship_types.py:16-45
clones/MuMuAINovel@600be703:backend/alembic/postgres/versions/20260212_1244_d4d253e3f4c6_添加角色心理状态追踪字段.py:24-34
clones/MuMuAINovel@600be703:backend/alembic/sqlite/versions/20260212_0922_642c76fc69d4_添加角色心理状态追踪字段.py:24-34
clones/MuMuAINovel@600be703:frontend/src/pages/Careers.tsx:1-500
clones/MuMuAINovel@600be703:frontend/src/pages/RelationshipGraph.tsx:100-250,563-1002,1071-1100
clones/MuMuAINovel@600be703:frontend/src/pages/Characters.tsx:40-85,277,1054,1333
clones/MuMuAINovel@600be703:frontend/src/components/CharacterCareerCard.tsx:1-45,67,104,122,142,165,183,223,385,407

# gaea 侧对照
internal/types/types.go:62-129
internal/character/character.go:1-718
internal/characterlib/model.go:1-118
internal/analysis/analysis.go:34-60,104-114,185-209
internal/novelcontext/novelcontext.go:435-471,782-790
internal/app/character_handler.go:58-178
internal/app/bindings_novel.go:30,79-95
frontend/src/components/RelationGraph.tsx:19-134
frontend/src/pages/CharacterPage.tsx:84,407,437,703-733
prompts/analysis-chapter.json（全文）
prompts/chapter-generate.json（全文）
prompts/character.json（全文）
```
