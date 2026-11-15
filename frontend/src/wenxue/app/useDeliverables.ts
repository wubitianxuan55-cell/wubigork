// useDeliverables — 会话产物/文件变更派生 + 新产物角标与自动置前。
// 纯派生（sessionDeliverables/sessionChanges useMemo）与「首现即新」基线 diff
// （freshDeliverablePaths）从 App.tsx 原样迁出，执行顺序与 deps 不变。
import { useEffect, useMemo, useRef, useState } from "react";
import type { SessionDeliverable } from "../components/DeliverablesPanel";
import { buildSessionChanges, extractDeliverablePaths, WRITE_TOOL_NAMES, type SessionChange } from "../lib/changes";
import { shouldAutoOpenDeliverables } from "../lib/deliverablePrefs";
import { DELIVERABLE_EXT_RE, deliverableMentions } from "../lib/fileLinks";
import type { ControllerState } from "../lib/store";
import type { WorkspaceTabId } from "../lib/workspaceTabs";

interface UseDeliverablesParams {
  state: ControllerState;
  currentSessionKey: string;
  rightTab: WorkspaceTabId | null;
  openPaneView: (viewId: WorkspaceTabId) => void;
}

export function useDeliverables({ state, currentSessionKey, rightTab, openPaneView }: UseDeliverablesParams) {
  // 会话产物：从会话消息文本中提取交付文件 + 写类工具落盘的显式登记
  // （保留首现顺序；同一文件多次出现计入 versions 次数——产物版本时间线
  // 数据源，对标 Hermes 版本步进器）。显式登记修复：Agent 未在正文提及
  // 路径时成果面板漏登记的启发式缺陷。
  const sessionDeliverables = useMemo<SessionDeliverable[]>(() => {
    const order = new Map<string, { sourceId: string; turn: number; versions: number }>();
    let turn = -1;
    const register = (p: string, sourceId: string) => {
      const rec = order.get(p);
      if (rec) {
        rec.versions++;
      } else {
        order.set(p, { sourceId, turn: Math.max(0, turn), versions: 1 });
      }
    };
    for (const it of state.items) {
      if (it.kind === "user") {
        turn++;
        continue;
      }
      // 写类工具落盘 = 显式登记：工具参数里的路径是真实写入，不依赖正文提及
      if (it.kind === "tool" && WRITE_TOOL_NAMES.has(it.name)) {
        for (const p of extractDeliverablePaths(it.args || "")) {
          if (DELIVERABLE_EXT_RE.test(p)) register(p, it.id);
        }
        continue;
      }
      if (it.kind !== "assistant" || !it.text) continue;
      for (const p of deliverableMentions(it.text)) {
        register(p, it.id);
      }
    }
    const out: SessionDeliverable[] = [];
    for (const [path, rec] of order) {
      out.push({ path, sourceId: rec.sourceId, turn: rec.turn, versions: rec.versions });
    }
    return out;
  }, [state.items]);

  // 文件变更（Kun 可观察性精华）：汇总本会话写/改过的文件及次数
  const sessionChanges = useMemo<SessionChange[]>(() => buildSessionChanges(state.items), [state.items]);

  // ── v4.30 产物自动置前/角标（Devin Auto-open 式）────────────────────────
  // 本会话内新出现的产物路径：diff sessionDeliverables（首现即新），产物 tab
  // 角标显示未读数、产物面板对应行显示「新」徽标+高亮；激活产物 tab（查看）
  // 即清零（与运行角标「激活即已读」语义一致）。
  // 会话切换：重置为「当前产物全集」基线（恢复会话不误标新），并清空角标。
  const [freshDeliverablePaths, setFreshDeliverablePaths] = useState<string[]>([]);
  const seenDeliverablePathsRef = useRef<Set<string>>(new Set());
  const baselinePendingRef = useRef(true);
  useEffect(() => {
    baselinePendingRef.current = true;
    seenDeliverablePathsRef.current = new Set(sessionDeliverables.map((d) => d.path));
    setFreshDeliverablePaths([]);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 仅会话切换时重置基线，快照取当前值
  }, [currentSessionKey]);
  useEffect(() => {
    // 切换后的首次运行：把当前会话产物全量预填为基线（不算新），之后流式
    // 增长中出现的路径才标新。
    if (baselinePendingRef.current) {
      baselinePendingRef.current = false;
      seenDeliverablePathsRef.current = new Set(sessionDeliverables.map((d) => d.path));
      return;
    }
    const seen = seenDeliverablePathsRef.current;
    const added: string[] = [];
    for (const d of sessionDeliverables) {
      if (!seen.has(d.path)) {
        seen.add(d.path);
        added.push(d.path);
      }
    }
    if (added.length > 0) {
      setFreshDeliverablePaths((prev) => [...prev, ...added]);
      // v4.32 产物自动弹出：偏好开 → 亮右栏并开「产物」视图 tab（pane 语义；
      // 激活即清零角标，自动弹出 = 已查看）。不动 FilePreview——产物 tab 与
      // 主区预览不冲突。
      if (shouldAutoOpenDeliverables() && rightTab !== "deliverables") {
        openPaneView("deliverables");
      }
    }
  }, [sessionDeliverables, rightTab, openPaneView]);
  // 激活产物 tab = 已查看 → 清零角标与「新」徽标
  useEffect(() => {
    if (rightTab === "deliverables") setFreshDeliverablePaths([]);
  }, [rightTab]);
  // v4.30 产物自动置前：未查看的新产物数角标（激活产物 tab 即清零，见上方 effect）
  const freshDeliverableCount = rightTab === "deliverables" ? 0 : freshDeliverablePaths.length;

  return {
    sessionDeliverables,
    sessionChanges,
    freshDeliverablePaths,
    freshDeliverableCount,
  };
}