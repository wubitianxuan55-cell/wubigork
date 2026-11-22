# gaea 外观系统 v4：Bloom Surface

> 2026-11-22 生效。覆盖旧 Constellation OS 中与配色、材质、背景相关的规则。
> 上游基准：`dsh-bloom-theme` 0.29.0，commit `bd3e93f`。顶栏外观面板采用
> 0.29 的 15 色、六列色块网格、单字国风印记与悬停/聚焦公用诗句字幕。

## 设计命题

gaea 是长时间运行的本地 AI 工作台。外观必须让正文、证据和操作持续可读，
主题色只负责身份、定位和状态，不承担内容层级。

默认主题为「黛蓝 / mist」。设置页与顶栏外观面板都提供 Bloom 全量 15 套配色；
自动换色也从同一 15 套中轮换。

## 四层分离

### 1. 原始色板

- 文件：`frontend/src/theme/bloomPalette.ts`
- 内容：15 套 OKLCH 配色，每套含可读强调轨、莫兰迪气质轨、底色、表面阶梯、
  文字与三色光谱。
- 规则：主题色字面量只允许存在于这一层。

### 2. 语义令牌

- 文件：`frontend/src/theme/tokens.ts`
- CSS 桥：`frontend/src/gaea/appearance/semantic.css`
- 角色：surface、ink、line、accent、state、chart、shadow、focus。
- 规则：组件不得直接读取 palette 名称或原始色值，只消费语义令牌。
- 强调色保留上游双轨：`accentL/accentD` 是调色输入，界面可读轨为
  `readableAccent`。亮色必须使用 `accentL 88% + black 12%` 的 OKLCH 混色，
  暗色直接使用 `accentD`；可读轨用于文字、按钮、焦点与选中态，原始 accent
  只用于表面染色和色块样本。
- 表面色阶固定为五档：画布 `surface`、低层 `surfaceContainerLow`、
  面板 `surfaceContainer`、高层 `surfaceContainerHigh`、最高层
  `surfaceContainerHighest`。五档都由当前配色生成，并保证明暗方向与正文对比度。
- 面板另有三层同色族层级：`surfaceTint1/2/3` 分别映射 Bloom 的
  `bg / layer-2 / layer-3`。大面积底色只由中性表面阶梯与目标的 accent
  彩度归一混入生成；莫兰迪气质轨只服务背景、光晕与细线。
- 所有 OKLCH 混色在 JS 中按 lightness、chroma、最短 hue 弧计算，再落为
  Ant Design 可消费的 hex；不得用 sRGB 通道插值冒充 `color-mix(in oklch)`。
- 文案与边框采用上游阶梯：亮色文案透明度依次为 28/32/35/37%，暗色为
  30/35/40/43%；亮色边框由正文色以 90/80/70/60% 透明混入，暗色边框由
  莫兰迪气质轨以 10/18/28/40% 不透明度生成。
- 状态色使用上游 `sharedDswTokens`：success/warning 的 tertiary 以
  `color-mix(in oklch, state, bg 78%)`（暗）或 `88%`（亮）生成，warning
  label 与 danger hover 背景也来自同一语义层，不在组件内临时调色。

### 3. 材质与氛围

- 文件：`frontend/src/gaea/appearance/materials.css`
- 表面材质：毛玻璃、磨砂玻璃、液态玻璃；实底仅用于系统无障碍降级。
- 背景氛围按亮暗分别保存：浅色 `斜光流转 / 水彩晕染 / 清透留白 / 不显示`，
  深色 `角落微光 / 对角漫染 / 漫天星辰 / 不显示`。
- 规则：材质负责 fill、filter、edge 和 shadow；背景只使用主题色低透明度派生。
  正文工作区不得叠加重模糊；系统降低透明度或高对比时自动退实底。
- 壳层材质契约：`--gaea-shell-*` 统一描述画布、导航轨、侧栏、栏条、对话画布、
  输入岛、控件与栏条填充，以及 sheen、edge、shadow、filter。组件只消费契约，
  `veil / soft / solid` 的差异只允许定义在 materials 层。
