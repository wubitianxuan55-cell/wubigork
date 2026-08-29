import { useCallback, useEffect, useMemo, useRef, useState, type ReactElement } from "react";
import { CheckCircle, Clock, Inbox, Loader, RefreshCw, X, XCircle } from "../icons";
import { workApp, onTaskEvent } from "../lib/bridge";
import type { TaskOutputView, TaskStatus, TaskView } from "../lib/types";
import { isWorkSpaceTask } from "../lib/taskSpace";
import { useToast } from "./Toast";

// TaskCenter — 通用任务中心（阶段 5 T5-1）：展示持久化任务队列
// （价格抓取/文件索引重建等）的实时进度，支持取消与重试。
// 数据源：GaeaTaskList 初始拉取 + gaea-task 事件实时增量。
// v3.2.0（C1）：任务行可选中 → 底部共享输出 dock 回放实时输出（2s 轮询、
// 运行中自动尾随滚动、截断标注）；结束态细分 stopping（取消请求后等待退出）。
// v3.6（C9）：输出 dock 事件即推——gaea-task 事件在输出变更/终态时携带
// outputTail 整尾回放，2s 轮询降级为兜底（对齐 Codex「事件为主、轮询兜底」）。
// v3「星枢」面板语言：v3-panel-head 细条头部；状态徽标 = 语义色 + 图标 + 文字三重传达。

const KIND_LABEL: Record<string, string> = {
  price_fetch: "价格抓取",
  price_fetch_all: "批量价格抓取",
  file_index: "语义索引",
};

function kindLabel(kind: string): string {
  return KIND_LABEL[kind] ?? kind;
}

// 状态 → { 图标, 语义色, 文字 }（不只靠颜色传达）
function statusMeta(status: TaskStatus): { icon: ReactElement; color: string; text: string } {
  switch (status) {
    case "queued":
      return { icon: <Clock size={10} aria-hidden />, color: "var(--md-sys-color-text-secondary)", text: "排队中" };
    case "running":
      return { icon: <Loader size={10} aria-hidden />, color: "var(--gaea-glow)", text: "进行中" };
    case "stopping":
      return { icon: <Loader size={10} className="animate-spin" aria-hidden />, color: "var(--md-sys-color-warning)", text: "停止中" };
    case "succeeded":
      return { icon: <CheckCircle size={10} aria-hidden />, color: "var(--md-sys-color-success)", text: "已完成" };
    case "failed":
      return { icon: <XCircle size={10} aria-hidden />, color: "var(--md-sys-color-destructive)", text: "失败" };
    case "cancelled":
      return { icon: <Clock size={10} aria-hidden />, color: "var(--md-sys-color-text-secondary)", text: "已取消" };
  }
}

function fmtTime(ms: number): string {
  if (!ms) return "—";
  const d = new Date(ms);
  return d.toLocaleTimeString("zh-CN", { hour12: false });
}

