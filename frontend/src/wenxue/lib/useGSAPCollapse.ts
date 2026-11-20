import { useEffect, useLayoutEffect, useRef } from "react";
import gsap from "gsap";
import { DUR_BASE, EASE_OUT, prefersReducedMotion } from "./gsapAnimations";

/**
 * useGSAPCollapse — animate a container's height between 0 and its
 * scrollHeight whenever `open` flips.  Replaces the old CSS max-height
 * hack with a precise pixel-level GSAP tween.
 *
 * Usage:
 *   const ref = useRef<HTMLDivElement>(null);
 *   useGSAPCollapse(ref, open);
 *   return <div ref={ref} style={{ overflow: "hidden" }}>{children}</div>;
 *
 * The container should have `overflow: hidden` in CSS.
 *
 * 2026-11-20 rAF 冻结兜底（实弹定性）：窗口隐藏/被遮挡时 Chromium 停
 * requestAnimationFrame，GSAP 按帧驱动的 tween 冻结在任意中间高度——
 * 实测已完成过程卡停在 112~380px 半开态；运行中卡更与「最近三条」尾巴
 * 同屏重复（open=false 高度未归零，正文与尾巴各渲染一份最近条目）。
 *
 * 单看 document.hidden 不够：run_skill 拉起外部 Office 窗口时，Wails
 * WebView 常常只是失焦/被遮挡，visibilityState 仍是 visible。此时关闭动画
 * 若停在 initial `height:auto` 或中间高度，返回办公窗口后就会永久摊开。
 * 兜底四路：①隐藏/失焦时跳过 tween 直接落终值；②blur/hidden 瞬间吸附
 * 在途 tween；③focus/pageshow 回收时再吸附一次；④前台 rAF 异常停摆时由
 * 不依赖帧的 watchdog 在动画预算后强制落终值。
 */
