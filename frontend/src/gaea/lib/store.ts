// Zustand store — replaces useController's useReducer state machine.

import { useCallback, useEffect } from "react";
import { create } from "zustand";
import { useShallow } from "zustand/shallow";
import { app, onEvent, onReady } from "./bridge";
import { noteEventSeq, resetEventSync } from "./eventSync";
import { parseTodos } from "./tools";
import type {
  BalanceInfo, ContextInfo, FactBaseView, HistoryMessage, JobView, MemoryView,
  Meta, ProjectGroup, QuestionAnswer, SessionMeta, SessionStatsView, TCCAReport, WireApproval, WireAsk,
  WireEvent, WireUsage,
} from "./types";

export type ToolStatus = "running" | "done" | "error" | "stopped";

export type Item =
  | { kind: "user"; id: string; text: string }
  // subagentRef：message 事件可选携带的子代理来源引用（v4.26），渲染层据此画
  // 「子代理」徽标；旧事件无此字段时为 undefined，行为不变。
  | { kind: "assistant"; id: string; text: string; reasoning: string; streaming: boolean; subagentRef?: string }
  | { kind: "phase"; id: string; text: string }
  | { kind: "notice"; id: string; level: "info" | "warn"; text: string }
  | { kind: "compaction"; id: string; pending: boolean; trigger: string; messages: number; summary: string; archive: string }
  | { kind: "tool"; id: string; name: string; args: string; readOnly: boolean; status: ToolStatus; output?: string; error?: string; truncated?: boolean; recoverable?: boolean; parentId?: string };

export interface ControllerState {
  items: Item[]; running: boolean; turnActive: boolean; approval?: WireApproval; ask?: WireAsk;
  usage?: WireUsage; context: ContextInfo; meta?: Meta; balance?: BalanceInfo; jobs: JobView[]; factBase: FactBaseView;
  tcca?: TCCAReport;
  currentAssistant?: string; pendingUser?: string; discardTurn?: boolean;
  lastAssistantIdx: number; // 最后一个 assistant 项的索引，避免流式 text/reasoning 事件中 O(n) 反向查找
  turnStartAt: number; turnTokens: number; seq: number;
  sessionTotal: number;
  perTurnUsage: WireUsage | null | undefined; // V5.30: whole-turn accumulated usage
  perTurnExecutorUsage?: WireUsage; // 执行模型 usage
  perTurnSubUsage?: WireUsage;      // V10.22: subagent usage only
  turnSteps: WireUsage[]; // V5.31: raw per-step usage within current turn
  sessionNonce: number; // V5.25: 每次新建/恢复会话递增，确保统计面板按会话区分
  // 会话级派生统计（事件日志重放）：恢复/加载会话后由后端回填，
  // 补足「恢复的长会话成本展示不完整」（评审缺陷 11）。
  sessionStats?: SessionStatsView;
  _dispatch: (a: Action) => void;
}

type Action =
  | { type: "event"; e: WireEvent } | { type: "user"; text: string } | { type: "unsend" }
  | { type: "localCancel" }
  | { type: "meta"; meta: Meta } | { type: "context"; context: ContextInfo }
  | { type: "balance"; balance: BalanceInfo } | { type: "jobs"; jobs: JobView[] } | { type: "factbase"; factBase: FactBaseView }
  | { type: "tcca"; report: TCCAReport }
  | { type: "sessionStats"; stats?: SessionStatsView }
  | { type: "history"; messages: HistoryMessage[] }
  // v4.26 序号防线补拉：items 为后端 GaeaResyncEvents 折叠快照（原始 JSON，
  // 由 reducer 内 parseResyncItems 校验后落库）。
  | { type: "resync"; items: unknown[] }
  | { type: "clearApproval" } | { type: "clearAsk" } | { type: "reset" };


export function flushPendingUser(s: ControllerState): ControllerState {
  if (s.pendingUser === undefined) return s;
  // 如果消息已通过 user action 立即加入 items，只清除 pendingUser 避免重复
  const last = s.items.length > 0 ? s.items[s.items.length - 1] : null;
  if (last && last.kind === "user" && last.text === s.pendingUser) {
    return { ...s, pendingUser: undefined };
  }
  return { ...s, seq: s.seq + 1, items: [...s.items, { kind: "user", id: `u${s.seq}`, text: s.pendingUser }], pendingUser: undefined };
}

// rebuildHistoryItems 把后端历史消息还原为对话项：用户/助手正文 + 工具
// dispatch（按 id 合并结果，渲染为完成态工具卡片），恢复会话后过程卡、
// 「变更」面板与待办提取依然可用。
export function rebuildHistoryItems(messages: HistoryMessage[]): { items: Item[]; lastAssistantIdx: number } {
  const results = new Map<string, string>();
  for (const m of messages) {
    if (m.role === "tool_result" && m.toolId) results.set(m.toolId, m.content ?? "");
  }
  const items: Item[] = [];
  messages.forEach((m, i) => {
    if (m.role === "user" && m.content.trim() !== "") {
      items.push({ kind: "user", id: `h${items.length}`, text: m.content } as Item);
    } else if (m.role === "assistant" && m.content.trim() !== "") {
      // v4.34 线B：assistant 历史条目透传子代理答复引用（Go HistoryMessage.SubagentRef，
      // 与实时 message 事件 / GaeaResyncItem 同键位）；空串归一为 undefined（后端
      // omitempty 下缺键即 undefined），避免恢复后「子代理」徽标空渲染。
      // user/tool 分支零改动。
      items.push({ kind: "assistant", id: `h${items.length}`, text: m.content, reasoning: "", streaming: false, subagentRef: m.subagentRef || undefined } as Item);
    } else if (m.role === "tool" && m.toolName) {
      const id = m.toolId || `ht${i}`;
      const hasResult = results.has(id);
      items.push({
        kind: "tool", id, name: m.toolName, args: m.toolArgs ?? "",
        readOnly: false,
        // 有 tool_result → 完成态；无结果 = 该调用未执行完（运行中被看门狗
        // localCancel / 中断保存），还原为 stopped 而非假完成，避免恢复后
        // 工具卡显示「已完成」却无任何输出（03-office-frontend.md §6 缺陷 2）。
        status: hasResult ? "done" : "stopped",
        output: results.get(id) ?? "",
      } as Item);
    }
  });
  const lastAssistantIdx = items.reduceRight((acc, it, idx) => acc >= 0 ? acc : it.kind === "assistant" ? idx : -1, -1);
  return { items, lastAssistantIdx };
}

