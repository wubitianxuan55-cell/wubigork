// WpsDockControl.tsx — P1-4 WPS 面板停靠控制（dock 档）：
// 把一个 WPS 文档窗口摆到预览面板矩形上（borderless），面板缩放跟随，
// 组件卸载/手动还原恢复顶层窗口。embed（SetParent）是第二步，默认不做。
// 坐标换算：DOM rect(CSS px) × devicePixelRatio = 屏幕物理像素（WebView2
// 全窗即视口，window.screenX/Y + rect 即屏幕 CSS 坐标）。
import { useCallback, useEffect, useRef, useState } from "react";
import { Scan, Rollback } from "../icons";
import { app } from "../lib/bridge";
import type { WpsDockWindow } from "../lib/bridge/office";

const HEAD_BTN =
  "inline-flex items-center gap-1 px-1.5 py-1 rounded border border-transparent hover:border-border-soft hover:bg-bg-soft text-fg-faint hover:text-fg transition-colors shrink-0";

interface Props {
  /** 停靠目标面板：缺省用最近祖先 .preview-pane（FilePreview 挂载点）。 */
  paneSelector?: string;
}

export function WpsDockControl({ paneSelector = ".preview-pane" }: Props) {
  const ref = useRef<HTMLSpanElement | null>(null);
  const [docked, setDocked] = useState<WpsDockWindow | null>(null);
  const [menu, setMenu] = useState<WpsDockWindow[] | null>(null);
  const [note, setNote] = useState("");
  const dockedRef = useRef<WpsDockWindow | null>(null);
  dockedRef.current = docked;

  const paneRect = useCallback((): { x: number; y: number; w: number; h: number } | null => {
    const el = ref.current?.closest(paneSelector) as HTMLElement | null;
    if (!el) return null;
    const r = el.getBoundingClientRect();
    if (r.width <= 5 || r.height <= 5) return null; // 面板折叠：跳过
    const dpr = window.devicePixelRatio || 1;
    return {
      x: Math.round((window.screenX + r.left) * dpr),
      y: Math.round((window.screenY + r.top) * dpr),
      w: Math.round(r.width * dpr),
      h: Math.round(r.height * dpr),
    };
  }, [paneSelector]);

  const dock = useCallback(async (win: WpsDockWindow) => {
    const rect = paneRect();
    if (!rect) {
      setNote("面板不可见或已折叠，先展开预览面板");
      return;
    }
    try {
      await app.WpsDockApply?.(win.hwnd, rect.x, rect.y, rect.w, rect.h, true);
      setDocked(win);
      setMenu(null);
      setNote("");
    } catch (e) {
      setNote(String(e instanceof Error ? e.message : e));
    }
  }, [paneRect]);

  const restore = useCallback(async (hwnd: number | null) => {
    const target = hwnd ?? dockedRef.current?.hwnd;
    if (target == null) return;
    try {
      await app.WpsDockRestore?.(target);
    } catch {
      /* 还原失败（窗口已关）：状态照清，不留死停靠态 */
    }
    setDocked(null);
    setMenu(null);
  }, []);

  // 面板缩放跟随：ResizeObserver 只在停靠态生效；rAF 合帧防抖。
  useEffect(() => {
    if (!docked) return;
    const el = ref.current?.closest(paneSelector) as HTMLElement | null;
    if (!el || typeof ResizeObserver === "undefined") return;
    let raf = 0;
    const ro = new ResizeObserver(() => {
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(() => {
        const win = dockedRef.current;
        const rect = paneRect();
        if (win && rect) {
          app.WpsDockApply?.(win.hwnd, rect.x, rect.y, rect.w, rect.h, true).catch(() => {});
        }
      });
    });
    ro.observe(el);
    return () => {
      ro.disconnect();
      cancelAnimationFrame(raf);
    };
  }, [docked, paneSelector, paneRect]);

  // 卸载兜底：面板关闭/切文件时还原，不留边界外的无框窗口。
  useEffect(() => () => {
    const win = dockedRef.current;
    if (win) app.WpsDockRestore?.(win.hwnd).catch(() => {});
  }, []);

  const listDocs = useCallback(async () => {
    try {
      const wins = (await app.WpsDockList?.()) ?? [];
      const docs = wins.filter((w) => w.isDocWin && w.visible);
      if (docs.length === 0) {
        setNote("没找到 WPS 文档窗口——先在 WPS 里打开文字/表格");
        return;
      }
      if (docs.length === 1) {
        await dock(docs[0]);
        return;
      }
      setMenu(docs); // 多个文档窗口：让用户点名
    } catch (e) {
      setNote(String(e instanceof Error ? e.message : e));
    }
  }, [dock]);

  return (
    <span ref={ref} className="relative inline-flex items-center shrink-0">
      {docked ? (
        <>
          <span
            className="max-w-[140px] truncate text-fg-faint text-[10px]"
            title={`已停靠：${docked.title || docked.class}`}
          >
            已停靠 {docked.title || docked.class}
          </span>
          <button
            className={HEAD_BTN}
            onClick={() => restore(null)}
            title="还原 WPS 窗口（取消面板停靠）"
            aria-label="还原 WPS 窗口"
          >
            <Rollback size={10} />
          </button>
        </>
      ) : (
        <button className={HEAD_BTN} onClick={listDocs} title="把 WPS 文档窗口停靠到预览面板" aria-label="停靠 WPS 窗口">
          <Scan size={10} />
          停靠 WPS
        </button>
      )}
      {menu && menu.length > 1 && (
        <span className="absolute top-full right-0 z-20 mt-1 min-w-[220px] rounded-md border border-border-soft bg-bg-elev1 shadow-lg py-1 max-h-[240px] overflow-auto">
          {menu.map((w) => (
            <button
              key={String(w.hwnd)}
              className="w-full text-left px-2.5 py-1.5 text-fg hover:bg-bg-soft truncate"
              onClick={() => dock(w)}
              title={w.title || w.class}
            >
              {w.title || w.class}
              <span className="text-fg-faint ml-1 text-[10px]">{w.proc}</span>
            </button>
          ))}
          <button
            className="w-full text-left px-2.5 py-1.5 text-fg-faint hover:bg-bg-soft"
            onClick={() => setMenu(null)}
          >
            取消
          </button>
        </span>
      )}
      {note && (
        <span className="absolute top-full right-0 z-20 mt-1 px-2 py-1 rounded-md border border-amber-400/30 bg-amber-400/10 text-amber-500 text-[10px] max-w-[240px]" title={note}>
          {note}
          <button className="ml-1 underline" onClick={() => setNote("")} aria-label="关闭提示">×</button>
        </span>
      )}
    </span>
  );
}
