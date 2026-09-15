# gaea t4-C3 余项 · 重写版本历史面板（版本库前端消费）· 实施规格（v4.320.0）

> 来源：todos.md 小说线「C3 余项=partial 局部重写+场景工程+版本历史列表面板」；
> 现状核实：v4.304 版本库五绑定中 `NovelListRewriteVersions`/`NovelGetRewriteVersion`
> **零 UI 消费**（RewriteModal 只用当前 run 的 Apply/Discard/Restore）——版本库
> 只活在后端，跨会话找回历史版本用户到不了（v4.305「用户到不了的功能等于没有」
> 同型）。纯前端刀（后端零改动、绑定面 696 不动、锁数不动）。

## 0. 论点

整章重写产生的版本（含原文快照+审计）落了盘，但只有「刚重写完」那一刻的
弹窗能操作它；关掉弹窗/换会话后，历史版本既不可看也不可用。本刀补「重写
历史」面板：按章列出全部版本 → 展开看指标与新文预览 → 应用/放弃/恢复原文
（后端状态机门控原样暴露，前端不重复裁决只镜像禁用）。

## 1. 组件 `RewriteHistoryPanel.tsx`（frontend/src/components/novel/）

- props：`{open, chapterNum: number|null, onClose, onApplied?}`；antd Modal，
  文案硬编码中文（小说域口径，RewriteModal 同款，不走 i18n）。
- open → `NovelListRewriteVersions(chapterNum)`（时间倒序）；loading Spin；
  空态 Empty（「该章还没有重写版本」）。
- 行=时间 + 模式 Tag（整章/局部/去味）+ 状态 Tag（排队中/重写中/失败/已完成/
  已应用/已放弃）+ 相似度%（有则显示）；点击行展开详情（懒拉
  `NovelGetRewriteVersion`）：指标行（原文字数→新文字数、相似度、AI 味分
  before→after）+ 新文预览滚动区 + 原文折叠（Collapse，防长文撑爆）。
- 动作按状态镜像后端门控：
  - completed → 〔应用此版本〕（Popconfirm，覆盖正文）+ 〔放弃版本〕；
  - discarded → 〔应用此版本〕（后端明确 discarded 可再应用——MuMu 升级语义）
    不给放弃；
  - applied → 〔恢复原文〕；
  - pending/running/failed → 无动作（历史留痕只读）。
- 动作成功 → message + 重拉列表；应用/恢复成功另调 `onApplied()`（父级刷新
  正文，与 RewriteModal onApplied 同口）。

## 2. 入口（frontend/src/pages/CreatePage.tsx）

- rail「整章重写」按钮后加 `重写历史`（size small 同款；无徽标计数——首屏
  零额外请求，打开面板才拉）。

## 3. mock（frontend/src/gaea/lib/mock/novel.ts）

- `NovelListRewriteVersions` 返回两条样本（applied 整章 + completed 局部），
  `NovelGetRewriteVersion` 返回配套全文样本（开发预览）；Apply/Discard/
  Restore 保持抛错（浏览器端无真实库，诚实）。

## 4. 测试

- `RewriteHistoryPanel.test.tsx`：vi.mock bridge——①列表渲染（模式/状态
  Tag、相似度）②completed 行动作（应用走 Popconfirm 确认后调用+刷新）③
  applied 行只有恢复、discarded 行只有应用④空态。④详情懒拉（点击行后
  GetRewriteVersion 调用一次）。
- CreatePage 增量：入口按钮存在（若既有 CreatePage 测试文件有 rail 断言则
  增一行，无则跳过——查实况）。

## 5. 收口清单（v4.320.0）

tsc -b / eslint 改动文件 / 定向 vitest → ci.ps1 恰一次 → 版本三处
（sync-version）→ build → exe/SUMS/桌面副本/冒烟 → releases/v4.320.0.md +
CHANGELOG/README（两处版本表）→ .gaea AGENTS 迁 1 插 1 + progress + todos
→ commit+tag。零 Go 改动：gen_bindings 不跑（drift 复核即可）。

## 6. 出口对照与观察池

- 判据：版本库五绑定全部有 UI 消费 ✅（本刀补齐 List/Get；Apply/Discard/
  Restore 第二入口）；跨会话找回历史版本可达 ✅。
- 观察池：版本对比双栏 diff 视图（V1 只做单侧预览+指标）；partial 局部重写
  入口（引擎字段已契约先行，UI 随 partial 刀）；按模式/状态过滤。