// parseResyncItems 校验后端 GaeaResyncEvents 折叠快照（v4.26 序号防线，配套
// reducer case "resync"）：items 必须是可全部识别的前端 Item 视图 JSON
// （user/assistant/phase/notice/compaction/tool 六种），任一条形状不合法即整体
// 判坏返回 null——reducer 据此静默保底、保留现有 items（不做宽容合并）。空数组
// 同样判坏：补拉只发生在对话进行中，空快照说明后端读日志失败，直接替换会清空
// 对话窗。快照 assistant 一律视为非流式（磁盘折叠没有「正在输出」的概念），
// 流式续接由 reducer 负责。
// v4.26.2 缺省键宽容：字符串/布尔字段**缺键**视为零值（""/false）——Go 侧
// omitempty 会把空串/false 的键整个省略，此前严格校验把真实流式回合的快照
// 100% 判坏弃用，序号防线静默失效（对话窗只剩 WorkHeader 读秒）。**类型错仍拒**
// （present 但非 string/boolean），kind/id/status 枚举校验不变。
export function parseResyncItems(raw: unknown): Item[] | null {
  if (!Array.isArray(raw) || raw.length === 0) return null;
  const str = (v: unknown): string => (typeof v === "string" ? v : "");
  const optStr = (v: unknown): string | undefined => (typeof v === "string" ? v : undefined);
  // absentOk：缺键 → 零值；present 但类型错 → null（调用方判坏整快照）。
  const absentStr = (v: unknown): string | null => (v === undefined ? "" : typeof v === "string" ? v : null);
  const absentBool = (v: unknown): boolean | null => (v === undefined ? false : typeof v === "boolean" ? v : null);
  const items: Item[] = [];
  for (const it of raw) {
    if (it === null || typeof it !== "object" || Array.isArray(it)) return null;
    const o = it as Record<string, unknown>;
    const id = typeof o.id === "string" && o.id !== "" ? o.id : null;
    if (id === null) return null;
    switch (o.kind) {
      case "user": {
        const text = absentStr(o.text);
        if (text === null) return null;
        items.push({ kind: "user", id, text });
        break;
      }
      case "assistant": {
        const text = absentStr(o.text);
        const reasoning = absentStr(o.reasoning);
        if (text === null || reasoning === null) return null;
        items.push({ kind: "assistant", id, text, reasoning, streaming: false, subagentRef: optStr(o.subagentRef) });
        break;
      }
      case "phase": {
        const text = absentStr(o.text);
        if (text === null) return null;
        items.push({ kind: "phase", id, text });
        break;
      }
      case "notice": {
        const text = absentStr(o.text);
        if (text === null) return null;
        items.push({ kind: "notice", id, level: o.level === "warn" ? "warn" : "info", text });
        break;
      }
      case "compaction": {
        if (typeof o.pending !== "boolean") return null;
        items.push({
          kind: "compaction", id, pending: o.pending, trigger: str(o.trigger),
          messages: typeof o.messages === "number" && Number.isFinite(o.messages) ? o.messages : 0,
          summary: str(o.summary), archive: str(o.archive),
        });
        break;
      }
      case "tool": {
        if (typeof o.name !== "string") return null;
        const readOnly = absentBool(o.readOnly);
        const args = absentStr(o.args);
        if (readOnly === null || args === null) return null;
        const st = o.status === undefined ? "done" : o.status;
        if (st !== "running" && st !== "done" && st !== "error" && st !== "stopped") return null;
        items.push({
          kind: "tool", id, name: o.name, args, readOnly, status: st,
          output: optStr(o.output), error: optStr(o.error),
          truncated: o.truncated === true ? true : undefined,
          recoverable: o.recoverable === true ? true : undefined,
          parentId: optStr(o.parentId),
        });
        break;
      }
      default:
        return null; // 未知 kind：整个快照不可信，保底不替换
    }
  }
  return items;
}

// 待办收尾：turn 正常结束但 todo 列表从未推进（没有 completed、也没有
// in_progress）时，说明 agent 干完活忘了回写状态；把展示状态置为
// completed，避免“任务已完成却一直显示 0/N 待办”。已被 agent 推进过的
// 列表保持原样（可能是跨轮计划，不能擅自收尾）。
function finalizeStaleTodos(items: Item[]): Item[] {
  let idx = -1;
  for (let i = items.length - 1; i >= 0; i--) {
    const it = items[i];
    if (it.kind === "tool" && it.name === "todo_write" && !it.parentId) {
      idx = i;
      break;
    }
  }
  if (idx < 0) return items;
  const it = items[idx];
  if (it.kind !== "tool") return items;
  const todos = parseTodos(it.args);
  if (todos.length === 0) return items;
  if (todos.some((t) => t.status === "completed" || t.status === "in_progress")) return items;
  const next = [...items];
  next[idx] = {
    ...it,
    args: JSON.stringify({ todos: todos.map((t) => ({ ...t, status: "completed" })) }),
  };
  return next;
}


