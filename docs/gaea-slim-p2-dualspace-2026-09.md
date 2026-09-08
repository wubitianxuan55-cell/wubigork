# gaea 瘦身 P2 双空间并列落地审计（2026-09-09）

> **状态=审计证据（2026-09-09，v4.169.0 收口时落档；证据行号随版漂移属正常）**
> **关系=masterplan 执行层/本文=P2 主干双空间证据层**（权威=docs/gaea-slim-masterplan-2026-09.md §轨道一·2/3）
> 前置=刀0 白名单解耦（v4.167）+ 刀1/刀2 schedule 并入办公（v4.168）；本文覆盖 P2 主干剩余：
> **双空间并列落地（切换器+rail 分域导航）→ home 空间感知化 → 走查待证项代码层收口 → 刀2 G-2 关闭**。

## 0. 口径

- P2 出口判据（masterplan §2）：工位 ≤6 且乐园 ≤4 一级导航；等价快照表全绿。
- 拍板红线：工位/乐园**并列平级**；藏≠删、合≠砍；引擎零改动；零绑定（本版无 Go 产品改动）。
- 本版仅代码层落地实证；壳内真机走查（WebView2）另列 §6 待真机清单。

## 1. 线A：rail 双空间分域导航 + 工位/乐园切换器

> 落地证据（v4.169.0 实测）：
> - `frontend/src/layouts/MainLayout.tsx`：CommandRail 增 `space`/`onSwitchSpace` props（命名导出）+
>   rail 顶部切换器（SHELL_SPACES 驱动，`t(labelKey)||label` 文案、`t(titleKey)||title` tooltip、
>   aria-pressed 激活态、data-testid=v3-rail-space-switch / v3-rail-space-<id>）；
>   rail 主体改 `getActiveMenuBoardsForSpace(space)`（shared 恒在、independent 剔除、
>   settings inMenu:false 不在 rail）；MainLayout `onSwitchSpace={switchSpace}`（既有
>   saveShellPage+setSpace+loadShellPage+pruneVisitedForSpace 全链直连，零新增状态逻辑）。
>   **code 双入口消解**（基线 §3.1 勘查项关闭）：independent 只经 foot 单列，
>   CommandRail.test 断言 `getAllByLabelText(/编程/)` 全长=1 且位于 `.v3-rail-foot`。
> - `frontend/src/v3/foundation.css`：`.v3-rail-space` + `.v3-rail-space-opt`（两态胶囊 11px：
>   active=primary-container+gaea-glow 辉光+glow-faint 柔光；hover 微缩放；focus-visible 描边；
>   全 token 零硬编码色）；更新 v4.3.2c 陈旧注释为 v4.169 双空间语义。
> - `frontend/src/boards/manifests.test.ts` +4：getActiveMenuBoardsForSpace work/play 精确派生序
>   （work=[home,gaea,cost,memoryhub,modelcenter,weixin] / play=[home,chat,novel,imagegen,
>   modelcenter,characterlib]）、code/settings/schedule 双侧剔除、各 6 项+中文标签锁定。
> - 新 `frontend/src/layouts/CommandRail.test.tsx` +5：切换器渲染/激活态/work 分域/play 分域/
>   编程单入口/点击回调（LocaleProvider 钉 zh，tools.test 先例）。
> - shell.space.* 三语值由旧语义「书房/庭院」更正为「工位/乐园」+新 title
>   （主代理统一落字典，契约键单一负责人；zh/en/zh-TW 同步）。

## 2. 线B：home 空间感知化（Bento 分域 + 工位最近文档）

> 落地证据（v4.169.0 实测）：
> - `frontend/src/boards/launcher.ts`：新增 `LAUNCHER_FEATURED = { work:'gaea', play:'chat' }`
>   （每空间旗舰锚点；板块不在空间清单时自然查不到→既有条件渲染兜底）。
> - `frontend/src/components/ModuleLauncher.tsx`：Bento 按当前空间过滤
>   （`deriveLauncherModules(activeBoards, LAUNCHER_DESC, space)`——shared 恒在、
>   **independent 编程不再上首页**、settings 两空间皆在）；旗舰按 LAUNCHER_FEATURED 查表；
>   hero 空间 chip（`t(labelKey)||label`，data-testid=ml-space-chip）；**工位右舷「最近文档」面板**
>   （loadRecentFiles 前 5，点击→onNavigate('gaea')，title=home.recentDocsHint，零新 binding）；
>   乐园右舷维持（会话+写作进度）；晨报本就 work-only。宽瓦片/独立窗口徽标代码随编程移出首页
>   移除（BentoCard wide 能力保留注释说明）。
> - `frontend/src/components/module-launcher.css`：`.ml-hero-tags`+`.ml-space-chip`+`.ml-recent-item`
>   （token 化，注释同步双空间语义）。
> - `frontend/src/boards/launcher.test.ts` +3：LAUNCHER_FEATURED 值/键全集、work Bento 包含
>   （gaea/cost/memoryhub/weixin/modelcenter/settings）排除（chat/novel/imagegen/characterlib/code）、
>   play Bento 反向对称。

