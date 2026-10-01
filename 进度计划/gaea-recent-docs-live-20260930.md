# 修复：首页「最近文档」不实时更新（2026-09-30）

> 用户报告：「首页的最近文档根本没有实时更新」。定位=**记账缺口**（打开文档的
> 两条主路径不入账），不是订阅/渲染问题；已修 + 4 例回归测试 + 真机路径走查。

## 1. 诊断（先证伪再定因）

订阅链路实测是好的：在 dev 页里直接调真正的 `recordRecentFile`，
首页卡片**立即**出现新条目（`.w-docs-name` 列表首项变化，无需刷新）——
`useSyncExternalStore(subscribeRecentFiles, …)` + 事件广播 + 快照引用稳定
三段都成立（v4.349 的实时化机制有效）。

所以问题在**写入侧**：`recordRecentFile` 只在 4 处被调用——
`useComposerMenus`（@ 菜单选文件）、`ExplorerView.reference`（右键「引用 @」）、
`usePreviewPanel.openFilePreview`（主区预览）、`RecentFilesBar`（点最近条）。
而用户最常走的两条「打开文档」路径**根本没记账**：

| 路径 | 入口 | 修复前 |
|---|---|---|
| 资源管理器行点击 / 产物·变更·Git 行内打开 | `usePaneTabsStore.openFile`（sidebarRegistry.openPaneFileTab / usePreviewPanel.openPaneFile） | ❌ 只开 pane 页签 |
| 右键预览 / 正文文件链接 / 记忆中枢·进度计划·DAG·上下文检查器文件行 / 附件条 | `usePreviewStore.openFilePreview` | ❌ 只入预览队列 |

即：**只有「引用」和「主区预览」两种冷门操作会进最近文档**，日常点开文件不动。

## 2. 修法（单点记账，不加调用方负担）

把记账收敛到两个「打开文件」的单一入口，调用方零改动：

1. `gaea/lib/paneTabs.ts` → `openFile(path, title)`：pane 文件页签即「打开文档」，
   打开即 `recordRecentFile(path, title)`（覆盖资源管理器行点击 / 产物 / 变更 /
   Git / 最近条四个消费方）。
2. `gaea/lib/store/preview.ts` → `openFilePreview(rel)`：预览队列即「打开文档」，
   打开即 `recordRecentFile(rel)`（覆盖右键预览 / 正文链接 / 记忆中枢 / 进度计划 /
   DAG / 上下文检查器 / 附件条）。
3. `gaea/app/usePreviewPanel.ts` 里原有的重复记账删除（同一次打开曾写两遍）——
   单一真源=上述两个 store 动作。

**顺带修**：首页「最近文档」点行原本只 `onNavigate('gaea')`（跳到办公板块，
不开该文档），与自身 hint 文案「在办公中打开」不符。改为
`openPaneFileOrPreview(f.path)` + 导航：办公工作台已挂载 → 开 pane 文件页签；
未挂载 → 落预览队列，导航过去即渲染该文件（`paneFileOpen` 是既有的统一入口）。

## 3. 验收

| 判据 | 证据 |
|---|---|
| 订阅链未坏（先证伪） | dev 页直接调 `recordRecentFile` → 首页 `.w-docs-name` 首项即时变化 |
| pane 路径 | dev 页调 `usePaneTabsStore.openFile` → 首页首项变「文件树点击探针.docx」✅ |
| 预览路径 | dev 页调 `usePreviewStore.openFilePreview` → 首页首项变「预览通道探针.docx」✅ |
| 点行真打开 | 点首页「点击打开探针.docx」行 → 导航到办公 + 右侧「文件 · 点击打开探针.docx」预览面板（截图 `.tmp/home-shots/click-after-office.png`）✅ |
| 单测 | `paneTabs.test.ts` +3（记账/去重置顶/空路径）、`previewQueue.test.ts` +1（记账+置顶） |
| 全量 | `tsc -b` 0 错；`eslint` 0 error；**vitest 410 文件 / 3568 例全绿** |

## 4. 遗留（未做）

- 首页最近文档点行**不置顶**该条（打开后它已在榜首，视觉上无差别；如要显式置顶
  可在点行时补一次 `recordRecentFile`——当前由上述两个 store 动作隐式完成）。
- 目录条目（isDir）不入最近文档（既有语义，未改）。
