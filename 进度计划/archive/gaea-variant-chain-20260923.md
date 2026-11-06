# gaea 绘梦·阶段二刀 D：编辑历史变体簇（ParentID 链 + 溯源）规格书

> 2026-09-23 立项。来源：用户指令「继续」（画室线既定刀序：T3 编辑力第四件，
> 不依赖新权重）。上游：长期规划 T3「编辑历史形成『变体簇』（同源图关系），
> 支持回到任一变体继续改」；T0 契约 AssetMeta 的 `ParentID` 预留位本次激活。
> 目标版本：v4.395.0。

## 1. 论点

编辑三件（指令编辑/蒙版/扩图）落地后，编辑行为天然成链：A → A' → A''。但现状
链不可见——台账（v4.98）有登记无关系字段，前端结果与历史无变体标识，用户改了
三轮后无法回到第 2 版继续分叉。「回到任一变体继续改」的编辑面已天然支持
（任何画布图都可发起编辑），缺的是**关系数据与呈现**。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 关系键 = 台账 `parent_id`（激活 T0 预留位）：编辑/扩图请求带 `sourcePath`
（源图 file_path），app 层按 path 在台账同空间查最近条目 id 作 ParentID；查不到
（源图非 gaea 产物/跨空间）留空=诚实（链条断在导入图，不造假） |
| Q2 | `recordImageHubGeneratedAsset` 签名 +parentID（4 生产调用点：chapter/
portrait/bookcover 传 ""，imagegen 传查找值）；meta/view 双侧加字段，旧 JSONL
无字段反序列化零值空=向后兼容 |
| Q3 | `imageItem` +`asset_id`/`parent_id` 回传：登记后按 path 反查自身 id 回填
（lookup 复用 Q1 的 byPath；运行态闸关闭时查得空=字段空，不影响主流程） |
| Q4 | 前端：GenResult/MediaParams 透传（`sourcePath` 提交、`asset_id/parent_id`
接收）；InstructionEditModal 编辑产物带 parent_id 时 meta 行显示「变体」徽标 +
「溯源」按钮打开 VariantChainModal |
| Q5 | VariantChainModal：拉 `imageHubAssets` 建链（沿 parent_id 上溯至根，倒序
展示 根→…→当前），每项缩略图（`readFileAsDataURL`）+模型/时间/指令摘要+「用到
画布」（onApply 构造 GenResult 并入画布=回到该变体继续改）；链断（祖先缺档）如实
标注 |
| Q6 | 零新绑定（714 不变）：读走既有 ImageHubAssets 只读绑定 |
| Q7 | sin 板块登记调用点补 ""（sin 编辑链不在本刀——sin 插图无编辑入口） |

## 3. 线 A：internal/app

- `imageHubAssetMeta`/`imageHubAssetView` + `ParentID`（json parent_id）。
- `imageHubAssetIDByPath(cwd, space, path) string`：ledger list 倒序找 Path 命中
  返回 ID（台账上限 2000/空间，线性扫可接受）。
- `recordImageHubGeneratedAsset(..., parentID string)`；`recordImageHubGeneratedFor
(..., parentID string)`。
- GenerateMedia：`p.SourcePath`（mode=edit/outpaint 消费）→ lookup → 登记 parentID；
  登记后 `item.AssetID/ParentID` byPath 回填。

## 4. 线 B：前端

- `api/image.ts`：`ImageHubAssetView`+parent_id；`MediaParams`+sourcePath；
  generateMedia 结果映射补 asset_id/parent_id。
- `types.ts`：GenResult+asset_id?/parent_id?。
- 新 `VariantChainModal.tsx`（见 Q5）。
- `InstructionEditModal.tsx`：run() 带 `sourcePath: source.file_path`；edited 带
  parent_id 时 meta 行「变体」+「溯源」按钮。

## 5. 测试

- Go：①meta ParentID 往返（record 带 parentID → list 出）；②byPath 查找（命中/
  未命中/倒序取最新）；③GenerateMedia edit+sourcePath（闸开态：登记链 parent 落
  位、item 回填 id；闸关态零 panic 字段空）。
- 前端：VariantChainModal（mock imageHubAssets 三代链+断链；链序/摘要/用到画布
  onApply）；Modal 集成（parent_id 徽标+溯源按钮派发）。

## 6. 验收与门禁

定向+全量 ci.ps1、tsc/eslint/vitest、drift PASS@714；真机走查挂池（与编辑班同批）。

## 7. 观察池（本刀不做）

HistoryRail 簇折叠/缩进呈现；跨空间溯源；sin 编辑链；变体簇导出；按簇删除；
抠图/透明底（T3 最后一件）。
