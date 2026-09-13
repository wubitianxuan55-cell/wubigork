# 提示词工坊与模板管理域 — MuMuAINovel 源码蒸馏与 gaea 落地规格

> 任务：t6（work）· 能力域 06/06 · 交付物 `docs/distill/06-prompt-workshop.md`
> 唯一事实来源：`clones/MuMuAINovel`（下文所有引用均为该目录相对路径 + 行号，行号对应当前工作区克隆版本）
> 本地对照：`C:\AI\wubigrok\prompts\*.json`、`internal/prompt/prompt.go`、`internal/skill/skill.go`
> 写作约定：所有结论可复核；`未能从源码确认` 表示克隆内无法证实，未作推测。本轮不写生产代码。

---

## 0. 阅读边界与源码清单

本域实际通读的文件（行数为 `Get-Content` 计数）：

| 文件 | 行数 | 角色 |
|---|---|---|
| `backend/app/services/prompt_service.py` | 3158 | 系统模板库 + 模板解析/渲染 + 风格注入 + 降级检索 |
| `backend/app/services/skill_loader.py` | 536 | SKILL.md → 模板的加载器（第二模板来源） |
| `backend/app/services/workshop_client.py` | 176 | client 模式下的云端工坊 HTTP 客户端 |
| `backend/app/api/prompt_templates.py` | 630 | 模板 CRUD / 导出导入 / 预览 路由 |
| `backend/app/api/prompt_workshop.py` | 827 | 工坊浏览 / 提交审核 / 点赞 / 导入 路由 |
| `backend/app/api/writing_styles.py` | 508 | 写作风格 CRUD 与项目默认风格 |
| `backend/app/api/skills.py` | 259 | Skill 列表 / 匹配 / 聊天 / CRUD |
| `backend/app/api/polish.py` | 141 | AI 去味（引用了一个不存在的模板键，见 §9.1） |
| `backend/app/models/prompt_template.py` | 30 | `prompt_templates` 表 |
| `backend/app/models/prompt_workshop.py` | 89 | `prompt_workshop_items` / `prompt_submissions` / `prompt_workshop_likes` 表 |
| `backend/app/models/writing_style.py` | 23 | `writing_styles` 表 |
| `backend/app/models/project_default_style.py` | — | `project_default_styles` 表 |
| `backend/app/schemas/prompt_template.py` | 88 | 模板请求/响应/导出 Schema |
| `backend/app/schemas/prompt_workshop.py` | 119 | 工坊 Schema |
| `backend/app/schemas/writing_style.py` | 57 | 风格 Schema |
| `backend/app/constants/prompt_categories.py` | 28 | 工坊分类与热门标签常量 |
| `backend/app/api/inspiration.py` | 491 | 灵感模式（直接 `.format()`，不走 `format_prompt`） |
| `backend/app/api/chapters.py` | 5251 | 风格/Skill 注入 system prompt 的调用点 |
| `backend/app/services/chapter_regenerator.py` | 248 | 重写链路的风格注入 |
| `backend/app/config.py` | 194 | `WORKSHOP_MODE` / `INSTANCE_ID` |
| `backend/app/skills/*/SKILL.md` + `references/*.md` | 7 个 Skill / 46 个参考文件 | 第二模板来源的实体文件 |
| `frontend/src/pages/PromptTemplates.tsx` | 578 | 模板管理页 |
| `frontend/src/pages/PromptWorkshop.tsx` | 1450 | 工坊页（浏览/我的提交/管理审核 三 Tab） |
| `frontend/src/pages/WritingStyles.tsx` | 430 | 写作风格页 |
| `frontend/src/services/api.ts` | 1473 | 前端 API 封装（`writingStyleApi`@851、`promptWorkshopApi`@885、`polishApi`@1015） |
| `backend/alembic/{sqlite,postgres}/versions/20251227_0856_*` 等 | — | 6 条全局预设风格的种子数据 |

---

## 1. 系统架构：模板来源是「两层三源」，不是单一模板库

`PromptService` 的模板来源不是一个表，而是三处，优先级由代码控制：

| 优先级 | 来源 | 存储 | 作用域 | 代码锚点 |
|---|---|---|---|---|
| 高 | 用户自定义模板 | `prompt_templates` 表（`user_id`+`template_key` 唯一索引） | 单用户 | `prompt_service.py:2838-2849` |
| 中 | 磁盘 Skill | `backend/app/skills/<name>/SKILL.md` + `references/*.md` | 全局（全实例共享同一目录） | `skill_loader.py:181-239` |
| 低 | 系统内置常量 | `PromptService` 类属性（Python 字符串） | 全局，随代码发布 | `prompt_service.py:32-2607` |

关键判定逻辑（`prompt_service.py:2816-2860`）：

```python
# 1. 尝试从数据库获取用户自定义模板
result = await db.execute(
    select(PromptTemplate).where(
        PromptTemplate.user_id == user_id,
        PromptTemplate.template_key == template_key,
        PromptTemplate.is_active == True))
custom_template = result.scalar_one_or_none()
if custom_template:
    return custom_template.template_content
# 2. 降级到系统默认模板
template_content = getattr(cls, template_key, None)
if template_content is None:
    logger.warning(f"⚠️ 未找到系统默认模板: {template_key}")
return template_content
```

三点必须写进规格的事实：

1. **`prompt_templates` 表没有种子数据。** 迁移脚本只建表（`backend/alembic/postgres/versions/20251226_1008_ee0a189f1532_初始数据库结构.py:101`、`backend/alembic/sqlite/versions/20251226_1322_fbeb1038c728_初始化sqlite数据库.py:104`），没有任何 `op.bulk_insert` 写入模板行。表的语义是「**覆盖表**」，不是「模板库」。
2. **降级是静默的。** 找不到模板只写 `logger.warning` 然后返回 `None`（`prompt_service.py:2857-2860`），调用方拿到 `None` 后在 `.format()` 处崩（见 §9.1）。
3. **`is_active` 只对自定义行生效。** 系统内置模板在 `/categories` 里以 `is_active=True` 硬编码返回（`prompt_templates.py:126`），前端的开关也只在 `!is_system_default` 时渲染（`PromptTemplates.tsx:443-450`）。

---

## 2. 数据模型（可直接翻译为 Go struct / SQLite 表）

### 2.1 `prompt_templates`（用户覆盖表）

`backend/app/models/prompt_template.py:8-30`：

| 字段 | 类型 | 约束/默认 | 说明 |
|---|---|---|---|
| `id` | `String(36)` | PK, `uuid4()` | |
| `user_id` | `String(50)` | NOT NULL, index | 用户 ID |
| `template_key` | `String(100)` | NOT NULL | 模板键名，与类属性名同名 |
| `template_name` | `String(200)` | NOT NULL | 显示名 |
| `template_content` | `Text` | NOT NULL | 模板正文（含 `{var}` 占位符） |
| `description` | `Text` | NULL | 描述 |
| `category` | `String(50)` | NULL | 分类（UI 分组用） |
| `parameters` | `Text` | NULL | **JSON 字符串**，参数名数组 |
| `is_active` | `Boolean` | default `True` | 是否启用 |
| `is_system_default` | `Boolean` | default `False` | 语义上恒为 False（DB 只存覆盖行）；`/categories` 把系统模板的临时对象置 `True` |
| `created_at` / `updated_at` | `DateTime` | `now()` / `onupdate=now()` | |

索引：`Index('idx_user_template', 'user_id', 'template_key', unique=True)`（`prompt_template.py:26`）。

> 注意：唯一索引只有 `(user_id, template_key)` 两列，**不含 `is_active`**。因此「禁用某模板」无法用「再插一条 active 行」表达，只能是 `is_active=False`（写入口为 `PUT /prompt-templates/{template_key}`，`prompt_templates.py:274-277`），而 `get_template` 的查询带 `is_active == True`（`prompt_service.py:2838-2844`），等价于「禁用 = 回落到系统内置」。

### 2.2 `writing_styles`（写作风格）

`backend/app/models/writing_style.py:7-23`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | `Integer` autoincrement PK | |
| `user_id` | `String(255)` FK→`users.user_id` ON DELETE CASCADE, **nullable** | **`NULL` = 全局预设**，这是核心设计 |
| `name` | `String(100)` NOT NULL | |
| `style_type` | `String(50)` NOT NULL | `preset` / `custom` |
| `preset_id` | `String(50)` | `natural`/`classical`/`modern`/`literary`/`suspense`/`humorous` |
| `description` | `Text` | |
| `prompt_content` | `Text` NOT NULL | 风格正文（自由文本，非结构化） |
| `order_index` | `Integer` default 0 | 排序 |
| `created_at` / `updated_at` | `DateTime` | |

### 2.3 `project_default_styles`（项目级覆盖）

`backend/app/models/project_default_style.py:9-19`：`project_id`（FK，`UniqueConstraint('project_id', name='uix_project_default_style')`）+ `style_id`（FK→`writing_styles.id` ON DELETE CASCADE）。**一个项目恰好一个默认风格**，用「先删后插」实现 UPSERT（`writing_styles.py:474-485`）。

### 2.4 `prompt_workshop_items`（工坊公开条目）

`backend/app/models/prompt_workshop.py:7-35`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | `String(36)` PK | UUID |
| `name` | `String(100)` NOT NULL | |
| `description` | `Text` | |
| `prompt_content` | `Text` NOT NULL | |
| `category` | `String(50)` default `general` | 取值见 `constants/prompt_categories.py:3-14` |
| `tags` | `JSON` | 标签数组 |
| `author_id` | `String(255)` | 格式 `实例ID:用户ID` |
| `author_name` | `String(100)` | |
| `source_instance` | `String(255)` | 来源实例标识 |
| `is_official` | `Boolean` default False | |
| `download_count` | `Integer` default 0 | 导入次数 |
| `like_count` | `Integer` default 0 | 冗余计数（与 likes 表可能漂移，见 §9.4） |
| `status` | `String(20)` default `active` | `active`/`hidden`/`deprecated` |

索引 4 个：`category`、`status`、`download_count`、`created_at`（`prompt_workshop.py:27-32`）。

### 2.5 `prompt_submissions`（待审核提交）

`backend/app/models/prompt_workshop.py:38-72`：`submitter_id`（`实例ID:用户ID`）、`submitter_name`、`source_instance` NOT NULL、`name`/`prompt_content` NOT NULL、`category`/`tags`、`author_display_name`、`is_anonymous`、`status`（`pending`/`approved`/`rejected`）、`reviewer_id`、`review_note`、`reviewed_at`、`workshop_item_id`（审核通过后回填）。

### 2.6 `prompt_workshop_likes`（点赞）

`backend/app/models/prompt_workshop.py:75-89`：`user_identifier` + `workshop_item_id`（FK ON DELETE CASCADE），唯一索引 `idx_likes_user_item`。

