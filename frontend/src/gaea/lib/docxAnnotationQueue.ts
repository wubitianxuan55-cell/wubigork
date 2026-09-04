// docxAnnotationQueue.ts — Word 预览「修改队列」纯逻辑（状态机 + 摘录定位 +
// 批量执行编排），零 DOM 依赖、零后端依赖，可独立单测。
//
// Why: DocxPreview 已有「框选即改」（单处选区 → 指令 → AI 替换 → Word 修订），
// 但用户往往要在长文档里改**多处**——每处都要走一遍「框选 → 生成 → 应用」。
// 本模块把修改意图攒成队列，再串行批量执行（对标 Genspark Send N edits 的
// 「多次圈画 → 一次提交」，docs/market-research-2026-09-03c.md A2）。
//
// How: 三块纯逻辑——
//   1) 队列状态机：条目 {id, 摘录, 指令, 状态, 错误}，增删改查/清空/去重
//      （同摘录同指令合并），状态迁移白名单（done 终态不可重跑）；
//   2) 摘录归一化与再定位：normalizeExcerpt 空白收敛 + locateExcerpt 在
//      当前全文中找摘录。多处命中取第一处但 unique=false 如实上报，由调用
//      方决策；找不到返回 null——绝不猜位；
//   3) 批量执行编排 runQueue：串行逐条，每条执行前对**最新文档全文**再定位
//      （前一条替换后文本位移以真实写回结果为准），失败/跳过不中断后续条目。
// 装配层（DocxPreview）负责把 deps 映射到 app.OfficeEditText /
// app.DocxApplyEdit / app.DocxAcceptChanges 通道与 React 状态同步。

// ── 队列条目与状态机 ─────────────────────────────────────────────

export type DocxQueueItemStatus = "pending" | "running" | "done" | "failed" | "skipped";

export interface DocxQueueItem {
  id: string;
  /** 摘录原文（用户框选的文本，保留原样——AI 生成与后端匹配都用它） */
  excerpt: string;
  /** 归一化摘录（去重与定位匹配用；空白收敛） */
  norm: string;
  /** 指令（预设动作的指令文本或用户自定义输入） */
  instruction: string;
  status: DocxQueueItemStatus;
  /** 失败/跳过原因。可能是装配层可读的原始错误，也可能是本模块的哨兵值 */
  error?: string;
  /** 非唯一命中：摘录在文档中出现多处，按第一处执行（如实暴露给 UI，不静默） */
  ambiguous?: boolean;
}

// 哨兵错误值：装配层据此渲染本地化文案，其余 error 为原始错误消息直出。
export const SKIP_NOT_LOCATED = "excerpt-not-located";
export const FAIL_EMPTY_REPLACEMENT = "empty-replacement";

let idSeq = 0;

/** 生成队列条目 id（模块内单调递增，进程内唯一即可）。 */
export function nextQueueItemId(): string {
  idSeq += 1;
  return `dq-${idSeq}`;
}

/** 摘录归一化：所有空白字符序列（含换行/制表/全角空格）收敛为单个空格并去首尾。 */
export function normalizeExcerpt(s: string): string {
  return s.replace(/\s+/g, " ").trim();
}

export function createQueueItem(
  excerpt: string,
  instruction: string,
  id: string = nextQueueItemId(),
): DocxQueueItem {
  return {
    id,
    excerpt,
    norm: normalizeExcerpt(excerpt),
    instruction,
    status: "pending",
  };
}

export interface QueueAddResult {
  /** 入队后的新队列（原数组不变） */
  queue: DocxQueueItem[];
  /** 新增的条目；合并时为被并入的既有条目 */
  item: DocxQueueItem;
  /** true = 摘录与指令均与既有条目重复，已并入（未新增） */
  merged: boolean;
}

/** 入队（去重：归一化后同摘录同指令合并进既有条目，不产生重复条目）。 */
export function addToQueue(
  queue: readonly DocxQueueItem[],
  excerpt: string,
  instruction: string,
): QueueAddResult {
  const norm = normalizeExcerpt(excerpt);
  const normInstr = normalizeExcerpt(instruction);
  if (!norm || !normInstr) throw new Error("摘录与指令均不能为空");
  const existing = queue.find(
    (q) => q.norm === norm && normalizeExcerpt(q.instruction) === normInstr,
  );
  if (existing) return { queue: [...queue], item: existing, merged: true };
  const item = createQueueItem(excerpt, instruction);
  return { queue: [...queue, item], item, merged: false };
}

/** 移除单条（执行中由装配层负责禁用，这里只做纯删除）。 */
export function removeFromQueue(queue: readonly DocxQueueItem[], id: string): DocxQueueItem[] {
  return queue.filter((q) => q.id !== id);
}

