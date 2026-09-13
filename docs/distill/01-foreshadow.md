# MuMuAINovel 伏笔管理域 · 源码级分析报告

- 任务：t1（能力域【伏笔管理】源码蒸馏）
- 唯一事实来源：`clones/MuMuAINovel`（v1.5.5）
- 分析方法：直读源码，所有函数名/类名/字段名/Prompt 原文均带 `文件:行号`，无推断
- 目标：为 gaea（Go + Wails + React，内部包名 `internal/...`）小说板块的伏笔能力升级提供落地依据

---

## 0. 源码清单与体量

| 文件 | 字节 | 角色 |
|---|---|---|
| `backend/app/services/foreshadow_service.py` | 71175 | 核心业务服务（25 个方法） |
| `backend/app/api/foreshadows.py` | 13257 | REST 路由（13 个端点） |
| `backend/app/models/foreshadow.py` | 8355 | SQLAlchemy ORM 模型 |
| `backend/app/schemas/foreshadow.py` | 7288 | Pydantic 请求/响应契约 |
| `frontend/src/pages/Foreshadows.tsx` | 37751 | Antd 管理页（5 个 Modal） |
| `backend/app/services/chapter_context_service.py` | — | 伏笔提醒注入（2 处重复实现） |
| `backend/app/services/plot_analyzer.py` | — | 分析侧伏笔分层注入 |
| `backend/app/services/prompt_service.py` | — | Prompt 模板（伏笔追踪指令 + 变量槽） |
| `backend/alembic/sqlite/versions/20260119_1005_951919659e0f_添加伏笔管理表.py` | — | 建表迁移 |

**foreshadow_service.py 全量方法索引**（`grep "^    (async )?def"`）：

```
24   generate_stable_foreshadow_id()        # 模块级函数，非类方法
54   get_project_foreshadows()              128  get_foreshadow()
139  create_foreshadow()                     193  update_foreshadow()
232  delete_foreshadow()                     372  mark_as_planted()
413  mark_as_resolved()                     458  mark_as_abandoned()
485  sync_from_analysis()                   557  get_pending_resolve_foreshadows()
599  get_overdue_foreshadows()              637  get_must_resolve_foreshadows()
676  get_foreshadows_to_plant()             713  build_chapter_context()
826  get_stats()                            903  get_planted_foreshadows_for_analysis()
977  delete_chapter_foreshadows()          1064 clean_chapter_analysis_foreshadows()
1125 _revert_chapter_resolutions()          1175 clear_project_foreshadows_for_reset()
1245 auto_update_from_analysis()            1481 auto_plant_pending_foreshadows()
1548 _match_foreshadow_by_content()         1657 _calculate_word_overlap()
```
末尾 `foreshadow_service = ForeshadowService()`（`foreshadow_service.py:1695`）为模块级全局单例。

---

## 1. 数据模型（ORM 源码逐字段）

表名 `foreshadows`（`backend/app/models/foreshadow.py:20`）。类注释声明 5 项能力：从章节分析自动同步、用户手动添加、关联埋入/计划回收章节、长线伏笔管理、章节生成时提醒（`foreshadow.py:9-19`）。

| 字段 | 类型 | 默认 | 语义（原文注释摘录） |
|---|---|---|---|
| `id` | String(36) PK | `uuid4()` | `foreshadow.py:22` |
| `project_id` | String(36) FK→projects.id CASCADE, index | — | `foreshadow.py:23` |
| `title` | String(200) NOT NULL | — | 伏笔标题 `:26` |
| `content` | Text NOT NULL | — | 伏笔详细内容/描述 `:27` |
| `hint_text` | Text | — | 埋伏笔时的暗示文本(原文摘录或概述) `:28` |
| `resolution_text` | Text | — | 回收伏笔时的揭示文本(原文摘录或概述) `:29` |
| `source_type` | String(20) | `'manual'` | `analysis`=分析提取 / `manual`=手动添加 `:32` |
| `source_memory_id` | String(100) | — | 来源记忆ID(如从分析结果同步) `:33` |
| `source_analysis_id` | String(36) | — | 来源分析任务ID `:34` |
| `plant_chapter_id` | String(36) FK→chapters.id SET NULL | — | 埋入章节ID `:38` |
| `plant_chapter_number` | Integer | — | 埋入章节号(**冗余存储便于查询**) `:39` |
| `target_resolve_chapter_id` | String(36) FK→chapters.id SET NULL | — | 计划回收章节ID `:42` |
| `target_resolve_chapter_number` | Integer | — | 计划回收章节号 `:43` |
| `actual_resolve_chapter_id` | String(36) FK→chapters.id SET NULL | — | 实际回收章节ID `:46` |
| `actual_resolve_chapter_number` | Integer | — | 实际回收章节号 `:47` |
| `status` | String(20), index | `'pending'` | 5 态枚举，见下表 `:50-57` |
| `is_long_term` | Boolean | `False` | 是否长线伏笔(跨多章的重要伏笔) `:59` |
| `importance` | Float | `0.5` | 重要性评分 0.0-1.0 `:62` |
| `strength` | Integer | `5` | 伏笔强度 1-10(影响读者多强烈) `:63` |
| `subtlety` | Integer | `5` | 隐藏度 1-10(越高越隐蔽) `:64` |
| `urgency` | Integer | `0` | 紧急度: 0=不紧急, 1=需关注, 2=急需回收 `:65` |
| `related_characters` | JSON | — | 关联角色名列表 `:68` |
| `related_foreshadow_ids` | JSON | — | 关联的其他伏笔ID列表(**伏笔链**) `:69` |
| `tags` | JSON | — | 标签列表 `:70` |
| `category` | String(50) | — | `identity/mystery/item/relationship/event` `:71` |
| `notes` | Text | — | 创作备注(仅作者可见) `:74` |
| `resolution_notes` | Text | — | 回收方式说明 `:75` |
| `auto_remind` | Boolean | `True` | 是否在章节生成时自动提醒 `:78` |
| `remind_before_chapters` | Integer | `5` | 提前几章开始提醒回收 `:79` |
| `include_in_context` | Boolean | `True` | 是否包含在生成上下文中 `:80` |
| `created_at` | DateTime | `func.now()` server_default | `:83` |
| `updated_at` | DateTime | `func.now()` + onupdate | `:84` |
| `planted_at` | DateTime | — | 埋入时间 `:85` |
| `resolved_at` | DateTime | — | 回收时间 `:86` |

**关键设计点（对 gaea 有直接价值）**：
1. **三章号套件**：`plant_chapter_number` / `target_resolve_chapter_number` / `actual_resolve_chapter_number`。埋入、计划回收、实际回收分离 → 「计划 vs 实际」的偏差本身就是数据（超期检测的全部依据）。
2. **章节号冗余存储**：注释明写「冗余存储便于查询」（`:39`）。因为服务层几乎所有查询（待回收/超期/必须回收/超期统计）都按 `number` 过滤排序，不做 join。
3. **`is_foreshadow` 与 `urgency` 的区别**：`urgency` 在 DB 中有默认值但**从未被服务层写入**（除 `create_foreshadow` 显式置 0，`:170`）；实际紧急度是**运行时算出来的**（`get_urgency_level()`，见 §3.2）。这是一个已固化的设计取舍，gaea 应保持「不落库、按需计算」。
4. **`source_type` 二值化驱动清理语义**：`analysis` 来源的伏笔视为「可再生副产物」，可被批量删除；`manual` 来源视为「作者资产」，只可重置状态（`clear_project_foreshadows_for_reset`，`foreshadow_service.py:1175-1243`）。这是整个表最重要的业务不变量。

### 1.1 状态机（5 态，原文枚举）

`backend/app/schemas/foreshadow.py:8-14`：

```python
class ForeshadowStatus(str, Enum):
    PENDING = "pending"                        # 待埋入
    PLANTED = "planted"                        # 已埋入
    RESOLVED = "resolved"                      # 已回收
    PARTIALLY_RESOLVED = "partially_resolved"  # 部分回收
    ABANDONED = "abandoned"                    # 已废弃
```

状态迁移（从服务层写操作反推，全部为真实赋值点）：

| 触发 | 入口 | 迁移 | 副作用 |
|---|---|---|---|
| 创建 | `create_foreshadow` `:139`（强制 `status="pending"`，`:165`；`source_type="manual"`，`:162`） | ∅→pending | — |
| 标记埋入 | `mark_as_planted` `:372` | pending→planted `:394` | 写 `plant_chapter_id` / `plant_chapter_number` / `planted_at=datetime.now()` `:395-397`；`hint_text` 若非空则覆盖 `:399-400` |
| 自动埋入 | `auto_plant_pending_foreshadows` `:1481` | pending→planted `:1526` | 写 `plant_chapter_id` / `planted_at` `:1527-1528`（**不写 plant_chapter_number**，因查询条件已保证相等） |
| 标记回收 | `mark_as_resolved` `:413` | planted→resolved，或 `is_partial=True` 时 planted→partially_resolved `:435-438` | 写 `actual_resolve_chapter_id/number` / `resolved_at` `:440-442`；`resolution_text` 非空则覆盖 `:444-445` |
| 分析自动回收 | `auto_update_from_analysis` `:1337-1340` | planted→resolved | 同上游 + `resolution_text = fs_data["content"]` `:1343-1344` |
| 废弃 | `mark_as_abandoned` `:458` | *→abandoned `:470` | `notes` 追加 `\n[废弃原因] {reason}` `:472` |
| 章节分析清理回退 | `_revert_chapter_resolutions` `:1125` | resolved/partially_resolved→planted `:1155-1161` | 清空 `actual_resolve_chapter_id/number`、`resolved_at`、`resolution_text` |
| 全新生成重置 | `clear_project_foreshadows_for_reset` `:1175` | planted/resolved/partially_resolved→pending `:1216-1226` | 清空三套章节关联 + 两个时间戳，**且 `target_resolve_chapter_number=None`** |

