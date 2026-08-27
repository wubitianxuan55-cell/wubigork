import React, { useEffect, useState } from "react";
import {
  CloseOutlined,
  FileSearchOutlined,
  HeartOutlined,
  HomeOutlined,
  LeftOutlined,
  NodeIndexOutlined,
  RightOutlined,
  RobotOutlined,
} from "@ant-design/icons";
import { BookOpen, Brain, FileText, Pin } from "../gaea/icons";
import { LocaleProvider } from "../gaea/lib/i18n";
import { app } from "../gaea/lib/bridge";
import { useComposerInsertStore, usePreviewStore } from "../gaea/lib/store";
import type { GraphNode, MemoryHubOverview, SemanticHitView, WorkspaceSearchHit } from "../gaea/lib/types";
import type { AppFacade } from "../types/wails";
import { KnowledgePanel } from "../gaea/components/KnowledgePanel";
import { ProfileLibrary } from "../gaea/components/memoryhub/ProfileLibrary";
import { OfficeMemoryLibrary } from "../gaea/components/memoryhub/OfficeMemoryLibrary";
import { WhisperMemoryLibrary } from "../gaea/components/memoryhub/WhisperMemoryLibrary";
import { GraphView } from "../gaea/components/memoryhub/GraphView";
import { MaterialsLibrary } from "../gaea/components/memoryhub/MaterialsLibrary";
import { DigitalLifeLibrary } from "../gaea/components/memoryhub/DigitalLifeLibrary";
import { FilePreviewModal } from "../gaea/components/FilePreviewModal";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "../gaea/components/memoryhub/hub.css";

type LibraryKey = "knowledge" | "profile" | "office" | "materials" | "whisper" | "graph" | "digitallife";

// 各库霓虹色（与 3D 图谱着色一致：indigo 知识 / amber 成本 / emerald 办公 / pink 聊天记忆）
const LIB_COLORS: Record<LibraryKey, string> = {
  knowledge: "#818cf8",
  profile: "#a78bfa",
  office: "#34d399",
  materials: "#38bdf8",
  whisper: "#f472b6",
  graph: "#22d3ee",
  digitallife: "#fb7185",
};

interface LibraryDef {
  key: LibraryKey;
  label: string;
  icon: React.ReactNode;
  hint: string;
}

const LIBRARIES: LibraryDef[] = [
  { key: "knowledge", label: "知识库", icon: <BookOpen size={17} />, hint: "规范/案例/经验条目" },
  { key: "profile", label: "用户画像", icon: <Brain size={17} />, hint: "跨板块共享画像" },
  { key: "office", label: "办公记忆", icon: <FileText size={17} />, hint: "跨会话办公事实" },
  { key: "materials", label: "项目资料", icon: <Pin size={17} />, hint: "固定常用文件 · 新会话带入" },
  { key: "whisper", label: "聊天记忆", icon: <HeartOutlined style={{ fontSize: 16 }} />, hint: "轻语人格记忆 · 只读" },
  { key: "graph", label: "记忆图谱", icon: <NodeIndexOutlined style={{ fontSize: 16 }} />, hint: "3D 关系图谱" },
  { key: "digitallife", label: "数字生命", icon: <RobotOutlined style={{ fontSize: 16 }} />, hint: "Herdsman 虚拟人格记忆" },
];

// 图谱节点类型 → 语义色/标签/来源（与 GraphView 节点着色同源）
const NODE_COLORS: Record<string, string> = {
  knowledge: LIB_COLORS.knowledge,
  profile: LIB_COLORS.profile,
  office: LIB_COLORS.office,
  whisper: LIB_COLORS.whisper,
  material: LIB_COLORS.materials,
  cost: "#fbbf24", // 成本库已独立为一级板块，图谱节点仍保留琥珀色
};
const NODE_LABELS: Record<string, string> = {
  knowledge: "知识",
  profile: "画像",
  office: "办公记忆",
  whisper: "聊天记忆",
  material: "项目资料",
  cost: "成本",
};
const NODE_SOURCES: Record<string, string> = {
  knowledge: "知识库",
  profile: "用户画像",
  office: "办公记忆",
  whisper: "聊天记忆",
  material: "项目资料",
  cost: "成本库",
};

// 三脑检索 + 工作区全文搜索的合并命中。
interface HubSearchHit {
  kind: "brain" | "file";
  brain: string; // 脑名或 "文件"
  entity: string;
  text: string;
  path?: string;
}

// 详情 inspector 内容：图谱节点 或 检索命中条目。
type InspectorDetail =
  | { kind: "node"; id: string; title: string; typeLabel: string; color: string; desc?: string; source: string }
  | { kind: "hit"; brain: string; entity: string; text: string; path?: string };

