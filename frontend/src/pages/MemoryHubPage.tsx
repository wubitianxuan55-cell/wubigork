import React, { useEffect, useState } from "react";
import { HeartOutlined, NodeIndexOutlined, ArrowLeftOutlined } from "@ant-design/icons";
import { BookOpen, Brain, Coins, FileText } from "../gaea/icons";
import { LocaleProvider } from "../gaea/lib/i18n";
import { app } from "../gaea/lib/bridge";
import type { MemoryHubOverview } from "../gaea/lib/types";
import { KnowledgePanel } from "../gaea/components/KnowledgePanel";
import { ProfileLibrary } from "../gaea/components/memoryhub/ProfileLibrary";
import { OfficeMemoryLibrary } from "../gaea/components/memoryhub/OfficeMemoryLibrary";
import { WhisperMemoryLibrary } from "../gaea/components/memoryhub/WhisperMemoryLibrary";
import { GraphView } from "../gaea/components/memoryhub/GraphView";
import { CostLibrary } from "../gaea/components/memoryhub/CostLibrary";
import { ModuleCard } from "../gaea/components/memoryhub/ModuleCard";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "../gaea/components/memoryhub/hub.css";

type LibraryKey = "knowledge" | "cost" | "profile" | "office" | "whisper" | "graph";

// 各库霓虹色（与 3D 图谱着色一致：indigo 知识 / amber 成本 / emerald 办公 / pink 聊天记忆）
const LIB_COLORS: Record<LibraryKey, string> = {
  knowledge: "#818cf8",
  cost: "#fbbf24",
  profile: "#a78bfa",
  office: "#34d399",
  whisper: "#f472b6",
  graph: "#22d3ee",
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
  { key: "whisper", label: "聊天记忆", icon: <HeartOutlined style={{ fontSize: 16 }} />, hint: "Hermes 人格记忆 · 只读" },
  { key: "graph", label: "记忆图谱", icon: <NodeIndexOutlined style={{ fontSize: 16 }} />, hint: "3D 关系图谱" },
];

// 首页卡片排布：左列 3 + 右列 3
const LEFT_CARDS: LibraryKey[] = ["knowledge", "cost", "profile"];
const RIGHT_CARDS: LibraryKey[] = ["office", "whisper", "graph"];

// 记忆中枢首页：中央 3D 图谱 + 四周霓虹玻璃模块卡片。
// 点击卡片切换到对应库面板；三脑记忆（主脑知识/画像 + 左脑办公 + 右脑聊天记忆）统一入口。
function MemoryHubPage() {
  const [active, setActive] = useState<"home" | LibraryKey>("home");
  const [overview, setOverview] = useState<MemoryHubOverview | null>(null);
  const [brainQuery, setBrainQuery] = useState("");
  const [brainHits, setBrainHits] = useState<Array<{ brain: string; entity: string; text: string }>>([]);

  const runBrainSearch = async () => {
    const bind = (window as any).go?.app?.App;
    if (!bind?.BrainSearch) return;
    try {
      setBrainHits(JSON.parse(await bind.BrainSearch(brainQuery, "")));
    } catch { /* 忽略单次检索失败 */ }
  };

  useEffect(() => {
    app.MemoryHubOverview().then(setOverview).catch(() => {});
  }, [active]);

  const counts: Partial<Record<LibraryKey, number>> = overview
    ? {
        knowledge: overview.knowledgeCount,
        profile: overview.profileCount,
        office: overview.officeCount,
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
                  onKeyDown={(e) => { if (e.key === "Enter") runBrainSearch(); }}
                  placeholder="三脑检索"
                  className="h-7 w-44 rounded-lg px-2.5 text-[12px] bg-bg-soft border border-border-soft focus:outline-none"
                />
                <button
                  onClick={runBrainSearch}
                  className="inline-flex items-center h-7 px-3 rounded-lg text-[12px] bg-indigo-500/20 text-indigo-300 hover:bg-indigo-500/30 transition-colors"
                >
                  检索
                </button>
              </div>
            </div>

            {brainHits.length > 0 && (
              <div className="shrink-0 mb-3 max-h-40 overflow-auto rounded-xl border border-border-soft bg-bg-soft/70 p-3">
                {brainHits.map((h, i) => (
                  <div key={i} className="flex items-start gap-2 py-1 text-[12px] border-b border-border-soft/50 last:border-0">
                    <span className={`shrink-0 px-1.5 rounded text-[10px] ${h.brain === "brain.right" ? "bg-pink-500/20 text-pink-300" : h.brain === "brain.left" ? "bg-emerald-500/20 text-emerald-300" : "bg-violet-500/20 text-violet-300"}`}>
                      {h.brain === "brain.right" ? "右脑" : h.brain === "brain.left" ? "左脑" : "主脑"}
                    </span>
                    <span className="text-fg font-medium shrink-0">{h.entity}</span>
                    <span className="text-fg-faint truncate">{h.text}</span>
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
          {active === "whisper" && <WhisperMemoryLibrary />}
          {active === "graph" && <GraphView variant="page" />}
        </LocaleProvider>
      </div>
    </div>
  );
}

export default MemoryHubPage;
