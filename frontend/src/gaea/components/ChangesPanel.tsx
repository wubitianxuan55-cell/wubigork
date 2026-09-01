import { useEffect, useMemo, useState } from "react";
import { ChevronDown, Diff, Eye, FileText, Loader2, Rollback } from "../icons";
import { useCompact } from "../hooks/useCompact";
import { useStore, type Item } from "../lib/store";
import { buildChangeCalls, pathsMatch } from "../lib/planDiff";
import type { SessionChange } from "../lib/changes";
import { app } from "../lib/bridge";
import type { JournalChangeRecord } from "../lib/types";
import { useToast } from "./Toast";
import { ChangesDiff } from "./ChangesDiff";

// ── 文件变更面板（Kun 可观察性精华）─────────────────────────────────
// 汇总本会话中 Agent 写/改过的文件（write_file / edit_file / move_file 等）。
//
// v4.25「变更 tab diff 化」：文件行可展开 → 每次写类调用的行级红绿 diff
// （edit_file/multi_edit 的 old_string→new_string，planDiff.ts 构造）；无
// old/new 的写工具（write_file/edit_lines）诚实降级为「写入内容预览」，
// 不伪造 diff。恢复：接入证据链 Journal 基线快照（GaeaJournalList →
// RollbackRecord，与交付面板同一绑定面）；无基线的文件显式标注暂不可回滚。
//
// v3「星枢」面板语言：v3-panel-head 细条头部；行 hover 柔光、图标/计数走令牌。

// 展开区单条调用徽标文案（中文工具名映射，未知工具显示原名）。
const TOOL_LABELS: Record<string, string> = {
  write_file: "写入",
  edit_file: "编辑",
  edit_lines: "按行编辑",
  multi_edit: "多处编辑",
  move_file: "移动",
  notebook_edit: "Notebook",
  delete_range: "删除片段",
  delete_symbol: "删除符号",
};

function relPath(path: string, cwd?: string): string {
  const p = path.replace(/\\/g, "/");
  const base = (cwd || "").replace(/\\/g, "/").replace(/\/+$/, "");
  if (base && p.startsWith(base + "/")) return p.slice(base.length + 1);
  return p;
}

// Journal 的最近证据卡（懒加载：首次展开才拉取；跨会话聚合、时间倒序）。
const JOURNAL_LIMIT = 200;

