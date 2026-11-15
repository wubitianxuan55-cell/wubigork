// verifyArtifacts.ts — Verifier 通道 B 产物目录（.gaea/work/journal/verify/<id>/）
// → 逐页缩略图视图模型的纯函数层（老账：行内逐页缩略预览）。真实目录布局
// （internal/app/gaea_verify.go runVisualDiff + internal/office/docmd/render.go
// RenderPDFPages）：
//
//   <工作区>/.gaea/work/journal/verify/<id>/
//     before.pdf / after.pdf            ← soffice 转换产物（非页面图，须过滤）
//     before/before-1.png …             ← pdftoppm 逐页 PNG（<prefix>-<N>.png）
//     after/after-1.png …               ← 同上；≥10 页时 pdftoppm 可能补零
//
// 本文件只做字符串/数组变换，不发任何桥接调用——目录列举与 Preview 取图在
// 组件层（components/verify/VerifyArtifactsThumbs.tsx），便于单测与降级判定。

// VerifyDirEntry 结构对齐后端 GaeaListDir 的 DirEntry（name/isDir/size），只取
// 缩略图所需字段，避免纯函数层依赖生成类型。
export interface VerifyDirEntry {
  name: string;
  isDir: boolean;
  size?: number;
}

// VerifyPageFile 是一张已解析页码的页面图（relPath 为工作区相对路径，直接可
// 传给 GaeaListDir/GaeaPreview —— 后者对相对路径 Join(工作区根) 后读取；
// GaeaListDir 亦已支持绝对路径，见 gaea_listdir.go IsAbs 分支）。
export interface VerifyPageFile {
  page: number;
  name: string;
  relPath: string;
}

// VerifyPagePair 是 before/after 同页码的一对（页数变化时单侧缺页为 undefined，
// 组件以「无此页」诚实占位，绝不伪造页面图）。
export interface VerifyPagePair {
  page: number;
  before?: VerifyPageFile;
  after?: VerifyPageFile;
}

// VerifyThumbCell / Row / Section 是缩略网格的渲染模型：before/after 齐备时
// 折成一个 pair 区（逐页成对），其余分组（或根目录平铺页面图）各成独立区。
export interface VerifyThumbCell {
  key: string;
  relPath: string;
  side: "before" | "after" | "single";
  page: number;
}
export interface VerifyThumbRow {
  key: string;
  page: number;
  cells: VerifyThumbCell[];
}
export interface VerifyThumbSection {
  key: string;
  rows: VerifyThumbRow[];
}

// VerifyArtifactsFault 是「诚实降级」的原因分类（组件映射到 i18n 文案）：
//   unreachable — channelBArtifacts 无法相对化（不含固定标记且是绝对路径，
//                 无法定位产物目录；只能外部打开，见汇报能力缺口）
//   missing     — 目录不存在（GAEADIR_NOT_FOUND/NOT_DIR 码，或旧后端探测
//                 error「文件不存在」文案兜底）
//   empty       — 目录在但列不出页面（探测 error「目录无法预览」= 存在而空）
//   noPages     — 列到了条目但没有可识别的逐页 PNG（如仅有 PDF：渲染降级）
export type VerifyArtifactsFault = "unreachable" | "missing" | "empty" | "noPages";

// DIR_ERROR_CODES 是后端 GaeaListDir 的结构化错误码（internal/app/gaea_listdir.go，
// 形态 `Error [CODE]: message`，对齐 internal/gaea/tool/builtin/errcode.go 口径）。
export const DIR_ERROR_CODES = {
  notFound: "GAEADIR_NOT_FOUND", // 目录不存在
  notDir: "GAEADIR_NOT_DIR", // 路径存在但不是目录（是文件）
  readFailed: "GAEADIR_READ_FAILED", // 读取目录失败（权限等 OS 错误透传）
} as const;

// parseErrorCode 从后端错误串提取结构化错误码（`Error [CODE]` 片段）；入参
// 兼容 string 与 Error（Wails 拒绝值是错误串原文）。无码（旧后端自然语言
// 文案 / 非错误负载）返回 null，调用方走文案匹配兜底。
export function parseErrorCode(message: unknown): string | null {
  const text =
    typeof message === "string" ? message : message instanceof Error ? message.message : "";
  const hit = /Error \[([A-Z0-9_]+)\]/.exec(text);
  return hit ? hit[1] : null;
}

