import { useCallback, useEffect, useRef, useState } from "react";
import { ArrowUp, Loader2, Rollback, Save, X } from "../icons";
import { app, onEvent, onSubagentText } from "../lib/bridge";
import { useT, type Translator } from "../lib/i18n";
import { useToast } from "./Toast";
import type { SubagentTranscriptView } from "../lib/types";
import { buildRenderItems, toolStatus, unwrapEnvelopeText } from "../lib/subagentRender";
import { AssistantMessage } from "./Message";
import { ToolCard } from "./ToolCard";
import type { Item } from "../lib/store";

type ToolItem = Extract<Item, { kind: "tool" }>;
import { usePollingGate } from "../../hooks/usePollingGate";
import { useLiveReload } from "../hooks/useLiveReload";

// SubagentThread — 子代理对话全面板视图（v4.27 对齐 Codex：点击子代理 →
// 右侧面板打开其对话，运行中实时刷新）。
//
// 此前子代理 transcript 只能经 AgentTree 内嵌窄小卡（10px 字号、max-h-64）
// 手动点「查看完整 transcript」读取一次；本组件把对话提到面板级：
//  - 头部：返回分工 + 任务标题 + 状态徽标 + 模型 + 消息数 + 手动刷新；
//  - 消息流：v4.63 起与主对话完全同款——assistant 正文/思考走
//    AssistantMessage（Markdown/折叠/复制/交付卡同源），tool 调用与结果
//    按 toolCallId 配对成主对话 ToolCard（可读摘要/状态/折叠输出），
//    system 弱化单行 / user 右对齐；运行中自动跟随底部；
//  - 实时：running 时每 3s 轮询（页面不可见门控空转）+ 事件驱动刷新
//    （turn_done 立即、运行中事件节流，useLiveReload 同看板语义）；
//    v4.62 P1：运行中子代理的助手文本增量经 subagent_text 事件流式实时
//    渲染（缓冲行 + 快照接管 reconcile，见 streamBuf 注释），不再等快照。

export type SubagentThreadStatus = "running" | "completed" | "failed";

const THREAD_POLL_MS = 3000;

function statusMeta(status: SubagentThreadStatus, t: Translator): { label: string; color: string; bg: string; border: string } {
  switch (status) {
    case "running":
      return {
        label: t("subagent.statusRunning"),
        color: "var(--gaea-glow)",
        bg: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
        border: "1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)",
      };
    case "failed":
      return {
        label: t("subagent.statusFailed"),
        color: "var(--md-sys-color-destructive)",
        bg: "color-mix(in srgb, var(--md-sys-color-destructive) 10%, transparent)",
        border: "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 30%, transparent)",
      };
    default:
      return {
        label: t("subagent.statusDone"),
        color: "var(--md-sys-color-success)",
        bg: "color-mix(in srgb, var(--md-sys-color-success) 10%, transparent)",
        border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 30%, transparent)",
      };
  }
}

// ── Codex 式渲染（v4.63）：与主对话同一套组件 ────────────────────
//
// transcript 消息 → 渲染项：assistant 的 toolCalls 与后续 tool 结果按
// toolCallId 配对成完整 ToolItem（主对话 ToolCard 同款：可读摘要/状态/
// 折叠输出），正文与思考走主对话 AssistantMessage（Markdown/思考折叠/
// 复制同款）。未配对的孤儿 tool 行降级为独立卡（name+输出，诚实展示）。

// ── mt_/长文本有界输出（v4.63 Codex 式）──────────────────────────
// 本地模型工具（summarize_file/vision）的输出是文档级长文本，快照渲染
// 全量铺开是一堵墙。默认限高内部滚动（Markdown 照常渲染），底部给
// 「展开全部/收起」与字数标注；流式实时行不做有界（跟随滚动语义）。
const OUTPUT_BOUND_CHARS = 4000;
const OUTPUT_BOUND_MAX_H = "26rem";