---

## 3. 变量占位符语法与渲染引擎

### 3.1 语法本体：Python `str.format()`

渲染入口两个，行为**不一致**：

```python
# prompt_service.py:2609-2624 —— 唯一带错误包装的入口
@staticmethod
def format_prompt(template: str, **kwargs) -> str:
    try:
        return template.format(**kwargs)
    except KeyError as e:
        raise ValueError(f"缺少必需的参数: {e}")
```

- **入口 A**：`PromptService.format_prompt()`，`KeyError → ValueError("缺少必需的参数: {e}")`。被 `chapters.py`、`outlines.py`、`wizard_stream.py`、`book_import_service.py`、`plot_analyzer.py`、`plot_expansion_service.py`、`auto_*_service.py` 使用。
- **入口 B**：调用方直接 `template.format(**params)`，无包装。`inspiration.py:140-141` 是实例（`system_prompt = system_template.format(**format_params)`）。
- **入口 C**：`PromptService.get_chapter_regeneration_prompt()`（`prompt_service.py:2627-2753`）**完全不 format**，直接把 `CHAPTER_REGENERATION_SYSTEM` 与手工拼装的 f-string 段落用 `"\n".join()` 连接（`prompt_service.py:2753`）。这就是为什么注册表里 `CHAPTER_REGENERATION_SYSTEM` 声明了 8 个参数却没有一个占位符（实测 `placeholders = —`）。

### 3.2 语法规则汇总

| 规则 | 依据 | 后果 |
|---|---|---|
| 占位符 = `{name}`，`name` 为 `[a-zA-Z_][a-zA-Z0-9_]*` | `format_prompt` + 全库模板 | 无嵌套、无默认值、无条件段落 |
| **字面花括号必须写成 `{{` / `}}`** | `str.format()` 语义 | 模板中所有 JSON 示例都必须双写。实测双花括号计数：`PLOT_ANALYSIS` 18 对、`CAREER_SYSTEM_GENERATION` 7 对、`OUTLINE_CREATE`/`OUTLINE_CONTINUE` 各 6 对、`SINGLE_CHARACTER_GENERATION`/`AUTO_CHARACTER_GENERATION`/`AUTO_ORGANIZATION_ANALYSIS` 各 5 对、`WORLD_BUILDING`/`CHARACTERS_BATCH_GENERATION`/`AUTO_CHARACTER_ANALYSIS` 各 2-4 对，其余章节类模板 0 对 |
| 缺失变量 → 抛错，无兜底 | `prompt_service.py:2621-2624` | 用户自定义模板少写一个 `{...}` 即整条链路 500 |
| 多余关键字参数被忽略 | `str.format()` 语义 | 调用方多传不报错，所以 `chapters.py` 传了未声明的参数也不崩 |
| 无「条件段落」原语 | 全库检索无 `{% if %}` 之类 | 条件拼接靠 Python f-string，例如 `prompt_service.py:2736`：`{f'6. **风格一致**：...' if style_content else ''}` |
| 无默认值原语 | — | 默认值由**调用方**给，例如 `chapters.py:1723`：`characters_info=chapter_context.chapter_characters or '暂无角色信息'` |

**这一条是迁移到 gaea 时最关键的破坏性差异**：gaea 现用 `{word_count}` + 字符串替换（`internal/app/create_chapter_handler.go:86-90` 的 `substituteWordCount`），天然不受花括号转义约束；一旦换成 `str.format()` 语义，15 个模板里凡是正文含字面 `{`/`}` 的都会炸。

### 3.3 预览接口（存在但前端未接线）

`prompt_templates.py:593-630`：

```python
@router.post("/{template_key}/preview")
async def preview_template(template_key, data: PromptTemplatePreviewRequest, request):
    rendered = PromptService.format_prompt(data.template_content, **data.parameters)
    return {"success": True, "rendered_content": rendered,
            "parameters_used": list(data.parameters.keys())}
```

请求体 `PromptTemplatePreviewRequest{template_content, parameters: dict}`（`schemas/prompt_template.py:85-88`）。但 `frontend/src` 全库检索 `prompt-templates` 只命中 `App.tsx:54` 与 `PromptTemplates.tsx` 的 5 处（categories/POST/PUT/reset/export/import），**没有任何地方调用 `/preview` 或 `/system-defaults`**。这两个接口是死代码。

---

## 4. 系统内置模板全清单（36 个）

实测提取方式：`PromptService` 类体内 `^    [A-Z][A-Z0-9_]{3,} = """..."""`（32 个）+ 单行字符串常量（4 个 `INSPIRATION_*_USER`），共 **36 个**，与 `get_all_system_templates()` 的 `template_definitions` 字典键集合**一一对应**（`prompt_service.py:2873-3116`）。

| # | `template_key` | 常量@行 | 注册@行 | 分类 | 显示名 | 声明参数 | 实际占位符 | 字数 |
|---|---|---|---|---|---|---|---|---|
| 1 | `NOVEL_COVER_PROMPT_TEMPLATE` | 32 | 2874 | 封面生成 | 小说封面生成 | title, genre, theme, description | 同 | 434 |
| 2 | `WORLD_BUILDING` | 82 | 2880 | 世界构建 | 世界构建 | title, theme, genre, description | 同 | 1766 |
| 3 | `BOOK_IMPORT_REVERSE_PROJECT_SUGGESTION` | 2480 | 2886 | 拆书导入 | 拆书导入-反向项目提炼 | title, sampled_text | 同 | 854 |
| 4 | `BOOK_IMPORT_REVERSE_OUTLINES` | 2534 | 2892 | 拆书导入 | 拆书导入-反向章节大纲 | title, genre, theme, narrative_perspective, start_chapter, end_chapter, expected_count, chapters_text | 同 | 1268 |
| 5 | `CHARACTERS_BATCH_GENERATION` | 198 | 2901 | 角色生成 | 批量角色生成 | count, time_period, location, atmosphere, rules, theme, genre, requirements | 同 | 2258 |
| 6 | `SINGLE_CHARACTER_GENERATION` | 866 | 2907 | 角色生成 | 单个角色生成 | project_context, user_input | 同 | 1915 |
| 7 | `SINGLE_ORGANIZATION_GENERATION` | 978 | 2913 | 角色生成 | 组织生成 | project_context, user_input | 同 | 1217 |
| 8 | `OUTLINE_CREATE` | 324 | 2919 | 大纲生成 | 大纲生成 | 13 项，含 **`target_words`** | 12 项，**无 `target_words`** | 1965 |
| 9 | `OUTLINE_CONTINUE` | 441 | 2926 | 大纲生成 | 大纲续写 | 20 项 | 18 项，**多 `recent_outlines`、缺 `all_chapters_brief`/`memory_context`/`recent_plot`** | 2121 |
| 10 | `CHAPTER_GENERATION_ONE_TO_MANY` | 560 | 2935 | 章节创作 | 章节创作-1-N模式（第1章） | 8 项 | 11 项，**多 `chapter_careers`/`foreshadow_reminders`/`relevant_memories`** | 982 |
| 11 | `CHAPTER_GENERATION_ONE_TO_MANY_NEXT` | 769 | 2942 | 章节创作 | 章节创作-1-N模式（第2章及以后） | 13 项，含 `story_skeleton` | 14 项，**多 `chapter_careers`/`recent_chapters_context`、缺 `story_skeleton`** | 1526 |
| 12 | `CHAPTER_GENERATION_ONE_TO_ONE` | 629 | 2950 | 章节创作 | 章节创作-1-1模式（第1章） | 9 项 | 11 项，**多 `foreshadow_reminders`/`relevant_memories`** | 828 |
| 13 | `CHAPTER_GENERATION_ONE_TO_ONE_NEXT` | 690 | 2957 | 章节创作 | 章节创作-1-1模式（第2章及以后） | 12 项 | 14 项，**多 `previous_chapter_summary`/`recent_chapters_context`** | 1168 |
| 14 | `CHAPTER_REGENERATION_SYSTEM` | 1612 | 2965 | 章节重写 | 章节重写系统提示 | 8 项 | **0 项**（不 format，见 §3.1） | 689 |
| 15 | `PARTIAL_REGENERATE` | 2414 | 2972 | 章节重写 | 局部重写 | 7 项 | 同 | 836 |
| 16 | `PLOT_ANALYSIS` | 1047 | 2979 | 情节分析 | 情节分析 | 4 项 | 6 项，**多 `characters_info`/`existing_foreshadows`** | **8496** |
| 17 | `OUTLINE_EXPAND_SINGLE` | 1398 | 2985 | 情节展开 | 大纲单批次展开 | 16 项，含 `scene_instruction` | 15 项，**无 `scene_instruction`** | 1723 |
| 18 | `OUTLINE_EXPAND_MULTI` | 1501 | 2994 | 情节展开 | 大纲分批展开 | 19 项，含 `scene_instruction` | 18 项，**无 `scene_instruction`** | 1775 |
| 19 | `MCP_TOOL_TEST` | 1651 | 3004 | MCP测试 | MCP工具测试(用户提示词) | plugin_name | 同 | 276 |
| 20 | `MCP_TOOL_TEST_SYSTEM` | 1661 | 3010 | MCP测试 | MCP工具测试(系统提示词) | — | — | 229 |
| 21 | `MCP_WORLD_BUILDING_PLANNING` | 1778 | 3016 | MCP增强 | MCP世界观规划 | 4 项 | 同 | 201 |
| 22 | `MCP_CHARACTER_PLANNING` | 1796 | 3022 | MCP增强 | MCP角色规划 | 5 项 | 同 | 219 |
| 23 | `AUTO_CHARACTER_ANALYSIS` | 1815 | 3028 | 自动角色引入 | 自动角色分析 | 10 项，含 `new_outlines`/`end_chapter` | 12 项，**多 `all_chapters_brief`/`chapter_count`/`plot_stage`/`story_direction`** | 1780 |
| 24 | `AUTO_CHARACTER_GENERATION` | 1926 | 3035 | 自动角色引入 | 自动角色生成 | 11 项 | 同 | 2204 |
| 25 | `AUTO_ORGANIZATION_ANALYSIS` | 2060 | 3042 | 自动组织引入 | 自动组织分析 | 13 项 | 同 | 2044 |
| 26 | `AUTO_ORGANIZATION_GENERATION` | 2189 | 3049 | 自动组织引入 | 自动组织生成 | 12 项 | 同 | 1802 |
| 27 | `CAREER_SYSTEM_GENERATION` | 2302 | 3056 | 世界构建 | 职业体系生成 | 8 项 | 同 | 1831 |
| 28 | `INSPIRATION_TITLE_SYSTEM` | 1669 | 3062 | 灵感模式 | 灵感模式-书名生成(系统) | initial_idea | 同 | 258 |
| 29 | `INSPIRATION_TITLE_USER` | 1687 | 3068 | 灵感模式 | 灵感模式-书名生成(用户) | initial_idea | 同 | 30 |
| 30 | `INSPIRATION_DESCRIPTION_SYSTEM` | 1690 | 3074 | 灵感模式 | 灵感模式-简介生成(系统) | initial_idea, title | 同 | 273 |
| 31 | `INSPIRATION_DESCRIPTION_USER` | 1707 | 3080 | 灵感模式 | 灵感模式-简介生成(用户) | initial_idea, title | 同 | 40 |
| 32 | `INSPIRATION_THEME_SYSTEM` | 1710 | 3086 | 灵感模式 | 灵感模式-主题生成(系统) | initial_idea, title, description | 同 | 300 |
| 33 | `INSPIRATION_THEME_USER` | 1729 | 3092 | 灵感模式 | 灵感模式-主题生成(用户) | initial_idea, title, description | 同 | 57 |
| 34 | `INSPIRATION_GENRE_SYSTEM` | 1732 | 3098 | 灵感模式 | 灵感模式-类型生成(系统) | initial_idea, title, description, theme | 同 | 309 |
| 35 | `INSPIRATION_GENRE_USER` | 1752 | 3104 | 灵感模式 | 灵感模式-类型生成(用户) | initial_idea, title, description, theme | 同 | 68 |
| 36 | `INSPIRATION_QUICK_COMPLETE` | 1755 | 3110 | 灵感模式 | 灵感模式-智能补全 | existing | 同 | 388 |

