# gaea 3.0 板块蓝图 · home 首页启动器

> 覆盖 MASTER（页面级优先）。实施参考 docs/2026-08-15-gaea3-ui-design-system.md §3/§4。

## 现状（facts）
- 组件：`frontend/src/components/ModuleLauncher.tsx`（玻璃 HUD：语音晶核 orb + 左右卡片列 + 遥测读数）
- 样式：`frontend/src/components/module-launcher.css`（ml-* 前缀，aurora 渐变 + 玻璃 + 霓虹角标）
- 3.0 已清单化：卡片列表 = `deriveLauncherModules(getActiveBoards(), LAUNCHER_DESC)`（manifest 驱动，含 knowledge）
- 入口：MainLayout home 特判渲染；语音入口 VOICE_LAUNCH_FLAG

## 目标态
- **视觉性格：中枢 / 仪式感**——3.0 的第一屏，强调「AI 在线、本地优先、可语音」。
- 顶部状态栏：品牌 + GAEA CORE 芯片 + 绑定模型名 + 聊天入口（保留现状，统一玻璃规范）。
- 模块卡片（manifest 驱动，8+1 卡）：
  - 玻璃卡 `md-glass` + 主题 aurora 水印（`--gaea-aurora-bg` 低透明度叠加）；
  - hover：`--elevation-2` + 位移 2px + 箭头右移（`--transition-fast`，reduced-motion 跳过）；
  - 图标走 `resolveBoardIcon`（antd icons，禁 emoji）；
  - focus 环 `--focus-ring`（键盘可达）。
- 语音晶核：保留 orb + 雷达脉冲 + 声谱；AI 回复中发光脉冲用 `--gaea-glow`（呼吸 2s）。
- 遥测 HUD：保留 CORE/LINK/VAD/VOL 读数，统一小字 `--color-text-secondary`。

## 落地清单
- [ ] module-launcher.css：卡片 hover 位移统一 ≤2px + `--focus-ring`；补 `cursor-pointer`
- [ ] 状态色走令牌（检查 ml-* 硬编码色值）
- [ ] 语音交互态（listening/speaking）发光统一 `--gaea-glow`

## 验收
- 键盘 Tab 遍历卡片可见焦点环；reduced-motion 下无位移动画；
- 12 主题下背景/卡片/发光均正常；manifest 加载后 knowledge 卡出现。
