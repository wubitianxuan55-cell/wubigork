// CapabilitiesPanel（T6-10.1 巨型组件拆分后的编排层，行为零变化）
// 职责：标签页/展开态 + 跨 hook/组件装配；能力快照数据（useCapabilitiesData）、
// MCP 服务器区（capabilities/ServersSection）、工具区（capabilities/ToolsSection）、
// 技能区（capabilities/SkillsSection）见各产物文件。
//
// CapabilitiesPanel is the desktop MCP & Skills drawer — the GUI counterpart to
// the CLI's /mcp + /skill, aligning with Claude Code's Customize → Connectors:
// each server shows a connected/failed dot, transport, and tool/prompt/resource
// counts, with add / remove / retry; skills list their scope and run mode.
import { useCallback, useState } from "react";
import { Globe, Cpu, RefreshCw } from "../icons";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import { DrawerHeader, DrawerTitle, DrawerSubtitle } from "./DrawerHeader";
import { ResizableDrawer } from "./ResizableDrawer";
import { useCapabilitiesData } from "../hooks/useCapabilitiesData";
import { ServerGroup, FailedServersNotice, AddServerForm } from "./capabilities/ServersSection";
import { ToolsTabContent } from "./capabilities/ToolsSection";
import { SkillsSection } from "./capabilities/SkillsSection";

