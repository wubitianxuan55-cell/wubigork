# gaea t6 提示词工坊·首刀（模板可编辑覆盖层）规格书

> 2026-09-16 立项。来源：用户指令「继续优化迭代 gaea，记得使用子代理」。
> 蒸馏依据：`docs/distill/06-prompt-workshop.md`（§5/§6/§9.3/§11）。
> 拆线：三线并行子代理（A 引擎纯函数 / B handler 装配 / C 前端三件）→ 主代理收口。
> 目标版本：v4.323.0。

## 1. 论点

t6 域核心缺口（06 规格 §11.2）：gaea 模板是「磁盘 JSON + embed 兜底」的只读两层，
**没有用户可编辑层**——20 个 `prompts/*.json` 用户改不了（改了也会被下次发版覆盖），
占位符只有 `{word_count}` 单点硬编码替换，模板保存零校验（MuMu §9.3 同型缺陷）。
本刀补「三级解析：全局覆盖 → 磁盘 JSON → embed JSON」+ `{{name}}` 渲染 + 保存校验 + 工坊面板。

对 MuMu 的升级/裁剪（06 规格 §11.4 裁决沿用）：
- **升级**：覆盖存储用 `<DataRoot>/prompt_overrides.json`（taskinbox/skillstats 同款
  容错读 + temp+rename 原子写），不上 SQLite 三表；校验在保存时做真（MuMu 零校验）；
  `{{name}}` 语法避免 JSON 字面花括号炸链路（MuMu §9.3 裸 format() 教训）。
- **裁剪**：云端社区工坊三表/审核流/X-Instance-ID 信任模型整域不做（单机无多租户）；
  列表接口不回传正文（MuMu 1.74MB 下发教训，06 规格 §9.2）。

## 2. 未决问题裁决（06 规格 §12）

| # | 裁决 |
|---|---|
| Q1 | 占位符语法 = `{{name}}`（迁移 create-chapter + rewrite-chapter 两文件；`substituteWordCount` 兼容双语法，用户手改盘上旧模板不炸） |
| Q2 | V1 只做**全局覆盖**；`Override` 不带 scope 字段（项目级覆盖挂观察池，等真实需求） |
| Q3 | Skill triggers 自动匹配不做（观察池） |
| Q4 | 风格文本注入不做——oh-story T6（v4.289）书级文风档案 style.md 注入链路已有，互补部分挂观察池 |
| Q5 | `version` 字段落库记录，**不做**「系统模板升级→提示合并」交互（观察池） |
| Q6 | 模板包导入导出（content_hash 三态算法）= t6-C2 下一刀，本刀不做 |

## 3. Go 契约（线 A：internal/prompt + internal/promptstore + prompts/*.json）

### 3.1 internal/prompt/prompt.go 增强（向后兼容，只增不改）

```go
// Template 新增字段（全部 omitempty，旧 JSON 零迁移）
Version     string   `json:"version,omitempty"`     // "1" 起步
Category    string   `json:"category,omitempty"`    // 分组用（zh 短词）
Description string   `json:"description,omitempty"` // 一句话说明
Parameters  []string `json:"parameters,omitempty"`  // 声明占位符名（校验契约）

// 包级函数：{{name}} 渲染。缺失变量保留原文并记名（不抛错）；只认双层花括号，
// 单层 {x}（JSON 字面量/旧语法）原样不动。返回 (渲染结果, 未解析变量名)。
func RenderPlaceholders(s string, vars map[string]string) (string, []string)

// Engine 增强：覆盖解析钩子（nil 回落内置表；启动装配时设置，见线 B）
func (e *Engine) SetOverride(fn func(name string) *Template)
// Get 语义变更：override 命中优先于 templates map。
func (e *Engine) Names() []string // 全部模板名（templates map 键，字典序稳定）
```

### 3.2 新包 internal/promptstore（零 IO 纯函数，taskinbox 先例）

