import type { ReactNode } from "react";

// MergedPanel — 右栏合并面板壳（v4.53 化繁为简：产物与变更、任务与分工归并）。
//
// 用户拍板：合并 = 两块内容**直接并成一个面板**（上下分区同屏全可见），
// 不是二级标签/段切换——零额外点击。主区（flex-1，自带滚动）承载第一块，
// 次区（border-t 分隔，高度随内容、封顶 45%，超出内部滚动）承载第二块；
// 两块各自保留 v3-panel-head 标题行与计数/动作，壳只负责分区不抢语义。
//
// 面板组件本体（DeliverablesPanel/ChangesPanel/TaskCenter/SubagentsPanel）
// 不感知本壳（学既有约定：框架与 tab 内容解耦）。

export function MergedPanel({
  primary,
  secondary,
}: {
  /** 主区（上半，flex-1 占大头、内部滚动）。 */
  primary: ReactNode;
  /** 次区（下半，分隔线之下，高度随内容封顶 45%、内部滚动）。 */
  secondary: ReactNode;
}) {
  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="flex-1 min-h-0">{primary}</div>
      <div className="shrink-0 min-h-[88px] max-h-[45%] border-t border-border-soft overflow-y-auto">
        {secondary}
      </div>
    </div>
  );
}
