# 双空间壳 S2.1 设计（前端两视图 + 空间切换 + 双首页）

> 依据：`docs/gaea-nextgen-roadmap-2026.md` §14 阶段 2 + §15 阶段 2 补丁。
> 前置：阶段 1（S1.1–S1.5 双空间内核）已收官（v3.8.0）。
> 目标：把「10 板块导航」壳层收敛为「工位 / 乐园」两视图；空间切换状态持久；
> 首页按空间拆双首页；store/localStorage 空间前缀迁移；事件订阅按空间过滤；
> Ctrl+K 命令/搜索面板提供显式 scope（默认当前空间）。

---

## 1. 现状与缺口

- 壳层 = `frontend/src/layouts/MainLayout.tsx`：左侧指挥轨道（CommandRail，10 板块
  全量菜单）+ 顶部轨道条 + 底部遥测轨道；首页 = `ModuleLauncher`（全部模块 Bento）。
- 板块 manifest 已清单化（`boards/manifests.ts` + 后端 `internal/app/board`），
  但**没有空间维度**——菜单/启动器/快捷键都不过滤空间。
- 后端空间维（S1）已就绪：`GaeaSpaceActive/List/Activate`（生效空间配置）、
  会话目录分区、记忆隔离、任务 `spaceId`、`UnifiedSearch(scope)`。
- 前端仅有 `SpaceChip`（gaea 工作台侧栏展示/切换生效空间）+ 记忆中枢 scope 切换；
  **壳层本身没有 currentSpace 状态**，两空间共用同一套导航与首页。

## 2. 设计原则

1. **壳层空间状态独立于后端生效空间**：壳层 `currentSpace`（本地视图状态，持久化）
   决定导航/首页/事件面；后端 `space.mode`/`session.space` 决定数据落点。二者默认
   一致（初始 work），互不同步也不破坏隔离（壳层只管呈现，后端管数据）。
2. **manifest 带空间归属**：每个板块声明 `space: work | play | shared`，导航/启动器/
   快捷键/导航事件白名单全部从 manifest 派生，杜绝第四处硬编码。
3. **旧数据按 work 兼容**：manifest 缺省 space → work（与 S1 旧数据回填 work 同语义）；
   旧 localStorage key 只读迁移，不破坏旧值。
4. **渐进迁移可回退**：单步独立提交；旧导航删除后仍可从 git 历史恢复。

## 3. 空间归属表（§10.3 处置表 → manifest.space）

| 板块 | space | 依据 |
|------|-------|------|
| home | shared | 壳层首页（按空间渲染不同内容） |
| chat | shared | §10.3 对话「共用」（S2.2 降为对话流） |
| novel | play | 乐园·小说 |
| imagegen | play | 乐园·绘梦 |
| characterlib | play | 乐园·角色资产 |
| gaea（办公） | work | 工位主体 |
| cost | work | 工位·造价领域包 |
| memoryhub | work | 工位·记忆中枢 |
| knowledge | work | 工位·知识库 |
| code | independent | **独立 DSH 窗口**（用户拍板：不并入工位、不共享工具面）——两空间导航/首页均不出现，壳层单独入口 |
| modelcenter | shared | 模型设置共用 |
| settings | shared | 全局设置 |
| weixin | work（缺省回退） | 工位触点（无前端页面，不进菜单） |

## 4. 改动面

### 4.1 manifest 空间字段（Go + TS 同构）

- Go `internal/app/board/manifest.go`：`Manifest.Space`（`json:"space,omitempty"`），
  `Validate` 允许 `"" | work | play | shared`。
- Go `internal/app/board/builtins.go`：按 §3 表赋值。
- TS `boards/types.ts`：`space?: 'work' | 'play' | 'shared'`。
- TS `boards/manifests.ts`：canonicalBoards 赋值；新增
  `getActiveMenuBoardsForSpace(space)`（派生视图按空间过滤）。

### 4.2 纯函数空间助手（boards/space.ts）

```ts
export type ShellSpace = 'work' | 'play'          // 壳层视图空间
export type BoardSpace = ShellSpace | 'shared' | 'independent'

boardSpace(b): BoardSpace          // b.space ?? (isHome ? 'shared' : 'work')
isBoardReachableInSpace(b, s): boolean  // shared 或等于当前空间；independent 恒 false
isIndependentBoard(b): boolean          // 独立窗口板块（rail 单独入口）
```

- `boards/launcher.ts`：`deriveLauncherModules(list, descMap, space?)`——可选空间过滤
  （不传 = 全量，保持既有调用与测试不变）。

### 4.3 壳层空间状态（appStore）

- `appStore` 新增 `space: ShellSpace` + `setSpace`；key `gaea.shell.space`，
  默认 `work`，写 localStorage（读取失败/非法值回退 work）。