export function applyEvent(s: ControllerState, e: WireEvent): ControllerState {
  if (s.discardTurn) { if (e.kind === "turn_done") return { ...s, discardTurn: false, running: false, turnActive: false, currentAssistant: undefined }; return s; }
  if (s.pendingUser !== undefined && e.kind !== "turn_started" && e.kind !== "turn_done") s = flushPendingUser(s);
  switch (e.kind) {
    case "turn_started": return { ...s, running: true, turnActive: true, currentAssistant: undefined, lastAssistantIdx: -1, turnStartAt: Date.now(), turnTokens: 0, perTurnUsage: null, perTurnExecutorUsage: undefined, perTurnSubUsage: undefined, turnSteps: [] };
    case "text": case "reasoning": {
      // O(1) 查找最后一个 assistant 项：用 lastAssistantIdx 避免流式时每 chunk O(n) 扫描。
      // 若最后 assistant 已终结（上一轮 turn_done 已将 streaming 置 false）且当前轮活跃，
      // 则创建新项而非追加到旧轮次消息——修复跨轮次文本覆盖。
      const delta = e.text ?? e.reasoning ?? "";
      let idx = s.lastAssistantIdx;
      // 验证缓存索引有效性（非流式事件间可能有 items 变更）
      if (idx < 0 || idx >= s.items.length || s.items[idx].kind !== "assistant") {
        for (let i = s.items.length - 1; i >= 0; i--) {
          if (s.items[i].kind === "assistant") { idx = i; break; }
        }
      }
      const needNew = idx < 0 || (
        (s.items[idx] as Extract<Item, { kind: "assistant" }>).streaming === false &&
        s.turnActive
      );
      if (!needNew) {
        const it = s.items[idx] as Extract<Item, { kind: "assistant" }>;
        const next = [...s.items];
        next[idx] = e.kind === "text"
          ? { ...it, text: it.text + delta, streaming: true }
          : { ...it, reasoning: it.reasoning + delta, streaming: true };
        return { ...s, items: next, currentAssistant: it.id, lastAssistantIdx: idx };
      }
      // 没有可追加的活跃 assistant 项时创建新的
      const id = `a${s.seq}`;
      const newIdx = s.items.length;
      return { ...s, seq: s.seq + 1, items: [...s.items, { kind: "assistant", id, text: e.kind === "text" ? delta : "", reasoning: e.kind === "reasoning" ? delta : "", streaming: true }], currentAssistant: id, lastAssistantIdx: newIdx };
    }
    case "message": {
      // 始终更新最后一个 assistant，不创建新的。
      // 若最后 assistant 已终结（上一轮结束）且当前轮活跃，则创建新项
      // 而非覆盖旧轮次消息——修复跨轮次文本覆盖。
      let idx = s.lastAssistantIdx;
      if (idx < 0 || idx >= s.items.length || s.items[idx].kind !== "assistant") {
        for (let i = s.items.length - 1; i >= 0; i--) {
          if (s.items[i].kind === "assistant") { idx = i; break; }
        }
      }
      const needNew = idx < 0 || (
        (s.items[idx] as Extract<Item, { kind: "assistant" }>).streaming === false &&
        s.turnActive
      );
      if (!needNew) {
        const it = s.items[idx] as Extract<Item, { kind: "assistant" }>;
        const next = [...s.items];
        // subagentRef 用整体替换而非保底透传：子代理答复回投后主回答的最终
        // message 不带该字段，若保底会把「子代理」徽标黏到主回答气泡上。
        next[idx] = { ...it, text: e.text ?? it.text, reasoning: e.reasoning ?? it.reasoning, streaming: false, subagentRef: e.subagentRef };
        return { ...s, items: next, currentAssistant: undefined, lastAssistantIdx: idx };
      }
      // 没有任何可更新的 assistant 项时创建新的（首轮且模型直接回了 message）
      const id = `a${s.seq}`;
      const newIdx = s.items.length;
      return { ...s, seq: s.seq + 1, items: [...s.items, { kind: "assistant", id, text: e.text ?? "", reasoning: e.reasoning ?? "", streaming: false, subagentRef: e.subagentRef }], currentAssistant: undefined, lastAssistantIdx: newIdx };
    }
    case "tool_dispatch": {
      const t = e.tool; if (!t) return s;
      const id = t.id || `tool${s.seq}`;
      const idx = s.items.findIndex(it => it.kind === "tool" && it.id === id);
      if (idx >= 0) { const next = [...s.items]; const it = next[idx]; if (it.kind === "tool") next[idx] = { ...it, name: t.name, args: t.args ? t.args : it.args, readOnly: t.readOnly }; return { ...s, currentAssistant: undefined, items: next }; }
      return { ...s, seq: s.seq + 1, currentAssistant: undefined, items: [...s.items, { kind: "tool", id, name: t.name, args: t.args ?? "", readOnly: t.readOnly, status: "running", parentId: t.parentId }] };
    }
    case "tool_result": {
      const t = e.tool; if (!t) return s; const next = [...s.items];
      let idx = t.id ? next.findIndex(it => it.kind === "tool" && it.id === t.id) : -1;
      // Fallback: no exact ID match — find the last still-running tool
      if (idx < 0) {
        for (let i = next.length - 1; i >= 0; i--) {
          const cand = next[i];
          if (cand.kind === "tool" && cand.status === "running") { idx = i; break; }
        }
      }
      if (idx >= 0) { const it = next[idx]; if (it.kind === "tool") next[idx] = { ...it, status: t.err ? "error" : "done", output: t.output, error: t.err, recoverable: t.recoverable, truncated: t.truncated }; }
      return { ...s, items: next };
    }
    case "usage": {
      const used = e.usage && s.context.window ? e.usage.promptTokens : s.context.used;
      const u = e.usage;
      // combined accumulator (backwards compat)
      const acc = s.perTurnUsage && u ? {
        promptTokens: s.perTurnUsage.promptTokens + (u.promptTokens ?? 0),
        completionTokens: s.perTurnUsage.completionTokens + (u.completionTokens ?? 0),
        totalTokens: s.perTurnUsage.totalTokens + (u.totalTokens ?? 0),
        cacheHitTokens: s.perTurnUsage.cacheHitTokens + (u.cacheHitTokens ?? 0),
        cacheMissTokens: s.perTurnUsage.cacheMissTokens + (u.cacheMissTokens ?? 0),
        sessionCacheHitTokens: u.sessionCacheHitTokens > 0 ? u.sessionCacheHitTokens : (s.perTurnUsage?.sessionCacheHitTokens ?? 0),
        sessionCacheMissTokens: u.sessionCacheMissTokens > 0 ? u.sessionCacheMissTokens : (s.perTurnUsage?.sessionCacheMissTokens ?? 0),
        costUsd: (s.perTurnUsage.costUsd ?? 0) + (u.costUsd ?? 0),
      } : u;
      // split by source — executor / subagent
      const isSub = u?.source === "subagent";
      const isExecutor = u && !isSub; // "main", "executor" or legacy without source
      const prevExecutor = s.perTurnExecutorUsage, prevSub = s.perTurnSubUsage;
      const accSrc = (prev?: WireUsage, cur?: WireUsage) => !cur ? prev : !prev ? cur : {
        promptTokens: prev.promptTokens + cur.promptTokens,
        completionTokens: prev.completionTokens + cur.completionTokens,
        totalTokens: prev.totalTokens + cur.totalTokens,
        cacheHitTokens: prev.cacheHitTokens + cur.cacheHitTokens,
        cacheMissTokens: prev.cacheMissTokens + cur.cacheMissTokens,
        sessionCacheHitTokens: cur.sessionCacheHitTokens > 0 ? cur.sessionCacheHitTokens : prev.sessionCacheHitTokens,
        sessionCacheMissTokens: cur.sessionCacheMissTokens > 0 ? cur.sessionCacheMissTokens : prev.sessionCacheMissTokens,
        costUsd: (prev.costUsd ?? 0) + (cur.costUsd ?? 0),
      };
      const tagged = u ? { ...u } : undefined; const steps = tagged ? [...s.turnSteps, tagged] : s.turnSteps;
      return { ...s, usage: tagged, perTurnUsage: acc, perTurnExecutorUsage: accSrc(prevExecutor, isExecutor ? u : undefined), perTurnSubUsage: accSrc(prevSub, isSub ? u : undefined), turnSteps: steps, context: { ...s.context, used }, turnTokens: s.turnTokens + (tagged?.completionTokens ?? 0) };
    }
    case "notice": return { ...s, running: s.turnActive ? s.running : false, seq: s.seq + 1, items: [...s.items, { kind: "notice", id: `n${s.seq}`, level: e.level ?? "info", text: e.text ?? "" }] };
    case "phase": return { ...s, seq: s.seq + 1, items: [...s.items, { kind: "phase", id: `p${s.seq}`, text: e.text ?? "" }] };
    case "approval_request": return { ...s, approval: e.approval };
    case "ask_request": return { ...s, ask: e.ask };
    case "turn_done": {
      if (s.pendingUser !== undefined) s = flushPendingUser(s);
      const finalized = s.items.map(it => { if (it.kind === "assistant" && it.streaming) return { ...it, streaming: false }; if (it.kind === "tool" && it.status === "running") return { ...it, status: "stopped" as const }; return it; });
      const finalItems: Item[] = e.err ? [...finalized, { kind: "notice", id: `e${s.seq}`, level: "warn", text: e.err }] : finalizeStaleTodos(finalized);
      const st = (s.usage?.totalTokens != null && s.usage.totalTokens > 0) ? s.sessionTotal + s.usage.totalTokens : s.sessionTotal;
      // V5.30: 设 perTurnUsage=null 触发 StatsPanel 创建末轮 TurnRecord
      return { ...s, items: finalItems, running: false, turnActive: false, currentAssistant: undefined, lastAssistantIdx: -1, approval: undefined, ask: undefined, perTurnUsage: null, seq: s.seq + 1, sessionTotal: st };
    }
    default: return s;
  }
  return s;
}

