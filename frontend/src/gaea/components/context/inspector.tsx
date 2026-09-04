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
import { useT } from "../../lib/i18n";
import { FileTypeIcon } from "../../lib/fileIcon";
import { fmtTokens } from "../../lib/stats";
import { usePreviewStore } from "../../lib/store";
import type { DictKey } from "../../locales/en";
import type { ContextSurfaceNode, FileActivity } from "../../lib/types";

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

// 节点行（搬运自 ContextView.tsx NodeRow：>56 字符可展开全文，归档节点带「已压缩」徽标）。
const ROW_LABELS: Record<ContextSurfaceNode["cat"], DictKey> = {
  system: "contextview.browseSystem",
  tools: "contextview.browseTools",
  user: "contextview.browseUser",
  inject: "contextview.browseInject",
  assistant: "contextview.browseAssistant",
  tool: "contextview.browseTool",
};

function NodeRow({ node, open, onToggle }: { node: ContextSurfaceNode; open: boolean; onToggle: () => void }) {
  const t = useT();
  const text = node.text || t("contextview.noPreview");
  const truncated = text.length > 56;
  const shown = open || !truncated ? text : `${text.slice(0, 56)}…`;
  return (
    <div className="flex items-start gap-1.5 border-b border-border-soft/40 py-1 text-[10px] last:border-0">
      <span className="mt-0.5 h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[node.cat] }} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5 text-fg-faint">
          <span className="font-medium" style={{ color: CAT_COLORS[node.cat] }}>{t(ROW_LABELS[node.cat])}</span>
          <span className="tabular-nums font-mono">≈{fmtTokens(node.tokens)}</span>
          {node.gone != null && <span className="text-warning">{t("contextview.compacted")}</span>}
          {truncated && (
            <button
              className="ml-auto cursor-pointer border-0 bg-transparent text-accent hover:underline"
              onClick={onToggle}
            >{open ? t("common.collapse") : t("common.expand")}</button>
          )}
        </div>
        <div className="mt-0.5 whitespace-pre-wrap break-all font-mono text-fg-dim">{shown}</div>
      </div>
    </div>
  );
}

// ─── 上下文浏览器（分类折叠组 + 行内搜索 + 归档折叠组） ──────────
export function ContextBrowserTree({ nodes, archive }: { nodes: ContextSurfaceNode[]; archive: ContextSurfaceNode[] }) {
  const t = useT();
  const [query, setQuery] = useState("");
  // 默认折叠态：六分类与归档全部收起（dsh 同款）。
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());
  const [openText, setOpenText] = useState<Set<number>>(() => new Set());
  const [showAll, setShowAll] = useState<Set<string>>(() => new Set());

  const q = query.trim().toLowerCase();

  // 组行右侧 ≈tokens 用未过滤的分类合计（占比分母=活跃节点总 tokens，数值稳定）；
  // 「N 项」徽标随搜索过滤（反映展开后实际列出的行数）。
  const groups = useMemo(
    () =>
      GROUPS.map((g) => {
        const all = nodes.filter((n) => n.cat === g.key);
        const filtered = q === "" ? all : all.filter((n) => (n.text ?? "").toLowerCase().includes(q));
        const tokens = all.reduce((s, n) => s + n.tokens, 0);
        return { ...g, all: all.length, count: filtered.length, tokens, nodes: filtered };
      }),
    [nodes, q],
  );
  const archiveShown = useMemo(
    () => (q === "" ? archive : archive.filter((n) => (n.text ?? "").toLowerCase().includes(q))),
    [archive, q],
  );
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
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">{t("contextview.browserTitle")}</div>
        <div className="text-[9px] tabular-nums text-fg-faint">
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
      {isEmpty ? (
        <div className="py-2 text-[10px] text-fg-faint">{t("contextview.noNodes")}</div>
      ) : (
        <div className="mt-1.5 max-h-72 overflow-y-auto">
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
                  className="flex w-full cursor-pointer items-center gap-1.5 rounded-md border-0 bg-transparent px-0.5 py-1 text-left text-[10.5px] transition-colors hover:bg-bg-soft/70"
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
                  <div className="ml-3.5 border-l border-border-soft/60 pl-2">
                    {g.count === 0 && <div className="py-1 text-[10px] text-fg-faint">{t("contextview.noCatNodes")}</div>}
                    {listed.map((n) => (
                      <NodeRow key={n.seq} node={n} open={openText.has(n.seq)} onToggle={() => toggleText(n.seq)} />
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
                  className="flex w-full cursor-pointer items-center gap-1.5 rounded-md border-0 bg-transparent px-0.5 py-1 text-left text-[10.5px] transition-colors hover:bg-bg-soft/70"
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
                  <div className="ml-3.5 border-l border-border-soft/60 pl-2">
                    {archiveShown.length === 0 && <div className="py-1 text-[10px] text-fg-faint">{t("contextview.noCatNodes")}</div>}
                    {listed.map((n) => (
                      <NodeRow key={n.seq} node={n} open={openText.has(n.seq)} onToggle={() => toggleText(n.seq)} />
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
}

function aggregate(files: FileActivity[]): FileAgg[] {
  const byPath = new Map<string, FileAgg>();
  for (const f of files) {
    let a = byPath.get(f.path);
    if (!a) {
      a = { path: f.path, read: 0, write: 0, move: 0, dir: 0, total: 0, latest: 0 };
      byPath.set(f.path, a);
    }
    a[f.action] += 1;
    a.total += 1;
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

  const rows = useMemo(() => {
    const filtered = files.filter((f) => chipMatch(f, chip) && (q === "" || f.path.toLowerCase().includes(q)));
    return aggregate(filtered).sort(SORT_CMP[sort]);
  }, [files, chip, q, sort]);

  const chips: { key: FileChip; labelKey: DictKey }[] = [
    { key: "all", labelKey: "contextview.filesAll" },
    { key: "read", labelKey: "contextview.filesRead" },
    { key: "write", labelKey: "contextview.filesWrite" },
    { key: "search", labelKey: "contextview.filesSearch" },
    { key: "image", labelKey: "contextview.filesImages" },
  ];

  return (
    <div className="rounded-lg border border-border-soft bg-bg p-3">
      <div className="flex items-center justify-between">
        <div className="text-[11px] font-medium text-fg">{t("contextview.filesTitle")}</div>
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
      <div className="mt-1 max-h-64 overflow-y-auto">
        {files.length === 0 && <div className="py-2 text-[10px] text-fg-faint">{t("contextview.noFiles")}</div>}
        {files.length > 0 && rows.length === 0 && (
          <div className="py-2 text-[10px] text-fg-faint">{t("contextview.filesNoMatch")}</div>
        )}
        {rows.map((a) => {
          // 纯目录行（无读/写/移）不可预览，与现有 FileActivityCard 的 dir 行为一致。
          const clickable = a.read + a.write + a.move > 0;
          return (
            <button
              key={a.path}
              type="button"
              data-file={a.path}
              disabled={!clickable}
              onClick={() => clickable && open(a.path)}
              title={clickable ? t("contextview.previewTitle", { path: a.path }) : undefined}
              className={`flex w-full items-center gap-1.5 border-b border-border-soft/40 py-1 text-left text-[10px] last:border-0 ${
                clickable ? "cursor-pointer transition-colors hover:bg-bg-soft/70" : "cursor-default"
              }`}
            >
              <FileTypeIcon name={a.path} size={12} />
              <span className="truncate font-mono text-fg-dim" title={a.path}>{a.path}</span>
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
          );
        })}
      </div>
    </div>
  );
}