未实现的迁移：`pending→abandoned` 无专用端点（但 `mark_as_abandoned` 不校验原状态，API 层允许）；`partially_resolved→resolved` 只能再调一次 `mark_as_resolved(is_partial=False)`（`:435-438` 不限制原状态）。

### 1.2 状态→首次埋入的语义边界

`get_planted_foreshadows_for_analysis`（`:903`）只取 `status == "planted"`（`:931`）。也就是说 **`partially_resolved` 的伏笔对后续分析不可见**，不会被再次回收。这是一个真实存在的口径缺口，gaea 落地时需显式决定（建议 `planted OR partially_resolved`）。

---

## 2. 服务层：25 个方法逐个拆解

### 2.1 CRUD 与查询

**`get_project_foreshadows`（`:54`）** — 列表 + 统计一次返回。
- 过滤条件累加：`project_id` + 可选 `status` / `category` / `source_type` / `is_long_term is not None`（注意用 `is not None` 而非真值，允许筛 `False`，`:91-92`）。
- 排序（`:103-107`）：
  ```python
  .order_by(
      Foreshadow.plant_chapter_number.asc().nulls_last(),
      desc(Foreshadow.importance),
      desc(Foreshadow.created_at)
  )
  ```
  **未计划的伏笔（无埋入章号）排最后**，再按重要性降序。`nulls_last()` 是关键——gaea 直接 SQL 时需注意 SQLite 的 NULL 排序默认相反。
- 分页 `offset((page-1)*limit)` `:108-109`，同时用独立 `count_query` 取 total `:95-97`（不做窗口函数）。
- 每次列表请求都**顺带调 `get_stats`**（`:116`），无 `current_chapter` → `overdue_count` 恒为 0。

**`get_foreshadow`（`:128`）** — `scalar_one_or_none()`，不存在返回 `None`（不抛）。这是所有写操作的统一前置检查模式。

**`create_foreshadow`（`:139`）** — 硬编码的不变量（`source_type="manual"` `:162`、`status="pending"` `:165`、`urgency=0` `:170`）；`plant_chapter_number` / `target_resolve_chapter_number` 允许创建时直接给定（`:163-164`），即可「先规划后埋入」。**不校验** `plant_chapter_number` 与 `target_resolve_chapter_number` 的先后关系。

**`update_foreshadow`（`:193`）** — 通用字段级更新：`data.model_dump(exclude_unset=True)` + `hasattr` 守卫逐个 `setattr`（`:216-219`）。**这是 DB 层的字段白名单兜底**，但意味着客户端可用 PUT 直接改 `status`（schema 允许，`schemas/foreshadow.py:83`），绕过状态机。gaea 若做严格状态机，应显式拒绝 `status` 出现在通用 update 中。

**`delete_foreshadow`（`:232`）** — 本域最复杂的删除：**四路联动清理**。
1. 取 `project` 拿 `user_id`（`:243-246`）。
2. 构造关键词集合：`title.strip()` + `content[:50].strip()`（`:252-259`）。
3. 关系库清理 `StoryMemory`：`project_id` + `memory_type=="foreshadow"` + (`content.contains(kw)` OR `title.contains(kw)`)，`delete().rowcount`（`:261-275`）。
4. 向量库清理：`memory_service.delete_foreshadow_memories(user_id, project_id, foreshadow_keywords)`（`:277-282`）。该方法自带注释承认局限：「当前记忆系统未持久化 reference_foreshadow_id / foreshadow_id 映射，因此这里采用内容关键词匹配作为清理策略」（`memory_service.py:556-558`）。
5. 历史分析清理：遍历该项目**全部** `PlotAnalysis`（`:284-287`），对 `analysis.foreshadows[]` 里每个 dict 判定是否移除：
   - 命中 A：`item["reference_foreshadow_id"] == foreshadow_id`（`:306`）→ 回收历史记录。
   - 命中 B（仅当 `source_type=="analysis"`，`:310`）：重新计算该 item 的稳定 ID，与该伏笔的 `source_memory_id` 比对；或 `item_title == foreshadow.title.strip()` 且 `content_snippet in item_content`（`:317-332`）。
   - **注释明确解释了动机**：「清理历史埋入记录，避免『手动同步分析伏笔』再次从 PlotAnalysis 重建已删除伏笔」（`:309`）。
6. 重算受影响分析的 `foreshadows_planted` / `foreshadows_resolved` 计数（`:342-349`），最后 `db.delete` + `commit`（`:357-358`）。
7. 返回计数日志：关系记忆/向量记忆/历史分析引用三项（`:360-364`）。

### 2.2 章节上下文构建（核心算法）

**`build_chapter_context`（`:713`）** — 「智能分层提醒策略」，docstring 四层（`:725-729`）：

> 1. 本章必须回收的伏笔 → 明确要求回收
> 2. 超期伏笔 → 强调需要尽快回收
> 3. 即将回收的伏笔 → 仅作为背景信息，**明确禁止提前回收**
> 4. 远期伏笔 → **不发送**，防止干扰

实现出的 `context_text` 四段（`:743-802`），逐段原文格式：

```
【🎯 本章必须回收的伏笔 - 请务必在本章完成回收】
- ID:{f.id[:8]} | {f.title}
  埋入章节：第{f.plant_chapter_number}章
  伏笔内容：{f.content[:100]}{'...' if len>100 else ''}
  回收提示：{f.resolution_notes}          # 仅当非空

【⚠️ 超期待回收伏笔 - 请尽快回收】
- ID:{f.id[:8]} | {f.title} [已超期{chapter_number - target}章]
  埋入章节：第{plant}章，原计划第{target}章回收
  伏笔内容：{f.content[:80]}...            # 注意：无条件加省略号

【📋 近期待回收伏笔（仅供参考，请勿在本章回收）】
⚠️ 以下伏笔尚未到回收时机，本章请勿提前回收，仅作为剧情背景了解
- {f.title}（计划第{target}章回收，还有{remaining}章）

【✨ 本章计划埋入伏笔】
- {f.title}
  伏笔内容：{content[:80]}                 # 未加省略号，与超期段不一致
  埋入提示：{f.hint_text}                  # 仅当非空
```

**配额（硬编码，`:766` `:784`）**：超期段最多 3 条 `overdue[:3]`，近期段最多 5 条 `upcoming[:5]`，必须回收段与待埋入段**无上限**。三处截断长度 100/80/80。

**筛选逻辑（易错点）**：
- `upcoming = [f for f in upcoming_raw if (f.target_resolve_chapter_number or 0) > chapter_number]`（`:778-779`）——从 `get_pending_resolve_foreshadows` 结果里**减去**本章到期与超期的，以免同一伏笔在「必须回收」和「近期参考」里重复出现。
- 注意 `get_pending_resolve_foreshadows` 自身带 `auto_remind == True` 过滤（`:586`），而 `get_overdue_foreshadows` **不带**（`:617-628`）→ **关闭自动提醒只影响「近期参考」段，超期与必须回收段照旧输出**。这是真实行为，gaea 若要「一键静音」需自行统一。
- 返回结构含 `"recently_planted": []`，注释「# 可扩展」（`:811`），是对应 schema `ForeshadowContextResponse.recently_planted` 的**永远为空**占位。

**返回字段与 schema 不一致（重要）**：`build_chapter_context` 返回 `must_resolve`（`:808`），但 `backend/app/schemas/foreshadow.py:190-197` 的 `ForeshadowContextResponse` **没有 `must_resolve` 字段**，只有 `pending_plant` / `pending_resolve` / `overdue` / `recently_planted`。而 `ForeshadowContextRequest`（`:182-187`）声明的 `include_pending/include_overdue/lookahead` 被 API 层读为 query 参数并透传（`api/foreshadows.py:96-117`）。→ **`must_resolve` 会被 FastAPI 响应模型静默丢弃**；前端若需要该层信息只能靠 `context_text` 文本。gaea 契约设计时必须修正此点。

### 2.3 四个「按章节号」检索器

