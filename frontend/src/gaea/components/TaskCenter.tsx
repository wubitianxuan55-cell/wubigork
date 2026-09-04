import { useCallback, useEffect, useMemo, useRef, useState, type ReactElement } from "react";
import { CheckCircle, Clock, Inbox, Loader, RefreshCw, X, XCircle } from "../icons";
import { workApp, onTaskEvent } from "../lib/bridge";
import { useT, type Translator } from "../lib/i18n";
import type { DictKey } from "../locales/en";
import type { TaskOutputView, TaskStatus, TaskView } from "../lib/types";
import { isWorkSpaceTask } from "../lib/taskSpace";
import { useToast } from "./Toast";
import { usePollingGate } from "../../hooks/usePollingGate";

// TaskCenter — 通用任务中心（阶段 5 T5-1）：展示持久化任务队列
// （价格抓取/文件索引重建等）的实时进度，支持取消与重试。
// 数据源：GaeaTaskList 初始拉取 + gaea-task 事件实时增量。
// v3.2.0（C1）：任务行可选中 → 底部共享输出 dock 回放实时输出（2s 轮询、
// 运行中自动尾随滚动、截断标注）；结束态细分 stopping（取消请求后等待退出）。
// v3.6（C9）：输出 dock 事件即推——gaea-task 事件在输出变更/终态时携带
// outputTail 整尾回放，2s 轮询降级为兜底（对齐 Codex「事件为主、轮询兜底」）。
// v3「星枢」面板语言：v3-panel-head 细条头部；状态徽标 = 语义色 + 图标 + 文字三重传达。

// 任务类型展示名：未知 kind 回退原始值（后端新增类型无需等前端发版）。
const KIND_LABEL: Record<string, DictKey> = {
  price_fetch: "tasks.kind.priceFetch",
  price_fetch_all: "tasks.kind.priceFetchAll",
  file_index: "tasks.kind.fileIndex",
};

function kindLabel(kind: string, t: Translator): string {
  const key = KIND_LABEL[kind];
  return key ? t(key) : kind;
}

// 状态 → { 图标, 语义色, 文字 }（不只靠颜色传达）
function statusMeta(status: TaskStatus, t: Translator): { icon: ReactElement; color: string; text: string } {
  switch (status) {
    case "queued":
      return { icon: <Clock size={10} aria-hidden />, color: "var(--md-sys-color-text-secondary)", text: t("tasks.statusQueued") };
    case "running":
      return { icon: <Loader size={10} aria-hidden />, color: "var(--gaea-glow)", text: t("tasks.statusRunning") };
    case "stopping":
      return { icon: <Loader size={10} className="animate-spin" aria-hidden />, color: "var(--md-sys-color-warning)", text: t("tasks.statusStopping") };
    case "succeeded":
      return { icon: <CheckCircle size={10} aria-hidden />, color: "var(--md-sys-color-success)", text: t("tasks.statusSucceeded") };
    case "failed":
      return { icon: <XCircle size={10} aria-hidden />, color: "var(--md-sys-color-destructive)", text: t("tasks.statusFailed") };
    case "cancelled":
      return { icon: <Clock size={10} aria-hidden />, color: "var(--md-sys-color-text-secondary)", text: t("tasks.statusCancelled") };
  }
}

function fmtTime(ms: number): string {
  if (!ms) return "—";
  const d = new Date(ms);
  return d.toLocaleTimeString("zh-CN", { hour12: false });
}

