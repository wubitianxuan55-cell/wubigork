# gaea 板块蓝图 · home 首页启动器（v4「星枢港 · 双舷驾驶舱」）

> 覆盖 MASTER（页面级优先）。实施参考 docs/2026-08-15-gaea3-ui-design-system.md §3/§4。
> v4（2026-09-03 重构，ui-ux-pro-max AI-Native UI 范式）：从 v3「五段纵排中庭」
> 升级为**双舷驾驶舱**——左舷主工作台一屏尽收，右舷状态侧栏聚合全部遥测。

## 现状（facts）
- 组件：`frontend/src/components/ModuleLauncher.tsx`（ml-dock 双舷网格）
- 样式：`frontend/src/components/module-launcher.css`（ml-* 前缀，全部令牌派生）
- 数据源：manifest 驱动——`deriveLauncherModules(getActiveBoards(), LAUNCHER_DESC)`；
  办公（gaea）为旗舰大卡，**编程/设置自 v4 起以瓦片编入矩阵尾部**（编程带
  「独立窗口」徽标，aria 用 `shell.launcher.progEntry`）
- 入口：MainLayout home 特判渲染；语音入口 VOICE_LAUNCH_FLAG（useChatVoice 依赖，勿删导出）
- 启动：`MainLayout` 每次启动默认落首页；`BootSplash` 全屏启动动画

## v4 布局骨架
```
.ml-dock（grid：minmax(0,1fr) + 300px，≥1181px 双舷）
├── .ml-main 左舷
│   ├── .ml-hero 紧凑 Hero（左对齐）：pill → 标题（27px）→ 副标题
│   │   ├── 对话气泡流（hasChat 时浮现）
│   │   ├── 中央命令条（orb / 打字 / 语音 / 发送 / ⌘K，撑满左舷 max 720px）
│   │   └── 语音状态行（待机/聆听/回复 + 错误 + 打断）
│   └── .ml-cap 能力矩阵 Bento（6 列：旗舰 span 4×2 + 瓦片 span 2）
│       └── 瓦片 = 板块×8 + 编程(独立窗口徽标) + 设置，manifest 驱动自动入格
└── .ml-side 右舷（v3-panel 玻璃容器，实底信息行）
    ├── 内核状态（活跃模型 / 已启用引擎 本地·云端 / CPU·MEM·GPU 三表 / ComfyUI 徽标）
    ├── 项目写作进度（conic-gradient 进度环 + 章节数·字数）
    ├── 最近会话（3 条）· 记忆脉搏 · 做梦晨报（仅 work 空间）
```

## 与 v3 的差异（零功能删除，纯重组降噪）
- v3「快捷 chips」→ 删：四个入口与 Bento 瓦片同目标，收敛进矩阵一级面
- v3「AI 状态细条 4 列」+「底部信息条」→ 合并进右舷四面板（系统状态并入内核遥测，
  ComfyUI 运行徽标化）
- v3「门廊（编程/设置横条）」→ 编程/设置瓦片化编入矩阵尾部
- Hero 从居中营销式改左对齐工作台式；标题 40→27px
- 响应式：≤1180 侧舷下落为底部双列网格 + 旗舰整行单行化（**内容 padding-left 78px
  改横向排布，避免标题与左上图标叠压**）；≤640 侧舷单列；≤480 全单列

## i18n
- 新增：home.kernel / home.sideAria / home.comfyRunning
- 更新文案：home.title / home.sub / home.capSub
- 删除（22 键）：4 个 home.suggest*（chips）、statResource 三件套 + sys* 四件套
  （遥测面板化）、openSettings/settingsDesc（设置瓦片化）、9 个 v3 遗留死键
  （courtyardPlaceholder/voiceOrb/voiceKernel/localModel/voiceStartTip/studyHint/
  gardenHint/homeTitle/homeSub）；三语同步，各 648 键无重复

## 落地清单
- [x] 双舷骨架 ml-dock（左 1fr + 右 300px，≤1180 单列降级）
- [x] 紧凑 Hero + 命令条五要素（orb 打字/语音/发送/⌘K）+ 气泡流保留
- [x] 能力矩阵：旗舰 4×2 + 板块/编程/设置全瓦片化（编程独立窗口徽标）
- [x] 右舷四面板 + 晨报（work 空间红线不变）；遥测三表 ≥85% 转 warning 色
- [x] 焦点环 / reduced-motion / ui-reduced-motion / gaea-raf-degraded 全降级
- [x] i18n 三语同步（键位增删见上）；judge 视觉验收双宽度通过

## 验收
- 1440×900：双舷一屏（滚动余量 ~72px），旗舰渐变/徽标/CTA 正常，右舷空态自然；
- 1100×800：旗舰整行横向排布不叠压，瓦片 2 列，侧舷下落双列；
- 键盘 Tab 焦点环可见；manifest 增减瓦片自动跟随；零硬编码色值（全令牌派生）。
