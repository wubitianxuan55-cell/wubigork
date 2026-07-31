// theme.ts manages the appearance override. The stylesheet is dark by default
// and follows the OS via prefers-color-scheme; this lets the user force a theme
// by setting data-theme on <html>, or "auto" to remove it and follow the OS.
// The choice persists in localStorage and is applied on load.
//
// 6 built-in themes: 3 dark (slate, earth, noir) + 3 light (paper, sand, mist)

export type Theme = "auto" | "slate" | "earth" | "noir" | "paper" | "sand" | "mist";

const KEY = "gaea-theme";

function normalizeTheme(value: unknown): Theme | null {
  if (typeof value === "object" && value !== null) {
    return normalizeTheme((value as { mode?: unknown }).mode);
  }
  if (typeof value !== "string") return null;
  switch (value) {
    case "auto":
      return "auto";
    // New names
    case "slate":
      return "slate";
    case "earth":
      return "earth";
    case "noir":
      return "noir";
    case "paper":
      return "paper";
    case "sand":
      return "sand";
    case "mist":
      return "mist";
    // Legacy name migration — map to closest new name
    case "dark":
    case "contrast":
    case "midnight":
    case "neon":
    case "mono":
    case "ice":       // 冰蓝 → slate（相近的暗冷色）
    case "forest":    // 森林 → slate（暗色默认）
      return "slate";
    case "light":
    case "focus":
    case "warm":      // 暖色 → sand（相近的暖亮色）
      return "sand";
    default:
      return null;
  }
}

export function getTheme(): Theme {
  const v = typeof localStorage !== "undefined" ? localStorage.getItem(KEY) : null;
  if (!v) return "auto";
  try {
    const parsed = JSON.parse(v) as unknown;
    return normalizeTheme(parsed) ?? normalizeTheme(v) ?? "auto";
  } catch {
    return normalizeTheme(v) ?? "auto";
  }
}

export function applyTheme(theme: Theme): void {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  root.removeAttribute("data-theme-mode");
  root.removeAttribute("data-theme-scheme");
  if (theme === "auto") root.removeAttribute("data-theme");
  else root.setAttribute("data-theme", theme);
  try {
    localStorage.setItem(KEY, theme);
  } catch {
    /* private mode / no storage — the in-DOM attribute still applies */
  }
}

// initTheme applies the saved choice once at startup (before React renders).
export function initTheme(): void {
  applyTheme(getTheme());
}
