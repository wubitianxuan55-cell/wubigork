import { useCallback, useEffect, useId, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { Globe, Cpu, Plus, RefreshCw, Wand2 } from "../../icons";
import { app } from "../../lib/bridge";
import { useT } from "../../lib/i18n";
import { useCapabilitiesData } from "../../hooks/useCapabilitiesData";
import { ServerGroup, FailedServersNotice, AddServerForm } from "./ServersSection";
import { ToolsTabContent } from "./ToolsSection";
import { SkillsSection } from "./SkillsSection";
import "./capabilities.css";

// CapabilitiesInline — 能力面板内联主体（设置中枢批 4 从 CapabilitiesPanel 拆出
// 单源）：MCP 服务器/工具/技能三 tab 全量逻辑与渲染。CapabilitiesPanel（办公
// 右栏抽屉）与本组件（设置页 office 组）共用，行为零变化——差异仅在壳：
// 抽屉版包 ResizableDrawer+DrawerHeader，内联版顶部自带 summary+reload 工具行
// （原抽屉头部 reload 按钮随主体走，两版同位）。

type CapTab = "servers" | "tools" | "skills";

const TAB_ORDER: CapTab[] = ["servers", "tools", "skills"];

export function CapabilitiesInline({
  toolCounts = {},
  skillCounts = {},
  variant = "inline",
}: {
  toolCounts?: Record<string, number>;
  skillCounts?: Record<string, number>;
  variant?: "drawer" | "inline";
}) {
  const t = useT();
  const tabId = useId();
  const data = useCapabilitiesData();
  const {
    view, busy, err, notice, reloading, adding, addingContext7, confirming,
    setConfirming, setAdding, reloadEngine, mutate, addContext7, summary, serverGroups,
  } = data;
  const [tab, setTab] = useState<CapTab>("servers");
  const [visitedTabs, setVisitedTabs] = useState<Set<CapTab>>(() => new Set(["servers"]));
  const tabRefs = useRef<Record<CapTab, HTMLButtonElement | null>>({
    servers: null,
    tools: null,
    skills: null,
  });
  const [expandedSkills, setExpandedSkills] = useState<Set<string>>(() => new Set());
  const [expandedErrors, setExpandedErrors] = useState<Set<string>>(() => new Set());
  const [expandedServers, setExpandedServers] = useState<Set<string>>(() => new Set());
  // 7.2-2 判据②：持久化技能调用计数（挂载拉一次，零轮询；reload 后随
  // 引擎热加载一并刷新）。
  const [skillUsage, setSkillUsage] = useState<Record<string, { calls: number; ok: number }>>({});
  // 装配面刷新键（2026-11-21 面板缝）：重载引擎后「工具」tab 重拉装配面
  //（点击即 bump 一次——切模式未重载时也能立刻反映登记档，重建完成后再 bump
  // 一次拿新面）。
  const [surfaceKey, setSurfaceKey] = useState(0);
  const bumpSurface = useCallback(() => setSurfaceKey((k) => k + 1), []);
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

  const connectedCount = view?.servers.filter((s) => s.status === "connected").length ?? 0;
  const failedCount = view?.servers.filter((s) => s.status === "failed").length ?? 0;
  const tabItems: { key: CapTab; label: string; shortLabel: string; icon: ReactNode; count?: number }[] = [
    { key: "servers", label: t("caps.connectorsTab"), shortLabel: "MCP", icon: <Globe size={13} aria-hidden />, count: view?.servers.length },
    { key: "tools", label: "工具", shortLabel: "工具", icon: <Cpu size={13} aria-hidden /> },
    { key: "skills", label: t("caps.skillsTab"), shortLabel: "技能", icon: <Wand2 size={13} aria-hidden />, count: view?.skills.length },
  ];

  const selectTab = (next: CapTab, focus = false) => {
    setTab(next);
    setVisitedTabs((prev) => {
      if (prev.has(next)) return prev;
      const updated = new Set(prev);
      updated.add(next);
      return updated;
    });
    if (focus) tabRefs.current[next]?.focus();
  };

  const onTabKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    const current = TAB_ORDER.indexOf(tab);
    let next = current;
    if (event.key === "Home") next = 0;
    else if (event.key === "End") next = TAB_ORDER.length - 1;
    else if (event.key === "ArrowLeft") next = current <= 0 ? TAB_ORDER.length - 1 : current - 1;
    else next = current >= TAB_ORDER.length - 1 ? 0 : current + 1;
    selectTab(TAB_ORDER[next], true);
  };

  return (
    <div className={`caps-shell ${variant === "drawer" ? "caps-shell--drawer" : ""}`}>
      <div className="caps-status">
        <div className="caps-status__metrics" aria-label={summary || t("caps.loading")}>
          <span className="caps-metric" data-tone="success">
            <span className="caps-metric__dot" aria-hidden />
            <strong>{view ? connectedCount : "—"}</strong>
            <span>已连接</span>
          </span>
          <span className="caps-metric-divider" aria-hidden />
          <span className="caps-metric" data-tone={failedCount > 0 ? "danger" : "accent"}>
            <span className="caps-metric__dot" aria-hidden />
            <strong>{view ? failedCount : "—"}</strong>
            <span>异常</span>
          </span>
          <span className="caps-metric-divider" aria-hidden />
          <span className="caps-metric" data-tone="accent">
            <span className="caps-metric__dot" aria-hidden />
            <strong>{view ? view.skills.length : "—"}</strong>
            <span>技能</span>
          </span>
        </div>
        <button
          className="caps-reload"
          type="button"
          disabled={reloading}
          onClick={() => { void reloadEngine().then(bumpSurface); refreshSkillUsage(); bumpSurface() }}
          title={t("caps.reloadHint")}
        >
          {reloading ? (
            <span className="animate-spin inline-block w-3 h-3 border border-current border-t-transparent rounded-full" />
          ) : (
            <RefreshCw size={13} aria-hidden />
          )}
          <span>{reloading ? t("caps.reloading") : t("caps.reload")}</span>
        </button>
      </div>

      <div
        className="caps-tabs"
        role="tablist"
        aria-label={t("caps.title")}
        onKeyDown={(event) => {
          if (event.target instanceof HTMLButtonElement && event.target.getAttribute("role") === "tab") {
            onTabKeyDown(event);
          }
        }}
      >
        {tabItems.map((item) => {
          const active = tab === item.key;
          return (
            <button
              key={item.key}
              ref={(node) => { tabRefs.current[item.key] = node; }}
              className="caps-tab"
              type="button"
              role="tab"
              id={`${tabId}-tab-${item.key}`}
              aria-selected={active}
              aria-controls={`${tabId}-panel-${item.key}`}
              aria-label={item.count !== undefined ? `${item.label}，${item.count}` : item.label}
              title={item.label}
              tabIndex={active ? 0 : -1}
              onClick={() => selectTab(item.key)}
            >
              <span className="caps-tab__icon">{item.icon}</span>
              <span className="caps-tab__label">{item.shortLabel}</span>
              {item.count !== undefined && <span className="caps-tab__count">{item.count}</span>}
            </button>
          );
        })}
      </div>

      {err && <div className="caps-banner" data-tone="error" role="alert">{err}</div>}
      {notice && <div className="caps-banner" data-tone="success" role="status">{notice}</div>}

      {!view ? (
        <div className="caps-loading" aria-label={t("caps.loading")}>
          <div className="caps-loading__line" />
          <div className="caps-loading__line caps-loading__line--short" />
          <div className="caps-loading__card caps-loading__card--tall" />
          <div className="caps-loading__card" />
        </div>
      ) : (
        <>
          <div
            className="caps-panel caps-panel--scroll"
            role="tabpanel"
            id={`${tabId}-panel-servers`}
            aria-labelledby={`${tabId}-tab-servers`}
            hidden={tab !== "servers"}
          >
            <div className="caps-scroll-body">
              <section aria-label={t("caps.connectorsTab")}>
                <div className="caps-server-toolbar">
                  <div className="caps-server-toolbar__copy">
                    <h3 className="caps-server-toolbar__title">{t("caps.connectorsTab")}</h3>
                    <p className="caps-server-toolbar__hint">连接外部服务与资源</p>
                  </div>
                  <div className="caps-server-toolbar__actions">
                    <button
                      className="caps-action"
                      type="button"
                      disabled={busy || addingContext7}
                      onClick={() => addContext7()}
                      title={t("caps.addContext7Hint")}
                    >
                      {addingContext7 ? (
                        <span className="animate-spin inline-block w-3 h-3 border border-current border-t-transparent rounded-full" />
                      ) : (
                        <Globe size={12} aria-hidden />
                      )}
                      <span>{addingContext7 ? t("caps.addContext7Busy") : "Context7"}</span>
                    </button>
                    {!adding && (
                      <button
                        className="caps-action caps-action--accent"
                        type="button"
                        disabled={busy}
                        onClick={() => setAdding(true)}
                      >
                        <Plus size={12} aria-hidden />
                        <span>{t("caps.addServer")}</span>
                      </button>
                    )}
                  </div>
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
                  <div className="caps-empty">
                    <span className="caps-empty__icon" aria-hidden="true"><Globe size={16} /></span>
                    <strong>还没有 MCP 服务器</strong>
                    <p>{t("caps.noServers")}</p>
                  </div>
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
            </div>
          </div>

          <div
            className="caps-panel caps-panel--tools"
            role="tabpanel"
            id={`${tabId}-panel-tools`}
            aria-labelledby={`${tabId}-tab-tools`}
            hidden={tab !== "tools"}
          >
            {visitedTabs.has("tools") && (
              <ToolsTabContent toolCounts={toolCounts} refreshKey={surfaceKey} board="gaea" />
            )}
          </div>

          <div
            className="caps-panel caps-panel--scroll"
            role="tabpanel"
            id={`${tabId}-panel-skills`}
            aria-labelledby={`${tabId}-tab-skills`}
            hidden={tab !== "skills"}
          >
            <div className="caps-scroll-body">
              {visitedTabs.has("skills") && (
                <SkillsSection skills={view.skills} counts={skillCounts} usage={skillUsage} expanded={expandedSkills} onToggle={toggleSkill} />
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
