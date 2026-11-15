import { useEffect, useRef, useState, type CSSProperties } from "react";
import ForceGraph3D from "3d-force-graph";
import type { ForceGraph3DInstance } from "3d-force-graph";
import { Modal, Spin } from "antd";
import { ArrowRightOutlined, DatabaseOutlined, ReloadOutlined } from "@ant-design/icons";
import { app } from "../../lib/bridge";
import { DOMAIN_COLORS, DOMAIN_KEYS, DOMAIN_LABELS } from "../../lib/domainColors";
import type { GraphLink, GraphNode, MemoryGraphView, SemanticGraphView } from "../../lib/types";

const TYPE_COLORS = DOMAIN_COLORS;
const TYPE_LABELS = DOMAIN_LABELS;
const TYPE_KEYS = DOMAIN_KEYS;
const SEMANTIC_TYPES = new Set(["entity", "event", "source"]);
const LINK_COLORS: Record<string, string> = {
  "same-tag": "rgba(129,140,248,0.30)",
  "same-category": "rgba(52,211,153,0.30)",
  reference: "rgba(244,114,182,0.40)",
  // 语义图谱（source="semantic"）三向边：来源→事件→实体 + 实体互引。
  produces: "rgba(251,146,60,0.30)",
  affects: "rgba(148,163,184,0.25)",
  references: "rgba(94,234,212,0.45)",
};

// 3d-force-graph 运行时导出是可调用 kapsule 工厂：ForceGraph3D()(domEl) → 链式实例；
// 包自带类型（1.76.0）却把默认导出声明成 `new` 构造器，与真实调用面不符。
// 这里用包内导出的 ForceGraph3DInstance 泛型类型精确桥接工厂签名（T6-10.2 类型清零）。
type ForceGraph = ForceGraph3DInstance<GraphNode, GraphLink>;
const createGraph = ForceGraph3D as unknown as () => (element: HTMLElement) => ForceGraph;

type GraphData = MemoryGraphView | SemanticGraphView;

/** GraphView 记忆 3D 图谱：节点=记忆实体，边=同标签/同分类/[[引用]]。
 *  variant="page" 带工具条（库面板内）；variant="home" 纯净展示（首页中央）。
 *  onSelect 可选回调：节点被点击 / 详情关闭时通知外层（详情 inspector 联动，不影响原有 Modal）。
 *  source="cloud"（默认）按标签/分类拼关联图；source="semantic" 读事件日志投影
 *  的语义图谱（entity/event/source 三向边，GaeaMemorySemanticGraph）。
 *  onEmptyAction 可选回调：空态入口（进入知识库添加第一条记忆）。 */
