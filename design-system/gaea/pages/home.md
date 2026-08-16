# gaea 3.0 板块蓝图 · home 首页启动器

> 覆盖 MASTER（页面级优先）。实施参考 docs/2026-08-15-gaea3-ui-design-system.md §3/§4。

## 现状（facts）
- 组件：`frontend/src/components/ModuleLauncher.tsx`（DeepSeek 风格 Hero：公告 pill + 大标题 + 双行动卡 / 右侧深色渐变 AI 视觉卡）
- 样式：`frontend/src/components/module-launcher.css`（ml-* 前缀，主题令牌派生渐变 + 细网格纹理 + 大圆角）
- 3.0 已清单化：卡片列表 = `deriveLauncherModules(getActiveBoards(), LAUNCHER_DESC)`（manifest 驱动，含 knowledge）
- 入口：MainLayout home 特判渲染；语音入口 VOICE_LAUNCH_FLAG

## 目标态
- **视觉性格：中枢 / 仪式感 / 科技首页**——3.0 的第一屏，参照 DeepSeek 官网：浅色/深色主题下均为「左文右卡」Hero。
- Hero 左侧：公告 pill（GAEA 3.0 已就绪 · 本地 AI 创作中枢，绿色脉冲点 + 箭头）+ 大标题「从灵光乍现，到星河成篇」+ 双行动卡（开始创作 → novel / 和 gaea 对话 → chat）。
- Hero 右侧：深色渐变视觉卡（`--color-primary` 派生渐变 + 细网格纹理 + 星云流光），承载语音晶核 orb、AI 状态、模型 pill 与语音按钮。
- AI 状态细条：活跃模型 / 已启用引擎 / 资源占用 / 写作进度（真实遥测，4 列）。
- 模块卡片（manifest 驱动，8+1 卡）：
  - 玻璃卡 `md-glass` + 主题 aurora 水印（`--gaea-aurora-bg` 低透明度叠加）；
  - hover：`--elevation-2` + 位移 2px + 箭头右移（`--transition-fast`，reduced-motion 跳过）；
  - 图标走 `resolveBoardIcon`（antd icons，禁 emoji）；
  - focus 环 `--focus-ring`（键盘可达）。
  - **featured 大卡 = 办公**（`m.key === 'gaea'`，2×2 Bento）；编程（code）卡为 DeepSeek Harness
    Web 进程管理入口（ProgrammingPage），manifest 驱动自动入网格。
- 语音晶核：收进 Hero 右侧视觉卡（orb + 雷达脉冲 + 声谱）；AI 回复中发光脉冲用 `--gaea-glow`（呼吸 2s）。
- 底部信息条：最近会话 / 记忆脉搏 / 系统状态，统一小字 `--color-text-secondary`。
- 模块区 `.ml-section` 不压缩不拉伸（`flex: 0 0 auto`），网格按内容高度排布、整页滚动，
  避免卡片溢出到下方信息条被遮挡。

## 落地清单
- [x] Hero 左文右卡 + 公告 pill + 双行动卡（参照 DeepSeek 首页风格）
- [x] 右侧视觉卡：主题令牌派生渐变 + 细网格纹理 + 星云流光，零硬编码色值
- [x] AI 状态细条（4 列真实遥测）与模块网格分区标题
- [x] 办公设为 featured 大卡；编程板块（Harness Web 进程管理）入网格
- [x] 修复网格溢出压住底部信息条（设置卡被遮挡问题）
- [x] 卡片 hover 位移统一 ≤2px + focus 环；语音交互态发光统一 `--gaea-glow`

## 验收
- 键盘 Tab 遍历卡片可见焦点环；reduced-motion 下无位移动画；
- 12 主题下 Hero 渐变/文字对比度/发光均正常；manifest 加载后 knowledge 卡出现。
