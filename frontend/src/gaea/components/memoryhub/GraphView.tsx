import { useEffect, useRef, useState } from "react";
import ForceGraph3D from "3d-force-graph";
import type { ForceGraph3DInstance } from "3d-force-graph";
import { Modal, Spin } from "antd";
import { app } from "../../lib/bridge";
import type { GraphLink, GraphNode, MemoryGraphView } from "../../lib/types";

const TYPE_COLORS: Record<string, string> = {
  knowledge: "#818cf8", // indigo
  profile: "#a78bfa", // violet
  office: "#34d399", // emerald
  whisper: "#f472b6", // pink
  material: "#38bdf8", // sky：项目资料（固定常用文件）
  cost: "#fbbf24", // amber：成本条目
};
const TYPE_LABELS: Record<string, string> = {
  knowledge: "知识", profile: "画像", office: "办公记忆", whisper: "聊天记忆", material: "项目资料", cost: "成本",
};
const TYPE_KEYS = ["knowledge", "profile", "office", "whisper", "material", "cost"] as const;
const LINK_COLORS: Record<string, string> = {
  "same-tag": "rgba(129,140,248,0.30)",
  "same-category": "rgba(52,211,153,0.30)",
  reference: "rgba(244,114,182,0.40)",
};

// 3d-force-graph 运行时导出是可调用 kapsule 工厂：ForceGraph3D()(domEl) → 链式实例；
// 包自带类型（1.76.0）却把默认导出声明成 `new` 构造器，与真实调用面不符。
// 这里用包内导出的 ForceGraph3DInstance 泛型类型精确桥接工厂签名（T6-10.2 类型清零）。
type ForceGraph = ForceGraph3DInstance<GraphNode, GraphLink>;
const createGraph = ForceGraph3D as unknown as () => (element: HTMLElement) => ForceGraph;

/** GraphView 记忆 3D 图谱：节点=记忆实体，边=同标签/同分类/[[引用]]。
 *  variant="page" 带工具条（库面板内）；variant="home" 纯净展示（首页中央）。
 *  onSelect 可选回调：节点被点击 / 详情关闭时通知外层（详情 inspector 联动，不影响原有 Modal）。 */
export function GraphView(p: { variant?: "page" | "home"; onSelect?: (node: GraphNode | null) => void }) {
  const { variant = "page", onSelect } = p;
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<ForceGraph | null>(null);
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
        setNodeCount((g.nodes ?? []).length);
        setLinkCount((g.links ?? []).length);
        setLoading(false);
        // 数据到达 → 若图已初始化立即构图（修复：此前数据返回后无任何触发，图谱空白）
        if (graphRef.current) {
          applyFilter(graphRef.current, g, typeFilter);
        }
      })
      .catch(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 初始化 ForceGraph3D（once）
  useEffect(() => {
    if (!containerRef.current || graphRef.current) return;
    // 图谱背景跟随主题（读 gaea 令牌 --bg-soft，亮暗切换时首帧一致）
    const themeBg =
      getComputedStyle(document.documentElement).getPropertyValue("--bg-soft").trim() || "#1B2336";
    const fg = createGraph()(containerRef.current)
      .backgroundColor(themeBg)
      .nodeVal((d) => d.val ?? 1)
      .nodeColor((d) => TYPE_COLORS[d.type] ?? "#64748b")
      .nodeLabel(
        (d) =>
          `<div style="font:12px sans-serif;color:var(--fg);background:color-mix(in srgb, var(--bg-elev) 92%, transparent);border:1px solid color-mix(in srgb, var(--accent) 40%, transparent);border-radius:8px;padding:6px 10px;max-width:260px">` +
          `<div style="font-weight:600;font-size:13px">${d.name}</div>` +
          `<div style="color:${TYPE_COLORS[d.type] ?? "var(--fg-dim)"};margin:2px 0">${TYPE_LABELS[d.type] ?? d.type}</div>` +
          `<div style="color:var(--fg-faint);line-height:1.4">${d.desc ?? ""}</div>` +
          `</div>`,
      )
      .linkColor((l) => LINK_COLORS[l.type] ?? "rgba(148,163,184,0.25)")
      .linkWidth(1.2)
      .d3AlphaDecay(0.022)
      .warmupTicks(80)
      .onNodeClick((d) => {
        setSelected(d);
        onSelect?.(d);
      });
    graphRef.current = fg;

    // 3d-force-graph 默认画布取 window 尺寸，不会自动适配容器：
    // 不显式设置的话球体会以整个窗口中心为圆心，在容器里偏移/被裁切。
    // 这里按容器实际尺寸设置画布，并跟随窗口缩放。
    const resizeGraph = () => {
      const el = containerRef.current;
      if (!el) return;
      const w = el.clientWidth;
      const h = el.clientHeight;
      if (w > 0 && h > 0) {
        fg.width(w).height(h);
      }
    };
    resizeGraph();
    window.addEventListener("resize", resizeGraph);
    // 容器尺寸变化（如侧栏/窗口布局调整）时同步画布。
    let ro: ResizeObserver | null = null;
    if (typeof ResizeObserver !== "undefined") {
      ro = new ResizeObserver(resizeGraph);
      ro.observe(containerRef.current);
    }

    // 数据就绪后首次渲染
    if (dataRef.current) applyFilter(fg, dataRef.current, typeFilter);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    return () => {
      window.removeEventListener("resize", resizeGraph);
      ro?.disconnect();
    };
  }, []);

  // 类型过滤变化 → 重新构图
  useEffect(() => {
    if (graphRef.current && dataRef.current) {
      applyFilter(graphRef.current, dataRef.current, typeFilter);
    }
  }, [typeFilter]);

  function applyFilter(fg: ForceGraph, data: MemoryGraphView, filter: Set<string>) {
    const nodes = (data.nodes ?? []).filter((n) => filter.has(n.type));
    const ids = new Set(nodes.map((n) => n.id));
    const links = (data.links ?? []).filter((l) => ids.has(l.source) && ids.has(l.target));
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
        onCancel={() => {
          setSelected(null);
          onSelect?.(null);
        }}
        footer={null}
        width={480}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
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
              来自：{selected.type === "knowledge" ? "知识库" : selected.type === "profile" ? "用户画像" : selected.type === "office" ? "办公记忆" : selected.type === "material" ? "项目资料（固定常用文件）" : "聊天记忆"}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
