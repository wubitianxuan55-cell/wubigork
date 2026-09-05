// GenUI spec guard — 白名单校验 + 确定性修复 + 预算计数。
//
// 与上游 dsh-genui 的区别（gaea 收紧版）：
//   - 未知 type 直接丢弃整节点（上游为插件扩展而透传；gaea 无扩展生态）；
//   - 渲染器只认 guard 的输出，任何恶意/病态规格在到达组件前已被裁剪。
// 修复幂等且前缀稳定：流式 chunk 已存活组件在其后 chunk 到达时位置不变。

import {
  GENUI_LIMITS,
  GENUI_NODE_TYPES,
  type GenuiAvatar,
  type GenuiAccordion,
  type GenuiBadge,
  type GenuiButton,
  type GenuiCallout,
  type GenuiCard,
  type GenuiChart,
  type GenuiCheckbox,
  type GenuiCode,
  type GenuiCol,
  type GenuiCopy,
  type GenuiDiff,
  type GenuiDivider,
  type GenuiGrid,
  type GenuiInput,
  type GenuiJson,
  type GenuiKeyValue,
  type GenuiList,
  type GenuiNode,
  type GenuiProgress,
  type GenuiQuiz,
  type GenuiRadio,
  type GenuiRow,
  type GenuiSelect,
  type GenuiSlider,
  type GenuiSpacer,
  type GenuiSpec,
  type GenuiStat,
  type GenuiSteps,
  type GenuiSubmit,
  type GenuiSwitch,
  type GenuiTable,
  type GenuiTabs,
  type GenuiText,
  type GenuiTextarea,
  type GenuiTimeline,
} from "./spec";

interface RepairCtx {
  used: number;
}

const MAX = GENUI_LIMITS;

function obj(v: unknown): Record<string, unknown> | null {
  if (typeof v !== "object" || v === null || Array.isArray(v)) return null;
  return v as Record<string, unknown>;
}

/** 必填字符串：缺失/类型错返回 undefined。 */
function str(v: unknown, cap: number = MAX.maxString): string | undefined {
  if (typeof v !== "string") return undefined;
  return v.length > cap ? v.slice(0, cap) : v;
}

/** 可选字符串：非字符串返回 undefined。 */
function optStr(v: unknown, cap: number = MAX.maxString): string | undefined {
  if (v === undefined) return undefined;
  return str(v, cap);
}

/** 有限数字并 clamp 到 [min,max]；非有限/非数字返回 undefined。 */
function num(v: unknown, min: number, max: number): number | undefined {
  if (typeof v !== "number" || !Number.isFinite(v)) return undefined;
  return Math.min(max, Math.max(min, v));
}

/** 可选数字。 */
function optNum(v: unknown, min: number, max: number): number | undefined {
  if (v === undefined) return undefined;
  return num(v, min, max);
}

/** 整数并 clamp。 */
function int(v: unknown, min: number, max: number): number | undefined {
  const n = num(v, min, max);
  if (n === undefined) return undefined;
  return Math.round(n);
}

function optInt(v: unknown, min: number, max: number): number | undefined {
  if (v === undefined) return undefined;
  return int(v, min, max);
}

function bool(v: unknown): boolean | undefined {
  if (typeof v !== "boolean") return undefined;
  return v;
}

function optBool(v: unknown): boolean | undefined {
  if (v === undefined) return undefined;
  return bool(v);
}

const HEX_COLOR = /^#([\da-fA-F]{3,4}|[\da-fA-F]{6}|[\da-fA-F]{8})$/;
const NAME_COLOR = /^[a-z]{3,24}$/;

function color(v: unknown): string | undefined {
  if (typeof v !== "string") return undefined;
  const c = v.trim();
  if (c.length > 40) return undefined;
  return HEX_COLOR.test(c) || NAME_COLOR.test(c) ? c : undefined;
}

function take(ctx: RepairCtx): boolean {
  if (ctx.used >= MAX.maxNodes) return false;
  ctx.used += 1;
  return true;
}

