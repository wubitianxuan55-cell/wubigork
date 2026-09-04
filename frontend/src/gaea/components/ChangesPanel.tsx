import { useEffect, useMemo, useState } from "react";
import { ChevronDown, Diff, Eye, FileText, Loader2, Rollback } from "../icons";
import { useCompact } from "../hooks/useCompact";
import { useStore, type Item } from "../lib/store";
import { buildChangeCalls, pathsMatch, type ChangeCall } from "../lib/planDiff";
import {
  EDIT_TOOL_NAMES, WRITE_ONLY_TOOL_NAMES,
  buildSessionChanges, buildSessionReads, categoryOf, type FileCat,
} from "../lib/changes";
import type { SessionChange } from "../lib/changes";
import { app } from "../lib/bridge";
import type { JournalChangeRecord } from "../lib/types";
import { useToast } from "./Toast";
import { ChangesDiff } from "./ChangesDiff";

// ── 文件变更面板（Kun 可观察性精华）─────────────────────────────────
// 汇总本会话中 Agent 接触过的文件。
//
// v4.25「变更 tab diff 化」：文件行可展开 → 每次写类调用的行级红绿 diff
// （edit_file/multi_edit 的 old_string→new_string，planDiff.ts 构造）；无
// old/new 的写工具（write_file/edit_lines）诚实降级为「写入内容预览」，
// 不伪造 diff。恢复：接入证据链 Journal 基线快照（GaeaJournalList →
// RollbackRecord，与交付面板同一绑定面）；无基线的文件显式标注暂不可回滚。
//
// v4.85「2a 三态折叠补全」（对标源 better-sidebar v0.18 统一文件变动）：
// 写入 / 编辑 / 读取三层独立折叠——同一文件可同时出现在多层（读过后又被
// 写=两层各一条，独立语义）；读取层轻量行（无 diff，点击直接开预览），
// 默认收起降噪；按扩展名类型筛选 chips（文档/表格/图片/代码/其他）横贯
// 三层过滤。

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

