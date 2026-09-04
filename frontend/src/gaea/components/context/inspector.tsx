// 上下文 inspector 两卡（对齐 dsh-context 的「上下文浏览器」与「文件活动」）。
// 独立前端线私有文件；主代理的 inspector 容器负责接线（ContextView.tsx 不动）。
// NodeRow / 预览打开逻辑自 ContextView.tsx 搬运：
// - ContextBrowserTree：六分类折叠组（色点+分类名+N项徽标+≈tokens/占比）→ 展开
//   节点列表（长文本可展开，交互同现有 NodeRow）；行内搜索框过滤节点文本；
//   末尾「归档 N」折叠组（行带「已压缩」徽标）。默认全部收起。
// - FileActivityTree：过滤 chips（全部/读取/写入/搜索/图片）+ 路径过滤 + 汇总行
//   + 排序三胶囊（按次数/按最新/按路径）+ 按文件聚合树行（点击打开预览）。
import { useMemo, useState } from "react";
// 六分类语义色：ContextView.tsx 已导出 CAT_COLORS，直接复用避免两处调色板漂移。
import { CAT_COLORS } from "../ContextView";
import { MemoMarkdown } from "../MemoMarkdown";
import { app } from "../../lib/bridge";
import "./context-view.css";
import { FolderTree, Layers } from "../../icons";
import { useT } from "../../lib/i18n";
import { FileTypeIcon } from "../../lib/fileIcon";
import { fmtTokens } from "../../lib/stats";
import { usePreviewStore } from "../../lib/store";
import type { DictKey } from "../../locales/en";
import type { ContextNodeDetailView, ContextSurfaceNode, FileActivity } from "../../lib/types";

// 分类折叠组定义（来源：ContextView.tsx 的 CATS / CAT_BROWSE_LABELS，未导出故本地
// 重声明；i18n 键复用 contextview.cat*（组行全名）与 contextview.browse*（节点行短名））。
const GROUPS: { key: ContextSurfaceNode["cat"]; labelKey: DictKey; rowKey: DictKey }[] = [
  { key: "system", labelKey: "contextview.catSystem", rowKey: "contextview.browseSystem" },
  { key: "tools", labelKey: "contextview.catTools", rowKey: "contextview.browseTools" },
  { key: "user", labelKey: "contextview.catUser", rowKey: "contextview.browseUser" },
  { key: "inject", labelKey: "contextview.catInject", rowKey: "contextview.browseInject" },
  { key: "assistant", labelKey: "contextview.catAssistant", rowKey: "contextview.browseAssistant" },
  { key: "tool", labelKey: "contextview.catTool", rowKey: "contextview.browseTool" },
];

// 分类内节点分页（dsh 同款防卡顿思路）：>200 项展开时先渲染前 100 行 + 「显示全部」。
const PAGE_THRESHOLD = 200;
const PAGE_SIZE = 100;
const ARCHIVE_KEY = "archive"; // 折叠态 key（与分类 key 不冲突的哨兵值）

// 节点详情懒加载状态机：undefined=未加载、loading、ok（缓存）、error。
type NodeDetailState = { s: "loading" } | { s: "ok"; d: ContextNodeDetailView } | { s: "error" };

const ROW_LABELS: Record<ContextSurfaceNode["cat"], DictKey> = {
  system: "contextview.browseSystem",
  tools: "contextview.browseTools",
  user: "contextview.browseUser",
  inject: "contextview.browseInject",
  assistant: "contextview.browseAssistant",
  tool: "contextview.browseTool",
};

// v4.81 节点详情懒加载 hook（浏览器与文件活动共用）：按 seq 懒加载 +
// 缓存（Map 状态机 loading/ok/error）+ 开合集合。
function useNodeDetails() {
  const [details, setDetails] = useState<Map<number, NodeDetailState>>(() => new Map());
  const [open, setOpen] = useState<Set<number>>(() => new Set());
  const toggle = (seq: number) => {
    setOpen((cur) => {
      const next = new Set(cur);
      if (next.has(seq)) next.delete(seq);
      else next.add(seq);
      return next;
    });
    if (!details.has(seq)) {
      setDetails((cur) => new Map(cur).set(seq, { s: "loading" }));
      app.ContextNodeDetail(seq)
        .then((d) => setDetails((cur) => new Map(cur).set(seq, { s: "ok", d })))
        .catch(() => setDetails((cur) => new Map(cur).set(seq, { s: "error" })));
    }
  };
  return { details, open, toggle };
}

