// SpaceChip.tsx — S4 双空间最小接线（设计 docs/gaea-space-dimension-design.md §6）：
// 侧边栏显示当前生效空间 + 切换入口。GaeaSpaceActivate 只写配置（非法 space
// 由 Go 侧拒绝），生效时机为下次引擎重建/重启；绑定不可用（旧后端）时整块
// 不渲染，不影响侧边栏其余功能。
import { useEffect, useState } from "react";
import { Layers } from "../icons";
import { useT } from "../lib/i18n";
import { app } from "../lib/bridge";
import { noteSpaceActivated } from "../lib/useSpaceScope";
import type { SpaceActiveView, SpaceOption } from "../lib/types";

export default function SpaceChip({ disabled }: { disabled?: boolean }) {
  const t = useT();
  const [active, setActive] = useState<SpaceActiveView | null>(null);
  const [options, setOptions] = useState<SpaceOption[]>([]);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const [v, opts] = await Promise.all([app.GaeaSpaceActive(), app.GaeaSpaceList()]);
        if (!cancelled) {
          setActive(v);
          setOptions(opts);
        }
      } catch {
        /* 绑定不可用（旧后端/异常）：入口隐藏 */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  if (!active) return null;

  const activate = async (id: string) => {
    if (disabled || busy || id === active.space) return;
    setBusy(true);
    try {
      const v = await app.GaeaSpaceActivate(id);
      // S1.2-C：广播给检索面（记忆中枢搜索/搜索面板），默认 scope 随之更新。
      noteSpaceActivated(v);
      setActive(v);
    } catch {
      /* 激活失败保持现状（Go 侧已拒绝非法值） */
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      className="flex items-center gap-2.5 min-h-9 w-full px-3 rounded-full border border-transparent text-fg-dim text-[13px] no-drag select-none"
      title={t("sidebar.spaceHint")}
    >
      <Layers size={15} className="shrink-0 text-fg-faint" />
      <span className="text-fg-faint shrink-0">{t("sidebar.space")}</span>
      <div className="flex flex-1 min-w-0 justify-end gap-1">
        {options.map((o) => {
          const on = o.id === active.space;
          return (
            <button
              key={o.id}
              className={`shrink-0 rounded-full px-2 py-0.5 text-[11px] border-0 cursor-pointer transition-colors duration-[var(--dur-fast)] disabled:cursor-default ${
                on
                  ? "bg-accent/12 text-accent"
                  : "bg-transparent text-fg-faint hover:text-fg hover:bg-sidebar-hover"
              } ${busy && !on ? "opacity-50" : ""}`}
              onClick={() => void activate(o.id)}
              disabled={busy || on}
              title={o.desc}
            >
              {o.title}
            </button>
          );
        })}
      </div>
    </div>
  );
}