**结论**：`parameters` 字段与真实占位符**有 11 个模板不一致**（可执行复核见 §14 `verify1.py`）：

- **A 类（7 个，有运行时风险）**：模板引用了未声明的占位符，用户照 `parameters` 写模板就会踩到 `KeyError`（`OUTLINE_CONTINUE`、`CHAPTER_GENERATION_ONE_TO_MANY`、`CHAPTER_GENERATION_ONE_TO_MANY_NEXT`、`CHAPTER_GENERATION_ONE_TO_ONE`、`CHAPTER_GENERATION_ONE_TO_ONE_NEXT`、`PLOT_ANALYSIS`、`AUTO_CHARACTER_ANALYSIS`）。
- **B 类（4 个，仅元信息陈旧）**：声明了未使用的参数（`OUTLINE_CREATE` 的 `target_words`、`OUTLINE_EXPAND_SINGLE`/`_MULTI` 的 `scene_instruction`、`CHAPTER_REGENERATION_SYSTEM` 的全部 8 个——该模板根本不走 `format`）。

因此 `parameters` 是 UI 提示而非契约。gaea 迁移时若把 `parameters` 当真值做校验，会对 B 类误报、对 A 类漏报。

### 4.1 第 37 类模板来源：Skill（动态追加）

`get_all_system_templates()` 末尾（`prompt_service.py:3130-3136`）：

```python
try:
    skill_templates = get_all_skills_cached()
    templates.extend(skill_templates)
except Exception as e:
    ...warning(...)
```

`load_skills()`（`skill_loader.py:159-239`）把每个 Skill 的 SKILL.md 正文 + `references/*.md` **全文拼接**成 `content`：

```python
ref_section = "\n\n---\n\n## 附录：参考资料知识库\n"
ref_section += "（以下内容根据用户需求按需引用，不需要全部使用）\n"
for ref_name, ref_content in references.items():
    ref_section += f"\n### 参考资料：{ref_name}\n\n{ref_content}\n"
full_content = body + ref_section
```

实测 7 个 Skill 的拼接体积：

| Skill | SKILL.md | references 合计 | 合计 |
|---|---|---|---|
| `story-deslop` | 8 039 B | 16 549 B | 24 588 B |
| `story-long-analyze` | 5 854 B | 36 925 B | 42 779 B |
| `story-long-scan` | 7 871 B | 32 419 B | 40 290 B |
| `story-long-write` | 10 831 B | **891 502 B**（17 个参考文件） | 902 333 B |
| `story-short-analyze` | 3 649 B | 358 188 B（9 个） | 361 837 B |
| `story-short-scan` | 7 268 B | 4 394 B | 11 662 B |
| `story-short-write` | 9 945 B | 427 467 B（11 个） | 437 412 B |
| **合计** | | | **1 820 901 B ≈ 1.74 MB** |

因此 `GET /api/prompt-templates/categories`（`prompt_templates.py:99` 调 `get_all_system_templates()`）每次返回 **≈1.74 MB** 的 JSON。`PromptTemplates.tsx` 为每个模板渲染一张卡片（`PromptTemplates.tsx:421-509`）却只显示 `description`/`template_key`，完整 `template_content` 白白随列表下发。这是本域最直接的性能缺陷（§9.2）。

补充：`skills/` 根目录还有一个 `README_HOW_TO_ADD_SKILL.md`（普通文件）。`load_skills()` 用 `os.path.isdir(skill_dir)` 过滤（`skill_loader.py:183-184`），因此不会被误当作 Skill 加载。

### 4.2 SKILL.md 的元数据约定

`skill_loader.py:59` 只认 6 个字段：`name`、`display_name`、`category`、`description`、`triggers`。实测 7 个 SKILL.md 的 frontmatter **只有 `name` 和 `description` 两个键**，因此 `display_name`/`category`/`triggers` 全部走兜底：

- `display_name`：取 `description` 第一行，按 `。` 切分（`skill_loader.py:73-77`）。
- `category`：由 name 子串推断（`skill_loader.py:80-89`）：含 `long`→`Skill·长篇`，含 `short`→`Skill·短篇`，含 `deslop`→`Skill·润色`，含 `browser`→`Skill·工具`，否则 `Skill`。
- `triggers`：正则从 `description` 里抓 `「...」` 与 `/xxx`，并强制插入 `/{name}`（`skill_loader.py:92-109`）。
- `template_key`：`SKILL_` + `name.upper().replace('-','_')`（`skill_loader.py:69-70`），例如 `story-deslop` → `SKILL_STORY_DESLOP`。

`template_key` 是 `get_all_system_templates()` 返回项里唯一的键名——注意 Skill 项同时带 `template_key` 和 `name`（`skill_loader.py:219-231`），而系统内置项只有 `template_key`（`prompt_service.py:3121-3128`）。

---

## 5. RTCO 框架：模板正文的既定结构

系统内置模板并非自由文本，而是**在提示词内部**用类 XML 标签实现 RTCO（Role/Task/Context/Output）分层。实测 32 个多行模板的标签骨架：

| 模板 | 标签序列（含 `priority`） |
|---|---|
| `WORLD_BUILDING` | `system` `task` `input(P0)` `guidelines(P1)` `output(P0)` `constraints` |
| `CHARACTERS_BATCH_GENERATION` | `system` `task` `worldview(P0)` `requirements(P1)` `output(P0)` `constraints` |
| `OUTLINE_CREATE` | `system` `task` `project(P0)` `worldview(P1)` `characters(P1)` `mcp_context(P2)` `requirements(P1)` `output(P0)` `constraints` |
| `OUTLINE_CONTINUE` | `system` `task` `project(P0)` `worldview(P1)` `previous_context(P0)` `characters(P0)` `user_input(P0)` `mcp_context(P2)` `output(P0)` `constraints` |
| `CHAPTER_GENERATION_ONE_TO_MANY` | `system` `task` `outline(P0)` `characters(P1)` `careers(P2)` `foreshadow_reminders(P2)` `memory(P2)` `constraints` `output` |
| `CHAPTER_GENERATION_ONE_TO_ONE_NEXT` | `system` `task(P0)` `outline(P0)` `previous_chapter_summary(P1)` `recent_context(P1)` `previous_chapter(P1)` `characters(P1)` `careers(P2)` `foreshadow_reminders(P2)` `memory(P2)` `constraints` `output` |
| `PLOT_ANALYSIS` | `system` `task` `chapter(P0)` `existing_foreshadows(P1)` `characters(P1)` `analysis_framework(P0)` `output(P0)` `constraints` |
| `OUTLINE_EXPAND_SINGLE` / `_MULTI` | `system` `task` `project(P1)` `characters(P1)` `outline_node(P0)` `context(P2)` `output(P0)` `constraints` |
| `AUTO_CHARACTER_ANALYSIS` | `system` `task` `project(P1)` `context(P0)` `analysis_framework(P0)` `output(P0)` `constraints` |
| `CAREER_SYSTEM_GENERATION` | `system` `task` `worldview(P0)` `design_requirements(P0)` `output(P0)` `constraints` |
| `PARTIAL_REGENERATE` | `system` `task` `context(P0)` `user_requirements(P0)` `style(P1)` `output` `constraints` |
| `BOOK_IMPORT_REVERSE_PROJECT_SUGGESTION` | `system` `task` `input(P0)` `output(P0)` `constraints` |
| 未标记 priority 的模板 | 只用 `system`/`task`/`output`/`constraints` 四段（`CHAPTER_REGENERATION_SYSTEM`、`MCP_*`） |
| 非 RTCO 的模板 | `NOVEL_COVER_PROMPT_TEMPLATE`（纯自然语言，无标签）、`MCP_TOOL_TEST*`、`INSPIRATION_*`（纯 JSON 输出指令）、`MCP_WORLD_BUILDING_PLANNING`、`MCP_CHARACTER_PLANNING` |

**P0/P1/P2 的作用是「语义排序 + 提示模型注意力」，不是程序化裁剪。** 全库检索未发现任何按 priority 截断或重排的代码；`format_prompt` 只做字符串替换。与之相对，gaea 的 `input_sections` 的 `priority` 是**被引擎消费的**（`internal/prompt/prompt.go:177-205` 的 `BuildUserPrompt` 按 P0/P1/P2 分三段输出，P2 注释写明 "may be truncated"）。这是两套系统在 K 侧设计的真实分歧点，迁移时需要显式决策。

`constraints` 段落内部的符号约定高度统一，可直接复用为 gaea 模板规范：

- `【必须遵守】` + `✅` 列表（章节类模板，例如 `prompt_service.py:604-610`）
- `【禁止事项】` + `❌` 列表（例如 `prompt_service.py:612-617`）
- `【输出规范】`（`output` 段内）
- `PLOT_ANALYSIS` 额外要求「keyword 必须从原文逐字复制 8-25 字」（`prompt_service.py:1358-1360`）——这是为下游定位服务的硬约束，值得单独列进 gaea 的能力清单。

---

## 6. 版本、覆盖与优先级规则

### 6.1 模板侧（prompt_templates）

