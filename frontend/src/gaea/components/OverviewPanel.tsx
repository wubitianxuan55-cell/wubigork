// 主区「概览」tab（v4.23 A4 迁移：右栏「分析组·统计」面板迁入 ChatTabs，
// 与轨迹/上下文同级）。本组件是 StatsPanel 的薄容器：
// - 不改 StatsPanel 本体，需要适配逻辑写在这里；
// - 数据持久化仍由 App 层的 useStatsPersistence 负责，本组件只接收 props 透传；
// - 容器滚动方式对齐主区看板（TrajectoryView/ContextView 同级页面）：
//   外层只给满高（.main 为 overflow hidden 的 flex 列），滚动统一收敛在
//   StatsPanel 自身的 overflow-y-auto 内，避免双滚动条。
import { StatsPanel, type StoredData } from "./StatsPanel";
import type { SessionStatsView, WireUsage } from "../lib/types";

export interface OverviewPanelProps {
  /** 会话累计统计（localStorage 持久化；来自 useStatsPersistence().data） */
  data: StoredData;
  /** 清空统计（来自 useStatsPersistence().clearData） */
  clearData: () => void;
  /** 会话级派生统计（后端事件日志重放；恢复会话后回填，available=false 时不展示） */
  sessionStats?: SessionStatsView;
  /** 本轮执行模型用量（store 累加器） */
  perTurnExecutorUsage?: WireUsage;
  /** 本轮子代理用量（store 累加器） */
  perTurnSubUsage?: WireUsage;
  /** 本轮逐步用量（App 原样透传，与 useStatsPersistence 同源） */
  turnSteps?: WireUsage[];
  /** 子代理模型标签（命中率趋势标题用） */
  subagentModel?: string;
  /** 工具调用计数（useToolStats） */
  toolCounts: Record<string, number>;
  /** 技能调用计数（useToolStats） */
  skillCounts: Record<string, number>;
}

export function OverviewPanel(props: OverviewPanelProps) {
  return (
    <div className="h-full min-h-0">
      <StatsPanel
        data={props.data}
        clearData={props.clearData}
        sessionStats={props.sessionStats}
        perTurnExecutorUsage={props.perTurnExecutorUsage}
        perTurnSubUsage={props.perTurnSubUsage}
        turnSteps={props.turnSteps}
        subagentModel={props.subagentModel}
        toolCounts={props.toolCounts}
        skillCounts={props.skillCounts}
      />
    </div>
  );
}
