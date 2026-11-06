# gaea t6-C2 提示词工坊·第二刀（模板包导入导出）规格书

> 2026-09-16 立项。来源：用户指令「继续优化迭代 gaea」。
> 蒸馏依据：`docs/distill/06-prompt-workshop.md` §6.4（导出/导入协议 + 内容哈希对账
> 三态算法——「本域最有借鉴价值的一段算法」）+ §11.4 裁决（社区分享裁剪为本地包）。
> 前置：t6 首刀 v4.323.0（三级解析/五绑定/工坊面板）已发布。
> 目标版本：v4.324.0。

## 1. 论点

首刀的覆盖表是「只有一份」的本地状态——换机/重装/多设备迁移时自定义模板
（prompt_overrides.json）没有搬运手段；也没有对内置基线升级后的对账工具。
本刀补**模板包导出/导入**：全量快照 JSON 包 + MuMu §6.4 三态导入算法，
同时是「导出旧版本 → 内置升级 → 导入」场景下识别「哪些该转自定义/哪些该
回落内置」的迁移对账机制。

对 MuMu 的升级/裁剪：
- **升级**：导入行过 `promptstore.Validate` 真校验（error 级跳行不阻断整包，
  记入结果）——MuMu 导入零校验、云端人工审核当唯一闸门（§7 信任边界），
  单机没有审核链，坏模板会直接毒化生成链路，必须本机自校验。
- **升级**：重复键跳过（skipped_duplicate，包内首见生效）——MuMu 无去重。
- **裁剪**：未知键不建行（skipped_unknown）——工坊 V1 不支持自建新模板键
  （首刀观察池），Save 侧同口径拒「未知模板键」。
- **沿用**：哈希只是导出侧冗余诊断，导入对账用**内容逐字比对**
  （canonical JSON 比对，gaea Template 是结构体，序列化即规范化文本；
  MuMu 同口径 strip() 后比对不用哈希）。

## 2. 未决问题裁决

| # | 裁决 |
|---|---|
| Q1 | 导出范围=引擎 Names() **全量**（含内置行）——备份语义，非仅自定义行（MuMu 43 全量同款） |
| Q2 | 行内容：覆盖行存在（无论启停）取覆盖行 Content 保真备份；否则基线 Content。IsCustomized=覆盖行存在 |
| Q3 | 导入不信包内 version（防倒退）——Version 走本地 Upsert 自增，包值仅快照展示 |
| Q4 | 导入 IsActive 用包行值（内置行导出恒 true；覆盖行如实） |
| Q5 | 包格式=明文 JSON（无用户隐私数据，仅模板正文）；version=1 |
| Q6 | 导出落盘走前端 `saveExportBlob` 双门（壳内 GaeaSaveFileAs / 浏览器 a[download]，v4.162 先例）；绑定只回 JSON 字符串 |
| Q7 | 导入选文件壳内走 `pickFileAsFile(['json'])`（GaeaPickFiles 系统对话框 + 后置扩展名校验），浏览器动态 input 回退（SkillModal 刀C-3 同款） |

## 3. 线 A：internal/promptstore/bundle.go（零 IO 纯函数）

```go
// ContentHash 导出侧冗余诊断哈希：sha256(TrimSpace(s)) hex 前 16 位（MuMu 同款算法）。
func ContentHash(s string) string

// SameTemplate 内容逐字比对：canonical JSON 序列化相等（map 键序 Go 序列化稳定）。
func SameTemplate(a, b prompt.Template) bool

type BundleTemplate struct { // 包内单行
    Key, Category, Description string
    Content                    prompt.Template // 覆盖行内容（存在时）或基线内容
    IsActive, IsCustomized     bool
    SystemContentHash          string // 当前本地基线 canonical JSON 哈希（诊断冗余）
    Version                    int    // 快照版本（导入不信，Q3）
}
type BundleExportStats struct{ Total, Customized, SystemDefault int }
type ExportBundle struct {
    Version    int              // 1
    ExportedAt int64            // unix ms
    Templates  []BundleTemplate // 引擎 Names() 字典序
    Statistics BundleExportStats
}

// BuildBundle names × 覆盖表 × 基线查找闭包 → 包（纯函数；Q1/Q2 口径）。
func BuildBundle(names []string, entries []Override,
    baseline func(string) *prompt.Template, nowMs int64) ExportBundle

type BundleImportStats struct {
    Total, KeptSystemDefault, ConvertedToCustom, CreatedOrUpdate int
    SkippedInvalid, SkippedUnknown, SkippedDuplicate              int
}
type BundleImportOutcome struct{ Key, Action, Reason string }
type BundleImportResult struct {
    Applied  bool // 至少一行变更（落盘判据）
    Statistics BundleImportStats
    Outcomes []BundleImportOutcome
}

// ImportBundle 三态导入决策+合并：entries 进 → 新表+结果出（不落盘，线 B 落）。
// 三态（MuMu §6.4 表 + gaea 两道闸）：
//   IsCustomized=false + 基线存在 + SameTemplate → 删覆盖行 kept_system_default
//   IsCustomized=false + 基线存在 + 不同         → Upsert 覆盖行 converted_to_custom
//   IsCustomized=true  + 已知键                  → Upsert 覆盖行 created_or_updated
//   未知键（engineHas=false）                     → skipped_unknown（不建行）
//   Validate error 级（写行路径前置）              → skipped_invalid（行级跳过）
//   包内重复键（NormalizeKey 后）                  → skipped_duplicate（首见生效）
func ImportBundle(b ExportBundle, entries []Override,
    engineHas func(string) bool, baseline func(string) *prompt.Template,
    nowMs int64) (merged []Override, res BundleImportResult)
```