export function reducer(s: ControllerState, a: Action): ControllerState {
  switch (a.type) {
    case "user": return { ...s, running: true, turnStartAt: Date.now(), turnTokens: 0, pendingUser: a.text, discardTurn: false, seq: s.seq + 1, items: [...s.items, { kind: "user", id: `u${s.seq}`, text: a.text }] };
    case "unsend": return { ...s, pendingUser: undefined, discardTurn: true, running: false };
    // 本地复位：turn_done 事件丢失/后端无实际任务可取消时，停止按钮必须能
    // 把界面从“执行中”拉回来。逻辑与 turn_done 一致但不触发任何后端调用。
    case "localCancel": {
      const finalized = s.items.map(it => {
        if (it.kind === "assistant" && it.streaming) return { ...it, streaming: false };
        if (it.kind === "tool" && it.status === "running") return { ...it, status: "stopped" as const };
        return it;
      });
      return { ...s, items: finalized, running: false, turnActive: false, currentAssistant: undefined, lastAssistantIdx: -1, approval: undefined, ask: undefined, perTurnUsage: null, seq: s.seq + 1 };
    }
    case "meta": return { ...s, meta: a.meta }; case "context": return { ...s, context: a.context };
    case "balance": return { ...s, balance: a.balance }; case "jobs": return { ...s, jobs: a.jobs }; case "factbase": return { ...s, factBase: a.factBase };
    case "tcca": return { ...s, tcca: a.report };
    case "sessionStats": return { ...s, sessionStats: a.stats };
    case "history": {
      const rebuilt = rebuildHistoryItems(a.messages);
      const items = finalizeStaleTodos(rebuilt.items);
      return { ...s, items, seq: s.seq + items.length, lastAssistantIdx: rebuilt.lastAssistantIdx };
    }
    case "resync": {
      // v4.26 序号防线补拉落库：后端折叠快照校验通过后整体替换 items，补齐
      // Wails 丢件缺口。只补历史——running/turnActive/approval/ask 等 live
      // 状态一律不动（不撤销进行中的回合）；坏形状静默忽略并保底（保留现有
      // items，等下一次缺口在冷却后重试）。刻意不做 finalizeStaleTodos：补拉
      // 常发生在回合中途，全 pending 的待办可能是刚写下的计划，不能擅自收尾
      // （那是 history/turn_done 的收尾语义）。
      if (s.discardTurn) return s; // unsend 未决时不替换（快照可能含已撤回的消息）
      const items = parseResyncItems(a.items);
      if (!items) return s;
      // 快照天然非流式；若本地正在流式输出（缺口期间流未断），把快照最后一个
      // assistant 续上 streaming，避免后续 text 增量按「上一条已终结」劈成新气泡。
      const localLast = s.lastAssistantIdx >= 0 && s.lastAssistantIdx < s.items.length ? s.items[s.lastAssistantIdx] : null;
      const localStreaming = localLast !== null && localLast.kind === "assistant" && localLast.streaming;
      if (localStreaming && s.turnActive && items.length > 0) {
        const lastIdx = items.length - 1;
        const lastIt = items[lastIdx];
        if (lastIt.kind === "assistant") items[lastIdx] = { ...lastIt, streaming: true };
      }
      const lastAssistantIdx = items.reduceRight((acc, it, idx) => acc >= 0 ? acc : it.kind === "assistant" ? idx : -1, -1);
      let next: ControllerState = { ...s, items, lastAssistantIdx, seq: s.seq + items.length };
      // pendingUser 去重：乐观上屏的用户消息若已进快照（后端已落盘），只清标记，
      // 防止下一条事件触发 flushPendingUser 时重复追加同一条用户气泡。
      if (next.pendingUser !== undefined) {
        const last = items.length > 0 ? items[items.length - 1] : null;
        if (last && last.kind === "user" && last.text === next.pendingUser) next = { ...next, pendingUser: undefined };
      }
      return next;
    }
    case "clearApproval": return { ...s, approval: undefined }; case "clearAsk": return { ...s, ask: undefined };
    case "reset": return { ...initialState, meta: s.meta, context: { ...s.context, used: 0 }, balance: s.balance, jobs: s.jobs, seq: s.seq, sessionNonce: s.sessionNonce + 1, _dispatch: s._dispatch };
    case "event": return applyEvent(s, a.e);
    default: return s;
  }
  return s;
}

