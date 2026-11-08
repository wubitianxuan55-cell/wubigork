import React, { Suspense, useEffect, useMemo, useRef, useState } from "react";
import {
  AimOutlined,
  CaretRightOutlined,
  CloseOutlined,
  DatabaseOutlined,
  FileSearchOutlined,
  HeartOutlined,
  HomeOutlined,
  LeftOutlined,
  NodeIndexOutlined,
  PauseOutlined,
  ReloadOutlined,
  RightOutlined,
  RobotOutlined,
  SearchOutlined,
  SettingOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import { BookOpen, Brain, FileText, Pin } from "../gaea/icons";
import { LocaleProvider } from "../gaea/lib/i18n";
import { app } from "../gaea/lib/bridge";
import { SCOPE_OPTIONS, useSpaceScope } from "../gaea/lib/useSpaceScope";
import { useComposerInsertStore, usePreviewStore } from "../gaea/lib/store";
import { DOMAIN_COLORS, DOMAIN_LABELS } from "../gaea/lib/domainColors";
import type { GraphLink, GraphNode, MemoryHubOverview, SearchScope } from "../gaea/lib/types";
import { ATLAS_NODE_LIMIT } from "../gaea/components/memoryhub/atlasConfig";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "../gaea/components/memoryhub/hub.css";
import "../gaea/components/memoryhub/atlas.css";

// ── 各库面板懒加载（沿用 H2 拆包：主 chunk 不背 3d-force-graph / Markdown）──
const KnowledgePanel = React.lazy(() =>
  import("../gaea/components/KnowledgePanel").then((m) => ({ default: m.KnowledgePanel })),
);
const ProfileLibrary = React.lazy(() =>
  import("../gaea/components/memoryhub/ProfileLibrary").then((m) => ({ default: m.ProfileLibrary })),
);
const OfficeMemoryLibrary = React.lazy(() =>
  import("../gaea/components/memoryhub/OfficeMemoryLibrary").then((m) => ({ default: m.OfficeMemoryLibrary })),
);
const WhisperMemoryLibrary = React.lazy(() =>
  import("../gaea/components/memoryhub/WhisperMemoryLibrary").then((m) => ({ default: m.WhisperMemoryLibrary })),
);
const GraphView = React.lazy(() =>
  import("../gaea/components/memoryhub/GraphView").then((m) => ({ default: m.GraphView })),
);
const MaterialsLibrary = React.lazy(() =>
  import("../gaea/components/memoryhub/MaterialsLibrary").then((m) => ({ default: m.MaterialsLibrary })),
);
const DigitalLifeLibrary = React.lazy(() =>
  import("../gaea/components/memoryhub/DigitalLifeLibrary").then((m) => ({ default: m.DigitalLifeLibrary })),
);
const DatabaseLibrary = React.lazy(() =>
  import("../gaea/components/memoryhub/DatabaseLibrary").then((m) => ({ default: m.DatabaseLibrary })),
);
const MemoryAtlasStage = React.lazy(() =>
  import("../gaea/components/memoryhub/MemoryAtlasStage").then((m) => ({ default: m.MemoryAtlasStage })),
);
const FilePreviewModal = React.lazy(() =>
  import("../gaea/components/FilePreviewModal").then((m) => ({ default: m.FilePreviewModal })),
);

type LibraryKey = "knowledge" | "profile" | "office" | "materials" | "whisper" | "graph" | "digitallife" | "database";

// 各库霓虹色（与 3D 图谱/领域色同源；仅用于识别，不承担主题语义）
const LIB_COLORS: Record<LibraryKey, string> = {
  knowledge: DOMAIN_COLORS.knowledge,
  profile: DOMAIN_COLORS.profile,
  office: DOMAIN_COLORS.office,
  materials: DOMAIN_COLORS.material,
  whisper: DOMAIN_COLORS.whisper,
  graph: "#22d3ee", // hex-exempt：图谱入口识别色
  digitallife: "#fb7185", // hex-exempt：数字生命识别色
  database: DOMAIN_COLORS.database,
};

interface LibraryDef {
  key: LibraryKey;
  label: string;
  icon: React.ReactNode;
  hint: string;
  en: string;
}

const LIBRARIES: LibraryDef[] = [
  { key: "knowledge", label: "知识库", icon: <BookOpen size={14} />, hint: "规范 / 案例 / 经验条目", en: "KNOWLEDGE" },
  { key: "profile", label: "用户画像", icon: <Brain size={14} />, hint: "跨板块共享画像", en: "PROFILE" },
  { key: "office", label: "办公记忆", icon: <FileText size={14} />, hint: "跨会话办公事实", en: "OFFICE" },
  { key: "materials", label: "项目资料", icon: <Pin size={14} />, hint: "固定常用文件", en: "MATERIALS" },
  { key: "whisper", label: "聊天记忆", icon: <HeartOutlined style={{ fontSize: 13 }} />, hint: "轻语人格记忆 · 只读", en: "WHISPER" },
  { key: "graph", label: "记忆图谱", icon: <NodeIndexOutlined style={{ fontSize: 13 }} />, hint: "3D 关系图谱", en: "GRAPH" },
  { key: "digitallife", label: "数字生命", icon: <RobotOutlined style={{ fontSize: 13 }} />, hint: "Herdsman 虚拟人格", en: "DIGITAL LIFE" },
  { key: "database", label: "数据库", icon: <DatabaseOutlined style={{ fontSize: 13 }} />, hint: "本设备聊天记录 · 上传文件", en: "ARCHIVE" },
];

const MEMORY_LIBS = LIBRARIES.slice(0, 5);
const EXPLORE_LIBS = LIBRARIES.slice(5);

const FALLBACK_NODE_COLOR = "#94a3b8"; // hex-exempt：未知类型节点的回退色

const TYPE_ORDER = ["knowledge", "profile", "office", "whisper", "material", "cost", "database"];

const NODE_LABELS: Record<string, string> = {
  ...DOMAIN_LABELS,
  material: "项目资料",
  database: "数据库",
};

// 右栏 tabs 的短标签（窄栏里 7 个域要挤进一行）。
const TAB_LABELS: Record<string, string> = {
  knowledge: "知识",
  profile: "画像",
  office: "办公",
  whisper: "聊天",
  material: "资料",
  cost: "成本",
  database: "存档",
  entity: "实体",
  event: "事件",
  source: "来源",
};

// 三脑检索 + 工作区全文搜索的合并命中。
interface HubSearchHit {
  kind: "brain" | "file";
  brain: string;
  entity: string;
  text: string;
  path?: string;
}

type Detail =
  | { kind: "node"; node: GraphNode }
  | { kind: "hit"; hit: HubSearchHit };

function HubViewFallback() {
  return (
    <div className="atlas-library-loading" role="status" aria-label="视图加载中">
      <span className="atlas-spinner" aria-hidden="true" />
      <span>视图加载中…</span>
    </div>
  );
}

function formatCount(n: number | undefined | null): string {
  if (n == null) return "···";
  return n.toLocaleString();
}

function sourceOf(type: string): string {
  const lib = LIBRARIES.find((l) => l.key === type || (type === "material" && l.key === "materials"));
  return lib?.label ?? NODE_LABELS[type] ?? type;
}

function brainLabel(brain: string): string {
  if (brain === "brain.left") return "左脑";
  if (brain === "brain.right") return "右脑";
  if (brain === "brain.main") return "主脑";
  return brain;
}

function libraryForType(type: string): LibraryKey | null {
  if (type === "material" || type === "materials") return "materials";
  if (type === "cost") return null;
  const hit = LIBRARIES.find((l) => l.key === type);
  return hit?.key ?? null;
}

/** 记忆中枢 · Living Constellation：左索引 | 中央活体星图 | 右记忆流。 */
function MemoryHubPage() {
  const [active, setActive] = useState<"home" | LibraryKey>("home");
  const [overview, setOverview] = useState<MemoryHubOverview | null>(null);
  const [overviewError, setOverviewError] = useState(false);
  const [overviewKey, setOverviewKey] = useState(0);

  const [graphNodes, setGraphNodes] = useState<GraphNode[]>([]);
  const [graphLinks, setGraphLinks] = useState<GraphLink[]>([]);
  const [graphLoading, setGraphLoading] = useState(true);
  const [graphError, setGraphError] = useState(false);
  const [graphKey, setGraphKey] = useState(0);

  const [brainQuery, setBrainQuery] = useState("");
  const [hits, setHits] = useState<HubSearchHit[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchOutcome, setSearchOutcome] = useState<"idle" | "empty" | "error">("idle");
  const searchSeqRef = useRef(0);
  const [referenced, setReferenced] = useState<string | null>(null);

  const [detail, setDetail] = useState<Detail | null>(null);
  const [typeFilter, setTypeFilter] = useState<string>("all");
  // 减少动态（设置里的 ui-reduced-motion）默认冻结星图；用户仍可手动继续流转。
  const [reducedMotion] = useState(
    () => typeof document !== "undefined" && !!document.querySelector(".ui-reduced-motion"),
  );
  const [paused, setPaused] = useState(reducedMotion);
  const [autoRotate, setAutoRotate] = useState(!reducedMotion);
  useEffect(() => {
    // 设置里切换「减少动态」时，页面常驻（keepAlive）不会重挂；观察壳层 class 跟上。
    const root = typeof document !== "undefined" ? document.getElementById("root") : null;
    if (!root || typeof MutationObserver === "undefined") return;
    const sync = () => {
      if (document.querySelector(".ui-reduced-motion")) setPaused(true);
    };
    const obs = new MutationObserver(sync);
    obs.observe(root, { attributes: true, subtree: true, attributeFilter: ["class"] });
    return () => obs.disconnect();
  }, []);
  const [resetSignal, setResetSignal] = useState(0);
  const [showRings, setShowRings] = useState(true);
  const [showDust, setShowDust] = useState(true);
  const [settingsOpen, setSettingsOpen] = useState(false);

  const [scope, setScope] = useState<SearchScope>("work");
  const scopeTouched = useRef(false);
  const { active: spaceActive } = useSpaceScope();
  useEffect(() => {
    if (spaceActive && !scopeTouched.current) {
      setScope(spaceActive.space === "play" ? "play" : "work");
    }
  }, [spaceActive]);

  const [graphSource, setGraphSource] = useState<"cloud" | "semantic">("cloud");

  // 总览
  useEffect(() => {
    setOverviewError(false);
    app.MemoryHubOverview().then(setOverview).catch(() => setOverviewError(true));
  }, [overviewKey]);

  // 图谱（活体星图的真实节点源）
  useEffect(() => {
    setGraphLoading(true);
    setGraphError(false);
    app.MemoryGraph()
      .then((g) => {
        setGraphNodes(g?.nodes ?? []);
        setGraphLinks(g?.links ?? []);
      })
      .catch(() => {
        setGraphNodes([]);
        setGraphLinks([]);
        setGraphError(true);
      })
      .finally(() => setGraphLoading(false));
  }, [graphKey]);

  const runSearch = async () => {
    const q = brainQuery.trim();
    if (!q) {
      setHits([]);
      setSearchOutcome("idle");
      return;
    }
    const seq = ++searchSeqRef.current;
    setSearching(true);
    setDetail(null);
    try {
      const v = await app.UnifiedSearch(q, 8, scope).catch(() => null);
      if (seq !== searchSeqRef.current) return;
      if (!v) {
        setHits([]);
        setSearchOutcome("error");
        return;
      }
      const kindLabel: Record<string, string> = { cost: "语义·成本", knowledge: "语义·知识", office: "语义·办公", file: "语义·资料" };
      const merged: HubSearchHit[] = [
        ...(v.brain ?? []).map((h) => ({ kind: "brain" as const, brain: h.brain, entity: h.entity, text: h.text })),
        ...(v.keyword ?? []).map((f) => ({
          kind: "file" as const, brain: "文件", entity: f.name, text: f.snippet || f.path, path: f.path,
        })),
        ...(v.semantic ?? []).map((h) => ({
          kind: "brain" as const, brain: kindLabel[h.kind] ?? "语义", entity: h.name, text: h.text,
        })),
        ...(v.files ?? []).map((h) => ({
          kind: "file" as const, brain: "语义·文件", entity: h.path.split("/").pop() ?? h.path, text: h.snippet, path: h.path,
        })),
      ];
      setHits(merged);
      setSearchOutcome(merged.length === 0 ? "empty" : "idle");
    } catch {
      if (seq !== searchSeqRef.current) return;
      setHits([]);
      setSearchOutcome("error");
    } finally {
      if (seq === searchSeqRef.current) setSearching(false);
    }
  };

  const referenceFile = (path: string) => {
    useComposerInsertStore.getState().requestAt(path);
    setReferenced(path);
    setTimeout(() => setReferenced((p) => (p === path ? null : p)), 1500);
  };

  const selectCategory = (key: "home" | LibraryKey) => {
    setActive(key);
    setDetail(null);
    setHits([]);
    setSearchOutcome("idle");
    setSettingsOpen(false);
    if (key === "home") {
      // 从库页回星图时刷新一次，避免刚编辑的记忆在星图上迟到。
      setGraphKey((v) => v + 1);
      setOverviewKey((v) => v + 1);
      setTypeFilter("all");
    }
  };

  const counts: Partial<Record<LibraryKey, number | "—" | "…">> = overviewError
    ? { knowledge: "—", profile: "—", office: "—", materials: "—", whisper: "—", database: "—" }
    : overview
    ? {
        knowledge: overview.knowledgeCount,
        profile: overview.profileCount,
        office: overview.officeCount,
        materials: overview.pinnedCount,
        whisper: overview.whisperCount,
        database: overview.sessionCount,
      }
    : { knowledge: "…", profile: "…", office: "…", materials: "…", whisper: "…", database: "…" };

  const totalOverview = overview
    ? overview.knowledgeCount + overview.profileCount + overview.officeCount + overview.pinnedCount + overview.whisperCount + overview.costCount
    : null;
  // 星图上的真实节点数优先（它包含成本等未单列库的域），失败时回退总览。
  const visualTotal = graphNodes.length || totalOverview;
  const visibleNodeCount = Math.min(graphNodes.length, ATLAS_NODE_LIMIT);
  const linksCount = graphLinks.length;

  const degree = useMemo(() => {
    const m = new Map<string, number>();
    for (const l of graphLinks) {
      m.set(String(l.source), (m.get(String(l.source)) ?? 0) + 1);
      m.set(String(l.target), (m.get(String(l.target)) ?? 0) + 1);
    }
    return m;
  }, [graphLinks]);

  const typeCounts = useMemo(() => {
    const m = new Map<string, number>();
    for (const n of graphNodes) m.set(n.type, (m.get(n.type) ?? 0) + 1);
    return m;
  }, [graphNodes]);

  const filterTypes = useMemo(() => {
    const present = TYPE_ORDER.filter((t) => (typeCounts.get(t) ?? 0) > 0);
    const extra = [...typeCounts.keys()].filter((t) => !TYPE_ORDER.includes(t));
    return [...present, ...extra];
  }, [typeCounts]);

  const sortedNodes = useMemo(() => {
    return [...graphNodes].sort((a, b) => {
      const da = degree.get(a.id) ?? 0;
      const db = degree.get(b.id) ?? 0;
      if (db !== da) return db - da;
      return (b.val ?? 0) - (a.val ?? 0);
    });
  }, [graphNodes, degree]);

  const filteredNodes = useMemo(() => {
    return typeFilter === "all" ? sortedNodes : sortedNodes.filter((n) => n.type === typeFilter);
  }, [sortedNodes, typeFilter]);

  const streamNodes = useMemo(() => filteredNodes.slice(0, 60), [filteredNodes]);

  const selectedNodeId = detail?.kind === "node" ? detail.node.id : null;

  const handleStageSelect = (id: string | null) => {
    if (!id) {
      setDetail(null);
      return;
    }
    const node = graphNodes.find((n) => n.id === id);
    if (node) setDetail({ kind: "node", node });
  };

  const openNode = (node: GraphNode) => {
    setDetail({ kind: "node", node });
  };

  const openLibraryForNode = (node: GraphNode) => {
    const lib = libraryForType(node.type);
    if (lib) selectCategory(lib);
    else selectCategory("knowledge");
  };

  const openHit = (h: HubSearchHit) => setDetail({ kind: "hit", hit: h });

  const lib = active === "home" ? null : LIBRARIES.find((l) => l.key === active) ?? null;

  const graphSourceSwitch = (
    <div className="hub-source-switch" role="group" aria-label="图谱事实源">
      {([["cloud", "关联图"], ["semantic", "事件图谱"]] as const).map(([v, label]) => (
        <button
          key={v}
          type="button"
          aria-pressed={graphSource === v}
          onClick={() => setGraphSource(v)}
          className={graphSource === v ? "is-active" : undefined}
        >
          {label}
        </button>
      ))}
    </div>
  );

  const composition = MEMORY_LIBS.map((l) => {
    const c = counts[l.key];
    const n = typeof c === "number" ? c : 0;
    return { ...l, n };
  });
  const compositionTotal = composition.reduce((s, x) => s + x.n, 0);

  const renderHitRow = (h: HubSearchHit, i: number) => (
    <div
      key={`${h.kind}:${h.brain}:${h.entity}:${i}`}
      className="hub-hit"
      role="button"
      tabIndex={0}
      onClick={() => openHit(h)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          openHit(h);
        }
      }}
    >
      <span className={`hub-hit-tag ${
        h.kind === "file"
          ? "hub-hit-tag--file"
          : h.brain === "brain.right"
            ? "hub-hit-tag--right"
            : h.brain === "brain.left"
              ? "hub-hit-tag--left"
              : "hub-hit-tag--main"
      }`}>
        {h.kind === "file" ? "文件" : h.brain === "brain.right" ? "右脑" : h.brain === "brain.left" ? "左脑" : "主脑"}
      </span>
      {h.kind === "file" && h.path ? (
        <>
          <button
            type="button"
            className="hub-hit-entity"
            onClick={(e) => { e.stopPropagation(); usePreviewStore.getState().openFilePreview(h.path!); }}
            title="点击预览"
          >
            {h.entity}
          </button>
          <span className="hub-hit-text">{h.text}</span>
          <button
            type="button"
            className={`hub-hit-ref${referenced === h.path ? " is-referenced" : ""}`}
            onClick={(e) => { e.stopPropagation(); referenceFile(h.path!); }}
            title="一键 @ 引用（回到办公板块自动插入输入框）"
          >
            {referenced === h.path ? "已引用" : "@ 引用"}
          </button>
        </>
      ) : (
        <>
          <span className="hub-hit-entity-static">{h.entity}</span>
          <span className="hub-hit-text">{h.text}</span>
        </>
      )}
    </div>
  );

  const detailView = detail ? (
    <div className="atlas-detail">
      <div className="atlas-detail-head">
        <button type="button" className="atlas-detail-back" onClick={() => setDetail(null)} aria-label="返回记忆流">
          <LeftOutlined aria-hidden="true" />
        </button>
        <span className="atlas-detail-kicker">ENTRY CONTEXT</span>
      </div>
      {detail.kind === "node" ? (
        <div className="hub-detail">
          <div className="hub-detail-head">
            <span
              className="hub-detail-dot"
              style={{ background: LIB_COLORS[(libraryForType(detail.node.type) ?? "knowledge") as LibraryKey] }}
            />
            <span className="hub-detail-type">{NODE_LABELS[detail.node.type] ?? detail.node.type}</span>
          </div>
          <div className="hub-detail-title">{detail.node.name}</div>
          {detail.node.desc && <div className="hub-detail-desc">{detail.node.desc}</div>}
          <div className="hub-detail-row">
            <span className="hub-detail-key">ID</span>
            <span className="hub-detail-val mono">{detail.node.id}</span>
          </div>
          <div className="hub-detail-row">
            <span className="hub-detail-key">来源</span>
            <span className="hub-detail-val">{sourceOf(detail.node.type)}</span>
          </div>
          <div className="hub-detail-row">
            <span className="hub-detail-key">关联</span>
            <span className="hub-detail-val">{degree.get(detail.node.id) ?? 0} 条</span>
          </div>
          <div className="hub-detail-actions">
            <button type="button" className="hub-detail-action" onClick={() => openLibraryForNode(detail.node)}>
              打开所属库
            </button>
            <button type="button" className="hub-detail-action" onClick={() => selectCategory("graph")}>
              在图谱中查看
            </button>
          </div>
        </div>
      ) : (
        <div className="hub-detail">
          <div className="hub-detail-head">
            <span className="hub-detail-badge">{brainLabel(detail.hit.brain)}</span>
          </div>
          <div className="hub-detail-title">{detail.hit.entity}</div>
          {detail.hit.text && <div className="hub-detail-desc">{detail.hit.text}</div>}
          {detail.hit.path && (
            <div className="hub-detail-row">
              <span className="hub-detail-key">路径</span>
              <span className="hub-detail-val mono">{detail.hit.path}</span>
            </div>
          )}
          {detail.hit.path && (
            <div className="hub-detail-actions">
              <button
                type="button"
                className="hub-detail-action"
                onClick={() => usePreviewStore.getState().openFilePreview(detail.hit.path!)}
                title="打开文件预览"
              >
                <FileSearchOutlined aria-hidden="true" style={{ fontSize: 11 }} />
                预览
              </button>
              <button
                type="button"
                className={`hub-detail-action${referenced === detail.hit.path ? " is-referenced" : ""}`}
                onClick={() => referenceFile(detail.hit.path!)}
                title="一键 @ 引用（回到办公板块自动插入输入框）"
              >
                {referenced === detail.hit.path ? "已引用" : "@ 引用"}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  ) : null;

  const searchActive = searching || hits.length > 0 || searchOutcome !== "idle";

  return (
    <div className={`atlas-page${active === "home" ? "" : " is-library"}`}>
      <div className="atlas-ambience" aria-hidden="true" />
      <Suspense
        fallback={(
          <div className="atlas-stage atlas-stage-loading" role="status" aria-label="星图加载中">
            <span className="atlas-spinner" aria-hidden="true" />
            <span>正在构建 3D 脑网格…</span>
          </div>
        )}
      >
        <MemoryAtlasStage
          nodes={graphNodes}
          links={graphLinks}
          selectedId={selectedNodeId}
          paused={paused || active !== "home"}
          autoRotate={autoRotate}
          resetSignal={resetSignal}
          showRings={showRings}
          showDust={showDust}
          onSelect={handleStageSelect}
        />
      </Suspense>
      <div className="atlas-bloom" aria-hidden="true" />
      <div className="atlas-vignette" aria-hidden="true" />
      <div className="atlas-grain" aria-hidden="true" />

      <div className="atlas-lobe-labels" aria-hidden="true">
        <div className="atlas-lobe-label is-left">
          <span>LEFT HEMISPHERE</span>
          <strong>事实 · 知识 · 办公</strong>
        </div>
        <div className="atlas-lobe-label is-right">
          <span>RIGHT HEMISPHERE</span>
          <strong>自我 · 画像 · 聊天</strong>
        </div>
      </div>

      <header className="atlas-brand">
        <span className="atlas-live" aria-hidden="true" />
        <span className="atlas-brand-kicker">A LIVING CONSTELLATION</span>
        <span className="atlas-brand-rule" aria-hidden="true" />
        <span className="atlas-brand-local">LOCAL</span>
        <span className="atlas-brand-sub">记忆中枢</span>
      </header>

      <aside className="atlas-index" aria-label="记忆中枢索引">
        <div className="atlas-index-scroll">
          <div className="atlas-hero">
            <h1>每一段记忆，<br />都有自己的光。</h1>
            <p>从片段到关系，<br />看见正在生长的记忆。</p>
          </div>

          <div className="atlas-stats">
            <div className="atlas-stat">
              <strong>{overviewError ? "—" : graphLoading ? "···" : formatCount(visualTotal)}</strong>
              <small>记忆节点</small>
            </div>
            <div className="atlas-stat">
              <strong>{graphError ? "—" : graphLoading ? "···" : formatCount(linksCount)}</strong>
              <small>关联实体</small>
            </div>
            <div className="atlas-stat">
              <strong>{LIBRARIES.length}</strong>
              <small>记忆库</small>
            </div>
          </div>

          <div className="atlas-search-block">
            <div className="atlas-search-caption">
              <span>探索</span>
              <small>三脑 + 工作区资料</small>
            </div>
            <div className="atlas-search-row">
              <SearchOutlined className="atlas-search-icon" aria-hidden="true" />
              <input
                value={brainQuery}
                onChange={(e) => {
                  setBrainQuery(e.target.value);
                  if (!e.target.value.trim()) { setHits([]); setSearchOutcome("idle"); }
                }}
                onKeyDown={(e) => { if (e.key === "Enter") runSearch(); }}
                placeholder="三脑检索 · 工作区资料"
                aria-label="三脑检索 · 工作区资料"
              />
              <span className="atlas-search-key" aria-hidden="true">ENTER</span>
              <button onClick={runSearch} disabled={searching} className="atlas-search-btn" type="button">
                {searching ? "检索中" : "检索"}
              </button>
            </div>
            <div className="atlas-scope-row">
              <span className="atlas-scope-label">范围</span>
              <div className="atlas-scope-switch" role="radiogroup" aria-label="检索范围">
                {SCOPE_OPTIONS.map((o) => (
                  <button
                    key={o.label}
                    type="button"
                    role="radio"
                    aria-checked={scope === o.value}
                    className={scope === o.value ? "is-active" : undefined}
                    title={o.title}
                    onClick={() => { scopeTouched.current = true; setScope(o.value); }}
                  >
                    {o.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <section className="atlas-nav-section" aria-label="记忆集合">
            <div className="atlas-section-head">
              <span>记忆集合</span>
              <small>COLLECTIONS</small>
            </div>
            <nav className="atlas-nav">
              <button
                type="button"
                className={`atlas-nav-row${active === "home" ? " is-active" : ""}`}
                onClick={() => selectCategory("home")}
                aria-current={active === "home" ? "page" : undefined}
                style={{ "--atlas-row": "var(--atlas-cyan)" } as React.CSSProperties}
              >
                <HomeOutlined aria-hidden="true" />
                <span className="atlas-nav-label">总览星图</span>
                <small>ATLAS</small>
                <i className="atlas-nav-dot" aria-hidden="true" />
              </button>
              {MEMORY_LIBS.map((l) => (
                <button
                  key={l.key}
                  type="button"
                  className={`atlas-nav-row${active === l.key ? " is-active" : ""}`}
                  onClick={() => selectCategory(l.key)}
                  title={`${l.label} · ${l.hint}`}
                  style={{ "--atlas-row": LIB_COLORS[l.key] } as React.CSSProperties}
                >
                  {l.icon}
                  <span className="atlas-nav-label">{l.label}</span>
                  <span className="atlas-nav-count">
                    {overviewError ? "—" : counts[l.key] ?? "…"}
                  </span>
                  <i className="atlas-nav-dot" aria-hidden="true" />
                </button>
              ))}
            </nav>
            <div className="atlas-section-head is-sub">
              <span>探索与存档</span>
              <small>EXPLORE</small>
            </div>
            <div className="atlas-explore">
              {EXPLORE_LIBS.map((l) => (
                <button
                  key={l.key}
                  type="button"
                  className={`atlas-explore-chip${active === l.key ? " is-active" : ""}`}
                  onClick={() => selectCategory(l.key)}
                  title={`${l.label} · ${l.hint}`}
                  style={{ "--atlas-row": LIB_COLORS[l.key] } as React.CSSProperties}
                >
                  {l.icon}
                  <span>{l.label}</span>
                  <i aria-hidden="true" />
                </button>
              ))}
            </div>
          </section>

          <section className="atlas-vitals" aria-label="记忆构成">
            <div className="atlas-section-head">
              <span>构成</span>
              <small>COMPOSITION</small>
            </div>
            {composition.map((c) => {
              const pct = compositionTotal > 0 ? Math.max(3, (c.n / compositionTotal) * 100) : 0;
              return (
                <button
                  key={c.key}
                  type="button"
                  className="atlas-vital"
                  onClick={() => selectCategory(c.key)}
                  title={`${c.label}：${c.n} 条`}
                  style={{ "--atlas-row": LIB_COLORS[c.key] } as React.CSSProperties}
                >
                  <span className="atlas-vital-label">{c.label}</span>
                  <span className="atlas-vital-bar" aria-hidden="true">
                    <i style={{ width: `${pct}%` }} />
                  </span>
                  <span className="atlas-vital-value">{overviewError ? "—" : c.n}</span>
                </button>
              );
            })}
          </section>
        </div>

        <div className="atlas-index-foot">
          <span className="atlas-live" aria-hidden="true" />
          <span>{graphLoading ? "正在接入记忆…" : graphError ? "星图信号中断" : `${formatCount(visibleNodeCount)} 个可视节点`}</span>
          <span className="atlas-foot-sep" aria-hidden="true">/</span>
          <span>{overview?.latestUpdated || "本地自持"}</span>
          {overviewError && (
            <button
              type="button"
              className="atlas-foot-retry"
              onClick={() => setOverviewKey((v) => v + 1)}
            >
              重试
            </button>
          )}
        </div>
      </aside>

      <aside className={`atlas-stream${active === "home" ? "" : " is-hidden"}`} aria-label="记忆流">
        <div className="atlas-stream-head">
          <div>
            <span className="atlas-stream-kicker">
              {detail ? "INSPECTOR" : searchActive ? "SEARCH SIGNAL" : "MEMORY STREAM"}
            </span>
            <strong>
              {detail
                ? "条目详情"
                : searchActive
                  ? searching
                    ? "正在检索…"
                    : searchOutcome === "error"
                      ? "检索失败"
                      : `检索结果 ${hits.length}`
                  : "记忆流"}
            </strong>
          </div>
          {searchActive && !searching && (
            <button
              type="button"
              className="atlas-icon-btn"
              onClick={() => { setHits([]); setSearchOutcome("idle"); setDetail(null); }}
              aria-label="关闭检索结果"
              title="关闭检索结果"
            >
              <CloseOutlined aria-hidden="true" />
            </button>
          )}
        </div>

        {detail ? (
          <>
            {detailView}
            {detail.kind === "hit" && hits.length > 0 && (
              <div className="atlas-detail-listhint">
                <button type="button" onClick={() => setDetail(null)}>返回检索结果</button>
              </div>
            )}
          </>
        ) : searchActive ? (
          <div className="atlas-stream-body">
            {searching ? (
              <div className="atlas-stream-status" role="status">
                <span className="atlas-search-pulse" aria-hidden="true" />
                <span>正在穿过知识、画像与工作区资料…</span>
              </div>
            ) : searchOutcome === "error" ? (
              <div className="atlas-stream-status is-error" role="status">
                <span className="atlas-status-mark" aria-hidden="true">!</span>
                <span>检索失败，请稍后再试</span>
                <button type="button" onClick={runSearch}>重试</button>
              </div>
            ) : searchOutcome === "empty" ? (
              <div className="atlas-stream-status" role="status">
                <span className="atlas-status-mark" aria-hidden="true">0</span>
                <span>没有找到相关内容，换个关键词试试</span>
              </div>
            ) : (
              <div className="atlas-hit-list">{hits.map(renderHitRow)}</div>
            )}
          </div>
        ) : graphLoading ? (
          <div className="atlas-stream-body">
            <div className="atlas-skeleton-list" aria-hidden="true">
              {Array.from({ length: 7 }).map((_, i) => (
                <div key={i} className="atlas-skeleton-row"><i /><span /><em /></div>
              ))}
            </div>
            <div className="atlas-stream-status" role="status">
              <span className="atlas-search-pulse" aria-hidden="true" />
              <span>正在把记忆送上星图…</span>
            </div>
          </div>
        ) : graphError ? (
          <div className="atlas-stream-body">
            <div className="atlas-stream-status is-error" role="status">
              <span className="atlas-status-mark" aria-hidden="true">!</span>
              <span>暂时无法读取图谱，星图停在最后一帧。</span>
              <button type="button" onClick={() => setGraphKey((v) => v + 1)}>
                <ReloadOutlined aria-hidden="true" /> 重新加载
              </button>
            </div>
          </div>
        ) : (
          <>
            {filterTypes.length > 0 && (
              <div className="atlas-tabs" role="tablist" aria-label="记忆类型">
                <button
                  type="button"
                  role="tab"
                  aria-selected={typeFilter === "all"}
                  className={typeFilter === "all" ? "is-active" : undefined}
                  onClick={() => setTypeFilter("all")}
                >
                  全部 <i>{graphNodes.length}</i>
                </button>
                {filterTypes.map((t) => (
                  <button
                    key={t}
                    type="button"
                    role="tab"
                    aria-selected={typeFilter === t}
                    className={typeFilter === t ? "is-active" : undefined}
                    onClick={() => setTypeFilter(t)}
                  >
                    {TAB_LABELS[t] ?? t} <i>{typeCounts.get(t) ?? 0}</i>
                  </button>
                ))}
              </div>
            )}
            <div className="atlas-stream-body">
              {streamNodes.length === 0 ? (
                <div className="atlas-stream-empty">
                  <div className="atlas-empty-orbit" aria-hidden="true">
                    <span /><span /><span />
                    <DatabaseOutlined />
                  </div>
                  <strong>记忆还未连成脉络</strong>
                  <p>在对话里保存一条偏好，或从知识库添加第一条资料。新的关系会在这里自然展开。</p>
                  <button type="button" onClick={() => selectCategory("knowledge")}>
                    从知识库开始 <RightOutlined aria-hidden="true" />
                  </button>
                </div>
              ) : (
                <ol className="atlas-memory-list">
                  {streamNodes.map((n, i) => {
                    const libKey = libraryForType(n.type);
                    const color = libKey ? LIB_COLORS[libKey] : FALLBACK_NODE_COLOR;
                    return (
                      <li key={n.id}>
                        <button
                          type="button"
                          className="atlas-memory-row"
                          onClick={() => openNode(n)}
                          style={{ "--atlas-row": color } as React.CSSProperties}
                        >
                          <span className="atlas-memory-index">{String(i + 1).padStart(2, "0")}</span>
                          <span className="atlas-memory-copy">
                            <span className="atlas-memory-title">
                              <span>{n.name}</span>
                              <em>{NODE_LABELS[n.type] ?? n.type}</em>
                            </span>
                            <span className="atlas-memory-desc">{n.desc || "这条记忆尚未写下摘要。"}</span>
                          </span>
                          <span className="atlas-memory-degree">{degree.get(n.id) ?? 0}</span>
                        </button>
                      </li>
                    );
                  })}
                </ol>
              )}
            </div>
            <div className="atlas-stream-foot">
              <span>{streamNodes.length} / {filteredNodes.length} 条</span>
              <span>点击星辰或条目查看上下文</span>
            </div>
          </>
        )}
      </aside>

      {active !== "home" && (
        <section className="atlas-library" aria-label={lib?.label ?? "记忆库"}>
          <header className="atlas-library-head">
            <button type="button" className="atlas-library-back" onClick={() => selectCategory("home")}>
              <LeftOutlined aria-hidden="true" /> 返回星图
            </button>
            <div className="atlas-library-title">
              <span className="atlas-library-kicker">{lib?.en ?? "LIBRARY"}</span>
              <h2>{lib?.label ?? "记忆库"}</h2>
              <p>{lib?.hint}</p>
            </div>
            <span className="atlas-library-mark" style={{ color: lib ? LIB_COLORS[lib.key] : undefined }} aria-hidden="true">
              {lib?.icon}
            </span>
          </header>
          <div className="atlas-library-body">
            <Suspense fallback={<HubViewFallback />}>
              <LocaleProvider>
                {active === "knowledge" && (
                  <div className="h-full">
                    <KnowledgePanel variant="page" onClose={() => selectCategory("home")} />
                  </div>
                )}
                {active === "profile" && <ProfileLibrary />}
                {active === "office" && <OfficeMemoryLibrary />}
                {active === "materials" && <MaterialsLibrary />}
                {active === "whisper" && <WhisperMemoryLibrary />}
                {active === "digitallife" && <DigitalLifeLibrary />}
                {active === "database" && <DatabaseLibrary />}
                {active === "graph" && (
                  <div className="atlas-graph-sheet">
                    <GraphView
                      variant="page"
                      source={graphSource}
                      sourceSwitch={graphSourceSwitch}
                      onEmptyAction={() => selectCategory("knowledge")}
                    />
                  </div>
                )}
              </LocaleProvider>
            </Suspense>
          </div>
        </section>
      )}

      {settingsOpen && (
        <div className="atlas-settings" role="group" aria-label="显示设置">
          <div className="atlas-settings-head">
            <span>显示设置</span>
            <button type="button" onClick={() => setSettingsOpen(false)} aria-label="关闭显示设置">
              <CloseOutlined aria-hidden="true" />
            </button>
          </div>
          <button
            type="button"
            className={`atlas-toggle${showRings ? " is-on" : ""}`}
            aria-pressed={showRings}
            onClick={() => setShowRings((v) => !v)}
          >
            <i aria-hidden="true" /><span>轨道环</span><small>ORBIT RINGS</small>
          </button>
          <button
            type="button"
            className={`atlas-toggle${showDust ? " is-on" : ""}`}
            aria-pressed={showDust}
            onClick={() => setShowDust((v) => !v)}
          >
            <i aria-hidden="true" /><span>星尘</span><small>STARDUST</small>
          </button>
          <button
            type="button"
            className={`atlas-toggle${autoRotate ? " is-on" : ""}`}
            aria-pressed={autoRotate}
            onClick={() => setAutoRotate((v) => !v)}
          >
            <i aria-hidden="true" /><span>自动旋转</span><small>AUTO SPIN</small>
          </button>
          <p>拖拽旋转 · 滚轮缩放 · 点击星辰</p>
          <p className="atlas-settings-credit">
            脑网格 ·{" "}
            <a href="https://3d.nih.gov/entries/3DPX-021161" target="_blank" rel="noreferrer">
              NIH 3D 3DPX-021161
            </a>{" "}
            · Johnson J · CC BY 4.0
          </p>
        </div>
      )}

      {active === "home" && (
        <>
          <div className="atlas-dock" role="toolbar" aria-label="星图控制">
            <button
              type="button"
              className={paused ? "is-on" : undefined}
              aria-pressed={paused}
              onClick={() => setPaused((v) => !v)}
              title={paused ? "继续流转" : "暂停流转"}
            >
              {paused ? <CaretRightOutlined aria-hidden="true" /> : <PauseOutlined aria-hidden="true" />}
              <span>{paused ? "继续流转" : "暂停流览"}</span>
            </button>
            <button
              type="button"
              className={autoRotate && !paused ? "is-on" : undefined}
              aria-pressed={autoRotate}
              onClick={() => setAutoRotate((v) => !v)}
              title={autoRotate ? "停止自动旋转" : "开始自动旋转"}
            >
              <SyncOutlined className={autoRotate && !paused ? "is-spinning" : undefined} aria-hidden="true" />
              <span>{autoRotate ? "旋转视角" : "视角锁定"}</span>
            </button>
            <button
              type="button"
              onClick={() => { setResetSignal((v) => v + 1); setAutoRotate(true); }}
              title="重置视角"
            >
              <AimOutlined aria-hidden="true" />
              <span>重置视角</span>
            </button>
            <button
              type="button"
              className={settingsOpen ? "is-on" : undefined}
              aria-pressed={settingsOpen}
              onClick={() => setSettingsOpen((v) => !v)}
              title="显示设置"
            >
              <SettingOutlined aria-hidden="true" />
              <span>显示设置</span>
            </button>
          </div>

          <div className="atlas-stage-foot">
            <span className="atlas-stage-foot-right">
              <span className="atlas-live" aria-hidden="true" />
              LOCAL ATLAS
              <span className="atlas-foot-sep" aria-hidden="true">·</span>
              {formatCount(visibleNodeCount)} NODES
            </span>
          </div>
        </>
      )}

      <Suspense fallback={null}>
        <FilePreviewModal />
      </Suspense>
    </div>
  );
}

export default MemoryHubPage;