export function GraphView(p: {
  variant?: "page" | "home";
  onSelect?: (node: GraphNode | null) => void;
  source?: "cloud" | "semantic";
  sourceSwitch?: React.ReactNode;
  onEmptyAction?: () => void;
}) {
  const { variant = "page", onSelect, source = "cloud", sourceSwitch, onEmptyAction } = p;
  const containerRef = useRef<HTMLDivElement>(null);
  const graphRef = useRef<ForceGraph | null>(null);
  const dataRef = useRef<GraphData | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [reloadKey, setReloadKey] = useState(0);
  const [typeFilter, setTypeFilter] = useState<Set<string>>(new Set([...TYPE_KEYS]));
  const [selected, setSelected] = useState<GraphNode | null>(null);
  const [nodeCount, setNodeCount] = useState(0);
  const [linkCount, setLinkCount] = useState(0);
  const visibleTypeKeys = TYPE_KEYS.filter((t) =>
    source === "semantic" ? SEMANTIC_TYPES.has(t) : !SEMANTIC_TYPES.has(t),
  );

  // 拉取图谱数据（cloud=关联图聚合；semantic=事件日志投影语义图）
  useEffect(() => {
    setLoading(true);
    setLoadError(false);
    const fetchGraph = source === "semantic" ? app.SemanticGraph() : app.MemoryGraph();
    fetchGraph
      .then((g: GraphData) => {
        dataRef.current = g;
        setNodeCount((g.nodes ?? []).length);
        setLinkCount((g.links ?? []).length);
        setLoading(false);
        // 数据到达 → 若图已初始化立即构图（修复：此前数据返回后无任何触发，图谱空白）
        if (graphRef.current) {
          applyFilter(graphRef.current, g, typeFilter);
        }
      })
      .catch(() => {
        dataRef.current = null;
        setNodeCount(0);
        setLinkCount(0);
        setLoadError(true);
        setLoading(false);
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source, reloadKey]);

  // 初始化 ForceGraph3D（once）
  useEffect(() => {
    if (!containerRef.current || graphRef.current) return;
    const fg = createGraph()(containerRef.current)
      // WebGL 画布透明，主题背景由 hub.css 的画布层负责。这样 12 套主题切换时
      // CSS 与三维场景永远同源，不会残留上一套主题的整块背景色。
      .backgroundColor("rgba(0,0,0,0)")
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
    return () => {
      window.removeEventListener("resize", resizeGraph);
      ro?.disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 图初始化 effect 只应运行一次：typeFilter 为随过滤变化的状态、onSelect 为父组件回调、applyFilter 为普通函数，均非稳定值；补依赖会在其变化时先执行 cleanup（断开 ResizeObserver、移除 resize 监听）再因 graphRef 已存在而早退，画布将失去自适应
  }, []);

  // 类型过滤变化 → 重新构图
  useEffect(() => {
    if (graphRef.current && dataRef.current) {
      applyFilter(graphRef.current, dataRef.current, typeFilter);
    }
  }, [typeFilter]);

  function applyFilter(fg: ForceGraph, data: GraphData, filter: Set<string>) {
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
    <div className="hub-graph">
      {/* 工具条（home 模式隐藏，纯净展示） */}
      {variant === "page" && (
      <div className="hub-graph-toolbar">
        <div className="hub-graph-title">{source === "semantic" ? "记忆事件图谱" : "记忆 3D 图谱"}</div>
        <span className="hub-graph-count">
          {nodeCount} 节点 · {linkCount} 边
        </span>
        {sourceSwitch}
        <div className="hub-graph-filters">
          {visibleTypeKeys.map((t) => (
            <button
              key={t}
              type="button"
              aria-pressed={typeFilter.has(t)}
              onClick={() => toggleType(t)}
              className={`hub-graph-filter${typeFilter.has(t) ? " is-active" : ""}`}
              style={{ "--filter-color": TYPE_COLORS[t] } as CSSProperties}
            >
              <span style={{ background: TYPE_COLORS[t] }} />
              {TYPE_LABELS[t]}
            </button>
          ))}
        </div>
      </div>
      )}

      {/* 图区 */}
      <div className={`hub-graph-stage${variant === "page" ? " is-page" : " is-home"}`}>
        {loading && (
          <div className="hub-graph-loading" role="status">
            <span className="hub-graph-loader" aria-hidden="true" />
            <Spin />
            <strong>构建记忆图谱</strong>
            <small>正在整理节点与关联关系…</small>
          </div>
        )}
        <div ref={containerRef} className="hub-graph-canvas" aria-label="记忆 3D 关系图" />

        {!loading && loadError && (
          <div className="hub-graph-empty is-error" role="alert">
            <span className="hub-graph-empty-mark" aria-hidden="true"><DatabaseOutlined /></span>
            <span className="hub-kicker">CONNECTION LOST</span>
            <h3>暂时无法读取图谱</h3>
            <p>本地记忆仍保存在原处。重新连接后即可继续探索。</p>
            <button type="button" onClick={() => setReloadKey((v) => v + 1)}>
              <ReloadOutlined aria-hidden="true" />
              重新加载
            </button>
          </div>
        )}

        {!loading && !loadError && nodeCount === 0 && (
          <div className="hub-graph-empty">
            <div className="hub-graph-orbit" aria-hidden="true">
              <span className="hub-graph-orbit-ring hub-graph-orbit-ring--one" />
              <span className="hub-graph-orbit-ring hub-graph-orbit-ring--two" />
              <span className="hub-graph-orbit-node hub-graph-orbit-node--a" />
              <span className="hub-graph-orbit-node hub-graph-orbit-node--b" />
              <span className="hub-graph-orbit-node hub-graph-orbit-node--c" />
              <span className="hub-graph-orbit-core"><DatabaseOutlined /></span>
            </div>
            <span className="hub-kicker">THE FIRST THREAD</span>
            <h3>记忆还未连成脉络</h3>
            <p>在对话里保存一条偏好，或从知识库添加第一条资料。新的关系会在这里自然展开。</p>
            {onEmptyAction && (
              <button type="button" onClick={onEmptyAction}>
                从知识库开始
                <ArrowRightOutlined aria-hidden="true" />
              </button>
            )}
          </div>
        )}

        {!loading && !loadError && nodeCount > 0 && (
          <div className="hub-graph-hint">
            拖拽旋转 · 滚轮缩放 · hover 高亮
          </div>
        )}
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