export function TaskCenter() {
  const t = useT();
  const [tasks, setTasks] = useState<TaskView[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();
  // C1：选中任务 → 底部共享输出 dock
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [output, setOutput] = useState<TaskOutputView>({ tail: "", truncated: false });
  const outputRef = useRef<HTMLPreElement>(null);
  // v4.5.2：输出轮询接入系统级后台轮询门控（页面不可见时空转零成本）
  const gate = usePollingGate();
  // C9：事件回调里读取当前选中项（ref 避免重订阅）
  const selectedIdRef = useRef<string | null>(null);
  selectedIdRef.current = selectedId;

  const load = useCallback(() => {
    workApp
      .TaskList([])
      .then((list) => setTasks((list ?? []).filter(isWorkSpaceTask)))
      .catch(() => setTasks([]))
      .finally(() => setLoading(false));
  }, []);

  // 初始拉取 + 事件增量更新（含重启续跑任务）
  useEffect(() => {
    load();
    // v4.5.1a：事件订阅层按 work 过滤（play 任务事件不打扰工位任务中心）
    const off = onTaskEvent((t) => {
      setTasks((prev) => {
        const next = prev.filter((x) => x.id !== t.id);
        return [t, ...next].slice(0, 50);
      });
      // C9：输出 dock 事件即推——事件携带 outputTail（输出变更/终态）时直接
      // 回放，2s 轮询降级为兜底（对齐 Codex「事件为主、轮询兜底」）。
      if (t.id === selectedIdRef.current && typeof t.outputTail === "string") {
        setOutput({ tail: t.outputTail, truncated: !!t.outputTruncated });
      }
    }, "work");
    return off;
  }, [load]);

  // C1：选中任务后回放输出；running/stopping/queued 时 2s 轮询（整尾回放，不消费游标）。
  // 任务到达终态后停止轮询（保留最后回放，便于复核）。
  const selectedTask = useMemo(() => tasks.find((t) => t.id === selectedId) ?? null, [tasks, selectedId]);
  const selectedActive = selectedTask !== null && (selectedTask.status === "running" || selectedTask.status === "stopping" || selectedTask.status === "queued");

  useEffect(() => {
    if (!selectedId) {
      setOutput({ tail: "", truncated: false });
      return;
    }
    const loadOutput = () => {
      if (!gate) return;
      workApp
        .TaskOutput(selectedId)
        .then((o) => setOutput(o))
        .catch(() => {});
    };
    loadOutput();
    if (selectedActive) {
      const timer = window.setInterval(loadOutput, 2000);
      return () => window.clearInterval(timer);
    }
    return undefined;
  }, [selectedId, selectedActive, gate]);

  // 运行中输出自动尾随滚动到底部（用户未手动上翻时）
  useEffect(() => {
    const el = outputRef.current;
    if (el && selectedActive) {
      el.scrollTop = el.scrollHeight;
    }
  }, [output, selectedActive]);

  const select = useCallback((id: string) => {
    setSelectedId((prev) => (prev === id ? null : id));
  }, []);

  const cancel = useCallback(
    async (id: string) => {
      try {
        await workApp.TaskCancel(id);
        toast.show(t("tasks.cancelRequested"), "info");
      } catch (e) {
        toast.show(t("tasks.cancelFail", { msg: String(e) }), "warn");
      }
    },
    [toast, t],
  );

  const retry = useCallback(
    async (id: string) => {
      try {
        await workApp.TaskRetry(id);
        toast.show(t("tasks.requeued"), "info");
      } catch (e) {
        toast.show(t("tasks.retryFail", { msg: String(e) }), "warn");
      }
    },
    [toast, t],
  );

  const { active, history } = useMemo(() => {
    const activeList: TaskView[] = [];
    const historyList: TaskView[] = [];
    for (const t of tasks) {
      if (t.status === "queued" || t.status === "running" || t.status === "stopping") activeList.push(t);
      else historyList.push(t);
    }
    return { active: activeList, history: historyList };
  }, [tasks]);

  const activeCount = active.length;

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full gap-2 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
        <Loader size={14} className="animate-spin" aria-hidden />
        {t("tasks.loading")}
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* v3 细条头部：标题 + 进行中计数 + 刷新 */}
      <div className="v3-panel-head">
        <Inbox size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">{t("tasks.title")}</span>
        {activeCount > 0 && (
          <span
            className="px-1.5 py-px rounded-full text-[10px]"
            style={{
              background: "color-mix(in srgb, var(--md-sys-color-primary-container) 55%, transparent)",
              color: "var(--gaea-glow)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
            }}
          >
            {t("tasks.activeCount", { n: activeCount })}
          </span>
        )}
        <span className="v3-panel-spacer" />
        <button
          className="p-1 rounded-md bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          title={t("tasks.refreshTitle")}
          aria-label={t("tasks.refreshAria")}
          onClick={load}
        >
          <RefreshCw size={12} aria-hidden />
        </button>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto px-3 py-2 space-y-2">
        {tasks.length === 0 && (
          <div className="flex flex-col items-center justify-center h-40 gap-2" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <CheckCircle size={20} aria-hidden style={{ color: "var(--md-sys-color-success)", opacity: 0.7 }} />
            <span>{t("tasks.empty")}</span>
            <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {t("tasks.emptyHint")}
            </span>
          </div>
        )}

        {active.length > 0 && (
          <div className="space-y-2">
            {active.map((t) => (
              <TaskRow key={t.id} task={t} selected={t.id === selectedId} onSelect={select} onCancel={cancel} onRetry={retry} />
            ))}
          </div>
        )}

        {history.length > 0 && (
          <>
            <div className="pt-2 text-[10px] uppercase tracking-wider" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {t("tasks.history")}
            </div>
            <div className="space-y-1">
              {history.map((t) => (
                <TaskRow key={t.id} task={t} selected={t.id === selectedId} onSelect={select} onCancel={cancel} onRetry={retry} />
              ))}
            </div>
          </>
        )}
      </div>

      {/* C1：共享输出 dock（选中任务时显示） */}
      {selectedTask && (
        <div
          className="shrink-0 border-t flex flex-col min-h-0"
          style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "color-mix(in srgb, var(--md-sys-color-surface-container) 60%, transparent)" }}
        >
          <div className="flex items-center gap-2 px-3 py-1.5">
            <span className="text-[10.5px] font-medium truncate" style={{ color: "var(--md-sys-color-text)" }}>
              {t("tasks.outputHeader", { label: selectedTask.label })}
            </span>
            <span className="text-[9.5px] shrink-0 font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {kindLabel(selectedTask.kind, t)}
            </span>
            {selectedActive && (
              <span className="text-[9.5px] shrink-0 animate-pulse" style={{ color: "var(--gaea-glow)" }}>
                {t("tasks.runningDot")}
              </span>
            )}
            <span className="v3-panel-spacer" />
            <button
              type="button"
              className="p-0.5 rounded cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high) transition-colors"
              style={{ color: "var(--md-sys-color-text-secondary)" }}
              onClick={() => setSelectedId(null)}
              title={t("tasks.closeOutput")}
              aria-label={t("tasks.closeOutput")}
            >
              <X size={11} aria-hidden />
            </button>
          </div>
          <pre
            ref={outputRef}
            className="m-0 px-3 pb-2 text-[10px] leading-relaxed whitespace-pre-wrap break-words overflow-y-auto font-mono"
            style={{ color: "var(--md-sys-color-text-secondary)", maxHeight: 128 }}
          >
            {output.tail || t("tasks.noOutput")}
          </pre>
          {output.truncated && (
            <div className="px-3 pb-1.5 text-[9.5px]" style={{ color: "var(--md-sys-color-warning)" }}>
              {t("tasks.outputTruncated")}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function TaskRow({
  task,
  selected,
  onSelect,
  onCancel,
  onRetry,
}: {
  task: TaskView;
  selected: boolean;
  onSelect: (id: string) => void;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
}) {
  const t = useT();
  const meta = statusMeta(task.status, t);
  const running = task.status === "running" || task.status === "queued" || task.status === "stopping";
  const cancelable = task.status === "running" || task.status === "queued" || task.status === "stopping";
  return (
    <div
      className="group rounded-[var(--radius-md)] p-2.5 space-y-1.5 cursor-pointer transition-colors"
      style={{
        background: selected ? "color-mix(in srgb, var(--gaea-glow) 7%, var(--md-sys-color-surface-container))" : "var(--md-sys-color-surface-container)",
        border: `1px solid ${selected ? "color-mix(in srgb, var(--gaea-glow) 45%, transparent)" : "var(--md-sys-color-outline-variant)"}`,
        boxShadow: "inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 6%, transparent)",
      }}
      onClick={() => onSelect(task.id)}
      title={task.status === "running" || task.status === "stopping" ? t("tasks.viewLiveTitle") : t("tasks.viewOutputTitle")}
    >
      <div className="flex items-center gap-2">
        <span className="text-[11px] font-medium truncate" style={{ color: "var(--md-sys-color-text)" }}>
          {task.label}
        </span>
        <span className="text-[10px] shrink-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {kindLabel(task.kind, t)}
        </span>
        {/* 状态徽标：语义色 + 图标 + 文字三重传达 */}
        <span
          className="ml-auto shrink-0 inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px]"
          style={{
            color: meta.color,
            background: `color-mix(in srgb, ${meta.color} 12%, transparent)`,
            border: `1px solid color-mix(in srgb, ${meta.color} 32%, transparent)`,
          }}
        >
          {meta.icon}
          {meta.text}
        </span>
      </div>

      {running && (
        <div className="flex items-center gap-2">
          <div
            className="flex-1 h-1.5 rounded-full overflow-hidden"
            style={{ background: "var(--md-sys-color-outline-variant)" }}
          >
            <div
              className="h-full rounded-full transition-[width] duration-300"
              style={{
                width: `${task.progress}%`,
                background: "var(--gaea-glow)",
                boxShadow: "0 0 6px color-mix(in srgb, var(--gaea-glow) 50%, transparent)",
              }}
            />
          </div>
          <span className="text-[10px] w-8 text-right shrink-0 font-mono tabular-nums" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {task.progress}%
          </span>
        </div>
      )}

      {(task.message || task.error) && (
        <div
          className="text-[10px] leading-relaxed"
          style={{
            color: task.error && task.status === "failed" ? "var(--md-sys-color-destructive)" : "var(--md-sys-color-text-secondary)",
          }}
        >
          {task.error || task.message}
          {/* 退出码透出（进程类任务真实记录；纯函数任务无退出码语义不渲染）：
              失败时随错误行常显——最直观；语言中性 "exit N" 格式（工程数值） */}
          {task.status === "failed" && task.exitCode !== undefined && <span className="ml-1 font-mono">· exit {task.exitCode}</span>}
        </div>
      )}

      {/* v4.30 行级降噪：时间/重试为次级信息，悬停次行显现（title 保留完整信息） */}
      <div className="flex items-center gap-2 text-[10px] transition-opacity duration-150 group-hover:opacity-100 opacity-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>
        {task.status === "running" && <Clock size={10} aria-hidden />}
        <span>
          {task.status === "running"
            ? t("tasks.startedAt", { time: fmtTime(task.startedAt) })
            : `${fmtTime(task.createdAt)} → ${fmtTime(task.finishedAt)}`}
        </span>
        {task.retryCount > 0 && <span style={{ color: "var(--md-sys-color-warning)" }}>{t("tasks.retried", { n: task.retryCount })}</span>}
        {/* 非失败终态的退出码放悬停次行（失败已在错误行常显，避免重复） */}
        {task.exitCode !== undefined && task.status !== "failed" && <span className="font-mono">exit {task.exitCode}</span>}
        <span className="ml-auto flex items-center gap-1">
          {cancelable && (
            <button
              className="px-1.5 py-0.5 rounded-md cursor-pointer transition-colors"
              style={{
                border: "1px solid var(--md-sys-color-outline-variant)",
                color: task.status === "stopping" ? "var(--md-sys-color-warning)" : "var(--md-sys-color-text-secondary)",
                background: "transparent",
              }}
              onClick={(e) => {
                e.stopPropagation();
                onCancel(task.id);
              }}
            >
              {task.status === "stopping" ? t("tasks.stoppingBtn") : t("tasks.cancelBtn")}
            </button>
          )}
          {(task.status === "failed" || task.status === "cancelled") && (
            <button
              className="px-1.5 py-0.5 rounded-md cursor-pointer transition-colors"
              style={{
                border: "1px solid color-mix(in srgb, var(--gaea-glow) 40%, transparent)",
                color: "var(--gaea-glow)",
                background: "color-mix(in srgb, var(--gaea-glow) 8%, transparent)",
              }}
              onClick={(e) => {
                e.stopPropagation();
                onRetry(task.id);
              }}
            >
              {t("tasks.retryBtn")}
            </button>
          )}
        </span>
      </div>
    </div>
  );
}
