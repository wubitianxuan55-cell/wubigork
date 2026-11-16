import type { Item } from "./store";

export interface SessionChange {
  path: string;
  count: number;
  lastTouched?: number;
}

// 会修改工作区文件的工具（用于「变更」面板汇总）。
// v4.519：摘除从未注册的幽灵名（notebook_edit/delete_range/delete_symbol，
// 与 Go 侧 evidence.isWriterTool 同批——名单只含工具注册表真实存在的名字）。
export const WRITE_TOOL_NAMES = new Set([
  "write_file", "edit_file", "edit_lines", "multi_edit",
  "move_file", "replace",
]);

function pushPath(out: string[], v: unknown): void {
  if (typeof v === "string" && v.trim() !== "") out.push(v.trim());
}

// deliverableDedupKey — 交付路径去重键归一化（2026-11-16「4 个相同交互文件」
// 根修）：同一文件在会话里有多种写法（绝对/相对、正/反斜杠、./ 前缀、大小
// 写、正文提及 vs 工具参数），此前三层去重全按原始字符串比对，把它们记成多
// 条面板条目。键恒为归一化绝对形态（小写、正斜杠、相对路径按工作区解析）；
// 展示层保留首次出现的原始写法，不做改写。Windows 文件系统大小写不敏感，
// 本板块是 Windows 桌面端。
export function deliverableDedupKey(path: string, cwd?: string): string {
  let s = path.trim().replace(/\\/g, "/");
  while (s.startsWith("./")) s = s.slice(2);
  s = s.toLowerCase();
  if (!cwd) return s;
  const root = cwd.trim().replace(/\\/g, "/").replace(/\/+$/, "").toLowerCase();
  if (s === root) return "";
  if (root !== "" && s.startsWith(root + "/")) return s; // 工作区内绝对路径
  if (/^[a-z]:\//i.test(s) || s.startsWith("/")) return s; // 工作区外绝对路径
  return root + "/" + s; // 相对路径按工作区解析
}

// 从写类工具的参数里提取被改动的工作区文件路径，与后端
// internal/gaea/evidence/evidence.go 的 extractPaths 及
// internal/gaea/agent/agent_helpers.go 的 extractFilePath 对齐。
export function extractChangedPaths(args: string, _tool?: string): string[] {
  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(args || "{}") as Record<string, unknown>;
  } catch {
    return [];
  }
  const out: string[] = [];
  for (const key of ["path", "file_path", "notebook_path", "source", "destination"]) {
    pushPath(out, parsed[key]);
  }
  for (const key of ["paths", "file_paths"]) {
    if (Array.isArray(parsed[key])) (parsed[key] as unknown[]).forEach((v) => pushPath(out, v));
  }
  // multi_edit / edit_file 可能把多个编辑片段放在 edits: [{path, ...}]。
  if (Array.isArray(parsed.edits)) {
    for (const edit of parsed.edits as unknown[]) {
      if (edit && typeof edit === "object") {
        const e = edit as Record<string, unknown>;
        pushPath(out, e.path);
        pushPath(out, e.file_path);
      }
    }
  }
  return out;
}

// 从会话消息里汇总“写/改过的工作区文件及次数”，按最近改动倒序返回。
// cwd 传入时按归一化键去重（同一文件多种写法合并且计数累加，2026-11-16）；
// 展示 path 保留首次出现的原始写法。
export function buildSessionChanges(
  items: Item[],
  writeTools: ReadonlySet<string> = WRITE_TOOL_NAMES,
  cwd?: string,
): SessionChange[] {
  const map = new Map<string, { path: string; count: number; lastTouched: number }>();
  items.forEach((it, idx) => {
    if (it.kind !== "tool" || !writeTools.has(it.name)) return;
    for (const p of extractChangedPaths(it.args || "", it.name)) {
      const key = deliverableDedupKey(p, cwd);
      const cur = map.get(key);
      if (cur) {
        cur.count++;
        cur.lastTouched = idx;
      } else {
        map.set(key, { path: p, count: 1, lastTouched: idx });
      }
    }
  });
  return [...map.entries()]
    .map(([, v]) => ({ path: v.path, count: v.count, lastTouched: v.lastTouched }))
    .sort((a, b) => b.lastTouched - a.lastTouched);
}