```go
package promptstore

type Issue struct {
    Code     string `json:"code"`     // empty-system / empty-task / brace-unbalanced / too-long / undeclared-var / legacy-brace
    Severity string `json:"severity"` // "error" | "warn"
    Message  string `json:"message"`
}

type Override struct {
    Key         string          `json:"key"`                 // 模板名（Template.Name）
    Category    string          `json:"category,omitempty"`
    Description string          `json:"description,omitempty"`
    Content     prompt.Template `json:"content"`             // 完整模板正文（整模板覆盖，MuMu 同款）
    IsActive    bool            `json:"isActive"`            // false = 禁用 = 回落磁盘/embed
    Version     int             `json:"version"`             // 保存次数（upsert 自增）
    CreatedAt   int64           `json:"createdAt"`           // 毫秒
    UpdatedAt   int64           `json:"updatedAt"`           // 毫秒
}

func NormalizeKey(key string) (string, error) // trim；非空；≤100；字符集 [A-Za-z0-9._-]
func Validate(t prompt.Template) []Issue      // 见下表
func ActiveOverride(entries []Override, key string) *Override // IsActive 且 Key 匹配；无则 nil
func Upsert(entries []Override, ov Override, nowMs int64) []Override // 同 key 替换（Version=旧+1，CreatedAt 保留）；新键追加（Version=1）；结果按 Key 字典序
func Remove(entries []Override, key string) ([]Override, bool)
```

`Validate` 规则（06 规格 §11.3 validate.go 的 gaea 化）：

| Code | Severity | 规则 |
|---|---|---|
| empty-system | error | System trim 后为空 |
| empty-task | error | Task trim 后为空 |
| brace-unbalanced | error | 正文中 `{{` 与 `}}` 计数不等，或存在未闭合 `{{`（其后无 `}}`） |
| too-long | error | System 或 Task > 20000 rune，或全模板序列化 > 65536 rune |
| undeclared-var | warn | `{{name}}` 出现但 name ∉ t.Parameters（仅 Parameters 非空时报） |
| legacy-brace | warn | 存在单层 `{word_count}` 样式占位符（提示旧语法在新渲染下不会被替换） |

存在任一 error → 调用方不得落盘（warn 不阻断）。

### 3.3 prompts/*.json 元数据补齐（20 个全量）

- 全部补 `version`("1") + `category` + `description`（zh 一句话）+ 有占位符的补 `parameters`。
- category 取值域（面板分组）：`worldview` / `character` / `outline` / `chapter` / `rewrite` /
  `analysis` / `summary` / `book-import` / `skill`（按模板实际归属，见 06 规格 §11.4 表 + 后续新增模板）。
- `create-chapter.json` 与 `rewrite-chapter.json` 的 `{word_count}` ×5 处全部改 `{{word_count}}`，
  并补 `"parameters": ["word_count"]`。
- **不改任何模板正文的语义内容**（除占位符语法迁移）；所有既有 Go 测试必须零改动通过。

### 3.4 线 A 测试

- internal/prompt：RenderPlaceholders（命中/缺失保留/单层花括号不动/多次出现/空 vars）；
  SetOverride 优先级（override 命中 > 内置；nil 回落）；Names 稳定排序。
- internal/promptstore：Validate 六规则矩阵（含 error/warn 分级与边界）；
  Upsert 替换自增/新键 Version=1/CreatedAt 保留/字典序稳定；Remove；ActiveOverride
  含 IsActive=false 回落 nil；NormalizeKey 全防御。
- 既有 prompt 测试（embed 优先级、Order 稳定序）零改动必须绿。

## 4. Handler 契约（线 B：internal/app，新文件 gaea_prompt_store.go）

### 4.1 状态文件

`<DataRoot>/prompt_overrides.json`，结构 `{"version":1,"templates":[Override]}`
（camelCase，taskinbox 同款：缺失/损坏回空表恒非 nil；temp+rename 原子写）。

### 4.2 App 方法（NovelB 门面，五绑定）