// v4.81 节点行（搬运自 ContextView.tsx NodeRow：>56 字符可展开全文，归档节点带「已压缩」徽标）。
// v4.80 深读：tool 行加「来源」chip（工具名）与 error 语义点；tool/user/assistant
// 行可懒加载「完整调用」详情（GaeaContextNodeDetail 按 seq 回读当前会话日志），
// 详情面板带 OK/error 状态、行数、截断提示与 原文/渲染 切换（默认原文）。
const DETAILABLE = new Set<ContextSurfaceNode["cat"]>(["tool", "user", "assistant"]);

// v4.81 节点/文件操作共用的完整调用详情面板（原文/渲染切换在面板内部自持；
// 默认原文防任意工具输出噪声；渲染走 MemoMarkdown）。
function NodeDetailPanel({ d }: { d: ContextNodeDetailView }) {
  const t = useT();
  const [rendered, setRendered] = useState(false);
  const body = d.kind === "tool_result" ? d.output ?? "" : d.text ?? "";
  return (
    <div data-testid="ctx-node-detail" className="mt-1 rounded-md border border-border-soft bg-bg-soft p-1.5">
      <div className="flex flex-wrap items-center gap-1.5 text-[9.5px] text-fg-faint">
        {d.kind === "tool_result" && (
          <>
            <span
              className="rounded px-1 font-mono"
              style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text-secondary)" }}
            >
              {d.tool || "tool"}
            </span>
            <span className="font-medium" style={{ color: d.err ? "var(--md-sys-color-destructive)" : "var(--md-sys-color-success)" }}>
              {d.err ? "error" : "OK"}
            </span>
          </>
        )}
        {d.lines != null && <span className="font-mono tabular-nums">{t("contextview.detailLines", { n: d.lines })}</span>}
        {(d.clamped || d.truncated) && <span className="text-warning">{t("contextview.detailClamped")}</span>}
        <span className="ml-auto flex items-center gap-1">
          {([false, true] as const).map((md) => (
            <button
              key={String(md)}
              className={`cursor-pointer rounded border-0 px-1 py-px ${rendered === md ? "bg-accent/15 text-accent" : "bg-transparent text-fg-faint hover:text-fg"}`}
              onClick={() => setRendered(md)}
            >{md ? t("contextview.detailRendered") : t("contextview.detailRaw")}</button>
          ))}
        </span>
      </div>
      {d.kind === "tool_result" && d.args && (
        <div className="mt-1 break-all font-mono text-[9.5px] text-fg-faint" title={d.args}>→ {d.args}</div>
      )}
      <div className="mt-1 max-h-[26rem] overflow-y-auto whitespace-pre-wrap break-all font-mono text-[10px] text-fg-dim">
        {rendered ? <MemoMarkdown text={body} streaming={false} /> : body || t("contextview.noPreview")}
      </div>
    </div>
  );
}