// faultFromDirCode 把 GaeaListDir 的 GAEADIR_* 码映射为诚实降级态：
//   NOT_FOUND / NOT_DIR — 预期的产物目录不存在（含「是文件不是目录」）→ missing
//   READ_FAILED（权限等）— 四态无对应文案（i18n 冻结零新键），返回 null 走
//                         原 Preview 探测路径，与旧后端同口径落 empty
//   无码 / 无关码        — null（调用方走兜底）
export function faultFromDirCode(code: string | null): VerifyArtifactsFault | null {
  if (code === DIR_ERROR_CODES.notFound || code === DIR_ERROR_CODES.notDir) return "missing";
  return null;
}

// verifyArtifactRelPath 把 verdict 里的产物目录（Go filepath.ToSlash 绝对路径；
// mock/旧数据可能是已相对路径）解析为工作区相对路径。目录恒在固定标记
// 「.gaea/work/journal/verify/」之下（gaea_verify.go 用 Join(gaeaCwd(), …) 构造，
// <id> 不含分隔符，标记至多出现一次），按标记截取即与工作区根无关。
// 无法定位时返回 null（调用方按 unreachable 降级，不臆测路径）。
export function verifyArtifactRelPath(artifacts: string): string | null {
  const p = (artifacts ?? "").trim().replace(/\\/g, "/");
  if (!p) return null;
  const marker = ".gaea/work/journal/verify/";
  const idx = p.lastIndexOf(marker);
  if (idx >= 0) {
    return p.slice(idx).replace(/\/+$/, "") || null;
  }
  // 已是相对路径（mock / 旧数据）：原样可用
  if (!/^[a-za-z]:\//i.test(p) && !p.startsWith("/")) {
    return p.replace(/\/+$/, "") || null;
  }
  return null;
}

// 页面图扩展名：pdftoppm 只产 PNG，这里放宽到常见位图（与后端 imageExts 对齐，
// 排除 svg/ico——渲染页不会是矢量/图标），.pdf/.jsonl 等审计文件据此过滤。
const PAGE_IMAGE_RE = /\.(png|jpe?g|gif|webp|bmp)$/i;

// imagePageNo 从文件名解析页码，规则对齐 RenderPDFPages 的读取侧（render.go）：
// 取最后一个「-」之后的数字段——「before-1.png」→1、「after-10.png」→10、
// 补零「before-03.png」→3；非图扩展名 / 无数字段 → null（过滤）。
export function imagePageNo(name: string): number | null {
  if (!PAGE_IMAGE_RE.test(name)) return null;
  const dot = name.lastIndexOf(".");
  const stem = dot > 0 ? name.slice(0, dot) : name;
  const idx = stem.lastIndexOf("-");
  if (idx <= 0 || idx === stem.length - 1) return null;
  const n = Number(stem.slice(idx + 1));
  return Number.isInteger(n) && n > 0 ? n : null;
}

// splitArtifactEntries 把一层目录条目拆成「子目录名 + 已排序页面图」。
// parentRel 是本层的工作区相对路径（用于拼 relPath，统一 / 分隔）。
// 非图文件（before.pdf 等）与解析不出页码的图片一律过滤；页码相同按名稳定序。
export function splitArtifactEntries(
  entries: readonly VerifyDirEntry[] | undefined,
  parentRel: string,
): { dirs: string[]; pages: VerifyPageFile[] } {
  const dirs: string[] = [];
  const pages: VerifyPageFile[] = [];
  const base = (parentRel ?? "").replace(/\/+$/, "");
  for (const e of entries ?? []) {
    if (!e || typeof e.name !== "string" || e.name === "") continue;
    if (e.isDir) {
      dirs.push(e.name);
      continue;
    }
    const page = imagePageNo(e.name);
    if (page === null) continue;
    pages.push({ page, name: e.name, relPath: base ? `${base}/${e.name}` : e.name });
  }
  pages.sort((a, b) => a.page - b.page || (a.name < b.name ? -1 : a.name > b.name ? 1 : 0));
  return { dirs, pages };
}

// orderGroupDirs 给子目录定呈现序：before → after → 其余按名（真实布局恒为
// before/after；顺序对齐 verdict 文案「N→M 页」的改前改后语义）。
const GROUP_ORDER = ["before", "after"];
export function orderGroupDirs(dirs: readonly string[]): string[] {
  const rank = (d: string): number => {
    const i = GROUP_ORDER.indexOf(d.toLowerCase());
    return i < 0 ? GROUP_ORDER.length : i;
  };
  return [...dirs].sort((a, b) => rank(a) - rank(b) || a.localeCompare(b));
}

