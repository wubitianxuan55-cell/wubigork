/* eslint-disable react-refresh/only-export-components -- BrowserObserveView/extractBrowserActions 等导出供 bridge 接线与测试复用（AnsweredByLine.tsx 同款模式） */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Eye, Globe, Loader, RefreshCw } from "../icons";
import { app } from "../lib/bridge";
import type { Trajectory } from "../lib/types";
import { usePollingGate } from "../../hooks/usePollingGate";
import { loadBrowserAutoOpen, saveBrowserAutoOpen } from "../lib/browserPrefs";
import { relativeTime } from "../lib/time";
import { Z_INDEX } from "../../utils/zIndex";

// BrowserPanel — 浏览器观察窗（v4.28 A2，docs/gaea-office-upgrade-plan-2026-09.md
// A2 + docs/research-2026-09-01/browser-observation.md）。受控 Edge 的「截图步进
// 流」观察面（CDP Page.captureScreenshot 单帧，不做实时帧流——帧流/人工接管属
// 远期），三段式：
//   ①顶部 URL/标题行 + 截图（点击放大到内置 zoom 覆盖层）；
//   ②操作时间线（对标 Playwright Trace Viewer 的 Actions 列表）：过滤当前会话
//     轨迹里的 browser_* 工具记录（名称/参数摘要/时间），倒序上限 20；
//   ③权限状态行：「只读观察 / 写入需批准」静态说明（动态权限卡属权限门后续）。
// 数据源（store.ts 零改动）：组件内自取——观察走 GaeaBrowserObserve（被动动作，
// 浏览器未运行返回 available=false 绝不拉起），时间线走 Trajectory() 过滤；
// 二者都留 props 注入 seam 供测试。刷新节奏：2.5s 自动轮询仅当页面可见且观察
// 可用（usePollingGate 门控）；不可见或未运行时退化为手动刷新。
// 「自动弹出」（gaea 差异化）：偏好持久化在 lib/browserPrefs（gaea.browserAutoOpen，
// 默认开），面板内提供开关胶囊；「新 browser_* 工具出现 → 自动切到本 tab」的
// 触发接线由 App 消费 shouldAutoOpenBrowser() 完成。

/** 后端 browser.ObserveView（internal/gaea/browser/screenshot.go）的前端镜像，
 *  json tag 全键小驼峰，与 Go 字段逐一对应。 */
export interface BrowserObserveView {
  available: boolean;
  url: string;
  title: string;
  /** 截图 data URL（"data:image/jpeg;base64,…"）；截图失败时为空串。 */
  image: string;
  width: number;
  height: number;
  /** 本帧观察时刻（Unix 毫秒）。 */
  updatedAt: number;
  /** 截图失败原因（available=true 但帧缺失时非空）。 */
  error?: string;
}

/** 操作时间线一行：一次 browser_* 工具调用的摘要。 */
export interface BrowserActionEntry {
  seq: number;
  name: string;
  args: string;
  ts: number;
  status: "ok" | "error" | "running";
  err?: string;
}

/** 时间线行数上限（Trace Viewer Actions 同款截断：只看最近动作）。 */
const TIMELINE_LIMIT = 20;
/** 自动轮询间隔（ms）：截图步进流的「步进」节奏。 */
const POLL_MS = 2500;

const UNAVAILABLE_VIEW: BrowserObserveView = {
  available: false,
  url: "",
  title: "",
  image: "",
  width: 0,
  height: 0,
  updatedAt: 0,
};

/** 默认观察数据源：bridge 的 GaeaBrowserObserve。bridge.ts（主代理维护）尚未
 *  声明该方法的类型，这里做运行时探测——绑定就绪（Wails 门面挂载）后直连；
 *  未就绪/旧后端/dev mock 按不可用兜底（绝不因观察缺失抛错）。 */