function NodeRow({
  node,
  open,
  onToggle,
  detailState,
  detailOpen,
  onToggleDetail,
}: {
  node: ContextSurfaceNode;
  open: boolean;
  onToggle: () => void;
  detailState?: NodeDetailState;
  detailOpen: boolean;
  onToggleDetail: () => void;
}) {
  const t = useT();
  const text = node.text || t("contextview.noPreview");
  const truncated = text.length > 56;
  const shown = open || !truncated ? text : `${text.slice(0, 56)}…`;
  const detailable = DETAILABLE.has(node.cat);
  const d = detailState?.s === "ok" ? detailState.d : null;
  return (
    <div className="ctx-row flex items-start gap-1.5 px-2 py-1.5 text-[10px]">
      <span className="mt-0.5 h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[node.cat] }} />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1.5 text-fg-faint">
          <span className="font-medium" style={{ color: CAT_COLORS[node.cat] }}>{t(ROW_LABELS[node.cat])}</span>
          <span className="tabular-nums font-mono">≈{fmtTokens(node.tokens)}</span>
          {/* v4.80 来源 chip（工具名）+ error 语义点（dsh 工具行 OK/error 同款语义） */}
          {node.cat === "tool" && node.tool && (
            <span
              className="max-w-[9rem] truncate rounded px-1 font-mono"
              style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text-secondary)" }}
              title={node.tool}
            >
              {node.tool}
            </span>
          )}
          {node.cat === "tool" && node.err && (
            <span className="font-medium" style={{ color: "var(--md-sys-color-destructive)" }}>✗ error</span>
          )}
          {node.gone != null && <span className="text-warning">{t("contextview.compacted")}</span>}
          {truncated && (
            <button
              className="ml-auto cursor-pointer border-0 bg-transparent text-accent hover:underline"
              onClick={onToggle}
            >{open ? t("common.collapse") : t("common.expand")}</button>
          )}
          {detailable && (
            <button
              data-testid="ctx-node-detail-btn"
              className={`cursor-pointer border-0 bg-transparent hover:underline ${truncated ? "" : "ml-auto"} ${detailOpen ? "text-fg-dim" : "text-accent"}`}
              onClick={onToggleDetail}
            >{detailOpen ? t("contextview.detailCollapse") : t("contextview.detailBtn")}</button>
          )}
        </div>
        <div className="mt-0.5 whitespace-pre-wrap break-all font-mono text-fg-dim">{shown}</div>
        {detailOpen && detailState?.s === "loading" && (
          <div className="mt-1 text-[10px] text-fg-faint">{t("contextview.detailLoading")}</div>
        )}
        {detailOpen && detailState?.s === "error" && (
          <div className="mt-1 text-[10px]" style={{ color: "var(--md-sys-color-destructive)" }}>{t("contextview.detailFail")}</div>
        )}
        {detailOpen && d && <NodeDetailPanel d={d} />}
      </div>
    </div>
  );
}