export const initialState: ControllerState = {
  items: [], running: false, turnActive: false,
  approval: undefined, ask: undefined, usage: undefined,
  context: { used: 0, window: 0 }, meta: undefined, balance: undefined,
  tcca: undefined,
  jobs: [], currentAssistant: undefined, pendingUser: undefined, discardTurn: false, lastAssistantIdx: -1,
  factBase: { facts: [], markdown: "", count: 0, path: "" },
  turnStartAt: 0, turnTokens: 0, seq: 0, sessionTotal: 0, sessionNonce: 0, perTurnUsage: null, turnSteps: [],
  _dispatch: () => {},
};

export const useStore = create<ControllerState>()((set) => ({ ...initialState, _dispatch: (a: Action) => set((s) => reducer(s, a)) } as ControllerState));

// 文件预览：全局 UI 状态，独立于 controller（对话内点击文件路径打开）。
// P1-1 多文件预览队列（调研 2026-08-16）：previewFile 保持兼容（= 当前索引
// 对应文件），新增 previewList/index 支持 ←/→ 切换与位置指示。
interface PreviewState {
  previewFile: string | null;
  previewList: string[];
  previewIndex: number;
  openFilePreview: (rel: string) => void;
  closeFilePreview: () => void;
  /** 在多文件队列中前后切换（dir=1 下一个 / dir=-1 上一个）；越界无操作。 */
  navPreview: (dir: 1 | -1) => void;
  /** 跳到队列指定索引（C7：预览队列 chip 点击）；越界无操作。 */
  navTo: (index: number) => void;
  /** 关闭队列中指定索引的文件（C7：预览队列可点条 × 关闭 / 中键关闭）。
   *  移除后当前索引收敛：删当前项跳到相邻项、删唯一项清空队列关闭预览。 */
  closePreviewAt: (index: number) => void;
}

const PREVIEW_MAX_QUEUE = 50;

export const usePreviewStore = create<PreviewState>()((set, get) => ({
  previewFile: null,
  previewList: [],
  previewIndex: -1,
  openFilePreview: (rel: string) => {
    if (!rel) return;
    const { previewList } = get();
    // 已在队列：移动为当前（不重复入列）
    const existIdx = previewList.indexOf(rel);
    if (existIdx >= 0) {
      set({ previewIndex: existIdx, previewFile: rel });
      return;
    }
    // 新文件：追加到队列末尾（上限裁剪），并设为当前
    const next = [...previewList, rel].slice(-PREVIEW_MAX_QUEUE);
    set({ previewList: next, previewIndex: next.length - 1, previewFile: rel });
  },
  closeFilePreview: () => set({ previewFile: null, previewIndex: -1, previewList: [] }),
  navPreview: (dir: 1 | -1) => {
    const { previewList, previewIndex } = get();
    if (previewList.length === 0 || previewIndex < 0) return;
    const next = Math.max(0, Math.min(previewList.length - 1, previewIndex + dir));
    if (next === previewIndex) return;
    set({ previewIndex: next, previewFile: previewList[next] });
  },
  navTo: (index: number) => {
    const { previewList, previewIndex } = get();
    if (index < 0 || index >= previewList.length || index === previewIndex) return;
    set({ previewIndex: index, previewFile: previewList[index] });
  },
  closePreviewAt: (index: number) => {
    const { previewList, previewIndex } = get();
    if (index < 0 || index >= previewList.length) return;
    const nextList = previewList.filter((_, i) => i !== index);
    if (nextList.length === 0) {
      set({ previewFile: null, previewIndex: -1, previewList: [] });
      return;
    }
    // 删除当前项：跳到相邻项（偏向前一个，头项则下一个）；删除其他项：保持当前。
    let nextIndex = previewIndex;
    if (index === previewIndex) {
      nextIndex = Math.min(index, nextList.length - 1);
    } else if (index < previewIndex) {
      nextIndex = previewIndex - 1;
    }
    set({ previewList: nextList, previewIndex: nextIndex, previewFile: nextList[nextIndex] });
  },
}));

// 已编辑文件标记：docx/xlsx 预览内编辑成功后写入，交付卡片/产物面板据此
// 显示「已更新」徽标（对标"改完即交付"的即时反馈）。
interface UpdatedFilesState {
  updatedAt: Record<string, number>;
  markUpdated: (path: string) => void;
}

export const useUpdatedFilesStore = create<UpdatedFilesState>()((set) => ({
  updatedAt: {},
  markUpdated: (path: string) =>
    set((s) => ({ updatedAt: { ...s.updatedAt, [path]: Date.now() } })),
}));

// 面板 → Composer 的「一键引用」通道：右侧资料概览把 @路径 插入输入框。
interface ComposerInsertState {
  pendingAt: string | null;
  requestAt: (path: string) => void;
  consumeAt: () => string | null;
  pendingText: string | null;
  requestText: (text: string) => void;
  consumeText: () => string | null;
}

export const useComposerInsertStore = create<ComposerInsertState>()((set, get) => ({
  pendingAt: null,
  requestAt: (path: string) => set({ pendingAt: path }),
  consumeAt: () => {
    const p = get().pendingAt;
    set({ pendingAt: null });
    return p;
  },
  pendingText: null,
  requestText: (text: string) => set({ pendingText: text }),
  consumeText: () => {
    const t = get().pendingText;
    set({ pendingText: null });
    return t;
  },
}));