```go
type PromptTemplateMeta struct {
    Key            string `json:"key"`
    Category       string `json:"category"`
    Description    string `json:"description"`
    Source         string `json:"source"`         // "override" | "builtin"
    HasOverride    bool   `json:"hasOverride"`    // 覆盖行存在（无论启停）
    OverrideActive bool   `json:"overrideActive"` // 覆盖行存在且 IsActive
    Version        int    `json:"version"`        // 生效版本（覆盖=保存次数，内置=0）
    UpdatedAt      int64  `json:"updatedAt"`      // 覆盖最近保存毫秒；内置 0
}
func (a *App) PromptTemplateList() []PromptTemplateMeta
// Names() 全量；Category/Description 优先取覆盖行、缺省回落内置模板字段。

type PromptTemplateDetail struct {
    Meta     PromptTemplateMeta `json:"meta"`
    Template prompt.Template    `json:"template"` // 生效模板（覆盖激活=覆盖内容）
    Base     prompt.Template    `json:"base"`     // 磁盘/embed 基线（对比/恢复参照）
}
func (a *App) PromptTemplateGet(key string) (PromptTemplateDetail, error)

type PromptSaveResult struct {
    Saved  bool              `json:"saved"`
    Issues []promptstore.Issue `json:"issues"` // 含 warn（已保存也带回提示）
    Version int              `json:"version"` // 保存后版本；未保存 0
}
func (a *App) PromptTemplateSave(key string, reqJSON string) (PromptSaveResult, error)
// reqJSON: {"isActive":bool(缺省 true),"category":"","description":"","content":{Template 全字段}}
// 流程：NormalizeKey → 引擎存在该 key（否则 error "未知模板键"）→ Validate(content)
//   有 error → 不落盘返回 Issues；否则 Upsert（Version 自增）→ 落盘 → 失效引擎覆盖缓存。

func (a *App) PromptTemplateReset(key string) error // 删覆盖行 → 回落内置；不存在也幂等成功

type PromptPreviewResult struct {
    SystemPrompt string   `json:"systemPrompt"`
    Warnings     []string `json:"warnings"` // RenderPlaceholders 未解析变量名
}
func (a *App) PromptTemplatePreview(reqJSON string, varsJSON string) (PromptPreviewResult, error)
// reqJSON: {"content":{Template}}；varsJSON: map[string]string（可空串）。
// 渲染 BuildSystemPrompt("") 的 {{}} 占位（预览未保存草稿；V1 只预览 system 段）。
```

### 4.3 引擎接线

- App 持覆盖缓存（mutex 串行；save/reset 后失效；override 闭包装载时惰性重读文件）。
- `app.go` 两处装配点（writingState 初始化 ~:274-279 与 `SetPromptFS` ~:674-680）统一走
  `a.applyPromptOverrides(eng)`：`eng.SetOverride(func(name) *prompt.Template { 读缓存 → ActiveOverride → 返回其 &Content 或 nil })`。
- `substituteWordCount`（create_chapter_handler.go）扩展：先替换 `{{word_count}}` 再替换
  旧 `{word_count}`（兼容用户手改盘上旧语法模板）；rewrite-chapter.json 迁移后
  novel_rewrite_handler 两处调用零改动自然生效。
