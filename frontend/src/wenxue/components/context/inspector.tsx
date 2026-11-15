// 上下文 inspector 两卡（对齐 dsh-context 的「上下文浏览器」与「文件活动」）。
// 独立前端线私有文件；主代理的 inspector 容器负责接线（ContextView.tsx 不动）。
// NodeRow / 预览打开逻辑自 ContextView.tsx 搬运：
// - ContextBrowserTree：六分类折叠组（色点+分类名+N项徽标+≈tokens/占比）→ 展开
//   节点列表（长文本可展开，交互同现有 NodeRow）；行内搜索框过滤节点文本；
//   末尾「归档 N」折叠组（行带「已压缩」徽标）。默认全部收起。
// - FileActivityTree：过滤 chips（全部/读取/写入/搜索/图片）+ 路径过滤 + 汇总行
//   + 排序三胶囊（按次数/按最新/按路径）+ 按文件聚合树行（点击打开预览）。
import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
// 六分类语义色与键名单源在 context/cats.ts（i18n 收口批起 CAT_COLORS 定义也在此）。
import { CAT_COLORS } from "./cats";
import { MemoMarkdown } from "../MemoMarkdown";
import { app } from "../../lib/bridge";
import "./context-view.css";
import { FolderTree, Layers } from "../../icons";
import { useT } from "../../lib/i18n";
import { FileTypeIcon } from "../../lib/fileIcon";
import { fmtTokens } from "../../lib/stats";
import { loadContextPrefs, saveContextPref, type CtxBrowserSort, type CtxFileSort } from "../../lib/contextPrefs";
import { usePreviewStore } from "../../lib/store";
import { CONTEXT_CATS, CONTEXT_CAT_BROWSE } from "./cats";
import type { DictKey } from "../../locales/en";
import type {
  ContextNodeDetailView,
  ContextNodeAttachment,
  ContextNodeImage,
  ContextRequestDelta,
  ContextRequestRecord,
  ContextSurfaceNode,
  FileActivity,
} from "../../lib/types";
import { getCachedAttachmentDataURL } from '../../../components/characterlib/usePortraitUrl';

// 分类折叠组定义：单一事实源 context/cats.ts（FE1-02，此前本地重声明漂移风险）。
const GROUPS = CONTEXT_CATS.map((c) => ({ key: c.key, labelKey: c.labelKey, rowKey: c.browseKey }));

// 分类内节点分页（dsh 同款防卡顿思路）：>200 项展开时先渲染前 100 行 + 「显示全部」。
const PAGE_THRESHOLD = 200;
const PAGE_SIZE = 100;
const ARCHIVE_KEY = "archive"; // 折叠态 key（与分类 key 不冲突的哨兵值）

// 节点详情懒加载状态机：undefined=未加载、loading、ok（缓存）、error。
type NodeDetailState = { s: "loading" } | { s: "ok"; d: ContextNodeDetailView } | { s: "error" };

const ROW_LABELS: Record<ContextSurfaceNode["cat"], DictKey> = CONTEXT_CAT_BROWSE;

type BrowserMapItem = {
  key: ContextSurfaceNode["cat"];
  tokens: number;
};

type BrowserMapRect = BrowserMapItem & {
  x: number;
  y: number;
  width: number;
  height: number;
};

// treemapRects 是浏览器 Map 视图的无依赖二维矩形切分：每层按 token 总量在
// 长边二分，递归到单分类。纯函数保证同一快照布局稳定。
function treemapRects(items: BrowserMapItem[]): BrowserMapRect[] {
  const sorted = [...items].filter((item) => item.tokens > 0).sort((a, b) => b.tokens - a.tokens);
  const out: BrowserMapRect[] = [];
  const split = (group: BrowserMapItem[], x: number, y: number, width: number, height: number): void => {
    if (group.length === 0) return;
    if (group.length === 1) {
      out.push({ ...group[0], x, y, width, height });
      return;
    }
    const total = group.reduce((sum, item) => sum + item.tokens, 0);
    let leftTokens = 0;
    let splitAt = 1;
    for (let i = 0; i < group.length - 1; i++) {
      leftTokens += group[i].tokens;
      splitAt = i + 1;
      if (leftTokens >= total / 2) break;
    }
    const left = group.slice(0, splitAt);
    const right = group.slice(splitAt);
    const ratio = Math.max(0.08, Math.min(0.92, leftTokens / Math.max(total, 1)));
    if (width >= height) {
      split(left, x, y, width * ratio, height);
      split(right, x + width * ratio, y, width * (1 - ratio), height);
    } else {
      split(left, x, y, width, height * ratio);
      split(right, x, y + height * ratio, width, height * (1 - ratio));
    }
  };
  split(sorted, 0, 0, 100, 64);
  return out;
}

export type BrowserCatFocus = {
  cat: ContextSurfaceNode["cat"] | typeof ARCHIVE_KEY;
  tick: number;
  query?: string;
};

function nodeMatchesQuery(node: ContextSurfaceNode, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (q === "") return true;
  return [node.text, node.tool, node.name].some((value) =>
    (value ?? "").toLowerCase().includes(q),
  );
}

// v4.81 节点详情懒加载 hook（浏览器与文件活动共用）：按 seq 懒加载 +
// 缓存（Map 状态机 loading/ok/error）+ 开合集合。
function useNodeDetails(sessionPath?: string, fetchDetail?: (seq: number) => Promise<ContextNodeDetailView>) {
  const [details, setDetails] = useState<Map<number, NodeDetailState>>(() => new Map());
  const [open, setOpen] = useState<Set<number>>(() => new Set());
  useEffect(() => {
    setDetails(new Map());
    setOpen(new Set());
  }, [sessionPath]);
  const toggle = (seq: number) => {
    setOpen((cur) => {
      const next = new Set(cur);
      if (next.has(seq)) next.delete(seq);
      else next.add(seq);
      return next;
    });
    if (!details.has(seq)) {
      setDetails((cur) => new Map(cur).set(seq, { s: "loading" }));
      // 自定义源优先（v4.412 原罪复用：SinContextNodeDetail 按故事 id 回读）；
      // 缺省按会话日志回读（GaeaContextNodeDetail）。
      const p = fetchDetail
        ? fetchDetail(seq)
        : app.ContextNodeDetail(seq, sessionPath ?? "");
      p.then((d) => setDetails((cur) => new Map(cur).set(seq, { s: "ok", d })))
        .catch(() => setDetails((cur) => new Map(cur).set(seq, { s: "error" })));
    }
  };
  return { details, open, toggle };
}