| 方法 | WHERE 条件 | 排序 | 上限 | 行号 |
|---|---|---|---|---|
| `get_pending_resolve_foreshadows(project, cur, lookahead=5)` | `status='planted'` AND `target != None` AND `target <= cur+lookahead` AND **`auto_remind=True`** | `target` asc | 无 | `:557-597` |
| `get_overdue_foreshadows(project, cur)` | `status='planted'` AND `target != None` AND `target < cur` | `target` asc | 无 | `:599-635` |
| `get_must_resolve_foreshadows(project, ch)` | `status='planted'` AND `target == ch` | `importance` desc | 无 | `:637-674` |
| `get_foreshadows_to_plant(project, ch)` | **`status='pending'`** AND `plant_chapter_number == ch` | `importance` desc | 无 | `:676-711` |

四个方法的 catch 分支全部 `return []`（`:597` `:635` `:674` `:711`）——**降级为空而非抛出**，是「章节生成不因伏笔数据坏而失败」的容错基线。

**`auto_plant_pending_foreshadows` 的语义（`:1481`）** 值得单独标注：文档字符串说「如果章节内容中包含相关关键词，则自动标记为 planted」（`:1493`），但实现里 `should_plant = True` 被**无条件硬编码**（`:1523`），且注释解释了原因：

> 用户明确指定了本章埋入的伏笔，自动标记为已埋入。注：只有 pending 状态且 plant_chapter_number == chapter_number 的伏笔才会被 get_foreshadows_to_plant 查出，所以这里直接标记即可（`:1520-1522`）

即：**参数 `chapter_content` 接收但完全不使用**（死参数），「关键词匹配」是文档与实现的偏差。gaea 不要照抄这段 docstring。

### 2.4 统计

**`get_stats`（`:826`）** — 两个查询 + 一次可选重查：
1. `GROUP BY status` 得各状态计数（`:845-855`），`total = sum(status_counts.values())`（`:858`）——**从分组和推导，不做第二次 count**。
2. `is_long_term == True` 单独 count（`:861-871`）。
3. `overdue_count` 只在 `current_chapter` 传入时计算，且**复用 `get_overdue_foreshadows` 全量查询再取 len**（`:874-877`）——不是 SQL count，存在 N 行传输。
4. 异常时返回**全 0 字典**（`:892-901`），绝不抛。

### 2.5 分析侧注入（含「分层注入」策略）

**`get_planted_foreshadows_for_analysis（project, current_chapter_number=None）`（`:903`）** — 只取 `status='planted'`（`:931`），按 `plant_chapter_number` asc（`:934`）。每条产出：

```python
{"id","title","content","hint_text","plant_chapter_number",
 "target_resolve_chapter_number","category","related_characters","is_long_term",
 "resolve_status", "resolve_hint"}
```

`resolve_status` 四值判定（`:955-967`）——这是全域最有复用价值的**枚举设计**：

| 条件 | `resolve_status` | `resolve_hint`（原文） |
|---|---|---|
| `target == current` | `must_resolve_now` | `"本章必须回收此伏笔"` |
| `target < current` | `overdue` | `f"已超期{current - target}章，应尽快回收"` |
| `target > current` | `not_yet` | `f"计划第{target}章回收，请勿提前回收"` |
| 无 `target` 或未传 `current` | `no_plan` | `"无明确回收计划，根据剧情自然回收"` |

消费方 `PlotAnalyzer._format_existing_foreshadows`（`plot_analyzer.py:196-268`）据此做**三层注入**：

- 第 1 层 `must_resolve`（最详细，`plot_analyzer.py:223-240`）：含 `【ID: {fs_id}】`、埋入章、内容 200 字、`埋入暗示` 100 字，并**显式指令** `⚠️ 回收时 reference_foreshadow_id 填写: {fs_id}`。
- 第 2 层 `overdue`（较详细，最多 5 条 `:245`）：仅 ID/标题/埋入章。
- 第 3 层 `others`（精简，最多 10 条 `:255`，超出追加 `... 还有{n}个伏笔未列出` `:260-261`）。
- 尾部统一操作指引（`:265-266`）：
  > 提示：如果章节内容回收了上述任一伏笔，请在 foreshadows 数组中添加 type='resolved' 的记录，并在 reference_foreshadow_id 填写对应ID。

调用点：`api/chapters.py:1020-1025`（后台分析准备）、`api/memories.py:79-83`。`current_chapter_number` 由调用方传当前章号以实现智能标记。

### 2.6 生命周期清理（四组，语义各不相同）

| 方法 | 行号 | 删除 | 重置/回退 | 保留 |
|---|---|---|---|---|
| `delete_chapter_foreshadows(project, chapter_id, only_analysis_source=True)` | `:977` | 匹配 `plant_chapter_id==ch` OR `actual_resolve_chapter_id==ch` OR `source_analysis_id IN (该章的分析ID)`（`:1013-1020`），默认叠加 `source_type=='analysis'`（`:1028-1029`） | — | 手动伏笔（默认） |
| `clean_chapter_analysis_foreshadows(project, chapter_id)` | `:1064` | `source_type=='analysis'` AND `plant_chapter_id==ch`（`:1089-1095`） | 调 `_revert_chapter_resolutions` 回退本章回收（`:1107`） | 手动伏笔 |
| `_revert_chapter_resolutions(project, chapter_id)` | `:1125` | — | `UPDATE ... WHERE actual_resolve_chapter_id==ch AND status IN ('resolved','partially_resolved') SET status='planted', actual_*=NULL, resolved_at=NULL, resolution_text=NULL`（`:1146-1162`），取 `rowcount` | — |
| `clear_project_foreshadows_for_reset(project)` | `:1175` | `DELETE WHERE project_id AND source_type=='analysis'`（`:1195-1200`） | `UPDATE WHERE source_type=='manual' AND status IN (...)` → `pending` + 清空全部章节关联与时间戳 + `target_resolve_chapter_number=NULL`（`:1207-1227`） | 手动伏笔记录本身 |

`delete_chapter_foreshadows` 有一个**必须继承的实现细节**：它先去 `PlotAnalysis` 查该章的 `analysis_id` 列表（`:1002-1004`），因为 `sync_from_analysis` 产生的伏笔用 `source_analysis_id` 关联——但 `auto_update_from_analysis` 创建记录时**并未写 `source_analysis_id`**（对比 `:1433-1455` 的构造参数，无该字段）。也就是说 `source_analysis_id` 在本版本里**恒为 NULL**，此 OR 分支是历史兼容死代码。gaea 落地时要么补写该字段，要么删掉该分支（不要留半通不通的关联）。

### 2.7 分析结果→伏笔的自动同步（全域最核心复杂逻辑）

**`sync_from_analysis(project, data: SyncFromAnalysisRequest)`（`:485`）** — 纯编排层：按 `chapter_ids`（可选，`data.chapter_ids` 为空则全项目，`:515-517`）拉 `PlotAnalysis`，逐条取其 `chapter` 得 `chapter_number`，**委托** `auto_update_from_analysis`（`:535-541`），汇总：
- `synced_count += planted_count + resolved_count`（`:544`）
- `resolved_count += resolved_count`（`:545`）
- `skipped_count += skipped_resolve_count`（`:546`）

注意 `SyncFromAnalysisRequest` 声明的 `overwrite_existing` 与 `auto_set_planted`（`schemas/foreshadow.py:169-170`）在服务层**从未被读取**——两个死参数。

**`auto_update_from_analysis(project, chapter_id, chapter_number, analysis_foreshadows)`（`:1245`）** — 决策引擎。

统计桶（`:1273-1281`）：
```python
{"planted_count", "resolved_count", "created_count",
 "updated_ids": [], "created_ids": [],
 "matched_by_content": 0, "errors": [],
 "skipped_resolve_count": <动态追加>}
```

**硬上限**：`MAX_NEW_FORESHADOWS_PER_CHAPTER = 5`（`:1287`），**仅约束新建**，更新已有记录不占额度（`:1424-1426`）。

**`type == "resolved"` 分支（`:1295-1367`）——三级匹配 + 两道防重**：

- 策略 1（优先，`:1303-1312`）：`reference_id` 存在 → `get_foreshadow` + `project_id == project_id` 校验。**关键防御**（原文注释 `:1300-1302`）：
  > 如果分析结果里已经给出了 reference_foreshadow_id，但该伏笔已被用户删除或已不属于当前项目，则直接跳过，不再退回内容匹配，避免把「已删除伏笔」的旧分析结果错误同步到其他同名/近似伏笔上。
  
  失败即 `skipped_resolve_count += 1` + `errors.append(f"reference_foreshadow_id 无效或已删除: {reference_id}")` + `continue`。
- 策略 2（仅当无 `reference_id`，`:1315-1323`）：`_match_foreshadow_by_content`，命中则记 `matched_by_content = True` 并重新取完整 ORM 对象。
- 防重（`:1326-1332`）：`existing.status == "resolved"` 时，若 `actual_resolve_chapter_number == chapter_number` 记 info「已在本章回收过，跳过」，否则记 warning「已在第N章回收，跳过」，两者均 `continue`。
- 执行（`:1335-1356`）：仅当 `existing.status == "planted"` 才写 resolved；成功后从 `planted_foreshadows` 内存列表剔除 `f['id'] != existing.id`（`:1356`），避免同一批分析多次匹配同一条。
- 状态非 planted（`:1357-1358`）：warning 跳过。
- **找不到匹配（`:1359-1367`）——重要的业务原则，原文注释**：
  > 核心原则：只有"埋入"操作会创建伏笔记录，"回收"只是更新已有记录。如果没有埋入的伏笔，就不可能存在回收（`:1361-1362`）
  
  → 计数 `skipped_resolve_count`，**绝不创建记录**。

