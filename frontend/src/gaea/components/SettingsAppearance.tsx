import { useState } from "react";
import { useI18n } from "../lib/i18n";

// AppearanceSection — 外观设置。主题已统一跟随主应用 darkMode，
// 此处仅提供暗/亮切换入口；字体偏好保留（独立于主题的本地设置）。
export function AppearanceSection({ darkMode, onToggle }: { darkMode: boolean; onToggle: () => void }) {
  const { t } = useI18n();

  // 字体偏好（localStorage + DOM attribute）
  const [uiFont, setUiFont] = useState(() => {
    try { return localStorage.getItem("gaea.uiFont") || ""; } catch { return ""; }
  });
  const [monoFont, setMonoFont] = useState(() => {
    try { return localStorage.getItem("gaea.monoFont") || ""; } catch { return ""; }
  });
  const applyFont = (kind: "ui" | "mono", value: string) => {
    const attr = kind === "ui" ? "data-font-family" : "data-mono-font-family";
    if (value) {
      document.documentElement.setAttribute(attr, value);
      localStorage.setItem(`gaea.${kind === "ui" ? "uiFont" : "monoFont"}`, value);
    } else {
      document.documentElement.removeAttribute(attr);
      localStorage.removeItem(`gaea.${kind === "ui" ? "uiFont" : "monoFont"}`);
    }
    if (kind === "ui") setUiFont(value); else setMonoFont(value);
  };

  return (
    <section className="mb-3">
      <div className="text-fg text-sm font-semibold mb-3">{t("settings.appearance")}</div>

      <div className="mb-4">
        <label className="text-fg-dim text-[13px] font-medium mb-2 block">{t("settings.theme")}</label>
        <div className="grid grid-cols-2 gap-2">
          <button
            onClick={() => { if (darkMode) onToggle(); }}
            className={`text-left bg-bg-soft border rounded-lg p-2.5 cursor-pointer transition-all ${
              darkMode ? "border-accent ring-1 ring-accent/50" : "border-border-soft hover:border-fg-faint/30"
            }`}
          >
            <div className="h-6 rounded-md overflow-hidden mb-2 flex" style={{ background: "#0F172A" }}>
              <div className="flex-1" style={{ background: "#0F172A" }} />
              <div className="w-3" style={{ background: "#6366F1" }} />
              <div className="w-6 flex items-center justify-center text-[7px] font-mono" style={{ background: "#F8FAFC", color: "#0F172A" }}>Aa</div>
            </div>
            <span className={`text-[11px] font-medium ${darkMode ? "text-accent" : "text-fg-dim"}`}>暗色 · Dark</span>
            {darkMode && <span className="ml-1.5 text-[10px] text-accent">✓</span>}
          </button>
          <button
            onClick={() => { if (!darkMode) onToggle(); }}
            className={`text-left bg-bg-soft border rounded-lg p-2.5 cursor-pointer transition-all ${
              !darkMode ? "border-accent ring-1 ring-accent/50" : "border-border-soft hover:border-fg-faint/30"
            }`}
          >
            <div className="h-6 rounded-md overflow-hidden mb-2 flex" style={{ background: "#F8FAFC" }}>
              <div className="flex-1" style={{ background: "#F8FAFC" }} />
              <div className="w-3" style={{ background: "#0D9488" }} />
              <div className="w-6 flex items-center justify-center text-[7px] font-mono" style={{ background: "#0F172A", color: "#F8FAFC" }}>Aa</div>
            </div>
            <span className={`text-[11px] font-medium ${!darkMode ? "text-accent" : "text-fg-dim"}`}>亮色 · Light</span>
            {!darkMode && <span className="ml-1.5 text-[10px] text-accent">✓</span>}
          </button>
        </div>
      </div>

      <div className="mb-4 p-3 bg-bg-soft border border-border-soft rounded-lg">
        <div className="text-[10px] font-semibold text-fg-faint uppercase tracking-wider mb-2">当前色板</div>
        <div className="flex flex-wrap gap-1.5">
          {[
            darkMode ? "#0F172A" : "#F8FAFC",
            darkMode ? "#1E293B" : "#FFFFFF",
            darkMode ? "#6366F1" : "#0D9488",
            darkMode ? "#F8FAFC" : "#0F172A",
            darkMode ? "#CBD5E1" : "#475569",
            darkMode ? "#22C55E" : "#059669",
            darkMode ? "#F59E0B" : "#D97706",
            darkMode ? "#EF4444" : "#DC2626",
          ].map((c, i) => (
            <span key={i} className="w-7 h-7 rounded-md border border-border-soft" style={{ background: c }} title={c} />
          ))}
        </div>
      </div>

      <div className="mb-4 p-3 bg-bg-soft border border-border-soft rounded-lg">
        <div className="text-[10px] font-semibold text-fg-faint uppercase tracking-wider mb-2">界面字体</div>
        <div className="flex gap-2">
          <input
            className="flex-1 min-w-0 bg-bg-elev border border-border-soft rounded-md px-2 py-1.5 text-xs text-fg outline-none focus:border-accent"
            placeholder="默认（系统字体）"
            value={uiFont}
            onChange={(e) => applyFont("ui", e.target.value)}
          />
        </div>
      </div>

      <div className="mb-4 p-3 bg-bg-soft border border-border-soft rounded-lg">
        <div className="text-[10px] font-semibold text-fg-faint uppercase tracking-wider mb-2">等宽字体</div>
        <div className="flex gap-2">
          <input
            className="flex-1 min-w-0 bg-bg-elev border border-border-soft rounded-md px-2 py-1.5 text-xs text-fg outline-none focus:border-accent"
            placeholder="默认（等宽字体）"
            value={monoFont}
            onChange={(e) => applyFont("mono", e.target.value)}
          />
        </div>
      </div>
    </section>
  );
}
