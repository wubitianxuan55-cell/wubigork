import React, { useEffect, useState } from "react";
import { HeartOutlined, NodeIndexOutlined, ArrowLeftOutlined, RobotOutlined } from "@ant-design/icons";
import { BookOpen, Brain, Coins, FileText, Pin } from "../gaea/icons";
import { LocaleProvider } from "../gaea/lib/i18n";
import { app } from "../gaea/lib/bridge";
import { useComposerInsertStore, usePreviewStore } from "../gaea/lib/store";
import type { MemoryHubOverview, SemanticHitView, WorkspaceSearchHit } from "../gaea/lib/types";
import type { AppFacade } from "../types/wails";
import { KnowledgePanel } from "../gaea/components/KnowledgePanel";
import { ProfileLibrary } from "../gaea/components/memoryhub/ProfileLibrary";
import { OfficeMemoryLibrary } from "../gaea/components/memoryhub/OfficeMemoryLibrary";
import { WhisperMemoryLibrary } from "../gaea/components/memoryhub/WhisperMemoryLibrary";
import { GraphView } from "../gaea/components/memoryhub/GraphView";
import { CostLibrary } from "../gaea/components/memoryhub/CostLibrary";
import { MaterialsLibrary } from "../gaea/components/memoryhub/MaterialsLibrary";
import { DigitalLifeLibrary } from "../gaea/components/memoryhub/DigitalLifeLibrary";
import { ModuleCard } from "../gaea/components/memoryhub/ModuleCard";
import { FilePreviewModal } from "../gaea/components/FilePreviewModal";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "../gaea/components/memoryhub/hub.css";

type LibraryKey = "knowledge" | "cost" | "profile" | "office" | "materials" | "whisper" | "graph" | "digitallife";

// 各库霓虹色（与 3D 图谱着色一致：indigo 知识 / amber 成本 / emerald 办公 / pink 聊天记忆）
const LIB_COLORS: Record<LibraryKey, string> = {
  knowledge: "#818cf8",
  cost: "#fbbf24",
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
  { key: "cost", label: "成本库", icon: <Coins size={17} />, hint: "单价/单位/来源" },
  { key: "profile", label: "用户画像", icon: <Brain size={17} />, hint: "跨板块共享画像" },
  { key: "office", label: "办公记忆", icon: <FileText size={17} />, hint: "跨会话办公事实" },
  { key: "materials", label: "项目资料", icon: <Pin size={17} />, hint: "固定常用文件 · 新会话带入" },
  { key: "whisper", label: "聊天记忆", icon: <HeartOutlined style={{ fontSize: 16 }} />, hint: "轻语人格记忆 · 只读" },
  { key: "graph", label: "记忆图谱", icon: <NodeIndexOutlined style={{ fontSize: 16 }} />, hint: "3D 关系图谱" },
  { key: "digitallife", label: "数字生命", icon: <RobotOutlined style={{ fontSize: 16 }} />, hint: "Herdsman 虚拟人格记忆" },
];

// 首页卡片排布：左列 3 + 右列 3
const LEFT_CARDS: LibraryKey[] = ["knowledge", "cost", "profile", "materials"];
const RIGHT_CARDS: LibraryKey[] = ["office", "whisper", "digitallife", "graph"];

// 三脑检索 + 工作区全文搜索的合并命中。
interface HubSearchHit {
  kind: "brain" | "file";
  brain: string; // 脑名或 "文件"
  entity: string;
  text: string;
  path?: string;
}