function defaultObserve(): Promise<BrowserObserveView> {
  const cand = (app as unknown as Record<string, unknown>).GaeaBrowserObserve;
  if (typeof cand !== "function") return Promise.resolve({ ...UNAVAILABLE_VIEW });
  return Promise.resolve()
    .then(() => (cand as () => Promise<BrowserObserveView>)())
    .catch(() => ({ ...UNAVAILABLE_VIEW }));
}

/** 从会话轨迹提取 browser_* 工具记录：倒序（新→旧）上限 20。 */
export function extractBrowserActions(traj: Trajectory | null): BrowserActionEntry[] {
  if (!traj) return [];
  const records = [
    ...(traj.turns ?? []).flatMap((t) => t.records ?? []),
    ...(traj.betweenTurns ?? []),
  ];
  const out: BrowserActionEntry[] = [];
  for (const rec of records) {
    const tool = rec.kind === "tool" ? rec.tool : undefined;
    if (!tool || !tool.name.startsWith("browser_")) continue;
    out.push({
      seq: rec.seq,
      name: tool.name,
      args: summarizeArgs(tool.args),
      ts: rec.ts,
      status: tool.status,
      err: tool.err,
    });
  }
  out.sort((a, b) => b.ts - a.ts || b.seq - a.seq);
  return out.slice(0, TIMELINE_LIMIT);
}

/** 参数摘要：JSON 对象取前 3 个非空字段 "k=v" 串联；解析失败按原文展示。
 *  超长截断（时间线是扫一眼的摘要，全量参数在轨迹视图）。 */
function summarizeArgs(raw: string | undefined): string {
  const text = (raw ?? "").trim();
  if (!text) return "";
  let summary = text;
  try {
    const obj = JSON.parse(text) as Record<string, unknown>;
    const parts: string[] = [];
    for (const [k, v] of Object.entries(obj)) {
      if (v === null || v === undefined || v === "") continue;
      parts.push(`${k}=${typeof v === "object" ? JSON.stringify(v) : String(v)}`);
      if (parts.length >= 3) break;
    }
    if (parts.length > 0) summary = parts.join(" · ");
  } catch {
    /* 非 JSON 参数：按原文 */
  }
  return summary.length > 80 ? `${summary.slice(0, 80)}…` : summary;
}