function BoundedAssistantMessage({ item, chars }: {
  item: { kind: "assistant"; id: string; text: string; reasoning: string; streaming: boolean };
  chars: number;
}) {
  const t = useT();
  const [full, setFull] = useState(false);
  return (
    <div data-testid="agent-thread-bounded" className="rounded-md border" style={{ borderColor: "var(--md-sys-color-outline-variant)" }}>
      <div
        className="px-2.5 py-2 overflow-y-auto"
        style={full ? undefined : { maxHeight: OUTPUT_BOUND_MAX_H }}
      >
        <AssistantMessage item={item} deliverTail={false} />
      </div>
      <div className="flex items-center gap-2 px-2.5 py-1 border-t" style={{ borderColor: "var(--md-sys-color-outline-variant)" }}>
        <span className="font-mono text-[9.5px] tabular-nums" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {t("subagent.outputChars", { n: chars })}
        </span>
        <span className="min-w-0 flex-1" />
        <button
          type="button"
          data-testid="agent-thread-bounded-toggle"
          className="cursor-pointer rounded border-0 bg-transparent px-1.5 py-0.5 text-[10px] transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--gaea-glow)" }}
          onClick={() => setFull((v) => !v)}
        >
          {full ? t("subagent.outputCollapse") : t("subagent.outputExpand", { n: chars })}
        </button>
      </div>
    </div>
  );
}

