import { useEffect, useMemo, useState } from "react";
import { subscribeSubagentRuns } from "../lib/subagentRunsStore";
import { usePaneTabsStore } from "../lib/paneTabs";
import { SubagentThread, type SubagentThreadStatus } from "./SubagentThread";
import type { SubagentRunView } from "../lib/types";

// SubagentPaneChat — 右栏「子会话」pane（2026-11-18，dsh ui-subagent S6 右侧
// Sidebar 聊天页形态）：SubagentThread 完整会话在右栏工作台的宿主壳。
//
// 数据：runs 快照（共享单轮询）供状态/标题/模型/历时/谱系链——SubagentThread
// 本体的 3s transcript 轮询照常（pane 只是宿主，不重复拉 transcript）。
// 谱系祖先点击=右栏内跳转（openSubagentPane 每 ref 单实例）；「返回」=关闭
// 本 pane tab。tab 由 SubagentCatalogButton 目录树的「在侧边栏打开」动态开
// （openSubagentPane，id=sub:<ref>）。

export function SubagentPaneChat({
  sessionPath,
  target,
  fallbackTitle,
  onClose,
}: {
  sessionPath: string;
  target: string;
  /** tab 条标题兜底（打开瞬间的任务名/ref；快照到齐后被 run.task 覆盖）。 */
  fallbackTitle: string;
  onClose: () => void;
}) {
  const [runs, setRuns] = useState<SubagentRunView[]>([]);
  useEffect(() => {
    if (!sessionPath) return;
    return subscribeSubagentRuns(sessionPath, setRuns);
  }, [sessionPath]);

  const run = useMemo(() => runs.find((r) => r.ref === target), [runs, target]);
  const openSubagentPane = usePaneTabsStore((s) => s.openSubagentPane);

  // 谱系链（root→…→父）：与 SubagentThread 面包屑同源口径，按 parentRef 上溯。
  const lineage = useMemo(() => {
    const byRef = new Map(runs.map((r) => [r.ref, r]));
    const chain: { ref: string; task: string }[] = [];
    let cur = run?.parentRef;
    let guard = 0;
    while (cur && guard < 16) {
      const r = byRef.get(cur);
      if (!r) break;
      chain.unshift({ ref: r.ref, task: r.task || r.ref });
      cur = r.parentRef;
      guard++;
    }
    return chain;
  }, [run, runs]);

  return (
    <div className="h-full min-h-0" data-testid="subagent-pane-chat">
      <SubagentThread
        sessionPath={sessionPath}
        target={target}
        task={run?.task || fallbackTitle}
        status={(run?.status ?? "completed") as SubagentThreadStatus}
        model={run?.model}
        createdAt={run?.createdAt}
        updatedAt={run?.updatedAt}
        lineage={lineage}
        onOpenRef={(ref) => {
          const r = runs.find((x) => x.ref === ref);
          openSubagentPane(ref, r?.task || ref);
        }}
        onBack={onClose}
      />
    </div>
  );
}