export function ChangesPanel({
  changes,
  cwd,
  onOpenFile,
  items: itemsProp,
}: {
  changes: SessionChange[];
  cwd?: string;
  onOpenFile: (path: string) => void;
  /** 会话 items（可选）：缺省时订阅全局会话 store（与 sidebarRegistry 传参解耦）。 */
  items?: Item[];
}) {
  const compact = useCompact();
  const toast = useToast();
  const storeItems = useStore((s) => s.items);
  const items = itemsProp ?? storeItems;

  // 路径 → 写类调用明细（与 App 的 buildSessionChanges 同源口径）。
  const callsByPath = useMemo(() => buildChangeCalls(items), [items]);

  // 当前展开的文件行（单开，保持面板紧凑）。
  const [expanded, setExpanded] = useState<string | null>(null);
  // 证据链 Journal（回滚原料）：expanded 变化时刷新。
  const [journal, setJournal] = useState<JournalChangeRecord[] | null>(null);
  const [journalLoading, setJournalLoading] = useState(false);
  const [rollingBack, setRollingBack] = useState(false);

  useEffect(() => {
    if (!expanded) return;
    let live = true;
    setJournalLoading(true);
    app
      .GaeaJournalList(JOURNAL_LIMIT)
      .then((r) => {
        if (live) setJournal(r ?? []);
      })
      .catch(() => {
        if (live) setJournal([]);
      })
      .finally(() => {
        if (live) setJournalLoading(false);
      });
    return () => {
      live = false;
    };
  }, [expanded]);

  // 该路径可用的回滚记录：匹配 target 且带基线快照、排除 rollback 自身记录，
  // 取最近一条（基线 = 写盘前完整内容快照）。target 与事件流 path 可能一边
  // 绝对一边相对：先带边界后缀匹配，再按 cwd 锚定的相对路径精确比对。
  const rollbackRecord = useMemo(() => {
    if (!expanded || !journal) return null;
    const candidates = journal.filter(
      (r) => r.tool !== "rollback" && !!r.baselinePath && r.target && r.id,
    );
    return (
      candidates.find((r) => pathsMatch(r.target, expanded)) ??
      (cwd
        ? candidates.find((r) => relPath(r.target, cwd) === relPath(expanded, cwd))
        : undefined) ??
      null
    );
  }, [expanded, journal, cwd]);

  const doRollback = async () => {
    if (!rollbackRecord || rollingBack) return;
    setRollingBack(true);
    try {
      await app.RollbackRecord(rollbackRecord.id);
      toast.show(`已回滚 ${relPath(expanded ?? "", cwd)}（基线快照恢复）`, "info");
    } catch (e) {
      toast.show(`回滚失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    } finally {
      setRollingBack(false);
    }
  };

  const totalChanges = changes.reduce((sum, c) => sum + c.count, 0);
  const sorted = [...changes].sort((a, b) => (b.lastTouched ?? 0) - (a.lastTouched ?? 0));
  if (changes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 px-6 text-center">
        <span
          className="w-10 h-10 rounded-[var(--radius-md)] flex items-center justify-center mb-3"
          style={{
            background: "color-mix(in srgb, var(--gaea-glow) 9%, transparent)",
            border: "1px solid color-mix(in srgb, var(--gaea-glow) 20%, transparent)",
            color: "var(--gaea-glow)",
          }}
        >
          <Diff size={18} aria-hidden />
        </span>
        <div className="text-(color:--md-sys-color-text-secondary) text-[12.5px] font-medium">
          本会话暂无文件变更
        </div>
        <div
          className="mt-1 text-[11px] leading-snug max-w-[220px]"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        >
          Agent 写入或修改工作区文件后，会在这里汇总，点击可打开预览
        </div>
      </div>
    );
  }
  return (
    <div className="flex flex-col py-1">
      {/* v3 细条头部：标题 + 汇总计数 */}
      <div className="v3-panel-head">
        <Diff size={12} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">文件变更</span>
        <span className="v3-panel-spacer" />
        <span className="font-mono text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {changes.length} 个文件 · {totalChanges} 次
        </span>
      </div>
      <div className="flex flex-col gap-px px-1.5 pt-1.5 pb-1">
        {sorted.map((c) => {
          const name = c.path.split(/[\\/]/).filter(Boolean).pop() || c.path;
          const rel = relPath(c.path, cwd);
          const isOpen = expanded === c.path;
          const calls = callsByPath.get(c.path) ?? [];
          // 展示顺序：最新一次改动在最上（与行排序同向）。
          const shownCalls = [...calls].reverse();
          return (
            <div key={c.path}>
              <button
                className="group flex items-center gap-2.5 px-2.5 py-2 text-left rounded-[var(--radius-md)] border border-transparent bg-transparent cursor-pointer transition-all duration-200 hover:bg-(color:--md-sys-color-surface-container-high) hover:shadow-[var(--v3-glow-faint)]"
                onClick={() => setExpanded(isOpen ? null : c.path)}
                title={isOpen ? "收起 diff" : `展开 ${rel} 的改动 diff`}
                aria-label={isOpen ? `收起 ${rel} 的改动 diff` : `展开 ${rel} 的改动 diff`}
                aria-expanded={isOpen}
              >
                <ChevronDown
                  size={11}
                  aria-hidden
                  className={`shrink-0 transition-transform duration-200 ${isOpen ? "" : "-rotate-90"}`}
                  style={{ color: "var(--md-sys-color-text-secondary)" }}
                />
                <span
                  className="w-7 h-7 rounded-md flex items-center justify-center shrink-0 transition-colors"
                  style={{
                    background: "color-mix(in srgb, var(--gaea-glow) 8%, transparent)",
                    border: "1px solid var(--md-sys-color-outline-variant)",
                    color: "var(--md-sys-color-text-secondary)",
                  }}
                >
                  <FileText size={13} aria-hidden />
                </span>
                <span className="min-w-0 flex-1">
                  <span
                    className={`block truncate text-fg-dim font-medium ${
                      compact ? "text-[11.5px]" : "text-[12.5px]"
                    }`}
                  >
                    {name}
                  </span>
                  {/* v4.30 行级降噪：相对路径为次级信息，悬停次行显现（title 保留完整寻回路径） */}
                  <span
                    className="block truncate font-mono text-[10px] transition-opacity duration-150 group-hover:opacity-100 opacity-0"
                    style={{ color: "var(--md-sys-color-text-secondary)" }}
                  >
                    {rel}
                  </span>
                </span>
                <span
                  className="shrink-0 rounded-full font-mono text-[10px] px-1.5 py-px"
                  style={{
                    background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
                    color: "var(--gaea-glow)",
                    border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
                  }}
                >
                  {c.count} 次
                </span>
              </button>
              {isOpen && (
                <div className="ml-9 mr-1.5 mb-1.5 flex flex-col gap-2 px-2.5 py-2 rounded-[var(--radius-md)] border border-border-soft bg-bg-soft/30">
                  {/* 操作条：打开预览 + 回滚（证据链基线快照） */}
                  <div className="flex flex-wrap items-center gap-1.5">
                    <button
                      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft hover:text-fg transition-colors"
                      onClick={() => onOpenFile(c.path)}
                      title="打开文件预览"
                    >
                      <Eye size={10} aria-hidden />
                      打开预览
                    </button>
                    {journalLoading ? (
                      <span className="inline-flex items-center gap-1 text-[10px] text-fg-faint">
                        <Loader2 size={10} className="animate-spin" />
                        检查快照…
                      </span>
                    ) : rollbackRecord ? (
                      <button
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md border border-err/40 bg-transparent text-err text-[10px] cursor-pointer hover:bg-err/10 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        onClick={() => void doRollback()}
                        disabled={rollingBack}
                        title={`回滚到写盘前基线快照（turn ${rollbackRecord.turn} · ${rollbackRecord.tool}）`}
                      >
                        {rollingBack ? <Loader2 size={10} className="animate-spin" /> : <Rollback size={10} aria-hidden />}
                        回滚此文件
                      </button>
                    ) : (
                      <span className="text-[10px] text-fg-faint" title="证据链 Journal 无该文件的基线快照">
                        暂无基线快照，不可回滚（远期：写类工具统一前后快照）
                      </span>
                    )}
                  </div>
                  {/* 每次写类调用的内容级 diff（最新在上） */}
                  {shownCalls.length === 0 ? (
                    <div className="text-[10.5px] text-fg-faint">
                      本会话事件流中没有该文件的内容级参数记录（可能是恢复自历史会话），点击「打开预览」查看当前内容。
                    </div>
                  ) : (
                    shownCalls.map((call) => (
                      <div key={call.itemId} className="flex flex-col gap-1" data-testid="changes-call">
                        <div className="flex items-center gap-1.5 text-[10px]">
                          <span
                            className="rounded-full px-1.5 py-px"
                            style={{
                              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
                              color: "var(--gaea-glow)",
                            }}
                          >
                            {TOOL_LABELS[call.tool] ?? call.tool}
                          </span>
                          {!call.applied && (
                            <span className="text-fg-faint" title="该次调用未成功，不构成实际变更">
                              未成功，不构成变更
                            </span>
                          )}
                        </div>
                        {call.applied ? (
                          <ChangesDiff diff={call.diff} />
                        ) : (
                          <div className="text-[10.5px] text-fg-faint">调用未成功落盘，无内容变化。</div>
                        )}
                      </div>
                    ))
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