export function useGSAPCollapse(
  ref: React.RefObject<HTMLElement | null>,
  open: boolean,
  opts?: {
    duration?: number;
    ease?: string;
    /** Called after the open animation completes. */
    onOpenComplete?: () => void;
    /** Called after the close animation completes. */
    onCloseComplete?: () => void;
    /** When closing, use this height as the starting point instead of
     * measuring scrollHeight (which may have already shrunk due to
     * content being conditionally removed). */
    prevHeight?: number;
  },
) {
  const prevOpen = useRef<boolean | null>(null);
  const onOpenRef = useRef(opts?.onOpenComplete);
  const onCloseRef = useRef(opts?.onCloseComplete);
  const durRef = useRef(opts?.duration ?? DUR_BASE);
  const easeRef = useRef(opts?.ease ?? EASE_OUT);
  const prevHeightRef = useRef(opts?.prevHeight);
  onOpenRef.current = opts?.onOpenComplete;
  onCloseRef.current = opts?.onCloseComplete;
  durRef.current = opts?.duration ?? DUR_BASE;
  easeRef.current = opts?.ease ?? EASE_OUT;
  prevHeightRef.current = opts?.prevHeight;
  // visibilitychange 闭包需要最新 open（监听器只挂一次）。
  const openRef = useRef(open);
  openRef.current = open;
  // 非 null = 当前目标态仍有一次 tween 未确认收尾。被遮挡的 WebView 可能
  // 既收不到可信的 visibilitychange，也永远不推进 rAF，故所有外部恢复
  // 信号与 watchdog 都以这份状态为准。
  const pendingOpenRef = useRef<boolean | null>(null);
  const settleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearSettleTimer = () => {
    if (settleTimerRef.current === null) return;
    clearTimeout(settleTimerRef.current);
    settleTimerRef.current = null;
  };

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;

    // Skip the very first render — don't animate from 0→auto on mount.
    // Use a direct style write (no GSAP overhead) for the initial state.
    if (prevOpen.current === null) {
      prevOpen.current = open;
      pendingOpenRef.current = null;
      el.style.height = open ? "auto" : "0px";
      return;
    }

    // No change — nothing to do.
    if (prevOpen.current === open) return;
    prevOpen.current = open;

    const reduced = prefersReducedMotion();
    // 隐藏态翻转：rAF 已停，tween 起不来（起来也是冻帧）——直接落终值。
    const hidden = typeof document !== "undefined" && document.hidden;
    // Office 外部窗口只带走焦点时 hidden=false；hasFocus 能覆盖这一类
    // Wails/WebView 遮挡，避免关闭动画停在 `auto`。
    const unfocused = typeof document.hasFocus === "function" && !document.hasFocus();
    if (reduced || hidden || unfocused) {
      clearSettleTimer();
      pendingOpenRef.current = null;
      gsap.killTweensOf(el);
      el.style.height = open ? "auto" : "0px";
      if (open) onOpenRef.current?.();
      else onCloseRef.current?.();
      return;
    }

    const dur = reduced ? 0.001 : durRef.current;
    const ease = easeRef.current;

    // Kill any in-flight GSAP animations on this element.
    gsap.killTweensOf(el);
    pendingOpenRef.current = open;

    const finish = () => {
      clearSettleTimer();
      pendingOpenRef.current = null;
      if (open) onOpenRef.current?.();
      else onCloseRef.current?.();
    };

    if (open) {
      // Phase 1 — measure the target (auto) height without visible change.
      gsap.set(el, { height: "auto" });
      const targetHeight = el.scrollHeight;
      // Phase 2 — animate from 0 to the measured target height.
      gsap.fromTo(
        el,
        { height: 0 },
        {
          height: targetHeight,
          duration: dur,
          ease,
          clearProps: "height",
          onComplete: finish,
        },
      );
    } else {
      // Close: if caller provided a pre-swap height use it as the start,
      // otherwise measure the current scrollHeight.
      const startHeight = prevHeightRef.current && prevHeightRef.current > 0
        ? prevHeightRef.current
        : el.scrollHeight;
      gsap.fromTo(
        el,
        { height: startHeight },
        {
          height: 0,
          duration: dur,
          ease,
          onComplete: finish,
        },
      );
    }

    // rAF 若因 WebView 遮挡/合成器异常而永久停摆，visibilitychange 与 blur
    // 都可能没有可靠时序。定时器只负责把“动画已过预算”的卡落到目标终值；
    // 正常 tween 完成时会先清掉它，不改变动画体验。
    clearSettleTimer();
    settleTimerRef.current = setTimeout(() => {
      if (pendingOpenRef.current !== open) return;
      gsap.killTweensOf(el);
      el.style.height = open ? "auto" : "0px";
      finish();
    }, Math.max(250, dur * 1000 + 120));

    return () => {
      clearSettleTimer();
      gsap.killTweensOf(el);
      pendingOpenRef.current = null;
    };
  }, [open, ref]);

  // 外部窗口切换时吸附：blur/hidden 可能发生在 tween 开始前后，focus/
  // pageshow 则覆盖返回办公窗口的时序。只处理尚未确认收尾的 tween，避免
  // 干扰已经稳定的展开态或正常点击动画。
  useEffect(() => {
    const settlePending = () => {
      if (pendingOpenRef.current === null) return;
      const el = ref.current;
      if (!el) return;
      clearSettleTimer();
      pendingOpenRef.current = null;
      gsap.killTweensOf(el);
      el.style.height = openRef.current ? "auto" : "0px";
    };
    const onVisibility = () => {
      if (!document.hidden) return;
      const el = ref.current;
      if (!el) return;
      // hidden 是明确的“rAF 不可信”信号：无论 pending 是否已丢，都按目标态
      // 吸附一次，修复历史半开残留。
      clearSettleTimer();
      pendingOpenRef.current = null;
      gsap.killTweensOf(el);
      el.style.height = openRef.current ? "auto" : "0px";
    };
    const onBlur = () => settlePending();
    const onResume = () => settlePending();
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("blur", onBlur);
    window.addEventListener("focus", onResume);
    window.addEventListener("pageshow", onResume);
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("blur", onBlur);
      window.removeEventListener("focus", onResume);
      window.removeEventListener("pageshow", onResume);
    };
  }, [ref]);
}