// logBridgeError 记录 bridge 调用失败（bridge 已把后端错误归一为
// BridgeError）到 gaea.log；日志通道自身故障不向上抛（.catch 吞掉），
// 避免掩盖业务错误。T6-1.2 去静默 catch：错误必须可见。
function logBridgeError(where: string, err: unknown): void {
  const e = (err ?? {}) as { code?: unknown; message?: unknown };
  const code = typeof e.code === "string" ? e.code : "BridgeError";
  const message = typeof e.message === "string" ? e.message : String(err);
  const lfe = app.LogFrontendError;
  if (typeof lfe !== "function") return;
  void Promise.resolve(lfe(`[${code}] ${where}: ${message}`)).catch(() => {});
}

// errText 提取用户可读的错误信息（BridgeError/Error/其他值统一字符串化）。
function errText(err: unknown): string {
  return err instanceof Error ? err.message : String(err ?? "未知错误");
}

// failWrite 写路径失败出口（T7-4）：bridge 层 invoke 已把错误记录到 gaea.log，
// 这里再补一条用户可见的 warn notice，保证写/提交/审批失败绝不静默、用户可重试。
function failWrite(dispatch: (a: Action) => void, what: string, err: unknown): void {
  logBridgeError(what, err);
  dispatch({ type: "event", e: { kind: "notice", level: "warn", text: `${what}失败：${errText(err)}，请重试` } });
}

// isFinalAnswerRendered：最终回答是否已完整渲染（T7-4 完整文本比较）。
// 旧实现用「前 120 字前缀是否包含在渲染文本里」判断，流式事件丢尾（后端
// 正文更长、前端只收到前半段）时前缀命中但正文缺失，最终回答依然看不到。
// 新实现要求渲染文本以完整正文结尾才算已渲染：正文为空（纯推理/纯工具轮）
// 视为已渲染，避免误补发。
export function isFinalAnswerRendered(rendered: string, finalContent: string): boolean {
  const full = finalContent.trim();
  if (!full) return true;
  return rendered.trimEnd().endsWith(full);
}