// v4.81 节点行（搬运自 ContextView.tsx NodeRow：>56 字符可展开全文，归档节点带「已压缩」徽标）。
// v4.80 深读：tool 行加「来源」chip（工具名）与 error 语义点；tool/user/assistant
// 行可懒加载「完整调用」详情（GaeaContextNodeDetail 按 seq 回读当前会话日志），
// 详情面板带 OK/error 状态、行数、截断提示与 原文/渲染 切换（默认原文）。
const DETAILABLE = new Set<ContextSurfaceNode["cat"]>(["system", "tools", "tool", "skill", "user", "assistant"]);

// v4.81 节点/文件操作共用的完整调用详情面板（原文/渲染切换在面板内部自持；
// 默认原文防任意工具输出噪声；渲染走 MemoMarkdown）。
// v4.83 图片缩略卡（2.5b 后半，dsh 图片 payload 缩略卡同款交互）：缩略图 +
// 「原始尺寸→标准档缩放尺寸」与 token 估算成对显示；token 按官方 patch 口径
// （⌈w/28⌉×⌈h/28⌉，先档位缩放再封顶）由 Go 侧估算，前端只渲染不重算。
// 文件缺失/不可解码诚实降级（灰态/尺寸未知），绝不猜数。
function NodeImageCard({ img }: { img: ContextNodeImage }) {
  const t = useT();
  const openPreview = usePreviewStore((s) => s.openFilePreview);
  const [dataUrl, setDataUrl] = useState<string | null>(null);
  useEffect(() => {
    let live = true;
    setDataUrl(null);
    if (img.exists) {
      getCachedAttachmentDataURL(img.path)
        .then((u: string) => {
          if (live) setDataUrl(u);
        })
        .catch(() => {});
    }
    return () => {
      live = false;
    };
  }, [img.path, img.exists]);
  const known = (img.width ?? 0) > 0;
  return (
    <div
      data-testid="ctx-img-card"
      data-attachment-kind="image"
      className={`flex w-56 shrink-0 gap-1.5 rounded-md border border-border-soft p-1 ${img.exists ? "" : "opacity-55"}`}
    >
      <button
        type="button"
        disabled={!img.exists}
        title={img.exists ? img.path : undefined}
        className={`flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded bg-bg-soft ${img.exists ? "cursor-pointer" : "cursor-default"}`}
        onClick={() => openPreview(img.path)}
      >
        {dataUrl ? (
          <img src={dataUrl} alt={img.ref} className="h-full w-full object-cover" />
        ) : (
          <FileTypeIcon name={img.ref} size={18} />
        )}
      </button>
      <div className="min-w-0 flex-1 py-0.5">
        <div className="truncate font-mono text-[9.5px] text-fg-dim" title={img.path}>{img.ref}</div>
        {img.exists ? (
          <>
            <div className="mt-0.5 font-mono text-[9px] tabular-nums text-fg-faint">
              {known
                ? t("contextview.imgDims", { w: img.width!, h: img.height!, sw: img.scaledW ?? img.width!, sh: img.scaledH ?? img.height! })
                : t("contextview.imgUnknownDims")}
            </div>
            {known && (
              <div
                className="mt-0.5 truncate font-mono text-[9px] tabular-nums text-warning"
                title={t("contextview.imgEstTip", { high: img.highTokens ?? 0 })}
              >
                {t("contextview.imgTokens", { tokens: img.stdTokens ?? 0 })}
              </div>
            )}
          </>
        ) : (
          <div className="mt-0.5 text-[9px]" style={{ color: "var(--md-sys-color-destructive)" }}>
            {t("contextview.imgMissing")}
          </div>
        )}
      </div>
    </div>
  );
}

function formatAttachmentSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function NodeAttachmentCard({ attachment }: { attachment: ContextNodeAttachment }) {
  const t = useT();
  const openPreview = usePreviewStore((state) => state.openFilePreview);
  const ext = attachment.name.split(".").pop()?.toUpperCase() || t("contextview.attachOther");
  return (
    <button
      type="button"
      data-testid="ctx-file-attachment-card"
      data-attachment-kind="file"
      disabled={!attachment.exists}
      title={attachment.path}
      className={`ctx-file-attachment${attachment.exists ? "" : " is-missing"}`}
      onClick={() => {
        if (attachment.exists) openPreview(attachment.path);
      }}
    >
      <span className="ctx-file-attachment-icon" aria-hidden>
        <FileTypeIcon name={attachment.name} size={17} />
      </span>
      <span className="min-w-0 flex-1 text-left">
        <span className="block truncate text-[10px] font-medium text-fg">{attachment.name}</span>
        <span className="mt-0.5 flex items-center gap-1.5 font-mono text-[8.5px] text-fg-faint">
          <b className="font-semibold text-fg-dim">{ext}</b>
          {attachment.exists
            ? attachment.size != null && <span>{formatAttachmentSize(attachment.size)}</span>
            : <span className="text-warning">{t("contextview.attachmentMissing")}</span>}
        </span>
      </span>
    </button>
  );
}

