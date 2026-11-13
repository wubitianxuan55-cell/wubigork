# gaea 板块蓝图 · 编队

> 2026-11-13 对齐 `dsh-context v0.84.1` 的 `MissionPulse`。

## 信息架构

- 顶部 Hero：`AGENT 编队 / 时间`、标题、说明，以及 Agent、运行中、工作项、消息四个 KPI。
- 主区双列：左侧 Agent 网络卡片拓扑，右侧上下叠放跨 Agent 协作与协作任务。
- 底部事件检查：左事件流、右详情面板；未选择事件时保留明确空态。
- 成员费用/消息卡仅在有子 Agent 时显示在页面后部。

## Agent 网络

- 使用卡片节点和层级连线，不使用径向圆环。
- 主 Agent 为根卡；子 Agent 按层级向下展开，贝塞尔式连接线表达委派拓扑。
- 卡片包含角色、状态、任务名、Token、工具调用、费用/消息（有数据时）与上下文占用条。
- 子 Agent 卡片可键盘聚焦并打开对应上下文。

## 响应式与可访问性

- `>=980px` 主区双列；更窄为单列，事件检查上下堆叠。
- `<=480px` 收紧 Hero 与卡片间距。
- 状态同时以文字、颜色与图标表达；所有按钮保留可见焦点环；减少动效时关闭脉冲与过渡。

## 实现

- `frontend/src/gaea/components/context/Fleet.tsx`
- `frontend/src/gaea/components/context/AgentTopology.tsx`
- `frontend/src/gaea/components/context/fleet-view.css`
- `frontend/src/gaea/components/context/agent-topology.css`