export function useController() {
  const store = useStore;
  const state = store(useShallow(s => s));
  const dispatch = store.getState()._dispatch;

  const loadSessionData = useCallback(async () => {
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
      const history = await app.History();
      if (history && history.length) dispatch({ type: "history", messages: history });
    } catch (err) {
      // 启动期后端未就绪时 Meta/Context/History 可能失败：记录错误，
      // 状态保持默认值，不再静默。
      logBridgeError("loadSessionData", err);
    }
  }, [dispatch]);

  // 最终回答兜底：turn_done（或看门狗检测到后端已停）时拉一次 History，
  // 如果最后一条 assistant 正文没有渲染过，就补发一条 message 事件。
  // 修复末端事件（message/turn_done 密集到达时被 Wails 事件流吞掉）
  // 导致的“最终回答只有重启才可见”。
  const reconcileFinalAnswer = useCallback(() => {
    app.History().then((ms) => {
      const last = [...ms].reverse().find(
        (m) => m.role === "assistant" && typeof m.content === "string" && m.content.trim() !== "",
      );
      if (!last) return;
      const st = store.getState();
      const rendered = st.items
        .filter((i) => i.kind === "assistant" && i.text)
        .map((i) => (i as Extract<Item, { kind: "assistant" }>).text)
        .join("\n");
      // T7-4：前缀启发式（只查前 120 字是否已渲染）改为完整文本比较——
      // 流式丢尾时前缀命中但正文不完整，仍会误判“已渲染”。要求渲染文本
      // 以完整正文结尾才算渲染过，缺失才补发 message。
      if (!isFinalAnswerRendered(rendered, last.content)) {
        dispatch({
          type: "event",
          e: { kind: "message", text: last.content, reasoning: (last as { reasoning?: string }).reasoning ?? "" },
        });
      }
    }).catch((err) => logBridgeError("reconcileFinalAnswer", err));
  }, [store, dispatch]);

  const refreshFactBase = useCallback(() => {
    app.FactBase().then(factBase => dispatch({ type: "factbase", factBase })).catch((err) => logBridgeError("refreshFactBase", err));
  }, [dispatch]);

  useEffect(() => {
    const off = onEvent((e) => {
      // v4.26 事件序号防线：payload 带 seq（可选字段）时做缺口检测，命中缺口
      // 经注入的 fetcher（App.tsx 挂 app.GaeaResyncEvents）补拉后端折叠快照，
      // 以 resync action 落库。旧后端无 seq / 未挂 fetcher 时整条防线静默旁路；
      // 5s 冷却 + 在途去重防补拉风暴。resync 只补 items，不动 running/turnActive
      // （见 reducer case "resync"）。会话切换的 seq 基线归零在各 reset 调用点
      // 经 resetEventSync() 完成。
      noteEventSeq(e, {
        onSnapshot: (snap) => dispatch({ type: "resync", items: snap.items }),
        onError: (err) => logBridgeError("eventSync 补拉", err),
      });
      // 流式 text/reasoning 用 queueMicrotask 确保每次 chunk 即时渲染，
      // 不被 React 18 自动批处理合并。同步 dispatch 会导致多个事件在同一
      // 微任务中批量更新从而不渲染中间态。
      if (e.kind === "text" || e.kind === "reasoning") {
        queueMicrotask(() => dispatch({ type: "event", e }));
      } else {
        dispatch({ type: "event", e });
      }
      if (e.kind === "turn_done") {
        app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("turn_done ContextUsage", err));
        app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("turn_done Balance", err));
        app.TCCAReport().then(raw => {
          try { dispatch({ type: "tcca", report: JSON.parse(raw) as TCCAReport }); }
          catch (err) { logBridgeError("TCCAReport JSON.parse", err); }
        }).catch((err) => logBridgeError("TCCAReport", err));
        reconcileFinalAnswer();
      }
      if (e.kind === "turn_done" || e.kind === "notice") {
        app.Jobs().then(j => dispatch({ type: "jobs", jobs: j })).catch((err) => logBridgeError("Jobs", err));
        refreshFactBase();
      }
      if (e.kind === "tool_result" && e.tool?.name?.startsWith("fact_")) {
        refreshFactBase();
      }
    });
    const offReady = onReady(() => {
      void loadSessionData();
      app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("onReady Balance", err));
      app.Jobs().then(j => dispatch({ type: "jobs", jobs: j })).catch((err) => logBridgeError("onReady Jobs", err));
      refreshFactBase();
      app.TCCAReport().then(raw => {
        try { dispatch({ type: "tcca", report: JSON.parse(raw) as TCCAReport }); }
        catch (err) { logBridgeError("TCCAReport JSON.parse", err); }
      }).catch((err) => logBridgeError("TCCAReport", err));
    });
    // 看门狗：running=true 时每 30s 用后端真实状态校准一次，防止
    // turn_done 事件丢失导致界面永久卡在“执行中”。
    const watchdog = window.setInterval(() => {
      const st = store.getState();
      if (!st.running) return;
      app.GaeaRunning().then((running) => {
        if (!running && store.getState().running) {
          dispatch({ type: "localCancel" });
          reconcileFinalAnswer();
        }
      }).catch((err) => logBridgeError("watchdog GaeaRunning", err));
    }, 30000);
    void loadSessionData();
    app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("init Balance", err));
    app.Jobs().then(j => dispatch({ type: "jobs", jobs: j })).catch((err) => logBridgeError("init Jobs", err));
    refreshFactBase();
    return () => { off(); offReady(); window.clearInterval(watchdog); };
  }, [loadSessionData, refreshFactBase, reconcileFinalAnswer, store, dispatch]);

  // T7-4：send 失败不再静默——保留已上屏的用户消息（可复制重发），
  // 记录 bridge 日志并给出用户可见的失败提示。
  const send = useCallback((displayText: string, submitText = displayText) => {
    dispatch({ type: "user", text: displayText });
    const display = displayText.trim(); const submit = submitText.trim();
    const p = display !== submit ? app.SubmitDisplay(display, submit) : app.Submit(submit);
    p.catch((err) => failWrite(dispatch, "发送消息", err));
  }, [dispatch]);

  // 运行中插话调整（2026-08-28，对齐豆包工作「边跑边改」）：消息注入当前
  // 回合作为补充指引，不打断执行、不开新回合；不落用户气泡（后端以 notice
  // 回显）。未运行时后端 Steer 内部兜底走 Submit 排队。
  const steer = useCallback((text: string) => {
    const t = text.trim();
    if (!t) return;
    app.Steer(t).catch((err) => failWrite(dispatch, "插话调整", err));
  }, [dispatch]);

  const cancel = useCallback((): string | undefined => {
    const cur = store.getState();
    const onFail = (err: unknown) => failWrite(dispatch, "取消", err);
    if (cur.running && cur.pendingUser !== undefined) { const text = cur.pendingUser; dispatch({ type: "unsend" }); app.Cancel().catch(onFail); return text; }
    if (cur.running) dispatch({ type: "localCancel" }); // 事件丢失时仍能复位本地运行态
    app.Cancel().catch(onFail); return undefined;
  }, [store, dispatch]);

  // T7-4：approve/answerQuestion 失败时不清掉弹窗（保留审批/提问界面），
  // 记录日志并提示用户重试；成功后才清除。
  const approve = useCallback((id: string, decision: "allow_once" | "allow_session" | "persist_allow" | "deny" | "abort") => {
    app.Approve(id, decision)
      .then(() => dispatch({ type: "clearApproval" }))
      .catch((err) => failWrite(dispatch, "审批提交", err));
  }, [dispatch]);
  const answerQuestion = useCallback((id: string, answers: QuestionAnswer[]) => {
    app.AnswerQuestion(id, answers)
      .then(() => dispatch({ type: "clearAsk" }))
      .catch((err) => failWrite(dispatch, "回答提交", err));
  }, [dispatch]);
  const setPermLevel = useCallback((level: string) => { app.SetPermLevel(level).catch((err) => failWrite(dispatch, "切换权限级别", err)); }, [dispatch]);
  const newSession = useCallback(async () => {
    try {
      await app.NewSession();
      dispatch({ type: "reset" });
      resetEventSync(); // v4.26：新会话事件 seq 从 1 重新单调递增，补拉防线基线归零
      refreshFactBase();
    } catch (err) {
      // 新建失败不重置界面（后端会话未切换），给出可见提示。
      failWrite(dispatch, "新建会话", err);
    }
  }, [dispatch, refreshFactBase]);
  const listSessions = useCallback((): Promise<SessionMeta[]> =>
    app.ListSessions().catch((err) => { logBridgeError("listSessions", err); return [] as SessionMeta[]; }), []);
  const listProjectSessions = useCallback((): Promise<ProjectGroup[]> =>
    app.ListProjectSessions().catch((err) => { logBridgeError("listProjectSessions", err); return [] as ProjectGroup[]; }), []);
  // fetchSessionStats 拉取会话级派生统计并写入 store；失败/无日志标记为不可用
  // （不阻塞恢复流程，仅影响统计面板的历史成本展示）。
  const fetchSessionStats = useCallback((path: string) => {
    app.SessionStats(path)
      .then((stats) => dispatch({ type: "sessionStats", stats }))
      .catch((err) => { logBridgeError("fetchSessionStats", err); dispatch({ type: "sessionStats", stats: undefined }); });
  }, [dispatch]);
  const resumeSession = useCallback(async (path: string) => {
    const ms = await app.ResumeSession(path).catch((e: unknown) => {
      // 恢复失败不要静默清空：给用户明确提示
      dispatch({
        type: "event",
        e: { kind: "notice", level: "warn", text: `恢复会话失败：${e instanceof Error ? e.message : String(e)}` },
      });
      return [] as HistoryMessage[];
    });
    dispatch({ type: "reset" });
    resetEventSync(); // v4.26：恢复会话同样归零 seq 基线
    if (ms.length) dispatch({ type: "history", messages: ms });
    // 恢复后回填会话级派生统计（成本/用量历史，评审缺陷 11 根治）
    void fetchSessionStats(path);
    app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("resumeSession ContextUsage", err));
    refreshFactBase();
  }, [dispatch, refreshFactBase, fetchSessionStats]);
  const archiveSession = useCallback((path: string) => app.ArchiveSession(path).catch((err) => failWrite(dispatch, "归档会话", err)), [dispatch]);
  const unarchiveSession = useCallback((path: string): Promise<string> => app.UnarchiveSession(path).catch((err) => { failWrite(dispatch, "取消归档", err); return ""; }), [dispatch]);
  const pinSession = useCallback((path: string, pinned: boolean) => app.PinSession(path, pinned).catch((err) => failWrite(dispatch, "更新固定状态", err)), [dispatch]);
  const deleteSession = useCallback((path: string) => app.DeleteSession(path).catch((err) => failWrite(dispatch, "删除会话", err)), [dispatch]);
  const renameSession = useCallback((path: string, title: string) => app.RenameSession(path, title).catch((err) => failWrite(dispatch, "重命名会话", err)), [dispatch]);
  const refreshMeta = useCallback(async () => {
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
    } catch (err) { logBridgeError("refreshMeta", err); }
  }, [dispatch]);
  const pickWorkspace = useCallback(async (): Promise<string> => {
    const p = await app.PickWorkspace().catch((err: unknown) => { failWrite(dispatch, "打开工作区", err); return ""; });
    if (p) {
      dispatch({ type: "reset" }); resetEventSync(); refreshFactBase();
      try {
        dispatch({ type: "meta", meta: await app.Meta() });
        dispatch({ type: "context", context: await app.ContextUsage() });
      } catch (err) { logBridgeError("pickWorkspace refresh", err); }
    }
    return p;
  }, [dispatch, refreshFactBase]);
  const switchWorkspace = useCallback(async (path: string): Promise<string> => {
    const n = await app.SwitchWorkspace(path).catch((err: unknown) => { failWrite(dispatch, "切换工作区", err); return ""; });
    if (n) {
      dispatch({ type: "reset" }); resetEventSync(); refreshFactBase();
      try {
        dispatch({ type: "meta", meta: await app.Meta() });
        dispatch({ type: "context", context: await app.ContextUsage() });
      } catch (err) { logBridgeError("switchWorkspace refresh", err); }
    }
    return n;
  }, [dispatch, refreshFactBase]);
  const compact = useCallback(() => { app.Compact().catch((err) => failWrite(dispatch, "压缩上下文", err)); }, [dispatch]);
  const setModel = useCallback(async (name: string) => {
    await app.SetModel(name).catch((err) => failWrite(dispatch, "切换模型", err));
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
    } catch (err) { logBridgeError("setModel refresh", err); }
  }, [dispatch]);
  const fetchMemory = useCallback((): Promise<MemoryView> => app.Memory().catch((err) => {
    logBridgeError("fetchMemory", err);
    return { docs: [], facts: [], scopes: [], storeDir: "", available: false } as MemoryView;
  }), []);
  const remember = useCallback(async (scope: string, note: string) => { await app.Remember(scope, note).catch((err) => failWrite(dispatch, "保存记忆", err)); }, [dispatch]);
  const forget = useCallback(async (name: string) => { await app.Forget(name).catch((err) => failWrite(dispatch, "删除记忆", err)); }, [dispatch]);
  const saveDoc = useCallback(async (path: string, body: string) => { await app.SaveDoc(path, body).catch((err) => failWrite(dispatch, "保存文档", err)); }, [dispatch]);
  const updateFact = useCallback(async (name: string, body: string) => { await app.UpdateFact(name, body).catch((err) => failWrite(dispatch, "更新画像", err)); }, [dispatch]);
  const changeFactType = useCallback(async (name: string, typ: string) => { await app.ChangeFactType(name, typ).catch((err) => failWrite(dispatch, "修改画像类型", err)); }, [dispatch]);
  const clearFactBase = useCallback(async () => {
    await app.FactBaseClear().catch((err) => failWrite(dispatch, "清空事实库", err));
    refreshFactBase();
  }, [dispatch, refreshFactBase]);
  const promoteFactBase = useCallback(async (): Promise<number> => {
    const n = await app.FactBasePromote().catch((err) => { failWrite(dispatch, "写入永久记忆", err); return 0; });
    return n;
  }, [dispatch]);
  const rewind = useCallback(async (turn: number, scope: string) => {
    // T7-4：回退失败不再静默，且不触发 reset——保留当前对话现场（否则刚
    // 弹的失败提示会被 reset 清空，用户连发生了什么都看不到）。
    const act = (p: Promise<unknown>): Promise<boolean> =>
      p.then(() => true).catch((err) => { failWrite(dispatch, "回退对话", err); return false; });
    let ok = false;
    if (scope === "fork") ok = await act(app.Fork(turn));
    else if (scope === "summ-from") ok = await act(app.SummarizeFrom(turn));
    else if (scope === "summ-upto") ok = await act(app.SummarizeUpTo(turn));
    else ok = await act(app.Rewind(turn, scope));
    if (!ok) return;
    const ms = await app.History().catch((err) => { logBridgeError("rewind History", err); return [] as HistoryMessage[]; });
    dispatch({ type: "reset" });
    resetEventSync(); // v4.26：回退重载历史，seq 基线归零
    if (ms.length) dispatch({ type: "history", messages: ms });
    app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("rewind ContextUsage", err));
  }, [dispatch]);

  return { state, send, steer, cancel, approve, answerQuestion, setPermLevel, newSession, listSessions, listProjectSessions, resumeSession, archiveSession, unarchiveSession, pinSession, deleteSession, renameSession, refreshMeta, pickWorkspace, switchWorkspace, compact, rewind, setModel, fetchMemory, remember, forget, saveDoc, updateFact, changeFactType, clearFactBase, promoteFactBase, fetchSessionStats };
}

// useItems 订阅 items 数组，与 useController 分离。
// 流式输出时 items 高频变化（每次 text/reasoning 事件），通过独立 hook 避免
// useController 的 store(s=>s) 全量订阅导致 App 树全局重渲染。
// 使用 useShallow 做浅比较：仅当 items 长度或元素引用变化时才触发重渲染，
// 非 items 字段（meta/context/balance 等）的变化不会影响此 hook。
export function useItems(): Item[] {
  return useStore(s => s.items);
}

// useTurnStartAt 返回当前回合开始时间戳(ms)，用于计算思考耗时。
export function useTurnStartAt(): number {
  return useStore(s => s.turnStartAt);
}
