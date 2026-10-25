// cadToolCache.ts — CAD 解析结果缓存（设计 docs/gaea-ai-workbench-ui-2026-10.md
// §7.3：ToolSheet 关闭即卸载 CadPanel，解析产物缓存在此按 path 键，重开不重跑
// CadOpen/CadRaster/CadTexts/CadCount——「临时面板，结果留产物」的体验兑现）。
//
// 与 cadRecent.ts 同款模块级单例（无 UI 订阅需求，命中时组件直接读）；图纸只读
// 红线（三绑定零写入）→ 文件不解析失效问题，缓存随会话存活、应用重启即清。
// 内存节制：raster dataUrl 很大，仅缓存默认宽度（width=0）渲图——「问图纸」的
// ASK_WIDTH 加宽图不缓存；每文件最多 MAX_RASTERS 张（LRU），最多 MAX_FILES 个
// 文件（LRU，touch 即续期）。
import type { CadOpenResult, CadRasterResult, CadTextsResult, CadCountResult } from "../../lib/bridge/cad";

export interface CadCacheEntry {
  openRes: CadOpenResult;
  /** 打开时自动选中的质量最佳图框（1 基）。 */
  bestSheet: number;
  rasters: Array<{ sheet: number; res: CadRasterResult }>;
  texts: CadTextsResult | null;
  count: CadCountResult | null;
}

const MAX_FILES = 8;
const MAX_RASTERS = 3;

let entries: Map<string, CadCacheEntry> = new Map();

function touch(path: string, entry: CadCacheEntry): void {
  entries.delete(path);
  entries.set(path, entry);
}

function evict(): void {
  while (entries.size > MAX_FILES) {
    const oldest = entries.keys().next().value;
    if (oldest === undefined) break;
    entries.delete(oldest);
  }
}

export function cadCacheGet(path: string): CadCacheEntry | null {
  const hit = entries.get(path);
  if (!hit) return null;
  touch(path, hit);
  return hit;
}

export function cadCachePutOpen(path: string, openRes: CadOpenResult, bestSheet: number): void {
  entries.set(path, { openRes, bestSheet, rasters: [], texts: null, count: null });
  evict();
}

export function cadCachePutRaster(path: string, sheet: number, res: CadRasterResult): void {
  const e = entries.get(path);
  if (!e) return;
  const rest = e.rasters.filter((r) => r.sheet !== sheet);
  rest.push({ sheet, res });
  e.rasters = rest.slice(-MAX_RASTERS);
  touch(path, e);
}

export function cadCachePutTexts(path: string, texts: CadTextsResult): void {
  const e = entries.get(path);
  if (!e) return;
  e.texts = texts;
  touch(path, e);
}

export function cadCachePutCount(path: string, count: CadCountResult): void {
  const e = entries.get(path);
  if (!e) return;
  e.count = count;
  touch(path, e);
}

/** 测试辅助：清空（vitest 用例间隔离；模块级状态跨用例存活）。 */
export function cadCacheResetForTest(): void {
  entries = new Map();
}