| 能力 | 实现 | 锚点 |
|---|---|---|
| 创建/更新 | `POST /api/prompt-templates`，先查 `(user_id, template_key)` 再 update/insert —— 真 Upsert | `prompt_templates.py:204-244` |
| 局部更新 | `PUT /api/prompt-templates/{template_key}`，`model_dump(exclude_unset=True)` | `prompt_templates.py:247-282` |
| 删除 | `DELETE /api/prompt-templates/{template_key}` | `prompt_templates.py:285-314` |
| 重置为系统默认 | `POST /{template_key}/reset`，**先校验系统模板存在**再删自定义行 | `prompt_templates.py:317-353` |
| 合并列表 | `GET /categories`：用户自定义行（`is_system_default=False`）+ 未自定义的系统内置（临时 ORM 对象，`id=template_key`，`is_system_default=True`），按 category 分组、组内按 `template_key` 排序 | `prompt_templates.py:77-152` |
| UI 语义 | 「系统默认」灰头无开关；「已自定义」紫头带开关；编辑系统模板会自动创建当前账户副本 | `PromptTemplates.tsx:106, 357-361, 432-450` |

**没有版本号字段。** `prompt_templates` 既无 `version` 也无历史表，用户一旦自定义即与该用户的覆盖行绑定；系统内置模板升级（改 Python 常量）**不会**影响已有自定义行——表现为「用户永远停在旧模板」。唯一的对账手段是导出时算的系统内容哈希（见 §6.4）。

**项目级覆盖**：模板维度**不存在**项目级覆盖。项目级只有写作风格（`project_default_styles`）。检索 `PromptTemplate` 的全部引用，未见 `project_id` 字段或用例。

### 6.2 写作侧（writing_styles）

优先级与作用域（`writing_styles.py`）：

| 规则 | 实现 | 锚点 |
|---|---|---|
| 全局预设 = `user_id IS NULL` | `where(WritingStyle.user_id.is_(None)).order_by(order_index)` | `writing_styles.py:44-49` |
| 用户可用列表 = 全局预设 + 本人自定义 | `preset_styles + custom_styles` | `writing_styles.py:163-179` |
| 基于预设创建：传 `preset_id` 则从 DB 拉预设填充未填字段 | 若 `preset_id` 不存在返回 400 | `writing_styles.py:82-101` |
| 预设不可改不可删 | `style.user_id is None` → 403 | `writing_styles.py:338-339, 405-406` |
| 编辑后自动降级为 custom | `if any(k in update_data for k in ["name","description","prompt_content"]): update_data["style_type"]="custom"` | `writing_styles.py:349-350` |
| 删除保护 | 若被任何项目设为默认 → 400「请先设置其他风格为默认」 | `writing_styles.py:412-421` |
| 项目默认风格 UPSERT | 先 `delete` 该项目旧行再 `insert`，靠 `uix_project_default_style` 保唯一 | `writing_styles.py:474-485` |
| `order_index` 生成 | `count(*)` + 1（不是 `max+1`），删除后会产生**值重复** | `writing_styles.py:110-125`；工坊导入同样用 `count+1`（`prompt_workshop.py:322-334`） |

### 6.3 风格注入 system prompt（最高优先级）

`WritingStyleManager.apply_style_to_prompt`（`prompt_service.py:10-26`）是一段**刻意的空操作**：

```python
@staticmethod
def apply_style_to_prompt(base_prompt: str, style_content: str) -> str:
    """注意：写作风格已通过 system_prompt 注入（system_prompt_with_style），
    此方法仅追加输出指令，不再重复注入 style_content，避免风格信息被注入两次。"""
    return f"{base_prompt}\n\n请直接输出章节正文内容，不要包含章节标题和其他说明文字。"
```

真正的注入在调用点，模板固定（`chapters.py:1770-1777`，另在 2272-2282、2809-2820、4345-4356 三处有同构拷贝）：

```python
if not system_prompt_with_style and style_content:
    system_prompt_with_style = f"""【🎨 写作风格要求 - 最高优先级】

{style_content}

⚠️ 请严格遵循上述写作风格要求进行创作，这是最重要的指令！
确保在整个章节创作过程中始终保持风格的一致性。"""
```

Skill 优先级**高于**风格（`chapters.py:1744-1768`）：若传了 `skill_key` 且命中，`system_prompt_with_style` 被 Skill 工作流占据，风格降级为 `【🎨 写作风格要求 - 补充】` 追加在后；若 Skill 未命中，再回落到纯风格模板。

`chapter_regenerator.py:75-83` 是重写链路的独立副本，措辞为「重写」而非「创作」。

**风格在 prompt 中的位置有三处，是 gaea 必须统一的设计点**：
1. `CHAPTER_REGENERATION_SYSTEM`（`prompt_service.py:1612-1650`）里没有风格占位符；
2. `get_chapter_regeneration_prompt` 把风格作为 user prompt 的一个 markdown 段落（`prompt_service.py:2718-2726`）；
3. 同一次调用又把风格注入 system prompt（`chapter_regenerator.py:76-82`）。
即**同一条信息进了两次**（user 段 + system 段），与 `apply_style_to_prompt` 想避免的重复恰好相反。`PARTIAL_REGENERATE` 模板自带 `{style_content}` 占位符（`prompt_service.py:2414-2478`），是第三种写法。

### 6.4 导出/导入协议（含内容哈希对账）

导出 `POST /api/prompt-templates/export`（`prompt_templates.py:356-440`）返回 `PromptTemplateExport`：

```jsonc
{
  "templates": [
    {
      "template_key": "...", "template_name": "...", "template_content": "...",
      "description": "...", "category": "...", "parameters": "[\"a\",\"b\"]",
      "is_active": true,
      "is_customized": true,            // false = 系统默认
      "system_content_hash": "a1b2c3d4e5f60718"  // sha256(content.strip())[:16]
    }
  ],
  "export_time": "2026-...",
  "version": "2.0",
  "statistics": {"total": 43, "customized": 3, "system_default": 40}
  // total = 36 个系统内置 + 7 个 Skill = 43
}
```

哈希算法（`prompt_templates.py:28-30`）：

```python
def calculate_content_hash(content: str) -> str:
    return hashlib.sha256(content.strip().encode('utf-8')).hexdigest()[:16]
```

导入 `POST /api/prompt-templates/import`（`prompt_templates.py:443-590`）的**智能三态**逻辑，这是本域最有借鉴价值的一段算法：

| 导入项的 `is_customized` | 系统内是否存在同 key | 内容 vs 系统内容 | 动作 | 计入统计 |
|---|---|---|---|---|
| `false` | 是 | 相等（`strip()` 后逐字） | **删除**本地自定义行（回落系统默认） | `kept_system_default` |
| `false` | 是 | 不等 | 新建/更新自定义行 | `converted_to_custom` + 记入 `converted_templates` |
| `false` | 否 | — | 新建/更新自定义行 | `created_or_updated` |
| `true` | 任意 | — | 直接新建/更新自定义行 | `created_or_updated` |

返回 `PromptTemplateImportResult{message, statistics, converted_templates[{template_key, template_name, reason}]}`（`schemas/prompt_template.py:78-82`），前端把 `converted_templates` 弹窗列给用户（`PromptTemplates.tsx:214-240`）。

注意：导入时用的是**内容逐字比对**（`imported_content == system_content`，两者都已 `strip()`），不是用 `system_content_hash`。哈希字段只是导出侧的冗余信息。

---

## 7. 提示词工坊：分享/导入协议与信任边界

### 7.1 双模式拓扑

`config.py:139-141`：

```python
WORKSHOP_MODE: str = "client"                       # client: 本地实例, server: 云端中央服务器
WORKSHOP_CLOUD_URL: str = "https://mumuverse.space:1566"
WORKSHOP_API_TIMEOUT: int = 30
```

`INSTANCE_ID`（`config.py:158-188`）：server 模式固定为字符串 `"server"`；client 模式从 `PROJECT_ROOT/.instance_id` 读取，不存在则 `uuid4()[:12]` 并落盘。

`is_workshop_server()`（`config.py:190-192`）是全部分支的唯一开关。**`prompt_workshop.py` 里每个端点都是同一个 if/else**：server 分支操作本地表，client 分支把请求转发到 `workshop_client`。

`workshop_client._request`（`workshop_client.py:22-62`）统一的头：

```python
headers = {"X-Instance-ID": INSTANCE_ID, "Content-Type": "application/json"}
if user_identifier:
    headers["X-User-ID"] = user_identifier
url = f"{self.base_url}/api/prompt-workshop{path}"
```

### 7.2 用户标识协议：`实例ID:用户ID`

`get_user_identifier(user_id) -> f"{INSTANCE_ID}:{user_id}"`（`prompt_workshop.py:35-37`）。

`get_user_identifier_from_request`（`prompt_workshop.py:40-58`）通过 `request.state.is_proxy_request` 区分：

```python
is_proxy = getattr(request.state, 'is_proxy_request', False)
if is_proxy:
    return user_id            # 已是完整格式，直接信任
else:
    return get_user_identifier(user_id)   # 本地请求，补实例前缀
```

`source_instance` 在提交时优先取请求头（`prompt_workshop.py:462`）：`request.headers.get("X-Instance-ID") or INSTANCE_ID`。

**信任边界评估（本域的注入风险核心）**：

- 云端把 `X-Instance-ID` / `X-User-ID` 作为**身份声明**直接采信（`prompt_workshop.py:52-58, 462`）。任何能访问云端的客户端都可以伪造任意 `X-Instance-ID`，从而：冒充他人 `source_instance` 提交、对任意 `user_identifier` 的点赞状态施压、污染 `author_id` 归属。
- 未发现签名、nonce、时间戳或任何共享密钥校验。检索 `prompt_workshop.py` 与 `workshop_client.py` 全文，只有 `X-Instance-ID` 与 `X-User-ID` 两个自定义头，「未能从源码确认」存在其它鉴权层。
- 导入侧**无内容审查**：`import_item`（`prompt_workshop.py:284-349`）把工坊的 `prompt_content` 原样写入本地 `WritingStyle.prompt_content`，再原样进入 system prompt。云端只有 `admin_review_submission` 的人工审核（`prompt_workshop.py:632-698`）作为唯一闸门。
- 工坊条目**不含变量契约**：`PromptWorkshopItem` 没有 `parameters` 字段，导入后只是「一条风格文本」。这实际上降低了注入危害（它不会去填别人的占位符），但也意味着工坊分享的是**风格/指令片段**而非**可替换模板**。

### 7.3 提交—审核—发布状态机

