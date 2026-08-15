import { useCallback, useEffect, useMemo, useState, type ReactElement } from "react";
import { CheckCircle, Clock, Inbox, Loader, RefreshCw, XCircle } from "../icons";
import { app, onTaskEvent } from "../lib/bridge";
import type { TaskStatus, TaskView } from "../lib/types";
import { useToast } from "./Toast";

// TaskCenter — 通用任务中心（阶段 5 T5-1）：展示持久化任务队列
// （价格抓取/文件索引重建等）的实时进度，支持取消与重试。
// 数据源：GaeaTaskList 初始拉取 + gaea-task 事件实时增量。
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

  const load = useCallback(() => {
    app
      .TaskList()
      .then((list) => setTasks(list ?? []))
      .catch(() => setTasks([]))
      .finally(() => setLoading(false));
  }, []);

  // 初始拉取 + 事件增量更新（含重启续跑任务）
  useEffect(() => {
    load();
    const off = onTaskEvent((t) => {
      setTasks((prev) => {
        const next = prev.filter((x) => x.id !== t.id);
        return [t, ...next].slice(0, 50);
      });
    });
    return off;
  }, [load]);

  const cancel = useCallback(
    async (id: string) => {
      try {
        await app.TaskCancel(id);
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
        await app.TaskRetry(id);
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
      if (t.status === "queued" || t.status === "running") activeList.push(t);
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
              <TaskRow key={t.id} task={t} onCancel={cancel} onRetry={retry} />
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
                <TaskRow key={t.id} task={t} onCancel={cancel} onRetry={retry} />
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function TaskRow({
  task,
  onCancel,
  onRetry,
}: {
  task: TaskView;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
}) {
  const meta = statusMeta(task.status);
  const running = task.status === "running" || task.status === "queued";
  return (
    <div
      className="rounded-[var(--radius-md)] p-2.5 space-y-1.5"
      style={{
        background: "var(--md-sys-color-surface-container)",
        border: "1px solid var(--md-sys-color-outline-variant)",
        boxShadow: "inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 6%, transparent)",
      }}
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
          {running && (
            <button
              className="px-1.5 py-0.5 rounded-md cursor-pointer transition-colors"
              style={{
                border: "1px solid var(--md-sys-color-outline-variant)",
                color: "var(--md-sys-color-text-secondary)",
                background: "transparent",
              }}
              onClick={() => onCancel(task.id)}
            >
              取消
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
              onClick={() => onRetry(task.id)}
            >
              重试
            </button>
          )}
        </span>
      </div>
    </div>
  );
}
