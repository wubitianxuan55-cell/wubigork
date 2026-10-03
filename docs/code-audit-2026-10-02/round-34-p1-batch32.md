# 全仓审计第 32 批 · FE6-07 ModuleLauncher 拆分（展示件+轮询 hook 出仓）· 2026-10-03

> 接续 [round-33（批次三十一）](round-33-p1-batch31.md)。前端配方第三刀：ModuleLauncher.tsx 1058 行的纯展示子组件与轮询 hook 整段出仓，主文件留双空间首页主体与两套变体。主代理直做，**不抬版本**。
> 快照：开工时 HEAD = `221b73ee`（批次三十一）；工作树干净。

---

## 一、预核与切口

- FE6-07 复核（1058 行）：文件天然三层——①遥测/会话/记忆最小类型+fmt 助手+九个纯展示子组件（SectionLabel/KernelRow/Meter/ChatBubble/WritingRing/MemoryPulse/TaskInboxEntry/TelemetryBody/SessionList，全 React.FC 只吃 props）+useLauncherData 轮询 hook（:70-443，自足段）；②SpaceSwitch/HomeLayoutToggle/主组件+两套变体（:445+，设计主体）。
- 两轮依赖普查定双向接口：出仓段用 React hooks/app/useT+Translator/getModelMonitor/recentFiles/usePollingGate/useAppStore/AtEntry/ShellSpace+三枚图标；主段反向消费 9 符号（SectionLabel/ChatBubble/WritingRing/MemoryPulse/TaskInboxEntry/TelemetryBody/SessionList/useLauncherData/LauncherData）。
- 外部消费通道核查：TasksFirstHome 导入 `CardHead/SpaceSwitch/SessionList/LauncherData`、useChatVoice 导入 `VOICE_LAUNCH_FLAG`、MainLayout 导入默认+`LauncherTarget`——SessionList/LauncherData 出仓后经主文件 re-export，**外部导入路径零改动**（CardHead/SpaceSwitch 留主文件不受影响）。

## 二、落地

- `launcher_parts.tsx`（391）：:70-443 整段搬移；尾部集中导出 `export { SectionLabel, ChatBubble, WritingRing, MemoryPulse, TaskInboxEntry, TelemetryBody, useLauncherData }`（SessionList/LauncherData 原文已带 export 关键字，不重复导出——首轮重复声明被 tsc 抓，删尾项即平）。
- `ModuleLauncher.tsx`（684）：主文件瘦身为变体+主组件；import 重算（react 去 useEffect/useMemo、去掉 recentFiles/usePollingGate/engines/AtEntry 行）+ 回接导入 + re-export 两条。
- 行数对账闭合：374（搬移段）+ 旧头 69 + 主段 614 + 导入/导出缝合 = 1058+新头。

## 三、源级守卫随迁（本批关键动作）

- **perf-guards.test.ts 源级断言**（「ModuleLauncher 首页账页用 useSyncExternalStore + subscribeRecentFiles」）在首轮全量 ci 拦下 1 红——订阅用法已随段出仓。**守卫意图不变、读源范围扩为主文件+展示件两份并集**（头注写明拆分缘由），复跑 17 例绿。教训：源级断言守卫必须与其守卫的代码同批随迁（同 App.tsx 源级锁家族）。
- 门禁：tsc 绿 / ModuleLauncher+TasksFirstHome 21 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例全绿）/ golangci 0 issues；提交后树干净+HEAD 可编译。

## 四、余量与下一批

- FE6-07 部分收敛（1058→684+391）：「两套变体」渲染分叉仍在主文件——变体级收敛属设计型（同 useController hook 分域），与机械搬移分界清晰。
- 下一批候选：FE3-11 controller.ts（963 行，useController 280 行单 hook 36 回调——预核判定=hook 分域设计型非机械，需先立「hook 分域」配方：自定义 hook 抽取+依赖数组保全）；coupling ~20 与零散死码等拍板。
