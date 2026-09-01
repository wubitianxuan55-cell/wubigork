import { useCallback, useEffect, useRef, useState } from "react";
import { Brain, ChevronRight, Loader2, Rollback } from "../icons";
import { app } from "../lib/bridge";
import type { SubagentTranscriptMessage, SubagentTranscriptView } from "../lib/types";
import { usePollingGate } from "../../hooks/usePollingGate";
import { useLiveReload } from "../hooks/useLiveReload";

// SubagentThread — 子代理对话全面板视图（v4.27 对齐 Codex：点击子代理 →
// 右侧面板打开其对话，运行中实时刷新）。
//
// 此前子代理 transcript 只能经 AgentTree 内嵌窄小卡（10px 字号、max-h-64）
// 手动点「查看完整 transcript」读取一次；本组件把对话提到面板级：
//  - 头部：返回分工 + 任务标题 + 状态徽标 + 模型 + 消息数 + 手动刷新；
//  - 消息流：Codex 式渲染（system 弱化单行 / user 右对齐 / assistant 正文+
//    可折叠思考 / tool 调用与结果小卡），运行中自动跟随底部；
//  - 实时：running 时每 3s 轮询（页面不可见门控空转）+ 事件驱动刷新
//    （turn_done 立即、运行中事件节流，useLiveReload 同看板语义）。

export type SubagentThreadStatus = "running" | "completed" | "failed";

const THREAD_POLL_MS = 3000;

function statusMeta(status: SubagentThreadStatus): { label: string; color: string; bg: string; border: string } {
  switch (status) {
    case "running":
      return {
        label: "进行中",
        color: "var(--gaea-glow)",
        bg: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
        border: "1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)",
      };
    case "failed":
      return {
        label: "失败",
        color: "var(--md-sys-color-destructive)",
        bg: "color-mix(in srgb, var(--md-sys-color-destructive) 10%, transparent)",
        border: "1px solid color-mix(in srgb, var(--md-sys-color-destructive) 30%, transparent)",
      };
    default:
      return {
        label: "已完成",
        color: "var(--md-sys-color-success)",
        bg: "color-mix(in srgb, var(--md-sys-color-success) 10%, transparent)",
        border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 30%, transparent)",
      };
  }
}

// 思考块：默认折叠，运行中的最后一段思考带呼吸点（与主对话 AssistantMessage
// 同一语言；内容太长时滚动）。
function ReasoningBlock({ text, live }: { text: string; live: boolean }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="mb-1">
      <button
        type="button"
        className="flex items-center gap-1.5 rounded-md px-2 py-0.5 text-[10.5px] transition-colors hover:bg-bg-soft"
        style={{ color: "var(--md-sys-color-text-secondary)" }}
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <span
          className="inline-block h-1 w-1 shrink-0 rounded-full"
          style={{ background: live ? "var(--gaea-glow)" : "transparent" }}
          aria-hidden
        />
        <Brain size={11} className="shrink-0" aria-hidden />
        <span className="font-medium">思考</span>
        <ChevronRight size={11} className={`shrink-0 transition-transform duration-200 ${open ? "rotate-90" : ""}`} aria-hidden />
      </button>
      {open && (
        <pre
          className="ml-2 mt-0.5 max-h-44 overflow-auto whitespace-pre-wrap break-words border-l-2 px-2 py-1 font-mono text-[10.5px] leading-relaxed"
          style={{ borderColor: "color-mix(in srgb, var(--gaea-glow) 25%, transparent)", color: "var(--md-sys-color-text-secondary)" }}
        >
          {text}
        </pre>
      )}
    </div>
  );
}