action 常量：`kept_system_default` / `converted_to_custom` / `created_or_updated`
/ `skipped_invalid` / `skipped_unknown` / `skipped_duplicate`（MuMu 统计名沿用）。

## 4. 线 B：internal/app/gaea_prompt_store.go 追加 + NovelB 门面

- `PromptBundleExport() (string, error)`：BuildBundle（引擎 Names + 覆盖快照 +
  基线闭包）→ MarshalIndent 回 JSON 字符串。引擎未初始化回 error。
- `PromptBundleImport(bundleJSON string) (BundleImportResult, error)`：
  解析（坏 JSON/非 1 版本报 error）→ ImportBundle（engineHas=引擎 Names 集、
  baseline=基线闭包）→ Applied 时原子落盘+invalidatePromptOverrides → 回结果。
  Applied=false（空包/全跳过）不落盘不算错。
- 门面：bindings_novel.go 按字典序补 `PromptBundleExport`/`PromptBundleImport`
  两行委托。绑定面 701→703。

## 5. 线 C：前端三件

- bridge/novel.ts：AppBindings 接口 +2 方法；载荷视图
  `PromptBundleImportResultView`（statistics/outcomes 数值与字符串数组，
  浏览器环境直接消费）。
- novel/api/prompt.ts：`exportBundle(): Promise<string>`（原文 JSON 字符串）、
  `importBundle(text: string): Promise<BundleImportResult>` 包装 + 类型。
- PromptWorkshopPanel：标题栏右侧「导出模板包 / 导入模板包」两钮：
  - 导出=exportBundle → Blob → `saveExportBlob(blob, 'gaea-prompt-bundle.json')`；
    取消静默。
  - 导入=双门选文件（Q7）→ 读文本 → importBundle → 结果 Modal（统计行 +
    outcomes 表：action 中文标签 Tag，skipped_* 橙/红标 + reason）→ Applied 后
    refreshList + 定向刷新当前选中键。
- mock/novel.ts：+2 名（PromptBundleExport 回内置两行包 JSON 字符串；
  PromptBundleImport 解析回统计+outcomes 全 created_or_updated 的演示结果）。
- 测试：mock-contract-prompt.test.ts 补两名；PromptWorkshopPanel.test.tsx 补
  导入导出交互（mock api 层）。

## 6. 验收与门禁

- 线 A 表驱动矩阵：三态四行 + 未知/重复/校验跳过 + 停用覆盖行导出保真 +
  BuildBundle 统计正确 + SameTemplate 稳定（map 序）+ ContentHash 长度/确定性。
- 线 B：round-trip（导出→清 DataRoot 状态→导入→覆盖行回魂+引擎 Get 生效）、
  坏 JSON 报错、空包 Applied=false 不落盘。
- 门禁：`go build ./...` + `go test ./internal/promptstore ./internal/app` +
  `tsc -b` + eslint + vitest 全绿 + `gen_bindings -names` 重生成 + drift PASS。

## 7. 出口对照（本刀完成判据）

- [ ] 导出的 JSON 包在干净环境导入后覆盖层逐字节回魂（生成链路同效果）；
- [ ] 内置升级场景（改基线后导入旧包）三态各分支有测试钉死；
- [ ] 坏模板行不落盘且整包其余行正常导入（信任边界）；
- [ ] 绑定面 703 三处同步（Go 门面/bindingNames/bridge 接口）；
- [ ] 壳内导出落盘/导入选文件走系统对话框（双门，不走死通道）。

## 8. 观察池（本刀不做）

项目级覆盖（scope）；自建新模板键；系统模板版本升级提示合并交互（Q5 首刀池）；
按书切换配方；Skill triggers 匹配；模板包签名/加密（单机明文够用）。
