// sin/sinThumbCache.ts — 原罪图片 data URL 的有界 LRU 缓存（v4.427）。
//
// 缩略图/大图/过程卡产物都要经附件通道把本地路径读成 data URL（WebView2 里
// 本地路径不能直接 img src）。面板常驻化后每张图数 MB 的 base64 字符串会随
// 组件 state 驻留，长故事几十张 = 数百 MB——这里给读取统一加一道容量上限的
// LRU：热图留在缓存，冷图重新读盘（重开画廊的代价是几次桥调用，可接受）。
// 只约束缓存本身；组件 state 持有的引用随卸载释放（面板收起即卸载卡体）。

import { readFileAsDataURL } from '../../api/image'

/** 缓存条目上限（每条是一张图的 data URL，按张数封顶足够——单图尺寸另有生成侧约束）。 */
const SIN_THUMB_CACHE_CAP = 16

const cache = new Map<string, string>() // 插入序即近用序（命中即重插到尾）

/** 读图（命中缓存免桥调用；未命中读盘并入缓存，超出上限淘汰最久未用）。 */
export async function readSinThumb(path: string): Promise<string> {
  const hit = cache.get(path)
  if (hit !== undefined) {
    cache.delete(path)
    cache.set(path, hit)
    return hit
  }
  const url = await readFileAsDataURL(path)
  cache.set(path, url)
  while (cache.size > SIN_THUMB_CACHE_CAP) {
    const oldest = cache.keys().next().value
    if (oldest === undefined) break
    cache.delete(oldest)
  }
  return url
}

/** 测试隔离：清空缓存（生产不使用）。 */
export function __resetSinThumbCacheForTest(): void {
  cache.clear()
}
