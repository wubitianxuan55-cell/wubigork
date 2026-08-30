# gaea 3.0 板块蓝图 · home 首页启动器（v3「AI 多功能平台 · 星枢指挥所」）

> 覆盖 MASTER（页面级优先）。实施参考 docs/2026-08-15-gaea3-ui-design-system.md §3/§4。
> v3（本轮重构）：从「双翼·中庭」升级为**未来感 AI 多功能平台首页**——对标
> Poe / Perplexity / DeepSeek 官网的「一句话直达」范式 + gaea 3.0 星枢驾驶舱令牌体系。

## 现状（facts）
- 组件：`frontend/src/components/ModuleLauncher.tsx`（Hero 中轴 + 能力矩阵 Bento）
- 样式：`frontend/src/components/module-launcher.css`（ml-* 前缀，全部令牌派生）
- 数据源：manifest 驱动——`deriveLauncherModules(getActiveBoards(), LAUNCHER_DESC)`；
  办公（gaea）为旗舰大卡，code/settings 入门廊，其余板块入 Bento 瓦片
- 入口：MainLayout home 特判渲染；语音入口 VOICE_LAUNCH_FLAG（useChatVoice 依赖，勿删导出）
- 启动：`MainLayout` 每次启动默认落首页（不再恢复上次页面）；`BootSplash` 全屏启动动画
  （index.html 静态启动屏 → React BootSplash 接管，reduced-motion / rAF 降级）

## 目标态（v3 视觉性格：中枢 / 仪式感 / 科技首页）
- **Hero 中轴**（居中）：
  - 公告 pill（`home.pill`，绿色脉冲点 + 箭头）→ 巨幅标题（`home.title`「从灵光乍现，到星河成篇」）
    → 副标题（`home.sub`）→ **中央命令条** → 语音状态行 → 快捷 chips（novel/imagegen/chat/code）。
  - 命令条 = AI 内核 orb（语音状态可视化：待机呼吸 / 聆听波纹 / 回复辉光）+ 打字输入
    （复用 VoiceChatText 管道）+ 语音按钮（本页直启麦克风）+ 发送 + ⌘K 提示（全局搜索）。
- **AI 状态细条**：活跃模型 / 已启用引擎 / 资源占用 / 项目写作进度（真实遥测，4 列，透明无边框）。
- **能力矩阵（Bento）**：
  - 办公（`m.key === 'gaea'`）为 **4×2 旗舰大卡**（badge「旗舰工作台」+ 细网格纹理 + aurora 水印
    + 底部 CTA 按钮）；其余板块 **2 列等宽瓦片**（icon 走 resolveBoardIcon，antd icons，禁 emoji）。
  - 瓦片 hover：`--v3-glow-soft` + 位移 2px + 箭头右移（`--transition-fast`，reduced-motion 跳过）；
    focus-visible 统一 2px 主色焦点环（与 v3 壳层约定一致）。
- **门廊**：编程（独立窗口，`independent` 徽标）+ 设置；**底部信息条**：最近会话 / 记忆脉搏 / 系统状态。
- **Bento 断点**：≥1181px 6 列（旗舰 span 4×2）；≤1180 4 列（旗舰整行 span 4×1）；
  ≤640 2 列（旗舰 span 2）；≤480 1 列。旗舰卡选择器用 `.ml-bento.ml-bento--featured`
  双类提权（0,2,0），防止与 `.ml-bento` 同特异性源码顺序覆盖。

## 落地清单
- [x] Hero 中轴：公告 pill + 巨幅标题 + 副标题 + 中央命令条（orb/打字/语音/发送/⌘K）+ 快捷 chips
- [x] 能力矩阵 Bento：办公 4×2 旗舰大卡 + 板块瓦片，manifest 驱动自动入格
- [x] AI 状态细条（4 列真实遥测）与底部信息条（会话/记忆/系统）保留真实联动
- [x] 语音晶核收进命令条 orb（呼吸/波纹/辉光三态）；AI 回复发光统一 `--gaea-glow`
- [x] 全部交互元素键盘焦点环（bento/chips/voice/send/util）；hover 位移 ≤2px
- [x] BootSplash 启动动画（两段式 + timer 驱动 + reduced-motion/rAF 降级）与默认首页落地
- [x] i18n 三语同步新增 home.* 与 boot.* 键（en/zh/zh-TW，DictKey 编译期锁死）

## 验收
- 12 主题（6 色系 × 明暗）下 Hero 渐变/文字对比度/发光正常（浅色已抽查）；
- 键盘 Tab 遍历可见焦点环；reduced-motion 下无位移动画；gaea-raf-degraded 下 v3-rise 恢复可见；
- manifest 加载后瓦片自动增减（knowledge 等后端板块入格）；旗舰卡断点跨度正确（6/4/2/1 列）。