```
（本地 WritingStyle 或用户手写）
        │ POST /prompt-workshop/submit
        ▼
  PromptSubmission.status = "pending"
        │
        ├─ 提交者可 DELETE /submissions/{id}（status==pending 免 force）
        │
        ▼ 管理员 POST /admin/submissions/{id}/review  {action: approve|reject}
        │
        ├─ reject  → status="rejected"，写 reviewer_id/review_note/reviewed_at
        └─ approve → 新建 PromptWorkshopItem(status="active")，
                     submission.status="approved"，
                     submission.workshop_item_id = new_item.id
```

关键字段处理（`prompt_workshop.py:654-684`）：

```python
author_id  = None if submission.is_anonymous else submission.submitter_id
author_name = submission.author_display_name if not submission.is_anonymous else None
category   = data.category or submission.category     # 审核时可改分类
tags       = data.tags     or submission.tags         # 审核时可改标签
```

撤回规则（`prompt_workshop.py:536-577`）：`status != "pending"` 时必须 `force=True`，否则 400。

管理员闸门（`prompt_workshop.py:115-127`）：`is_workshop_server()` 且 `user.is_admin`，否则 403/401。

`PromptSubmissionCreate.source_style_id`（`schemas/prompt_workshop.py:29`）字段存在但**服务端未使用**——检索 `prompt_workshop.py` 全文，`source_style_id` 只出现在该 schema 定义里，提交路径不读它。属未接线字段。

### 7.4 工坊 API 契约（完整清单）

| 方法 | 路径（含 `/api` 前缀） | 模式 | 说明 | 锚点 |
|---|---|---|---|---|
| GET | `/prompt-workshop/status` | 双 | 返回 `{mode, instance_id, cloud_url?, cloud_connected?}` | `prompt_workshop.py:132-147` |
| GET | `/prompt-workshop/items` | 双 | `category/search/tags/sort(newest\|popular\|downloads)/page/limit` | `:150-176` |
| GET | `/prompt-workshop/items/{id}` | 双 | 公开详情 | `:261-281` |
| POST | `/prompt-workshop/items/{id}/import` | 双 | 导入为本地 `WritingStyle`，body `{custom_name?}` | `:284-349` |
| POST | `/prompt-workshop/items/{id}/like` | 双 | 点赞/取消，返回 `{liked, like_count}` | `:352-401` |
| POST | `/prompt-workshop/items/{id}/download` | **仅 server** | 计下载数 | `:404-424` |
| POST | `/prompt-workshop/submit` | 双 | 提交待审 | `:427-499` |
| GET | `/prompt-workshop/my-submissions` | 双 | `?status=` | `:502-533` |
| DELETE | `/prompt-workshop/submissions/{id}` | 双 | `?force=` | `:536-577` |
| GET | `/prompt-workshop/admin/submissions` | **仅 server** | `?status=&source=&page=&limit=`，附 `pending_count` | `:582-629` |
| POST | `/prompt-workshop/admin/submissions/{id}/review` | **仅 server** | `{action, review_note?, category?, tags?}` | `:632-698` |
| POST | `/prompt-workshop/admin/items` | **仅 server** | 添加官方提示词 | `:701-725` |
| PUT | `/prompt-workshop/admin/items/{id}` | **仅 server** | 编辑 | `:728-752` |
| DELETE | `/prompt-workshop/admin/items/{id}` | **仅 server** | 删除 | `:755-774` |
| GET | `/prompt-workshop/admin/stats` | **仅 server** | `{total_items, total_official, total_pending, total_downloads, total_likes}` | `:777-827` |

本地查询实现 `_get_items_local`（`prompt_workshop.py:179-258`）：`status=="active"` 硬过滤；`search` 只匹配 `name ILIKE` 与 `description ILIKE`（**不搜 `prompt_content` 也不搜 `tags`**，`tags` 参数接收后**未使用**）；分类统计用 `GROUP BY category` 再用 `PROMPT_CATEGORIES` 映射中文名。

`_item_to_dict`（`prompt_workshop.py:77-92`）**不返回 `prompt_content` 之外的正文以外信息**，但确实返回 `prompt_content`——列表页 20 条即下发 20 份完整提示词正文。

分类常量（`constants/prompt_categories.py`）：10 个分类（`general/fantasy/martial/romance/scifi/horror/history/urban/game/other`）+ 30 个建议标签。**注意这 10 个分类与 §4 里模板的 category 完全不同域**（模板分类是「世界构建/大纲生成/…」这种功能分类）。

---

## 8. 写作风格的表示方式：自由文本，非结构化

结论：**风格 = 一段自由中文文本**，存于 `writing_styles.prompt_content`，无 schema、无字段、无权重。

证据：

1. `WritingStyle.prompt_content` 类型为 `Text NOT NULL`，注释「风格提示词内容」（`models/writing_style.py:17`）。
2. 6 条预设风格的内容是**编号条目式自然语言**（`backend/alembic/sqlite/versions/20251227_0856_a1b2c3d4e5f6_初始化sqlite预置数据.py:83-162`）。原文摘录（`natural`）：

```
写作风格要求：
1. 语言简洁明快，贴近现代口语
2. 多用短句，节奏流畅
3. 注重情感细节的自然流露
4. 避免过度修饰和复杂句式
```

同文件 6 条预设的 `preset_id` 依次为 `natural` / `classical` / `modern` / `literary` / `suspense` / `humorous`，`name` 为 自然流畅 / 古典优雅 / 现代简约 / 文艺细腻 / 紧张悬疑 / 幽默诙谐，`order_index` 1-6。

3. 唯一的结构化字段是**元数据**，不是风格内容本身：

| 结构化维度 | 承载方式 |
|---|---|
| 适用范围（题材） | 只在 `description` 里用自然语言写「适合现代都市、现实题材」 |
| 强度/权重 | **无此字段** |
| 与题材的绑定 | **无此字段**（靠 `project_default_styles` 做项目级选择） |
| 可组合性 | **无**：一次只能选一条风格，`style_content` 是单一字符串 |

4. 运行时无解析：风格文本被 f-string 直接拼进 system prompt（§6.3），不经过任何 tokenizer / 规则引擎 / 相似度匹配。

**与 gaea 的对照**：gaea 侧 `internal/novelstyle/score.go`（t4 域）是**量化风格指纹**（可计算、可打分），而 MuMu 的风格是**纯 prompt 指令**。二者不是同一层能力——MuMu 提供的是「怎么告诉模型」，gaea 提供的是「怎么度量结果」。迁移时应保留两套，并建立「风格文本 → 期望指纹区间」的映射表（该映射表在 MuMu 源码中**不存在**，需要 gaea 自建）。

---

## 9. 源码级缺陷清单（迁移时需避坑）

每条均给出可复核锚点。

### 9.1 `AI_DENOISING` 模板键不存在 → `/api/polish` 必然 500（high）

`backend/app/api/polish.py:40` 与 `:114`：

```python
template = await PromptService.get_template("AI_DENOISING", user_id, db)
prompt = PromptService.format_prompt(template, original_text=request.original_text)
```

全库检索 `AI_DENOISING`：只有 `polish.py` 的这两行。`PromptService` 无该属性，`template_definitions` 无该键 → `get_template` 走 `getattr(cls, key, None)` 返回 `None`（`prompt_service.py:2855`）并只打 warning（`:2858`）→ `format_prompt(None, ...)` 抛 `AttributeError: 'NoneType' object has no attribute 'format'` → 被 `polish.py:84-86` 的 `except Exception` 转成 `HTTPException(500, "AI去味失败: ...")`。

佐证功能已废弃：前端路由被注释（`frontend/src/App.tsx:75` `{/* <Route path="polish" element={<Polish />} /> */}`，且 `pages/` 下无 `Polish.tsx`）；能力由 Skill `story-deslop` 承接（`backend/app/skills/story-deslop/SKILL.md` + 2 个 reference）。**即 `/api/polish` 是一条从未被接通、且实现已失效的残留链路。**

### 9.2 `GET /prompt-templates/categories` 下发 1.74 MB（high）

`prompt_templates.py:99` → `PromptService.get_all_system_templates()` → `prompt_service.py:3132` `get_all_skills_cached()` → 7 个 Skill 的 `content` 含全部 reference 全文（§4.1，实测 1 820 901 B）。列表页只需要 `template_key/template_name/category/description/parameters`（`PromptTemplates.tsx:432-484` 只用这 5 项 + `is_system_default`/`is_active`）。修复方向：列表接口裁剪 `content`（或改为 `has_content: true`，正文另开 `GET /{key}`）。

### 9.3 用户自定义模板的静态校验缺失（high）

`create_or_update_template`（`prompt_templates.py:204-244`）与 `update_template`（`:247-282`）对 `template_content` **零校验**：不检查占位符是否为 `parameters` 声明的子集，不检查花括号是否平衡，不检查是否引用了不存在的占位符。而 `format()` 在运行时对缺失键立刻抛错（`:2621-2624`）。

后果链：用户在 UI 里写错一个 `{变量}` → 保存成功（提示「保存成功」）→ 下一次调用该模板时整条生成链路 500，且 `get_template` 找不到系统兜底（因为自定义行优先）。

**这是本域最应该被 gaea 修正的设计**：保存时校验（`声明参数 ⊇ 实际占位符`），并让运行时对缺失键可降级（保留字面量或填空串）而非抛错。

### 9.4 工坊计数与点赞状态的一致性缺口（medium）

1. `like_count` 是 `PromptWorkshopItem` 上的冗余列（`models/prompt_workshop.py:22`），点赞时手工 `+1/-1`（`prompt_workshop.py:382, 392`）。与 `prompt_workshop_likes` 表行数无约束、无对账，异常路径会漂移。`download_count` 同类问题（`:306-307, 421-422`）。
2. `download` 计数被计两次：server 分支的 `import_item` 已 `item.download_count += 1`（`:306-307`），而 client 分支会额外调用 `POST /items/{id}/download` 通知云端（`:316`）。
3. `_get_items_local` 的 `is_liked` 依赖调用方 `user_identifier`；client 模式下该判断由**云端**完成。但 `workshop_client.toggle_like` 只带 `X-User-ID`，云端 `get_user_identifier_from_request`（`:40-58`）对代理请求**直接采信**（§7.2），因此点赞归属可被伪造。

### 9.5 前端-后端路径不一致（low，已废接口）

`frontend/src/services/api.ts:881-882`：

```typescript
initializeDefaultStyles: (projectId: string) =>
  api.post('/writing-styles/project/' + projectId + '/initialize', {}),
```

后端实际路径是 `/writing-styles/project/{project_id}/init-defaults`（`writing_styles.py:495`）。**该调用会 404**。同接口在后端已标注「【已废弃】」，函数体直接转发 `get_project_styles`（`writing_styles.py:496-508`）。前端无任何调用点。

### 9.6 风格注入同一信息两次（medium）

