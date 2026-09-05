// GenUI 围栏解析：markdown 文本拆分 + 完整/部分规格解析。
//
// 流式部分解析策略与上游一致：完整 parse 优先；失败后对括号平衡前缀做
// 有界候选收集（预算 32），候选按最长优先试 parse——结果永远是完整组件的
// 前缀，不会半渲染。

import { GENUI_FENCE_LANGS, GENUI_LIMITS, type GenuiSpec } from "./spec";
import { repairGenuiSpec } from "./guard";

/** 文本段或围栏段。 */
export type GenuiSegment =
  | { kind: "text"; text: string }
  | { kind: "fence"; lang: string; body: string; closed: boolean; fenceNo: number };

export interface GenuiFenceSplit {
  segments: GenuiSegment[];
  /** 围栏序号 → 段索引（便于宿主快速定位）。 */
  fenceIndex: number[];
}

/**
 * 把 markdown 文本按 ```genui / ```dsh-ui 围栏切段。
 * 其他语言的代码块原样留在 text 段内（由 markdown 渲染器处理）。
 */
export function splitGenuiFences(text: string): GenuiFenceSplit {
  const lines = text.split("\n");
  const segments: GenuiSegment[] = [];
  const fenceIndex: number[] = [];
  let buf: string[] = [];
  let pending: { lang: string; fenceNo: number } | null = null;
  let fenceNo = 0;

  const flushText = (): void => {
    if (buf.length > 0) {
      segments.push({ kind: "text", text: buf.join("\n") });
      buf = [];
    }
  };

  for (const line of lines) {
    const open = /^```([\w-]+)\s*$/.exec(line);
    if (open && GENUI_FENCE_LANGS.has(open[1])) {
      flushText();
      pending = { lang: open[1], fenceNo };
      fenceIndex[fenceNo] = segments.length;
      segments.push({ kind: "fence", lang: open[1], body: "", closed: false, fenceNo });
      fenceNo += 1;
      continue;
    }
    if (pending !== null) {
      if (/^\s*```\s*$/.test(line)) {
        const seg = segments[segments.length - 1];
        if (seg.kind === "fence" && seg.fenceNo === pending.fenceNo) {
          segments[segments.length - 1] = {
            ...seg,
            body: seg.body.replace(/\n$/, ""),
            closed: true,
          };
        }
        pending = null;
        continue;
      }
      const seg = segments[segments.length - 1];
      if (seg.kind === "fence" && seg.fenceNo === pending.fenceNo) {
        segments[segments.length - 1] = { ...seg, body: `${seg.body}${line}\n` };
      }
      continue;
    }
    buf.push(line);
  }
  flushText();
  return { segments, fenceIndex };
}

/**
 * 修复性标点清理：去掉字符串外紧邻 ] / } 前的尾随逗号。
 * 只做这一种确定性小修；结构性错误一律不猜。
 */
export function stripTrailingCommas(json: string): string {
  let out = "";
  let inString = false;
  let escaped = false;
  for (let i = 0; i < json.length; i++) {
    const ch = json[i];
    const next = json[i + 1];
    if (inString) {
      out += ch;
      if (escaped) escaped = false;
      else if (ch === "\\") escaped = true;
      else if (ch === '"') inString = false;
      continue;
    }
    if (ch === '"') {
      inString = true;
      out += ch;
      continue;
    }
    if (ch === "," && (next === "]" || next === "}")) {
      continue;
    }
    out += ch;
  }
  return out;
}

/** 解析一段完整（已闭合）的围栏体。失败返回 null。 */
export function parseGenuiFenceBody(body: string): GenuiSpec | null {
  if (body.length > GENUI_LIMITS.maxFenceBody) return null;
  const text = body.trim();
  if (text === "") return null;
  const attempts = [text, stripTrailingCommas(text)];
  for (const candidate of attempts) {
    try {
      const parsed: unknown = JSON.parse(candidate);
      const spec = repairGenuiSpec(parsed);
      if (spec !== null) return spec;
    } catch {
      // 尝试下一个候选
    }
  }
  return null;
}

/** 部分解析候选预算（测试可覆写）。 */
export const MAX_PARTIAL_REPAIR_ATTEMPTS = 32;
let partialAttemptsLimit = MAX_PARTIAL_REPAIR_ATTEMPTS;

export function setMaxPartialRepairAttempts(n: number): void {
  partialAttemptsLimit = Math.max(1, n);
}

interface PartialCandidate {
  end: number;
  closingSuffix: string;
}

/**
 * 收集可试 parse 的括号平衡前缀（字符串/转义感知，单次左到右扫描）。
 * 直接照上游算法：只在“根 items 内组件闭合”或“整体平衡”处记候选。
 */
export function collectPartialCandidates(raw: string): { candidates: PartialCandidate[]; scannedChars: number } {
  const stack: string[] = [];
  const candidates: PartialCandidate[] = [];
  let inString = false;
  let escaped = false;
  let scanned = 0;
  const push = (c: PartialCandidate): void => {
    if (candidates.length >= partialAttemptsLimit) candidates.shift();
    candidates.push(c);
  };
  for (; scanned < raw.length; scanned++) {
    const ch = raw[scanned];
    if (inString) {
      if (escaped) escaped = false;
      else if (ch === "\\") escaped = true;
      else if (ch === '"') inString = false;
      continue;
    }
    if (ch === '"') {
      inString = true;
      continue;
    }
    if (ch === "{" || ch === "[") {
      stack.push(ch);
      continue;
    }
    if (ch === "}" || ch === "]") {
      const open = stack.pop();
      const expects = ch === "}" ? "{" : "[";
      if (open !== expects) break;
      if (ch === "}" && stack.length === 2 && stack[0] === "{" && stack[1] === "[") {
        push({ end: scanned + 1, closingSuffix: "]}" });
      }
      if (stack.length === 0) {
        push({ end: scanned + 1, closingSuffix: "" });
      }
      continue;
    }
  }
  candidates.sort((a, b) => b.end - a.end);
  const deduped: PartialCandidate[] = [];
  for (const c of candidates) {
    if (deduped.length === 0 || deduped[deduped.length - 1].end !== c.end) deduped.push(c);
  }
  return { candidates: deduped.slice(0, partialAttemptsLimit), scannedChars: scanned };
}

/** 解析可能仍在增长的围栏体：只返回已完整写入的组件前缀。 */
export function parsePartialGenuiSpec(body: string): GenuiSpec | null {
  const text = body.trim();
  if (text === "") return null;
  if (text.length > GENUI_LIMITS.maxFenceBody) return null;

  const full = parseGenuiFenceBody(text);
  if (full !== null) return full;

  const { candidates } = collectPartialCandidates(text);
  for (const candidate of candidates) {
    const candidateText = text.slice(0, candidate.end) + candidate.closingSuffix;
    try {
      const parsed: unknown = JSON.parse(stripTrailingCommas(candidateText));
      const spec = repairGenuiSpec(parsed);
      if (spec !== null) return spec;
    } catch {
      // 下一个候选
    }
  }
  return null;
}