// pairPages 把 before/after 两套页面按页码并集配对（升序）。单侧缺页保留
// undefined，由组件渲染「无此页」占位——诚实呈现页数变化，不伪造页面。
export function pairPages(
  before: readonly VerifyPageFile[],
  after: readonly VerifyPageFile[],
): VerifyPagePair[] {
  const map = new Map<number, VerifyPagePair>();
  for (const f of before) {
    const e = map.get(f.page) ?? { page: f.page };
    e.before = f;
    map.set(f.page, e);
  }
  for (const f of after) {
    const e = map.get(f.page) ?? { page: f.page };
    e.after = f;
    map.set(f.page, e);
  }
  return [...map.values()].sort((a, b) => a.page - b.page);
}

// buildThumbSections 把各分组的页面列表组装成渲染区：before/after 至少一侧
// 存在时折成一个 pair 区（逐页成对，side 标注 before/after）；其余分组（含
// 根目录平铺页面图的 key="flat"）各自成区、单列呈现。
export function buildThumbSections(
  groups: ReadonlyArray<{ key: string; pages: readonly VerifyPageFile[] }>,
): VerifyThumbSection[] {
  const byKey = new Map(groups.map((g) => [g.key, g]));
  const before = byKey.get("before");
  const after = byKey.get("after");
  const sections: VerifyThumbSection[] = [];
  if (before || after) {
    sections.push({
      key: "pair",
      rows: pairPages(before?.pages ?? [], after?.pages ?? []).map((p) => ({
        key: `pair-p${p.page}`,
        page: p.page,
        cells: [
          ...(p.before
            ? [{ key: `pair-b${p.page}`, relPath: p.before.relPath, side: "before" as const, page: p.page }]
            : []),
          ...(p.after
            ? [{ key: `pair-a${p.page}`, relPath: p.after.relPath, side: "after" as const, page: p.page }]
            : []),
        ],
      })),
    });
  }
  for (const g of groups) {
    if (g.key === "before" || g.key === "after") continue;
    if (g.pages.length === 0) continue; // 空组不成区（半空目录不渲染空壳）
    sections.push({
      key: g.key,
      rows: g.pages.map((f) => ({
        key: `${g.key}-p${f.page}`,
        page: f.page,
        cells: [{ key: `${g.key}-s${f.page}`, relPath: f.relPath, side: "single" as const, page: f.page }],
      })),
    });
  }
  return sections.filter((s) => s.rows.length > 0);
}

// classifyDirProbe 把「目录列不出/列空」时的降级原因定性——结构化错误码优先：
// 错误串带 `Error [GAEADIR_*]` 码时按码路由（新后端，不解析散文）；无码退回
// 旧后端错误文案匹配兜底。后端语义（gaea_preview.go / gaea_listdir.go）：目录
// 不存在 → kind=error「文件不存在」/码 GAEADIR_NOT_FOUND；目录存在 →
// kind=error「目录无法预览」（此时列为空 = 目录为空）；非 error（mock 或异常
// 负载）→ 目录可访问只是没有可识别页面。
export function classifyDirProbe(kind: string, error: string): VerifyArtifactsFault {
  if (kind !== "error") return "noPages";
  const byCode = faultFromDirCode(parseErrorCode(error));
  if (byCode) return byCode;
  return (error ?? "").includes("不存在") ? "missing" : "empty";
}

// mapPool 定并发映射（Promise 池）：至多 limit 个 fn 在飞，保持输入顺序返回。
// 缩略图逐页 Preview 取 dataUrl 用（展开才拉 + 并发≤4，加载失败互不拖垮）。
export async function mapPool<T, R>(
  items: readonly T[],
  limit: number,
  fn: (item: T, index: number) => Promise<R>,
): Promise<R[]> {
  const results = new Array<R>(items.length);
  let next = 0;
  const width = Math.max(1, Math.min(limit, items.length));
  const workers = Array.from({ length: width }, async () => {
    for (;;) {
      const i = next++;
      if (i >= items.length) break;
      results[i] = await fn(items[i], i);
    }
  });
  await Promise.all(workers);
  return results;
}
