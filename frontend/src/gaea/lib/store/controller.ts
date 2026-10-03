// Zustand store — 主 controller reducer 状态机（useStore/applyEvent/reducer/useController）。
// P4 结构刀：由原 lib/store.ts 按 store 域拆分而来，本文件为 controller 域，导出名逐字节保留。


import { useCallback, useEffect } from "react";
import { invalidateTurnCaches } from "../deliverablesTurn";
import { create } from "zustand";
import { useShallow } from "zustand/shallow";
import { app, onEvent, onReady } from "../bridge";
import { noteEventSeq, resetEventSync } from "../eventSync";
import { errText, failWrite, logBridgeError, useMemoryActions } from "./controller_actions_memory";
import { parseTodos } from "../tools";
import type {
  BalanceInfo, ContextInfo, FactBaseView, HistoryMessage, JobView,
  Meta, ProjectGroup, QuestionAnswer, SessionMeta, SessionStatsView, WireApproval, WireAsk,
  WireEvent, WireUsage,
} from "../types";
export * from "./controller_state";
import { applyEvent, flushPendingUser, initialState, parseResyncItems, rebuildHistoryItems, reducer, resumeSnapshotItems } from "./controller_state";
import type { Action, ControllerState, Item } from "./controller_state";
export const useStore = create<ControllerState>()((set) => ({ ...initialState, _dispatch: (a: Action) => set((s) => reducer(s, a)) } as ControllerState));

// logBridgeError 记录 bridge 调用失败（bridge 已把后端错误归一为
// BridgeError）到 gaea.log；日志通道自身故障不向上抛（.catch 吞掉），
// 旧实现用「前 120 字前缀是否包含在渲染文本里」判断，流式事件丢尾（后端
// 正文更长、前端只收到前半段）时前缀命中但正文缺失，最终回答依然看不到。
// 新实现要求渲染文本以完整正文结尾才算已渲染：正文为空（纯推理/纯工具轮）
// 视为已渲染，避免误补发。
export function isFinalAnswerRendered(rendered: string, finalContent: string): boolean {
  const full = finalContent.trim();
  if (!full) return true;
  return rendered.trimEnd().endsWith(full);
}

// ── 全局事件/就绪绑定（恰好一次，v4.119 刀10）────────────────────────
// 进度计划板块的 AI 对话 pane 与办公板块 GaeaApp 共享本模块的会话 store
// （keepAlive 树下两者可能同时挂载）。事件绑定必须**恰好一次**：多订阅会让
// text/reasoning 增量重复 dispatch（气泡翻倍）。首个 useController 挂载时绑定，
// 之后任何宿主复用；卸载不退订——隐藏期间事件继续入库，回到板块即热态。
// 注入的回调都是 store 单例上的稳定闭包，跨宿主语义一致。
interface EventBindDeps {
  loadSessionData: () => Promise<void>;
  refreshFactBase: () => void;
  reconcileFinalAnswer: () => void;
  store: typeof useStore;
  dispatch: (a: Action) => void;
}
let eventsBound = false;
function ensureEventsBound(deps: EventBindDeps): void {
  if (eventsBound) return;
  eventsBound = true;
  const { dispatch, refreshFactBase, reconcileFinalAnswer } = deps;
  onEvent((e) => {
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
      // v4.382 失效接线（v4.354 记录的 dirListingsCache 欠账）：轮完成后
      // 登记与目录探测缓存整包失效——新轮交付物/新建文件不再被 TTL 内的
      // 旧探测误标缺失。invalidateTurnCaches 纯内存清空，零桥接成本。
      invalidateTurnCaches();
      app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("turn_done ContextUsage", err));
      app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("turn_done Balance", err));
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
  onReady(() => {
    void deps.loadSessionData();
    app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("onReady Balance", err));
    app.Jobs().then(j => dispatch({ type: "jobs", jobs: j })).catch((err) => logBridgeError("onReady Jobs", err));
    refreshFactBase();
  });
  void deps.loadSessionData();
  app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("init Balance", err));
  app.Jobs().then(j => dispatch({ type: "jobs", jobs: j })).catch((err) => logBridgeError("init Jobs", err));
  refreshFactBase();
}

