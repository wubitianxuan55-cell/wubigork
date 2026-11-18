import { useEffect, useMemo, useRef, useState } from "react";
import { ChevronDown, MessageSquare, PanelRight } from "../icons";
import { useT } from "../lib/i18n";
import { subscribeSubagentRuns } from "../lib/subagentRunsStore";
import type { SubagentRunView } from "../lib/types";
import type { SubagentThreadStatus } from "./SubagentThread";

// SubagentCatalogButton — 主会话顶栏「N 个子智能体」目录树（2026-11-18，
// dsh ui-subagent S2/S3/S4 形态：数量触发器 + 下拉多级缩进树 + 行尾「在侧
// 边栏打开」）。数据 = subagentRunsStore 共享单轮询快照（零新增轮询），按
// parentRef 血缘建树（嵌套委派后任意深度）。交互：点击开合（钉住）、Esc/
// 点击遮罩关闭；行点击=主区打开会话 tab，侧栏钮=右栏子会话 pane。
//
// 只在 runs>0 时渲染触发器（dsh S2 可见性：目录存在且有条目）。

interface CatalogNode {
  run: SubagentRunView;
  children: CatalogNode[];
  depth: number;
}

function statusDotColor(status: SubagentThreadStatus): string {
  switch (status) {
    case "running":
      return "var(--gaea-glow)";
    case "failed":
      return "var(--md-sys-color-destructive)";
    case "aborted":
    case "queued":
      return "var(--fg-faint)";
    default:
      return "var(--md-sys-color-success)";
  }
}

function buildCatalog(runs: SubagentRunView[]): CatalogNode[] {
  const byRef = new Map<string, CatalogNode>();
  for (const run of runs) byRef.set(run.ref, { run, children: [], depth: 0 });
  const roots: CatalogNode[] = [];
  for (const node of byRef.values()) {
    const parent = node.run.parentRef ? byRef.get(node.run.parentRef) : undefined;
    if (parent) {
      node.depth = parent.depth + 1;
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  }
  // 稳定序：created 倒序（runs 快照已是倒序，递归保持）。
  return roots;
}

function flattenCollapsed(nodes: CatalogNode[]): CatalogNode[] {
  const out: CatalogNode[] = [];
  const walk = (list: CatalogNode[]) => {
    for (const n of list) {
      out.push(n);
      walk(n.children);
    }
  };
  walk(nodes);
  return out;
}

export function SubagentCatalogButton({
  sessionPath,
  onOpenMain,
  onOpenAside,
}: {
  sessionPath?: string;
  /** 行点击/「打开对话」：主区子代理会话 tab（App 层打开链）。 */
  onOpenMain: (run: SubagentRunView) => void;
  /** 「在侧边栏打开」：右栏子会话 pane（dsh S4）。缺省=不渲染侧栏钮。 */
  onOpenAside?: (run: SubagentRunView) => void;
}) {
  const t = useT();
  const [runs, setRuns] = useState<SubagentRunView[]>([]);
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!sessionPath) {
      setRuns([]);
      return;
    }
    return subscribeSubagentRuns(sessionPath, (next) => {
      setRuns(next.filter((r) => r.kind !== "model_tool"));
    });
  }, [sessionPath]);

  const nodes = useMemo(() => buildCatalog(runs), [runs]);
  const rows = useMemo(() => flattenCollapsed(nodes), [nodes]);
  const runningCount = runs.filter((r) => r.status === "running").length;

  // Esc / 点击面板外关闭（钉住语义：开合只由触发钮驱动）。
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    const onClick = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("mousedown", onClick);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("mousedown", onClick);
    };
  }, [open]);

  if (!sessionPath || runs.length === 0) return null;

  return (
    <div ref={wrapRef} className="relative" data-testid="subagent-catalog">
      <button
        type="button"
        data-testid="subagent-catalog-trigger"
        className="flex cursor-pointer items-center gap-1 rounded-md border-0 bg-transparent px-1.5 py-0.5 text-[11px] transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
        style={{ color: "var(--md-sys-color-text-secondary)" }}
        aria-haspopup="true"
        aria-expanded={open}
        title={t("subagent.catalogTitle")}
        onClick={() => setOpen((v) => !v)}
      >
        <MessageSquare size={12} aria-hidden />
        {runningCount > 0 && (
          <span className="inline-block h-1.5 w-1.5 rounded-full animate-pulse" style={{ background: "var(--gaea-glow)" }} aria-hidden />
        )}
        {t("subagent.catalogTrigger", { n: runs.length })}
        <ChevronDown size={10} aria-hidden />
      </button>
      {open && (
        <div
          data-testid="subagent-catalog-dropdown"
          role="menu"
          className="absolute right-0 top-full z-40 mt-1 max-h-80 min-w-[260px] overflow-y-auto rounded-lg border py-1 shadow-lg"
          style={{
            borderColor: "var(--md-sys-color-outline-variant)",
            background: "var(--md-sys-color-surface-container)",
          }}
        >
          {rows.map((node) => {
            const dot = statusDotColor(node.run.status);
            return (
              <div
                key={node.run.ref}
                role="menuitem"
                tabIndex={-1}
                data-testid="subagent-catalog-row"
                className="group flex cursor-pointer items-center gap-1.5 px-2 py-1 transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
                style={{ paddingLeft: 8 + node.depth * 14 }}
                onClick={() => {
                  onOpenMain(node.run);
                  setOpen(false);
                }}
              >
                <span
                  className={"inline-block h-1.5 w-1.5 shrink-0 rounded-full" + (node.run.status === "running" ? " animate-pulse" : "")}
                  style={{ background: dot }}
                  aria-hidden
                />
                <span className="min-w-0 flex-1 truncate text-[11.5px]" style={{ color: "var(--md-sys-color-text)" }} title={node.run.task || node.run.ref}>
                  {node.run.task || node.run.ref}
                </span>
                {node.depth > 0 && (
                  <span className="shrink-0 rounded px-1 font-mono text-[9px]" style={{ color: "var(--fg-faint)", background: "var(--md-sys-color-surface-container-high)" }}>
                    L{node.depth}
                  </span>
                )}
                {onOpenAside && (
                  <button
                    type="button"
                    data-testid="subagent-catalog-aside"
                    aria-label={t("subagent.openAside")}
                    title={t("subagent.openAside")}
                    className="flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center rounded border-0 bg-transparent p-0 opacity-0 transition-opacity hover:bg-(color:--md-sys-color-surface-container-high) group-hover:opacity-100"
                    style={{ color: "var(--md-sys-color-text-secondary)" }}
                    onClick={(e) => {
                      e.stopPropagation();
                      onOpenAside(node.run);
                      setOpen(false);
                    }}
                  >
                    <PanelRight size={11} aria-hidden />
                  </button>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
