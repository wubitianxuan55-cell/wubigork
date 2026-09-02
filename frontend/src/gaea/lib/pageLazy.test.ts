import { describe, expect, it } from "vitest";
import {
  LAZY_PAGE_ASPECT,
  addForcedPage,
  computeInitialLazyPages,
  expandMountedPages,
  lazySupported,
  nextPageAspect,
  placeholderAspect,
  shouldRenderLazyPage,
} from "./pageLazy";

describe("lazySupported", () => {
  it("无 IntersectionObserver 的环境（jsdom 默认）返回 false，组件走全量降级", () => {
    expect(lazySupported()).toBe(typeof IntersectionObserver !== "undefined");
  });

  it("挂上 IntersectionObserver 后返回 true", () => {
    const g = globalThis as unknown as { IntersectionObserver?: unknown };
    const prev = g.IntersectionObserver;
    g.IntersectionObserver = class {};
    try {
      expect(lazySupported()).toBe(true);
    } finally {
      if (prev === undefined) delete g.IntersectionObserver;
      else g.IntersectionObserver = prev;
    }
  });
});

describe("computeInitialLazyPages", () => {
  it("总页数充足时初始窗口 = 前 INITIAL_LAZY_WINDOW 页", () => {
    expect(computeInitialLazyPages(60)).toEqual(new Set([1, 2, 3, 4]));
  });

  it("总页数不足窗口时全量（≤4 页文档实际不懒加载差距）", () => {
    expect(computeInitialLazyPages(2)).toEqual(new Set([1, 2]));
  });

  it("无页时返回空集", () => {
    expect(computeInitialLazyPages(0).size).toBe(0);
    expect(computeInitialLazyPages(-3).size).toBe(0);
  });

  it("窗口大小可覆盖（组件测试/未来调整用）", () => {
    expect(computeInitialLazyPages(10, 1)).toEqual(new Set([1]));
    expect(computeInitialLazyPages(3, 8)).toEqual(new Set([1, 2, 3]));
  });
});

describe("expandMountedPages", () => {
  it("进入视口的页连同前后 buffer 邻页一并挂载", () => {
    expect(expandMountedPages(new Set([1, 2]), [5], 60)).toEqual(new Set([1, 2, 4, 5, 6]));
  });

  it("单向：绝不移除既有成员", () => {
    const current = new Set([1, 2, 3, 9]);
    const next = expandMountedPages(current, [6], 60);
    expect(next).toEqual(new Set([1, 2, 3, 5, 6, 7, 9]));
  });

  it("边界裁剪：首页 / 末页邻页不越界", () => {
    expect(expandMountedPages(new Set<number>(), [1], 60)).toEqual(new Set([1, 2]));
    expect(expandMountedPages(new Set<number>(), [60], 60)).toEqual(new Set([59, 60]));
  });

  it("越界的可见页（脏数据）被忽略", () => {
    expect(expandMountedPages(new Set<number>(), [0, -1, 61], 60).size).toBe(0);
  });

  it("buffer 可覆盖为 0：只挂进入视口的页本身", () => {
    expect(expandMountedPages(new Set<number>(), [5], 60, 0)).toEqual(new Set([5]));
  });

  it("无新增时原样返回同一引用（setState 凭引用跳过重渲染）", () => {
    const current = new Set([4, 5, 6]);
    expect(expandMountedPages(current, [5], 60)).toBe(current);
    expect(expandMountedPages(current, [], 60)).toBe(current);
  });

  it("有新增时返回新集合，不改入参", () => {
    const current = new Set([1]);
    const next = expandMountedPages(current, [3], 60);
    expect(current).toEqual(new Set([1]));
    expect(next).toEqual(new Set([1, 2, 3, 4]));
  });

  it("连续扩张幂等收敛：重复触发同一页不产生新成员", () => {
    let mounted: ReadonlySet<number> = new Set<number>([1, 2, 3, 4]);
    mounted = expandMountedPages(mounted, [5], 60);
    const after5 = mounted;
    mounted = expandMountedPages(mounted, [5], 60);
    expect(mounted).toBe(after5);
  });
});

describe("shouldRenderLazyPage", () => {
  it("已挂载或强制渲染任一命中即渲染真身", () => {
    const mounted = new Set([1, 2]);
    const forced = new Set([7]);
    expect(shouldRenderLazyPage(1, mounted, forced)).toBe(true);
    expect(shouldRenderLazyPage(7, mounted, forced)).toBe(true);
    expect(shouldRenderLazyPage(3, mounted, forced)).toBe(false);
  });

  it("两个集合皆空时全不渲染（懒加载初始前由初始窗口兜底）", () => {
    expect(shouldRenderLazyPage(1, new Set(), new Set())).toBe(false);
  });
});

describe("addForcedPage", () => {
  it("并入目标页，不改入参", () => {
    const forced = new Set<number>();
    const next = addForcedPage(forced, 7);
    expect(forced.size).toBe(0);
    expect(next).toEqual(new Set([7]));
  });

  it("重复添加幂等：已存在时返回同一引用", () => {
    const forced = new Set([7]);
    expect(addForcedPage(forced, 7)).toBe(forced);
  });

  it("强制渲染集合与挂载集合合并生效（大纲跳转未触达页也渲染真身）", () => {
    const mounted = computeInitialLazyPages(8);
    const forced = addForcedPage(new Set<number>(), 7);
    expect(shouldRenderLazyPage(7, mounted, forced)).toBe(true);
    expect(shouldRenderLazyPage(6, mounted, forced)).toBe(false);
  });
});

// v4.33 B（收 v4.32 欠账「弹窗 pdf 占位高为 A4 估计值」）：页图 onLoad 测量
// 本档文档真实宽高比，未测量页占位盒按测量比例撑高，无测量回落 A4 估计值。
describe("nextPageAspect", () => {
  it("首个有效测量：naturalHeight/naturalWidth 记为文档比例", () => {
    expect(nextPageAspect(null, 1000, 1414)).toBeCloseTo(1.414);
    expect(nextPageAspect(null, 800, 600)).toBe(0.75);
  });

  it("无效测量（0 / 负数 / 非有限值）不记录：prev 为 null 时仍为 null", () => {
    expect(nextPageAspect(null, 0, 100)).toBeNull();
    expect(nextPageAspect(null, 100, 0)).toBeNull();
    expect(nextPageAspect(null, -3, 100)).toBeNull();
    expect(nextPageAspect(null, Number.NaN, 100)).toBeNull();
    expect(nextPageAspect(null, 100, Number.POSITIVE_INFINITY)).toBeNull();
  });

  it("首个有效测量固定：后续页测得不同比例不推翻（确定性）", () => {
    const first = nextPageAspect(null, 800, 600);
    expect(nextPageAspect(first, 1000, 2000)).toBe(first);
  });

  it("已有测量时任何输入原样返回 prev（含无效测量）", () => {
    const measured = 1.5;
    expect(nextPageAspect(measured, 0, 0)).toBe(measured);
    expect(nextPageAspect(measured, 999, 999)).toBe(measured);
  });
});

describe("placeholderAspect", () => {
  it("有测量：String 化比例，可直填 aspectRatio", () => {
    expect(placeholderAspect(0.75)).toBe("0.75");
    expect(placeholderAspect(1.414)).toBe("1.414");
  });

  it("无测量：回落 A4 估计值 LAZY_PAGE_ASPECT", () => {
    expect(placeholderAspect(null)).toBe(LAZY_PAGE_ASPECT);
  });
});
