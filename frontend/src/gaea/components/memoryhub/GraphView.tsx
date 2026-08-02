import { useEffect, useRef, useState } from "react";
import ForceGraph3D from "three-forcegraph";
import { Modal, Spin } from "antd";
import { app } from "../../lib/bridge";
import type { GraphNode, MemoryGraphView } from "../../lib/types";

const TYPE_COLORS: Record<string, string> = {
  knowledge: "#818cf8", // indigo
  profile: "#fbbf24", // amber
  office: "#34d399", // emerald
  whisper: "#f472b6", // pink
};
const TYPE_LABELS: Record<string, string> = {
  knowledge: "知识", profile: "画像", office: "办公记忆", whisper: "轻语",
};
const TYPE_KEYS = ["knowledge", "profile", "office", "whisper"] as const;
const LINK_COLORS: Record<string, string> = {
  "same-tag": "rgba(129,140,248,0.30)",
  "same-category": "rgba(52,211,153,0.30)",
  reference: "rgba(244,114,182,0.40)",
};

/** GraphView 记忆 3D 图谱：节点=记忆实体，边=同标签/同分类/[[引用]]。
 *  variant="page" 带工具条（库面板内）；variant="home" 纯净展示（首页中央）。 */
export function GraphView(p: { variant?: "page" | "home" }) {
  const variant = p.variant ?? "page";
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<any>(null);
  const dataRef = useRef<MemoryGraphView | null>(null);
  const [loading, setLoading] = useState(true);
  const [typeFilter, setTypeFilter] = useState<Set<string>>(new Set([...TYPE_KEYS]));
  const [selected, setSelected] = useState<GraphNode | null>(null);
  const [nodeCount, setNodeCount] = useState(0);
  const [linkCount, setLinkCount] = useState(0);

  // 拉取图谱数据
  useEffect(() => {
    app
      .MemoryGraph()
      .then((g) => {
        dataRef.current = g;
        setNodeCount(g.nodes.length);
        setLinkCount(g.links.length);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  // 初始化 ForceGraph3D（once）
  useEffect(() => {
    if (!containerRef.current || graphRef.current) return;
    const fg: any = (ForceGraph3D as any)()(containerRef.current)
      .backgroundColor("#0b1020")
      .nodeVal((d: any) => d.val ?? 1)
      .nodeColor((d: any) => TYPE_COLORS[d.type] ?? "#64748b")
      .nodeLabel(
        (d: any) =>
          `<div style="font:12px sans-serif;color:#e2e8f0;background:rgba(15,23,42,0.92);border:1px solid rgba(99,102,241,0.4);border-radius:8px;padding:6px 10px;max-width:260px">` +
          `<div style="font-weight:600;font-size:13px">${d.name}</div>` +
          `<div style="color:${TYPE_COLORS[d.type] ?? "#94a3b8"};margin:2px 0">${TYPE_LABELS[d.type] ?? d.type}</div>` +
          `<div style="color:#94a3b8;line-height:1.4">${d.desc ?? ""}</div>` +
          `</div>`,
      )
      .linkColor((l: any) => LINK_COLORS[l.type] ?? "rgba(148,163,184,0.25)")
      .linkWidth(1.2)
      .d3AlphaDecay(0.022)
      .warmupTicks(80)
      .onNodeClick((d: any) => setSelected(d));
    graphRef.current = fg;

    // 数据就绪后首次渲染
    if (dataRef.current) applyFilter(fg, dataRef.current, typeFilter);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 类型过滤变化 → 重新构图
  useEffect(() => {
    if (graphRef.current && dataRef.current) {
      applyFilter(graphRef.current, dataRef.current, typeFilter);
    }
  }, [typeFilter]);

  function applyFilter(fg: any, data: MemoryGraphView, filter: Set<string>) {
    const nodes = data.nodes.filter((n) => filter.has(n.type));
    const ids = new Set(nodes.map((n) => n.id));
    const links = data.links.filter((l) => ids.has(l.source) && ids.has(l.target));
    fg.graphData({ nodes, links });
    setNodeCount(nodes.length);
    setLinkCount(links.length);
  }

  const toggleType = (t: string) => {
    setTypeFilter((prev) => {
      const next = new Set(prev);
      if (next.has(t)) next.delete(t);
      else next.add(t);
      return next;
    });
  };

  return (
    <div className="h-full flex flex-col">
      {/* 工具条（home 模式隐藏，纯净展示） */}
      {variant === "page" && (
      <div className="shrink-0 flex items-center gap-2 px-4 pt-3 pb-2">
        <div className="text-fg text-[13px] font-medium">记忆 3D 图谱</div>
        <span className="text-fg-faint text-[11px]">
          {nodeCount} 节点 · {linkCount} 边
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          {TYPE_KEYS.map((t) => (
            <button
              key={t}
              onClick={() => toggleType(t)}
              className={`inline-flex items-center gap-1.5 px-2.5 h-7 rounded-full text-[11.5px] transition-colors border ${
                typeFilter.has(t)
                  ? "border-transparent text-white"
                  : "border-border text-fg-faint hover:text-fg"
              }`}
              style={typeFilter.has(t) ? { background: TYPE_COLORS[t] } : undefined}
            >
              <span className="w-2 h-2 rounded-full" style={{ background: typeFilter.has(t) ? "rgba(255,255,255,0.9)" : TYPE_COLORS[t] }} />
              {TYPE_LABELS[t]}
            </button>
          ))}
        </div>
      </div>
      )}

      {/* 图区 */}
      <div className={`flex-1 min-h-0 relative ${variant === "page" ? "mx-4 mb-4 rounded-xl overflow-hidden border border-border-soft" : "w-full h-full"}`}>
        {loading && (
          <div className="absolute inset-0 flex items-center justify-center bg-bg/60 z-10">
            <Spin tip="构建记忆图谱…" />
          </div>
        )}
        <div ref={containerRef} className="w-full h-full" />
        <div className="absolute left-3 top-3 text-fg-faint text-[10.5px] leading-relaxed pointer-events-none">
          拖拽旋转 · 滚轮缩放 · hover 高亮 · 点击查看详情
        </div>
      </div>

      {/* 节点详情 */}
      <Modal
        open={!!selected}
        onCancel={() => setSelected(null)}
        footer={null}
        width={480}
        title={
          <span>
            <span style={{ color: selected ? TYPE_COLORS[selected.type] : undefined }}>
              {selected ? TYPE_LABELS[selected.type] ?? selected.type : ""}
            </span>
            {" · "}{selected?.name}
          </span>
        }
      >
        {selected && (
          <div className="space-y-2">
            <div className="text-[12px] text-fg-faint">ID: {selected.id}</div>
            {selected.desc && (
              <div className="p-3 rounded-lg bg-bg-soft border border-border text-fg-dim text-[13px] leading-relaxed whitespace-pre-wrap">
                {selected.desc}
              </div>
            )}
            <div className="text-fg-faint text-[11px]">
              来自：{selected.type === "knowledge" ? "知识库" : selected.type === "profile" ? "用户画像" : selected.type === "office" ? "办公记忆" : "轻语记忆"}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