- 毛玻璃：58% 表面填充 + `blur(24px) saturate(1.15)`。
- 磨砂玻璃：48% 表面填充 + `blur(12px) saturate(1.02)` + 黑白细颗粒。
- 液态玻璃：25% 表面填充 + `blur(3px) url("#gaea-liquid-lens") saturate(1.2)`；
  边缘折射由 SVG displacement lens 完成。
- 浅色雾光复用上游 `stream / wash / clear` 三张 SVG fog mask；深色使用四角
  radial ambience；`corner / diagonal / galaxy` 复用上游确定性星尘序列，
  分别显示右上角、左下角与全屏星系层，`galaxy` 另含 60 颗流星候选。
- 材质档位：毛玻璃、磨砂玻璃、液态玻璃、实底。一键玻璃态按钮或 `Alt+G`
  在液态玻璃与上次非液态材质之间切换，并持久化。

### 4. 组件与壳层

- 文件：`frontend/src/gaea/appearance/components.css`
- 范围：导航轨道、状态条、遥测条、侧栏、对话画布、输入岛、工作区、浮层、焦点。
- 规则：组件映射只引用第 2、3 层变量。除语法高亮、图表分类和明确标注的 mock
  数据外，UI 文件不得保留硬编码颜色。
- `v3-rail` 与 `v3-telemetry` 属于全局壳层，位于路由级 `.gaea-app-layout` 之外；
  其映射不得加路由容器前缀，否则主题切换只更新根变量而不会命中真实 DOM。
- 组件层不得出现 `data-gaea-surface="soft"` 一类模式特例，模式差异必须回到
  materials 层解决。
- 设置页、模型中心、办公与文学共享同一表面阶梯；面板差异来自
  `surfaceContainerLow → High → Highest` 与材质填充，不再按卡序号主观轮换色相。
- 对话主画布保持透明并透出 body 氛围；顶栏、侧栏、输入岛与右侧工作区才承载
  材质表面。主内容 `Content` 也必须透明；旧 `.cyber-grid / star-dots /
  space-dust` 常驻层已删除，避免盖住 body 的四角氛围。
- Bloom 星尘保留上游 `z-index: -1`；`.gaea-app-layout` 必须建立独立的
  `isolation: isolate` stacking context，让星河位于壳层背景之上、业务内容
  之下。缺少该上下文时，全屏 galaxy 粒子会被整块布局背景吞掉。

## 配色全集

| 色族 | 15 套 |
|---|---|
| 红 | 朱砂、粉霞 |
| 橙 | 橙黄、落霞、赭石、琥珀 |
| 黄 | 轻黄 |
| 绿 | 草色、竹青 |
| 青 | 天青 |
| 蓝 | 黛蓝、青金 |
| 紫 | 青莲、银红 |
| 粉 | 桃夭 |

15 套全部进入菜单与自动换色序列。顶栏外观面板采用 Bloom 0.29 的
312px 窄面板：6 列色块网格、国风单字、色名/诗句共用字幕、三图标模式切换
与设置入口；材质、亮暗背景、自动换色和动效细节只在设置页调整。

0.29 上游的“彩度归一”规则只用于表面令牌：低彩度主题获得更高 accent
混入比，高彩度主题自动降低混入比，使大面积的侧栏与面板保持相近感知强度。

## 交互与可访问性

- 15 套 × 明暗两档均通过正文、次级文字、三级文字与状态色对比度机检。
- 五档表面也纳入对比度机检；状态主色按上游契约在 base 背景达到 AA，
  高层状态表面使用 tertiary 档，不再把主状态色自动漂白或私自提亮。
- 主按钮前景根据强调色亮度自动选择。
- 焦点环统一使用 accent。
- `prefers-reduced-transparency`、`prefers-contrast`、`forced-colors` 自动退实底。
- 动效只表达状态变化，不使用常驻霓虹呼吸。

## 防回流

`frontend/src/gaea/appearance/architecture.test.ts` 检查：

- 四层 CSS 加载顺序正确。
- UI 壳层与通用组件不出现 hex/rgb/hsl 颜色字面量。
- 原始色值仅存在于色板、语法数据色和 mock 数据。
