# gaea 图像域 T0「契约」设计稿（2026-09-05）

> 状态：**已落地随 v4.98.0 发版**。基线：gaea v4.97.0。
> v0.2（2026-09-05 实现前修订）：实机盘点确认 `GenerateSceneIllustration` 当前只返回
> URL、**不落盘**——章节配图登记推迟到 T1（与配图落盘/素材回流同刀）；试点调整为
> 「书封登记 + 绘梦工作台落盘登记 + OCR/识图入口校验」。
> v0.3（2026-09-05 实现后回写）：本版实际交付 = T0 三试点 **+ T1 提前量**（章节配图
> 落盘并登记、TaskCenter「素材库」tab 读 ledger、角色剧照登记）**+ T2 参考槽 v0**
> （绘梦选角色 → 参考图图生图近似，ipadapter/pulid 诚实拒绝）。绑定面 **579→581**
> （+`ImageHubAssets`/`ChapterArtList` 两个只读绑定，§8「T0 零绑定」按实际修订）。
> 另修一处真缺陷：登记运行态闸原只看 `gaeaCfgSnapshot()!=nil`，app 包测试初始化过
> 全局配置导致登记写进源码树——改为 `App.Startup` 显式武装位，测试进程恒不落盘。
> 权威路线：`docs/gaea-image-domain-longterm-plan-2026.md`（T0 主题）。
> 输入：`docs/gaea-dream-studio-nextgen-2026-09.md`（调研 + 用户定位补充）。
> 纪律：每步独立提交可回退；旧数据只读兼容；与工位零交叉红线；绑定面漂移 PASS；
> Go/vitest 全绿。**T0 的目标是立契约，不是做新能力**（改图/一致性/故事板属 T2/T3+）。

---

## 1. 目标与范围

### 1.1 目标

以**零行为回归**为前提，交付「图像能力域」的第一层主干：

1. **五原语契约**：识图-读 / 识图-懂 / 生图 / 改图 / 图示的规范类型、能力注册表与
   空间/来源元数据；
2. **溯源登记最小集（Asset Ledger v1）**：域内产物统一登记（模型/参数/来源/成本/
   时间戳/许可），落各空间目录；
3. **与模型中心的目录协议**：画室与各前台展示的模型档位一律读模型中心目录的只读
   视图；图像域不另立模型目录事实源；
4. **存量管线对照表**：现有图相关绑定/入口 → 原语映射 + 接线状态 + 排期；
5. **试点接线**：先让 2–3 条低风险管线真正走域协调器 + 登记，验证契约可用。

### 1.2 非目标（T0 不做）

- 不搬引擎：`internal/ai` 图片后端、vision/OCR 链、diagram 实现全部原地保留；
- 不合并实现：OCR/识图多链只收编入口，不收编实现；
- 不做新能力：指令编辑（T3）、一致性参考槽（T2）、蒙版/抠图（T3）只立契约位；
- 不迁移存量历史：绘梦 localStorage 历史、既有 exports 文件**不动**（T1 收口）；
- 不改壳/导航/双空间存储红线；不新增公共模型目录事实源。

---

## 2. 现状盘点（v4.97.0 已核）

### 2.1 图相关绑定/入口 → 现状

| 入口（绑定） | 门面 | 现状行为 | 落点 |
|---|---|---|---|
| `GenerateFreeImage` | ImageB（绘梦页） | txt2img（多引擎） | ImageSaveDir（默认 `Pictures/gaea`）+ 前端 localStorage 历史 |
| `GenerateMedia` | ImageB（绘梦页） | txt2img / img2img / t2v | 同上 |
| `GenerateDiagram` | ImageB（绘梦页伪模型 + 办公 agent diagram 工具） | LLM → Mermaid → 返回 code/渲染 | `.mmd`/PNG（工作区） |
| `GenerateCharacterPortrait` / `CharacterGeneratePortrait(WithRef)` | CharacterLib | 角色立绘/参考图生剧照 | characterlib 数据目录（portraits/refs） |
| `GenerateSceneIllustration(chapterNum)` | NovelB → writingState | 章节场景配图（Aurora 管线） | `.gaea/play/exports` |
| `GaeaGenerateBookCover(projectID)` | NovelB（play） | 3:4 书封 | `.gaea/play/exports` |
| `GaeaOCRText(imagePath)` | OfficeB | 办公「提取文字」：herdsman OCR → docmd 链 | 文本返回 |
| `GaeaRecognizeImage(imagePath, prompt)` | OfficeB | 本地 vision 模型识图（理解/问答） | 文本返回 |
| `visionOCRText`（私有） | 微信识图（v4.8.3） | 多模态主模型 → OCR 兜底 | 文本返回 |
| `GaeaSavePastedImage` / `GaeaSaveAttachmentFile` | OfficeB | 粘贴/附件图片入工作区 | `.gaea/uploads` |
| 造价 OCR | `gaea_cost_import_vision.go` | 报价单 OCR → 表格解析 | 造价库 |
| `GaeaCaptureScreen` | OfficeB | 截屏 PNG data URL | 内存返回 |
| 模型中心 | ModelCenter 各引擎/目录区 | 引擎注册/模型分类（kind: llm/tts/stt/image/ocr…） | 配置 + 目录视图 |

