import { useEffect, useState } from "react";
import { subscribeWailsEvent } from "../lib/wailsEvents";
import "./computer-status.css";

// ComputerStatusBar — 电脑操控状态条（施工单 W6）：监听后端
// computer-status 事件（state=using|idle，app=process:<exe>，window=标题），
// 动作期间在会话顶部显示「正在操控 <窗口>」，动作收口后自动消失。
// 事件不走 base64；截图一律落盘 .gaea/uploads 由工具回执 markdown 呈现。

interface ComputerStatusPayload {
  state?: string;
  app?: string;
  window?: string;
  turn_id?: string;
}

function appDisplayName(app?: string): string {
  if (!app) return "";
  const p = app.startsWith("process:") ? app.slice("process:".length) : app;
  const base = p.split(/[\\/]/).pop() ?? p;
  return base.replace(/\.[^.]+$/, "") || p;
}

export function ComputerStatusBar() {
  const [using, setUsing] = useState<ComputerStatusPayload | null>(null);

  useEffect(() => {
    const rt = (window as unknown as { runtime?: unknown }).runtime;
    if (!rt) return undefined;
    const off = subscribeWailsEvent(
      rt as Parameters<typeof subscribeWailsEvent>[0],
      "computer-status",
      (raw: unknown) => {
        const ev = (raw ?? {}) as ComputerStatusPayload;
        if (ev.state === "using") {
          setUsing(ev);
        } else if (ev.state === "idle") {
          setUsing(null);
        }
      },
    );
    return () => {
      off();
    };
  }, []);

  if (!using) return null;
  const name = using.window || appDisplayName(using.app) || "目标窗口";
  return (
    <div
      className="computer-status-bar"
      role="status"
      aria-live="polite"
      data-testid="computer-status-bar"
    >
      <span className="computer-status-bar__dot" aria-hidden="true" />
      <span>正在操控 {name}</span>
      <span className="computer-status-bar__app">{appDisplayName(using.app)}</span>
    </div>
  );
}
