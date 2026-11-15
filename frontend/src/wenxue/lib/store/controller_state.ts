// controller_state.ts — controller 状态机纯函数层（批 33 FE3-11 拆分自
// controller.ts，原位搬移零逻辑改动）：Item/ControllerState/Action 类型、
// item id 取号与 slot 口径（GenUI 审计 §8-5）、历史重建/resync/resume 快照
// 解析、applyEvent 事件应用与 reducer、initialState。React 编排层
// （useStore/ensureEventsBound/useController/logBridgeError 族）见
// controller.ts——该文件以 `export *` 原路 re-export 本层，外部导入路径不变。
import { parseTodos } from "../tools";
import type {
  BalanceInfo, ContextInfo, FactBaseView, HistoryMessage, JobView,
  Meta, SessionStatsView, WireApproval, WireAsk,
  WireEvent, WireUsage,
} from "../types";

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

export type Action =
  | { type: "event"; e: WireEvent } | { type: "user"; text: string } | { type: "unsend" }
  | { type: "localCancel" }
  | { type: "meta"; meta: Meta } | { type: "context"; context: ContextInfo }
  | { type: "balance"; balance: BalanceInfo } | { type: "jobs"; jobs: JobView[] } | { type: "factbase"; factBase: FactBaseView }
  | { type: "sessionStats"; stats?: SessionStatsView }
  | { type: "history"; messages: HistoryMessage[] }
  // v4.26 序号防线补拉：items 为后端 GaeaResyncEvents 折叠快照（原始 JSON，
  // 由 reducer 内 parseResyncItems 校验后落库）。
  | { type: "resync"; items: unknown[] }
  | { type: "clearApproval" } | { type: "clearAsk" } | { type: "reset" };


// ── item id 取号（GenUI 审计 §8-5 slot 口径统一）────────────────
// 对话项 id 有三条生成路径，genui 交互状态 key（genuiStateKey 的 slot 段=
// assistant 条目 id）要求「同一条 assistant 消息」三路同 id 才能跨重启命中：
//   - 实时：事件 payload 可选字段 logSeq（=磁盘日志行 seq，EventLogSink 落盘
//     时回填；后端落地前恒缺省 → 行为与旧计数器口径逐字节一致）；
//   - resume / 补拉：后端折叠器统一以日志序取号（u/a/n/sa<seq>，见
//     internal/app/gaea_resync.go resyncID）。
// 日志 seq 是唯一跨重启稳定的会话内身份。本地计数器（s.seq）与日志 seq 是
// 两个独立数字空间，resume/补拉落库的日志序 id 与后续本地新建 id 会互相穿越
// ——counterSlot 从 s.seq 起跳过已占用后缀，保证新建 id 不撞已有条目（React
// key 唯一 + genui 槽位不串状态）。

// eventLogSeq 提取事件 payload 的可选日志序：非有限非负数视为缺省（旧后端）。
function eventLogSeq(e: WireEvent): number | null {
  const raw = (e as { logSeq?: unknown }).logSeq;
  if (typeof raw !== "number" || !Number.isFinite(raw) || raw < 0) return null;
  return Math.floor(raw);
}

// idTaken 判断 id 是否已被现有条目占用（logSeq 命中已占用时回退计数器）。
function idTaken(items: Item[], id: string): boolean {
  return items.some((it) => it.id === id);
}

// counterSlot 返回 prefix+start 起、第一个未被现有条目占用的数字后缀。
// 只比对同前缀 id（不同前缀天然不相交；sa 不被 a 前缀误吸——startsWith 边界）。
function counterSlot(items: Item[], prefix: string, start: number): number {
  const used = new Set<number>();
  for (const it of items) {
    if (!it.id.startsWith(prefix)) continue;
    const suffix = it.id.slice(prefix.length);
    if (suffix === "") continue;
    const n = Number(suffix);
    if (Number.isInteger(n) && n >= 0) used.add(n);
  }
  let n = Math.max(0, Math.floor(start));
  while (used.has(n)) n++;
  return n;
}