// ─── 上下文浏览器（分类折叠组 + 行内搜索 + 归档折叠组） ──────────
// v4.80 深读：分类内排序（时间序/大小序，dsh size/name 同款）+ 节点完整
// 调用懒加载（GaeaContextNodeDetail，按 seq 缓存）。
export function ContextBrowserTree({ nodes, archive }: { nodes: ContextSurfaceNode[]; archive: ContextSurfaceNode[] }) {
  const t = useT();
  const [query, setQuery] = useState("");
  // 默认折叠态：六分类与归档全部收起（dsh 同款）。
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());
  const [openText, setOpenText] = useState<Set<number>>(() => new Set());
  const [showAll, setShowAll] = useState<Set<string>>(() => new Set());
  const [sort, setSort] = useState<"time" | "size">("time");
  const { details, open: openDetails, toggle: toggleDetail } = useNodeDetails();

  const q = query.trim().toLowerCase();
  const sorter = useMemo(
    () => (sort === "size" ? (a: ContextSurfaceNode, b: ContextSurfaceNode) => b.tokens - a.tokens || a.seq - b.seq : (a: ContextSurfaceNode, b: ContextSurfaceNode) => a.seq - b.seq),
    [sort],
  );

  // 组行右侧 ≈tokens 用未过滤的分类合计（占比分母=活跃节点总 tokens，数值稳定）；
  // 「N 项」徽标随搜索过滤（反映展开后实际列出的行数）。
  const groups = useMemo(
    () =>
      GROUPS.map((g) => {
        const all = nodes.filter((n) => n.cat === g.key);
        const hit = q === "" ? all : all.filter((n) => (n.text ?? "").toLowerCase().includes(q));
        const filtered = [...hit].sort(sorter); // v4.80 分类内排序（时间序/大小序）
        const tokens = all.reduce((s, n) => s + n.tokens, 0);
        return { ...g, all: all.length, count: filtered.length, tokens, nodes: filtered };
      }),
    [nodes, q, sorter],
  );
  const archiveShown = useMemo(() => {
    const hit = q === "" ? archive : archive.filter((n) => (n.text ?? "").toLowerCase().includes(q));
    return [...hit].sort(sorter);
  }, [archive, q, sorter]);
  const archiveTokens = useMemo(() => archive.reduce((s, n) => s + n.tokens, 0), [archive]);
  const totalTokens = useMemo(() => nodes.reduce((s, n) => s + n.tokens, 0), [nodes]);

  const toggleGroup = (key: string) => {
    setExpanded((cur) => {
      const next = new Set(cur);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };
  const toggleText = (seq: number) => {
    setOpenText((cur) => {
      const next = new Set(cur);
      if (next.has(seq)) next.delete(seq);
      else next.add(seq);
      return next;
    });
  };
  const showWhole = (key: string) => {
    setShowAll((cur) => {
      const next = new Set(cur);
      next.add(key);
      return next;
    });
  };

  const isEmpty = nodes.length === 0 && archive.length === 0;

  return (
    <div className="ctx-card p-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="ctx-head-ic" aria-hidden>
            <Layers size={12} />
          </span>
          <span className="truncate text-[12.5px] font-semibold text-fg">{t("contextview.browserTitle")}</span>
        </div>
        <div className="shrink-0 text-[9px] tabular-nums text-fg-faint">
          {t("contextview.tabActive", { n: nodes.length })} · ≈{fmtTokens(totalTokens)}
        </div>
      </div>
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t("contextview.browserSearch")}
        aria-label={t("contextview.browserSearch")}
        className="mt-1.5 h-6 w-full rounded-md border border-border-soft bg-bg-soft px-2 text-[10px] text-fg placeholder:text-fg-faint focus:border-accent focus:outline-none"
      />
      {/* v4.80 分类内排序（dsh size/name 同款；只影响展开后的行序，不改组行聚合） */}
      <div className="mt-1 flex items-center gap-1 text-[9.5px]">
        {([["time", "contextview.sortTime"], ["size", "contextview.sortSize"]] as const).map(([k, key]) => (
          <button
            key={k}
            aria-pressed={sort === k}
            className={`cursor-pointer rounded border-0 px-1.5 py-0.5 ${sort === k ? "bg-accent/15 text-accent" : "bg-transparent text-fg-faint hover:text-fg"}`}
            onClick={() => setSort(k)}
          >{t(key)}</button>
        ))}
      </div>
      {isEmpty ? (
        <div className="py-2 text-[10px] text-fg-faint">{t("contextview.noNodes")}</div>
      ) : (
        <div className="mt-1.5 flex max-h-72 flex-col gap-1 overflow-y-auto pr-0.5">
          {groups.map((g) => {
            const isOpen = expanded.has(g.key);
            const paged = !showAll.has(g.key) && g.count > PAGE_THRESHOLD;
            const listed = paged ? g.nodes.slice(0, PAGE_SIZE) : g.nodes;
            const share = totalTokens > 0 ? Math.round((g.tokens / totalTokens) * 100) : 0;
            return (
              <div key={g.key}>
                <button
                  type="button"
                  aria-expanded={isOpen}
                  onClick={() => toggleGroup(g.key)}
                  className="ctx-row flex w-full cursor-pointer items-center gap-1.5 border-0 px-1.5 py-1.5 text-left text-[10.5px]"
                >
                  <span className="h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[g.key] }} aria-hidden />
                  <span className="font-medium text-fg-dim">{t(g.labelKey)}</span>
                  <span
                    className="rounded-full px-1.5 text-[9px] tabular-nums"
                    style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text-secondary)" }}
                  >
                    {g.count} 项 {/* i18n 待主代理收口 */}
                  </span>
                  <span className="ml-auto shrink-0 font-mono text-[9.5px] tabular-nums text-fg-faint">
                    ≈{fmtTokens(g.tokens)} ({share}%)
                  </span>
                </button>
                {isOpen && (
                  <div className="ml-3.5 mt-0.5 flex flex-col gap-1 border-l border-border-soft/60 pl-2">
                    {g.count === 0 && <div className="py-1 text-[10px] text-fg-faint">{t("contextview.noCatNodes")}</div>}
                    {listed.map((n) => (
                      <NodeRow
                        key={n.seq}
                        node={n}
                        open={openText.has(n.seq)}
                        onToggle={() => toggleText(n.seq)}
                        detailState={details.get(n.seq)}
                        detailOpen={openDetails.has(n.seq)}
                        onToggleDetail={() => toggleDetail(n.seq)}
                      />
                    ))}
                    {paged && (
                      <button
                        type="button"
                        onClick={() => showWhole(g.key)}
                        className="mt-1 w-full cursor-pointer rounded-md border-0 bg-transparent py-1 text-[10px] text-accent hover:underline"
                      >
                        {t("contextview.showAll", { n: g.count })}
                      </button>
                    )}
                  </div>
                )}
              </div>
            );
          })}
          {archive.length > 0 && (() => {
            const isOpen = expanded.has(ARCHIVE_KEY);
            const paged = !showAll.has(ARCHIVE_KEY) && archiveShown.length > PAGE_THRESHOLD;
            const listed = paged ? archiveShown.slice(0, PAGE_SIZE) : archiveShown;
            return (
              <div className="mt-0.5 border-t border-border-soft/40 pt-0.5">
                <button
                  type="button"
                  aria-expanded={isOpen}
                  onClick={() => toggleGroup(ARCHIVE_KEY)}
                  className="ctx-row flex w-full cursor-pointer items-center gap-1.5 border-0 px-1.5 py-1.5 text-left text-[10.5px]"
                >
                  <span
                    className="h-2 w-2 shrink-0 rounded-sm"
                    style={{ background: "var(--md-sys-color-text-tertiary)" }}
                    aria-hidden
                  />
                  <span className="font-medium text-fg-dim">{t("contextview.tabArchive", { n: archiveShown.length })}</span>
                  <span
                    className="rounded-full px-1.5 text-[9px] tabular-nums"
                    style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text-secondary)" }}
                  >
                    {archiveShown.length} 项 {/* i18n 待主代理收口 */}
                  </span>
                  <span className="ml-auto shrink-0 font-mono text-[9.5px] tabular-nums text-fg-faint">≈{fmtTokens(archiveTokens)}</span>
                </button>
                {isOpen && (
                  <div className="ml-3.5 mt-0.5 flex flex-col gap-1 border-l border-border-soft/60 pl-2">
                    {archiveShown.length === 0 && <div className="py-1 text-[10px] text-fg-faint">{t("contextview.noCatNodes")}</div>}
                    {listed.map((n) => (
                      <NodeRow
                        key={n.seq}
                        node={n}
                        open={openText.has(n.seq)}
                        onToggle={() => toggleText(n.seq)}
                        detailState={details.get(n.seq)}
                        detailOpen={openDetails.has(n.seq)}
                        onToggleDetail={() => toggleDetail(n.seq)}
                      />
                    ))}
                    {paged && (
                      <button
                        type="button"
                        onClick={() => showWhole(ARCHIVE_KEY)}
                        className="mt-1 w-full cursor-pointer rounded-md border-0 bg-transparent py-1 text-[10px] text-accent hover:underline"
                      >
                        {t("contextview.showAll", { n: archiveShown.length })}
                      </button>
                    )}
                  </div>
                )}
              </div>
            );
          })()}
        </div>
      )}
      <div className="mt-1 text-[9px] text-fg-faint">{t("contextview.browserLegend")}</div>
    </div>
  );
}