### 2.2 功能域模型绑定结论（重要事实）

`feature_model_handler.go` 的功能域（chat/whisper/novel/office/gaea/characterlib/routine）
绑定的是**文本 LLM 引擎 + 模型**（注释明示「各功能板块可独立指定 LLM」）。**绘梦/生图
不是文本功能域**，图像选型现状由 `image_backend/image_model` 配置 + `portrait_backend/
model` 覆盖 + 引擎激活状态决定。因此：

- **不把 imagegen 强行塞进 LLM feature 域**（语义错位）；
- 图像「域 × 空间 × 能力」的模型选择规则 = 显式调用模型 → 领域覆盖（portrait 等）
   → 绘梦后端配置 → 引擎激活默认，四级解析；
- 模型中心继续负责**目录**（有哪些引擎/模型/kind/健康），T0 为其补「图像能力标签 +
  成本档位」的只读扩展（见 §5），归属仍是模型中心侧代码。

### 2.3 资产与溯源现状（缺口即 T0 登记要补的）

| 资产 | 存储 | 溯源现状 |
|---|---|---|
| 绘梦成图/视频 | ImageSaveDir + 前端 `gaea.imagegen.historyMeta`（localStorage） | 元数据仅前端，跨重启靠 file_path 恢复 |
| 角色立绘/参考图 | characterlib 数据目录 + characterlib.db | 角色字段自带引用，无统一资产 ID |
| 章节配图/书封 | `.gaea/play/exports`（S4 分区既有） | 无登记，无法按项目/章节检索与溯源 |
| 粘贴/附件图 | `.gaea/uploads` | 无登记 |
| 办公 diagram/图表 | 工作区 `.mmd`/PNG / `.gaea/exports` | 办公交付物登记表（事件日志折叠）覆盖写类工具产物 |
| OCR/识图结果 | 纯文本返回 | 无产物登记需求（非资产） |

### 2.4 空间与门面分类现状

- `spaceBindings.ts` 只约束 **gaea bridge（AppBindings）方法**（work/play/shared/
  independent 全量显式分类）；
- 小说/绘梦/角色库等 legacy 页面走 `wailsjsCompat` + `api/*` 包装（不经 spaceBindings）；
- 空间落地主要由目录与事件承担（`spaces.ExportsDir`：work=`.gaea/exports`、
  play=`.gaea/play/exports`）。

**T0 结论**：域协调器在 Go 侧统一收空间/来源入参（绑定层转发时显式给值），
不依赖前端 spaceBindings 覆盖 legacy 面。

---

## 3. 五原语契约设计

### 3.1 原则

- **类型化入口，不用通用 JSON blob**：每个原语一个规范请求/响应类型，便于静态检查、
  单测与演进；
- **资产返回 AssetRef（路径优先，不塞 base64）**；展示侧读图继续走既有
  `GaeaAttachmentDataURL`；
- **失败诚实透传**：引擎不支持某能力（如 GLM 无 img2img）沿用现有错误口径，
  不静默降级；
- **空间显式入参**：调用入口必须声明 `Space` 与 `SourceBoard`；无来源时用当前有效
  空间并记录来源=“shell”`。

### 3.2 类型草案（Go，示意非最终签名）

```go
// 文件建议：internal/app/image_domain.go（协调器 + 类型 + 注册表）

type ImageCapability string

const (
    CapVisionRead      ImageCapability = "vision.read"      // 识图-读：OCR/取字
    CapVisionUnderstand ImageCapability = "vision.understand" // 识图-懂：理解/描述/问答
    CapMediaGenerate   ImageCapability = "media.generate"   // 生图（含 txt2img/img2img/t2v）
    CapMediaEdit       ImageCapability = "media.edit"       // 改图（T3 起有实现）
    CapMediaDiagram    ImageCapability = "media.diagram"    // 图示：流程图/导图/架构图
)

type SpaceName string // 复用 gaea spaces 常量（work/play）

