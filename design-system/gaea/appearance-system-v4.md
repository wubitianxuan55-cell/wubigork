# gaea 外观系统 v4：Bloom Surface

> 2026-11-22 生效。覆盖旧 Constellation OS 中与配色、材质、背景相关的规则。
> 上游参考：`dsh-bloom-theme` 0.28.0，commit `44f5013`。

## 设计命题

gaea 是长时间运行的本地 AI 工作台。外观必须让正文、证据和操作持续可读，
主题色只负责身份、定位和状态，不承担内容层级。

默认主题为「天青」。设置页提供 Bloom 全量 15 套配色，顶栏快捷入口提供 12 套精选。

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

### 3. 材质与氛围

- 文件：`frontend/src/gaea/appearance/materials.css`
- 表面材质：轻雾、柔雾、实底。
- 背景氛围：雾光、淡彩、网格、纯色。网格保留旧工作台结构感，但改用语义色低对比发丝线。
- 规则：材质负责 fill、filter、edge 和 shadow；背景只使用主题色低透明度派生。
  正文工作区不得叠加重模糊；系统降低透明度或高对比时自动退实底。
- 壳层材质契约：`--gaea-shell-*` 统一描述画布、导航轨、侧栏、栏条、对话画布、
  输入岛、控件与栏条填充，以及 sheen、edge、shadow、filter。组件只消费契约，
  `veil / soft / solid` 的差异只允许定义在 materials 层。
- 一键玻璃态：顶栏按钮或 `Alt+G` 在柔雾与上次轻雾/实底之间切换，并持久化。
  柔雾同时改变全局环境光、画布透明渐变、大面积 blur/saturate、内侧高光与分层阴影。

### 4. 组件与壳层

- 文件：`frontend/src/gaea/appearance/components.css`
- 范围：导航轨道、状态条、遥测条、侧栏、对话画布、输入岛、工作区、浮层、焦点。
- 规则：组件映射只引用第 2、3 层变量。除语法高亮、图表分类和明确标注的 mock
  数据外，UI 文件不得保留硬编码颜色。
- `v3-rail` 与 `v3-telemetry` 属于全局壳层，位于路由级 `.gaea-app-layout` 之外；
  其映射不得加路由容器前缀，否则主题切换只更新根变量而不会命中真实 DOM。
- 组件层不得出现 `data-gaea-surface="soft"` 一类模式特例，模式差异必须回到
  materials 层解决。

## 配色全集

| 色族 | 全量 15 套 | 12 套精选 |
|---|---|---|
| 红 | 朱砂、粉霞 | 朱砂、粉霞 |
| 橙 | 橙黄、落霞、赭石、琥珀 | 橙黄、落霞 |
| 黄 | 轻黄 | 轻黄 |
| 绿 | 草色、竹青 | 草色 |
| 青 | 天青 | 天青 |
| 蓝 | 黛蓝、青金 | 黛蓝、青金 |
| 紫 | 青莲、银红 | 青莲、银红 |
| 粉 | 桃夭 | 桃夭 |

赭石、琥珀、竹青仍可在设置页选择。顶栏快捷切换显示 12 套精选；如果当前使用
非精选色，顶栏临时把当前色加入快捷列表，避免当前选择不可见。

## 交互与可访问性

- 15 套 × 明暗两档均通过 WCAG AA 正文对比度机检。
- 主按钮前景根据强调色亮度自动选择。
- 焦点环统一使用 accent。
- `prefers-reduced-transparency`、`prefers-contrast`、`forced-colors` 自动退实底。
- 动效只表达状态变化，不使用常驻霓虹呼吸。

## 防回流

`frontend/src/gaea/appearance/architecture.test.ts` 检查：

- 四层 CSS 加载顺序正确。
- UI 壳层与通用组件不出现 hex/rgb/hsl 颜色字面量。
- 原始色值仅存在于色板、语法数据色和 mock 数据。