function MessageRow({ m, live }: { m: SubagentTranscriptMessage; live: boolean }) {
  switch (m.role) {
    case "system":
      return (
        <div className="px-2 py-0.5 text-center text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {m.content}
        </div>
      );
    case "user":
      return (
        <div className="flex justify-end">
          <div
            className="max-w-[88%] whitespace-pre-wrap break-words rounded-lg px-2.5 py-1.5 text-[12.5px] leading-relaxed"
            style={{ background: "color-mix(in srgb, var(--md-sys-color-surface-container-high) 70%, transparent)", color: "var(--md-sys-color-text)" }}
          >
            {m.content}
          </div>
        </div>
      );
    case "assistant":
      return (
        <div className="max-w-[94%]">
          {m.reasoning && <ReasoningBlock text={m.reasoning} live={live} />}
          {m.toolCalls?.map((tc) => (
            <div
              key={tc.id}
              className="mb-0.5 flex items-start gap-1.5 rounded-md px-2 py-1 font-mono text-[11px]"
              style={{ background: "color-mix(in srgb, var(--gaea-glow) 6%, transparent)", color: "var(--md-sys-color-text-secondary)" }}
            >
              <span aria-hidden>⚙</span>
              <span className="min-w-0 break-all">
                <span style={{ color: "var(--md-sys-color-text)" }}>{tc.name}</span> {tc.arguments}
              </span>
            </div>
          ))}
          {m.content && (
            <div className="whitespace-pre-wrap break-words text-[12.5px] leading-relaxed" style={{ color: "var(--md-sys-color-text)" }}>
              {m.content}
            </div>
          )}
        </div>
      );
    case "tool":
      return (
        <div
          className="flex items-start gap-1.5 rounded-md border px-2 py-1.5 text-[11px]"
          style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "color-mix(in srgb, var(--md-sys-color-surface-container-high) 40%, transparent)" }}
        >
          <span aria-hidden className="shrink-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>↳</span>
          <div className="min-w-0 flex-1">
            <div className="truncate font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {m.name}
              {m.toolCallId && <span className="ml-1 text-[9.5px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>· {m.toolCallId}</span>}
            </div>
            {m.content && (
              <pre
                className="mt-0.5 max-h-40 overflow-auto whitespace-pre-wrap break-words font-mono text-[10.5px] leading-relaxed"
                style={{ color: "var(--md-sys-color-text-secondary)" }}
              >
                {m.content}
              </pre>
            )}
          </div>
        </div>
      );
  }
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
  const [transcript, setTranscript] = useState<SubagentTranscriptView | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const gate = usePollingGate();

  const load = useCallback(() => {
    if (!sessionPath || !target) return;
    app
      .SubagentTranscript(sessionPath, target)
      .then((v) => {
        setTranscript(v);
        setFailed(false);
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
  useEffect(() => {
    if (!running) return;
    const timer = window.setInterval(() => { if (gate) load(); }, THREAD_POLL_MS);
    return () => window.clearInterval(timer);
  }, [running, gate, load]);
  useLiveReload(running, load);

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
  }, [transcript?.messages.length, running, transcript]);

  const meta = statusMeta(status);
  const messages = transcript?.messages ?? [];
  const lastIdx = messages.length - 1;

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" data-testid="agent-thread" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* 头部：返回 + 标题 + 状态 + 模型 + 消息数 + 刷新 */}
      <div className="v3-panel-head">
        <button
          type="button"
          className="flex shrink-0 cursor-pointer items-center gap-1 rounded-md border-0 bg-transparent px-1.5 py-0.5 text-[11px] transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          onClick={onBack}
          title="返回分工列表"
        >
          <Rollback size={12} aria-hidden />
          分工
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
            {messages.length} 条
          </span>
        )}
        <button
          type="button"
          className="flex h-6 w-6 items-center justify-center rounded-md border-0 bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          onClick={() => void load()}
          title="刷新对话"
          aria-label="刷新对话"
        >
          <Loader2 size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>

      {/* 消息流 */}
      <div ref={scrollRef} onScroll={onScroll} className="flex-1 min-h-0 overflow-y-auto px-3 py-2">
        {loading && messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <Loader2 size={14} className="animate-spin" />
            读取子代理对话…
          </div>
        ) : failed ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-6 text-center text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <span>读取子代理对话失败</span>
            <button
              type="button"
              className="cursor-pointer rounded-md border-0 px-2 py-1 text-[11px]"
              style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--gaea-glow)" }}
              onClick={() => void load()}
            >
              重试
            </button>
          </div>
        ) : messages.length === 0 ? (
          <div className="flex h-full items-center justify-center text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            暂无消息
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {messages.map((m, idx) => (
              <MessageRow key={idx} m={m} live={running && idx === lastIdx} />
            ))}
          </div>
        )}
      </div>

      {/* 运行中实时提示 */}
      {running && (
        <div
          className="flex shrink-0 items-center gap-1.5 px-3 py-1 text-[10px]"
          style={{ borderTop: "var(--v3-split)", color: "var(--md-sys-color-text-secondary)" }}
        >
          <Loader2 size={10} className="animate-spin" style={{ color: "var(--gaea-glow)" }} />
          实时刷新中…
        </div>
      )}
    </div>
  );
}