type CallerRef struct {
    Space       SpaceName `json:"space"`
    SourceBoard string    `json:"source_board"` // novel | characterlib | imagegen | office | weixin | shell ...
}

type AssetKind string // image | video | diagram

type AssetRef struct {
    ID       string    `json:"id"`
    Kind     AssetKind `json:"kind"`
    Path     string    `json:"path"`      // 本地绝对/相对路径（展示侧经既有绑定读回）
    MIME     string    `json:"mime"`
    Width    int       `json:"width,omitempty"`
    Height   int       `json:"height,omitempty"`
    Duration int       `json:"duration_ms,omitempty"` // 视频
}

type ModelTrace struct {
    Backend string `json:"backend"`
    Model   string `json:"model"`
    Cost    string `json:"cost,omitempty"` // "0" | "0.1 CNY/张" | "未定价"
}

type AssetMeta struct {
    Caller     CallerRef    `json:"caller"`
    Capability ImageCapability `json:"capability"`
    Trace      ModelTrace   `json:"trace"`
    Prompt     string       `json:"prompt,omitempty"`
    Params     map[string]interface{} `json:"params,omitempty"` // seed/size/loras/denoise/frames...
    Refs       []string     `json:"refs,omitempty"`   // 角色/参考图 AssetID 或文件路径（T2 起用）
    ParentID   string       `json:"parent_id,omitempty"` // 变体簇（T3 编辑链）
    CreatedAt  string       `json:"created_at"`
    LicenseHint string      `json:"license_hint,omitempty"` // 模型许可提示（FLUX dev 等）
    AIFlag     bool         `json:"ai_flag"` // 恒 true（生成物标记）
}
```

**原语请求/响应（各自独立类型）**：

| 原语 | 请求要点 | 响应 | T0 落地 |
|---|---|---|---|
| vision.read | ImagePath + 语言/取字模式 | Text + EngineChain（如 herdsman→docmd） | 包装 `GaeaOCRText` |
| vision.understand | ImagePath + Prompt | Answer + ModelTrace | 包装 `GaeaRecognizeImage` / 微信 visionOCRText |
| media.generate | Prompt/Negative/Size/Model/Seed/N+（现有字段）+ CallerRef | []AssetRef + Meta | 包装 GenerateFreeImage/GenerateMedia/剧照/章节配图/书封 |
| media.edit | ImageRef + Instruction + CallerRef（+Mask 位，T3） | []AssetRef | **契约先立，T3 实现** |
| media.diagram | Kind（flowchart/mindmap/…）+ Topic + CallerRef | Code + AssetRef(PNG) | 包装 `GenerateDiagram` / agent diagram 工具 |

### 3.3 能力注册表（内部路由/元数据，非模型目录）

```go
type capabilityEntry struct {
    Cap       ImageCapability
    Kind      AssetKind // 产不产物（识图不产物）
    Spaces    []SpaceName // 允许空间；空 = 按调用方显式空间执行
    Available bool       // T0: 已有实现者 true；edit=false
}
```

注册表只回答「原语是否可用、是否产物、允许哪些空间」；**模型档位与成本一律问模型中心
目录（§5）**。未知原语 fail-closed 报错。

### 3.4 域协调器职责

1. 接收绑定层转发的**类型化调用**（绑定签名不变，内部转发）；
2. 校验：原语可用、空间合法、资产路径必须落在空间允许根（`.gaea/play/exports`、
   `.gaea/exports`、ImageSaveDir 等登记根），跨空间注入拒绝；
3. 调用既有引擎/链（原实现函数原地保留）；
4. 成功产物 → 写 Asset Ledger（§4）并返回 AssetRef；
5. 用量/成本事件回报（模型中心用量视图 + 画室创作记录共用同一事件，§5.3）。

---

## 4. 溯源登记（Asset Ledger v1）

### 4.1 存储

- 按空间分文件、**append-only JSONL**：
  - play：`.gaea/play/imagehub/assets.jsonl`
  - work：`.gaea/imagehub/assets.jsonl`
- 不存 base64，只存路径 + 元数据；文件本体留在各既有落点（不搬家）。
- 上限建议 2000 条/空间，超出按最旧折叠（保留条数可配）；**折叠不删文件**，只丢
  索引条目（索引是辅助视图，不是文件真相）。

### 4.2 记录内容（AssetMeta + AssetRef 合并一行）

见 §3.2 类型；v1 必填：id/space/source_board/capability/backend/model/path/kind/
created_at/ai_flag。可选：prompt/params/cost/license_hint/refs/parent_id。

### 4.3 读写与安全

- `ledger.Record(meta)` / `ledger.List(space, filter)` / `ledger.Get(id)`；
- 路径安全：只接受登记根下路径（防穿越），非法路径拒绝登记并报错；
- 并发：单机单写者为主，写前 flaky 锁 + 原子改名（沿用项目 AtomicWrite 纪律）；
- JSONL 坏行跳过 + 计数（损坏不致命，索引可重建为空后重新积累）。

### 4.4 与既有登记的关系

- 办公 diagram/图表已进「交付物登记表」（事件日志折叠）；T0 登记与它**并列不重复**：
  ledger 管「图像域产物语义」（能力/参数/成本），交付物表管「agent 写盘轨迹」；
  T5 需要时再做视图合并，不在 T0 建桥。

---

## 5. 与模型中心的目录协议

### 5.1 只读目录视图（模型中心侧扩展，仍归模型中心）

```go
// 建议落 internal/app/modelcatalog.go（模型中心侧 owner；图像域只读）
type ImageModelView struct {
    EngineID   string   `json:"engine_id"`
    ModelID    string   `json:"model_id"`
    Kind       string   `json:"kind"`                 // 恒 image（或 diagram/vision 入口另列）
    Modes      []string `json:"modes"`                // txt2img|img2img|edit|video|ref
    Tier       string   `json:"tier"`                 // local_free|cloud_paid|free_fallback|remote_heavy
    UnitCost   string   `json:"unit_cost,omitempty"`  // "0" | "0.1 CNY/张" | "" = 未定价
    License    string   `json:"license_hint,omitempty"`
    Available  bool     `json:"available"`            // 引擎健康/模型就绪
}
```

- 引擎与健康来自现有引擎注册；模型清单来自现有 engines/models；
- `Modes/Tier/UnitCost/License` 用**静态能力映射表**维护（模型中心侧代码 +
  market-research 每 2 周轻扫更新），不做动态猜测；未知模型只给 kind 与 Available，
  其余诚实留空；
- 前端画室展示「能力徽标 × 档位 × 成本」读此视图，不另存。

### 5.2 图像选型解析（域 × 空间）

解析顺序：显式 model（调用方指定）→ 领域覆盖（`portrait_backend/model` 等既有）→
绘梦 `image_backend/image_model` → 引擎激活默认。T0 只在目录视图标注每条解析来源，
不改选择逻辑。

### 5.3 用量/成本事件

- 域协调器在每次 media.generate/edit/diagram 成功后发一条内部事件
  `imagehub.usage`（space/capability/backend/model/asset_id/duration/cost）；
- 模型中心用量与画室「创作记录」是两个只读视图，同一事件源；T0 先落事件类型与
  后端侧计数，前端面板 T1。

---

## 6. 存量管线 → 原语对照与接线状态

| 存量入口 | 原语 | 空间 | 接线状态 |
|---|---|---|---|
| GenerateFreeImage / GenerateMedia | media.generate | play（显式） | **T0 试点接线**（落盘后登记） |
| GenerateCharacterPortrait(WithRef) | media.generate（board=characterlib） | play | T1 收口 |
| GenerateSceneIllustration | media.generate（board=novel） | play | T1 收口（当前仅返回 URL 不落盘，登记与配图落盘同刀） |
| GaeaGenerateBookCover | media.generate（board=novel） | play | **T0 试点接线** |
| GenerateDiagram（ImageB / agent diagram） | media.diagram | 跟随调用方 | T1 收口 |
| GaeaOCRText | vision.read | work（office）/共享 | **T0 试点接线**（入口校验，行为不变） |
| GaeaRecognizeImage | vision.understand | work（office）/共享 | **T0 试点接线**（入口校验，行为不变） |
| 微信 visionOCRText | vision.understand（board=weixin） | work（触点默认） | T5 收口 |
| 造价 OCR/图表 | vision.read + diagram（board=cost） | work | T5 收口 |
| 粘贴/附件图 | （素材入域，非原语） | work | T5 收口 |
| 模型中心目录 | — | shared | T0 协议 + 视图 |

---

## 7. 试点接线方案（T0 交付即验证契约）

**试点 1：书封登记（低风险）**：`GaeaGenerateBookCover` 成功后写 ledger（space=play、
board=novel、capability=media.generate），返回路径不变。

**试点 2：绘梦工作台落盘登记（低风险，主生图原语）**：`GenerateFreeImage` /
`GenerateMedia` 每张成功落盘（`item.FilePath != ""`）后写 ledger（source_board=imagegen、
capability=media.generate、mode/seed/size 入 params、kind 按 image/video）；登记失败只
warn。返回结构与历史元数据不变。

**试点 3：识图原语收口（低风险）**：`GaeaOCRText` → vision.read、`GaeaRecognizeImage`
→ vision.understand，入口先经域能力注册表校验再落既有实现；返回值与引擎选择不变。
识图不产物，无登记（文本返回不入 ledger）。

> 章节配图（`GenerateSceneIllustration`）**不进 T0 试点**：实机确认其当前只返回 URL、
> 不落盘；登记没有资产可记。配图落盘 + 画室素材库 + 登记统一放 T1 收口。

> 试点只加「转发 + 登记」，不挪引擎、不改返回结构、不动存储根。失败登记不影响主流程
> 返回（记录失败仅 warn——登记是辅助视图，不能拖垮生成主路径）。

---

## 8. 绑定面变更清单

**T0 期望绑定面零变更**：

- 不新增 Wails 绑定（试点沿用现有绑定签名，内部转发）；
- `bindingNames.ts` / `spaceBindings.ts` / bridge 类型**零漂移**；
- 新增内部函数与类型只在 `internal/app`（建议 image_domain.go、modelcatalog.go、
  ledger 独立文件），不导出到 Wails。

预期后续（T1/T5，仅预告不实施）：`GaeaImageHubAssetsList(space)`、能力目录只读
绑定（供画室/办公消费）等，届时按流程同步绑定面 + spaceBindings 分类。

---

## 9. 测试与验收

### 9.1 单元/回归

- 能力注册表：五原语齐全、未知原语 fail-closed、edit=T0 Available=false；
- ledger：record/list/get、路径白名单防穿越、坏行容错、折叠上限；
- modelcatalog 静态映射：已知模型（krea2/z-image/flux/grok-imagine/glm 系）给出
  modes/tier/cost；未知模型诚实留空；未定价不伪装 0；
- 试点行为回归：书封/绘梦落盘既有测试全绿 + 登记断言；OCR/识图既有测试全绿
  （返回值逐字一致）。

### 9.2 全量门禁

- `go test ./...` 全绿；`tsc -b` / eslint 0；vitest 全绿；drift PASS（579 绑定不变）；
- 冒烟 200；`?mock=1` DOM 走查（绘梦/小说/角色库/办公识图页面无回归）。

### 9.3 手动验收

1. 乐园打开小说项目 → 生成书封 → `.gaea/play/imagehub/assets.jsonl` 出现该图登记
   （字段含 play/novel/media.generate/backend/model/path）；
2. 绘梦页文生图/图生图各生成一张 → 同样有登记（board=imagegen、含 seed/size/mode），
   历史列表与图片保存行为与 T0 前一致；
3. 办公「提取文字」/「识图」输出与 T0 前逐字一致；
4. 重启后 ledger 可读、坏行不炸；
5. 画室/小说/角色库页面无回归。

---

## 10. 回退方案

- 每试点独立提交可独立 revert；revert 后即回到原绑定直连实现（登记不残留副作用，
  最多留下已写 JSONL，可一并删除或忽略）；
- 无 DB 迁移、无数据搬家、无 schema 变更：JSONL 新文件即全部写入面；
- ledger 丢失只丢辅助索引，不影响任何既有历史/产物。

---

## 11. 风险与待拍板

| 项 | 判断 | 处置 |
|---|---|---|
| 试点选 3 条还是只做书封 1 条 | 3 条覆盖 play 产物（书封/绘梦）+ work 识图两类面 | 已按 §7 三试点落地；每试点独立提交可回退 |
| ledger 上限/折叠策略 | 2000 条/空间 v1 够用 | 可配；折叠不删文件 |
| 成本表维护 | 静态表会过时 | 归属模型中心侧 + 轻扫更新 + 「未定价」诚实展示 |
| 空间标签来源 | legacy 面不显式传空间 | 绑定层显式给（试点处写死 play/work），不给的入口一律 fail-closed |
| 与交付物登记表重复 | 语义不同（域产物 vs agent 写盘轨迹） | T0 并列；T5 再做视图合并评估 |
| 代码放 app 包 | app 已大 | T0 薄文件（≤3 个），观察 >5 调用方后抽 internal/imagehub |

---

## 12. 本设计对长期规划的修订

长期规划 §2.1 曾写「评估把 imagegen 纳入模型中心功能域绑定」——T0 盘点后**裁定不
纳入**：feature 域绑定面向文本 LLM；图像选型维持 image_backend/portrait 覆盖 + 四级
解析，模型中心只做目录 owner。已同步回填长期规划文档。