type CapTab = "servers" | "tools" | "skills";
export function CapabilitiesPanel({
  onClose,
  toolCounts = {},
  skillCounts = {},
}: {
  onClose: () => void;
  toolCounts?: Record<string, number>;
  skillCounts?: Record<string, number>;
}) {
  const t = useT();
  const data = useCapabilitiesData();
  const {
    view, busy, err, notice, reloading, adding, addingContext7, confirming,
    setConfirming, setAdding, reloadEngine, mutate, addContext7, summary, serverGroups,
  } = data;
  const [tab, setTab] = useState<CapTab>("servers");
  const [expandedSkills, setExpandedSkills] = useState<Set<string>>(() => new Set());
  const [expandedErrors, setExpandedErrors] = useState<Set<string>>(() => new Set());
  const [expandedServers, setExpandedServers] = useState<Set<string>>(() => new Set());

  const toggleSkill = useCallback((name: string) => {
    setExpandedSkills((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  const toggleError = useCallback((name: string) => {
    setExpandedErrors((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  const toggleServer = useCallback((name: string) => {
    setExpandedServers((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  return (
    <ResizableDrawer onClose={onClose} subtle>
        <DrawerHeader onClose={onClose}>
          <div>
            <DrawerTitle text={t("caps.title")} />
            {view && <DrawerSubtitle text={summary} />}
          </div>
          <button
            className="flex items-center gap-1.5 px-2 py-1 text-xs border border-border-soft rounded text-fg-dim cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors disabled:opacity-40 disabled:cursor-default"
            disabled={reloading}
            onClick={() => void reloadEngine()}
            title={t("caps.reloadHint")}
          >
            {reloading ? (
              <span className="animate-spin inline-block w-3 h-3 border border-current border-t-transparent rounded-full" />
            ) : (
              <RefreshCw size={13} />
            )}
            <span>{reloading ? t("caps.reloading") : t("caps.reload")}</span>
          </button>
        </DrawerHeader>

        {!view ? (
          <div className="empty-state">{t("caps.loading")}</div>
        ) : (
          <div className="overflow-y-auto px-4 py-3.5 flex flex-col gap-5">
            {err && <div className="shrink-0 px-4 py-2 text-[12.5px] bg-del-bg text-err border-b border-border-soft">{err}</div>}
            {notice && <div className="shrink-0 px-4 py-2 text-[12.5px] bg-ok/10 text-ok border-b border-ok/20">{notice}</div>}
            <div className="flex border-b border-border-soft mb-3" role="tablist" aria-label={t("caps.title")}>
              <button
                className={`flex-1 px-4 py-2 border-0 border-b-2 bg-transparent text-[13px] font-medium cursor-pointer transition-[color,border] duration-[var(--dur-fast)] ${
                  tab === "servers" ? "text-accent border-accent" : "text-fg-dim border-transparent hover:text-fg hover:border-fg-faint"
                }`}
                role="tab" aria-selected={tab === "servers"} onClick={() => setTab("servers")}
              >{t("caps.connectorsTab")}</button>
              <button
                className={`flex-1 px-4 py-2 border-0 border-b-2 bg-transparent text-[13px] font-medium cursor-pointer transition-[color,border] duration-[var(--dur-fast)] ${
                  tab === "tools" ? "text-accent border-accent" : "text-fg-dim border-transparent hover:text-fg hover:border-fg-faint"
                }`}
                role="tab" aria-selected={tab === "tools"} onClick={() => setTab("tools")}
              >
                <Cpu size={12} className="inline mr-1 align-middle -mt-px" />
                <span>工具</span>
              </button>
              <button
                className={`flex-1 px-4 py-2 border-0 border-b-2 bg-transparent text-[13px] font-medium cursor-pointer transition-[color,border] duration-[var(--dur-fast)] ${
                  tab === "skills" ? "text-accent border-accent" : "text-fg-dim border-transparent hover:text-fg hover:border-fg-faint"
                }`}
                role="tab" aria-selected={tab === "skills"} onClick={() => setTab("skills")}
              >{t("caps.skillsTab")}</button>
            </div>

            {tab === "servers" ? (
              <section className="mb-3">
                <div className="flex justify-end mb-2">
                  {/* Context7 一键添加 */}
                  <button
                    className="flex items-center gap-1.5 mr-2 px-2.5 py-1 text-xs border border-accent/30 rounded bg-accent/5 text-accent cursor-pointer hover:bg-accent/10 transition-colors disabled:opacity-40"
                    disabled={busy || addingContext7}
                    onClick={() => addContext7()}
                    title={t("caps.addContext7Hint")}
                  >
                    {addingContext7 ? (
                      <span className="animate-spin inline-block w-3 h-3 border border-current border-t-transparent rounded-full" />
                    ) : (
                      <Globe size={12} />
                    )}
                    <span>{addingContext7 ? t("caps.addContext7Busy") : t("caps.addContext7")}</span>
                  </button>
                  {!adding && (
                    <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={() => setAdding(true)}>
                      {t("caps.addServer")}
                    </button>
                  )}
                </div>
                {serverGroups.failed.length > 0 && (
                  <FailedServersNotice
                    servers={serverGroups.failed}
                    expanded={expandedErrors}
                    onToggle={toggleError}
                    onRetry={(name) => void mutate(() => app.RetryMCPServer(name))}
                    confirming={confirming}
                    onConfirm={setConfirming}
                    onCancelConfirm={() => setConfirming(null)}
                    onRemove={(name) => mutate(() => app.RemoveMCPServer(name)).then(() => setConfirming(null))}
                    busy={busy}
                  />
                )}
                {view.servers.length === 0 && !adding && (
                  <div className="text-fg-faint text-xs text-center py-4">{t("caps.noServers")}</div>
                )}
                <ServerGroup
                  busy={busy}
                  servers={serverGroups.active}
                  expanded={expandedServers}
                  confirming={confirming}
                  onConfirm={setConfirming}
                  onCancelConfirm={() => setConfirming(null)}
                  onRemove={(name) => mutate(() => app.RemoveMCPServer(name)).then(() => setConfirming(null))}
                  onRetry={(name) => void mutate(() => app.RetryMCPServer(name))}
                  onToggle={(name, on) => void mutate(() => app.SetMCPServerEnabled(name, on))}
                  onToggleDetails={toggleServer}
                />
                {adding ? (
                  <AddServerForm busy={busy} onCancel={() => setAdding(false)} onAdd={async (input) => (await mutate(() => app.AddMCPServer(input))) && setAdding(false)} />
                ) : null}
              </section>
            ) : tab === "tools" ? (
              <ToolsTabContent toolCounts={toolCounts} />
            ) : (
              <SkillsSection skills={view.skills} counts={skillCounts} expanded={expandedSkills} onToggle={toggleSkill} />
            )}
          </div>
        )}
    </ResizableDrawer>
  );
}
