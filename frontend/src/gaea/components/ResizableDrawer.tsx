import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, KeyboardEvent, PointerEvent as ReactPointerEvent, ReactNode } from "react";
import { useT } from "../lib/i18n";
import { loadLayoutSize, saveLayoutSize, type LayoutSizeKey } from "../lib/layoutPreferences";

const DRAWER_DEFAULT_WIDTH = 440;
const DRAWER_MIN_WIDTH = 360;
const DRAWER_MAX_WIDTH = 760;
const DRAWER_MAX_RATIO = 0.62;
const SETTINGS_DRAWER_DEFAULT_WIDTH = 720;
const SETTINGS_DRAWER_MIN_WIDTH = 620;
const SETTINGS_DRAWER_MAX_WIDTH = 1120;
const SETTINGS_DRAWER_MAX_RATIO = 0.82;

function drawerConfig(wide: boolean) {
  return wide
    ? {
        key: "settingsDrawerWidth" as LayoutSizeKey,
        defaultWidth: SETTINGS_DRAWER_DEFAULT_WIDTH,
        minWidth: SETTINGS_DRAWER_MIN_WIDTH,
        maxWidth: SETTINGS_DRAWER_MAX_WIDTH,
        maxRatio: SETTINGS_DRAWER_MAX_RATIO,
      }
    : {
        key: "drawerWidth" as LayoutSizeKey,
        defaultWidth: DRAWER_DEFAULT_WIDTH,
        minWidth: DRAWER_MIN_WIDTH,
        maxWidth: DRAWER_MAX_WIDTH,
        maxRatio: DRAWER_MAX_RATIO,
      };
}

function clampDrawerWidth(width: number, wide: boolean, viewportWidth = 1440): number {
  const config = drawerConfig(wide);
  const maxByViewport = Math.floor(viewportWidth * config.maxRatio);
  const max = Math.max(config.minWidth, Math.min(config.maxWidth, maxByViewport));
  return Math.min(max, Math.max(config.minWidth, Math.round(width)));
}