## 3. 线C：走查待证项代码层收口 + 刀2 G-2

- **knowledge 孤儿页处置**（走查待证项③）：`frontend/src/main.tsx` 移除 KnowledgePage 注册 + 删除
  `frontend/src/pages/KnowledgePage.tsx`（全仓 grep 实证仅剩两条测试 fixture 字符串字面量；
  tsc -b 全项目通过=被删模块零引用）；知识库功能=记忆中枢 KnowledgePanel 子面等价；
  后端 D7 板块保留（normalizeManifests 前端无条件过滤，导航侧不出现）。
- **刀2 G-2 基线数计数关闭**：GschedSummary 增 `baselineCount`（project.baselines 长度，
  v4.137 #11 多基线 max3；未保存=0）；ScheduleFileCard 增「基线 N」StatChip（baselineCount>0 时，
  活跃基线快照+漂移卡体保留）；gschedSummary.test +3（多槽/空/坏形状）、ScheduleFileCard.test +2。
- **后端 nav 子项差异（走查待证项②）落档**：见 §4（双源回退 by-design，后端字段优先，不改码）。

## 4. 后端 nav 子项差异（代码取证，只读）

> **对比结论（2026-09-09 实测）**：
> | 板块 | 后端 builtins.go Nav | 前端静态 Nav（manifests.ts） | 差 |
> |---|---|---|---|
> | cost | 7（概览/成本条目/**测算项目**/**造价参考**/**复盘笔记**/价格源/价格仓库，builtins.go:76-81） | 4（概览/成本条目/价格源/价格仓库，COST_NAV:62-67） | 后端多 3 |
> | novel | 6（书架/设定/角色/创作/阅读/**导出**，builtins.go:38-42） | 5（书架/设定/角色/创作/阅读，NOVEL_NAV:42-48） | 后端多 1 |
> | knowledge | 独立板块 Page=KnowledgePage（builtins.go:148-155） | 静态清单无；normalizeManifests 无条件过滤（manifests.ts:254） | 导航侧不出现 |
>
> **结论=有意的双源回退，无需改码**：`normalizeManifests` 对 nav 取 `r.nav ? r.nav : base?.nav`（后端字段优先，
> manifests.ts:270），运行时以后端为准；前端静态 NOVEL_NAV/COST_NAV 仅在后端未下发 nav 时兜底，
> 属陈旧回退基线（含设置页 SETTINGS_NAV 9 项与后端一致）。后端多出的子面（cost 测算项目/造价参考/复盘笔记、
> novel 导出）是真实生效面，前端静态清单缺项不构成功能缺口——兜底路径下不出现 ≠ 运行时丢失。
> 已封口：不新增前端静态条目凑数（会与后端优先语义打架），后端为唯一运行时事实源。

## 5. 度量看板（masterplan §3 六行，v4.169 实测）

> 每空间一级板块数（工位/乐园）｜ entry chunk gz (KB) ｜ 冷启动可交互 (ms) ｜ 常驻内存 (MB) ｜ exe (MB) ｜ >50KB 源文件数
> - 工位 4（gaea/cost/memoryhub/weixin）· 乐园 4（chat/novel/imagegen/characterlib）——home+shared 壳层不计入，双双达标 ≤6/≤4 ✓
> - entry gz **375.96 kB**（基线 375.36，净 +0.6：rail 切换器+home 空间 chip/最近文档面板；vite build 27.93s 实测）
> - 冷启动/常驻内存待 P4 建基线（真机）；exe 46.2MB 系（strip 属 P4）；>50KB 源文件 16（本版未触结构面）

## 6. 真机待走查清单（本版不做，如实挂账）

- rail 切换器/分域导航壳内渲染实态（WebView2：图标/切换器高度/自动隐藏 dock）；
- .gsched 摘要卡「基线 N」chips 壳内布局；
- knowledge 孤儿页删除后无残留入口（后端 GetBoardManifests 仍含 knowledge，导航侧过滤）；
- home 右舷「最近文档」面板壳内渲染（localStorage 生效性）。

## 7. 审计边界

- 只读审计基线（docs/gaea-slim-baseline-2026-09.md §3）已重走查：§3.1 rail 双入口/§3.3 首页平铺/§3.4 并列缺口
  均在本版闭合或转真机清单；等价快照表 13 板块可达性不变。
- 本版零 Go 产品改动、零绑定；引擎（schedule/cost）零改动。