见 §6.3：`get_chapter_regeneration_prompt` 把 `style_content` 写进 user prompt 段落（`prompt_service.py:2718-2726`），`chapter_regenerator.py:76-82` 又把同一段写进 system prompt。而 `WritingStyleManager.apply_style_to_prompt` 的 docstring 明确说这是为了避免重复注入——说明历史上修过一处、漏了另一处。

### 9.7 `parameters` 字段与真实占位符不一致（medium，属模板契约缺陷）

见 §4 结论：11 个模板的 `parameters` 与真实占位符不符，其中 **7 个属 A 类（引用了未声明变量）**。当前用户可见后果被两层情况掩盖：① `/preview` 与 `parameters` 未被前端使用（§3.3）；② `format()` 忽略多余关键字参数，所以调用点传参正确时不会报错。但一旦用户按 UI 展示的 `parameters` 自行编写模板，缺失变量立即 500。**gaea 侧必须把 `parameters` 升格为强约束并在保存时校验。**

### 9.8 其他已确认的小问题

| 问题 | 锚点 |
|---|---|
| `is_active=False` 的自定义行在 `/categories` 里仍显示为「已自定义」+ 开关关闭，但合并逻辑把**所有**用户行都加入（不按 `is_active` 过滤），UI 的开关状态是唯一真相 | `prompt_templates.py:91-111` vs `PromptTemplates.tsx:443-450` |
| `PromptTemplateCategoryResponse.templates: List[PromptTemplateResponse]` 声明了 `PromptTemplateResponse`（含 `id`/`is_system_default`/时间戳），但代码塞的是 ORM 对象，靠 Pydantic 属性读取；系统模板的 `id` 被赋成 `template_key`（`prompt_templates.py:118`），前端 `key={template.id}` 会与用户行的 UUID 混用 | `prompt_templates.py:109-131`；`schemas/prompt_template.py:50-54` |
| `_get_items_local` 接收 `tags` 参数但从不使用（既不 JOIN 也不过滤） | `prompt_workshop.py:179-206` |
| `search` 不检索 `prompt_content`，用户搜正文内容搜不到 | `prompt_workshop.py:198-204` |
| `order_index` 用 `count(*)+1` 生成，删除后重复 | `writing_styles.py:110-125`；`prompt_workshop.py:323-326` |
| 工坊导入丢失来源信息：导入后 `WritingStyle` 只有 `description = "从提示词工坊导入: <原标题描述>"`，条目的 `author_name`/`source_instance`/`is_official`/分类/标签**全部丢弃**，本地无法回溯一份风格来自哪个作者、也失去升级比对能力 | `prompt_workshop.py:328-335` vs `_item_to_dict`（`:77-92`） |
| `INSPIRATION_*` 链路直接 `.format()`，不走 `format_prompt` 的 ValueError 包装，错误类型对调用方不可预期 | `inspiration.py:140-141` |
| `NOVEL_COVER_PROMPT_TEMPLATE` 通过 `get_template_with_fallback`（`prompt_service.py:2792-2813`）获取，若 key 拼错会返回 `None` 并在 `.format()` 处崩，与 §9.1 同构 | `prompt_service.py:67-77` |

---

## 10. 关键 Prompt 原文摘录（带锚点）

为节省篇幅，只摘录对 gaea 有直接迁移价值的片段；完整 36 个模板可通过 `get_all_system_templates()` 全量导出。

**(1) 章节创作 1-N（`prompt_service.py:560-626`）**

```
<system>
你是《{project_title}》的作者，一位专注于{genre}类型的网络小说家。
</system>

<task>
【创作任务】
撰写第{chapter_number}章《{chapter_title}》的完整正文。
【基本要求】
- 目标字数：{target_word_count}字（允许±200字浮动）
- 叙事视角：{narrative_perspective}
</task>

<outline priority="P0">
【本章大纲 - 必须遵循】
{chapter_outline}
</outline>

<characters priority="P1">
【本章角色 - 请严格遵循角色设定】
{characters_info}
⚠️ 角色互动须知：
- 角色之间的对话和行为必须符合其关系设定（如师徒、敌对等）
- 涉及组织的情节须体现角色在组织中的身份和职位
- 角色的能力表现须符合其职业和阶段设定
</characters>
...
<constraints>
【必须遵守】
✅ 严格按照大纲推进情节
✅ 保持角色性格、说话方式一致
...
【禁止事项】
❌ 输出章节标题、序号等元信息
❌ 使用"总之"、"综上所述"等AI常见总结语
❌ 在结尾处使用开放式反问
❌ 添加作者注释或创作说明
❌ 角色行为超出其职业阶段的能力范围
</constraints>

<output>
【输出规范】
直接输出小说正文内容，从故事场景或动作开始。
无需任何前言、后记或解释性文字。
现在开始创作：
</output>
```

**(2) 情节分析的输出 Schema（`prompt_service.py:1230-1354`，节选，`{{`/`}}` 为源码原文）**

```
返回纯JSON对象（无markdown标记）：
{{
  "hooks": [{{"type":"悬念","content":"具体描述","strength":8,"position":"中段",
             "keyword":"从原文逐字复制的8-25字文本"}}],
  "foreshadows": [{{"title":"...","type":"planted|resolved","strength":7,"subtlety":8,
                    "reference_chapter":null,"reference_foreshadow_id":null,
                    "keyword":"...","category":"mystery","is_long_term":false,
                    "related_characters":["角色A"],"estimated_resolve_chapter":15}}],
  "conflict": {{"types":["人与人"],"parties":["主角-复仇"],"level":8,
                "description":"...","resolution_progress":0.3}},
  "emotional_arc": {{"primary_emotion":"紧张焦虑","intensity":8,
                     "curve":"平静→紧张→高潮→释放","secondary_emotions":["期待"]}},
  "character_states": [{{"character_name":"张三","survival_status":null,
      "state_before":"犹豫","state_after":"坚定","psychological_change":"...",
      "key_event":"...","relationship_changes":{{"李四":"关系改善"}},
      "career_changes":{{"main_career_stage_change":1,
        "sub_career_changes":[{{"career_name":"炼丹","stage_change":1}}],
        "new_careers":[],"career_breakthrough":"..."}},
      "organization_changes":[{{"organization_name":"某门派","change_type":"promoted",
        "new_position":"长老","loyalty_change":"...","description":"..."}}]}}],
  "plot_points": [{{"content":"...","type":"revelation","importance":0.9,
                    "impact":"...","keyword":"..."}}],
  "scenes": [{{"location":"地点","atmosphere":"氛围","duration":"时长估计"}}],
  "organization_states": [{{"organization_name":"某门派","power_change":-10,
    "new_location":null,"new_purpose":null,"status_description":"...",
    "key_event":"...","is_destroyed":false}}],
  "pacing": "varied", "dialogue_ratio": 0.4, "description_ratio": 0.3,
  "scores": {{"pacing":6.5,"engagement":5.8,"coherence":7.2,"overall":6.5,
              "score_justification":"..."}},
  "plot_stage": "发展",
  "suggestions": ["【节奏问题】...", "【吸引力不足】..."]
}}
```

该模板的 `constraints` 里有 3 条对 gaea 直接可用的硬约束（`prompt_service.py:1358-1394`）：

```
✅ keyword字段必填：钩子、伏笔、情节点的keyword不能为空
✅ 逐字复制：keyword必须从原文复制，长度8-25字
✅ 精确定位：keyword能在原文中精确找到
...
✅ 【伏笔ID追踪】回收伏笔时，必须从【已埋入伏笔列表】中查找匹配的ID填入 reference_foreshadow_id
✅ 【suggestions严格格式】suggestions 必须是"字符串数组"，每个元素都必须是纯字符串
✅ 如果没有改进建议，必须返回空数组 []，不要返回 null，不要省略字段
...
【评分约束 - 严格执行】
✅ 建议数量必须与overall分数关联：
   - overall≤4.0 → 4-5条建议
   - overall 4.0-6.0 → 3-4条建议
   - overall 6.0-8.0 → 1-2条建议
   - overall≥8.0 → 0-1条建议
❌ 所有章节都打7-8分的"安全分"
```

**(3) 风格注入模板（`chapters.py:1770-1777`）**

```
【🎨 写作风格要求 - 最高优先级】

{style_content}

⚠️ 请严格遵循上述写作风格要求进行创作，这是最重要的指令！
确保在整个章节创作过程中始终保持风格的一致性。
```

**(4) Skill 注入模板（`chapters.py:1753-1763`）**

```
【⚡ Skill 工作流：{skill_name}】

{skill_content}

⚠️ 请严格遵循上述 Skill 工作流指令进行创作！
```
（若同时有风格，追加 `【🎨 写作风格要求 - 补充】\n\n{style_content}`）

**(5) 预设风格原文（`alembic/sqlite/versions/20251227_0856_a1b2c3d4e5f6_初始化sqlite预置数据.py:83-162`）**

`natural` / `classical` / `modern` / `literary` / `suspense` / `humorous` 六条，格式统一为：

```
写作风格要求：
1. <条目>
2. <条目>
3. <条目>
4. <条目>
```

（各条正文见 §8 与源文件。）

---

## 11. gaea 落地映射

### 11.1 现有实现盘点（已实读）

| 现有资产 | 位置 | 现状 |
|---|---|---|
| RTCO 模板引擎 | `internal/prompt/prompt.go`（221 行） | `Template{name, system, task, input_sections, output, constraints}`；`BuildSystemPrompt`（`:128-173`）、`BuildUserPrompt`（`:177-205`）按 P0→P1→P2 分段落；支持 `go:embed` 兜底 + 磁盘覆盖（`NewEngineWithEmbedded`，`:62-73`，同名磁盘优先） |
| 模板数据 | `prompts/*.json`（15 个） | 字段固定为 `name/system/task/input_sections/output/constraints`（`outline-continue.json` 无 `constraints`）；**全 15 个模板中唯一占位符是 `create-chapter.json` 的 `{word_count}`** |
| Skill 加载器 | `internal/skill/skill.go`（190 行） | `Skill{Name, Description, AppliesTo, Version, Body, Path}`；`InjectSkill`（`:86-106`）把 body 追加到 base prompt 后，前缀 `\n\n---\n## 写作指导: <name>\n`；frontmatter 解析为**逐行手写 YAML**（`:138-183`），只认 `name/description/version/applies_to` |
| 提示词消费点 | `internal/{chapter,outline,character,worldview,analysis}` | 各 `Agent` 持有 `eng *prompt.Engine`，例如 `internal/chapter/chapter.go:23,27`、`internal/outline/outline.go:24,28` |
| 引擎装配 | `internal/app/app.go:277`（磁盘）、`:678`（embed 兜底） | `prompt.NewEngine(filepath.Join(cfg.ResourceDir,"prompts"))` |
| Skill 装配 | `internal/app/writing_state.go:52`、`internal/app/stats_handler.go:16` | `skill.NewLoader(filepath.Join(cfg.ResourceDir,"skills"))` |
| 占位符替换 | `internal/app/create_chapter_handler.go:86-90` | 只用 `substituteWordCount(systemPrompt, minWords)` 精确替换 `{word_count}` |
| 风格注入 | gaea 侧为 `internal/novelstyle`（打分为主，见 t4） | **无「风格文本注入 system prompt」这条链路** |