**`type == "planted"` 分支（`:1369-1464`）**：
1. `content` 为空 → warning 跳过（`:1370-1373`）。
2. `title` 为空 → 用 `content[:50] + "..."`（`:1375-1377`）。
3. 生成稳定 ID：`generate_stable_foreshadow_id(chapter_id, fs_content, "planted")`（`:1380-1382`）。
4. 去重查询（`:1385-1402`）——**双路径 OR**：
   - `source_memory_id == {稳定ID}`
   - 兼容旧数据：`title == fs_title AND plant_chapter_id == chapter_id AND source_type == 'analysis'`
5. 命中已有 → **原地更新 8 个字段 + 回写稳定 ID**（`:1406-1417`）：`title`、`content`、`strength`、`subtlety`（都带 `fs_data.get(k, 现有值)` 兜底）、`hint_text ← fs_data["keyword"]`、`category`、`is_long_term`、`related_characters`；`estimated_resolve_chapter` 非空时写 `target_resolve_chapter_number`；最后 `source_memory_id = source_memory_id`（`:1417`）。→ 幂等：**重新分析同一章只更新不新增**。
6. 新建（`:1433-1464`）：
   ```python
   importance = min(fs_data.get("strength", 5) / 10.0, 1.0)   # :1447 ★ 派生关系
   status = "planted"                                          # :1445 直接处于已埋入
   planted_at = datetime.now()                                 # :1443
   source_type = "analysis"                                    # :1439
   target_resolve_chapter_number = fs_data.get("estimated_resolve_chapter")  # :1444
   auto_remind = True / remind_before_chapters = 5 / include_in_context = True  # :1452-1454
   ```
   **`estimated_resolve_chapter` 为 None 时不设默认值**（`:1428-1431`，原文注释：「不再为 estimated_resolve_chapter 设置默认值，避免误报『超期』」）——这是防误报超期的关键决策。
7. 单条 `except` 包裹整个循环体（`:1466-1469`）：单条失败只记 `errors`，不中断批次。末尾统一 `commit`（`:1471`）。

**`_match_foreshadow_by_content(resolved_fs_data, planted_foreshadows, min_similarity=0.5)`（`:1548`）——六策略加权评分**：

先做标题后缀清理（`:1584-1589`）：对 `["回收","揭示","解答","兑现"]` 逐个 `endswith` 命中即剥离（只剥一个）。这是对 Prompt 中「不要加『回收』后缀」指令的**兜底**。

逐候选累加/取最大（注意：**取最大 vs 累加语义不同**）：

| 策略 | 条件 | 得分 | 行号 |
|---|---|---|---|
| 1a | `resolved_title == fs_title` | `score = 1.0` | `:1604-1606` |
| 1b | `resolved_title_clean == fs_title` | `score = 0.95` | `:1607-1609` |
| 1c | 双向包含（原标题） | `max(score, 0.8)` | `:1610-1612` |
| 1d | 双向包含（清理标题） | `max(score, 0.75)` | `:1613-1615` |
| 1e | 标题 2/3-gram Jaccard | `max(score, overlap * 0.7)`；`>0.3` 才 debug 日志 | `:1616-1620` |
| 2 | `resolved_keyword in fs_content` | `max(score, 0.75)` | `:1623-1625` |
| 3 | 内容 n-gram 相似度 | `max(score, overlap * 0.6)` | `:1628-1630` |
| 4 | `reference_chapter == fs_plant_chapter` | `score += 0.15`（**累加**） | `:1633-1635` |
| 5 | `resolved_category == fs_category` | `score += 0.1`（**累加**） | `:1638-1640` |
| 6 | 关联角色 Jaccard | `score += character_overlap * 0.1`（**累加**） | `:1643-1645` |

采纳条件（`:1648-1650`）：`score > best_score and score >= min_similarity` → 即**阈值 0.5 硬门限**，且取分数最高者（`>` 而非 `>=`，同分取先出现者）。

**`_calculate_word_overlap(text1, text2)`（`:1657`）——字符级 n-gram Jaccard**：
```python
def get_ngrams(text, n):
    text = text.lower().replace(" ", "").replace("\n", "")
    if len(text) < n: return {text}
    return {text[i:i+n] for i in range(len(text) - n + 1)}

overlap_2 = |A2 ∩ B2| / max(|A2 ∪ B2|, 1)     # 2-gram
overlap_3 = |A3 ∩ B3| / max(|A3 ∪ B3|, 1)     # 3-gram
return overlap_2 * 0.4 + overlap_3 * 0.6       # ★ 3-gram 权重更高
```
**这是纯确定性、零依赖、对中文友好的相似度实现**——gaea 用 Go 落地时可直接用 `map[string]struct{}` + rune 切片复刻（注意 Go 必须按 rune 而非 byte 切，否则中文 n-gram 全错）。

### 2.8 稳定 ID 生成

**`generate_stable_foreshadow_id(chapter_id, content, foreshadow_type="planted")`（`foreshadow_service.py:24-48`）**，模块级函数（非类方法）：

```python
content_normalized = content.strip().lower()                              # :42
content_hash = hashlib.md5(content_normalized.encode('utf-8')).hexdigest()[:12]  # :43
chapter_hash = hashlib.md5(chapter_id.encode('utf-8')).hexdigest()[:8]    # :46
return f"{foreshadow_type}_{chapter_hash}_{content_hash}"                 # :48
```

docstring 声明的三条保证（`:28-32`）：
> 1. 同一章节、相同内容的伏笔只有一个唯一ID
> 2. 重新分析同一章节不会产生新ID
> 3. 标识符足够短且可读

**设计对照**：gaea 现役 `analysis.GenerateStableID`（`internal/analysis/analysis.go:211`）也是 `{type}_{chapter}_{content_hash}` 形态，语义同构。

### 2.9 运行时紧急度计算

**`Foreshadow.get_urgency_level(current_chapter)`（`models/foreshadow.py:156-178`）**：

```python
if self.status != 'planted' or not self.target_resolve_chapter_number:
    return 0
chapters_remaining = self.target_resolve_chapter_number - current_chapter
if chapters_remaining < 0:      return 3   # 已超期
elif chapters_remaining <= 2:   return 2   # 急需回收
elif chapters_remaining <= self.remind_before_chapters: return 1  # 需关注
else:                           return 0   # 不紧急
```

**注意文档与实现不一致**：docstring 写 `0=不紧急, 1=需关注, 2=急需回收, 3=已超期`（`:164`）；而 DB 列注释只到 2（`models/foreshadow.py:65`），`schemas.ForeshadowUpdate.urgency` 的校验是 `ge=0, le=3`（`schemas/foreshadow.py:89`）。**权威口径是运行时函数的 0-3**。

阈值体系（全域散落的硬编码常量汇总）：

| 常量 | 值 | 位置 | 语义 |
|---|---|---|---|
| 急需回收阈值 | `<= 2` 章 | `models/foreshadow.py:173` | urgency=2 |
| 需关注阈值（可配） | `remind_before_chapters` 默认 5 | `models/foreshadow.py:79` / `:175` | urgency=1 |
| schema 上限 | `1..20` | `schemas/foreshadow.py:64` | 配置边界 |
| lookahead 默认 | 5（context）/ 3（章节生成提醒） | `:562` / `chapter_context_service.py:1043` | 未来窗口 |
| lookahead 上限 | `1..20` | `schemas/foreshadow.py:187` | |
| 每章新伏笔上限 | 5 | `foreshadow_service.py:1287` | |
| 超期段展示 | 3 条 | `:766` | |
| 近期段展示 | 5 条 | `:784` | |
| 内容截断 | 100 / 80 字 | `:756` / `:770` | |
| 模型侧标题预览截断 | 100 字 | `models/foreshadow.py:143` | `to_context_string` |
| 分析注入折叠 | 超期 5 / 其他 10 | `plot_analyzer.py:245` / `:255` | |
| 分析注入内容截断 | 200 / 暗示 100 | `plot_analyzer.py:230` / `:238` | |
| 匹配阈值 | `min_similarity = 0.5` | `:1552` | |

**两个 `to_context_string` / `to_dict` 辅助方法**：
- `to_dict()`（`models/foreshadow.py:91-127`）：**全字段 + 数组归一化** `self.related_characters or []`、`related_foreshadow_ids or []`、`tags or []`（`:114-116`），时间统一 `.isoformat()`（`:123-126`）。这是 API 响应的实际载体（API 层 `return foreshadow.to_dict()` 而非 Pydantic 序列化）。
- `to_context_string()`（`:129-154`）：`f"伏笔「{title}」" + f"(第{n}章埋下)" + f": {content[:100]}" + f" [计划第{n}章回收]" + f" 涉及: {', '.join(related_characters[:3])}"`。**实际服务层未调用它**（提醒文本在 `build_chapter_context` 里另写）。