// ─── 文件活动（按文件聚合树 + chips 过滤 + 三排序） ──────────────
type FileChip = "all" | "read" | "write" | "search" | "image";
type FileSort = "count" | "recent" | "path";

// dsh 的「搜索/图片」在本仓 FileActivity.action（read/write/move/dir）中无对应值：
// 按工具名/扩展名启发式推断；chips 是过滤器，与读/写允许重叠（注释待主代理收口语义）。
const SEARCH_TOOL_RE = /grep|glob|search|find|ls|list/i;
const IMAGE_PATH_RE = /\.(png|jpe?g|gif|webp|bmp|svg)$/i;

function chipMatch(f: FileActivity, k: FileChip): boolean {
  switch (k) {
    case "all":
      return true;
    case "read":
      return f.action === "read";
    case "write":
      return f.action === "write";
    case "search":
      return f.action !== "dir" && SEARCH_TOOL_RE.test(f.tool);
    case "image":
      return IMAGE_PATH_RE.test(f.path);
  }
}

// 动作徽标（搬运自 ContextView.tsx FILE_ACTION_META，未导出故本地重声明）。
const FILE_ACTION_META: Record<FileActivity["action"], { labelKey: DictKey; cls: string }> = {
  read: { labelKey: "contextview.actRead", cls: "bg-cyan-500/15 text-cyan-400" },
  write: { labelKey: "contextview.actWrite", cls: "bg-amber-500/15 text-amber-400" },
  move: { labelKey: "contextview.actMove", cls: "bg-purple-500/15 text-purple-400" },
  dir: { labelKey: "contextview.actDir", cls: "bg-slate-500/15 text-slate-400" },
};