function strList(v: unknown, cap: number = MAX.maxOptions): string[] | undefined {
  if (!Array.isArray(v)) return undefined;
  const out: string[] = [];
  for (const item of v) {
    const s = str(item, MAX.maxString);
    if (s !== undefined) out.push(s);
    if (out.length >= cap) break;
  }
  return out;
}

function isTone(v: unknown, tones: ReadonlySet<string>): string | undefined {
  return typeof v === "string" && tones.has(v) ? v : undefined;
}

const BADGE_TONES = new Set(["success", "warn", "danger", "accent"]);
const CALL_TONES = new Set(["info", "success", "warning", "error"]);
const BUTTON_TONES = new Set(["primary", "danger", "success", "ghost"]);
const INPUT_TYPES = new Set(["text", "email", "password"]);
const CHART_KINDS = new Set(["bars", "line", "donut"]);
const TEXT_SIZES = new Set(["h1", "h2", "h3", "body", "muted", "caption"]);

function chartPoint(v: unknown): { label: string; value: number; color?: string } | undefined {
  const o = obj(v);
  if (!o) return undefined;
  const label = str(o.label ?? "");
  const value = num(o.value, -1e12, 1e12);
  if (label === undefined || value === undefined) return undefined;
  const c = color(o.color);
  return c === undefined ? { label, value } : { label, value, color: c };
}

function repairChildren(v: unknown, ctx: RepairCtx, depth: number): GenuiNode[] {
  if (!Array.isArray(v)) return [];
  if (depth > MAX.maxDepth) return [];
  const out: GenuiNode[] = [];
  for (const child of v) {
    const n = repairNode(child, ctx, depth + 1);
    if (n !== null) out.push(n);
    if (ctx.used >= MAX.maxNodes) break;
  }
  return out;
}