---

## 3. 集成点：谁在调用伏笔能力

| 调用方 | 位置 | 调用内容 | 时机 |
|---|---|---|---|
| 章节生成 | `api/chapters.py:1580` `:1605` `:2151` `:2664` `:4186` … | 把 `foreshadow_service` 传给 `ChapterContextService` | 生成前编译上下文 |
| 章节生成后 | `api/chapters.py:1869-1877` `:2368` `:2902` `:4430` | `auto_plant_pending_foreshadows(...)` | commit + refresh 之后 |
| 章节分析前 | `api/chapters.py:1020-1025` | `get_planted_foreshadows_for_analysis(current_chapter_number)` | 后台分析任务 |
| 章节分析后 | `api/chapters.py:1383-1405` | `auto_update_from_analysis(...)`，**独立 try/except** 且只 warning | 分析完成 |
| 章节清空/重生成 | `api/chapters.py:400` `:490` | `delete_chapter_foreshadows` | 删章 |
| 重新分析前 | `api/chapters.py:1206` | `clean_chapter_analysis_foreshadows` | |
| 大纲删除 | `api/outlines.py:314` `:358` | `delete_chapter_foreshadows` | 一对多/一对一删章 |
| 全新生成 | `api/outlines.py:1213` `:1984` | `clear_project_foreshadows_for_reset` | 重置项目 |
| 大纲续写 | `api/outlines.py:1511-1529` | `build_chapter_context` → 失败时降级为 `get_stats` 拼 `【📊 伏笔统计】已埋设:{p} 已回收:{r} 部分回收:{pr} 待埋入:{pd}` | |
| 记忆分析 | `api/memories.py:79-83` `:114-116` `:257-269` | 分析前取伏笔、写回 `PlotAnalysis.foreshadows`、分析后 `auto_update_from_analysis` | |
| 项目删除 | `api/projects.py:328-331` | 直接删 Foreshadow 行 | DB 级 CASCADE 之外的显式清理 |
| 拆书导入 | `book_import_service.py:714` | `delete(Foreshadow).where(project_id==)` | 导入前清空 |

**注入链路（章节生成）**：`ChapterContextService._build_context`（`chapter_context_service.py:343-348`）→ `_get_foreshadow_reminders` → 写入 `context.foreshadow_reminders`（dataclass 字段，`:142` / `:195`）→ 进入 `context_stats["foreshadow_length"]`（`:360`）→ 在 `prompt_service.py` 的生成模板中被 `{foreshadow_reminders}` 槽位消费。

**重复实现警告**：`_get_foreshadow_reminders` 在 `chapter_context_service.py` 中出现 **两次**（`:985-1061` 与 `:1793-1869`），分别属于两个类（第二个类 `__init__` 在 `:1137`）。两处逻辑几乎逐行相同（差异仅日志文案与 `【📋 即将到期的伏笔（仅供参考）】` 尾部行）。gaea 落地时**只应有一份**。

### 3.1 章节上下文与 Prompt 注入的耦合

`chapter_context_service._get_foreshadow_reminders` 有自己的配额（与 `build_chapter_context` **不同**）：必须回收段**无上限**；超期段 `[:3]`（`:1031`）；近期段 `[:3]`（`:1052`）+ `lookahead=3`（`:1043`）。文本长度分别为 100 / 80（`:1017` / `:1035`）。→ **同一套数据在两条路径上有两套配额**，gaea 必须统一。

---

## 4. API 契约（13 个端点）

路由前缀 `/api/foreshadows`（`api/foreshadows.py:25`）。**全部端点**首行都是 `verify_project_access`（`api/common.py`），且 `HTTPException` 直接 re-raise、其余异常统一包 `HTTPException(500, ...)`。

| # | 方法 | 路径 | 请求 | 响应模型 | 行号 |
|---|---|---|---|---|---|
| 1 | GET | `/projects/{project_id}` | query: `status`,`category`,`source_type`,`is_long_term`,`page≥1`,`limit 1..100` | `ForeshadowListResponse` | `:28-66` |
| 2 | GET | `/projects/{project_id}/stats` | query: `current_chapter≥1` 可选 | `ForeshadowStatsResponse` | `:69-88` |
| 3 | GET | `/projects/{project_id}/context/{chapter_number}` | query: `include_pending`,`include_overdue`,`lookahead 1..20` | `ForeshadowContextResponse`（**丢 `must_resolve`**） | `:91-125` |
| 4 | GET | `/projects/{project_id}/pending-resolve` | query: `current_chapter`**必填**,`lookahead 1..20` | `{total, items[]}`（**未声明 response_model**） | `:128-157` |
| 5 | GET | `/{foreshadow_id}` | — | `ForeshadowResponse` | `:160-183` |
| 6 | POST | `` (前缀根) | body `ForeshadowCreate` | `ForeshadowResponse` | `:186-208` |
| 7 | PUT | `/{foreshadow_id}` | body `ForeshadowUpdate` | `ForeshadowResponse` | `:211-235` |
| 8 | DELETE | `/{foreshadow_id}` | — | `{message, id}` | `:238-262` |
| 9 | POST | `/{foreshadow_id}/plant` | body `PlantForeshadowRequest` | `ForeshadowResponse` | `:265-293` |
| 10 | POST | `/{foreshadow_id}/resolve` | body `ResolveForeshadowRequest` | `ForeshadowResponse` | `:296-324` |
| 11 | POST | `/{foreshadow_id}/abandon` | query `reason` 可选 | `ForeshadowResponse` | `:327-355` |
| 12 | POST | `/projects/{project_id}/sync-from-analysis` | body `SyncFromAnalysisRequest` | `SyncFromAnalysisResponse` | `:358-381` |
| — | GET | `/projects/{project_id}/foreshadows`（**记忆路由下**） | `api/memories.py:409-433` | `{foreshadows[], total}` | 旧版未回收伏笔（走 `memory_service.find_unresolved_foreshadows`，基于向量库 `is_foreshadow`，**与 `foreshadows` 表是两套数据源**） |

**路由顺序陷阱（必须注意）**：`/{foreshadow_id}`（`:160`）定义在 `/projects/{project_id}/...` 之后，FastAPI 按注册顺序匹配，因此无冲突；但若 gaea 用 Go 的 `http.ServeMux` 或 chi 实现，**必须保证 `/projects/` 前缀路由先注册**，否则 `projects` 会被 `{foreshadow_id}` 吃掉（chi 的 `/projects/{id}` 模式不冲突，但 Go 1.22 ServeMux 的 `{foreshadow_id}` 通配会命中 `/projects/xxx`）。

**请求体字段约束（Pydantic，`schemas/foreshadow.py`）**：
- `ForeshadowCreate` = `ForeshadowBase` + `project_id: str`（`:68-70`）。`title` `min=1,max=200`；`content` `min=1`；`plant_chapter_number` / `target_resolve_chapter_number` 均 `ge=1`；`importance` `0.0..1.0`；`strength` / `subtlety` `1..10`；`remind_before_chapters` `1..20`。
- `ForeshadowUpdate`（`:73-101`）：全 `Optional`，独有 `status: Optional[ForeshadowStatus]`、`urgency: Optional[int] ge=0 le=3`、`related_foreshadow_ids`。**`plant_chapter_number` 等仍受 `ge=1` 约束**，无法显式清空。
- `PlantForeshadowRequest`（`:151-155`）：`chapter_id` 必填、`chapter_number ge=1` 必填、`hint_text` 可选。
- `ResolveForeshadowRequest`（`:158-163`）：`chapter_id`/`chapter_number` 必填、`resolution_text` 可选、`is_partial: bool = False`。
- `SyncFromAnalysisRequest`（`:166-170`）：`chapter_ids: Optional[List[str]]`、`overwrite_existing: bool = False`（死）、`auto_set_planted: bool = True`（死）。
- `SyncFromAnalysisResponse`（`:173-179`）：`synced_count`、`skipped_count`、`resolved_count: int = 0`、`new_foreshadows: List[ForeshadowResponse] = []`（**服务层从不填充，恒空数组**）、`skipped_reasons: List[dict] = []`（**同样恒空**——服务层把原因写进 `stats["errors"]` 而非 `skipped_reasons`，字段名对不上）。
- `ForeshadowStatsResponse`（`:139-148`）：`total/pending/planted/resolved/partially_resolved/abandoned/long_term_count/overdue_count` —— 与 `get_stats` 返回 dict 的键**完全一致**（这是唯一严丝合缝的契约）。
- `ForeshadowListResponse`（`:132-136`）：`total`、`items`、`stats: Optional[dict]`（**用裸 dict 而非 `ForeshadowStatsResponse`**，类型信息丢失）。

**响应体来源**：所有单条响应走 `foreshadow.to_dict()`（ORM 方法），**不是** Pydantic `from_attributes` 序列化——`ForeshadowResponse.Config.from_attributes = True`（`:128-129`）实际未被使用。

---