interface FileAgg {
  path: string;
  read: number;
  write: number;
  move: number;
  dir: number;
  total: number;
  latest: number;
  added: number; // v4.81 行级增量合计（写类操作参数确定性提取）
  removed: number;
}

function aggregate(files: FileActivity[]): FileAgg[] {
  const byPath = new Map<string, FileAgg>();
  for (const f of files) {
    let a = byPath.get(f.path);
    if (!a) {
      a = { path: f.path, read: 0, write: 0, move: 0, dir: 0, total: 0, latest: 0, added: 0, removed: 0 };
      byPath.set(f.path, a);
    }
    a[f.action] += 1;
    a.total += 1;
    a.added += f.added ?? 0;
    a.removed += f.removed ?? 0;
    if (f.ts > a.latest) a.latest = f.ts;
  }
  return [...byPath.values()];
}

const SORT_PILLS: { key: FileSort; labelKey: DictKey }[] = [
  { key: "count", labelKey: "contextview.filesSortCalls" },
  { key: "recent", labelKey: "contextview.filesSortLatest" },
  { key: "path", labelKey: "contextview.filesSortPath" },
];

// 排序稳定性：并列时依次以最新时间/路径决胜，保证任意输入下顺序确定。
const SORT_CMP: Record<FileSort, (a: FileAgg, b: FileAgg) => number> = {
  count: (a, b) => b.total - a.total || b.latest - a.latest || a.path.localeCompare(b.path),
  recent: (a, b) => b.latest - a.latest || a.path.localeCompare(b.path),
  path: (a, b) => a.path.localeCompare(b.path),
};

