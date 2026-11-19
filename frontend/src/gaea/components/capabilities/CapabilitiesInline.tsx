import { useCallback, useEffect, useState } from "react";
import { Globe, Cpu, RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";
import { useT } from "../../lib/i18n";
import { useCapabilitiesData } from "../../hooks/useCapabilitiesData";
import { ServerGroup, FailedServersNotice, AddServerForm } from "./ServersSection";
import { ToolsTabContent } from "./ToolsSection";
import { SkillsSection } from "./SkillsSection";

// CapabilitiesInline — 能力面板内联主体（设置中枢批 4 从 CapabilitiesPanel 拆出
// 单源）：MCP 服务器/工具/技能三 tab 全量逻辑与渲染。CapabilitiesPanel（办公
// 右栏抽屉）与本组件（设置页 office 组）共用，行为零变化——差异仅在壳：
// 抽屉版包 ResizableDrawer+DrawerHeader，内联版顶部自带 summary+reload 工具行
// （原抽屉头部 reload 按钮随主体走，两版同位）。

type CapTab = "servers" | "tools" | "skills";

export function CapabilitiesInline({
  toolCounts = {},
  skillCounts = {},
}: {
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
  // 7.2-2 判据②：持久化技能调用计数（挂载拉一次，零轮询；reload 后随
  // 引擎热加载一并刷新）。
  const [skillUsage, setSkillUsage] = useState<Record<string, { calls: number; ok: number }>>({});
  const refreshSkillUsage = useCallback(() => {
    void app.SkillStats().then((rows: { name: string; calls: number; ok: number }[]) => {
      const out: Record<string, { calls: number; ok: number }> = {};
      for (const r of rows ?? []) out[r.name] = { calls: r.calls, ok: r.ok };
      setSkillUsage(out);
    }).catch(() => null);
  }, []);
  useEffect(() => { refreshSkillUsage() }, [refreshSkillUsage]);

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
    <>
      {/* 工具行：状态摘要 + 重载（原抽屉 DrawerHeader 行内 reload 随主体单源走） */}
      <div className="flex items-center justify-between gap-3 px-4 pt-3">
        {view && <span className="text-[11.5px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{summary}</span>}
        <button
          className={reloadBtn}
          style={{
            border: "1px solid var(--md-sys-color-outline-variant)",
            color: "var(--md-sys-color-text-secondary)",
            background: "transparent",
          }}
          disabled={reloading}
          onClick={() => { void reloadEngine(); refreshSkillUsage() }}
          title={t("caps.reloadHint")}
        >
          {reloading ? (
            <span className="animate-spin inline-block w-3 h-3 border border-current border-t-transparent rounded-full" />
          ) : (
            <RefreshCw size={13} />
          )}
          <span>{reloading ? t("caps.reloading") : t("caps.reload")}</span>
        </button>
      </div>

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
            <SkillsSection skills={view.skills} counts={skillCounts} usage={skillUsage} expanded={expandedSkills} onToggle={toggleSkill} />
          )}
        </div>
      )}
    </>
  );
}