export function useController() {
  const store = useStore;
  const state = store(useShallow(s => s));
  const dispatch = store.getState()._dispatch;

  // loadItemsFoldedFirst 对话项装载统一入口（GenUI 审计 §8-5 slot 口径统一）：
  // GaeaResyncEvents(0) 恒返回会话全量折叠快照（id=日志序 resyncID），resume/
  // 回退/启动三路同 id → genui 交互状态跨重启可命中；快照不可用（旧后端异常/
  // 形状坏 null/空会话）回退 History+rebuildHistoryItems（h<index> legacy 保底）。
  // 恒 resolve 不抛：装载失败不阻断会话流（错误走 logBridgeError 记录）。
  const loadItemsFoldedFirst = useCallback(async (): Promise<void> => {
    try {
      const snap = await app.ResyncEvents(0);
      const items = resumeSnapshotItems(snap?.items);
      if (items) {
        if (items.length) dispatch({ type: "resync", items });
        return;
      }
    } catch (err) {
      logBridgeError("loadItemsFoldedFirst ResyncEvents", err);
    }
    const ms = await app.History().catch((err: unknown) => {
      logBridgeError("loadItemsFoldedFirst History", err);
      return [] as HistoryMessage[];
    });
    if (ms.length) dispatch({ type: "history", messages: ms });
  }, [dispatch]);

  const loadSessionData = useCallback(async () => {
    // v4.382 失效接线：会话切换/恢复即清空登记与目录探测缓存（缓存随会话走）。
    invalidateTurnCaches();
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
      await loadItemsFoldedFirst();
    } catch (err) {
      // 启动期后端未就绪时 Meta/Context/History 可能失败：记录错误，
      // 状态保持默认值，不再静默。
      logBridgeError("loadSessionData", err);
    }
  }, [dispatch, loadItemsFoldedFirst]);

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
    // v4.119 刀10：事件/就绪绑定提升为模块级恰好一次（见 ensureEventsBound）——
    // 进度计划板块的 AI 对话 pane 与本板块 GaeaApp 在 keepAlive 树下可能同时
    // 挂载，多订阅会让 text/reasoning 增量重复 dispatch（气泡翻倍）。
    ensureEventsBound({ loadSessionData, refreshFactBase, reconcileFinalAnswer, store, dispatch });
    // 看门狗：running=true 时每 30s 用后端真实状态校准一次，防止
    // turn_done 事件丢失导致界面永久卡在“执行中”。（每宿主各自运行：
    // 只读校准 + localCancel 幂等，多宿主共存无害。）
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
    return () => { window.clearInterval(watchdog); };
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
    await app.ResumeSession(path).catch((e: unknown) => {
      // 恢复失败不要静默清空：给用户明确提示
      dispatch({
        type: "event",
        e: { kind: "notice", level: "warn", text: `恢复会话失败：${e instanceof Error ? e.message : String(e)}` },
      });
      return [] as HistoryMessage[];
    });
    dispatch({ type: "reset" });
    resetEventSync(); // v4.26：恢复会话同样归零 seq 基线
    await loadItemsFoldedFirst(); // §8-5：折叠快照优先（日志序 id），History 保底
    // 恢复后回填会话级派生统计（成本/用量历史，评审缺陷 11 根治）
    void fetchSessionStats(path);
    app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("resumeSession ContextUsage", err));
    refreshFactBase();
  }, [dispatch, loadItemsFoldedFirst, refreshFactBase, fetchSessionStats]);
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
  // FE3-04：Compact 是 mock-only 绑定（Go 侧无对应绑定，drift.ts MOCK_ONLY_NAMES
  // 单源），真机上下文压缩由后端会话事件自动执行、无手动入口。真机调用现在从
  // proxy 拿到显式 BridgeError（MockOnlyBinding），失败**不可重试**——不走
  // failWrite（其固定追加「请重试」会误导），诚实提示一次性原因。dev mock 下
  // Compact 正常 resolve，本 catch 不触发，行为不变。
  const compact = useCallback(() => {
    app.Compact().catch((err) => {
      logBridgeError("compact", err);
      dispatch({ type: "event", e: { kind: "notice", level: "warn", text: `压缩上下文失败：${errText(err)}` } });
    });
  }, [dispatch]);
  const setModel = useCallback(async (name: string) => {
    await app.SetModel(name).catch((err) => failWrite(dispatch, "切换模型", err));
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
    } catch (err) { logBridgeError("setModel refresh", err); }
  }, [dispatch]);
  const { fetchMemory, remember, forget, saveDoc, updateFact, changeFactType, clearFactBase, promoteFactBase } = useMemoryActions(dispatch, refreshFactBase);
  const rewind = useCallback(async (turn: number, scope: string): Promise<boolean> => {
    // T7-4：回退失败不再静默，且不触发 reset——保留当前对话现场（否则刚
    // 弹的失败提示会被 reset 清空，用户连发生了什么都看不到）。
    // 返回是否成功（v4.232：regenerate 编排需要据此决定是否重发）。
    const act = (p: Promise<unknown>): Promise<boolean> =>
      p.then(() => true).catch((err) => { failWrite(dispatch, "回退对话", err); return false; });
    let ok = false;
    if (scope === "fork") ok = await act(app.Fork(turn));
    else if (scope === "summ-from") ok = await act(app.SummarizeFrom(turn));
    else if (scope === "summ-upto") ok = await act(app.SummarizeUpTo(turn));
    else ok = await act(app.Rewind(turn, scope));
    if (!ok) return false;
    dispatch({ type: "reset" });
    resetEventSync(); // v4.26：回退重载历史，seq 基线归零
    await loadItemsFoldedFirst(); // §8-5：折叠快照优先（日志序 id），History 保底
    app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("rewind ContextUsage", err));
    return true;
  }, [dispatch, loadItemsFoldedFirst]);

  // 重新生成（v4.232，对齐同类产品 ChatGPT 式 regenerate）：丢弃最后一轮
  // （含用户消息），用同一文本原样重发。复用 GaeaRewind+GaeaSend 既有原语
  // （零新绑定），编排层保证「先截断成功、再重发」，失败不动现场。
  // 只对最后一轮开放：更早的轮次要先生成中间轮才有意义，语义上交给回退。
  const regenerate = useCallback(async (turn: number): Promise<boolean> => {
    const cur = store.getState();
    if (cur.running || cur.pendingUser !== undefined) return false;
    const users = cur.items.filter((i): i is Extract<Item, { kind: "user" }> => i.kind === "user");
    if (turn < 0 || turn !== users.length - 1) return false;
    const target = users[turn];
    if (!target || !target.text.trim()) return false;
    const ok = await rewind(turn, "conversation");
    if (!ok) return false;
    send(target.text);
    return true;
  }, [store, rewind, send]);

  return { state, send, steer, cancel, approve, answerQuestion, setPermLevel, newSession, listSessions, listProjectSessions, resumeSession, archiveSession, unarchiveSession, pinSession, deleteSession, renameSession, refreshMeta, pickWorkspace, switchWorkspace, compact, rewind, setModel, fetchMemory, remember, forget, saveDoc, updateFact, changeFactType, clearFactBase, promoteFactBase, fetchSessionStats, regenerate };
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
