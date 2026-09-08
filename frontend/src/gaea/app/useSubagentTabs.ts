// useSubagentTabs — 独立子代理会话 tabs + task 卡歧义选择器候选的快照与实时同步。
// 从 App.tsx 按原执行顺序迁出：会话切换复位、打开/关闭 tab、subscribeSubagentRuns
// 单轮询同步、以及任务卡多候选 taskPickCandidates（App 层的点击瞬间候选快照）。
import { useCallback, useEffect, useRef, useState } from "react";
import type { ChatTabId } from "../components/ChatTabs";
import type { SubagentThreadStatus } from "../components/SubagentThread";
import { subscribeSubagentRuns } from "../lib/subagentRunsStore";
import type { SubagentRunView } from "../lib/types";

interface UseSubagentTabsParams {
  currentSessionKey: string;
  currentSessionPath: string | undefined;
  setChatTab: (id: ChatTabId) => void;
}

export function useSubagentTabs({ currentSessionKey, currentSessionPath, setChatTab }: UseSubagentTabsParams) {
  // 独立子代理会话 tabs（better-sidebar openSubagent 语义）：点击任务页里的
  // 子代理节点 → 主对话区上方新增一个独立会话 tab（可关闭、可并行切换），
  // 不替换主会话、也不在右栏开轨迹式全面板。
  const subRunsCacheRef = useRef<SubagentRunView[]>([]);
  const [subagentTabs, setSubagentTabs] = useState<
    Array<{
      id: string;
      sessionPath: string;
      ref: string;
      task?: string;
      model?: string;
      kind?: "subagent" | "model_tool";
      tool?: string;
      status: SubagentThreadStatus;
    }>
  >([]);
  const [subagentTabId, setSubagentTabId] = useState<string | null>(null);
  useEffect(() => {
    setSubagentTabs([]);
    setSubagentTabId(null);
  }, [currentSessionKey]);
  const openSubagentThread = useCallback(
    (p: {
      sessionPath: string;
      ref: string;
      task?: string;
      model?: string;
      status: SubagentThreadStatus;
    }) => {
      const id = `sub:${p.ref}`;
      setSubagentTabs((prev) => (
        prev.some((x) => x.id === id)
          ? prev
          : [...prev, { ...p, id, kind: p.ref.startsWith("mt_") ? "model_tool" as const : undefined }]
      ));
      setSubagentTabId(id);
      setChatTab("chat");
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setChatTab 为 React dispatch（稳定引用），原 deps [] 语义保留
    [],
  );
  const closeSubagentTab = useCallback(
    (id: string) => {
      const next = subagentTabs.filter((x) => x.id !== id);
      setSubagentTabs(next);
      setSubagentTabId((cur) => (cur === id ? (next[next.length - 1]?.id ?? null) : cur));
    },
    [subagentTabs],
  );
  // v4.68 task 卡多候选歧义选择器：空 ref 命中 ≥2 个 running 时点击弹此
  // 轻量选择器，人工挑一个跳转（宁缺勿错：选择器只由用户点击触发，绝不
  // 自动跳转；0/1 候选走原「唯一 running 命中」直跳链路，行为不变）。
  // state 只存点击瞬间的候选快照（SubagentRunView 原样引用），App 本地即可。
  const [taskPickCandidates, setTaskPickCandidates] = useState<SubagentRunView[] | null>(null);
  // 独立子代理 tab 实时状态同步（v4.63 换共享单轮询 store）：打开后运行/
  // 完成/失败与模型随 GaeaSubagentRuns 刷新——tab 状态点与 SubagentThread
  // 头部的状态徽标不再停留在点击瞬间的快照。轮询本身由 subagentRunsStore
  // 收敛为每会话单定时器（与 task 卡活动/任务树同源），App 只做派生合并。
  const [subagentRuns, setSubagentRuns] = useState<SubagentRunView[]>([]);
  useEffect(() => {
    if (!currentSessionPath) {
      setSubagentRuns([]);
      return;
    }
    return subscribeSubagentRuns(currentSessionPath, setSubagentRuns);
  }, [currentSessionPath]);
  useEffect(() => {
    subRunsCacheRef.current = subagentRuns;
    setSubagentTabs((prev) => {
      if (prev.length === 0 || subagentRuns.length === 0) return prev;
      let changed = false;
      const next = prev.map((tab) => {
        const run = subagentRuns.find((r) => r.ref === tab.ref);
        if (!run) return tab;
        const model = run.model ?? tab.model;
        const task = run.task || tab.task;
        const kind = run.kind ?? tab.kind;
        const tool = run.tool ?? tab.tool;
        if (run.status === tab.status && model === tab.model && task === tab.task &&
            kind === tab.kind && tool === tab.tool) {
          return tab;
        }
        changed = true;
        return { ...tab, status: run.status, model, task, kind, tool };
      });
      return changed ? next : prev;
    });
  }, [subagentRuns]);
  const handleChatTabSelect = useCallback((id: string) => {
    if (id === "chat" || id === "trajectory" || id === "context" || id === "memory") {
      setSubagentTabId(null);
      setChatTab(id);
    } else {
      setSubagentTabId(id);
      setChatTab("chat");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setChatTab 为 React dispatch（稳定引用），原 deps [] 语义保留
  }, []);

  return {
    subRunsCacheRef,
    subagentTabs,
    subagentTabId,
    openSubagentThread,
    closeSubagentTab,
    taskPickCandidates,
    setTaskPickCandidates,
    subagentRuns,
    handleChatTabSelect,
  };
}
