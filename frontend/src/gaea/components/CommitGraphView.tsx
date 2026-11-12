import { useEffect, useMemo, useRef, useState } from "react";
import { GitBranch, Loader2 } from "../icons";
import type { GitCommitDetailView, GitGraphPageView, GitGraphNodeView } from "../lib/types";
import { buildGitDiff } from "../lib/planDiff";
import { commitLanes, MAX_LANES, type GraphRow } from "../lib/commitGraph";
import { ChangesDiff } from "./ChangesDiff";
import { app } from "../lib/bridge";

// CommitGraphView — G2 只读提交图镜头：分页提交（lane 拓扑 SVG）+ 分支筛选
// （只读，不 checkout/fetch/push）+ 单提交详情展开（消息/引用/numstat/ChangesDiff）。
// 数据自取 GaeaGitGraph/GaeaGitCommitDetail；revision 由父面板刷新钮驱动重拉。
// 上限：单页 50、--skip 分页续读；lane 数 MAX_LANES 钳制（病理拓扑有界渲染）。

const PAGE_SIZE = 50;

// lane 配色：主干 lane 0 用 accent 光色，其余轮换次要令牌（不引入裸 hex）。
const LANE_COLORS = [
  "var(--gaea-glow)",
  "var(--md-sys-color-secondary)",
  "var(--md-sys-color-outline)",
];
const laneColor = (i: number) => LANE_COLORS[i % LANE_COLORS.length];

const LANE_W = 11;
const ROW_H = 24;

function GraphLane({ row, prevTos }: { row: GraphRow; prevTos: Set<number> }) {
  const w = MAX_LANES * LANE_W;
  const h = ROW_H;
  const x = (lane: number) => lane * LANE_W + LANE_W / 2;
  const through = new Set(row.through);
  const lines: React.ReactNode[] = [];
  // 穿行 lane：全高竖线
  through.forEach((l) => {
    lines.push(
      <line key={`t${l}`} x1={x(l)} y1={0} x2={x(l)} y2={h} stroke={laneColor(l)} strokeWidth={1.2} opacity={0.55} />,
    );
  });
  // 上半段承接：上一行边汇入的 lane 若非本行 dot lane 且不在 through（理论不在），
  // 画顶到中；本行 dot lane 若被上一行边指向，画顶到中（dot 接线）。
  prevTos.forEach((l) => {
    if (!through.has(l)) {
      lines.push(
        <line key={`p${l}`} x1={x(l)} y1={0} x2={x(l)} y2={h / 2} stroke={laneColor(l)} strokeWidth={1.2} opacity={0.55} />,
      );
    }
  });
  // 本行边：dot → 行底（下半段），下一行由其 through/承接上半段续上
  row.edges.forEach((e, i) => {
    lines.push(
      <line
        key={`e${i}`}
        x1={x(e.from)}
        y1={h / 2}
        x2={x(e.to)}
        y2={h}
        stroke={laneColor(e.to)}
        strokeWidth={1.2}
        opacity={0.7}
      />,
    );
  });
  return (
    <svg width={w} height={h} className="shrink-0" aria-hidden>
      {lines}
      <circle cx={x(row.lane)} cy={h / 2} r={3} fill={laneColor(row.lane)} />
    </svg>
  );
}

function refChipLabel(tok: string): { text: string; head: boolean; tag: boolean } {
  if (tok.startsWith("HEAD -> ")) return { text: tok.slice(8), head: true, tag: false };
  if (tok === "HEAD") return { text: "HEAD", head: true, tag: false };
  if (tok.startsWith("tag: ")) return { text: `🏷 ${tok.slice(5)}`, head: false, tag: true };
  return { text: tok, head: false, tag: false };
}

function RefChips({ refs }: { refs?: string[] }) {
  if (!refs || refs.length === 0) return null;
  return (
    <span className="flex shrink-0 items-center gap-1">
      {refs.map((tok) => {
        const c = refChipLabel(tok);
        return (
          <span
            key={tok}
            className={`rounded-full px-1.5 py-0.5 text-[9px] font-mono ${
              c.head ? "border border-accent/40 bg-accent/10 text-accent" : "border border-border-soft text-fg-faint"
            }`}
          >
            {c.text}
          </span>
        );
      })}
    </span>
  );
}