/** 左侧分类轨道条目（对齐 v3 轨道语言：激活 = 主色容器底 + 左缘光条）。 */
function RailItem(p: {
  active: boolean;
  onClick: () => void;
  label: string;
  icon: React.ReactNode;
  count?: number;
  title: string;
}) {
  const { active, onClick, label, icon, count, title } = p;
  return (
    <button
      type="button"
      onClick={onClick}
      className={`hub-rail-item${active ? " is-active" : ""}`}
      title={title}
      aria-label={label}
      aria-current={active ? "page" : undefined}
    >
      <span className="hub-rail-icon" aria-hidden="true">{icon}</span>
      <span className="hub-rail-label">{label}</span>
      {typeof count === "number" && <span className="hub-rail-count">{count}</span>}
    </button>
  );
}

/** 记忆中枢「记忆图谱舰桥」：左分类轨道 | 中主区视图（列表/图谱） | 右详情 inspector。 */
function MemoryHubPage() {
  const [active, setActive] = useState<"home" | LibraryKey>("home");
  const [overview, setOverview] = useState<MemoryHubOverview | null>(null);
  const [brainQuery, setBrainQuery] = useState("");
  const [hits, setHits] = useState<HubSearchHit[]>([]);
  const [searching, setSearching] = useState(false);
  const [referenced, setReferenced] = useState<string | null>(null);
  const [detail, setDetail] = useState<InspectorDetail | null>(null);
  const [inspectorOpen, setInspectorOpen] = useState(true);

  // 三脑检索 + 工作区全文搜索合并：记忆与资料一次找齐。
  const runSearch = async () => {
    const q = brainQuery.trim();
    if (!q) {
      setHits([]);
      return;
    }
    setSearching(true);
    const bind = window.go?.app?.App as AppFacade;
    try {
      const [brainRaw, files, sem, fileSem] = await Promise.all([
        bind?.BrainSearch ? bind.BrainSearch(q, "") : Promise.resolve("[]"),
        app.WorkspaceSearch(q, 8).catch(() => [] as WorkspaceSearchHit[]),
        app.SemanticSearch(q).catch(() => [] as SemanticHitView[]),
        app.FileSemanticSearch(q, 6).catch(() => [] as import("../gaea/lib/types").FileSemanticHit[]),
      ]);
      let brain: Array<{ brain: string; entity: string; text: string }> = [];
      try { brain = JSON.parse(typeof brainRaw === 'string' ? brainRaw : (brainRaw ? JSON.stringify(brainRaw) : '[]')); } catch { brain = []; }
      const kindLabel: Record<string, string> = { cost: "语义·成本", knowledge: "语义·知识", office: "语义·办公", file: "语义·资料" };
      setHits([
        ...brain.map((h) => ({ kind: "brain" as const, brain: h.brain, entity: h.entity, text: h.text })),
        ...(files ?? []).map((f) => ({
          kind: "file" as const,
          brain: "文件",
          entity: f.name,
          text: f.snippet || f.path,
          path: f.path,
        })),
        ...(sem ?? []).map((h) => ({
          kind: "brain" as const,
          brain: kindLabel[h.kind] ?? "语义",
          entity: h.name,
          text: h.text,
        })),
        ...(fileSem ?? []).map((h) => ({
          kind: "file" as const,
          brain: "语义·文件",
          entity: h.path.split("/").pop() ?? h.path,
          text: h.snippet,
          path: h.path,
        })),
      ]);
    } catch { /* 忽略单次检索失败 */ }
    finally { setSearching(false); }
  };

  // 一键 @ 引用：写入全局输入框通道，回到办公板块后自动插入
  const referenceFile = (path: string) => {
    useComposerInsertStore.getState().requestAt(path);
    setReferenced(path);
    setTimeout(() => setReferenced((p) => (p === path ? null : p)), 1500);
  };

  // 图谱节点 → 详情 inspector
  const handleGraphSelect = (node: GraphNode | null) => {
    if (!node) {
      setDetail(null);
      return;
    }
    setDetail({
      kind: "node",
      id: node.id,
      title: node.name,
      typeLabel: NODE_LABELS[node.type] ?? node.type,
      color: NODE_COLORS[node.type] ?? "var(--gaea-glow)",
      desc: node.desc || undefined,
      source: NODE_SOURCES[node.type] ?? node.type,
    });
  };

  // 检索命中 → 详情 inspector
  const selectHit = (h: HubSearchHit) => {
    setDetail({ kind: "hit", brain: h.brain, entity: h.entity, text: h.text, path: h.path });
  };

  const selectCategory = (key: "home" | LibraryKey) => {
    setActive(key);
    setDetail(null); // 切换分类时清空详情，避免展示陈旧条目
  };

  useEffect(() => {
    app.MemoryHubOverview().then(setOverview).catch(() => {});
  }, [active]);

  const counts: Partial<Record<LibraryKey, number>> = overview
    ? {
        knowledge: overview.knowledgeCount,
        profile: overview.profileCount,
        office: overview.officeCount,
        materials: overview.pinnedCount,
        whisper: overview.whisperCount,
      }
    : {};

  const activeDef = active === "home" ? null : LIBRARIES.find((l) => l.key === active) ?? null;
  const activeCount = activeDef ? counts[activeDef.key] : undefined;

  return (
    <div className="hub-workspace">
      {/* 氛围层：网格 + 扫描线（保留，降透明避免抢戏） */}
      <div className="hub-bg" aria-hidden="true" />
      <div className="hub-grid" aria-hidden="true" />
      <div className="hub-scanline" aria-hidden="true" />

      {/* 细条头部：板块名收敛 + 三脑检索统一入口 */}
      <div className="hub-strip">
        <span className="hub-strip-title">
          <NodeIndexOutlined aria-hidden="true" />
          <span>记忆图谱舰桥</span>
        </span>
        <span className="hub-strip-sub">三脑记忆 · 统一入口</span>
        {overview?.latestUpdated && (
          <span className="hub-strip-meta">更新 {overview.latestUpdated}</span>
        )}
        <div className="ml-auto flex items-center gap-2 relative">
          <input
            value={brainQuery}
            onChange={(e) => {
              setBrainQuery(e.target.value);
              if (!e.target.value.trim()) setHits([]);
            }}
            onKeyDown={(e) => { if (e.key === "Enter") runSearch(); }}
            placeholder="三脑检索 · 工作区资料"
            aria-label="三脑检索 · 工作区资料"
            className="hub-search-input"
          />
          <button
            onClick={runSearch}
            disabled={searching}
            className="hub-search-btn"
            type="button"
          >
            {searching ? "检索中…" : "检索"}
          </button>
          {hits.length > 0 && (
            <div className="hub-search-pop">
              <div className="hub-search-pop-head">
                <span>{hits.length} 条命中</span>
                <button
                  type="button"
                  className="hub-search-clear"
                  onClick={() => setHits([])}
                  aria-label="关闭检索结果"
                  title="关闭检索结果"
                >
                  <CloseOutlined aria-hidden="true" style={{ fontSize: 10 }} />
                </button>
              </div>
              {hits.map((h, i) => (
                <div key={i} className="hub-hit" onClick={() => selectHit(h)}>
                  <span className={`shrink-0 px-1.5 rounded text-[10px] ${
                    h.kind === "file"
                      ? "bg-sky-500/20 text-sky-300"
                      : h.brain === "brain.right"
                        ? "bg-pink-500/20 text-pink-300"
                        : h.brain === "brain.left"
                          ? "bg-emerald-500/20 text-emerald-300"
                          : "bg-violet-500/20 text-violet-300"
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
              ))}
            </div>
          )}
        </div>
      </div>

      {/* 3 分区工作台：左分类轨道 | 中主区视图 | 右详情 inspector */}
      <div className="hub-body">
        {/* 左：分类轨道（图标 + 标签，替代顶部横排 tab） */}
        <aside className="hub-rail v3-panel" aria-label="记忆分类导航">
          <div className="v3-panel-head">
            <span className="v3-panel-title">记忆分类</span>
          </div>
          <nav className="hub-rail-nav">
            <RailItem
              active={active === "home"}
              onClick={() => selectCategory("home")}
              label="总览"
              icon={<HomeOutlined />}
              title="记忆中枢总览 · 3D 图谱"
            />
            {LIBRARIES.map((lib) => (
              <RailItem
                key={lib.key}
                active={active === lib.key}
                onClick={() => selectCategory(lib.key)}
                label={lib.label}
                icon={lib.icon}
                count={counts[lib.key]}
                title={`${lib.label} · ${lib.hint}`}
              />
            ))}
          </nav>
        </aside>

        {/* 中：主区视图（总览/图谱保持 GraphView 沉浸感；分类 = 各库列表） */}
        <main className="hub-main" aria-label="主区视图">
          {active === "home" ? (
            <GraphView variant="home" onSelect={handleGraphSelect} />
          ) : active === "graph" ? (
            <GraphView variant="page" onSelect={handleGraphSelect} />
          ) : (
            <div className="hub-library-zone">
              <LocaleProvider>
                {active === "knowledge" && (
                  <div className="h-full">
                    <KnowledgePanel variant="page" onClose={() => {}} />
                  </div>
                )}
                {active === "profile" && <ProfileLibrary />}
                {active === "office" && <OfficeMemoryLibrary />}
                {active === "materials" && <MaterialsLibrary />}
                {active === "whisper" && <WhisperMemoryLibrary />}
                {active === "digitallife" && <DigitalLifeLibrary />}
              </LocaleProvider>
            </div>
          )}
        </main>

        {/* 右：详情 inspector（仅总览/图谱 tab 显示——它只联动图谱节点与检索命中；
            库类 tab 下各库面板自带详情/编辑，外壳 inspector 会与面板嵌套成多余竖条） */}
        {(active === "home" || active === "graph") && (
        <aside
          className={`hub-inspector v3-panel${inspectorOpen ? "" : " is-collapsed"}`}
          aria-label="详情面板"
        >
          <div className="v3-panel-head">
            <span className="v3-panel-title">详情</span>
            <span className="v3-panel-spacer" />
            <button
              type="button"
              className="hub-inspector-toggle"
              onClick={() => setInspectorOpen((v) => !v)}
              aria-label={inspectorOpen ? "折叠详情面板" : "展开详情面板"}
              aria-expanded={inspectorOpen}
              title={inspectorOpen ? "折叠详情面板" : "展开详情面板"}
            >
              {inspectorOpen ? (
                <LeftOutlined aria-hidden="true" style={{ fontSize: 11 }} />
              ) : (
                <RightOutlined aria-hidden="true" style={{ fontSize: 11 }} />
              )}
            </button>
          </div>
          {!inspectorOpen && <div className="hub-inspector-collapsed-label">详情</div>}
          {inspectorOpen && (
            <div className="hub-inspector-body">
              {detail ? (
                detail.kind === "node" ? (
                  /* 图谱节点详情 */
                  <div className="hub-detail">
                    <div className="hub-detail-head">
                      <span
                        className="hub-detail-dot"
                        style={{ background: detail.color, boxShadow: `0 0 8px ${detail.color}` }}
                      />
                      <span className="hub-detail-type">{detail.typeLabel}</span>
                    </div>
                    <div className="hub-detail-title">{detail.title}</div>
                    {detail.desc && <div className="hub-detail-desc">{detail.desc}</div>}
                    <div className="hub-detail-row">
                      <span className="hub-detail-key">ID</span>
                      <span className="hub-detail-val mono">{detail.id}</span>
                    </div>
                    <div className="hub-detail-row">
                      <span className="hub-detail-key">来源</span>
                      <span className="hub-detail-val">{detail.source}</span>
                    </div>
                  </div>
                ) : (
                  /* 检索命中详情 */
                  <div className="hub-detail">
                    <div className="hub-detail-head">
                      <span className="hub-detail-badge">{detail.brain}</span>
                    </div>
                    <div className="hub-detail-title">{detail.entity}</div>
                    {detail.text && <div className="hub-detail-desc">{detail.text}</div>}
                    {detail.path && (
                      <div className="hub-detail-row">
                        <span className="hub-detail-key">路径</span>
                        <span className="hub-detail-val mono">{detail.path}</span>
                      </div>
                    )}
                    {detail.path && (
                      <div className="hub-detail-actions">
                        <button
                          type="button"
                          className="hub-detail-action"
                          onClick={() => usePreviewStore.getState().openFilePreview(detail.path!)}
                          title="打开文件预览"
                        >
                          <FileSearchOutlined aria-hidden="true" style={{ fontSize: 11 }} />
                          预览
                        </button>
                        <button
                          type="button"
                          className={`hub-detail-action${referenced === detail.path ? " is-referenced" : ""}`}
                          onClick={() => referenceFile(detail.path!)}
                          title="一键 @ 引用（回到办公板块自动插入输入框）"
                        >
                          {referenced === detail.path ? "已引用" : "@ 引用"}
                        </button>
                      </div>
                    )}
                  </div>
                )
              ) : (
                /* 优雅空态 */
                <div className="hub-empty">
                  <FileSearchOutlined aria-hidden="true" className="hub-empty-icon" />
                  <div className="hub-empty-title">未选中条目</div>
                  <div className="hub-empty-text">
                    {activeDef
                      ? `${activeDef.label} · ${activeDef.hint}${
                          typeof activeCount === "number" ? `（${activeCount} 条）` : ""
                        }`
                      : "记忆中枢总览 · 三脑记忆统一入口"}
                  </div>
                  <div className="hub-empty-tip">在图谱中点击节点，或点击检索命中条目，详情将在此展示</div>
                </div>
              )}
            </div>
          )}
        </aside>
        )}
      </div>

      {/* 文件预览弹层：项目资料 / 检索命中共用 */}
      <FilePreviewModal />
    </div>
  );
}

export default MemoryHubPage;
