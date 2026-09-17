# gaea 7.3-2 板块降级为任务视图（阶段七「层跃升」刀）规格书

> 2026-09-17 立项。来源：用户指令「开始吧」（在 7.3-2 解锁讨论后明确提前
> 拍板——原窗口 ≈09-23）。规格 docs/gaea-stage7-plan-2026-09.md §3 7.3-2；
> 差异化身位依据 docs/gaea-platform-market-survey-2026-09-15.md §1.1
> （任务持久化红海、「多入口统一收件箱+板块降级为任务视图」无人发布）。
> 目标版本：v4.332.0。**形态标志：本刀合入的 release=阶段七「层跃升」版本
> （nextgen §10 口径；版本号续 v4.x 列车，跳号与否已由用户默认不跳）。**

## 1. 论点

现状首页=板块入口优先（书斋 w-cap 能力区 FeaturedBand+案牌；闲庭同构），
打开应用第一问是「用哪个工具」；任务收件箱只是右舷第五节。7.3-2 翻转：
**收件箱+最近上下文成为首屏，板块入口降为「能力的视图」**——打开第一问
变成「什么在等我」。四家大厂已验证「任务=持久单元」，但无人发布收件箱
第一界面形态（调研 §1.1 空白）。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 形态开关 `homeLayout: 'classic' \| 'tasks'`：appStore 持久化（localStorage，SHELL_SPACE_KEY 同款范式）；**缺省 classic**——渐进：现有体验零变化，「层跃升」由开关启用 |
| Q2 | TaskInboxPanel 抽**内联 TaskInboxBoard**（四档 tab+新建+状态机动作+跳转原样），Modal 壳保留包它——面板 props/DOM/既有 9 用例零改动 |
| Q3 | TasksFirstHome（两空间共用、space 感知）：首屏=TaskInboxBoard 大区 + 最近会话（SessionList 复用）+ **能力视图**=全部板块紧凑 chip 行（manifest 数据源不动，onNavigate 直达——藏≠删）+「切回经典首页」按钮 |
| Q4 | classic 首页零改动，仅在 SpaceSwitch 行加「任务优先」入口按钮（两首页共用于该组件旁） |
| Q5 | 回退开关双位：设置页 HomeLayoutPanel（首页形态：经典/任务优先，ThemeOrb Select 同款交互）+ 任务首页「切回经典」钮（判据③） |
| Q6 | 零新绑定（GaeaTaskInboxList/Save/SetStatus/Delete 复用）；manifest/pageRegistry 结构零改动（形态合并引擎零改动原则） |
| Q7 | i18n 三语 +N 键 zh=en=zh-TW 逐键相等 |

## 3. 落地

- `stores/appStore.ts`：+HomeLayout 类型/态/setter/persist loader（classic 合法值外回退 classic）。
- `gaea/components/TaskInboxPanel.tsx`：抽 TaskInboxBoard 导出。
- 新 `components/TasksFirstHome.tsx`：头部（空间印章/标题/lede+切回钮）+
  收件箱大区 + 最近会话 + 能力 chips + 底部口径脚注（纯同步：关闭即停）。
- `components/ModuleLauncher.tsx`：导出 useLauncherData/SessionList 复用件；
  渲染分支 homeLayout==='tasks' → TasksFirstHome；SpaceSwitch 旁入口钮。
- `components/settings/AppearancePanel.tsx`：+HomeLayoutPanel；SettingsPage 挂。
- 三语 locales +键。
- 测试：TaskInboxPanel 既有 9 用例零改动绿（提取不破坏）；TasksFirstHome
  （收件箱大区渲染/能力 chips 全量/切回回调/空态）；appStore homeLayout
  读写与非法值回退；ModuleLauncher tasks 分支渲染与 classic 零变化。

## 4. 出口对照（阶段七判据）

- [ ] 收件箱+最近任务成为首页首屏（tasks 形态下）；
- [ ] 既有 14 板块全部入口无损（能力 chips 全量可达——藏≠删）；
- [ ] 回退开关（设置页+首页快捷，可切回经典首页）；
- [ ] classic 形态零回归（既有 ModuleLauncher 测试零改动全绿）。

## 5. 观察池（本刀不做）

任务首页的「最近任务」独立区（收件箱 doing/done 档已覆盖）；能力 chips
分组/搜索；任务拖拽排序；晨报/脉息在 tasks 形态的取舍（V1 不带入，切回
经典可看）；默认形态翻转（用户试用后拍板）。

## 6. 门禁

go build（零 Go 改动仍跑）、tsc -b、eslint、vitest 全量、drift OK@707
（零新绑定）、ci.ps1、版本三处 4.332.0。