// 2a 类型筛选 chips（按扩展名分桶，categoryOf 同口径）。
const CAT_LABELS: { key: FileCat; label: string }[] = [
  { key: "all", label: "全部" },
  { key: "doc", label: "文档" },
  { key: "sheet", label: "表格" },
  { key: "image", label: "图片" },
  { key: "code", label: "代码" },
  { key: "other", label: "其他" },
];

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

  // ── 2a 三态派生：items 为权威源（与 App sessionChanges 同源）；prop
  // changes 仅在 items 为空时作合并层兼容回退（历史恢复场景）。
  const hasItems = items.length > 0;
  const writeChanges = useMemo(
    () => (hasItems ? buildSessionChanges(items, WRITE_ONLY_TOOL_NAMES) : changes),
    [hasItems, items, changes],
  );
  const editChanges = useMemo(
    () => (hasItems ? buildSessionChanges(items, EDIT_TOOL_NAMES) : []),
    [hasItems, items],
  );
  const readChanges = useMemo(() => (hasItems ? buildSessionReads(items) : []), [hasItems, items]);
  const mergedChanges = useMemo(() => (hasItems ? [] : changes), [hasItems, changes]);
  const writeCalls = useMemo(() => buildChangeCalls(items, WRITE_ONLY_TOOL_NAMES), [items]);
  const editCalls = useMemo(() => buildChangeCalls(items, EDIT_TOOL_NAMES), [items]);

  // 三层折叠态（读取层默认收起降噪；写/编辑保持原行为=默认展开可见）。
  const [layerOpen, setLayerOpen] = useState<{ write: boolean; edit: boolean; read: boolean }>({
    write: true,
    edit: true,
    read: false,
  });
  // 类型筛选（按扩展名分桶，横贯三层过滤）。
  const [cat, setCat] = useState<FileCat>("all");

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

  // 头部汇总：三并集口径（文件数去重、次数按层合计；兼容层并进汇总）。
  const unionPaths = useMemo(() => {
    const s = new Set<string>();
    for (const c of [...writeChanges, ...editChanges, ...readChanges]) s.add(c.path);
    return s;
  }, [writeChanges, editChanges, readChanges]);
  const totalChanges = [...writeChanges, ...editChanges, ...readChanges].reduce(
    (sum, c) => sum + c.count,
    0,
  );

  // 类型筛选 chips 计数：并集路径按扩展名分桶（文件数口径）。
  const catCounts = useMemo(() => {
    const counts: Record<FileCat, number> = {
      all: unionPaths.size, doc: 0, sheet: 0, image: 0, code: 0, other: 0,
    };
    for (const p of unionPaths) counts[categoryOf(p)] += 1;
    return counts;
  }, [unionPaths]);
  const inCat = (path: string) => cat === "all" || categoryOf(path) === cat;
  const filt = (list: SessionChange[]) =>
    [...list].sort((a, b) => (b.lastTouched ?? 0) - (a.lastTouched ?? 0)).filter((c) => inCat(c.path));

  const fWrite = filt(writeChanges);
  const fEdit = filt(editChanges);
  const fRead = filt(readChanges);
  const fMerged = filt(mergedChanges);
  const anyVisible = fWrite.length + fEdit.length + fRead.length + fMerged.length > 0;

  if (unionPaths.size === 0) {
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

  // 渲染一层「写/编辑/兼容合并」文件行（展开=该层调用的内容级 diff+操作条）。
  const renderChangeRows = (list: SessionChange[], calls: Map<string, ChangeCall[]>) =>
    list.map((c) => {
      const name = c.path.split(/[\\/]/).filter(Boolean).pop() || c.path;
      const rel = relPath(c.path, cwd);
      const isOpen = expanded === c.path;
      const shownCalls = [...(calls.get(c.path) ?? [])].reverse();
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
              {/* 每次调用的内容级 diff（最新在上） */}
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
                      <ChangesDiff diff={call.diff} path={c.path} />
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
    });

  // 折叠层头部（独立折叠）。
  const layerHead = (key: "write" | "edit" | "read", label: string, n: number) => {
    const open = layerOpen[key];
    return (
      <button
        type="button"
        aria-expanded={open}
        data-testid={`changes-layer-${key}`}
        className="mt-1 flex w-full cursor-pointer items-center gap-1.5 border-0 bg-transparent px-2.5 py-1 text-left first:mt-0"
        onClick={() => setLayerOpen((cur) => ({ ...cur, [key]: !cur[key] }))}
      >
        <ChevronDown
          size={10}
          aria-hidden
          className={`shrink-0 transition-transform duration-200 ${open ? "" : "-rotate-90"}`}
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        />
        <span className="text-[10.5px] font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {label}
        </span>
        <span
          className="rounded-full px-1.5 text-[9px] tabular-nums"
          style={{
            background: "var(--md-sys-color-surface-container-high)",
            color: "var(--md-sys-color-text-secondary)",
          }}
        >
          {n}
        </span>
      </button>
    );
  };

  // 读取层轻量行：无 diff 语义，点击直接开预览。
  const renderReadRows = (list: SessionChange[]) =>
    list.map((c) => {
      const name = c.path.split(/[\\/]/).filter(Boolean).pop() || c.path;
      const rel = relPath(c.path, cwd);
      return (
        <button
          key={c.path}
          className="group flex items-center gap-2.5 px-2.5 py-1.5 text-left rounded-[var(--radius-md)] border border-transparent bg-transparent cursor-pointer transition-all duration-200 hover:bg-(color:--md-sys-color-surface-container-high)"
          onClick={() => onOpenFile(c.path)}
          title={`打开 ${rel}`}
          aria-label={`打开 ${rel} 预览`}
        >
          <span
            className="w-6 h-6 rounded-md flex items-center justify-center shrink-0"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 6%, transparent)",
              border: "1px solid var(--md-sys-color-outline-variant)",
              color: "var(--md-sys-color-text-secondary)",
            }}
          >
            <FileText size={11} aria-hidden />
          </span>
          <span className="min-w-0 flex-1">
            <span className={`block truncate text-fg-dim ${compact ? "text-[11px]" : "text-[12px]"}`}>{name}</span>
            <span
              className="block truncate font-mono text-[9.5px] transition-opacity duration-150 group-hover:opacity-100 opacity-0"
              style={{ color: "var(--md-sys-color-text-secondary)" }}
            >
              {rel}
            </span>
          </span>
          <span
            className="shrink-0 font-mono text-[9.5px] tabular-nums"
            style={{ color: "var(--md-sys-color-text-secondary)" }}
          >
            读 {c.count} 次
          </span>
        </button>
      );
    });

  return (
    <div className="flex flex-col py-1">
      {/* v3 细条头部：标题 + 汇总计数 */}
      <div className="v3-panel-head">
        <Diff size={12} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">文件变更</span>
        <span className="v3-panel-spacer" />
        <span className="font-mono text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {unionPaths.size} 个文件 · {totalChanges} 次
        </span>
      </div>
      {/* 2a 类型筛选 chips（按扩展名分桶；计数=并集文件数） */}
      <div className="flex flex-wrap items-center gap-1 px-2 pt-1.5">
        {CAT_LABELS.map(({ key, label }) => (
          <button
            key={key}
            type="button"
            aria-pressed={cat === key}
            data-testid={`changes-cat-${key}`}
            className={`cursor-pointer rounded-full border-0 px-2 py-0.5 text-[10px] transition-colors ${
              cat === key ? "bg-accent/15 text-accent" : "bg-transparent hover:text-fg"
            }`}
            style={{ color: cat === key ? undefined : "var(--md-sys-color-text-secondary)" }}
            onClick={() => setCat(key)}
          >
            {label}
            <span className="ml-1 tabular-nums opacity-70">{catCounts[key]}</span>
          </button>
        ))}
      </div>
      <div className="flex flex-col gap-px px-1.5 pt-1 pb-1">
        {!hasItems ? (
          <>
            {layerHead("write", "变更", fMerged.length)}
            {layerOpen.write && fMerged.length > 0 && renderChangeRows(fMerged, writeCalls)}
          </>
        ) : (
          <>
            {layerHead("write", "写入", fWrite.length)}
            {layerOpen.write && fWrite.length > 0 && renderChangeRows(fWrite, writeCalls)}
            {layerHead("edit", "编辑", fEdit.length)}
            {layerOpen.edit && fEdit.length > 0 && renderChangeRows(fEdit, editCalls)}
            {layerHead("read", "读取", fRead.length)}
            {layerOpen.read && fRead.length > 0 && renderReadRows(fRead)}
            {!anyVisible && (
              <div className="px-2.5 py-3 text-[10.5px] text-fg-faint">该类型暂无文件，换个分类试试。</div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