// 记忆中枢首页：中央 3D 图谱 + 四周霓虹玻璃模块卡片。
// 点击卡片切换到对应库面板；三脑记忆（主脑知识/画像 + 左脑办公 + 右脑聊天记忆）统一入口。
function MemoryHubPage() {
  const [active, setActive] = useState<"home" | LibraryKey>("home");
  const [overview, setOverview] = useState<MemoryHubOverview | null>(null);
  const [brainQuery, setBrainQuery] = useState("");
  const [hits, setHits] = useState<HubSearchHit[]>([]);
  const [searching, setSearching] = useState(false);
  const [referenced, setReferenced] = useState<string | null>(null);

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

  useEffect(() => {
    app.MemoryHubOverview().then(setOverview).catch(() => {});
  }, [active]);

  const counts: Partial<Record<LibraryKey, number>> = overview
    ? {
        knowledge: overview.knowledgeCount,
        profile: overview.profileCount,
        office: overview.officeCount,
        cost: overview.costCount,
        materials: overview.pinnedCount,
        whisper: overview.whisperCount,
      }
    : {};

  // ── 科幻首页 ─────────────────────────────────────────────────
  if (active === "home") {
    return (
      <div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
        <div className="flex-1 min-h-0 relative">
          <div className="hub-bg" />
          <div className="hub-grid" />
          <div className="hub-scanline" />

          <div className="relative h-full flex flex-col px-5 pt-4 pb-4">
            {/* 标题 + 总览 */}
            <div className="shrink-0 flex items-end gap-3 mb-3">
              <div className="hub-title text-[22px] font-bold leading-none tracking-wide">记忆中枢</div>
              <div className="text-fg-faint text-[11px] leading-tight pb-0.5">
                三脑记忆 · 统一入口
                {overview?.latestUpdated && (
                  <span className="ml-2 font-mono text-fg-faint/70">更新 {overview.latestUpdated}</span>
                )}
              </div>
              <div className="ml-auto flex items-center gap-2">
                <input
                  value={brainQuery}
                  onChange={(e) => setBrainQuery(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") runSearch(); }}
                  placeholder="三脑检索 · 工作区资料"
                  className="h-7 w-44 rounded-lg px-2.5 text-[12px] bg-bg-soft border border-border-soft focus:outline-none"
                />
                <button
                  onClick={runSearch}
                  disabled={searching}
                  className="inline-flex items-center h-7 px-3 rounded-lg text-[12px] bg-indigo-500/20 text-indigo-300 hover:bg-indigo-500/30 transition-colors"
                >
                  {searching ? "检索中…" : "检索"}
                </button>
              </div>
            </div>

            {hits.length > 0 && (
              <div className="shrink-0 mb-3 max-h-44 overflow-auto rounded-xl border border-border-soft bg-bg-soft/70 p-3">
                {hits.map((h, i) => (
                  <div key={i} className="flex items-start gap-2 py-1 text-[12px] border-b border-border-soft/50 last:border-0">
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
                          className="text-fg font-medium shrink-0 hover:text-accent cursor-pointer"
                          onClick={() => usePreviewStore.getState().openFilePreview(h.path!)}
                          title="点击预览"
                        >
                          {h.entity}
                        </button>
                        <span className="text-fg-faint truncate min-w-0 flex-1">{h.text}</span>
                        <button
                          type="button"
                          className={`shrink-0 px-1.5 rounded text-[10px] cursor-pointer transition-colors ${
                            referenced === h.path ? "bg-accent/25 text-accent" : "text-fg-faint hover:text-accent"
                          }`}
                          onClick={() => referenceFile(h.path!)}
                          title="一键 @ 引用（回到办公板块自动插入输入框）"
                        >
                          {referenced === h.path ? "已引用" : "@ 引用"}
                        </button>
                      </>
                    ) : (
                      <>
                        <span className="text-fg font-medium shrink-0">{h.entity}</span>
                        <span className="text-fg-faint truncate">{h.text}</span>
                      </>
                    )}
                  </div>
                ))}
              </div>
            )}

            {/* 主区：左卡列 | 中央图谱 | 右卡列 */}
            <div className="flex-1 min-h-0 flex gap-4">
              {/* 左卡列 */}
              <div className="w-52 shrink-0 flex flex-col gap-3 justify-center">
                {LEFT_CARDS.map((key, i) => {
                  const lib = LIBRARIES.find((l) => l.key === key)!;
                  return (
                    <ModuleCard
                      key={key}
                      index={i}
                      label={lib.label}
                      icon={lib.icon}
                      hint={lib.hint}
                      count={counts[key]}
                      color={LIB_COLORS[key]}
                      onClick={() => setActive(key)}
                    />
                  );
                })}
              </div>

              {/* 中央 3D 图谱 */}
              <div className="hub-graph-shell flex-1 min-w-0">
                <GraphView variant="home" />
              </div>

              {/* 右卡列 */}
              <div className="w-52 shrink-0 flex flex-col gap-3 justify-center">
                {RIGHT_CARDS.map((key, i) => {
                  const lib = LIBRARIES.find((l) => l.key === key)!;
                  return (
                    <ModuleCard
                      key={key}
                      index={i + 3}
                      label={lib.label}
                      icon={lib.icon}
                      hint={lib.hint}
                      count={counts[key]}
                      color={LIB_COLORS[key]}
                      onClick={() => setActive(key)}
                    />
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // ── 库面板（从首页点入）────────────────────────────────────
  const lib = LIBRARIES.find((l) => l.key === active)!;
  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
      {/* 返回首页条 */}
      <div className="shrink-0 flex items-center gap-2 px-4 pt-3 pb-1">
        <button
          onClick={() => setActive("home")}
          className="inline-flex items-center gap-1.5 px-2.5 h-7 rounded-lg text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[12px]"
        >
          <ArrowLeftOutlined style={{ fontSize: 11 }} /> 返回首页
        </button>
        <span className="w-px h-4 bg-border-soft mx-1" />
        <span style={{ color: LIB_COLORS[active] }} className="text-[13px] font-semibold">
          {lib.label}
        </span>
        <span className="text-fg-faint text-[11px]">{lib.hint}</span>
      </div>

      {/* 库内容 */}
      <div className="flex-1 min-h-0">
        <LocaleProvider>
          {active === "knowledge" && (
            <div className="h-full">
              <KnowledgePanel variant="page" onClose={() => {}} />
            </div>
          )}
          {active === "cost" && <CostLibrary />}
          {active === "profile" && <ProfileLibrary />}
          {active === "office" && <OfficeMemoryLibrary />}
          {active === "materials" && <MaterialsLibrary />}
          {active === "whisper" && <WhisperMemoryLibrary />}
          {active === "graph" && <GraphView variant="page" />}
          {active === "digitallife" && <DigitalLifeLibrary />}
        </LocaleProvider>
      </div>

      {/* 文件预览弹层：项目资料 / 检索命中共用 */}
      <FilePreviewModal />
    </div>
  );
}

export default MemoryHubPage;