export function TaskCenter() {
  const [tasks, setTasks] = useState<TaskView[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();
  // C1：选中任务 → 底部共享输出 dock
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [output, setOutput] = useState<TaskOutputView>({ tail: "", truncated: false });
  const outputRef = useRef<HTMLPreElement>(null);
  // C9：事件回调里读取当前选中项（ref 避免重订阅）
  const selectedIdRef = useRef<string | null>(null);
  selectedIdRef.current = selectedId;

  const load = useCallback(() => {
    workApp
      .TaskList()
      .then((list) => setTasks((list ?? []).filter(isWorkSpaceTask)))
      .catch(() => setTasks([]))
      .finally(() => setLoading(false));
  }, []);

  // 初始拉取 + 事件增量更新（含重启续跑任务）
  useEffect(() => {
    load();
    const off = onTaskEvent((t) => {
      if (!isWorkSpaceTask(t)) return;
      setTasks((prev) => {
        const next = prev.filter((x) => x.id !== t.id);
        return [t, ...next].slice(0, 50);
      });
      // C9：输出 dock 事件即推——事件携带 outputTail（输出变更/终态）时直接
      // 回放，2s 轮询降级为兜底（对齐 Codex「事件为主、轮询兜底」）。
      if (t.id === selectedIdRef.current && typeof t.outputTail === "string") {
        setOutput({ tail: t.outputTail, truncated: !!t.outputTruncated });
      }
    });
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
  }, [selectedId, selectedActive]);

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
        toast.show("已请求取消任务", "info");
      } catch (e) {
        toast.show(`取消失败：${String(e)}`, "warn");
      }
    },
    [toast],
  );

  const retry = useCallback(
    async (id: string) => {
      try {
        await workApp.TaskRetry(id);
        toast.show("任务已重新排队", "info");
      } catch (e) {
        toast.show(`重试失败：${String(e)}`, "warn");
      }
    },
    [toast],
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
        加载任务中…
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* v3 细条头部：标题 + 进行中计数 + 刷新 */}
      <div className="v3-panel-head">
        <Inbox size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">任务中心</span>
        {activeCount > 0 && (
          <span
            className="px-1.5 py-px rounded-full text-[10px]"
            style={{
              background: "color-mix(in srgb, var(--md-sys-color-primary-container) 55%, transparent)",
              color: "var(--gaea-glow)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
            }}
          >
            {activeCount} 个进行中
          </span>
        )}
        <span className="v3-panel-spacer" />
        <button
          className="p-1 rounded-md bg-transparent cursor-pointer transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
          title="刷新"
          aria-label="刷新任务列表"
          onClick={load}
        >
          <RefreshCw size={12} aria-hidden />
        </button>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto px-3 py-2 space-y-2">
        {tasks.length === 0 && (
          <div className="flex flex-col items-center justify-center h-40 gap-2" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            <CheckCircle size={20} aria-hidden style={{ color: "var(--md-sys-color-success)", opacity: 0.7 }} />
            <span>暂无任务</span>
            <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              价格抓取、语义索引等长任务会出现在这里
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
              历史
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
              输出 · {selectedTask.label}
            </span>
            <span className="text-[9.5px] shrink-0 font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {kindLabel(selectedTask.kind)}
            </span>
            {selectedActive && (
              <span className="text-[9.5px] shrink-0 animate-pulse" style={{ color: "var(--gaea-glow)" }}>
                ● 运行中
              </span>
            )}
            <span className="v3-panel-spacer" />
            <button
              type="button"
              className="p-0.5 rounded cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high) transition-colors"
              style={{ color: "var(--md-sys-color-text-secondary)" }}
              onClick={() => setSelectedId(null)}
              title="关闭输出"
              aria-label="关闭输出"
            >
              <X size={11} aria-hidden />
            </button>
          </div>
          <pre
            ref={outputRef}
            className="m-0 px-3 pb-2 text-[10px] leading-relaxed whitespace-pre-wrap break-words overflow-y-auto font-mono"
            style={{ color: "var(--md-sys-color-text-secondary)", maxHeight: 128 }}
          >
            {output.tail || "（暂无输出）"}
          </pre>
          {output.truncated && (
            <div className="px-3 pb-1.5 text-[9.5px]" style={{ color: "var(--md-sys-color-warning)" }}>
              输出过长已截断（仅保留最近 200 行 / 64KB）
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
  const meta = statusMeta(task.status);
  const running = task.status === "running" || task.status === "queued" || task.status === "stopping";
  const cancelable = task.status === "running" || task.status === "queued" || task.status === "stopping";
  return (
    <div
      className="rounded-[var(--radius-md)] p-2.5 space-y-1.5 cursor-pointer transition-colors"
      style={{
        background: selected ? "color-mix(in srgb, var(--gaea-glow) 7%, var(--md-sys-color-surface-container))" : "var(--md-sys-color-surface-container)",
        border: `1px solid ${selected ? "color-mix(in srgb, var(--gaea-glow) 45%, transparent)" : "var(--md-sys-color-outline-variant)"}`,
        boxShadow: "inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 6%, transparent)",
      }}
      onClick={() => onSelect(task.id)}
      title={task.status === "running" || task.status === "stopping" ? "点击查看实时输出" : "点击查看输出"}
    >
      <div className="flex items-center gap-2">
        <span className="text-[11px] font-medium truncate" style={{ color: "var(--md-sys-color-text)" }}>
          {task.label}
        </span>
        <span className="text-[10px] shrink-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {kindLabel(task.kind)}
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
        </div>
      )}

      <div className="flex items-center gap-2 text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
        {task.status === "running" && <Clock size={10} aria-hidden />}
        <span>
          {task.status === "running"
            ? `开始于 ${fmtTime(task.startedAt)}`
            : `${fmtTime(task.createdAt)} → ${fmtTime(task.finishedAt)}`}
        </span>
        {task.retryCount > 0 && <span style={{ color: "var(--md-sys-color-warning)" }}>已重试 {task.retryCount} 次</span>}
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
              {task.status === "stopping" ? "停止中…" : "取消"}
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
              重试
            </button>
          )}
        </span>
      </div>
    </div>
  );
}
