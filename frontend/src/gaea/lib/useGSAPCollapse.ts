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
 * 兜底两路：①open 翻转发生在隐藏态（含 reduced-motion）时跳过 tween
 * 直接写终值；②visibilitychange→hidden 瞬间杀掉在途 tween 吸附到目标
 * 高度（隐藏期 rAF 本就不再推进，吸附即终态；回窗无半开残留）。
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

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;

    // Skip the very first render — don't animate from 0→auto on mount.
    // Use a direct style write (no GSAP overhead) for the initial state.
    if (prevOpen.current === null) {
      prevOpen.current = open;
      el.style.height = open ? "auto" : "0px";
      return;
    }

    // No change — nothing to do.
    if (prevOpen.current === open) return;
    prevOpen.current = open;

    const reduced = prefersReducedMotion();
    // 隐藏态翻转：rAF 已停，tween 起不来（起来也是冻帧）——直接落终值。
    const hidden = typeof document !== "undefined" && document.hidden;
    if (reduced || hidden) {
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
          onComplete: () => onOpenRef.current?.(),
        },
      );
    } else {
      // Close: if caller provided a pre-swap height use it as the start,
      // otherwise measure the current scrollHeight.
      const startHeight = prevHeightRef.current && prevHeightRef.current > 0
        ? prevHeightRef.current
        : (gsap.set(el, { height: "auto" }), el.scrollHeight);
      gsap.fromTo(
        el,
        { height: startHeight },
        {
          height: 0,
          duration: dur,
          ease,
          onComplete: () => onCloseRef.current?.(),
        },
      );
    }
  }, [open, ref]);

  // 隐藏瞬间吸附：在途 tween 在隐藏期不会推进（rAF 停），停在中间高度
  // 就是实弹的半开卡+尾巴重复。visibilitychange→hidden 时杀 tween 直接
  // 写目标高度（当前 openRef 即目标）。
  useEffect(() => {
    const onVisibility = () => {
      if (!document.hidden) return;
      const el = ref.current;
      if (!el) return;
      gsap.killTweensOf(el);
      el.style.height = openRef.current ? "auto" : "0px";
    };
    document.addEventListener("visibilitychange", onVisibility);
    return () => document.removeEventListener("visibilitychange", onVisibility);
  }, [ref]);
}
