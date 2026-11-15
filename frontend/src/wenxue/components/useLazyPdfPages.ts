// PDF 逐页懒加载的 IO 接线 hook（审计 P1 FE1-01）：FilePreviewModal（v4.32 C）
// 与 FilePreview（v4.33.0 C 对齐弹窗）此前的 pageElsRef/observer/lazyPdf/
// anchor/aspect/scrollTo 四件套逐行同构存在两份、注释互指——收敛到本 hook，
// 两处各留一行调用。纯函数判定仍收敛在 lib/pageLazy.ts（不动）。
import { useCallback, useEffect, useRef, useState } from "react";
import type { PreviewResult } from "../lib/types";
import {
  LAZY_ROOT_MARGIN_PX,
  addForcedPage,
  computeInitialLazyPages,
  expandMountedPages,
  lazySupported,
  nextPageAspect,
} from "../lib/pageLazy";

/** 单档 PDF 的懒加载状态（src 记录所属 preview，换载荷时渲染期重置）。 */
export interface LazyPdfState {
  src: PreviewResult | null;
  mounted: ReadonlySet<number>;
  forced: ReadonlySet<number>;
  /** 文档级实测宽高比（首个有效测量固定）；null = 无测量，占位回落 A4 估计。 */
  aspect: number | null;
}

export function useLazyPdfPages(
  containerRef: React.RefObject<HTMLElement | null>,
  preview: PreviewResult | null,
  pdfPageCount: number,
) {
  const pageElsRef = useRef(new Map<number, HTMLElement>());
  const pdfObserverRef = useRef<IntersectionObserver | null>(null);
  const [lazyPdf, setLazyPdf] = useState<LazyPdfState>(() => ({
    src: null, mounted: new Set<number>(), forced: new Set<number>(), aspect: null,
  }));
  const lazyPdfPages = pdfPageCount > 0 && lazySupported();
  // preview 换载荷时渲染期重置（初始窗口 + 清空强制/测量集合，React「props
  // 变化时调整 state」模式，避免 effect 时序上的旧集合闪烁）。
  if (lazyPdf.src !== preview) {
    setLazyPdf({
      src: preview,
      mounted: pdfPageCount > 0 ? computeInitialLazyPages(pdfPageCount) : new Set<number>(),
      forced: new Set<number>(),
      aspect: null,
    });
  }

  // 页图容器注册表：figure 挂载即登记（data-pptx-page 既是大纲滚动锚点也是
  // IO 目标键），卸载时清掉已断连的条目；登记时若观察器已存在则立即补
  // observe，保证任意挂载顺序都进观察集。
  const pdfPageAnchorRef = useCallback((el: HTMLElement | null) => {
    const els = pageElsRef.current;
    if (el) {
      const page = Number(el.getAttribute("data-pptx-page"));
      if (page > 0) {
        els.set(page, el);
        pdfObserverRef.current?.observe(el);
      }
      return;
    }
    for (const [page, node] of els) {
      if (!node.isConnected) {
        pdfObserverRef.current?.unobserve(node);
        els.delete(page);
      }
    }
  }, []);

  // 页图 onLoad 记录本档文档真实宽高比（naturalWidth/naturalHeight 与容器宽
  // 无关、jsdom 可 stub）；测量无变化时返回同一引用，setState 凭引用跳过重渲染。
  const handlePdfPageLoad = useCallback((e: React.SyntheticEvent<HTMLImageElement>) => {
    const { naturalWidth, naturalHeight } = e.currentTarget;
    setLazyPdf((prev) => {
      const aspect = nextPageAspect(prev.aspect, naturalWidth, naturalHeight);
      return aspect === prev.aspect ? prev : { ...prev, aspect };
    });
  }, []);

  // IntersectionObserver 接线：页容器进入视口（rootMargin 800px，root 为调用方
  // 滚动容器）→ 并入挂载集合挂真身 <img>。IO 缺失时本 effect 直接跳过。
  useEffect(() => {
    if (!lazyPdfPages || typeof IntersectionObserver === "undefined") return;
    const io = new IntersectionObserver(
      (entries) => {
        const visible: number[] = [];
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          const page = Number((entry.target as HTMLElement).getAttribute("data-pptx-page"));
          if (Number.isFinite(page) && page > 0) visible.push(page);
        }
        if (visible.length === 0) return;
        setLazyPdf((prev) => ({
          ...prev,
          mounted: expandMountedPages(prev.mounted, visible, pdfPageCount),
        }));
      },
      { root: containerRef.current, rootMargin: `${LAZY_ROOT_MARGIN_PX}px` },
    );
    pdfObserverRef.current = io;
    for (const el of pageElsRef.current.values()) io.observe(el);
    return () => {
      io.disconnect();
      pdfObserverRef.current = null;
    };
  }, [lazyPdfPages, pdfPageCount, containerRef]);

  // 点大纲页条目 → 滚动到对应页锚点：编程式跳转的目标页强制渲染真身（占位
  // 高度是估计值，停在占位盒会跳偏；先并入强制渲染集合再滚）。
  const scrollToPptxPage = useCallback((page: number) => {
    setLazyPdf((prev) => ({ ...prev, forced: addForcedPage(prev.forced, page) }));
    const el = containerRef.current?.querySelector<HTMLElement>(`[data-pptx-page="${page}"]`);
    el?.scrollIntoView?.({ block: "start", behavior: "smooth" });
  }, [containerRef]);

  return { lazyPdf, setLazyPdf, lazyPdfPages, pdfPageAnchorRef, handlePdfPageLoad, scrollToPptxPage };
}