## 5. AI Prompt 原文摘录（含文件:行号）

### 5.1 分析侧：伏笔 ID 追踪任务（`prompt_service.py:1047-1060`）

```
PLOT_ANALYSIS = """<system>
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

### 5.2 分析侧：已埋入伏笔列表槽位（`prompt_service.py:1072-1077`）

```
<existing_foreshadows priority="P1">
【已埋入伏笔列表 - 用于回收匹配】
以下是本项目中已埋入但尚未回收的伏笔，分析时如发现章节内容回收了某个伏笔，请使用对应的ID：

{existing_foreshadows}
</existing_foreshadows>
```

### 5.3 分析侧：伏笔字段规范（`prompt_service.py:1103-1126`，逐字）

```
**2. 伏笔分析 (Foreshadowing) - 🔴 支持ID追踪**
- 埋下的新伏笔：内容、预期作用、隐藏程度(1-10)
- 回收的旧伏笔：【必须】从已埋入伏笔列表中匹配ID
- 伏笔质量：巧妙性和合理性
- **关键词**：【必填】从原文逐字复制8-25字

每个伏笔需要：
- **title**：简洁标题（10-20字，概括伏笔核心）
  - ⚠️ 回收伏笔时，标题应与原伏笔标题保持一致，不要添加"回收"等后缀
  - 例如：原伏笔标题是"绿头发的视觉符号"，回收时标题仍为"绿头发的视觉符号"，而非"绿头发的视觉符号回收"
- **content**：详细描述伏笔内容和预期作用
- **type**：planted（埋下）或 resolved（回收）
- **strength**：强度1-10（对读者的吸引力）
- **subtlety**：隐藏度1-10（越高越隐蔽）
- **reference_chapter**：回收时引用的原埋入章节号，埋下时为null
- **reference_foreshadow_id**：【回收时必填】被回收伏笔的ID（从已埋入伏笔列表中选择），埋下时为null
  - 🔴 重要：回收伏笔时，必须从【已埋入伏笔列表】中找到对应的伏笔ID并填写
  - 如果列表中有标注【ID: xxx】的伏笔，回收时必须使用该ID
  - 如果无法确定是哪个伏笔，才填写null（但应尽量避免）
- **keyword**：【必填】从原文逐字复制8-25字的定位文本
- **category**：分类（identity=身世/mystery=悬念/item=物品/relationship=关系/event=事件/ability=能力/prophecy=预言）
- **is_long_term**：是否长线伏笔（跨10章以上回收为true）
- **related_characters**：涉及的角色名列表
- **estimated_resolve_chapter**：【必填】预估回收章节号（埋下时必须预估，回收时为当前章节）
```

> 注：`is_long_term` 的判定口径在此处被明确为「跨 10 章以上回收」，与 gaea 现役 lint 的 `foreshadowStaleAfterChapters = 10`（`internal/app/novel_foreshadow_lint_handler.go:21`）**巧合一致**。

### 5.4 分析侧：JSON 输出样例（`prompt_service.py:1242-1271`，逐字）

```json
"foreshadows": [
  {
    "title": "伏笔简洁标题",
    "content": "伏笔详细内容和预期作用",
    "type": "planted",
    "strength": 7,
    "subtlety": 8,
    "reference_chapter": null,
    "reference_foreshadow_id": null,
    "keyword": "从原文逐字复制的8-25字文本",
    "category": "mystery",
    "is_long_term": false,
    "related_characters": ["角色A", "角色B"],
    "estimated_resolve_chapter": 15
  },
  {
    "title": "回收的伏笔标题",
    "content": "伏笔如何被回收的描述",
    "type": "resolved",
    "strength": 8,
    "subtlety": 6,
    "reference_chapter": 5,
    "reference_foreshadow_id": "abc123-已埋入伏笔的ID",
    "keyword": "从原文逐字复制的8-25字文本",
    "category": "mystery",
    "is_long_term": false,
    "related_characters": ["角色A"],
    "estimated_resolve_chapter": 10
  }
]
```

（该模板使用 `{{` / `}}` 转义花括号，实际渲染由 `PromptService.format_prompt` 处理。）

### 5.5 分析侧：约束与自检（`prompt_service.py:1366`）

```
✅ 【伏笔ID追踪】回收伏笔时，必须从【已埋入伏笔列表】中查找匹配的ID填入 reference_foreshadow_id
```

### 5.6 注入侧：动态生成的「已埋入伏笔列表」文本（`plot_analyzer.py:196-268`）

```python
# 第1层（:224-240）
lines.append("=" * 40)
lines.append("【🎯 本章必须回收的伏笔】")
lines.append("=" * 40)
lines.append(f"{i}. 【ID: {fs_id}】{fs_title}")
lines.append(f"   埋入章节：第{plant_chapter}章")
lines.append(f"   伏笔内容：{fs_content}{'...' if len(fs.get('content',''))>200 else ''}")
lines.append(f"   埋入暗示：{hint_text[:100]}")
lines.append(f"   ⚠️ 回收时 reference_foreshadow_id 填写: {fs_id}")

# 第2层（:244-250）
lines.append("【⚠️ 超期未回收伏笔 - 如章节内容回收了请标记】")
lines.append(f"- 【ID: {fs_id}】{fs_title}（第{plant_chapter}章埋入）")

# 第3层（:254-262）
lines.append("【📋 其他已埋入伏笔 - 如章节内容自然回收了请标记】")
lines.append(f"- 【ID: {fs_id}】{fs_title}（第{plant_chapter}章埋入）")
lines.append(f"  ... 还有{len(others)-10}个伏笔未列出")