- MainLayout 按空间持久化最后页面：`gaea.shell.page.<space>`；切换空间时
  当前页若在新空间不可达 → 回首页（home 两空间共享，渲染空间化首页）。

### 4.4 壳层两视图（MainLayout / CommandRail）

- CommandRail 顶部加「工位 / 乐园」切换（与 dark 切换共存）；
- 导航 = `getActiveMenuBoardsForSpace(space)`（shared + 当前空间）；
- Ctrl+1~9 目标 = 当前空间菜单；NAVIGATE 事件白名单叠加空间可达性校验；
- 内容区不变（visitedPages keepAlive 逻辑保留；home 渲染 ModuleLauncher）。

### 4.5 双首页（ModuleLauncher）

- `ModuleLauncher` 接收 `space` prop：
  - work：Hero 标题/行动卡 = 工位门面（进入办公工作台 / 造价 / 记忆）；
  - play：Hero = 乐园门面（开始创作 / 会客厅聊天）；
  - 模块卡 = `deriveLauncherModules(boards, LAUNCHER_DESC, space)`。

### 4.6 store/localStorage 空间前缀迁移

- 新增壳层 key 即带空间语义（`gaea.shell.space` / `gaea.shell.page.<space>`）。
- **S2.2 已落地**：工作台 localStorage 空间分键（`gaea/lib/workbenchStorage.ts`）——
  旧 key `gaea.<x>` → 新 key `gaea.work.<x>`（读优先分键、旧 key 只读回退迁移，
  写只走分键）。覆盖：rightTab 全局/会话键、layoutPreferences JSON blob、
  chatTab/compactMode/focusMode。

### 4.7 事件订阅过滤（events.ts）

```ts
subscribeForSpace(event, handler, space?)  // payload.spaceId ?? payload.space
```
空间不匹配的事件直接丢弃（payload 无空间字段 = 全局事件，放行）。
消费方：TaskCenter / useRunningBadge（gaea-task 事件只收当前空间任务），
前端 `TaskView` 补 `spaceId?` 字段（后端 Task.Space 已带 `json:"spaceId"`）。

### 4.8 Ctrl+K 搜索面板 scope

- SearchModal 加三档 scope（工位/乐园/全部，默认当前空间）：
  - play → 既有项目搜索（小说章节/角色）；
  - work → `GaeaUnifiedSearch(q, 8, 'work')`（记忆+文件语义面）；
  - 全部 → 两路结果合并展示。
- scope 选择不持久化（每次打开默认跟随当前空间；「全部」需显式选择，符合
  双空间红线「默认不跨空间」）。

## 7. S2.2 增量（页面迁入/i18n 第一刀/性能门控）

- **工作台空间分键**（§4.6）：`workbenchStorage.ts` + 旧 key 只读迁移（补丁项
  「store 与 localStorage 迁移」完成）。
- **性能门控**：空间切换时 `pruneVisitedForSpace` 剪枝 keepAlive 保活页——
  跨空间页面卸载，后台轮询/渲染归零（审计「keepAlive 挂载策略系统级后台轮询」
  风险项；同空间 keepAlive 不受影响）。
- **i18n 全铺第一刀**：`LocaleProvider` 提升到根级（壳层与页面共用 gaea 字典）；
  S2.1 新增的全部壳层 chrome 文案（空间切换/双首页 Hero/搜索 scope）进入
  zh/zh-TW/en 字典并消费；存量硬编码中文按同一模式逐步收敛（后续切片）。
- **页面迁入**：导航层已完成（S2.1 manifest.space）；「对话降为对话流 / 记忆降为
  侧栏 / 模型中心并入设置 / 创作间合并」属 v4.x 深层重构，随版本推进，不在本步。

## 5. 验收

1. 切到乐园：导航只剩 shared+play 板块，首页为乐园门面；切回工位恢复工位首页。
2. 刷新/重启后空间与每空间最后页面保持（localStorage 持久化）。
3. 工位内 Ctrl+K 默认「工位」scope；乐园内默认「乐园」。
4. gaea-task 事件：play 任务不出现在工位任务中心（TaskCenter 过滤）。
5. 旧 manifest（无 space 字段）按 work 兼容，壳层不崩。
6. Go `go test ./internal/app/board` + vitest 新增用例全绿；tsc/eslint 0。

## 6. 不做（留到后续 Step）

- S2.2 页面迁入与「对话降为对话流」、gaea 工作台按空间分键、i18n 全铺。
- S2.3 bridge 分面 + types 生成化。
- 主题「工位冷色 / 乐园暖色」氛围包（UI 层，S2.2 随页面迁入评估）。