- 不触碰：internal/prompt/*、internal/promptstore/*、prompts/*、前端任何文件、bindings_*.go。

### 4.4 线 B 测试（internal/app，新 gaea_prompt_store_test.go）

- 状态文件 roundtrip（保存→List/Get 生效→Reset 回落→坏文件容错回空表）。
- Save 校验闸（error 不落盘 / warn 落盘带回 Issues）+ 未知模板键拒绝 + isActive=false
  保存后引擎 Get 回落内置。
- 引擎覆盖生效 e2e：改 create-chapter system → NewEngine 后 CreateChapter 链路取到的
  system prompt 带覆盖标记（可只测 writingState.eng.Get 层，不起真模型）。
- substituteWordCount 双语法矩阵（{{}} 新语法 / {} 旧语法 / 混合 / 无占位符）。
- Preview 渲染 + 未解析变量警告。

## 5. 前端契约（线 C）

### 5.1 新文件

- `frontend/src/components/novel/api/prompt.ts`：五函数封装（listTemplates/getTemplate/
  saveTemplate/resetTemplate/previewTemplate），character.ts 同款 `app.*` 直调范式。
- `frontend/src/components/novel/PromptWorkshopPanel.tsx`（+ .test.tsx）：antd Modal，
  硬编码中文（小说域口径，RewriteHistoryPanel 先例）。三区：
  1. **列表**：按 category 分组（Collapse 或分组列表），行 = 模板名 + 来源徽标
     （自定义=orange / 内置=default；禁用的自定义行加灰 Tag「已停用」）+ version；
     打开才拉一次（零轮询，RewriteHistoryPanel 同款）。
  2. **详情编辑**（选中行右侧/抽屉）：Meta 行 + `Base` 对照 Collapse + 编辑表单
     （System/Task/输出说明 textarea + category/description 输入 + 启用开关）；
     「保存」→ Save，Issues 逐条渲染（error 红 / warn 橙）；「恢复内置」Popconfirm
     显式 okText（Reset）。
  3. **预览**：变量名输入（按 Template.parameters 动态生成 Input 列）+ 渲染按钮 →
     Preview 结果只读展示 + 未解析变量 warnings Tag。
- 挂点：`frontend/src/pages/CreatePage.tsx` rail「重写历史」后加「提示词工坊」按钮
  （无徽标打开才拉；既有 CreatePage 测试 13/13 不得破坏——如断言按钮数需同步补）。

### 5.2 mock 与锁

- `frontend/src/gaea/lib/mock/novel.ts`：五方法 + 名单 union 同步（List 两样本
  含一个 override 态；Get 配套全文；Save 简单校验回 Issues；Reset 清空；Preview
  本地 `{{}}` 替换）；mock-contract 新测试文件（先例 mock-contract-*.test.ts）。
- `frontend/src/gaea/lib/spaceBindings.ts`：五名 → `"play"`（play 锁计数 +5，注释同步）。
- **不碰** `bindingNames.ts`（生成物，主代理收口 regen）；tsc 中仅允许
  api/prompt.ts 与 mock/novel.ts 里这五个名字的解析错误作为「待 regen 项」上报，
  其余类型错误必须清零。

### 5.3 线 C 测试

PromptWorkshopPanel ≥5 用例（列表分组与徽标/编辑保存含 Issues 渲染/恢复内置二次确认/
预览变量渲染/空态）；mock-contract 测试钉死五方法签名样本；CreatePage 既有用例零破坏。

## 6. 验收与门禁（主代理收口）

1. `go run ./scripts/gen_bindings`（bindings_novel +5 / bindingNames.ts 再生）→
   `TestBindingsCompleteness` + `check-bindings-drift` OK（绑定面 696→701，play 锁 +5）。
2. `cd frontend && npm run build`（tsc -b + vite）→ `wails build -s`。
3. `scripts/ci.ps1` 全绿 exit 0（Go 129 包 + vitest ~366 文件；flaky 复跑先例）。
4. 版本三处（wails.json / app_info.go / README + versioninfo.rc sync-version.ps1）4.323.0；
   releases/gaea-v4.323.0.exe + SHA256SUMS + releases/v4.323.0.md + releases/README
   （本地产物只留最近 5 版）；桌面副本同哈希；smoke /api/health 200。
5. 文档：CHANGELOG/README/.gaea AGENTS（迁 1 插 1）/progress/todos/本规格书补出口对照。

## 7. 出口对照（本刀完成判据）

- 20 模板全部可看/可改/可恢复，来源与版本可见 ✅
- 覆盖即时生效于生成链路（引擎 Get 三级解析）✅
- 保存有真校验（error 阻断/warn 提示）✅
- 占位符统一 `{{name}}` 且旧语法兼容 ✅

## 8. 观察池（本刀不做）

模板包导入导出（content_hash 三态，t6-C2）；项目级覆盖（scope）；自建新模板键；
Skill triggers 匹配；系统模板版本升级合并交互；风格文本注入与 novelstyle 量化打通；
MuMu PROMPT_CATEGORIES 题材标签。
