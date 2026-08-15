// CapabilitiesPanel（T6-10.1 巨型组件拆分后的编排层，行为零变化）
// 职责：标签页/展开态 + 跨 hook/组件装配；能力快照数据（useCapabilitiesData）、
// MCP 服务器区（capabilities/ServersSection）、工具区（capabilities/ToolsSection）、
// 技能区（capabilities/SkillsSection）见各产物文件。
//
// CapabilitiesPanel is the desktop MCP & Skills drawer — the GUI counterpart to
// the CLI's /mcp + /skill, aligning with Claude Code's Customize → Connectors:
// each server shows a connected/failed dot, transport, and tool/prompt/resource
// counts, with add / remove / retry; skills list their scope and run mode.
// v3「星枢」面板语言：分段式 v3 标签页（激活 = 主色容器 + 柔光），令牌化操作按钮。
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

  const reloadBtn =
    "flex items-center gap-1.5 px-2 py-1 text-xs rounded-md cursor-pointer transition-colors disabled:opacity-40 disabled:cursor-default";

  return (
    <ResizableDrawer onClose={onClose} subtle>
        <DrawerHeader onClose={onClose}>
          <div>
            <DrawerTitle text={t("caps.title")} />
            {view && <DrawerSubtitle text={summary} />}
          </div>
          <button
            className={reloadBtn}
            style={{
              border: "1px solid var(--md-sys-color-outline-variant)",
              color: "var(--md-sys-color-text-secondary)",
              background: "transparent",
            }}
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
            {err && (
              <div
                className="shrink-0 px-4 py-2 text-[12.5px]"
                style={{
                  background: "color-mix(in srgb, var(--md-sys-color-destructive) 9%, transparent)",
                  color: "var(--md-sys-color-destructive)",
                  borderBottom: "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 22%, transparent)",
                }}
              >
                {err}
              </div>
            )}
            {notice && (
              <div
                className="shrink-0 px-4 py-2 text-[12.5px]"
                style={{
                  background: "color-mix(in srgb, var(--md-sys-color-success) 9%, transparent)",
                  color: "var(--md-sys-color-success)",
                  borderBottom: "1px solid color-mix(in srgb, var(--md-sys-color-success) 22%, transparent)",
                }}
              >
                {notice}
              </div>
            )}
            {/* v3 分段标签页：激活 = 主色容器 + 柔光 */}
            <div
              className="flex gap-1 p-1 rounded-[var(--radius-md)]"
              role="tablist"
              aria-label={t("caps.title")}
              style={{
                background: "var(--md-sys-color-surface-container)",
                border: "1px solid var(--md-sys-color-outline-variant)",
              }}
            >
              {(
                [
                  { key: "servers" as CapTab, label: t("caps.connectorsTab") },
                  { key: "tools" as CapTab, label: "工具", icon: <Cpu size={12} aria-hidden className="inline mr-1 align-middle -mt-px" /> },
                  { key: "skills" as CapTab, label: t("caps.skillsTab") },
                ]
              ).map((item) => {
                const active = tab === item.key;
                return (
                  <button
                    key={item.key}
                    role="tab"
                    aria-selected={active}
                    onClick={() => setTab(item.key)}
                    className={`flex-1 px-3 py-1.5 rounded-[var(--radius-sm)] text-[13px] font-medium cursor-pointer transition-all duration-200 ${
                      active
                        ? "bg-(color:--md-sys-color-primary-container) text-(color:--md-sys-color-on-primary-container) shadow-[var(--v3-glow-faint)]"
                        : "text-(color:--md-sys-color-text-secondary) hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high)"
                    }`}
                  >
                    {item.icon}
                    <span>{item.label}</span>
                  </button>
                );
              })}
            </div>

            {tab === "servers" ? (
              <section className="mb-3">
                <div className="flex justify-end mb-2">
                  {/* Context7 一键添加 */}
                  <button
                    className="flex items-center gap-1.5 mr-2 px-2.5 py-1 text-xs rounded-md cursor-pointer transition-colors disabled:opacity-40"
                    style={{
                      border: "1px solid color-mix(in srgb, var(--gaea-glow) 34%, transparent)",
                      background: "color-mix(in srgb, var(--gaea-glow) 7%, transparent)",
                      color: "var(--gaea-glow)",
                    }}
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
                    <button
                      className="px-2.5 py-1 text-xs rounded-md cursor-pointer"
                      style={{
                        border: "1px solid var(--md-sys-color-outline-variant)",
                        color: "var(--md-sys-color-text-secondary)",
                        background: "transparent",
                      }}
                      disabled={busy}
                      onClick={() => setAdding(true)}
                    >
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
                  <div className="text-xs text-center py-4" style={{ color: "var(--md-sys-color-text-secondary)" }}>{t("caps.noServers")}</div>
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
