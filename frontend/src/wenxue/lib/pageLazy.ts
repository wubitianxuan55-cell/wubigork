// v4.32 C 弹窗 PDF 逐页懒加载（收 v4.31 欠账「弹窗 pdf 不虚拟化」）。
//
// Why: FilePreviewModal 的 kind=pdf 分支原先一次性渲染全部 ≤60 页 <img>
// （base64 大图），弹窗打开即卡。页类型 PreviewPageThumb 只有 { page, dataUrl }
// 没有页宽高，做不了精准占位的双向虚拟化；因此采用 IntersectionObserver
// 单向懒加载：初始只渲染首窗页，进入视口（rootMargin 800px）才挂 <img>，
// 已挂载页永不卸载（双向卸载会让滚动条跳动，不做）。
//
// v4.33 B（收 v4.32 欠账「弹窗 pdf 占位高为 A4 估计值」）：占位盒比例不再
// 永远是 A4 估计值——某页 <img> onLoad 时用 naturalWidth/naturalHeight（图片
// 真实像素比，与容器宽无关，jsdom 可 stub，不用 getBoundingClientRect）测出
// 本档文档真实宽高比记入 per-document 状态，未测量页的占位盒改用该比例；
// 无任何测量时仍回落 LAZY_PAGE_ASPECT。存比例（height/width）而非绝对高度，
// 容器宽度变化（拖拽分割条）时占位依然成立。测量收敛为纯函数
// nextPageAspect / placeholderAspect。
//
// How: 「初始渲染哪些页 / 某页是否渲染 / IO 触发后集合如何扩张」全部收敛为
// 纯函数（对齐 lib/paletteRank.ts 先例），FilePreviewModal 只做 IO 接线与
// state 持有。jsdom / 旧环境无 IntersectionObserver 时组件走全量渲染降级
// （= v4.31 行为），本模块的 lazySupported() 即该判定。

/** IO rootMargin：视口外 800px 内的页提前挂载，抵消快速滚动时的白屏。 */
export const LAZY_ROOT_MARGIN_PX = 800;

/** 初始渲染窗口：打开弹窗先挂前 4 页（首屏 + 少量预备），其余为占位盒。 */
export const INITIAL_LAZY_WINDOW = 4;

/** 挂载 buffer：某页进入视口时，连同其前后各 1 页一并挂载（预挂邻页，
 *  翻过当前页时下一页已在）。空间 buffer 由 IO rootMargin 兜底，这里取 1。 */
export const LAZY_MOUNT_BUFFER = 1;

/** 占位盒回落宽高比（A4 纵向 1/1.414）。PreviewPageThumb 无尺寸信息；
 *  v4.33 B 起页图 onLoad 会测量本档真实比例（见 nextPageAspect），
 *  仅在无任何测量时回落到此估计值。 */
export const LAZY_PAGE_ASPECT = "1 / 1.414";

/** 当前环境是否支持 IntersectionObserver（jsdom / 旧 webview → 全量降级）。 */
export function lazySupported(): boolean {
  return typeof IntersectionObserver !== "undefined";
}

/** 初始懒加载窗口：第 1..min(windowSize, totalPages) 页视为已挂载，
 *  其余页渲染占位盒等 IO 触发。totalPages ≤ 0 时返回空集。 */
export function computeInitialLazyPages(
  totalPages: number,
  windowSize = INITIAL_LAZY_WINDOW,
): Set<number> {
  const pages = new Set<number>();
  for (let p = 1; p <= Math.min(windowSize, totalPages); p++) pages.add(p);
  return pages;
}

/** 单向扩张已挂载集合：把 visiblePages（本次进入视口的页）连同其前后
 *  buffer 邻页并入 current，绝不移除既有成员（单向懒加载，杜绝滚动跳动）。
 *  越界页裁剪到 1..totalPages。无新增时原样返回 current（调用方 setState
 *  可凭同一引用跳过重渲染）；有新增时返回新 Set，不改入参。 */
export function expandMountedPages(
  current: ReadonlySet<number>,
  visiblePages: readonly number[],
  totalPages: number,
  buffer = LAZY_MOUNT_BUFFER,
): ReadonlySet<number> {
  let next: Set<number> | null = null;
  for (const v of visiblePages) {
    if (v < 1 || v > totalPages) continue;
    for (let p = Math.max(1, v - buffer); p <= Math.min(totalPages, v + buffer); p++) {
      if (current.has(p)) continue;
      if (!next) next = new Set(current);
      next.add(p);
    }
  }
  return next ?? current;
}

/** 某页是否渲染真身 <img>：已挂载（初始窗口或 IO 触发过）或被强制渲染
 *  （大纲卡跳转目标）。两者合并即最终渲染集合。 */
export function shouldRenderLazyPage(
  page: number,
  mounted: ReadonlySet<number>,
  forced: ReadonlySet<number>,
): boolean {
  return mounted.has(page) || forced.has(page);
}

/** 把大纲卡跳转目标页并入强制渲染集合：编程式 scrollIntoView 落点页的真身
 *  必须立即可见——占位高度是估计值，若目标页还停留在占位盒，滚动会跳偏。
 *  页已在集合中时原样返回（setState 凭同一引用跳过重渲染）。 */
export function addForcedPage(forced: ReadonlySet<number>, page: number): ReadonlySet<number> {
  if (forced.has(page)) return forced;
  const next = new Set(forced);
  next.add(page);
  return next;
}

// v4.33 B：占位盒真实比例测量（纯函数）。同一 PDF 各页尺寸通常一致，取
// 「首个有效测量」为整档比例且不被后续页推翻（简单确定性：多数派/均值需
// 排序计数，收益抵不过复杂度）；无效测量（naturalWidth/Height 非正或非
// 有限值，如图片尚未解码）不记录。存比例而非绝对高度：容器宽度（拖拽分割
// 条）变化时比例仍成立，占位盒 style 直填 aspectRatio: placeholderAspect(...)。

/** 把某页 <img> onLoad 测得的图片真实像素比并入 per-document 测量状态：
 *  prev 为 null 且本次测量有效 → 记为文档比例（height/width）；已有测量或
 *  本次测量无效（宽/高非正、非有限值）→ 原样返回 prev（首个有效测量固定，
 *  确定性，不被后续页推翻）。 */
export function nextPageAspect(
  prev: number | null,
  naturalWidth: number,
  naturalHeight: number,
): number | null {
  if (prev !== null) return prev;
  if (
    !Number.isFinite(naturalWidth) ||
    !Number.isFinite(naturalHeight) ||
    naturalWidth <= 0 ||
    naturalHeight <= 0
  ) {
    return prev;
  }
  return naturalHeight / naturalWidth;
}

/** 占位盒 aspectRatio 字符串：有测量用测量比例（String 化后可直填 style），
 *  无测量回落 A4 估计值 LAZY_PAGE_ASPECT。 */
export function placeholderAspect(measured: number | null): string {
  return measured === null ? LAZY_PAGE_ASPECT : String(measured);
}