# 操作指引（:265-266）
lines.append("提示：如果章节内容回收了上述任一伏笔，请在 foreshadows 数组中")
lines.append("添加 type='resolved' 的记录，并在 reference_foreshadow_id 填写对应ID。")
```

### 5.7 生成侧：`{foreshadow_reminders}` 槽位（4 个模板，优先级不同）

| 模板 | 行号 | priority | 标题文案 |
|---|---|---|---|
| `CHAPTER_GENERATION_ONE_TO_MANY`（首章） | `:593-596` | **P2** | `【🎯 伏笔提醒】` |
| `CHAPTER_GENERATION_ONE_TO_ONE`（首章） | `:657-660` | **P2** | — |
| `CHAPTER_GENERATION_ONE_TO_ONE_NEXT` | `:733-736` | **P2** | — |
| `CHAPTER_GENERATION_ONE_TO_MANY_NEXT` | `:822-825` | **P1** | `【🎯 伏笔提醒 - 需关注】` |

渲染形态（`:593-596`）：
```
<foreshadow_reminders priority="P2">
【🎯 伏笔提醒】
{foreshadow_reminders}
</foreshadow_reminders>
```

对应约束条款（`:610` 与 `:840`，逐字相同）：
```
✅ 如有伏笔提醒，请在本章中适当埋入或回收相应伏笔
```

**P2→P1 的升级**是本域最重要的 Prompt 工程信号：**首章/无前文时伏笔提醒是 P2（背景），有前文续写时升为 P1（需关注）**。

### 5.8 伏笔变量注册（`prompt_service.py:2948` / `:2963`）

```python
"...", "foreshadow_reminders", "relevant_memories", "story_skeleton", "previous_chapter_summary"]   # :2948
"...", "chapter_careers", "foreshadow_reminders", "relevant_memories"]                                # :2963
```
`"PLOT_ANALYSIS": {` 变量集合定义在 `:2979`。

---

## 6. 前端交互清单（`frontend/src/pages/Foreshadows.tsx`，1037 行）

### 6.1 常量表

`STATUS_CONFIG`（`:26-32`）：
| status | label | color | icon |
|---|---|---|---|
| `pending` | 待埋入 | `default` | `ClockCircleOutlined` |
| `planted` | 已埋入 | `green` | `BulbOutlined` |
| `resolved` | 已回收 | `blue` | `CheckCircleOutlined` |
| `partially_resolved` | 部分回收 | `orange` | `ExclamationCircleOutlined` |
| `abandoned` | 已废弃 | `default` | `CloseCircleOutlined` |

`CATEGORY_CONFIG`（`:35-43`）：`identity` 身世/purple · `mystery` 悬念/magenta · `item` 物品/gold · `relationship` 关系/cyan · `event` 事件/blue · `ability` 能力/green · `prophecy` 预言/volcano。

`statusOrder`（`:358-364`）——**默认排序口径**：`planted:1, pending:2, partially_resolved:3, resolved:4, abandoned:5`（已埋入优先，因为需要关注回收）。

### 6.2 状态与数据加载

- 5 个 Modal 状态：`editModalVisible` / `syncModalVisible` / `detailModalVisible` / `plantModalVisible` / `resolveModalVisible`（`:62-66`）。
- 3 个筛选状态：`statusFilter` / `categoryFilter` / `sourceFilter`（`:57-59`）；分页 `currentPage` / `pageSize`（默认 20）。
- `loadForeshadows`（`:80-103`）：一次性拿 `items` + `total` + `stats`（利用 API 顺带返回的 stats）。
- `loadStats`（`:128-141`）：**只统计有内容的章节**（`chapters.filter(c => c.content)`），取 `maxChapter` 作为 `current_chapter` → 前端自己算超期基准。
- `loadChapters` / `loadCharacters`（`:106-125`）：为埋入/回收章节选择器、关联角色选择器供数。
- **事件总线联动**（`:149-160`）：监听 `EventNames.BACKGROUND_TASK_SETTLED`，当 `payload.projectId` 匹配且 `payload.resources` 包含 `'foreshadows'` 时自动刷新列表与统计。→ **后台分析任务完成后页面自刷新**，这是「分析→伏笔」闭环的 UI 触点。
- 表格高度自适应：`tableScrollY = max(containerHeight - 55, 200)`，监听 `resize` + 100ms 延迟重算，依赖 `stats` 变化（`:163-182`）。

### 6.3 表格列（7 列，`:367-512`）

| 列 | dataIndex | 宽 | 渲染要点 |
|---|---|---|---|
| 状态 | `status` | 100 | `Tag color` + icon；`sorter` 按 `statusOrder` |
| 标题 | `title` | — | 点击 `<a>` → 详情；`is_long_term` 追加紫色「长线」Tag；下方叠加 `getUrgencyBadge` |
| 分类 | `category` | 80 | 无值显示 `-`，未知值回落原串 |
| 埋入章节 | `plant_chapter_number` | 120 | `第N章`；**`defaultSortOrder: 'ascend'`**；缺值按 `999999` 排后 |
| 计划回收 | `target_resolve_chapter_number` | 120 | 同上排序口径 |
| 重要性 | `importance` | 100 | **`'★'.repeat(round(importance*5)) + '☆'.repeat(5-...)`**（5 星制） |
| 来源 | `source_type` | 80 | `analysis` → 蓝色「分析」，否则绿色「手动」 |
| 操作 | — | 200 | 见下 |

操作列（`:473-510`）**按状态条件渲染**：
- 「查看详情」`EyeOutlined` → `openDetailModal`（恒显）
- 「编辑」`EditOutlined` → `openEditModal`（恒显）
- 「标记埋入」`FlagOutlined` → `openPlantModal`，**仅 `status === 'pending'`**（`:481`）
- 「标记回收」`CheckCircleOutlined` → `openResolveModal`，**仅 `status === 'planted'`**（`:486`）
- 「废弃」`CloseCircleOutlined` + Popconfirm「确定要废弃这个伏笔吗？」→ `handleAbandon`，**仅当 `status !== 'abandoned' && status !== 'resolved'`**（`:491-500`）
- 「删除」`DeleteOutlined` + Popconfirm「确定要删除这个伏笔吗？」→ `handleDelete`（恒显）

### 6.4 紧急度徽标（`getUrgencyBadge`，`:337-355`）

```tsx
if (status !== 'planted' || !target_resolve_chapter_number) return null;
const currentMaxChapter = max(chapters 且 content 非空的 chapter_number, 默认 0);
const remaining = target_resolve_chapter_number - currentMaxChapter;
if (remaining < 0)      <Badge status="error"   text={`已超期${Math.abs(remaining)}章`} />
else if (remaining <= 3) <Badge status="warning" text={`还剩${remaining}章`} />
else null;
```
**注意**：前端阈值是 `<= 3`，后端 `get_urgency_level` 是 `<= 2`（`models/foreshadow.py:173`）→ **两处口径不一致**，gaea 必须统一（建议后端为准，前端只渲染后端算好的 `urgency`）。

### 6.5 统计卡片（7 张，`:517-560`）

`Row gutter=16` + 7 × `Col span=3`：总计 / 待埋入（`colorTextSecondary`）/ 已埋入（`colorSuccess`）/ 已回收（`colorPrimary`）/ 长线伏笔（`colorInfo`）/ 超期未回收（`:541-548`，红色高亮，值 `stats.overdue_count`）。最后一张卡触发 `loadStats` 的重算链。

### 6.6 筛选与工具栏（`:580-624`）

3 个 `Select allowClear`（状态/分类/来源，placeholder 分别为「状态筛选」「分类筛选」「来源筛选」）+ 刷新按钮（`Tooltip title="刷新列表"`）。每个筛选变化触发 `currentPage` 重置与 `loadForeshadows`。

### 6.7 五个 Modal 的字段与校验

**① 添加/编辑伏笔**（`:702-836`，`title = currentForeshadow ? '编辑伏笔' : '添加伏笔'`）：

| Form.Item name | label | 控件 | 校验/约束 |
|---|---|---|---|
| `title` | 伏笔标题 | `Input maxLength={200}` | `required`「请输入标题」 |
| `category` | 分类 | `Select allowClear` | — |
| `content` | 伏笔内容 | `TextArea rows={3}` | `required`「请输入内容」 |
| `plant_chapter_number` | 计划埋入 | `InputNumber min={1}` | — |
| `target_resolve_chapter_number` | 计划回收 | `InputNumber min={1}` | — |
| `related_characters` | 关联角色 | `Select`（角色列表） | — |
| `importance` | 重要性 (0-1) | 滑块/数字 | — |
| `strength` | 强度 (1-10) | 数字 | — |
| `subtlety` | 隐藏度 (1-10) | 数字 | — |
| `is_long_term` | 长线伏笔 | `Switch valuePropName="checked"` | — |
| `hint_text` | 暗示文本 | `TextArea rows={2}` | — |
| `notes` | 备注 | `TextArea rows={2}` | — |
| `auto_remind` | 自动提醒 | `Switch` | — |
| `include_in_context` | 包含在生成上下文 | `Switch` | — |
| `remind_before_chapters` | 提前几章提醒 | 数字 | — |

编辑态用 `form.setFieldsValue({...foreshadow, tags: foreshadow.tags || [], related_characters: foreshadow.related_characters || []})` 做**数组字段 null 归一**（`:305-309`）——对应后端 `to_dict` 的 `or []`。

**② 伏笔详情**（`:839-948`，`width={600}`，footer = 关闭 + 编辑）：正文 `whiteSpace: 'pre-wrap'`；展示 title、状态 Tag、长线 Tag、分类 Tag、伏笔内容、暗示文本、揭示文本、埋入章节（缺值「未设定」）、计划回收（缺值「未设定」）、实际回收（仅非空展示）、重要性（`★ × round(importance*5)`）、强度 `{n}/10`、隐藏度 `{n}/10`、关联角色 Tag 列表、备注、来源（「章节分析提取」/「手动添加」）。

**③ 标记伏笔埋入**（`:951-979`）：`chapter_id`「选择埋入章节」`Select` **required**「请选择章节」 + `hint_text`「暗示文本（可选）」`TextArea rows={3}`。`handlePlant`（`:224-244`）从前端 `chapters` 数组按 id 反查 `chapter_number` 一并发给后端。

**④ 标记伏笔回收**（`:981-1011`）：`chapter_id`「选择回收章节」required + `resolution_text`「揭示文本（可选）」`TextArea rows={3}` + `is_partial`「是否部分回收」`Switch`。同样前端反查章号。

**⑤ 手动同步分析伏笔**（`:1013+`）：`handleSync`（`:282-298`）只传 `{ auto_set_planted: true }`，成功文案：
```
`同步完成: 新增${result.synced_count}个伏笔, 跳过${result.skipped_count}个`
```
→ 印证 `overwrite_existing` 未被前端使用（死参数）。

### 6.8 API 客户端（`frontend/src/services/api.ts:1290-1370`）

13 个方法一一对应后端端点，路径前缀 `/foreshadows`。注意 `abandonForeshadow` 把 `reason` 放 **query params**（`:1355-1360`），与后端 `Query(None)` 一致。

### 6.9 TypeScript 类型（`frontend/src/types/index.ts:973-1108`）

`ForeshadowStatus`（`:974`）与 `ForeshadowCategory`（`:976`）为字符串字面量联合，与后端枚举严格对齐。`Foreshadow`（`:978-1013`）逐字段对齐 ORM。`ForeshadowStats`（`:1061-`）含 8 个计数字段。`ForeshadowListResponse.stats?: ForeshadowStats`（`:1075`）。

---

## 7. 与 gaea 现状的差距分析（事实对照，非建议）

gaea 现役伏笔实现（读自本工作区）：

| 维度 | MuMuAINovel | gaea 现状 | 差距类型 |
|---|---|---|---|
| 存储 | SQLAlchemy 表 `foreshadows`（`project_id` FK，索引 `project_id`/`status`） | JSON 文件 `foreshadows.json`（`types.ForeshadowFile{Items []Foreshadow}`，`internal/types/types.go:209-212`） | **架构差异**：gaea 无库表、无 SQL 检索能力 |
| 状态 | 5 态 `pending/planted/resolved/partially_resolved/abandoned` | 3 态 `planted/hinted/revealed`（`types.go:185-189`） | 缺 `pending`（先规划后埋入）、`partially_resolved`、`abandoned` |
| 章节引用 | 埋入/计划回收/实际回收**三套**（id+number） | `PlantedIn` / `RevealedIn` 两个**章节文件名**（`types.go:196-197`），**无计划回收字段** | 缺「计划回收章」→ 无法做超期预测 |
| 评分维度 | `importance`(0-1) / `strength`(1-10) / `subtlety`(1-10) / `urgency`(0-3 运行时) | 无评分字段 | 缺优先级排序依据 |
| 长线 | `is_long_term` + lint 豁免 | `IsLongTerm` + lint 豁免（`novel_foreshadow_lint_handler.go:102`） | **已对齐** |
| 关联 | `related_characters[]` / `related_foreshadow_ids[]`（伏笔链）/ `tags[]` / `category`(7 值) | 仅 `Category`（`character/plot/world/relationship`，`types.go:194`） | 缺伏笔链、缺角色关联、分类维度不同 |
| 提示文本 | `hint_text`（埋入暗示）+ `resolution_text`（回收揭示），可作原文摘录 | 仅 `Description` | 缺原文锚点 → 无法做「文本级回收核验」 |
| 分层注入 | 4 层（必须回收/超期/近期-禁止提前/待埋入），带预算与条数封顶 | 1 层（「未回收伏笔（创作约束）」，按章名过滤 revealed，条数封顶 15、预算 1600 rune，`create_chapter_handler.go:569-573`,`:638-660`） | 缺「必须回收/超期/禁止提前回收」的**时序约束表达** |
| 分析→回收 | 三级匹配（`reference_foreshadow_id` 精确 → 内容六策略加权评分 → 落空则跳过），`min_similarity=0.5` | `syncForeshadows` 按 StableID 或**描述**匹配（`analysis.go:116-160`），无相似度评分、无阈值 | 缺模糊匹配兜底 |
| 稳定 ID | `{type}_{md5(chapter_id)[:8]}_{md5(content)[:12]}` | `GenerateStableID(type, chapterFile, content)`（`analysis.go:211`） | 形态同构（**已对齐**） |
| 每章新建上限 | 5（`MAX_NEW_FORESHADOWS_PER_CHAPTER`） | 无 | 缺防爆量保护 |
| 防误报超期 | `estimated_resolve_chapter` 为空则**不设默认值**（`:1428-1431`） | 无该概念 | 缺 |
| Lint | 无独立 lint（靠 `build_chapter_context` 分层 + `get_stats.overdue_count`） | 5 类确定性检查 `ordering/status-mismatch/dangling/stale/duplicate`（`novel_foreshadow_lint_handler.go:64-121`） | **gaea 更强**，应保留 |
| 统计 | `get_stats` 8 项，异常返回全 0 | `stats.go:64-67` 读伏笔算总计/已回收 | gaea 缺 `overdue_count`/`long_term_count`/分状态 |
| 删除联动 | 关系库 + 向量库 + 历史分析三路清理 | 无 | 缺（gaea 无向量库依赖，但「历史分析引用清理」语义需保留） |
| 前端 | Antd 表格 + 7 统计卡 + 5 Modal + 后台任务事件刷新 | `internal/app/foreshadow_handler.go`（全量写回）+ `novel_foreshadow_lint_handler.go`（体检报告）；React 页面未在本报告范围内核实 | 需查 gaea 前端是否已有伏笔面板 |

**gaea 已有的、不应回退的能力**：`lintForeshadowItems` 的 5 类确定性检查（MuMu 完全没有）——这是 gaea 独有资产，升级时应作为**新数据模型上的增强**而非替换。

---

## 8. 已知缺陷清单（落地时必须规避/修正）

| # | 缺陷 | 证据 | 影响 |
|---|---|---|---|
| D1 | `must_resolve` 被响应模型静默丢弃 | 服务返回 `:808`，schema 无该字段 `schemas/foreshadow.py:190-197` | 前端拿不到「本章必须回收」结构化列表，只能解析文本 |
| D2 | `SyncFromAnalysisRequest.overwrite_existing` / `auto_set_planted` 从未被读取 | 定义 `:169-170`；服务层无引用 | 死参数，误导读调用方 |
| D3 | `SyncFromAnalysisResponse.new_foreshadows` / `skipped_reasons` 恒为空 | 服务层只填 `errors`（`:1311` `:1468`），字段名不匹配 | 客户端拿不到失败原因明细 |
| D4 | `auto_plant_pending_foreshadows` 的 `chapter_content` 参数完全未使用 | `:1487` 形参 vs `:1523` `should_plant = True` 硬编码 | docstring 与实现不符（声称关键词匹配） |
| D5 | `source_analysis_id` 恒为 NULL | 创建时不写入（`:1433-1455` 无该参数）；清理逻辑却依赖它（`:1019-1020`） | `delete_chapter_foreshadows` 的第三个 OR 分支为死代码 |
| D6 | `partially_resolved` 伏笔对后续分析不可见 | `get_planted_foreshadows_for_analysis` 只取 `planted`（`:931`） | 部分回收的伏笔永远无法被二次回收完成 |
| D7 | 紧急度阈值前后端不一致（后端 `<=2`、前端 `<=3`） | `models/foreshadow.py:173` vs `Foreshadows.tsx:351` | 同一伏笔两处显示不同紧急度 |
| D8 | `auto_remind` 只作用于「近期参考」段，不影响超期/必须回收段 | `get_pending_resolve_foreshadows` 带该过滤（`:586`），`get_overdue_foreshadows` 不带（`:617-628`） | 「关闭自动提醒」语义不完整 |
| D9 | `_get_foreshadow_reminders` 在两个类里重复实现且配额不同 | `chapter_context_service.py:985-1061` vs `:1793-1869`；配额 `[:3]/[:3]/lookahead=3` vs `build_chapter_context` 的 `[:3]/[:5]/lookahead=5` | 同一数据两条路径两套口径 |
| D10 | `PUT /{id}` 可绕过状态机改 `status` | `ForeshadowUpdate.status`（`schemas:83`）+ 通用 `setattr`（`:216-219`） | 状态一致性无强制保护 |
| D11 | `ForeshadowContextResponse.recently_planted` 永远为空 | 服务硬编码 `[]`（`:811`，注释「# 可扩展」） | 契约承诺了未实现的能力 |
| D12 | 列表响应 `stats` 用裸 dict | `schemas/foreshadow.py:136` | 类型约束丢失（Pydantic 不校验） |
| D13 | `get_stats` 的 `overdue_count` 走全量行传输 | `:876` 调 `get_overdue_foreshadows` 取 len | 大项目性能隐患 |
| D14 | 超期段文本无条件加 `...`（80 字内也加） | `:770` `f"  伏笔内容：{f.content[:80]}..."` | 显示瑕疵（与近期段 `:795` 不一致） |
| D15 | `get_urgency_level` 用 `self.status != 'planted'` 前置判断 | `models/foreshadow.py:166` | `partially_resolved` 紧急度恒为 0，丢失回收压力 |

---

## 9. 由本域得出的 Prompt 工程可复用要点

1. **ID 显式回填**：把 `⚠️ 回收时 reference_foreshadow_id 填写: {fs_id}`（`plot_analyzer.py:239`）放在每条候选**紧邻**位置，比只在说明区写一次有效——这是「分层注入」里第 1 层独有的指令。
2. **禁止提前回收要显式写**：`⚠️ 以下伏笔尚未到回收时机，本章请勿提前回收，仅作为剧情背景了解`（`foreshadow_service.py:783`）。模型的默认倾向是「看到伏笔就忍不住回收」，负向指令必需。
3. **标题一致性用示例锚定**：不要只写「保持一致」，要给出 `原伏笔标题是"绿头发的视觉符号"，回收时标题仍为"绿头发的视觉符号"，而非"绿头发的视觉符号回收"`（`prompt_service.py:1112`），并在代码侧用后缀剥离兜底（`:1584-1589`）。
4. **无计划的伏笔不要给默认回收章**：`不再为 estimated_resolve_chapter 设置默认值，避免误报"超期"`（`foreshadow_service.py:1428-1431`）——这是「缺失值优于猜值」的范式。
5. **「无埋入不可能有回收」硬原则**：`:1361-1362` 的注释把「回收只更新、绝不创建」上升为不变式，防止 AI 幻觉污染登记表。
6. **精确 ID 失效时宁可不匹配**：`:1300-1302` 明确「不退回内容匹配」，防止旧分析结果错绑到同名伏笔。这是**宁可漏收不可错收**的取舍。
7. **priority 随信息完备度升级**：`foreshadow_reminders` 在首章模板是 P2、续写模板是 P1（`prompt_service.py:593` vs `:822`）。
8. **变量槽位用 XML 标签 + priority 属性**：`<foreshadow_reminders priority="P2">` 而非裸 `{foreshadow_reminders}`，为模型提供结构信号。

---

## 10. 交付物索引

- 本文件：源码级分析报告（事实层）
- `t1-foreshadow-gaea-spec.md`：gaea 落地实施规格（DDL / API / Prompt / 前端 / 分阶段方案 / 测试）

---

*报告完成：全部函数名、字段名、常量值、Prompt 文本均取自 `clones/MuMuAINovel` v1.5.5 源码，行号已逐条核对。*