export function ResizableDrawer({
  children,
  onClose,
  subtle = false,
  wide = false,
  label,
}: {
  children: ReactNode;
  onClose: () => void;
  subtle?: boolean;
  wide?: boolean;
  /** 弹层可访问名（v4.352 补：role=dialog 必须有名）；缺省退「面板」 */
  label?: string;
}) {
  const t = useT();
  const [exiting, setExiting] = useState(false);
  const config = drawerConfig(wide);
  const drawerRef = useRef<HTMLElement>(null);
  // 焦点还原：打开时记录当前焦点元素，卸载时还原（v4.352；ApprovalModal 先例）
  const prevFocusRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    prevFocusRef.current = document.activeElement as HTMLElement | null;
    // 开时聚焦抽屉容器（tabIndex=-1，Tab 流从内容第一个可聚焦元素继续）
    drawerRef.current?.focus();
    return () => {
      // 关闭动画后真实卸载：焦点还原到打开者（元素可能已消失，try 兜底）
      try { prevFocusRef.current?.focus(); } catch { /* 元素已卸载则忽略 */ }
    };
  }, []);

  // Intercept close: play exit animation, then call the real onClose.
  const handleClose = useCallback(() => {
    if (exiting) return;
    setExiting(true);
    setTimeout(() => onClose(), 120); // matches drawer-out duration
  }, [exiting, onClose]);

  // Esc 关闭（对照 ApprovalModal：输入控件内的 Esc 不劫持）。放在 handleClose
  // 声明之后——effect 回调虽延迟执行，依赖数组读 handleClose 绑定是即时的（TDZ）。
  useEffect(() => {
    const onKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key !== "Escape") return;
      const target = event.target as HTMLElement | null;
      if (target?.isContentEditable) return;
      event.preventDefault();
      handleClose();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [handleClose]);
  const [viewportWidth, setViewportWidth] = useState(() => (typeof window === "undefined" ? 1440 : window.innerWidth));
  const [width, setWidth] = useState(() =>
    loadLayoutSize(config.key, config.defaultWidth, (value) => clampDrawerWidth(value, wide)),
  );
  const [resizing, setResizing] = useState(false);
  const effectiveWidth = useMemo(() => clampDrawerWidth(width, wide, viewportWidth), [viewportWidth, wide, width]);
  const style = useMemo(() => ({ "--drawer-width": `${effectiveWidth}px` }) as CSSProperties, [effectiveWidth]);

  useEffect(() => {
    const onResize = () => setViewportWidth(window.innerWidth);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  const saveWidth = useCallback(
    (nextWidth: number) => {
      const next = clampDrawerWidth(nextWidth, wide, viewportWidth);
      setWidth(next);
      saveLayoutSize(config.key, next);
    },
    [config.key, viewportWidth, wide],
  );

  const startResize = useCallback(
    (event: ReactPointerEvent<HTMLButtonElement>) => {
      if (event.button !== 0) return;
      event.preventDefault();
      setResizing(true);
      let nextWidth = effectiveWidth;
      const onMove = (moveEvent: PointerEvent) => {
        nextWidth = clampDrawerWidth(window.innerWidth - moveEvent.clientX, wide, window.innerWidth);
        setWidth(nextWidth);
      };
      const onDone = () => {
        setWidth(nextWidth);
        saveLayoutSize(config.key, nextWidth);
        setResizing(false);
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onDone);
        window.removeEventListener("pointercancel", onDone);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onDone);
      window.addEventListener("pointercancel", onDone);
    },
    [config.key, effectiveWidth, wide],
  );

  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLButtonElement>) => {
      if (event.key === "ArrowLeft" || event.key === "ArrowRight") {
        event.preventDefault();
        saveWidth(effectiveWidth + (event.key === "ArrowLeft" ? 16 : -16));
      } else if (event.key === "Home") {
        event.preventDefault();
        saveWidth(config.minWidth);
      } else if (event.key === "End") {
        event.preventDefault();
        saveWidth(config.maxWidth);
      }
    },
    [config.maxWidth, config.minWidth, effectiveWidth, saveWidth],
  );

  return (
    <div className="fixed inset-0 z-90">
      {/* transparent click-to-close layer */}
      <div className="absolute inset-0" onClick={handleClose} />
      {/* visual backdrop only — does not intercept scroll */}
      <div className={"absolute inset-0 pointer-events-none " + (subtle ? "bg-bg/16" : "bg-bg/60")} />
      {/* positioning layer */}
      <div className="absolute inset-0 flex justify-end pointer-events-none">
      <aside
        ref={drawerRef}
        role="dialog"
        aria-modal="true"
        aria-label={label ?? t("drawer.resize")}
        tabIndex={-1}
        className={"relative flex flex-col h-full bg-bg-elev border-l border-border pointer-events-auto outline-none " + (
          exiting ? "anim-drawer-out" : "anim-drawer-in"
        ) + (resizing ? " drawer--resizing" : "") + " " + (
          wide
            ? "w-[min(var(--drawer-width,720px),94vw)]"
            : "w-[min(var(--drawer-width,440px),92vw)]"
        )}
        onClick={(e) => e.stopPropagation()}
        style={{...style, boxShadow: "var(--ds-shadow-panel)"}}
      >
        <button
          className="absolute top-0 bottom-0 left-[-4px] z-[4] w-2 p-0 border-0 bg-transparent cursor-col-resize no-drag group"
          type="button"
          role="separator"
          aria-orientation="vertical"
          aria-label={t("drawer.resize")}
          aria-valuemin={config.minWidth}
          aria-valuemax={config.maxWidth}
          aria-valuenow={effectiveWidth}
          onPointerDown={startResize}
          onKeyDown={onKeyDown}
          onDoubleClick={() => saveWidth(config.defaultWidth)}
          title={t("drawer.resize")}
        >
          <span className="absolute top-0 bottom-0 left-1/2 w-px -translate-x-1/2 bg-transparent transition-[background,box-shadow] duration-[var(--dur-fast)] group-hover:bg-accent group-hover:shadow-[0_0_0_1px_color-mix(in_srgb,var(--accent)_24%,transparent)] group-focus-visible:bg-accent group-focus-visible:shadow-[0_0_0_1px_color-mix(in_srgb,var(--accent)_24%,transparent)] pointer-events-none" />
        </button>
        {children}
      </aside>
      </div>
    </div>
  );
}
