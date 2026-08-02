import React, { useEffect, useState } from "react";
import { HeartOutlined, NodeIndexOutlined } from "@ant-design/icons";
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
import { ComingSoon } from "../gaea/components/memoryhub/ComingSoon";
import "../gaea/styles.css";
import "../gaea/tailwind.css";

type LibraryKey = "knowledge" | "cost" | "profile" | "office" | "whisper" | "graph";

interface LibraryDef {
  key: LibraryKey;
  label: string;
  icon: React.ReactNode;
  hint: string;
}

const LIBRARIES: LibraryDef[] = [
  { key: "knowledge", label: "知识库", icon: <BookOpen size={15} />, hint: "工程知识条目（规范/案例/经验）" },
  { key: "cost", label: "成本库", icon: <Coins size={15} />, hint: "成本条目（单价/单位/来源）" },
  { key: "profile", label: "用户画像", icon: <Brain size={15} />, hint: "跨板块共享画像" },
  { key: "office", label: "办公记忆", icon: <FileText size={15} />, hint: "Hephaestus 工作事实" },
  { key: "whisper", label: "轻语记忆", icon: <HeartOutlined style={{ fontSize: 14 }} />, hint: "Hermes 人格记忆（只读）" },
  { key: "graph", label: "记忆图谱", icon: <NodeIndexOutlined style={{ fontSize: 14 }} />, hint: "3D 记忆关系图（下阶段）" },
];

// 记忆中枢：三脑架构（主脑/左脑/右脑）的统一前端入口。
// 左脑办公记忆 + 主脑知识/画像集中管理，右脑轻语记忆只读浏览，图谱下阶段。
function MemoryHubPage() {
  const [active, setActive] = useState<LibraryKey>("knowledge");
  const [overview, setOverview] = useState<MemoryHubOverview | null>(null);

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

  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
      <div style={{ flex: 1, minHeight: 0, display: "flex" }}>
        {/* ── 左侧分库导航 ─────────────────────────────────── */}
        <aside className="w-48 shrink-0 border-r border-border-soft flex flex-col bg-bg-soft/40">
          <div className="px-4 pt-4 pb-2">
            <div className="text-fg text-[14px] font-semibold tracking-tight">记忆中枢</div>
            <div className="text-fg-faint text-[10.5px] mt-0.5">三脑记忆统一管理</div>
          </div>
          <nav className="flex-1 min-h-0 overflow-y-auto px-2 pb-2 space-y-0.5">
            {LIBRARIES.map((lib) => {
              const count = counts[lib.key];
              const isActive = active === lib.key;
              return (
                <button
                  key={lib.key}
                  onClick={() => setActive(lib.key)}
                  className={`w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-left transition-colors ${
                    isActive ? "bg-sidebar-active text-fg" : "text-fg-dim hover:text-fg hover:bg-bg-soft"
                  }`}
                  title={lib.hint}
                >
                  <span className={`shrink-0 ${isActive ? "text-accent" : "text-fg-faint"}`}>{lib.icon}</span>
                  <span className="flex-1 min-w-0 truncate text-[12.5px]">{lib.label}</span>
                  {typeof count === "number" && (
                    <span className="shrink-0 px-1.5 py-0.5 rounded-full bg-bg-elev text-fg-faint text-[10px]">{count}</span>
                  )}
                </button>
              );
            })}
          </nav>
          <div className="px-4 py-3 border-t border-border-soft text-fg-faint text-[10.5px] leading-relaxed">
            主脑统一记忆 API
            <br />
            Hephaestus.db · hermes.db
          </div>
        </aside>

        {/* ── 右侧内容区 ─────────────────────────────────── */}
        <main className="flex-1 min-w-0 flex flex-col bg-bg">
          <LocaleProvider>
          {active === "knowledge" && (
            <div className="flex-1 min-h-0">
              <KnowledgePanel variant="page" onClose={() => {}} />
            </div>
          )}
          {active === "cost" && <CostLibrary />}
          {active === "profile" && <ProfileLibrary />}
          {active === "office" && <OfficeMemoryLibrary />}
          {active === "whisper" && <WhisperMemoryLibrary />}
          {active === "graph" && <GraphView />}
          </LocaleProvider>
        </main>
      </div>
    </div>
  );
}

export default MemoryHubPage;