export function BrowserPanel({
  observe,
  fetchTrajectory,
}: {
  /** 观察数据源（默认 bridge 的 GaeaBrowserObserve；测试注入）。 */
  observe?: () => Promise<BrowserObserveView>;
  /** 轨迹数据源（默认 bridge 的 Trajectory；测试注入）——时间线过滤用。 */
  fetchTrajectory?: () => Promise<Trajectory>;
}) {
  const [view, setView] = useState<BrowserObserveView | null>(null);
  const [actions, setActions] = useState<BrowserActionEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [zoom, setZoom] = useState(false);
  // 自动弹出偏好（默认开，localStorage 持久化，键 gaea.browserAutoOpen）。
  const [autoOpen, setAutoOpen] = useState(() => loadBrowserAutoOpen());

  // 轮询防重叠与门控：页面不可见（gate=false）或浏览器未运行时 tick 空转，
  // 退化为手动刷新；in-flight 时跳过本拍（宁慢勿堵）。
  const gate = usePollingGate();
  const viewRef = useRef(view);
  viewRef.current = view;
  const busyRef = useRef(false);

  const load = useCallback(() => {
    if (busyRef.current) return;
    busyRef.current = true;
    setLoading(true);
    // 单轮询并行拉两源（学 SubagentsPanel）：任一失败各自降级不拖垮另一侧。
    void Promise.all([
      (observe ?? defaultObserve)().catch(() => ({ ...UNAVAILABLE_VIEW })),
      Promise.resolve(fetchTrajectory ? fetchTrajectory() : app.Trajectory()).catch(() => null),
    ])
      .then(([v, traj]) => {
        setView(v);
        setActions(extractBrowserActions(traj));
      })
      .finally(() => {
        busyRef.current = false;
        setLoading(false);
      });
  }, [observe, fetchTrajectory]);

  useEffect(() => {
    // tick 门控：挂载首拍必拉（探测可用性）；此后仅「可见且可用」才自动轮询。
    const tick = () => {
      const cur = viewRef.current;
      if (cur !== null && (!gate || !cur.available)) return;
      load();
    };
    tick();
    const timer = window.setInterval(tick, POLL_MS);
    return () => window.clearInterval(timer);
  }, [load, gate]);

  // Esc 关闭放大层。
  useEffect(() => {
    if (!zoom) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setZoom(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [zoom]);

  const toggleAutoOpen = useCallback(() => {
    setAutoOpen((prev) => {
      const next = !prev;
      saveBrowserAutoOpen(next);
      return next;
    });
  }, []);

  const url = view?.url ?? "";
  const title = view?.title ?? "";
  const statusText = useMemo(() => {
    if (!view) return "读取观察帧…";
    if (!view.available) return "受控浏览器未运行";
    return view.error ? `观察降级：${view.error}` : relativeTime(view.updatedAt);
  }, [view]);

  const iconBtn =
    "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* 细条头部：标题 + 帧龄 + 自动弹出胶囊 + 刷新（对齐分工面板头部形状） */}
      <div className="v3-panel-head">
        <Globe size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">浏览器</span>
        <span className="v3-panel-spacer" />
        <button
          type="button"
          data-testid="browser-auto-open-toggle"
          className="inline-flex shrink-0 cursor-pointer items-center gap-1 rounded-full px-1.5 py-px text-[10px] leading-none transition-colors"
          aria-pressed={autoOpen}
          title={autoOpen
            ? "自动弹出已开：出现新 browser 操作时自动切到本面板（点击关闭）"
            : "自动弹出已关：browser 操作只更新数据不切换面板（点击开启）"}
          onClick={toggleAutoOpen}
          style={autoOpen
            ? {
                background: "color-mix(in srgb, var(--gaea-glow) 12%, transparent)",
                color: "var(--gaea-glow)",
                border: "1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)",
              }
            : {
                background: "transparent",
                color: "var(--md-sys-color-text-secondary)",
                border: "1px solid var(--md-sys-color-outline-variant)",
              }}
        >
          <span
            className="inline-block h-1.5 w-1.5 rounded-full"
            style={{ background: autoOpen ? "var(--gaea-glow)" : "var(--md-sys-color-outline-variant)" }}
            aria-hidden
          />
          自动弹出 {autoOpen ? "开" : "关"}
        </button>
        <button type="button" className={iconBtn} onClick={load} title="刷新观察帧" aria-label="刷新观察帧">
          <Loader size={12} className={loading ? "animate-spin" : "hidden"} />
          <RefreshCw size={12} className={loading ? "hidden" : ""} />
        </button>
      </div>

      {!view ? (
        <div className="flex flex-1 items-center justify-center gap-2 text-[11px]">
          <Loader size={14} className="animate-spin" />
          读取观察帧…
        </div>
      ) : !view.available ? (
        /* 可用性门控：浏览器未运行 → 空态（绝不拉起，浏览器工具执行时自动拉起） */
        <div className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center">
          <Eye size={24} aria-hidden className="opacity-40" />
          <span className="text-[11px] leading-relaxed" data-testid="browser-empty">
            受控浏览器未运行
            <br />
            浏览器工具执行时自动拉起，这里会实时跟随页面
          </span>
        </div>
      ) : (
        <div className="flex flex-1 flex-col gap-2 overflow-y-auto min-h-0 p-2">
          {/* ① URL/标题行 + 截图（点击放大） */}
          <div className="flex flex-col gap-px rounded-md px-1.5 py-1" style={{ background: "color-mix(in srgb, var(--gaea-glow) 6%, transparent)" }}>
            <span data-testid="browser-title" className="truncate text-[11px] font-medium" style={{ color: "var(--md-sys-color-text)" }} title={title || url}>
              {title || "（无标题页面）"}
            </span>
            <span data-testid="browser-url" className="truncate font-mono text-[10px]" title={url} style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {url}
            </span>
            <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {statusText}
              {view.width > 0 ? ` · ${view.width}×${view.height}` : ""}
            </span>
          </div>
          {view.image ? (
            <img
              data-testid="browser-shot"
              src={view.image}
              alt={`页面截图：${title || url}`}
              className="w-full cursor-zoom-in rounded-md"
              style={{ border: "1px solid var(--md-sys-color-outline-variant)" }}
              onClick={() => setZoom(true)}
            />
          ) : view.error ? (
            <div data-testid="browser-shot-error" className="rounded-md px-2 py-3 text-center text-[11px]" style={{ border: "1px dashed var(--md-sys-color-outline-variant)" }}>
              {view.error}
            </div>
          ) : null}

          {/* ② 操作时间线（对标 Trace Viewer Actions 列表）：browser_* 记录倒序 */}
          <div className="flex flex-col gap-1" data-testid="browser-timeline">
            <div className="flex items-center gap-1.5 px-0.5 text-[10px] font-medium" style={{ color: "var(--gaea-glow)" }}>
              操作时间线
              <span className="font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>{actions.length}</span>
            </div>
            {actions.length === 0 ? (
              <span className="px-1 py-1 text-[10.5px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                本会话暂无 browser_* 操作记录
              </span>
            ) : (
              actions.map((a) => (
                <div
                  key={a.seq}
                  data-testid="browser-timeline-row"
                  className="flex flex-col gap-px rounded-md px-1.5 py-1 text-[10.5px] leading-relaxed"
                  style={{ border: "1px solid var(--md-sys-color-outline-variant)" }}
                >
                  <span className="flex items-center gap-1.5">
                    <span
                      className="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
                      aria-hidden
                      style={{ background: a.status === "error" ? "var(--md-sys-color-error)" : "var(--gaea-glow)" }}
                    />
                    <span className="font-mono font-medium" style={{ color: "var(--md-sys-color-text)" }}>{a.name}</span>
                    <span className="ml-auto shrink-0 text-[10px]">{relativeTime(a.ts)}</span>
                  </span>
                  {a.args && (
                    <span className="truncate pl-3 font-mono text-[10px]" title={a.args} style={{ color: "var(--md-sys-color-text-secondary)" }}>
                      {a.args}
                    </span>
                  )}
                  {a.err && (
                    <span className="truncate pl-3 text-[10px]" title={a.err} style={{ color: "var(--md-sys-color-error)" }}>
                      {a.err}
                    </span>
                  )}
                </div>
              ))
            )}
          </div>

          {/* ③ 权限状态行：静态说明（browser 工具 readOnly 判定已有；动态权限卡属权限门后续） */}
          <div
            data-testid="browser-perm-note"
            className="flex items-center gap-1.5 rounded-md px-1.5 py-1 text-[10px]"
            style={{ background: "color-mix(in srgb, var(--gaea-glow) 6%, transparent)", color: "var(--md-sys-color-text-secondary)" }}
          >
            <Eye size={11} aria-hidden style={{ color: "var(--gaea-glow)" }} />
            只读观察 · 写入操作（点击/输入/导航）需批准后执行
          </div>
        </div>
      )}

      {/* 截图放大层（内置轻量 zoom：Lightbox 组件是绘图域专用、必填 onDownload/
          onReuse/onSetPortrait 等回调，不适用观察帧，故面板内自实现） */}
      {zoom && view?.image ? (
        <div
          data-testid="browser-zoom"
          className="fixed inset-0 flex items-center justify-center"
          style={{ zIndex: Z_INDEX.MODAL + 10, background: "rgba(0,0,0,0.9)" }}
          onClick={() => setZoom(false)}
        >
          <img
            src={view.image}
            alt="页面截图放大"
            className="max-h-[95vh] max-w-[95vw] rounded-md object-contain"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      ) : null}
    </div>
  );
}