/** 清空队列。 */
export function clearDocxQueue(): DocxQueueItem[] {
  return [];
}

export type QueueItemPatch = Partial<Omit<DocxQueueItem, "id" | "norm">>;

/** 更新单条字段（不可变；patch 显式置 undefined 的键会被清除）。 */
export function updateQueueItem(
  queue: readonly DocxQueueItem[],
  id: string,
  patch: QueueItemPatch,
): DocxQueueItem[] {
  return queue.map((q) => (q.id === id ? { ...q, ...patch } : q));
}

// 状态迁移白名单：pending 出发执行；running 落到三个终态之一；failed/skipped
// 可重试（回 pending 或直接再出发）；done 终态——重跑请重新入队。
const NEXT_STATUS: Record<DocxQueueItemStatus, readonly DocxQueueItemStatus[]> = {
  pending: ["running", "skipped"],
  running: ["done", "failed", "skipped"],
  failed: ["pending", "running"],
  skipped: ["pending", "running"],
  done: [],
};

/** 状态迁移（白名单校验，非法迁移抛错——宁抛不静默）。 */
export function transitionStatus(item: DocxQueueItem, next: DocxQueueItemStatus): DocxQueueItem {
  if (!NEXT_STATUS[item.status].includes(next)) {
    throw new Error(`非法的队列状态迁移：${item.status} → ${next}`);
  }
  return { ...item, status: next };
}

/** 失败/跳过条目重置为待执行（清错误与歧义标记）；running/done 不可重试，抛错。 */
export function resetItemForRetry(queue: readonly DocxQueueItem[], id: string): DocxQueueItem[] {
  return queue.map((q) => {
    if (q.id !== id) return q;
    if (q.status !== "failed" && q.status !== "skipped") {
      throw new Error(`该条目当前状态为 ${q.status}，不可重试`);
    }
    return { ...q, status: "pending" as const, error: undefined, ambiguous: false };
  });
}

/** 待执行条目（执行全部的实际范围：pending + 可重试的 failed/skipped）。 */
export function runnableItems(queue: readonly DocxQueueItem[]): DocxQueueItem[] {
  return queue.filter((q) => q.status === "pending" || q.status === "failed" || q.status === "skipped");
}

export interface QueueStats {
  total: number;
  pending: number;
  running: number;
  done: number;
  failed: number;
  skipped: number;
}

/** 各状态计数（面板徽标/进度条用）。 */
export function queueStats(queue: readonly DocxQueueItem[]): QueueStats {
  const stats: QueueStats = { total: queue.length, pending: 0, running: 0, done: 0, failed: 0, skipped: 0 };
  for (const q of queue) stats[q.status] += 1;
  return stats;
}

// ── 摘录定位 ─────────────────────────────────────────────────────

export interface ExcerptLocation {
  /** 命中区间在 fullText 中的坐标，[start, end)（原文 UTF-16 码元坐标） */
  start: number;
  end: number;
  /** false = 全文出现多处，当前取第一处（必须如实暴露给调用方决策） */
  unique: boolean;
  /** 命中总次数（有上限，见 MAX_OCCURRENCE_SCAN） */
  occurrences: number;
}

// 出现次数统计上限：超出即停止计数（unique 必为 false，只防病态长文档的
// 全文多次扫描开销，不影响判定正确性）。
const MAX_OCCURRENCE_SCAN = 1000;

interface NormalizedWithMap {
  text: string;
  /** map[i] = 归一化文本第 i 个字符在原字符串中的下标 */
  map: number[];
}

// 空白收敛但保留到原文的坐标映射：连续空白折叠为单个空格，该空格锚定在
// 空白序列的起点；随后去首尾空格（连同映射）。
function normalizeWithMap(s: string): NormalizedWithMap {
  const chars: string[] = [];
  const map: number[] = [];
  let i = 0;
  while (i < s.length) {
    if (/\s/.test(s[i])) {
      chars.push(" ");
      map.push(i);
      while (i < s.length && /\s/.test(s[i])) i += 1;
    } else {
      chars.push(s[i]);
      map.push(i);
      i += 1;
    }
  }
  let start = 0;
  let end = chars.length;
  while (start < end && chars[start] === " ") start += 1;
  while (end > start && chars[end - 1] === " ") end -= 1;
  return { text: chars.slice(start, end).join(""), map: map.slice(start, end) };
}

/**
 * 在全文中定位摘录（归一化后子串匹配，返回原文坐标）。
 * - 唯一命中：unique=true；
 * - 多处命中：取第一处但 unique=false 如实上报（由调用方决策，绝不静默）；
 * - 找不到（含摘录为空/全文为空）：返回 null——不定位，绝不猜位。
 */