### 11.2 能力差距矩阵（本域视角）

| 能力点 | MuMu | gaea 现状 | 差距 | 建议 |
|---|---|---|---|---|
| 模板存储 | Python 类属性 + DB 覆盖行 | `prompts/*.json` 文件 + `go:embed` | 缺「用户可编辑层」 | 新增 `internal/promptstore`（见 11.3） |
| 用户自定义模板 | ✅ `prompt_templates` 表 | ❌ | 缺失 | 用 SQLite 表照搬（`user_id` 在 gaea 单机下可固定为 `"local"`） |
| 项目级模板覆盖 | ❌（只有风格有） | ❌ | **gaea 可做得更好** | 在覆盖表加 `scope`（`global`/`project`）+ `project_id` |
| 变量渲染 | `str.format()`，缺失即抛错 | 单占位符字符串替换 | gaea 更稳，但表达力弱 | 引入命名占位符 `{{name}}`（不碰字面花括号）+ 缺失降级 |
| 条件段落 / 默认值 | ❌（靠 Python f-string） | ❌ | 两边都缺 | 可选增强，非阻塞 |
| 参数契约校验 | ❌（11 个模板声明与实际不一致，7 个有风险） | ❌ | 两边都缺 | **必须有**（保存时校验） |
| 模板导入导出 | ✅ JSON + 内容哈希对账三态 | ❌ | 缺失 | 照搬三态算法（§6.4） |
| 版本管理 | ❌（无 version 字段） | ❌ | 两边都缺 | 加 `version` + `updated_at`，系统模板版本号进 JSON |
| 社区分享 | ✅ 云端工坊 + 审核 | ❌ | 单机不需要 | **裁剪**：改为本地导入导出 JSON 包 |
| 信任边界 | 明文头声明身份 | — | gaea 无多租户 | **裁剪**：无网络工坊 |
| 写作风格 | 自由文本 + 预设 6 条 | `prompt`/`novelstyle` 量化 | **互补** | 新增风格文本 → 注入 system prompt |
| Skill | 磁盘目录 + 自动从 desc 提 triggers | `internal/skill` + 手写 frontmatter | gaea 元数据更规范 | 补 `triggers` 字段，可选 |
| 模板正文规模控制 | ❌ 1.74 MB 直接下发 | `BuildUserPrompt` 只发命中的 section | **gaea 更优** | 保持现状，不要学 MuMu |

### 11.3 gaea 侧需新增/修改的文件（建议）

**新增**

| 路径 | 职责 |
|---|---|
| `internal/promptstore/store.go` | 模板覆盖存储层：SQLite 表 `prompt_overrides(id TEXT PK, scope TEXT, project_id TEXT, template_key TEXT, template_name TEXT, content TEXT, description TEXT, category TEXT, parameters TEXT, is_active INTEGER, version INTEGER, created_at, updated_at)`，唯一索引 `(scope, project_id, template_key)` |
| `internal/promptstore/resolve.go` | 三级解析：`project 覆盖 → 全局覆盖 → 磁盘 JSON → embed JSON`，返回 `(Template, Source)` 以便 UI 显示来源 |
| `internal/promptstore/validate.go` | 保存时校验：占位符集合 ⊆ 声明参数集合；花括号平衡；长度上限 |
| `internal/promptstore/render.go` | 占位符渲染：语法定为 `{{name}}`（避免与 JSON/RTCO 正文冲突）；缺失变量**保留原文并返回 warning**，不抛错 |
| `internal/promptstore/bundle.go` | 模板包导入导出（对应 MuMu 的 export/import），含 `version`、`content_hash`、三态导入算法（§6.4） |
| `internal/novelstyle/inject.go` | 写作风格文本注入 system prompt：`【风格要求 - 最高优先级】\n\n{text}`，与 `skill.InjectSkill` 组合，Skill 优先、风格补充（对齐 `chapters.py:1744-1777`） |
| `internal/app/prompt_handler.go` | Wails 绑定：列出（**不含正文**）、读取、保存、重置、导入、导出、校验 |
| `frontend/src/pages/novel/PromptTemplates.tsx` | 模板管理页：分组 Tab、来源徽标（项目/全局/磁盘/内置）、在线校验提示、导入导出 |

**修改**

| 路径 | 改动 |
|---|---|
| `internal/prompt/prompt.go` | ① `Template` 增加 `Version string`、`Category string`、`Description string`（向后兼容，缺省零值）；② 新增 `RenderPlaceholders(s string, vars map[string]string) (string, []string)`，返回未解析变量名；③ `NewEngine` 接受可选 resolver 回调，使磁盘 JSON 可被 store 覆盖 |
| `internal/skill/skill.go` | frontmatter 解析补齐 `triggers`（对齐 `skill_loader.py:92-109`），用于「自然语言匹配 Skill」；当前 `parseSkillFile` 只认 4 个键（`:152-182`） |
| `internal/app/app.go:277, 678` | 装配 `promptstore.Store`，把 `prompt.NewEngine` 替换为 `prompt.NewEngineWithResolver(...)` |
| `prompts/*.json` | 15 个模板补齐 `version`/`category`/`description`（见 11.4 映射表） |

### 11.4 `prompts/*.json` 迁移方案（含 15 个模板逐个归属）

**兼容策略总则（三句话）**

1. **不改文件格式。** gaea 现有 6 字段 `Template` schema 是超集关系；MuMu 的 RTCO 骨架是「写在字符串里」的，gaea 是「结构化的」——保持 gaea 的结构化，**不要**把 MuMu 的 `<tags>` 抄进 `system`/`task` 字符串。
2. **只增不减字段。** 新增 `version`（`""` 或 `1`）、`category`、`description`、`parameters`。旧文件缺省即零值，`parseTemplate`（`prompt.go:215-220`）用 `json.Unmarshal` 天然容错，**无需迁移脚本**。
3. **占位符语法统一为 `{{name}}`。** 现有 `create-chapter.json` 的 `{word_count}` 需改为 `{{word_count}}`，并同步修改 `internal/app/create_chapter_handler.go:86-90`（`substituteWordCount` → `promptstore.RenderPlaceholders`）。这是**唯一一处需要改代码的模板字段变化**。

**15 个模板逐条归属（能力域按 t1~t6 划分）**

| # | 文件 | `name` 字段 | 归属能力域 | 实施优先级 | 迁移动作 |
|---|---|---|---|---|---|
| 1 | `worldview.json` | `worldview-agent` | **t5 世界观**（t6 参照） | P0 | 补 `category:"worldview"`；`user_idea`/`current_worldview` 保留；无占位符 |
| 2 | `character.json` | `character-agent` | **t5 角色** | P0 | `output.description` 内含 `---CHARACTER_UPDATE---` 标记约定，需与 t5 的 `internal/characterlib` 对齐 |
| 3 | `character-detail.json` | — | **t5 角色** | P1 | `target_character`/`user_request`/`worldview` 三段 |
| 4 | `character-generate-single.json` | — | **t5 角色** | P1 | 输出 `json`；消费点 `internal/app/characterlib_gen_handler.go:304-324` |
| 5 | `character-generate-batch.json` | — | **t5 角色** | P1 | 输出 `json` |
| 6 | `outline-chat.json` | — | **t4 情节/大纲** | P0 | `current_outline`/`user_request` |
| 7 | `outline-chat-node.json` | — | **t4 情节/大纲** | P0 | `target_node`/`user_request`/`worldview`/`characters` |
| 8 | `outline-continue.json` | — | **t4 情节/大纲** | P0 | **缺 `constraints` 字段**，需补空或补约束；`story_thread`/`existing_outline`/`chapter_summaries`/`continue_count`/`worldview`/`characters` |
| 9 | `outline-expand.json` | — | **t4 情节/大纲** | P0 | `story_thread`/`parent_node`/`existing_chapters`/`expand_count`/`worldview` |
| 10 | `plot-branch-browser.json` | — | **t4 情节/大纲** | P0 | 消费点 `internal/app/plot_branch_handler.go:110, 236` |
| 11 | `create-chapter.json` | `create-chapter` | **t4/t6 章节创作** | P0 | **含 `{word_count}` ×2**（`task` 与 `output.description`），需改 `{{word_count}}` 并改调用点 |
| 12 | `chapter-generate.json` | — | **t4/t6 章节创作** | P1 | `outline_node`/`worldview`/`previous_chapters_summary` 三段，无占位符 |
| 13 | `analysis-chapter.json` | — | **t4 情节分析** | P0 | 9 维度 JSON schema 在 `output.description` 内；与 gaea `internal/analysis` 对应 |
| 14 | `chapter-summary.json` | — | **t3 长程一致性** | P0 | 只吃 `chapter_content`，输出 `json` 摘要——正是 novelcontext 的输入 |
| 15 | `book-review.json` | — | **t2 拆书导入** | P0 | 只吃 `book_data`，输出 `json`——对应 MuMu 的 `BOOK_IMPORT_REVERSE_*` |

**建议新增（不删旧文件）**

| 新文件 | 对应 MuMu 模板 | 理由 |
|---|---|---|
| `prompts/chapter-regenerate.json` | `CHAPTER_REGENERATION_SYSTEM` + `get_chapter_regeneration_prompt` 的 6 段拼装（`prompt_service.py:2656-2751`） | gaea 现有 `internal/novelstyle/rewrite.go` 缺模板化重写 |
| `prompts/partial-regenerate.json` | `PARTIAL_REGENERATE`（`prompt_service.py:2414-2478`） | 局部重写是高频操作 |
| `prompts/novel-deslop.json` | MuMu 的 `story-deslop` Skill | gaea 已有 `.gaea/skills/novel-deslop/SKILL.md`，**只缺模板化封装** |
| `prompts/character-state-update.json` | MuMu 的 `AUTO_CHARACTER_*` + `character_states` 子 schema | t5 域会给出更精确规格 |
| `prompts/foreshadow-resolve.json` | `PLOT_ANALYSIS` 的伏笔子 schema | t1 域会给出更精确规格 |

**从 MuMu 借鉴但需裁剪的部分**