// assistantSlot 为新建 assistant 条目取 id：logSeq 可用且未被占用 → 日志序
// （与 resume/补拉折叠快照同口径）；否则回退本地计数器。subagent 答复气泡与
// 折叠器同前缀 sa（foldResyncItems 的 subagent_message 条目）。返回 seq 为
// 落库后的计数器值：日志序取号不消耗计数器（原值保留），计数器取号跳到 n+1。
function assistantSlot(s: ControllerState, e: WireEvent, subagent: boolean): { id: string; seq: number } {
  const prefix = subagent ? "sa" : "a";
  const lseq = eventLogSeq(e);
  if (lseq !== null && !idTaken(s.items, `${prefix}${lseq}`)) {
    return { id: `${prefix}${lseq}`, seq: s.seq };
  }
  const n = counterSlot(s.items, prefix, s.seq);
  return { id: `${prefix}${n}`, seq: n + 1 };
}

export function flushPendingUser(s: ControllerState): ControllerState {
  if (s.pendingUser === undefined) return s;
  // 如果消息已通过 user action 立即加入 items，只清除 pendingUser 避免重复
  const last = s.items.length > 0 ? s.items[s.items.length - 1] : null;
  if (last && last.kind === "user" && last.text === s.pendingUser) {
    return { ...s, pendingUser: undefined };
  }
  const n = counterSlot(s.items, "u", s.seq);
  return { ...s, seq: n + 1, items: [...s.items, { kind: "user", id: `u${n}`, text: s.pendingUser }], pendingUser: undefined };
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

// resumeSnapshotItems 校验「恢复会话」用的日志折叠快照并做恢复语义收尾
//（GenUI 审计 §8-5 slot 口径统一）：形状坏/空返回 null（调用方回退
// History+rebuildHistoryItems 的 h<index> 口径，legacy 会话保底）；快照里
// status=running 的工具卡收尾为 stopped——重启后没有任何在跑的工具，与
// rebuildHistoryItems 的 stopped 语义对齐（补拉场景刻意不做此收尾：回合
// 可能真在跑，见 reducer case "resync"）。返回规范化后的数组，reducer 内
// parseResyncItems 会二次校验（幂等）。
export function resumeSnapshotItems(raw: unknown): unknown[] | null {
  // 空数组=空会话（合法，返回 [] 让调用方短路，不必回退 History）；
  // parseResyncItems 对空数组返回 null 是补拉场景的「无有效内容」口径，二者有意区分。
  if (Array.isArray(raw) && raw.length === 0) return [];
  const items = parseResyncItems(raw);
  if (items === null) return null;
  return items.map((it) =>
    it.kind === "tool" && it.status === "running" ? { ...it, status: "stopped" as const } : it,
  );
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
      // 没有可追加的活跃 assistant 项时创建新的（id 取号见 assistantSlot：
      // 优先事件日志序，回退本地计数器）
      const slot = assistantSlot(s, e, false);
      const newIdx = s.items.length;
      return { ...s, seq: slot.seq, items: [...s.items, { kind: "assistant", id: slot.id, text: e.kind === "text" ? delta : "", reasoning: e.kind === "reasoning" ? delta : "", streaming: true }], currentAssistant: slot.id, lastAssistantIdx: newIdx };
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
      // 没有任何可更新的 assistant 项时创建新的（首轮且模型直接回了 message；
      // 子代理答复气泡与折叠快照同前缀 sa，id 取号见 assistantSlot）
      const slot = assistantSlot(s, e, !!e.subagentRef);
      const newIdx = s.items.length;
      return { ...s, seq: slot.seq, items: [...s.items, { kind: "assistant", id: slot.id, text: e.text ?? "", reasoning: e.reasoning ?? "", streaming: false, subagentRef: e.subagentRef }], currentAssistant: undefined, lastAssistantIdx: newIdx };
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
  jobs: [], currentAssistant: undefined, pendingUser: undefined, discardTurn: false, lastAssistantIdx: -1,
  factBase: { facts: [], markdown: "", count: 0, path: "" },
  turnStartAt: 0, turnTokens: 0, seq: 0, sessionTotal: 0, sessionNonce: 0, perTurnUsage: null, turnSteps: [],
  _dispatch: () => {},
};
