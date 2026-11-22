import { useEffect, useState } from "react";
import { normalizeBrowserUrl } from "../lib/browserPolicy";
import { ArrowUp, ChevronRight, ExternalLink, Globe, RefreshCw, Shield } from "../icons";

// 浏览器视图 tab（对标 better-sidebar BrowserView）：地址栏 + 沙箱 iframe。
// 安全模型：iframe 永不带 allow-same-origin / allow-top-navigation；地址栏只
// 收 http(s)，拒绝 loopback；URL 经 pane tab meta 随会话持久化。
const BROWSER_IFRAME_SANDBOX =
  "allow-scripts allow-forms allow-popups allow-downloads allow-modals allow-popups-to-escape-sandbox";

const ICON_BTN =
  "browser-pane__icon-button flex items-center justify-center w-7 h-7 rounded-md border border-transparent bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors disabled:opacity-35 disabled:cursor-default";

export function BrowserPane({
  url,
  onUrlChange,
}: {
  url?: string;
  onUrlChange: (url: string) => void;
}) {
  const [input, setInput] = useState(url ?? "");
  const [history, setHistory] = useState<string[]>(url ? [url] : []);
  const [cursor, setCursor] = useState(url ? 0 : -1);
  const [reloadKey, setReloadKey] = useState(0);
  const [message, setMessage] = useState<string | null>(null);
  const [unlock, setUnlock] = useState(false);

  // 会话恢复（pane tab meta）或外部改变 URL 时同步输入框
  useEffect(() => {
    if (url !== undefined) setInput(url);
  }, [url]);

  const navigateTo = (raw: string, push = true): void => {
    const result = normalizeBrowserUrl(raw);
    if (result.kind === "ok") {
      const next = result.url;
      setMessage(null);
      setReloadKey((k) => k + 1);
      onUrlChange(next);
      if (push) {
        setHistory((prev) => [...prev.slice(0, cursor + 1), next]);
        setCursor((c) => c + 1);
      }
      return;
    }
    setMessage(
      result.kind === "invalid"
        ? "无效的网址"
        : result.reason === "scheme"
          ? "已阻止：仅支持 http/https 链接"
          : "已阻止：不允许访问本机或内部地址",
    );
  };

  const goBack = (): void => {
    if (cursor <= 0) return;
    const next = history[cursor - 1];
    if (next) {
      setCursor(cursor - 1);
      navigateTo(next, false);
    }
  };
  const goForward = (): void => {
    if (cursor >= history.length - 1) return;
    const next = history[cursor + 1];
    if (next) {
      setCursor(cursor + 1);
      navigateTo(next, false);
    }
  };

  const current = cursor >= 0 ? history[cursor] : url;
  return (
    <div className="browser-pane flex flex-col h-full min-h-0 text-[12px]">
      <div className="browser-pane__toolbar flex items-center gap-1 px-2 py-1.5 border-b border-border-soft">
        <button type="button" className={ICON_BTN} aria-label="后退" disabled={cursor <= 0} onClick={goBack}>
          <ChevronRight size={13} style={{ transform: "rotate(180deg)" }} aria-hidden />
        </button>
        <button type="button" className={ICON_BTN} aria-label="前进" disabled={cursor >= history.length - 1} onClick={goForward}>
          <ChevronRight size={13} aria-hidden />
        </button>
        <button
          type="button"
          className={ICON_BTN}
          aria-label="刷新"
          onClick={() => { if (url) setReloadKey((k) => k + 1); }}
        >
          <RefreshCw size={13} aria-hidden />
        </button>
        <label className="browser-pane__address flex-1 min-w-0 flex items-center gap-1.5 h-7 px-2 rounded-md border border-border-soft bg-bg-soft/45 focus-within:border-accent transition-colors">
          <Globe size={11} aria-hidden className="shrink-0 text-fg-faint" />
          <input
            value={input}
            placeholder="输入网址，例如 example.com"
            spellCheck={false}
            aria-label="浏览器地址"
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") navigateTo(input);
            }}
            className="min-w-0 flex-1 border-0 bg-transparent text-fg placeholder:text-fg-faint/70 outline-none"
          />
        </label>
        <button type="button" className={ICON_BTN} aria-label="前往" onClick={() => navigateTo(input)}>
          <ArrowUp size={13} style={{ transform: "rotate(90deg)" }} aria-hidden />
        </button>
        {current && (
          <button
            type="button"
            className={ICON_BTN}
            aria-label="在浏览器中打开"
            title="在浏览器中打开"
            onClick={() => window.open(current, "_blank", "noopener")}
          >
            <ExternalLink size={13} aria-hidden />
          </button>
        )}
      </div>
      {message && (
        <div role="alert" className="browser-pane__message px-3 py-1.5 text-[11px] text-(color:--md-sys-color-text-secondary) bg-(color:--md-sys-color-surface-container-high)">
          {message}
        </div>
      )}
      <div className="browser-pane__security flex items-center gap-2 px-3 py-1 border-b border-border-soft text-[10.5px] text-fg-dim">
        <Shield size={11} aria-hidden className="shrink-0" />
        <span
          className="w-1.5 h-1.5 rounded-full"
          style={{ background: unlock ? "var(--md-sys-color-error)" : "var(--color-success)" }}
        />
        <span className="truncate">
          {unlock
            ? "沙箱已关闭：页面与界面同源（仅在临时解锁期间）"
            : "沙箱模式：已启用 · 页面无法访问界面数据与本地文件"}
        </span>
        {!unlock && (
          <button
            type="button"
            className="browser-pane__unlock ml-auto shrink-0 rounded px-1.5 py-0.5 border border-border-soft bg-transparent text-fg-dim cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
            onClick={() => setUnlock(true)}
          >
            临时解锁（不安全）
          </button>
        )}
        {unlock && (
          <button
            type="button"
            className="browser-pane__unlock ml-auto shrink-0 rounded px-1.5 py-0.5 border border-border-soft bg-transparent text-fg-dim cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
            onClick={() => setUnlock(false)}
          >
            恢复沙箱
          </button>
        )}
      </div>
      <div className="flex-1 min-h-0">
        {url ? (
          <iframe
            key={`${url}:${reloadKey}:${unlock ? "u" : "s"}`}
            className="w-full h-full border-0"
            src={url}
            sandbox={unlock ? undefined : BROWSER_IFRAME_SANDBOX}
            referrerPolicy="no-referrer"
            allow=""
            title={url}
          />
        ) : (
          <div className="browser-pane__empty flex h-full flex-col items-center justify-center gap-3 px-6 text-center">
            <span className="browser-pane__empty-icon" aria-hidden>
              <Globe size={22} />
            </span>
            <span className="flex flex-col gap-1">
              <strong>在沙箱中打开网页</strong>
              <span>输入公开网址后回车，页面不会接触本地文件或界面数据</span>
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