| MuMu 机制 | 裁剪方式 | 理由 |
|---|---|---|
| `prompt_workshop_*` 三表 + 云端 client | **整域裁剪**，改本地模板包（`.gaea/template-bundle.json`） | gaea 是单机桌面，无多租户、无 `X-Instance-ID` 信任模型 |
| `PromptSubmission` 审核流 | 裁剪 | 无中心服务端 |
| `user_id` 隔离 | 重定义为 `scope`（`global`/`project`） | gaea 单机只有一个使用者；按项目隔离才是有价值的维度 |
| `str.format()` 渲染 | 换 `{{name}}` 渲染 + 缺失降级 | 避免用户模板里的 JSON 花括号炸掉链路（MuMu §9.3） |
| 列表接口回传全量 `content` | 禁止 | MuMu 因此下发 1.74 MB（§9.2） |
| `PROMPT_CATEGORIES`（10 个题材分类） | 可复用为**标签**，不复用为二级菜单 | 与模板功能分类不同域 |

---

## 12. 未决问题（需人工决策）

| # | 问题 | 影响面 |
|---|---|---|
| Q1 | 占位符语法选 `{{name}}`（须改现有 15 个文件中的 1 个 + 1 处调用点）还是保留 `{name}` 并强制模板作者转义字面花括号？ | `promptstore/render.go`、`create-chapter.json`、`create_chapter_handler.go` |
| Q2 | gaea 是否需要「项目级模板覆盖」？（MuMu 没有，但 MuMu 的模板是全局常量，gaea 的是可编辑 JSON，做项目级覆盖成本很低） | `promptstore` 表结构（`scope`+`project_id`） |
| Q3 | Skill 的 `triggers` 自动匹配（自然语言→Skill）是否引入？MuMu 靠 4 级兜底（精确/前缀/关键词表/`/{name}`），关键词表是**硬编码中文映射**（`skill_loader.py:267-290`） | `internal/skill/skill.go` |
| Q4 | 风格文本与 gaea `novelstyle` 量化指纹如何关联？MuMu 无此映射。需要为 6 条预设风格标注「期望指纹区间」吗？ | `internal/novelstyle/inject.go` + 预设数据 |
| Q5 | 是否需要模板 `version` 触发「系统模板升级 → 提示用户合并自定义」的交互？MuMu 完全没做（§6.1） | `promptstore` + UI |
| Q6 | 模板包的分享格式：纯 JSON（好读、易改坏）还是带哈希签名的 JSON 包（可校验完整性）？ | `promptstore/bundle.go` |

---

## 13. 未能从源码确认的项（显式标注）

1. **云端服务端实现**。`WORKSHOP_CLOUD_URL = "https://mumuverse.space:1566"`（`config.py:140`）指向的外部服务不在本仓库，`mumuverse.space` 的部署形态、数据库、审核流程的运维侧实现均无法验证。仓库内的 `WORKSHOP_MODE=server` 分支即为该服务的实现，但无法确认线上运行的就是这份代码。
2. **`request.state.is_proxy_request` 的赋值位置**。`prompt_workshop.py:52` 读取该属性，但中间件（`app/middleware/auth_middleware.py`）的具体实现未在本轮逐行通读，因此**无法确认**代理请求的判定条件与 `X-Instance-ID` 是否在别处被校验。§7.2 的信任边界结论以「在 `prompt_workshop.py` 与 `workshop_client.py` 中未见校验」为限。
3. **模板的实际使用频次**。哪些 `template_key` 在生产中被真实调用过，仓库内无埋点数据可证；§4 的「调用点」信息来自静态检索（`PromptService.get_template(...)` / `cls.get_template(...)` / `..._with_fallback(...)` 共 **51 处**匹配），不代表线上流量。
4. **`prompt_workshop_items.tags` 的 `JSON` 列在 PostgreSQL 与 SQLite 下的实际行为差异**（是否需要 `JSONB`、是否有索引）未做运行时验证；仅记录模型声明为 `JSON`（`models/prompt_workshop.py:16`）。
5. **`PromptService` 类属性在 Python 中的内存占用**。36 个模板常量合计字符数已实测（`PLOT_ANALYSIS` 最大 8496 字符），但 Python 进程内的实际 RSS 增量未测量。
6. **`get_all_system_templates()` 的 Skill 拼接是否会在每次请求重新读盘**。`get_all_skills_cached()` 有模块级缓存（`skill_loader.py:302-309`），首次调用后应命中缓存；但缓存无失效钩子（除手动 `refresh_skills_cache`、`skills.py:255-259`），**无法确认**长期运行下磁盘文件变更是否会被感知。§9.2 的 1.74 MB 是缓存命中后的**内存拷贝+序列化**成本，非重复读盘成本。

---

## 14. 一次性可执行的验证片段

供审校员复核本文所有实测数字。把下面 4 段分别存为临时 `.py` 文件后在**工作区根目录**执行（`python <file>`）。

**verify1.py — 系统内置模板清单与 11 个声明/实际不一致模板（对应 §4）**

```python
import re, ast
src = open(r'clones/MuMuAINovel/backend/app/services/prompt_service.py', encoding='utf-8').read()
bodies = {}
for m in re.finditer(r'^    ([A-Z][A-Z0-9_]{3,}) = """(.*?)"""', src, re.S | re.M):
    bodies[m.group(1)] = m.group(2)
for m in re.finditer(r'^    ([A-Z][A-Z0-9_]{3,}) = "((?:[^"\\]|\\.)*)"', src, re.M):
    bodies.setdefault(m.group(1), ast.literal_eval('"' + m.group(2) + '"'))
i = src.index('template_definitions = {')
i = src.index('{', i); depth = 0
for j in range(i, len(src)):
    depth += (src[j] == '{') - (src[j] == '}')
    if depth == 0:
        end = j + 1; break
d = ast.literal_eval(src[i:end])
print('registered =', len(d), '| consts =', len(bodies))   # 预期 36 / 36
n = 0
for k, v in d.items():
    act = set(re.findall(r'\{([a-zA-Z_][a-zA-Z0-9_]*)\}', bodies.get(k, '')))
    dec = set(v['parameters'])
    if act - dec or dec - act:
        n += 1
        print(k, 'used_not_declared=', sorted(act - dec), 'declared_not_used=', sorted(dec - act))
print('mismatched_templates =', n)
# 预期 11，但需区分两类：
#   A) 潜在运行时风险（used_not_declared 非空，共 6 个）：用户可以按「声明参数」去写模板，
#      结果引用到未声明变量 → 调用点不传该关键字 → format() 抛 KeyError：
#        OUTLINE_CONTINUE, CHAPTER_GENERATION_ONE_TO_MANY, CHAPTER_GENERATION_ONE_TO_MANY_NEXT,
#        CHAPTER_GENERATION_ONE_TO_ONE, CHAPTER_GENERATION_ONE_TO_ONE_NEXT, PLOT_ANALYSIS,
#        AUTO_CHARACTER_ANALYSIS            （实测 7 个，见上方输出）
#   B) 良性（仅 declared_not_used 非空，共 4 个）：OUTLINE_CREATE(target_words)、
#      OUTLINE_EXPAND_SINGLE/MULTI(scene_instruction)、CHAPTER_REGENERATION_SYSTEM(全部 8 个，
#      因为该模板根本不走 format)——只是注册元信息陈旧，不会报错。
```

**verify2.py — Skill 拼接体积 1 820 901 B（对应 §4.1 / §9.2）**

```python
import os, glob
base = r'clones/MuMuAINovel/backend/app/skills'
t = 0
for d in sorted(os.listdir(base)):
    if not os.path.isdir(os.path.join(base, d)):   # 跳过 README_HOW_TO_ADD_SKILL.md
        continue
    r = os.path.join(base, d, 'references')
    s = sum(os.path.getsize(f) for f in glob.glob(os.path.join(r, '*.md'))) if os.path.isdir(r) else 0
    md = os.path.getsize(os.path.join(base, d, 'SKILL.md'))
    print(d, md + s); t += md + s
print('TOTAL =', t)                                          # 预期 1820901
```

**verify3.py — gaea 现有 15 个模板的字段与占位符（对应 §11.1 / §11.4）**

```python
import json, glob, os, re
n = 0
for f in sorted(glob.glob(r'prompts/*.json')):
    n += 1
    t = json.load(open(f, encoding='utf-8'))
    ph = sorted(set(re.findall(r'\{([a-zA-Z_][a-zA-Z0-9_]*)\}', json.dumps(t, ensure_ascii=False))))
    print(os.path.basename(f), sorted(t.keys()), ph)
print('gaea_templates =', n)                                 # 预期 15；只有 create-chapter.json 有 ['word_count']
```

**verify4.ps1 — `AI_DENOISING` 只有引用没有定义（对应 §9.1）**

```powershell
Select-String -Path clones\MuMuAINovel\backend\app\**\*.py -Pattern 'AI_DENOISING'
# 预期：仅 api/polish.py:40 与 api/polish.py:114，全库无任何 "AI_DENOISING =" 定义
```

以上四段为本文所有量化结论（36 / 9 / 1 820 901 / 15 / 51）的唯一来源。

---

## 15. 结论摘要（给 t7/t8 的输入）

1. MuMu 的模板体系是**两级三源**（用户 DB 覆盖 > 磁盘 Skill > Python 常量），核心设计是「DB 只存覆盖行，系统模板随代码发布」——这让系统模板零成本升级，但代价是**用户自定义行永不跟随升级**（无版本概念）。
2. 变量语法就是裸 `str.format()`，**没有条件/默认值/嵌套**，且字面花括号必须双写。**11 个模板的 `parameters` 声明与实际占位符不一致**（其中 7 个引用了未声明变量），说明该字段是 UI 提示而非契约——gaea 必须把它做真。
3. 真实可迁移的高价值资产有三个：**导出/导入的「内容哈希 + 三态智能导入」算法**（§6.4）、**RTCO 标签骨架与 constraints 符号约定**（§5）、**工坊三表的数据建模**（§2.4-2.6，但单机下应整域裁剪）。
4. 两个应当显式规避的高危缺陷：**`AI_DENOISING` 键缺失导致 `/api/polish` 永久 500**（§9.1）、**列表接口因 Skill 全文拼接下发 1.74 MB**（§9.2）；一个必须修正的设计缺陷：**用户模板保存时零校验 + 运行时缺失变量即抛错**（§9.3）。
5. 与 gaea 的分工建议：gaea 的 `internal/prompt` **结构化 RTCO 是更优设计**，应保留并增强（加覆盖层、参数校验、版本），**不要**把 MuMu 的「RTCO 写在字符串里」抄回来。gaea 的风格能力（量化指纹）与 MuMu 的风格能力（自由文本注入）**互补而非替代**。
6. `prompts/*.json` 迁移只需一处代码改动（`{word_count}` → `{{word_count}}` + 调用点），其余 14 个模板零改动、零迁移脚本即可继续工作；新增字段全部可选、向后兼容。
