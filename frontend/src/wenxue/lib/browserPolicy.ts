// 浏览器视图地址栏策略（对标 dsh-better-sidebar client/browser.ts）：
// 只允许 http/https；javascript:/data:/file: 等一律拒；loopback 拒
// （防已加载页面探测本机服务）。纯函数，便于单测。

export type BrowserNavigateResult =
  | { kind: "ok"; url: string }
  | { kind: "blocked"; reason: "scheme" | "loopback" }
  | { kind: "invalid" };

const FORBIDDEN_SCHEMES = new Set([
  "javascript", "data", "file", "about", "vbscript", "blob",
  "mailto", "tel", "ftp", "ftps", "ws", "wss", "sftp", "ssh",
  "chrome", "chrome-extension", "moz-extension", "edge", "resource", "view-source",
]);

function isLoopback(hostname: string): boolean {
  const host = hostname.replace(/^\[|\]$/g, "").toLowerCase();
  if (host === "localhost" || host === "::1" || host === "0.0.0.0") return true;
  const parts = host.split(".");
  return (
    parts.length === 4 &&
    parts[0] === "127" &&
    parts.every((p) => /^\d{1,3}$/.test(p) && Number(p) <= 255)
  );
}

export function normalizeBrowserUrl(input: string): BrowserNavigateResult {
  const trimmed = input.trim();
  if (trimmed === "") return { kind: "invalid" };
  const schemeMatch = /^([a-zA-Z][a-zA-Z0-9+.-]*):/.exec(trimmed);
  let withScheme: string;
  if (schemeMatch === null) {
    withScheme = `https://${trimmed}`;
  } else {
    const scheme = schemeMatch[1]!.toLowerCase();
    if (scheme === "http" || scheme === "https") withScheme = trimmed;
    else if (FORBIDDEN_SCHEMES.has(scheme)) return { kind: "blocked", reason: "scheme" };
    else withScheme = `https://${trimmed}`;
  }
  let url: URL;
  try {
    url = new URL(withScheme);
  } catch {
    return { kind: "invalid" };
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    return { kind: "blocked", reason: "scheme" };
  }
  if (isLoopback(url.hostname)) return { kind: "blocked", reason: "loopback" };
  return { kind: "ok", url: url.href };
}

// ── 渲染文档外链分流（1c）───────────────────────────────────────────
// Markdown/HTML 渲染出的 <a> 点击统一走此判定（此前对任意 href 直接
// openExternal——javascript:/file: 等会原样交给系统，是真实风险面）。
// http/https → 系统浏览器（loopback 拒：渲染文档不得探测本机服务）；
// mailto/tel → 交系统处理器（无 webview 风险，办公文档常见）；其余
// （javascript:/data:/file:/blob:/相对路径等）→ blocked。

export type ExternalLinkDecision =
  | { kind: "open"; url: string }
  | { kind: "blocked"; reason: "scheme" | "loopback" };

export function classifyExternalLink(href: string): ExternalLinkDecision {
  const trimmed = (href ?? "").trim();
  const schemeMatch = /^([a-zA-Z][a-zA-Z0-9+.-]*):/.exec(trimmed);
  if (schemeMatch === null) return { kind: "blocked", reason: "scheme" };
  const scheme = schemeMatch[1]!.toLowerCase();
  if (scheme === "mailto" || scheme === "tel") return { kind: "open", url: trimmed };
  if (scheme !== "http" && scheme !== "https") return { kind: "blocked", reason: "scheme" };
  try {
    const url = new URL(trimmed);
    if (isLoopback(url.hostname)) return { kind: "blocked", reason: "loopback" };
    return { kind: "open", url: url.href };
  } catch {
    return { kind: "blocked", reason: "scheme" };
  }
}