// 从写类工具参数提取「交付物」路径（成果面板显式登记用）：与
// extractChangedPaths 的差别是 move_file 只计 destination——源路径
// 不是交付物；去重保持出现顺序。
export function extractDeliverablePaths(args: string): string[] {
  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(args || "{}") as Record<string, unknown>;
  } catch {
    return [];
  }
  const out: string[] = [];
  const seen = new Set<string>();
  const push = (v: unknown) => {
    if (typeof v === "string" && v.trim() !== "") {
      const p = v.trim();
      if (!seen.has(p)) {
        seen.add(p);
        out.push(p);
      }
    }
  };
  for (const key of ["path", "file_path", "notebook_path", "destination"]) {
    push(parsed[key]);
  }
  for (const key of ["paths", "file_paths"]) {
    if (Array.isArray(parsed[key])) (parsed[key] as unknown[]).forEach(push);
  }
  if (Array.isArray(parsed.edits)) {
    for (const edit of parsed.edits as unknown[]) {
      if (edit && typeof edit === "object") {
        const e = edit as Record<string, unknown>;
        push(e.path);
        push(e.file_path);
      }
    }
  }
  return out;
}

// ── 三态追踪（2a 本轮文件折叠补全）────────────────────────────────
// 源 better-sidebar「统一文件变动」语义：读/写/编辑三态独立成层，同一文件
// 可同时出现在多层（读过后又被写 = 两层各有一条）。写入/编辑沿用
// WRITE_TOOL_NAMES 既有拆分，读取对齐后端 contextview.fileActionByTool 的
// read 白名单（read_file/grep/vision/format_convert；ls 列目录无文件语义不入层）。

/** 「编辑」= 修改既有内容：行级 diff 可展开（edit 族）。 */
export const EDIT_TOOL_NAMES = new Set([
  "edit_file", "edit_lines", "multi_edit",
]);

/** 「写入」= 新建/整写/移动/生成：写入内容预览降级态（write_file 等）。
 *  replace（D4）= 跨文件字面量批量替换，无行级锚点，同归写入层。 */
export const WRITE_ONLY_TOOL_NAMES = new Set(["write_file", "move_file", "replace"]);

/** 「读取」= 模型读过的输入文件（与后端 fileActionByTool read 白名单对齐）。 */
export const READ_TOOL_NAMES = new Set(["read_file", "grep", "vision", "format_convert"]);

// 读取工具的路径键候选：source（format_convert）、image_path（vision）。
function extractReadPathsImpl(args: string): string[] {
  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(args || "{}") as Record<string, unknown>;
  } catch {
    return [];
  }
  const out: string[] = [];
  for (const key of ["path", "file_path", "source", "image_path"]) {
    pushPath(out, parsed[key]);
  }
  for (const key of ["paths", "file_paths"]) {
    if (Array.isArray(parsed[key])) (parsed[key] as unknown[]).forEach((v) => pushPath(out, v));
  }
  return out;
}

// 从会话消息里汇总「模型读取过的文件及次数」，按最近读取倒序。
export function buildSessionReads(
  items: Item[],
  readTools: ReadonlySet<string> = READ_TOOL_NAMES,
): SessionChange[] {
  const map = new Map<string, { count: number; lastTouched: number }>();
  items.forEach((it, idx) => {
    if (it.kind !== "tool" || !readTools.has(it.name)) return;
    for (const p of extractReadPathsImpl(it.args || "")) {
      const cur = map.get(p) ?? { count: 0, lastTouched: idx };
      map.set(p, { count: cur.count + 1, lastTouched: idx });
    }
  });
  return [...map.entries()]
    .map(([path, v]) => ({ path, count: v.count, lastTouched: v.lastTouched }))
    .sort((a, b) => b.lastTouched - a.lastTouched);
}

// ── 类型筛选（按扩展名分桶；与 FileThumb/FileTypeIcon 的图片族口径一致）──

export type FileCat = "all" | "doc" | "sheet" | "image" | "code" | "other";

const CAT_RE: Record<Exclude<FileCat, "all" | "other">, RegExp> = {
  doc: /\.(md|markdown|txt|docx?|pdf|rtf|ofd|wps|html?|mdx)$/i,
  sheet: /\.(xlsx?|csv|et)$/i,
  image: /\.(png|jpe?g|gif|webp|bmp|svg|ico)$/i,
  code: /\.(go|tsx?|jsx?|py|java|c|h|cpp|hpp|rs|rb|php|sh|bat|ps1|sql|json|ya?ml|toml|xml|css)$/i,
};

export function categoryOf(path: string): Exclude<FileCat, "all"> {
  for (const [cat, re] of Object.entries(CAT_RE) as [Exclude<FileCat, "all" | "other">, RegExp][]) {
    if (re.test(path)) return cat;
  }
  return "other";
}