export function NodeDetailPanel({ d, headerFocus }: {
  d: ContextNodeDetailView;
  headerFocus?: ContextSurfaceNode["cat"];
}) {
  const t = useT();
  const [rendered, setRendered] = useState(false);
  const body = d.kind === "tool_result" ? d.output ?? "" : d.text ?? "";
  const orderedAttachments = (() => {
    const positionOf = (ref: string, name?: string, path?: string) => {
      const candidates = [ref, name, path]
        .filter((value): value is string => Boolean(value))
        .map((value) => body.indexOf(value))
        .filter((index) => index >= 0);
      return candidates.length > 0 ? Math.min(...candidates) : Number.MAX_SAFE_INTEGER;
    };
    const images = (d.images ?? []).map((img, order) => ({
      kind: "image" as const,
      order,
      position: positionOf(img.ref, undefined, img.path),
      img,
    }));
    const files = (d.attachments ?? []).map((attachment, order) => ({
      kind: "file" as const,
      order,
      position: positionOf(attachment.ref, attachment.name, attachment.path),
      attachment,
    }));
    return [...images, ...files].sort((a, b) => a.position - b.position || a.order - b.order);
  })();
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
      {d.kind === "header" && headerFocus !== "system" && (d.tools?.length ?? 0) > 0 && (
        <div className="mt-1" data-testid="ctx-header-tools">
          <div className="text-[9px] text-fg-faint">{t("contextview.headerToolSchemas", { n: d.tools!.length })}</div>
          <div className="mt-0.5 space-y-1">
            {d.tools!.map((tool) => (
              <details key={tool.name} className="rounded border border-border-soft bg-bg px-1.5 py-1">
                <summary className="flex cursor-pointer list-none items-center gap-1.5 font-mono text-[9.5px] text-fg-dim">
                  <span className="font-medium text-fg">{tool.name}</span>
                  {tool.source && (
                    <span className="ctx-tool-source" title={t("contextview.headerToolSourceTip")}>
                      {tool.source}
                    </span>
                  )}
                  {tool.tokens != null && <span className="ml-auto text-fg-faint">≈{fmtTokens(tool.tokens)}</span>}
                </summary>
                {tool.schema ? (
                  <pre className="mt-1 max-h-56 overflow-auto whitespace-pre-wrap break-all font-mono text-[9px] text-fg-faint">{tool.schema}</pre>
                ) : (
                  <div className="mt-1 text-[9px] text-fg-faint">{t("contextview.headerNoSchema")}</div>
                )}
              </details>
            ))}
          </div>
        </div>
      )}
      {/* 上游附件语义：image/file block 按消息中的出现顺序混排为同一组卡片。 */}
      {orderedAttachments.length > 0 && (
        <div data-testid="ctx-node-attachments" className="mt-1">
          <div className="text-[9px] text-fg-faint">{t("contextview.attachmentsSection", { n: orderedAttachments.length })}</div>
          <div className="mt-0.5 flex flex-wrap gap-1.5">
            {orderedAttachments.map((entry) => (
              entry.kind === "image"
                ? <NodeImageCard key={`${entry.img.ref}|${entry.img.path}`} img={entry.img} />
                : <NodeAttachmentCard key={`${entry.attachment.ref}|${entry.attachment.path}`} attachment={entry.attachment} />
            ))}
          </div>
        </div>
      )}
      {(d.kind !== "header" || headerFocus !== "tools") && (
        <div className="mt-1 max-h-[26rem] overflow-y-auto whitespace-pre-wrap break-all font-mono text-[10px] text-fg-dim">
          {rendered ? <MemoMarkdown text={body} streaming={false} /> : body || t("contextview.noPreview")}
        </div>
      )}
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
  focused,
}: {
  node: ContextSurfaceNode;
  open: boolean;
  onToggle: () => void;
  detailState?: NodeDetailState;
  detailOpen: boolean;
  onToggleDetail: () => void;
  focused?: boolean;
}) {
  const t = useT();
  const text = node.text || t("contextview.noPreview");
  const truncated = text.length > 56;
  const shown = open || !truncated ? text : `${text.slice(0, 56)}…`;
  const detailable = DETAILABLE.has(node.cat);
  const d = detailState?.s === "ok" ? detailState.d : null;
  return (
    <div
      data-node-seq={node.seq}
      data-testid={`ctx-browser-node-${node.seq}`}
      className={`ctx-row flex items-start gap-1.5 px-2 py-1.5 text-[10px] ${
        focused ? "rounded-md bg-accent/10 outline outline-1 -outline-offset-1 outline-accent/50" : ""
      }`}
    >
      <span className="mt-0.5 h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[node.cat] }} />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-1.5 text-fg-faint">
          <span className="font-medium" style={{ color: CAT_COLORS[node.cat] }}>{t(ROW_LABELS[node.cat])}</span>
          <span className="tabular-nums font-mono">≈{fmtTokens(node.tokens)}</span>
          {/* v4.80 来源 chip（工具名）+ error 语义点（dsh 工具行 OK/error 同款语义） */}
          {(node.cat === "tool" || node.cat === "skill") && node.tool && (
            <span
              className="max-w-[9rem] truncate rounded px-1 font-mono"
              style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text-secondary)" }}
              title={node.tool}
            >
              {node.tool}
            </span>
          )}
          {node.cat === "skill" && node.name && (
            <span
              className="max-w-[9rem] truncate rounded px-1 font-mono"
              style={{ background: "color-mix(in srgb, " + CAT_COLORS.skill + " 18%, transparent)", color: CAT_COLORS.skill }}
              title={node.name}
            >
              {node.name}
            </span>
          )}
          {(node.cat === "tool" || node.cat === "skill") && node.err && (
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
        {detailOpen && d && <NodeDetailPanel d={d} headerFocus={node.cat} />}
      </div>
    </div>
  );
}

// ─── 上下文浏览器（分类折叠组 + 行内搜索 + 归档折叠组） ──────────
// v4.80 深读：分类内排序（时间序/大小序，dsh size/name 同款）+ 节点完整
// 调用懒加载（GaeaContextNodeDetail，按 seq 缓存）。
// 2.5d：排序初值读设置中心偏好、变更写回；focus 跳转——趋势卡 brief 行
// 点击后展开目标组（含分页全量）、滚动进视图并高亮 3s；锚点无对应节点时
// 诚实不跳。
export function ContextBrowserTree({
  nodes,
  archive,
  focus,
  catFocus,
  sessionPath,
  fetchNodeDetail,
  toolFilter,
  requests,
  previewRequest,
  previewing,
  surfaceMode,
  onSurfaceModeChange,
  pinnedSeq,
  onPickRequest,
  surfaceState,
  surfaceDropped,
  delta,
}: {
  nodes: ContextSurfaceNode[];
  archive: ContextSurfaceNode[];
  focus?: { seq: number; tick: number } | null;
  catFocus?: BrowserCatFocus | null;
  sessionPath?: string;
  /** 节点详情自定义源（v4.412 原罪复用）；缺省 GaeaContextNodeDetail 按会话日志。 */
  fetchNodeDetail?: (seq: number) => Promise<ContextNodeDetailView>;
  /** 1:1 上游 chips 联动：工具名过滤（来自 Context Stats 工具 chip 点击）。 */
  toolFilter?: string | null;
  /** 请求边界：DNA 行按 node.seq/request.seq 的确定关系标注引入步骤。 */
  requests?: ContextRequestRecord[];
  /** 当前浏览器 source 请求（悬停优先，否则为钉住步骤）。 */
  previewRequest?: ContextRequestRecord | null;
  previewing?: boolean;
  surfaceMode?: "live" | "step";
  onSurfaceModeChange?: (mode: "live" | "step") => void;
  pinnedSeq?: number | null;
  onPickRequest?: (seq: number | null) => void;
  surfaceState?: "live" | "loading" | "ready" | "error" | "unavailable";
  surfaceDropped?: number;
  /** 当前钉住请求相对上一步的逐类差分，用于组行 badge。 */
  delta?: ContextRequestDelta;
}) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [groupQueries, setGroupQueries] = useState<Record<string, string>>({});
  // 默认折叠态：六分类与归档全部收起（dsh 同款）。
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());
  const [openText, setOpenText] = useState<Set<number>>(() => new Set());
  const [showAll, setShowAll] = useState<Set<string>>(() => new Set());
  const [sort, setSort] = useState<CtxBrowserSort>(() => loadContextPrefs().browserSort);
  const { details, open: openDetails, toggle: toggleDetail } = useNodeDetails(sessionPath, fetchNodeDetail);
  // focus 跳转状态：handledFocus 去重（同一 tick 只处理一次）；focusedSeq
  // 高亮当前目标节点，3s 后自动清除。
  const listRef = useRef<HTMLDivElement | null>(null);
  const [focusedSeq, setFocusedSeq] = useState<number | null>(null);
  const handledFocus = useRef(0);
  const handledCatFocus = useRef(0);

  const pickSort = (k: CtxBrowserSort) => {
    setSort(k);
    saveContextPref("browserSort", k);
  };
  // v4.542 刀4：树/平铺（DNA）双视图。DNA=当前活跃 surface 按模型读取顺序
  // （seq 序）平铺编号，系统提示词/工具定义作两行分组收尾（dsh 同款语义）；
  // 「引入步骤」日志无逐节点记录，诚实不造列。助手 kind 筛选（思考/工具/
  // 正文）用刀1 三拆子计数推导，一次回复可命中多种。
  const [view, setView] = useState<"tree" | "dna" | "map">("tree");
  const [assistantKind, setAssistantKind] = useState<"thinking" | "tools" | "answer" | null>(null);
  const dnaEntries = useMemo(
    () => nodes
      .filter((n) => n.cat !== "system" && n.cat !== "tools")
      .sort((a, b) => a.seq - b.seq)
      .map((node, index) => ({ node, pos: index + 1 })),
    [nodes],
  );
  const dnaEntriesShown = useMemo(() => {
    return dnaEntries
      .filter(({ node }) => nodeMatchesQuery(node, query))
      .filter(({ node }) => nodeMatchesQuery(node, groupQueries[node.cat] ?? ""))
      .filter(({ node }) => !toolFilter || node.tool === toolFilter)
      .slice()
      .reverse();
  }, [dnaEntries, query, groupQueries, toolFilter]);
  const stampForNode = (node: ContextSurfaceNode): string | null => {
    if (!requests?.length) return null;
    let hit: ContextRequestRecord | null = null;
    for (const request of requests) {
      if (request.seq <= node.seq) hit = request;
      else break;
    }
    return hit ? t("contextview.browserStepAt", { turn: hit.turn, step: hit.step }) : null;
  };
  const lastSystem = useMemo(() => nodes.filter((n) => n.cat === "system").at(-1), [nodes]);
  const lastTools = useMemo(() => nodes.filter((n) => n.cat === "tools").at(-1), [nodes]);
  const stepOptions = useMemo(() => {
    const options = (requests ?? []).slice(-200).reverse();
    if (pinnedSeq != null && !options.some((request) => request.seq === pinnedSeq)) {
      const pinned = requests?.find((request) => request.seq === pinnedSeq);
      if (pinned) options.unshift(pinned);
    }
    return options;
  }, [pinnedSeq, requests]);
  const kindMatch = (n: ContextSurfaceNode): boolean => {
    if (assistantKind === null) return true;
    if (assistantKind === "thinking") return (n.reasoning ?? 0) > 0;
    if (assistantKind === "tools") return (n.callTokens ?? 0) > 0;
    return (n.answerTokens ?? 0) > 0;
  };
  useEffect(() => {
    if (!focus || focus.tick === handledFocus.current) return;
    handledFocus.current = focus.tick;
    const inArchive = archive.some((n) => n.seq === focus.seq);
    const node = inArchive
      ? archive.find((n) => n.seq === focus.seq)
      : nodes.find((n) => n.seq === focus.seq);
    if (!node) return; // 锚点 0 / 节点已不在当前 surface → 诚实不跳
    const groupKey = inArchive ? ARCHIVE_KEY : node.cat;
    setExpanded((cur) => new Set(cur).add(groupKey));
    setShowAll((cur) => new Set(cur).add(groupKey)); // 目标可能在分页后半段
    setQuery("");
    setGroupQueries((cur) => ({ ...cur, [groupKey]: "" }));
    setFocusedSeq(focus.seq);
    requestAnimationFrame(() => {
      listRef.current?.querySelector(`[data-node-seq="${focus.seq}"]`)?.scrollIntoView({ block: "nearest" });
    });
    const timer = window.setTimeout(() => setFocusedSeq(null), 3000);
    return () => window.clearTimeout(timer);
  }, [focus, nodes, archive]);

  useEffect(() => {
    if (!catFocus || catFocus.tick === handledCatFocus.current) return;
    handledCatFocus.current = catFocus.tick;
    setView("tree");
    setExpanded((cur) => new Set(cur).add(catFocus.cat));
    setShowAll((cur) => new Set(cur).add(catFocus.cat));
    setQuery("");
    setGroupQueries((cur) => ({ ...cur, [catFocus.cat]: catFocus.query ?? "" }));
    requestAnimationFrame(() => {
      listRef.current
        ?.querySelector(`[data-browser-group="${catFocus.cat}"]`)
        ?.scrollIntoView({ block: "nearest" });
    });
  }, [catFocus]);

  const sorter = useMemo(
    () => {
      if (sort === "size") {
        return (a: ContextSurfaceNode, b: ContextSurfaceNode) => b.tokens - a.tokens || a.seq - b.seq;
      }
      if (sort === "name") {
        return (a: ContextSurfaceNode, b: ContextSurfaceNode) => {
          const an = a.name ?? a.tool ?? a.text ?? "";
          const bn = b.name ?? b.tool ?? b.text ?? "";
          return an.localeCompare(bn) || a.seq - b.seq;
        };
      }
      return (a: ContextSurfaceNode, b: ContextSurfaceNode) => a.seq - b.seq;
    },
    [sort],
  );

  // 组行右侧 ≈tokens 用未过滤的分类合计（占比分母=活跃节点总 tokens，数值稳定）；
  // 「N 项」徽标随搜索过滤（反映展开后实际列出的行数）。
  const groups = useMemo(
    () =>
      GROUPS.map((g) => {
        const all = nodes.filter((n) => n.cat === g.key);
        let hit = all.filter((n) => nodeMatchesQuery(n, query));
        hit = hit.filter((n) => nodeMatchesQuery(n, groupQueries[g.key] ?? ""));
        if (toolFilter) hit = hit.filter((n) => n.tool === toolFilter);
        const filtered = [...hit].sort(sorter); // v4.80 分类内排序（时间序/大小序）
        const tokens = all.reduce((s, n) => s + n.tokens, 0);
        const deltaCat = delta?.byCat?.find((entry) => entry.cat === g.key);
        return { ...g, all: all.length, count: filtered.length, tokens, nodes: filtered, deltaCat };
      }),
    [nodes, query, groupQueries, toolFilter, sorter, delta],
  );
  const archiveShown = useMemo(() => {
    let hit = archive.filter((n) => nodeMatchesQuery(n, query));
    hit = hit.filter((n) => nodeMatchesQuery(n, groupQueries[ARCHIVE_KEY] ?? ""));
    return [...hit].sort(sorter);
  }, [archive, query, groupQueries, sorter]);
  const archiveTokens = useMemo(() => archive.reduce((s, n) => s + n.tokens, 0), [archive]);
  const totalTokens = useMemo(() => nodes.reduce((s, n) => s + n.tokens, 0), [nodes]);
  const kindCounts = useMemo(() => {
    const c = { thinking: 0, tools: 0, answer: 0 };
    const assistantNodes = groups.find((group) => group.key === "assistant")?.nodes ?? [];
    for (const n of assistantNodes) {
      if ((n.reasoning ?? 0) > 0) c.thinking++;
      if ((n.callTokens ?? 0) > 0) c.tools++;
      if ((n.answerTokens ?? 0) > 0) c.answer++;
    }
    return c;
  }, [groups]);
  const mapRects = useMemo(
    () => treemapRects(
      groups
        .filter((group) => group.tokens > 0)
        .map((group) => ({ key: group.key, tokens: group.tokens })),
    ),
    [groups],
  );

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
  const openCategory = (key: ContextSurfaceNode["cat"]) => {
    setView("tree");
    setExpanded((cur) => new Set(cur).add(key));
    setShowAll((cur) => new Set(cur).add(key));
    requestAnimationFrame(() => {
      listRef.current
        ?.querySelector(`[data-browser-group="${key}"]`)
        ?.scrollIntoView({ block: "nearest" });
    });
  };

  const surfaceBlocked = surfaceState === "loading" || surfaceState === "error" || surfaceState === "unavailable";
  const isEmpty = !surfaceBlocked && nodes.length === 0 && archive.length === 0;

  return (
    <div className="ctx-card p-3" data-testid="ctx-browser">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="ctx-head-ic" aria-hidden>
            <Layers size={12} />
          </span>
          <span className="truncate text-[12.5px] font-semibold text-fg">{t("contextview.browserTitle")}</span>
        </div>
        <div className="shrink-0 text-[9px] tabular-nums text-fg-faint">
          {t("contextview.tabActive", { n: nodes.length })} · ≈{fmtTokens(totalTokens)}
          {(surfaceDropped ?? 0) > 0 && ` · −${surfaceDropped}`}
        </div>
      </div>
      <div className="ctx-browser-source mt-1" data-testid="ctx-browser-source">
        <button
          type="button"
          aria-pressed={(surfaceMode ?? "live") === "live"}
          className="ctx-browser-source-live"
          data-testid="ctx-browser-source-live"
          onClick={() => onSurfaceModeChange?.("live")}
        >
          {t("contextview.surfaceLive")}
        </button>
        <select
          aria-label={t("contextview.surfaceStepPick")}
          data-testid="ctx-browser-source-step"
          value={(surfaceMode === "step" && pinnedSeq != null) ? String(pinnedSeq) : ""}
          onChange={(event) => {
            const value = event.target.value;
            if (!value) {
              onPickRequest?.(null);
              return;
            }
            onSurfaceModeChange?.("step");
            onPickRequest?.(Number(value));
          }}
        >
          <option value="">{t("contextview.surfaceStepPick")}</option>
          {stepOptions.map((request) => (
            <option key={request.seq} value={request.seq}>
              {t("contextview.surfaceStepOption", { turn: request.turn, step: request.step })}
            </option>
          ))}
        </select>
      </div>
      {previewRequest && (
        <div
          className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 rounded-md bg-accent/8 px-1.5 py-1 text-[9.5px] text-fg-faint"
          data-testid="ctx-browser-preview"
          data-preview-seq={previewRequest.seq}
        >
          <b className="font-mono text-fg-dim">
            {t("contextview.browserPreview", { turn: previewRequest.turn, step: previewRequest.step })}
          </b>
          <span>{previewing ? t("contextview.browserPreviewLive") : t("contextview.browserPinnedSurface")}</span>
          <span className="ml-auto font-mono tabular-nums">
            ≈{fmtTokens(Object.values(previewRequest.category).reduce((sum, value) => sum + value, 0))}
          </span>
        </div>
      )}
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
        {([
          ["time", "contextview.sortTime"],
          ["size", "contextview.sortSize"],
          ["name", "contextview.sortName"],
        ] as const).map(([k, key]) => (
          <button
            key={k}
            aria-pressed={sort === k}
            className={`cursor-pointer rounded border-0 px-1.5 py-0.5 ${sort === k ? "bg-accent/15 text-accent" : "bg-transparent text-fg-faint hover:text-fg"}`}
            onClick={() => pickSort(k)}
          >{t(key)}</button>
        ))}
              <span className="ml-auto" />
        {(["tree", "dna", "map"] as const).map((v) => (
          <button
            key={v}
            aria-pressed={view === v}
            data-testid={`ctx-browser-view-${v}`}
            className={`cursor-pointer rounded border-0 px-1.5 py-0.5 ${view === v ? "bg-accent/15 text-accent" : "bg-transparent text-fg-faint hover:text-fg"}`}
            onClick={() => setView(v)}
          >{v === "tree" ? t("contextview.viewTree") : v === "dna" ? t("contextview.viewDna") : t("contextview.viewMap")}</button>
        ))}
      </div>
      {surfaceState === "loading" ? (
        <div className="py-2 text-[10px] text-fg-faint" data-testid="ctx-browser-surface-loading">
          {t("contextview.surfaceLoading")}
        </div>
      ) : surfaceState === "error" ? (
        <div className="py-2 text-[10px] text-err" data-testid="ctx-browser-surface-error">
          {t("contextview.surfaceError")}
        </div>
      ) : surfaceState === "unavailable" ? (
        <div className="py-2 text-[10px] text-warning" data-testid="ctx-browser-surface-unavailable">
          {t("contextview.surfaceUnavailable")}
        </div>
      ) : isEmpty ? (
        <div className="py-2 text-[10px] text-fg-faint">{t("contextview.noNodes")}</div>
      ) : view === "map" ? (
        <div ref={listRef} className="mt-1.5" data-testid="ctx-browser-map">
          <div className="ctx-browser-map-hint">{t("contextview.mapHint")}</div>
          <div className="ctx-browser-map-shell" role="group" aria-label={t("contextview.viewMap")}>
            {mapRects.map((rect) => {
              const group = groups.find((item) => item.key === rect.key);
              if (!group) return null;
              const share = totalTokens > 0 ? Math.round((rect.tokens / totalTokens) * 1000) / 10 : 0;
              const label = t("contextview.mapItemLabel", {
                cat: t(group.labelKey),
                items: group.all,
                tokens: fmtTokens(rect.tokens),
                pct: share,
              });
              const compact = rect.width < 20 || rect.height < 22;
              return (
                <button
                  key={rect.key}
                  type="button"
                  data-map-cat={rect.key}
                  data-testid={`ctx-browser-map-item-${rect.key}`}
                  data-compact={compact ? "true" : undefined}
                  aria-label={label}
                  title={label}
                  className="ctx-browser-map-item"
                  style={{
                    left: `${rect.x}%`,
                    top: `${rect.y}%`,
                    width: `calc(${rect.width}% - 2px)`,
                    height: `calc(${rect.height}% - 2px)`,
                    "--map-color": CAT_COLORS[rect.key],
                  } as CSSProperties}
                  onClick={() => openCategory(rect.key)}
                >
                  <span className="ctx-browser-map-name">
                    <i aria-hidden />
                    <span>{t(group.labelKey)}</span>
                  </span>
                  <span className="ctx-browser-map-value">≈{fmtTokens(rect.tokens)} · {share}%</span>
                  <span className="ctx-browser-map-count">{t("contextview.itemsCount", { n: group.all })}</span>
                </button>
              );
            })}
            {mapRects.length === 0 && (
              <div className="ctx-browser-map-empty">{t("contextview.noNodes")}</div>
            )}
          </div>
        </div>
      ) : view === "dna" ? (
        /* v4.542 刀4 DNA 平铺（dsh 同名能力诚实子集）：当前活跃 surface 按
           模型读取顺序编号平铺；系统提示词/工具定义作分组行收尾；行交互
           （展开预览/懒加载详情）与树视图共用 NodeRow。 */
        <div ref={listRef} className="mt-1.5 flex max-h-72 flex-col gap-1 overflow-y-auto pr-0.5" data-testid="ctx-browser-dna">
          {dnaEntriesShown.map(({ node: n, pos }) => (
            <div key={n.seq} className="flex items-start gap-1" data-testid={`ctx-browser-dna-row-${n.seq}`}>
              <span className="w-9 shrink-0 pt-1.5 text-right font-mono text-[9px] text-fg-faint" title={t("contextview.dnaPosTip")}>#{pos}</span>
              <div className="min-w-0 flex-1">
                {stampForNode(n) && (
                  <div className="px-2 pt-1 font-mono text-[8.5px] text-fg-faint">{stampForNode(n)}</div>
                )}
                <NodeRow
                  node={n}
                  open={openText.has(n.seq)}
                  onToggle={() => toggleText(n.seq)}
                  detailState={details.get(n.seq)}
                  detailOpen={openDetails.has(n.seq)}
                  onToggleDetail={() => toggleDetail(n.seq)}
                  focused={focusedSeq === n.seq}
                />
              </div>
            </div>
          ))}
          {[lastSystem, lastTools].map((n, i) =>
            n && nodeMatchesQuery(n, query) && nodeMatchesQuery(n, groupQueries[n.cat] ?? "") ? (
              <div key={`tail-${n.seq}`} className="ctx-row flex items-center gap-1.5 border-t border-border-soft/60 px-2 py-1.5 text-[10px]">
                <span className="h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[n.cat] }} aria-hidden />
                <span className="font-medium text-fg-dim">{t(i === 0 ? "contextview.dnaTailSystem" : "contextview.dnaTailTools")}</span>
                <span className="font-mono tabular-nums text-fg-faint">≈{fmtTokens(n.tokens)}</span>
                <span className="ml-auto max-w-[50%] truncate font-mono text-fg-faint" title={n.text}>{n.text}</span>
              </div>
            ) : null,
          )}
          {dnaEntriesShown.length === 0 &&
            ![lastSystem, lastTools].some((n) => n && nodeMatchesQuery(n, query) && nodeMatchesQuery(n, groupQueries[n.cat] ?? "")) && (
              <div className="py-2 text-[10px] text-fg-faint">{t("contextview.noCatNodes")}</div>
            )}
        </div>
      ) : (
        <div ref={listRef} className="mt-1.5 flex max-h-72 flex-col gap-1 overflow-y-auto pr-0.5">
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
                  data-browser-group={g.key}
                  onClick={() => toggleGroup(g.key)}
                  className="ctx-row flex w-full cursor-pointer items-center gap-1.5 border-0 px-1.5 py-1.5 text-left text-[10.5px]"
                >
                  <span className="h-2 w-2 shrink-0 rounded-sm" style={{ background: CAT_COLORS[g.key] }} aria-hidden />
                  <span className="font-medium text-fg-dim">{t(g.labelKey)}</span>
                  <span
                    className="rounded-full px-1.5 text-[9px] tabular-nums"
                    style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text-secondary)" }}
                  >
                      {t("contextview.itemsCount", { n: g.count })}
                      </span>
                      {g.deltaCat && (g.deltaCat.items !== 0 || g.deltaCat.tokens !== 0) && (
                        <span className="ctx-browser-delta" data-sign={g.deltaCat.tokens >= 0 ? "up" : "down"}>
                          {(g.deltaCat.items ?? 0) !== 0 && `${(g.deltaCat.items ?? 0) > 0 ? "+" : "−"}${Math.abs(g.deltaCat.items ?? 0)}`}
                          {(g.deltaCat.items ?? 0) !== 0 && g.deltaCat.tokens !== 0 && " · "}
                          {g.deltaCat.tokens !== 0 && `${g.deltaCat.tokens > 0 ? "+" : "−"}${fmtTokens(Math.abs(g.deltaCat.tokens))}`}
                        </span>
                      )}
                      <span className="ml-auto shrink-0 font-mono text-[9.5px] tabular-nums text-fg-faint">
                        ≈{fmtTokens(g.tokens)} ({share}%)
                      </span>
                </button>
                {isOpen && (
                  <div className="ml-3.5 mt-0.5 flex flex-col gap-1 border-l border-border-soft/60 pl-2">
                    <input
                      type="text"
                      value={groupQueries[g.key] ?? ""}
                      onChange={(event) => {
                        const value = event.target.value;
                        setGroupQueries((cur) => ({ ...cur, [g.key]: value }));
                      }}
                      placeholder={t("contextview.browserCategorySearch", { cat: t(g.labelKey) })}
                      aria-label={t("contextview.browserCategorySearch", { cat: t(g.labelKey) })}
                      data-testid={`ctx-browser-search-${g.key}`}
                      className="h-5 rounded border border-border-soft bg-bg-soft px-1.5 text-[9.5px] text-fg placeholder:text-fg-faint focus:border-accent focus:outline-none"
                    />
                    {g.key === "assistant" && (
                      /* v4.542 刀4：助手 kind 筛选 chips（dsh Thinking/Tools/Answer
                         同款；次数=该类行数；一次回复可含多种，再点清除） */
                      <div className="flex flex-wrap items-center gap-1 py-1" data-testid="ctx-assistant-kinds">
                        {([["thinking", kindCounts.thinking], ["tools", kindCounts.tools], ["answer", kindCounts.answer]] as const).map(([k, n]) => (
                          <button
                            key={k}
                            type="button"
                            aria-pressed={assistantKind === k}
                            disabled={n === 0}
                            className={`rounded-full border-0 px-1.5 py-px text-[9.5px] ${assistantKind === k ? "bg-accent/20 text-accent" : "bg-bg-soft text-fg-dim hover:text-fg"} ${n === 0 ? "cursor-default opacity-40" : "cursor-pointer"}`}
                            onClick={() => setAssistantKind((cur) => (cur === k ? null : k))}
                          >
                            {t(k === "thinking" ? "contextview.kindThinking" : k === "tools" ? "contextview.kindTools" : "contextview.kindAnswer")} {n}
                          </button>
                        ))}
                      </div>
                    )}
                    {g.count === 0 && <div className="py-1 text-[10px] text-fg-faint">{t("contextview.noCatNodes")}</div>}
                    {(g.key === "assistant" ? listed.filter(kindMatch) : listed).map((n) => (
                      <NodeRow
                        key={n.seq}
                        node={n}
                        open={openText.has(n.seq)}
                        onToggle={() => toggleText(n.seq)}
                        detailState={details.get(n.seq)}
                        detailOpen={openDetails.has(n.seq)}
                        onToggleDetail={() => toggleDetail(n.seq)}
                        focused={focusedSeq === n.seq}
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
                        {t("contextview.itemsCount", { n: archiveShown.length })}
                      </span>
                      <span className="ml-auto shrink-0 font-mono text-[9.5px] tabular-nums text-fg-faint">≈{fmtTokens(archiveTokens)}</span>
                </button>
                {isOpen && (
                  <div className="ml-3.5 mt-0.5 flex flex-col gap-1 border-l border-border-soft/60 pl-2">
                    <input
                      type="text"
                      value={groupQueries[ARCHIVE_KEY] ?? ""}
                      onChange={(event) => {
                        const value = event.target.value;
                        setGroupQueries((cur) => ({ ...cur, [ARCHIVE_KEY]: value }));
                      }}
                      placeholder={t("contextview.browserCategorySearch", { cat: t("contextview.tabArchive", { n: archive.length }) })}
                      aria-label={t("contextview.browserCategorySearch", { cat: t("contextview.tabArchive", { n: archive.length }) })}
                      data-testid={`ctx-browser-search-${ARCHIVE_KEY}`}
                      className="h-5 rounded border border-border-soft bg-bg-soft px-1.5 text-[9.5px] text-fg placeholder:text-fg-faint focus:border-accent focus:outline-none"
                    />
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
                        focused={focusedSeq === n.seq}
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
export type FileChip = "all" | "read" | "write" | "search" | "image";
export type FileChipFocus = { chip: Exclude<FileChip, "all">; tick: number };
type FileSort = CtxFileSort;

// dsh 的「搜索/图片」在本仓 FileActivity.action（read/write/move/dir）中无对应值：
// 按工具名/扩展名启发式推断。chips 是过滤器而非互斥分类——搜索/图片与读/写
// 允许重叠（一个文件可以既被 grep 命中又是图片路径）。
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
  { key: "lines", labelKey: "contextview.filesSortLines" },
  { key: "recent", labelKey: "contextview.filesSortLatest" },
  { key: "path", labelKey: "contextview.filesSortPath" },
  { key: "name", labelKey: "contextview.filesSortName" },
];

// 排序稳定性：并列时依次以最新时间/路径决胜，保证任意输入下顺序确定。
const SORT_CMP: Record<FileSort, (a: FileAgg, b: FileAgg) => number> = {
  count: (a, b) => b.total - a.total || b.latest - a.latest || a.path.localeCompare(b.path),
  lines: (a, b) => (b.added + b.removed) - (a.added + a.removed) || b.latest - a.latest || a.path.localeCompare(b.path),
  recent: (a, b) => b.latest - a.latest || a.path.localeCompare(b.path),
  path: (a, b) => a.path.localeCompare(b.path),
  name: (a, b) => {
    const an = a.path.split(/[\\/]/).pop() ?? a.path;
    const bn = b.path.split(/[\\/]/).pop() ?? b.path;
    return an.localeCompare(bn) || a.path.localeCompare(b.path);
  },
};

export function FileActivityTree({ files, onOpenFile, sessionPath, fetchNodeDetail, chipFocus, onLocateOp, step }: {
  files: FileActivity[];
  onOpenFile?: (path: string) => void;
  sessionPath?: string;
  fetchNodeDetail?: (seq: number) => Promise<ContextNodeDetailView>;
  chipFocus?: FileChipFocus | null;
  onLocateOp?: (seq: number) => void;
  step?: ContextRequestRecord | null;
}) {
  const t = useT();
  // 预览打开二选一：默认内部走 usePreviewStore（与现有 FileActivityCard 同款链路），
  // onOpenFile 作为可选注入覆盖（契约签名保持兼容：传或不传均可）。
  const openPreview = usePreviewStore((s) => s.openFilePreview);
  const open = onOpenFile ?? openPreview;
  const [chip, setChip] = useState<FileChip>("all");
  const [query, setQuery] = useState("");
  // 2.5d：排序初值读设置中心偏好、变更写回（与浏览器分类内排序同模式）。
  const [sort, setSort] = useState<CtxFileSort>(() => loadContextPrefs().fileSort);
  // 节点详情读取源与浏览器同源下传（FE1-04）：原罪等自定义源页面若不下传，
  // 这里会按缺省会话日志回读，同一页面两套数据源串台。
  const { details, open: openDetails, toggle: toggleDetail } = useNodeDetails(sessionPath, fetchNodeDetail);
  // v4.81 操作日志展开（dsh「展开完整操作日志」同款）：按路径展开逐次操作行。
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(() => new Set());

  const pickSort = (k: CtxFileSort) => {
    setSort(k);
    saveContextPref("fileSort", k);
  };

  useEffect(() => {
    if (!chipFocus) return;
    setChip(chipFocus.chip);
    setQuery("");
  }, [chipFocus]);

  const scopedFiles = useMemo(() => {
    if (!step) return files;
    return files.filter((file) =>
      file.turn < step.turn || (file.turn === step.turn && file.step <= step.step),
    );
  }, [files, step]);

  const q = query.trim().toLowerCase();

  const counts = useMemo(() => {
    const c: Record<FileChip, number> = { all: scopedFiles.length, read: 0, write: 0, search: 0, image: 0 };
    for (const f of scopedFiles) {
      if (chipMatch(f, "read")) c.read += 1;
      if (chipMatch(f, "write")) c.write += 1;
      if (chipMatch(f, "search")) c.search += 1;
      if (chipMatch(f, "image")) c.image += 1;
    }
    return c;
  }, [scopedFiles]);

  const { rows, opIndex } = useMemo(() => {
    const filtered = scopedFiles.filter((f) => chipMatch(f, chip) && (q === "" || f.path.toLowerCase().includes(q)));
    const ops = new Map<string, FileActivity[]>();
    for (const f of filtered) {
      const list = ops.get(f.path);
      if (list) list.push(f);
      else ops.set(f.path, [f]);
    }
    return { rows: aggregate(filtered).sort(SORT_CMP[sort]), opIndex: ops };
  }, [scopedFiles, chip, q, sort]);
  const shownTotals = useMemo(() => ({
    added: rows.reduce((sum, row) => sum + Math.max(0, row.added), 0),
    removed: rows.reduce((sum, row) => sum + Math.max(0, row.removed), 0),
  }), [rows]);

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
    <div className="ctx-card p-3 ctx-files-card" data-testid="ctx-files">
      <div className="ctx-file-head">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="ctx-head-ic" aria-hidden>
            <FolderTree size={12} />
          </span>
          <span className="text-[12.5px] font-semibold text-fg">{t("contextview.filesTitle")}</span>
          {step && (
            <span className="ctx-file-scope" data-testid="ctx-files-step-scope">
              {t("contextview.filesAtStep", { turn: step.turn, step: step.step })}
            </span>
          )}
          <span className="ctx-file-meta">
            {t("contextview.filesCount", { n: rows.length })}
            {shownTotals.added > 0 && <b className="ctx-file-added">+{fmtTokens(shownTotals.added)}</b>}
            {shownTotals.removed > 0 && <b className="ctx-file-removed">-{fmtTokens(shownTotals.removed)}</b>}
          </span>
        </div>
        <div className="ctx-file-chips">
        {chips.map((c) => (
          <button
            key={c.key}
            type="button"
            aria-pressed={chip === c.key}
            data-tone={c.key}
            data-testid={`ctx-file-chip-${c.key}`}
            className="ctx-file-chip"
            onClick={() => setChip(c.key)}
          >
            {t(c.labelKey)}
            <span className="ml-1 tabular-nums">{counts[c.key]}</span>
          </button>
        ))}
        </div>
      </div>
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t("contextview.filesFilter")}
        aria-label={t("contextview.filesFilter")}
        className="mt-1.5 h-6 w-full rounded-md border border-border-soft bg-bg-soft px-2 text-[10px] text-fg placeholder:text-fg-faint focus:border-accent focus:outline-none"
      />
      <div className="ctx-file-controls">
        <span className="ctx-file-sorts">
          {SORT_PILLS.map((p) => (
            <button
              key={p.key}
              type="button"
              aria-pressed={sort === p.key}
              className="ctx-file-sort"
              onClick={() => pickSort(p.key)}
            >
              {t(p.labelKey)}
            </button>
          ))}
        </span>
      </div>
      <div className="ctx-files-list mt-1 flex max-h-64 flex-col gap-1 overflow-y-auto pr-0.5">
        {scopedFiles.length === 0 && <div className="py-2 text-[10px] text-fg-faint">{t("contextview.noFiles")}</div>}
        {scopedFiles.length > 0 && rows.length === 0 && (
          <div className="py-2 text-[10px] text-fg-faint">{t("contextview.filesNoMatch")}</div>
        )}
        {rows.map((a) => {
          // 纯目录行（无读/写/移）不可预览，与现有 FileActivityCard 的 dir 行为一致。
          const clickable = a.read + a.write + a.move > 0;
          const ops = opIndex.get(a.path) ?? [];
          const isOpen = expandedPaths.has(a.path);
          const slash = Math.max(a.path.lastIndexOf("/"), a.path.lastIndexOf("\\"));
          const dir = slash >= 0 ? a.path.slice(0, slash + 1) : "";
          const base = slash >= 0 ? a.path.slice(slash + 1) : a.path;
          return (
            <div key={a.path}>
              <div className="ctx-file-row">
                <button
                  type="button"
                  data-file={a.path}
                  disabled={!clickable}
                  onClick={() => clickable && open(a.path)}
                  title={clickable ? t("contextview.previewTitle", { path: a.path }) : undefined}
                  className={`ctx-file-open flex min-w-0 flex-1 items-center gap-1.5 border-0 bg-transparent p-0 text-left text-[10px] ${
                    clickable ? "cursor-pointer" : "cursor-default"
                  }`}
                >
                  <FileTypeIcon name={a.path} size={12} />
                  <span className="ctx-file-path" title={a.path}>
                    {dir && <span className="ctx-file-dir">{dir}</span>}
                    <span className="ctx-file-name">{base}</span>
                  </span>
                  {/* v4.81 行级增量徽标（dsh ±added/−removed 同款） */}
                  {(a.added > 0 || a.removed > 0) && (
                    <span className="ctx-file-delta">
                      {a.added > 0 && <span className="ctx-file-added">+{a.added}</span>}
                      {a.added > 0 && a.removed > 0 && <span className="text-fg-faint">/</span>}
                      {a.removed > 0 && <span className="ctx-file-removed">−{a.removed}</span>}
                    </span>
                  )}
                  <span className="ctx-file-badges">
                    {a.read > 0 && (
                      <span className="ctx-file-badge" data-kind="read">
                        {`${t(FILE_ACTION_META.read.labelKey)} ${a.read}`}
                      </span>
                    )}
                    {a.write > 0 && (
                      <span className="ctx-file-badge" data-kind="write">
                        {`${t(FILE_ACTION_META.write.labelKey)} ${a.write}`}
                      </span>
                    )}
                    {a.move > 0 && (
                      <span className="ctx-file-badge" data-kind="move">
                        {`${t(FILE_ACTION_META.move.labelKey)} ${a.move}`}
                      </span>
                    )}
                    {a.dir > 0 && (
                      <span className="ctx-file-badge" data-kind="dir">
                        {`${t(FILE_ACTION_META.dir.labelKey)} ${a.dir}`}
                      </span>
                    )}
                    <span className="ctx-file-time">{new Date(a.latest * 1000).toLocaleTimeString()}</span>
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
                    onLocate={onLocateOp}
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
  onLocate,
}: {
  op: FileActivity;
  detailState?: NodeDetailState;
  detailOpen: boolean;
  onToggleDetail: () => void;
  onLocate?: (seq: number) => void;
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
        {onLocate && (
          <button
            type="button"
            data-testid="file-op-locate-btn"
            className="cursor-pointer border-0 bg-transparent text-accent hover:underline"
            onClick={() => onLocate(op.seq)}
          >
            {t("contextview.locateInBrowser")}
          </button>
        )}
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