export function locateExcerpt(fullText: string, excerpt: string): ExcerptLocation | null {
  const hay = normalizeWithMap(fullText);
  const needle = normalizeExcerpt(excerpt);
  if (!hay.text || !needle) return null;
  const positions: number[] = [];
  let from = 0;
  while (from <= hay.text.length - needle.length) {
    const at = hay.text.indexOf(needle, from);
    if (at < 0) break;
    positions.push(at);
    if (positions.length >= MAX_OCCURRENCE_SCAN) break;
    from = at + 1;
  }
  if (positions.length === 0) return null;
  const start = positions[0];
  return {
    start: hay.map[start],
    end: hay.map[start + needle.length - 1] + 1,
    unique: positions.length === 1,
    occurrences: positions.length,
  };
}

/** 把 [start, end) 区间替换为 replacement（纯字符串拼接，编排测试模拟位移用）。 */
export function applyReplacementToText(fullText: string, loc: ExcerptLocation, replacement: string): string {
  return fullText.slice(0, loc.start) + replacement + fullText.slice(loc.end);
}

// ── 批量执行编排 ─────────────────────────────────────────────────

export interface QueueRunDeps {
  /** 生成替换文（装配层映射 app.OfficeEditText）；返回空串按失败计 */
  generate(excerpt: string, instruction: string): Promise<string>;
  /** 写回文档（装配层映射 app.DocxApplyEdit + DocxAcceptChanges accept=true） */
  apply(excerpt: string, replacement: string): Promise<void>;
  /** 读当前文档全文（每条执行前再定位；写回成功后装配层必须反映最新内容） */
  readText(): Promise<string>;
  /** 单条状态变更回调（装配层同步 React 状态；串行保证逐条到达） */
  onUpdate(id: string, patch: QueueItemPatch): void;
}

export interface QueueRunSummary {
  /** 本轮实际执行的条数（pending + 重试的 failed/skipped） */
  total: number;
  done: number;
  failed: number;
  skipped: number;
}

/**
 * 串行执行队列：逐条「再定位 → 生成 → 写回」。
 * - 每条执行前对 deps.readText() 给出的**最新全文**再定位（前一条替换后
 *   文本位移以真实写回结果为准，不做本地模拟漂移）；
 * - 定位不到 → 该条 skipped（哨兵 SKIP_NOT_LOCATED，绝不错位替换）；
 * - 生成/写回失败 → 该条 failed（原始错误信息），继续下一条；
 * - 非唯一命中 → 照常执行第一处，ambiguous=true 如实暴露给 UI。
 */
export async function runQueue(
  queue: readonly DocxQueueItem[],
  deps: QueueRunDeps,
): Promise<QueueRunSummary> {
  const items = runnableItems(queue);
  const summary: QueueRunSummary = { total: items.length, done: 0, failed: 0, skipped: 0 };
  let text: string | null = null;
  let textLoaded = false;
  for (const item of items) {
    deps.onUpdate(item.id, { status: "running", error: undefined, ambiguous: false });
    if (!textLoaded) {
      try {
        text = await deps.readText();
        textLoaded = true;
      } catch (e) {
        // 全文读不出来就无法定位——按失败计（不是 skip：是环境/文档层故障），
        // 下一条会重试 readText。
        deps.onUpdate(item.id, { status: "failed", error: errorMessage(e), ambiguous: false });
        summary.failed += 1;
        continue;
      }
    }
    const loc = locateExcerpt(text ?? "", item.excerpt);
    if (!loc) {
      deps.onUpdate(item.id, { status: "skipped", error: SKIP_NOT_LOCATED, ambiguous: false });
      summary.skipped += 1;
      continue;
    }
    try {
      const replacement = await deps.generate(item.excerpt, item.instruction);
      if (!replacement || !replacement.trim()) {
        deps.onUpdate(item.id, { status: "failed", error: FAIL_EMPTY_REPLACEMENT, ambiguous: !loc.unique });
        summary.failed += 1;
        continue;
      }
      await deps.apply(item.excerpt, replacement);
      // 写回成功：下一条必须用刷新后的全文再定位（位移以真实文档为准）。
      textLoaded = false;
      deps.onUpdate(item.id, { status: "done", error: undefined, ambiguous: !loc.unique });
      summary.done += 1;
    } catch (e) {
      deps.onUpdate(item.id, { status: "failed", error: errorMessage(e), ambiguous: !loc.unique });
      summary.failed += 1;
    }
  }
  return summary;
}

function errorMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