export function SubagentThread({
  sessionPath,
  target,
  task,
  status,
  model,
  onBack,
}: {
  sessionPath: string;
  target: string;
  task?: string;
  status: SubagentThreadStatus;
  model?: string;
  onBack: () => void;
}) {
  const t = useT();
  const [transcript, setTranscript] = useState<SubagentTranscriptView | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const gate = usePollingGate();
  // P1 逐 token 流式（v4.62）：后端把运行中子代理的助手文本增量经
  // subagent_text 事件打到前端（wire-only，不落主会话日志）。缓冲作为
  // 「正在打出的字」的实时行渲染在消息流尾部；~1s 快照轮询拿到权威数据后
  // 按下述口径接管（见 load 内 reconcile），SubagentMessage 收尾兜底清空。
  const [streamBuf, setStreamBuf] = useState("");
  // 缓冲起点时的消息数基线：快照出现比基线新的消息 → 该段已落 transcript，
  // 缓冲让位（防第二轮文本接在第一轮尾部）。
  const bufBaseCountRef = useRef(0);
  // transcript 的镜像 ref：事件回调里读最新消息数基线用（state 闭包会过期）。
  const transcriptRef = useRef<SubagentTranscriptView | null>(null);

  const load = useCallback(() => {
    if (!sessionPath || !target) return;
    app
      .SubagentTranscript(sessionPath, target)
      .then((v) => {
        setTranscript(v);
        transcriptRef.current = v;
        setFailed(false);
        // v4.66：meta 侧车的最近追问失败摘要随快照带出（等待态气泡在场时
        // 由下方 effect 转失败态；无等待气泡时仅存镜像，不骚扰输入框）。
        setFollowUpRemoteErr(v.followUpError ?? "");
        // 流式缓冲 ↔ 权威快照接管：快照尾条已含缓冲开头（快照追上流），
        // 或快照长过了缓冲基线（新消息已落盘）→ 清缓冲，交给权威渲染。
        setStreamBuf((prev) => {
          if (!prev) return prev;
          const msgs = v.messages ?? [];
          const lastContent = msgs.length > 0 ? msgs[msgs.length - 1].content ?? "" : "";
          const head = prev.trimStart().slice(0, 20);
          if ((head !== "" && lastContent.includes(head)) || msgs.length > bufBaseCountRef.current) {
            return "";
          }
          return prev;
        });
      })
      .catch(() => setFailed(true))
      .finally(() => setLoading(false));
  }, [sessionPath, target]);

  // 首次加载 / 切换子代理（target 变化）重新拉取
  useEffect(() => {
    setLoading(true);
    setTranscript(null);
    load();
  }, [load]);

  // 实时：运行中每 3s 轮询（不可见门控空转）+ 事件驱动（turn_done 立即、
  // 运行中事件节流）；running→done 由 useLiveReload 触发一次收尾刷新。
  const running = status === "running";

  useLiveReload(running, load);
  // 事件驱动刷新：子代理的工具活动（nested tool_dispatch/tool_result）会经
  // subSinkFor 转发到主事件流；运行时收到即补拉 transcript（transcript 由
  // 后端 ~1s 快照写盘），把「最多等 3s 轮询」收敛到工具边界即时更新。节流
  // 800ms 防事件风暴，turn_done 由 useLiveReload 兜底。
  const lastEventReloadRef = useRef(0);
  useEffect(() => {
    if (!running) return;
    const off = onEvent((e: { kind: string }) => {
      if (e.kind !== "tool_dispatch" && e.kind !== "tool_result") return;
      const now = Date.now();
      if (now - lastEventReloadRef.current < 800) return;
      lastEventReloadRef.current = now;
      load();
    });
    return off;
  }, [running, load]);

  // P1 流式（v4.62.1 分道）：订阅专用通道 gaea-subagent-text（无 seq、有损
  // 无妨），按 subagentRef 路由到本会话 tab。增量绝不走 gaea-event——那条
  // 通道的 seq 与账本 1:1（v4.26 缺口防线），装饰性流上去会制造不可愈合缺
  // 口、触发反复 resync 打断对话窗（v4.62.0 回归）。缓冲从空变非空时记录
  // 消息数基线（reconcile 用）。挂载期常开订阅——非 running 时不渲染缓冲行，
  // 完成态的尾部增量由 load 收尾接管。
  useEffect(() => {
    return onSubagentText((e) => {
      if (!e.text || e.subagentRef !== target) return;
      setStreamBuf((prev) => {
        if (!prev) {
          const msgs = transcriptRef.current?.messages ?? [];
          bufBaseCountRef.current = msgs.length;
        }
        return prev + e.text;
      });
    });
  }, [target]);

  // 切换子代理（target 变化）：缓冲与基线一并作废。
  useEffect(() => {
    setStreamBuf("");
    transcriptRef.current = null;
    setFollowUpRemoteErr(""); // 失败摘要是 per-ref 的，切换即作废
  }, [target]);

  // 运行中强制跟随底部；完成/空闲时保留用户滚动位置（near-bottom 才跟）。
  const scrollRef = useRef<HTMLDivElement>(null);
  const nearBottomRef = useRef(true);
  const onScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    nearBottomRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
  }, []);
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    if (running || nearBottomRef.current) el.scrollTop = el.scrollHeight;
  }, [transcript?.messages.length, running, transcript, streamBuf]);

  const meta = statusMeta(status, t);
  const messages = transcript?.messages ?? [];
  // Codex 式渲染项：tool 调用/结果按 toolCallId 配对成 ToolCard，正文/思考
  // 走主对话 AssistantMessage（同款 Markdown/思考折叠/复制）。
  const isMtTab = target.startsWith("mt_");
  const renderItems = buildRenderItems(messages, running);
  const toast = useToast();

  // ── 保存为新会话（v4.66，dsh Side Chat promote 语义）──────────
  // 把该子代理运行忠实投影为一个独立顶层会话（可从会话列表打开/继续），
  // 原运行不动；running/mt_ 不提供（后端同守卫，双保险）。
  const [promoteBusy, setPromoteBusy] = useState(false);
  const promote = useCallback(() => {
    if (!sessionPath || isMtTab || running || promoteBusy) return;
    setPromoteBusy(true);
    app.PromoteSubagent(sessionPath, target)
      .then(() => toast.show(t("subagent.promoteOk"), "info"))
      .catch((e) => toast.show(t("subagent.promoteFail", { msg: String(e) }), "warn"))
      .finally(() => setPromoteBusy(false));
  }, [sessionPath, isMtTab, running, promoteBusy, target, toast, t]);

  // ── Side Chat 式追问（v4.64）─────────────────────────────────
  // 用户在子代理会话 tab 里对已完结的运行追加提问：乐观上屏本地用户气泡，
  // 派发后台运行（绑定即刻返回），transcript 快照轮询带回真实内容后清掉
  // 乐观态。mt_ 运行是单次工具调用，不提供追问。
  // v4.64.x 失败诚实化：派发被守卫拒绝（主回合运行中/未接线/重复追问）时
  // 乐观气泡不悄悄蒸发，改标失败态并保留原文本——气泡内联「重试」（复用
  // followUpBusy 守卫重发同一文本）与「撤销」；错误条内联展示失败原因。
  // v4.66 后台失败可感知：受理成功但后台 goroutine 真正失败时，失败原因经
  // meta（followUpError）随快照轮询带回，等待态气泡同样转失败态——不再永久
  // 「等待中」。
  const [followUpText, setFollowUpText] = useState("");
  const [followUpBusy, setFollowUpBusy] = useState(false);
  const [followUpErr, setFollowUpErr] = useState("");
  const [followUpPending, setFollowUpPending] = useState("");
  const [followUpPendingFailed, setFollowUpPendingFailed] = useState(false);
  // v4.66 后台失败可感知：绑定受理只代表「已开跑」，后台 goroutine 里真正
  // 失败时后端把原因摘要写回 meta（followUpError），随 transcript 快照带出。
  // 本地镜像最新一次读到的摘要：等待态气泡在场且非空 → 气泡转失败态。
  const [followUpRemoteErr, setFollowUpRemoteErr] = useState("");
  const followUpBaseRef = useRef(0);
  useEffect(() => {
    // 失败优先判定：meta 带回最近一次追问的失败摘要 → 等待态气泡转失败态
    //（复用 v4.64.x 失败态机制：错误条文案 = 该原因，保留重试/撤销）。该
    // 判定先于「快照增长 = 成功」——失败前可能已落了部分内容，单看消息数
    // 会把失败误判成成功。
    if (followUpPending && !followUpPendingFailed && followUpRemoteErr) {
      setFollowUpPendingFailed(true);
      setFollowUpErr(followUpRemoteErr);
      return;
    }
    // 权威快照长过派发时基线 → 真实内容已落盘，清乐观气泡。
    // 失败态气泡例外：该次派发不会有内容落盘，快照增长与本条无关，
    // 清掉会让重试入口凭空消失（诚实保留到用户重试或撤销）。
    if (followUpPending && !followUpPendingFailed && messages.length > followUpBaseRef.current) setFollowUpPending("");
  }, [messages.length, followUpPending, followUpPendingFailed, followUpRemoteErr]);
  const dispatchFollowUp = useCallback((text: string) => {
    followUpBaseRef.current = messages.length;
    setFollowUpPending(text);
    setFollowUpPendingFailed(false);
    setFollowUpRemoteErr(""); // 新一次派发与上一枪的失败摘要一刀两断（后端受理时也同步清盘）
    setFollowUpText("");
    setFollowUpBusy(true);
    setFollowUpErr("");
    app
      .SubagentFollowUp(sessionPath, target, text)
      .then(() => load())
      .catch((e) => {
        setFollowUpErr(e instanceof Error ? e.message : String(e));
        // 诚实处理：气泡转失败态保留文本供一键重试，不留永久「等待中」假象
        setFollowUpPendingFailed(true);
      })
      .finally(() => setFollowUpBusy(false));
  }, [messages.length, sessionPath, target, load]);
  const sendFollowUp = useCallback(() => {
    const text = followUpText.trim();
    if (!text || followUpBusy || isMtTab) return;
    dispatchFollowUp(text);
  }, [followUpText, followUpBusy, isMtTab, dispatchFollowUp]);
  // 失败气泡一键重试：重发同一文本（守卫与 sendFollowUp 相同的 busy 闸）
  const retryFollowUp = useCallback(() => {
    const text = followUpPending.trim();
    if (!text || followUpBusy || isMtTab) return;
    dispatchFollowUp(text);
  }, [followUpPending, followUpBusy, isMtTab, dispatchFollowUp]);
  // 撤销失败气泡：放弃这条追问，连同错误条一并清掉
  const dismissFollowUp = useCallback(() => {
    setFollowUpPending("");
    setFollowUpPendingFailed(false);
    setFollowUpErr("");
  }, []);
  // 追问运行期间保持 3s 轮询拉取新增内容（status prop 可能尚未翻转）。
  // 失败态气泡不算运行中——该次派发没有后台运行，轮询是空转。
  const followUpActive = followUpBusy || (!!followUpPending && !followUpPendingFailed);
  useEffect(() => {
    if (!running && !followUpActive) return;
    const timer = window.setInterval(() => { if (gate) load(); }, THREAD_POLL_MS);
    return () => window.clearInterval(timer);
  }, [running, followUpActive, gate, load]);

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" data-testid="agent-thread" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* 头部：返回 + 标题 + 状态 + 模型 + 消息数 + 刷新 */}
      <div className="v3-panel-head">
        <button
          type="button"
          className="flex shrink-0 cursor-pointer items-center gap-1 rounded-md border-0 bg-transparent px-1.5 py-0.5 text-[11px] transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          onClick={onBack}
          title={t("subagent.backTitle")}
        >
          <Rollback size={12} aria-hidden />
          {t("subagent.title")}
        </button>
        <span className="v3-panel-title min-w-0 truncate" style={{ color: "var(--md-sys-color-text)" }}>{task || target}</span>
        <span
          className="inline-flex shrink-0 items-center gap-1 rounded-full px-1.5 py-px text-[10px] font-medium"
          style={{ color: meta.color, background: meta.bg, border: meta.border }}
        >
          {running && <span className="inline-block h-1 w-1 rounded-full animate-pulse" style={{ background: meta.color }} aria-hidden />}
          {meta.label}
        </span>
        {model && (
          <span className="shrink-0 rounded px-1 py-px font-mono text-[9.5px]" style={{ background: "var(--md-sys-color-surface-container-high)" }}>
            {model}
          </span>
        )}
        <span className="v3-panel-spacer" />
        {!loading && transcript && (
          <span className="shrink-0 font-mono text-[9.5px] tabular-nums" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {t("subagent.msgCount", { n: messages.length })}
          </span>
        )}
        {!isMtTab && (
          <button
            type="button"
            data-testid="subagent-promote"
            className="flex shrink-0 cursor-pointer items-center gap-1 rounded-md border-0 bg-transparent px-1.5 py-0.5 text-[11px] transition-colors hover:bg-(color:--md-sys-color-surface-container-high) disabled:cursor-default disabled:opacity-50"
            style={{ color: "var(--md-sys-color-text-secondary)" }}
            onClick={promote}
            disabled={promoteBusy || running}
            title={t("subagent.promoteSave")}
          >
            {promoteBusy ? <Loader2 size={12} className="animate-spin" aria-hidden /> : <Save size={12} aria-hidden />}
            {t("subagent.promoteSave")}
          </button>
        )}
        <button
          type="button"
          className="flex h-6 w-6 items-center justify-center rounded-md border-0 bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          onClick={() => void load()}
          title={t("subagent.refreshThread")}
          aria-label={t("subagent.refreshThread")}
        >
          <Loader2 size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>

      {/* 消息流 */}
      <div ref={scrollRef} onScroll={onScroll} className="flex-1 min-h-0 overflow-y-auto px-3 py-2">
        {loading && messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <Loader2 size={14} className="animate-spin" />
            {t("subagent.loadingThread")}
          </div>
        ) : failed ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-6 text-center text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <span>{t("subagent.loadFail")}</span>
            <button
              type="button"
              className="cursor-pointer rounded-md border-0 px-2 py-1 text-[11px]"
              style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--gaea-glow)" }}
              onClick={() => void load()}
            >
              {t("subagent.retry")}
            </button>
          </div>
        ) : messages.length === 0 && !streamBuf ? (
          <div className="flex h-full items-center justify-center text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {t("subagent.noMessages")}
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {renderItems.map((it, idx) => {
              if (it.type === "assistant") {
                // mt_ 历史转录里可能存着 JSON 信封串（写端修复前的旧数据）：
                // 渲染前同语义递归拆包，救回被转义墙埋掉的正文。
                const text = isMtTab ? unwrapEnvelopeText(it.text) : it.text;
                const asItem = { kind: "assistant" as const, id: it.id, text, reasoning: it.reasoning ?? "", streaming: it.live };
                // Codex 式有界：mt_ 标签页（文档级输出）或超长文本默认限高滚动
                if (!it.live && (isMtTab || it.text.length > OUTPUT_BOUND_CHARS)) {
                  return <BoundedAssistantMessage key={it.id} item={asItem} chars={text.length} />;
                }
                return (
                  <AssistantMessage
                    key={it.id}
                    item={asItem}
                    deliverTail={false}
                  />
                );
              }
              if (it.type === "tool") {
                const item: ToolItem = {
                  kind: "tool",
                  id: it.id,
                  name: it.name,
                  args: it.args,
                  readOnly: false,
                  status: toolStatus(it, running),
                  output: it.output,
                };
                return <ToolCard key={`${it.id}-${idx}`} item={item} />;
              }
              if (it.type === "system") {
                return (
                  <div key={`s-${idx}`} className="px-2 py-0.5 text-center text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                    {it.text}
                  </div>
                );
              }
              return (
                <div key={`u-${idx}`} className="flex justify-end">
                  <div
                    className="max-w-[88%] whitespace-pre-wrap break-words rounded-lg px-2.5 py-1.5 text-[12.5px] leading-relaxed"
                    style={{ background: "color-mix(in srgb, var(--md-sys-color-surface-container-high) 70%, transparent)", color: "var(--md-sys-color-text)" }}
                  >
                    {it.text}
                  </div>
                </div>
              );
            })}
            {/* Side Chat 追问的乐观用户气泡：快照带回真实内容后清除；
                失败态保留文本并内联「重试/撤销」，不留永久等待假象 */}
            {followUpPending && (
              <div className="flex justify-end">
                <div
                  data-testid="agent-follow-up-pending"
                  data-failed={followUpPendingFailed || undefined}
                  className={
                    "max-w-[88%] whitespace-pre-wrap break-words rounded-lg px-2.5 py-1.5 text-[12.5px] leading-relaxed " +
                    (followUpPendingFailed ? "" : "opacity-70")
                  }
                  style={
                    followUpPendingFailed
                      ? {
                          background: "color-mix(in srgb, var(--md-sys-color-destructive) 8%, transparent)",
                          color: "var(--md-sys-color-text)",
                          border: "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 30%, transparent)",
                        }
                      : { background: "color-mix(in srgb, var(--md-sys-color-surface-container-high) 70%, transparent)", color: "var(--md-sys-color-text)" }
                  }
                >
                  {followUpPending}
                  {followUpPendingFailed && (
                    <div className="mt-1 flex items-center gap-1.5 text-[10.5px]">
                      <span style={{ color: "var(--md-sys-color-destructive)" }}>{t("subagent.followUpFailLabel")}</span>
                      <span className="min-w-0 flex-1" />
                      <button
                        type="button"
                        data-testid="agent-follow-up-retry"
                        className="cursor-pointer rounded border-0 bg-transparent px-1 py-0.5 text-[10.5px] disabled:cursor-not-allowed disabled:opacity-40"
                        style={{ color: "var(--gaea-glow)" }}
                        disabled={followUpBusy}
                        onClick={retryFollowUp}
                      >
                        {t("subagent.retry")}
                      </button>
                      <button
                        type="button"
                        data-testid="agent-follow-up-dismiss"
                        aria-label={t("subagent.close")}
                        title={t("subagent.close")}
                        className="flex h-4 w-4 shrink-0 cursor-pointer items-center justify-center rounded border-0 bg-transparent p-0 transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
                        style={{ color: "var(--md-sys-color-text-secondary)" }}
                        onClick={dismissFollowUp}
                      >
                        <X size={10} aria-hidden />
                      </button>
                    </div>
                  )}
                </div>
              </div>
            )}
            {/* P1 流式实时行：运行中「正在打出的字」，主对话 AssistantMessage
                同款渲染（流式光标/思考自动展开）；快照追上后由 reconcile 清
                缓冲交给权威渲染。 */}
            {running && streamBuf !== "" && (
              <div data-testid="agent-thread-streaming">
                <AssistantMessage
                  item={{ kind: "assistant", id: "live", text: streamBuf, reasoning: "", streaming: true }}
                  deliverTail={false}
                />
              </div>
            )}
          </div>
        )}
      </div>

      {/* Side Chat 式追问输入框（v4.64）：仅 sa_ 真子代理；mt_ 是单次工具
          调用没有可追问的线程。运行中禁用（后端同样拒绝 running 续跑）。 */}
      {!isMtTab && (
        <div className="shrink-0 px-2.5 pb-2" data-testid="agent-follow-up">
          {followUpErr && (
            <div
              data-testid="agent-follow-up-error"
              className="mb-1 break-words rounded-md px-2 py-1 text-[10.5px]"
              style={{
                color: "var(--md-sys-color-destructive)",
                background: "color-mix(in srgb, var(--md-sys-color-destructive) 8%, transparent)",
                border: "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 30%, transparent)",
              }}
            >
              {t("subagent.followUpFail", { msg: followUpErr })}
            </div>
          )}
          <div className="flex items-center gap-1.5">
            <input
              value={followUpText}
              onChange={(e) => setFollowUpText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.nativeEvent.isComposing) {
                  e.preventDefault();
                  sendFollowUp();
                }
              }}
              placeholder={t("subagent.followUpPh")}
              data-testid="agent-follow-up-input"
              className="min-w-0 flex-1 rounded-lg border px-2.5 py-1.5 text-[12px] outline-none"
              style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container-low)", color: "var(--md-sys-color-text)" }}
            />
            <button
              type="button"
              data-testid="agent-follow-up-send"
              aria-label={t("subagent.followUpSend")}
              title={t("subagent.followUpSend")}
              className="flex h-7 w-7 shrink-0 cursor-pointer items-center justify-center rounded-lg border-0 disabled:cursor-not-allowed disabled:opacity-40"
              style={{ background: "var(--gaea-glow)", color: "var(--md-sys-color-surface-container-low)" }}
              disabled={!followUpText.trim() || followUpBusy || running}
              onClick={sendFollowUp}
            >
              {followUpBusy ? <Loader2 size={12} className="animate-spin" /> : <ArrowUp size={13} />}
            </button>
          </div>
        </div>
      )}

      {/* 运行中实时提示 */}
      {running && (
        <div
          className="flex shrink-0 items-center gap-1.5 px-3 py-1 text-[10px]"
          style={{ borderTop: "var(--v3-split)", color: "var(--md-sys-color-text-secondary)" }}
        >
          <Loader2 size={10} className="animate-spin" style={{ color: "var(--gaea-glow)" }} />
          {t("subagent.liveRefreshing")}
        </div>
      )}
    </div>
  );
}