function repairNode(v: unknown, ctx: RepairCtx, depth: number): GenuiNode | null {
  const o = obj(v);
  if (!o || typeof o.type !== "string") return null;
  const type = o.type;
  if (!GENUI_NODE_TYPES.has(type)) return null;
  if (!take(ctx)) return null;
  if (depth > MAX.maxDepth) return null;

  switch (type) {
    case "text": {
      const content = str(o.content);
      if (content === undefined) return null;
      const n: GenuiText = {
        type: "text",
        content,
        ...(optStr(o.size, 16) !== undefined && TEXT_SIZES.has(o.size as string)
          ? { size: o.size as GenuiText["size"] }
          : {}),
        ...(optBool(o.center) !== undefined ? { center: o.center as boolean } : {}),
      };
      return n;
    }
    case "row":
    case "col": {
      const items = repairChildren(o.items, ctx, depth);
      const n: GenuiRow | GenuiCol = {
        type,
        items,
        ...(optBool(o.wrap) !== undefined ? { wrap: o.wrap as boolean } : {}),
        ...(optInt(o.gap, 0, 32) !== undefined ? { gap: o.gap as number } : {}),
      };
      return n;
    }
    case "grid": {
      const cols = int(o.cols, 1, MAX.maxGridCols);
      if (cols === undefined) return null;
      return { type: "grid", cols, items: repairChildren(o.items, ctx, depth) } as GenuiGrid;
    }
    case "card": {
      const n: GenuiCard = {
        type: "card",
        ...(optStr(o.title) !== undefined ? { title: o.title as string } : {}),
        items: repairChildren(o.items, ctx, depth),
      };
      return n;
    }
    case "divider":
      return { type: "divider" } as GenuiDivider;
    case "spacer":
      return { type: "spacer" } as GenuiSpacer;

    case "stat": {
      const label = str(o.label);
      const value = str(o.value);
      if (label === undefined || value === undefined) return null;
      const n: GenuiStat = {
        type: "stat",
        label,
        value,
        ...(optStr(o.delta, 200) !== undefined ? { delta: o.delta as string } : {}),
      };
      return n;
    }
    case "badge": {
      const label = str(o.label, 200);
      if (label === undefined) return null;
      const n: GenuiBadge = {
        type: "badge",
        label,
        ...(isTone(o.tone, BADGE_TONES) !== undefined ? { tone: o.tone as GenuiBadge["tone"] } : {}),
        ...(optStr(o.icon, 24) !== undefined ? { icon: o.icon as string } : {}),
      };
      return n;
    }
    case "progress": {
      const value = num(o.value, 0, 100);
      if (value === undefined) return null;
      const n: GenuiProgress = {
        type: "progress",
        value,
        ...(optStr(o.label, 200) !== undefined ? { label: o.label as string } : {}),
        ...(optStr(o.valueLabel, 200) !== undefined ? { valueLabel: o.valueLabel as string } : {}),
      };
      return n;
    }
    case "keyvalue": {
      if (!Array.isArray(o.pairs)) return null;
      const pairs: { key: string; value: string }[] = [];
      for (const raw of o.pairs) {
        const p = obj(raw);
        if (!p) continue;
        const key = str(p.key, 200);
        const value = str(p.value, MAX.maxString);
        if (key === undefined || value === undefined) continue;
        pairs.push({ key, value });
        if (pairs.length >= MAX.maxKeyValuePairs) break;
      }
      return { type: "keyvalue", pairs } as GenuiKeyValue;
    }
    case "list": {
      if (!Array.isArray(o.items)) return null;
      const items: GenuiList["items"] = [];
      for (const item of o.items) {
        if (typeof item === "string") {
          const t = str(item);
          if (t !== undefined) items.push(t);
        } else {
          const node = repairNode(item, ctx, depth + 1);
          if (node !== null) items.push(node);
        }
        if (items.length >= MAX.maxListItems || ctx.used >= MAX.maxNodes) break;
      }
      return { type: "list", items } as GenuiList;
    }
    case "table": {
      const columns = strList(o.columns, MAX.maxTableCols);
      if (columns === undefined) return null;
      const rows: (string | number)[][] = [];
      if (Array.isArray(o.rows)) {
        for (const raw of o.rows) {
          if (!Array.isArray(raw)) continue;
          const row: (string | number)[] = [];
          for (let i = 0; i < columns.length; i++) {
            const cell = raw[i];
            if (typeof cell === "string") {
              const s = str(cell, MAX.maxString);
              if (s !== undefined) row.push(s);
              else row.push("");
            } else if (typeof cell === "number" && Number.isFinite(cell)) {
              row.push(Math.min(1e15, Math.max(-1e15, cell)));
            } else {
              row.push("");
            }
          }
          rows.push(row);
          if (rows.length >= MAX.maxTableRows) break;
        }
      }
      return { type: "table", columns, rows } as GenuiTable;
    }
    case "timeline": {
      if (!Array.isArray(o.items)) return null;
      const items: GenuiTimeline["items"] = [];
      for (const raw of o.items) {
        const p = obj(raw);
        if (!p) continue;
        const title = str(p.title, 500);
        if (title === undefined) continue;
        const item: GenuiTimeline["items"][number] = { title };
        if (p.desc !== undefined) {
          const d = optStr(p.desc, MAX.maxString);
          if (d !== undefined) item.desc = d;
        }
        if (p.time !== undefined) {
          const t = optStr(p.time, 200);
          if (t !== undefined) item.time = t;
        }
        items.push(item);
        if (items.length >= MAX.maxTimelineItems) break;
      }
      return { type: "timeline", items } as GenuiTimeline;
    }
    case "callout": {
      const content = str(o.content);
      if (content === undefined) return null;
      const n: GenuiCallout = {
        type: "callout",
        ...(isTone(o.tone, CALL_TONES) !== undefined ? { tone: o.tone as GenuiCallout["tone"] } : {}),
        ...(optStr(o.title, 500) !== undefined ? { title: o.title as string } : {}),
        content,
      };
      return n;
    }
    case "steps": {
      if (!Array.isArray(o.steps)) return null;
      const steps: GenuiSteps["steps"] = [];
      for (const raw of o.steps) {
        const p = obj(raw);
        if (!p) continue;
        const title = str(p.title, 500);
        if (title === undefined) continue;
        const step: GenuiSteps["steps"][number] = { title };
        const desc = p.desc === undefined ? undefined : optStr(p.desc, MAX.maxString);
        if (desc !== undefined) step.desc = desc;
        steps.push(step);
        if (steps.length >= MAX.maxSteps) break;
      }
      if (steps.length === 0) return null;
      const n: GenuiSteps = {
        type: "steps",
        ...(optInt(o.current, 1, steps.length) !== undefined
          ? { current: o.current as number }
          : {}),
        steps,
      };
      return n;
    }
    case "avatar": {
      const name = str(o.name, 100);
      if (name === undefined) return null;
      const n: GenuiAvatar = {
        type: "avatar",
        name,
        ...(color(o.color) !== undefined ? { color: o.color as string } : {}),
      };
      return n;
    }
    case "copy": {
      const text = str(o.text);
      if (text === undefined) return null;
      const n: GenuiCopy = {
        type: "copy",
        text,
        ...(optStr(o.label, 200) !== undefined ? { label: o.label as string } : {}),
      };
      return n;
    }

    case "chart": {
      if (!Array.isArray(o.data)) return null;
      const kind = isTone(o.kind, CHART_KINDS);
      const data: GenuiChart["data"] = [];
      for (const p of o.data) {
        const point = chartPoint(p);
        if (point !== undefined) data.push(point);
        if (data.length >= MAX.maxChartPoints) break;
      }
      const series: GenuiChart["series"] = [];
      if (Array.isArray(o.series)) {
        for (const raw of o.series) {
          const s = obj(raw);
          if (!s) continue;
          const label = str(s.label, 200);
          if (label === undefined || !Array.isArray(s.data)) continue;
          const points: { label: string; value: number; color?: string }[] = [];
          for (const p of s.data) {
            const point = chartPoint(p);
            if (point !== undefined) points.push(point);
            if (points.length >= MAX.maxChartPoints) break;
          }
          const item: { label: string; color?: string; data: { label: string; value: number; color?: string }[] } = {
            label,
            data: points,
            ...(color(s.color) !== undefined ? { color: s.color as string } : {}),
          };
          series.push(item);
          if (series.length >= 8) break;
        }
      }
      if (data.length === 0 && series.length === 0) return null;
      const n: GenuiChart = {
        type: "chart",
        ...(kind !== undefined ? { kind: kind as GenuiChart["kind"] } : {}),
        data,
        ...(series.length > 0 ? { series } : {}),
      };
      return n;
    }

    case "code": {
      const code = str(o.code, MAX.maxCode);
      if (code === undefined) return null;
      const n: GenuiCode = {
        type: "code",
        code,
        ...(optStr(o.lang, 40) !== undefined ? { lang: o.lang as string } : {}),
      };
      return n;
    }
    case "json": {
      if (!("value" in o)) return null;
      let serialized = "";
      try {
        serialized = JSON.stringify(o.value);
      } catch {
        return null;
      }
      if (serialized === undefined || serialized.length > MAX.maxCode) return null;
      const n: GenuiJson = { type: "json", value: o.value };
      return n;
    }
    case "diff": {
      if (!Array.isArray(o.diffs)) return null;
      const diffs: GenuiDiff["diffs"] = [];
      for (const raw of o.diffs) {
        const d = obj(raw);
        if (!d) continue;
        const path = str(d.path, 500);
        if (path === undefined) continue;
        const newText = str(d.newText, MAX.maxString);
        if (newText === undefined) continue;
        const item: GenuiDiff["diffs"][number] = {
          path,
          newText,
          ...(d.oldText === null || d.oldText === undefined
            ? {}
            : (() => {
                const oldText = optStr(d.oldText, MAX.maxString);
                return oldText !== undefined ? { oldText } : {};
              })()),
        };
        diffs.push(item);
        if (diffs.length >= MAX.maxDiffs) break;
      }
      return { type: "diff", diffs } as GenuiDiff;
    }

    case "button": {
      const label = str(o.label, 200);
      if (label === undefined) return null;
      const n: GenuiButton = {
        type: "button",
        label,
        ...(isTone(o.tone, BUTTON_TONES) !== undefined
          ? { tone: o.tone as GenuiButton["tone"] }
          : {}),
        ...(optBool(o.full) !== undefined ? { full: o.full as boolean } : {}),
        ...(optBool(o.small) !== undefined ? { small: o.small as boolean } : {}),
        ...(optStr(o.icon, 24) !== undefined ? { icon: o.icon as string } : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
      };
      return n;
    }
    case "input": {
      const n: GenuiInput = {
        type: "input",
        ...(optStr(o.label, 200) !== undefined ? { label: o.label as string } : {}),
        ...(optStr(o.placeholder, 200) !== undefined ? { placeholder: o.placeholder as string } : {}),
        ...(optStr(o.value, 1000) !== undefined ? { value: o.value as string } : {}),
        ...(isTone(o.inputType, INPUT_TYPES) !== undefined
          ? { inputType: o.inputType as GenuiInput["inputType"] }
          : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
        ...(optStr(o.id, 64) !== undefined ? { id: o.id as string } : {}),
      };
      return n;
    }
    case "select": {
      const options = strList(o.options);
      if (options === undefined || options.length === 0) return null;
      const n: GenuiSelect = {
        type: "select",
        options,
        ...(optStr(o.label, 200) !== undefined ? { label: o.label as string } : {}),
        ...(optInt(o.selected, -1, options.length - 1) !== undefined && (o.selected as number) >= 0
          ? { selected: Math.round(o.selected as number) }
          : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
        ...(optStr(o.id, 64) !== undefined ? { id: o.id as string } : {}),
      };
      return n;
    }
    case "checkbox":
    case "switch": {
      const label = str(o.label, 200);
      if (label === undefined) return null;
      const n: GenuiCheckbox | GenuiSwitch = {
        type,
        label,
        ...(optBool(o.checked) !== undefined ? { checked: o.checked as boolean } : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
      };
      return n;
    }
    case "radio": {
      const options = strList(o.options);
      if (options === undefined || options.length === 0) return null;
      const n: GenuiRadio = {
        type: "radio",
        options,
        ...(optStr(o.label, 200) !== undefined ? { label: o.label as string } : {}),
        ...(optInt(o.selected, -1, options.length - 1) !== undefined && (o.selected as number) >= 0
          ? { selected: Math.round(o.selected as number) }
          : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
        ...(optStr(o.group, 64) !== undefined ? { group: o.group as string } : {}),
        ...(o.answer !== undefined && (typeof o.answer === "string" || typeof o.answer === "number")
          ? { answer: o.answer }
          : {}),
        ...(optStr(o.explanation, 2000) !== undefined
          ? { explanation: o.explanation as string }
          : {}),
      };
      return n;
    }
    case "slider": {
      const min = optNum(o.min, -1e6, 1e6) ?? 0;
      const maxRaw = optNum(o.max, -1e6, 1e6) ?? 100;
      const max = maxRaw <= min ? min + 1 : maxRaw;
      const step = optNum(o.step, 0.001, 1e6) ?? 1;
      const value = optNum(o.value, min, max);
      const n: GenuiSlider = {
        type: "slider",
        min,
        max,
        step,
        ...(value !== undefined ? { value } : {}),
        ...(optStr(o.label, 200) !== undefined ? { label: o.label as string } : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
        ...(optStr(o.id, 64) !== undefined ? { id: o.id as string } : {}),
      };
      return n;
    }
    case "textarea": {
      const n: GenuiTextarea = {
        type: "textarea",
        ...(optStr(o.label, 200) !== undefined ? { label: o.label as string } : {}),
        ...(optStr(o.placeholder, 500) !== undefined ? { placeholder: o.placeholder as string } : {}),
        ...(optStr(o.value, 4000) !== undefined ? { value: o.value as string } : {}),
        ...(optInt(o.rows, 1, 30) !== undefined ? { rows: o.rows as number } : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
        ...(optStr(o.id, 64) !== undefined ? { id: o.id as string } : {}),
      };
      return n;
    }
    case "submit": {
      const n: GenuiSubmit = {
        type: "submit",
        ...(optStr(o.label, 200) !== undefined ? { label: o.label as string } : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
        ...(optStr(o.resetAction, MAX.maxAction) !== undefined
          ? { resetAction: o.resetAction as string }
          : {}),
        ...(Array.isArray(o.groups)
          ? { groups: strList(o.groups, MAX.maxOptions) ?? [] }
          : {}),
      };
      return n;
    }
    case "tabs": {
      if (!Array.isArray(o.tabs)) return null;
      const tabs: GenuiTabs["tabs"] = [];
      for (const raw of o.tabs) {
        const t = obj(raw);
        if (!t) continue;
        const label = str(t.label, 200);
        if (label === undefined) continue;
        tabs.push({ label, items: repairChildren(t.items, ctx, depth) });
        if (tabs.length >= MAX.maxTabs) break;
      }
      if (tabs.length === 0) return null;
      return { type: "tabs", tabs } as GenuiTabs;
    }
    case "accordion": {
      if (!Array.isArray(o.items)) return null;
      const items: GenuiAccordion["items"] = [];
      for (const raw of o.items) {
        const a = obj(raw);
        if (!a) continue;
        const title = str(a.title, 500);
        if (title === undefined) continue;
        items.push({ title, items: repairChildren(a.items, ctx, depth) });
        if (items.length >= MAX.maxAccordionItems) break;
      }
      if (items.length === 0) return null;
      return { type: "accordion", items } as GenuiAccordion;
    }
    case "quiz": {
      const question = str(o.question, 2000);
      if (question === undefined || !Array.isArray(o.options)) return null;
      const options: GenuiQuiz["options"] = [];
      for (const raw of o.options) {
        const q = obj(raw);
        if (!q) continue;
        const label = str(q.label, 500);
        if (label === undefined) continue;
        const item: GenuiQuiz["options"][number] = { label };
        if (optBool(q.correct) !== undefined) item.correct = q.correct as boolean;
        const feedback = q.feedback === undefined ? undefined : optStr(q.feedback, 2000);
        if (feedback !== undefined) item.feedback = feedback;
        options.push(item);
        if (options.length >= MAX.maxQuizOptions) break;
      }
      if (options.length < 2) return null;
      const n: GenuiQuiz = {
        type: "quiz",
        question,
        options,
        ...(optStr(o.explanation, 4000) !== undefined
          ? { explanation: o.explanation as string }
          : {}),
        ...(optStr(o.id, 64) !== undefined ? { id: o.id as string } : {}),
        ...(optStr(o.action, MAX.maxAction) !== undefined ? { action: o.action as string } : {}),
      };
      return n;
    }
    default:
      return null;
  }
}

/**
 * 修复单组件 root（围栏体是裸 {"type":"callout",…} 时用）。
 * 内部会消耗节点预算；调用方应自行重置 ctx（经 repairSingleComponent）。
 */
export function repairSingleComponent(v: unknown): GenuiNode | null {
  const ctx: RepairCtx = { used: 0 };
  return repairNode(v, ctx, 0);
}

/** 修复整个规格；结构不可用返回 null（调用方降级代码块）。 */
export function repairGenuiSpec(value: unknown): GenuiSpec | null {
  const o = obj(value);
  if (!o) return null;
  const ctx: RepairCtx = { used: 0 };

  if (Array.isArray(o.items)) {
    const items = repairChildren(o.items, ctx, 0);
    if (items.length === 0) return null;
    const spec: GenuiSpec = { items };
    const title = optStr(o.title, 500);
    if (title !== undefined) spec.title = title;
    const gap = optInt(o.gap, 0, 64);
    if (gap !== undefined) spec.gap = gap;
    if (optBool(o.panel) === true) spec.panel = true;
    if (optBool(o.append) === true) spec.append = true;
    return spec;
  }

  const single = repairSingleComponent(o);
  if (single !== null) return { items: [single] };
  return null;
}

/** 深度优先统计规格节点数（渲染器与工具共用；上限仅为防御性计数）。 */
export function countGenuiNodes(value: unknown, cap = Number.POSITIVE_INFINITY): number {
  const o = obj(value);
  if (!o || typeof o.type !== "string" || !GENUI_NODE_TYPES.has(o.type)) {
    return 0;
  }
  let n = 1;
  if (n >= cap) return n;
  for (const key of ["items", "tabs", "steps", "options"] as const) {
    const arr = o[key];
    if (Array.isArray(arr)) {
      for (const child of arr) {
        n += countGenuiNodes(child, cap - n);
        if (n >= cap) return n;
      }
    }
  }
  if (Array.isArray(o.diffs)) {
    for (const d of o.diffs) {
      if (n >= cap) return n;
      n += countGenuiNodes(d, cap - n);
    }
  }
  if (Array.isArray(o.series)) {
    for (const s of o.series) {
      if (n >= cap) return n;
      n += countGenuiNodes(s, cap - n);
    }
  }
  return n;
}

/** 结构校验：返回人类可读错误（供工具/前端诊断，最多 50 条）。 */
export function validateGenuiSpec(value: unknown): { ok: boolean; errors: string[] } {
  const errors: string[] = [];
  const o = obj(value);
  if (!o) {
    return { ok: false, errors: ["规格必须是 JSON 对象"] };
  }
  const checkNode = (v: unknown, path: string): void => {
    if (errors.length >= 50) return;
    const node = obj(v);
    if (!node) {
      errors.push(`${path}: 应为对象`);
      return;
    }
    if (typeof node.type !== "string" || !GENUI_NODE_TYPES.has(node.type)) {
      errors.push(`${path}: 未知组件 type ${JSON.stringify(node.type)}`);
      return;
    }
    const req = requiredFields(node.type);
    for (const field of req) {
      const exists = node[field] !== undefined;
      if (!exists) errors.push(`${path}: ${node.type} 缺少必填字段 ${field}`);
    }
    for (const key of ["items", "tabs", "options", "steps"] as const) {
      const arr = node[key];
      if (Array.isArray(arr)) {
        arr.forEach((child, i) => checkNode(child, `${path}.${key}[${i}]`));
      }
    }
  };

  if (!Array.isArray(o.items)) {
    errors.push("根规格缺少 items 数组");
  } else {
    if (o.items.length === 0) errors.push("items 不能为空");
    o.items.forEach((child, i) => checkNode(child, `items[${i}]`));
    if (o.items.length > GENUI_LIMITS.maxNodes) {
      errors.push(`items 数量 ${o.items.length} 超出上限 ${GENUI_LIMITS.maxNodes}`);
    }
  }
  if (!Array.isArray(o.items) && !(typeof o.type === "string" && GENUI_NODE_TYPES.has(o.type))) {
    errors.push("规格根必须是 {items:[…]} 或单个组件对象");
  }
  return { ok: errors.length === 0, errors };
}

function requiredFields(type: string): string[] {
  switch (type) {
    case "text":
      return ["content"];
    case "row":
    case "col":
    case "grid":
    case "card":
    case "list":
    case "tabs":
    case "accordion":
      return ["items"];
    case "stat":
      return ["label", "value"];
    case "badge":
      return ["label"];
    case "progress":
      return ["value"];
    case "keyvalue":
      return ["pairs"];
    case "table":
      return ["columns", "rows"];
    case "timeline":
      return ["items"];
    case "callout":
      return ["content"];
    case "steps":
      return ["steps"];
    case "avatar":
      return ["name"];
    case "copy":
      return ["text"];
    case "chart":
      return ["data"];
    case "code":
      return ["code"];
    case "json":
      return ["value"];
    case "diff":
      return ["diffs"];
    case "button":
    case "checkbox":
    case "switch":
      return ["label"];
    case "input":
    case "slider":
    case "textarea":
      return [];
    case "select":
    case "radio":
      return ["options"];
    case "submit":
      return [];
    case "quiz":
      return ["question", "options"];
    default:
      return [];
  }
}
