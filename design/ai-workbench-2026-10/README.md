# gaea AI 工作台 · 高保真原型（2026-10）

> 设计说明见 [docs/gaea-ai-workbench-ui-2026-10.md](../../docs/gaea-ai-workbench-ui-2026-10.md)。
> 本目录是**独立静态原型**，不接业务代码、不改生产前端；用于先对齐「办公是唯一工作台，CAD/进度只是工具」的信息架构。

## 打开

直接双击 `index.html`，或：

```powershell
start design\ai-workbench-2026-10\index.html
```

可用 URL 参数：

| 参数 | 值 | 说明 |
|---|---|---|
| `view` | `chat` / `trajectory` / `context` / `memory` / `desk` | 办公主视图 |
| `tool` | `cad` / `schedule` | 直接打开一个临时工具面板 |
| `theme` | `light` | 亮色（默认暗色） |

例：`index.html?tool=cad`、`index.html?view=desk&theme=light`。

## 可交互

- 办公页签：对话 / 轨迹 / 上下文 / 记忆；CAD 与进度**不在页签里**。
- 对话里的 CAD 工具卡「展开查看」→ 工具面板浮出；关闭/Esc/遮罩回到对话。
- 输入框「工具」菜单：CAD 看图 / 进度计划 / 文档 / 表格；文档与表格沿用现有办公工具。
- 左侧「进度纠错」工作对象、产物行、桌面工具 tile 都打开同一个工具面板。
- 工具打开时右栏检查器切到该工具（图纸图层/图框；计划任务/体检）；关闭后回对话检查器。
- 图纸检查器：图层勾选会真实隐藏 SVG 图层。
- 审批 dock「允许 / 暂不」有完成态；顶栏「待你确认」可定位到它。
- 明暗切换；页签键盘 ←/→ 漫游；Esc 关闭工具。

## 这个原型要证明的三件事

1. **办公是唯一工作台**：CAD 与进度没有导航项、没有一级 Tab、没有独立页面；只在工具菜单、AI 工具卡、产物行里出现。
2. **工具是临时面板**：打开时浮在对话上方，带「工具」标识与关闭；关闭不删产物、不重置会话、不重跑解析。
3. **产物与依据可回看**：工具结果进产物；回答里的引用 chips 可点回图纸/计划；检查器给出图层、图框、任务、偏差依据。

## 截图

| 状态 | 文件 |
|---|---|
| 对话（暗） | `screens-dark-chat.png` |
| CAD 工具面板（暗） | `screens-dark-cad.png` |
| 进度工具面板（暗） | `screens-dark-schedule.png` |
| 工作台空态（暗） | `screens-dark-desk.png` |
| 对话（亮） | `screens-light-chat.png` |
| CAD 工具面板（亮） | `screens-light-cad.png` |

## 边界

- 原型里的图纸是 SVG 示意，不是真实 DWG 渲染；进度数据是手工样例。
- 原型不调用 `window.go` / 后端绑定，不涉及持久化。
- 不替代生产实现；落地时按设计说明 §7 的映射逐项接线。

## 文件

```
index.html   结构 + 内联 SVG 图标/图纸
styles.css   令牌、布局、组件样式（暗/亮）
app.js       视图/工具面板/检查器/图层/明暗等交互
```