export function FileActivityTree({ files, onOpenFile }: { files: FileActivity[]; onOpenFile?: (path: string) => void }) {
  const t = useT();
  // 预览打开二选一：默认内部走 usePreviewStore（与现有 FileActivityCard 同款链路），
  // onOpenFile 作为可选注入覆盖（契约签名保持兼容：传或不传均可）。
  const openPreview = usePreviewStore((s) => s.openFilePreview);
  const open = onOpenFile ?? openPreview;
  const [chip, setChip] = useState<FileChip>("all");
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<FileSort>("count");
  const { details, open: openDetails, toggle: toggleDetail } = useNodeDetails();
  // v4.81 操作日志展开（dsh「展开完整操作日志」同款）：按路径展开逐次操作行。
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(() => new Set());

  const q = query.trim().toLowerCase();

  const counts = useMemo(() => {
    const c: Record<FileChip, number> = { all: files.length, read: 0, write: 0, search: 0, image: 0 };
    for (const f of files) {
      if (chipMatch(f, "read")) c.read += 1;
      if (chipMatch(f, "write")) c.write += 1;
      if (chipMatch(f, "search")) c.search += 1;
      if (chipMatch(f, "image")) c.image += 1;
    }
    return c;
  }, [files]);

  const { rows, opIndex } = useMemo(() => {
    const filtered = files.filter((f) => chipMatch(f, chip) && (q === "" || f.path.toLowerCase().includes(q)));
    const ops = new Map<string, FileActivity[]>();
    for (const f of filtered) {
      const list = ops.get(f.path);
      if (list) list.push(f);
      else ops.set(f.path, [f]);
    }
    return { rows: aggregate(filtered).sort(SORT_CMP[sort]), opIndex: ops };
  }, [files, chip, q, sort]);

  const togglePath = (path: string) => {
    setExpandedPaths((cur) => {
      const next = new Set(cur);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const chips: { key: FileChip; labelKey: DictKey }[] = [
    { key: "all", labelKey: "contextview.filesAll" },
    { key: "read", labelKey: "contextview.filesRead" },
    { key: "write", labelKey: "contextview.filesWrite" },
    { key: "search", labelKey: "contextview.filesSearch" },
    { key: "image", labelKey: "contextview.filesImages" },
  ];

  return (
    <div className="ctx-card p-3">
      <div className="flex items-center gap-1.5">
        <span className="ctx-head-ic" aria-hidden>
          <FolderTree size={12} />
        </span>
        <span className="text-[12.5px] font-semibold text-fg">{t("contextview.filesTitle")}</span>
      </div>
      <div className="mt-1 flex flex-wrap items-center gap-1 text-[10px]">
        {chips.map((c) => (
          <button
            key={c.key}
            type="button"
            aria-pressed={chip === c.key}
            className={`cursor-pointer rounded border-0 px-1.5 py-0.5 ${chip === c.key ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
            onClick={() => setChip(c.key)}
          >
            {t(c.labelKey)}
            <span className="ml-1 tabular-nums">{counts[c.key]}</span>
          </button>
        ))}
      </div>
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t("contextview.filesFilter")}
        aria-label="按路径过滤" // i18n 待主代理收口
        className="mt-1.5 h-6 w-full rounded-md border border-border-soft bg-bg-soft px-2 text-[10px] text-fg placeholder:text-fg-faint focus:border-accent focus:outline-none"
      />
      <div className="mt-1 flex items-center justify-between gap-2">
        <span className="shrink-0 text-[9.5px] tabular-nums text-fg-faint">
          {rows.length} 个文件 {/* i18n 待主代理收口（dsh 的 +X−Y 增删行数无数据，省略） */}
        </span>
        <span className="flex items-center gap-1 text-[10px]">
          {SORT_PILLS.map((p) => (
            <button
              key={p.key}
              type="button"
              aria-pressed={sort === p.key}
              className={`cursor-pointer rounded border-0 px-1.5 py-0.5 ${sort === p.key ? "bg-accent/15 text-accent" : "text-fg-dim hover:text-fg"}`}
              onClick={() => setSort(p.key)}
            >
              {t(p.labelKey)}
            </button>
          ))}
        </span>
      </div>
      <div className="mt-1 flex max-h-64 flex-col gap-1 overflow-y-auto pr-0.5">
        {files.length === 0 && <div className="py-2 text-[10px] text-fg-faint">{t("contextview.noFiles")}</div>}
        {files.length > 0 && rows.length === 0 && (
          <div className="py-2 text-[10px] text-fg-faint">{t("contextview.filesNoMatch")}</div>
        )}
        {rows.map((a) => {
          // 纯目录行（无读/写/移）不可预览，与现有 FileActivityCard 的 dir 行为一致。
          const clickable = a.read + a.write + a.move > 0;
          const ops = opIndex.get(a.path) ?? [];
          const isOpen = expandedPaths.has(a.path);
          return (
            <div key={a.path}>
              <div className="ctx-row flex w-full items-center gap-1.5 px-2 py-1.5 text-[10px]">
                <button
                  type="button"
                  data-file={a.path}
                  disabled={!clickable}
                  onClick={() => clickable && open(a.path)}
                  title={clickable ? t("contextview.previewTitle", { path: a.path }) : undefined}
                  className={`flex min-w-0 flex-1 items-center gap-1.5 border-0 bg-transparent p-0 text-left text-[10px] ${
                    clickable ? "cursor-pointer" : "cursor-default"
                  }`}
                >
                  <FileTypeIcon name={a.path} size={12} />
                  <span className="truncate font-mono text-fg-dim" title={a.path}>{a.path}</span>
                  {/* v4.81 行级增量徽标（dsh ±added/−removed 同款） */}
                  {(a.added > 0 || a.removed > 0) && (
                    <span className="shrink-0 font-mono text-[9px] tabular-nums">
                      {a.added > 0 && <span style={{ color: "#22c55e" }}>+{a.added}</span>} {/* hex-exempt 增量语义色（绿=增） */}
                      {a.added > 0 && a.removed > 0 && <span className="text-fg-faint">/</span>}
                      {a.removed > 0 && <span style={{ color: "#ef4444" }}>−{a.removed}</span>} {/* hex-exempt 增量语义色（红=减） */}
                    </span>
                  )}
                  <span className="ml-auto flex shrink-0 items-center gap-1">
                    {a.read > 0 && (
                      <span className={`rounded px-1 text-[9px] tabular-nums ${FILE_ACTION_META.read.cls}`}>
                        {`${t(FILE_ACTION_META.read.labelKey)} ${a.read}`}
                      </span>
                    )}
                    {a.write > 0 && (
                      <span className={`rounded px-1 text-[9px] tabular-nums ${FILE_ACTION_META.write.cls}`}>
                        {`${t(FILE_ACTION_META.write.labelKey)} ${a.write}`}
                      </span>
                    )}
                    {a.move > 0 && (
                      <span className={`rounded px-1 text-[9px] tabular-nums ${FILE_ACTION_META.move.cls}`}>
                        {`${t(FILE_ACTION_META.move.labelKey)} ${a.move}`}
                      </span>
                    )}
                    {a.dir > 0 && (
                      <span className={`rounded px-1 text-[9px] tabular-nums ${FILE_ACTION_META.dir.cls}`}>
                        {`${t(FILE_ACTION_META.dir.labelKey)} ${a.dir}`}
                      </span>
                    )}
                    <span className="tabular-nums font-mono text-fg-faint">{new Date(a.latest * 1000).toLocaleTimeString()}</span>
                  </span>
                </button>
                {ops.length > 0 && (
                  <button
                    type="button"
                    aria-expanded={isOpen}
                    data-testid="file-ops-toggle"
                    className="shrink-0 cursor-pointer rounded border-0 bg-transparent px-1 text-[9px] text-fg-faint hover:text-fg"
                    onClick={() => togglePath(a.path)}
                  >
                    {t("contextview.filesOps", { n: ops.length })}
                  </button>
                )}
              </div>
              {isOpen &&
                ops.map((op) => (
                  <FileOpRow
                    key={op.seq}
                    op={op}
                    detailState={details.get(op.seq)}
                    detailOpen={openDetails.has(op.seq)}
                    onToggleDetail={() => toggleDetail(op.seq)}
                  />
                ))}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// v4.81 单次文件操作行（操作日志展开层）：工具 chip + 该次 ±行/命中数 +
// 完整调用详情懒加载（复用浏览器详情面板，跳转到对应工具结果）。
function FileOpRow({
  op,
  detailState,
  detailOpen,
  onToggleDetail,
}: {
  op: FileActivity;
  detailState?: NodeDetailState;
  detailOpen: boolean;
  onToggleDetail: () => void;
}) {
  const t = useT();
  const d = detailState?.s === "ok" ? detailState.d : null;
  return (
    <div className="ml-4 border-l border-border-soft/60 pl-2">
      <div className="ctx-row flex flex-wrap items-center gap-1.5 px-1 py-1 text-[9.5px] text-fg-faint">
        <span
          className="rounded px-1 font-mono"
          style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text-secondary)" }}
        >
          {op.tool}
        </span>
        {((op.added ?? 0) > 0 || (op.removed ?? 0) > 0) && (
          <span className="font-mono tabular-nums">
            {(op.added ?? 0) > 0 && <span style={{ color: "#22c55e" }}>+{op.added}</span>} {/* hex-exempt 增量语义色 */}
            {(op.added ?? 0) > 0 && (op.removed ?? 0) > 0 && <span> </span>}
            {(op.removed ?? 0) > 0 && <span style={{ color: "#ef4444" }}>−{op.removed}</span>} {/* hex-exempt 增量语义色 */}
          </span>
        )}
        {(op.hits ?? 0) > 0 && <span className="font-mono tabular-nums">≈{op.hits} {t("contextview.filesHits")}</span>}
        <span className="font-mono tabular-nums">{new Date(op.ts * 1000).toLocaleTimeString()}</span>
        <button
          type="button"
          data-testid="file-op-detail-btn"
          className={`ml-auto cursor-pointer border-0 bg-transparent hover:underline ${detailOpen ? "text-fg-dim" : "text-accent"}`}
          onClick={onToggleDetail}
        >
          {detailOpen ? t("contextview.detailCollapse") : t("contextview.detailBtn")}
        </button>
      </div>
      {detailOpen && detailState?.s === "loading" && (
        <div className="px-1 py-0.5 text-[9.5px] text-fg-faint">{t("contextview.detailLoading")}</div>
      )}
      {detailOpen && detailState?.s === "error" && (
        <div className="px-1 py-0.5 text-[9.5px]" style={{ color: "var(--md-sys-color-destructive)" }}>{t("contextview.detailFail")}</div>
      )}
      {detailOpen && d && <NodeDetailPanel d={d} />}
    </div>
  );
}