export function CommitGraphView({ revision }: { revision: number }) {
  const [page, setPage] = useState<GitGraphPageView | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [branch, setBranch] = useState("");
  const [reachedEnd, setReachedEnd] = useState(false);
  const [detailHash, setDetailHash] = useState<string | null>(null);
  const [detail, setDetail] = useState<GitCommitDetailView | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const detailCache = useRef<Map<string, GitCommitDetailView>>(new Map());

  useEffect(() => {
    let live = true;
    setLoading(true);
    Promise.resolve(app.GaeaGitGraph?.(0, PAGE_SIZE, branch))
      .then((r) => {
        if (!live) return;
        setPage(r && Array.isArray(r.commits) ? r : { commits: [], branches: [] });
        setReachedEnd(!r || !Array.isArray(r.commits) || r.commits.length < PAGE_SIZE);
        setError("");
      })
      .catch((e: unknown) => {
        if (!live) return;
        setPage({ commits: [], branches: [] });
        setReachedEnd(true);
        setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (live) setLoading(false);
      });
    return () => {
      live = false;
    };
  }, [branch, revision]);

  const loadMore = async () => {
    if (!page || loading || reachedEnd) return;
    setLoading(true);
    try {
      const r = await Promise.resolve(app.GaeaGitGraph?.(page.commits.length, PAGE_SIZE, branch));
      const incoming = r && Array.isArray(r.commits) ? r.commits : [];
      const seen = new Set(page.commits.map((c) => c.hash));
      const merged = [...page.commits, ...incoming.filter((c) => !seen.has(c.hash))];
      setPage({ commits: merged, branches: r?.branches ?? page.branches });
      if (incoming.length < PAGE_SIZE) setReachedEnd(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  const toggleDetail = async (hash: string) => {
    if (detailHash === hash) {
      setDetailHash(null);
      return;
    }
    setDetailHash(hash);
    const cached = detailCache.current.get(hash);
    if (cached) {
      setDetail(cached);
      return;
    }
    setDetail(null);
    setDetailLoading(true);
    try {
      const d = await Promise.resolve(app.GaeaGitCommitDetail?.(hash));
      if (d && typeof d === "object") {
        detailCache.current.set(hash, d);
        setDetail(d);
      } else {
        setDetail(null);
      }
    } catch {
      setDetail(null);
    } finally {
      setDetailLoading(false);
    }
  };

  const rows = useMemo(
    () => commitLanes((page?.commits ?? []).map((c) => ({ hash: c.hash, parents: c.parents ?? [] }))),
    [page],
  );
  const prevTosSeq = useMemo(() => rows.map((r) => new Set(r.edges.map((e) => e.to))), [rows]);

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-12 px-6 text-center">
        <GitBranch size={18} aria-hidden style={{ color: "var(--md-sys-color-text-secondary)" }} />
        <div className="mt-3 text-[12.5px] font-medium" style={{ color: "var(--md-sys-color-text)" }}>
          提交图不可用
        </div>
        <div className="mt-1 max-w-[260px] text-[11px] leading-snug" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center gap-1.5 px-2 py-1">
        <select
          aria-label="按分支筛选提交图"
          value={branch}
          onChange={(e) => setBranch(e.target.value)}
          className="cursor-pointer rounded-md border border-border-soft bg-bg px-1.5 py-0.5 text-[10px] text-fg focus:border-accent focus:outline-none"
        >
          <option value="">全部分支</option>
          {(page?.branches ?? []).map((b) => (
            <option key={b} value={b}>{b}</option>
          ))}
        </select>
        <span className="text-[9.5px] text-fg-faint">只读视图（不 checkout）</span>
        {loading && <Loader2 size={10} className="animate-spin text-fg-faint" />}
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        {!page && <div className="px-2.5 py-2 text-[10px] text-fg-faint">读取中…</div>}
        {page && page.commits.length === 0 && (
          <div className="px-2.5 py-2 text-[10px] text-fg-faint">
            {branch ? "该分支暂无提交" : "暂无提交"}
          </div>
        )}
        {page?.commits.map((c: GitGraphNodeView, i: number) => {
          const open = detailHash === c.hash;
          return (
            <div key={c.hash}>
              <button
                type="button"
                className="group flex w-full cursor-pointer items-center gap-1.5 border-0 bg-transparent px-2 py-1 text-left hover:bg-bg-soft/40"
                onClick={() => void toggleDetail(c.hash)}
                aria-expanded={open}
                title={`${c.subject}（${c.short}）`}
              >
                <GraphLane row={rows[i]} prevTos={prevTosSeq[i - 1] ?? new Set()} />
                <span className="shrink-0 font-mono text-[9.5px] text-fg-faint">{c.short}</span>
                <span className="min-w-0 flex-1 truncate text-[11px] text-fg-dim">{c.subject}</span>
                <RefChips refs={c.refs} />
                <span className="shrink-0 text-[9.5px] text-fg-faint" title={c.author}>{c.author}</span>
              </button>
              {open && (
                <div className="ml-2 mr-1.5 mb-1.5 rounded-[var(--radius-md)] border border-border-soft bg-bg-soft/30 px-2.5 py-2">
                  {detailLoading && (
                    <div className="flex items-center gap-1 text-[10px] text-fg-faint">
                      <Loader2 size={10} className="animate-spin" />
                      读取详情…
                    </div>
                  )}
                  {!detailLoading && !detail && (
                    <div className="text-[10px] text-fg-faint">详情读取失败，可重试展开。</div>
                  )}
                  {detail && (
                    <>
                      <div className="text-[11px] font-medium text-fg">{detail.subject}</div>
                      {detail.body && (
                        <div className="mt-0.5 whitespace-pre-wrap text-[10px] leading-snug text-fg-dim">{detail.body}</div>
                      )}
                      <div className="mt-1 flex flex-wrap items-center gap-1.5 text-[9.5px] text-fg-faint">
                        <span className="font-mono">{detail.hash.slice(0, 12)}</span>
                        <span>{detail.author}</span>
                        {!!detail.parents?.length && (
                          <span className="font-mono">合并自 {detail.parents.map((p) => p.slice(0, 7)).join(" + ")}</span>
                        )}
                        <RefChips refs={detail.refs} />
                      </div>
                      {detail.files.length > 0 && (
                        <div className="mt-1.5 flex flex-col gap-0.5">
                          {detail.files.map((f) => (
                            <div key={f.path} className="flex items-baseline gap-1.5 text-[10px]">
                              <span className="min-w-0 flex-1 truncate font-mono text-fg-dim" title={f.path}>{f.path}</span>
                              {f.add === -1 ? (
                                <span className="shrink-0 text-fg-faint">二进制</span>
                              ) : (
                                <span className="shrink-0 font-mono">
                                  <span style={{ color: "var(--add-fg)" }}>+{f.add}</span>
                                  <span className="ml-1" style={{ color: "var(--del-fg)" }}>−{f.del}</span>
                                </span>
                              )}
                            </div>
                          ))}
                        </div>
                      )}
                      <div className="mt-1.5">
                        {detail.diff === "" && !!detail.parents?.length && detail.parents.length > 1 ? (
                          <div className="text-[10px] text-fg-faint">合并提交默认无 diff（git diff-tree 语义）。</div>
                        ) : (
                          <ChangesDiff diff={buildGitDiff(detail.diff)} path={undefined} />
                        )}
                      </div>
                    </>
                  )}
                </div>
              )}
            </div>
          );
        })}
        {page && !reachedEnd && page.commits.length > 0 && (
          <button
            type="button"
            className="mx-2 mb-2 cursor-pointer rounded-md border border-border-soft bg-transparent px-2 py-1 text-[10px] text-fg-dim hover:text-fg disabled:opacity-40"
            disabled={loading}
            onClick={() => void loadMore()}
          >
            {loading ? "读取中…" : "加载更多"}
          </button>
        )}
      </div>
    </div>
  );
}
