import { useCallback, useEffect, useMemo, useState } from "react";
import { CheckCircle, Clock, Inbox, Loader, RefreshCw, XCircle } from "../icons";
import { app, onTaskEvent } from "../lib/bridge";
import type { TaskStatus, TaskView } from "../lib/types";
import { useToast } from "./Toast";

// TaskCenter — 通用任务中心（阶段 5 T5-1）：展示持久化任务队列
// （价格抓取/文件索引重建等）的实时进度，支持取消与重试。
// 数据源：GaeaTaskList 初始拉取 + gaea-task 事件实时增量。

const KIND_LABEL: Record<string, string> = {
  price_fetch: "价格抓取",
  price_fetch_all: "批量价格抓取",
  file_index: "语义索引",
};

function kindLabel(kind: string): string {
  return KIND_LABEL[kind] ?? kind;
}

function statusChip(status: TaskStatus): { cls: string; text: string } {
  switch (status) {
    case "queued":
      return { cls: "text-fg-dim border-neutral-500/40", text: "排队中" };
    case "running":
      return { cls: "text-accent border-accent/50", text: "进行中" };
    case "succeeded":
      return { cls: "text-green-500 border-green-500/50", text: "已完成" };
    case "failed":
      return { cls: "text-red-400 border-red-500/50", text: "失败" };
    case "cancelled":
      return { cls: "text-fg-dim border-neutral-500/40", text: "已取消" };
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
      <div className="flex items-center justify-center h-full text-fg-dim text-xs gap-2">
        <Loader size={14} className="animate-spin" />
        加载任务中…
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs min-h-0">
      <div className="flex items-center gap-2 px-3 py-2 border-b border-neutral-800/60">
        <Inbox size={13} className="text-accent" />
        <span className="text-fg font-medium">任务中心</span>
        {activeCount > 0 && (
          <span className="px-1.5 py-0.5 rounded-full bg-accent/15 text-accent text-[10px]">{activeCount} 个进行中</span>
        )}
        <button
          className="ml-auto p-1 rounded hover:bg-neutral-800/60 text-fg-dim hover:text-fg cursor-pointer"
          title="刷新"
          onClick={load}
        >
          <RefreshCw size={12} />
        </button>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto px-3 py-2 space-y-2">
        {tasks.length === 0 && (
          <div className="flex flex-col items-center justify-center h-40 gap-2 text-fg-dim">
            <CheckCircle size={20} className="text-green-500/70" />
            <span>暂无任务</span>
            <span className="text-[10px]">价格抓取、语义索引等长任务会出现在这里</span>
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
            <div className="pt-2 text-[10px] uppercase tracking-wider text-fg-dim/70">历史</div>
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
  const chip = statusChip(task.status);
  const running = task.status === "running" || task.status === "queued";
  return (
    <div className="rounded-lg border border-neutral-800/70 bg-neutral-900/40 p-2.5 space-y-1.5">
      <div className="flex items-center gap-2">
        <span className="text-fg text-[11px] font-medium truncate">{task.label}</span>
        <span className="text-[10px] text-fg-dim/80 shrink-0">{kindLabel(task.kind)}</span>
        <span className={`ml-auto shrink-0 px-1.5 py-0.5 rounded border text-[10px] ${chip.cls}`}>{chip.text}</span>
      </div>

      {running && (
        <div className="flex items-center gap-2">
          <div className="flex-1 h-1.5 rounded-full bg-neutral-800 overflow-hidden">
            <div
              className="h-full rounded-full bg-accent transition-[width] duration-300"
              style={{ width: `${task.progress}%` }}
            />
          </div>
          <span className="text-[10px] w-8 text-right shrink-0">{task.progress}%</span>
        </div>
      )}

      {(task.message || task.error) && (
        <div className={`text-[10px] leading-relaxed ${task.error && task.status === "failed" ? "text-red-400" : "text-fg-dim/90"}`}>
          {task.error || task.message}
        </div>
      )}

      <div className="flex items-center gap-2 text-[10px] text-fg-dim/70">
        {task.status === "running" && <Clock size={10} />}
        <span>
          {task.status === "running"
            ? `开始于 ${fmtTime(task.startedAt)}`
            : `${fmtTime(task.createdAt)} → ${fmtTime(task.finishedAt)}`}
        </span>
        {task.retryCount > 0 && <span className="text-amber-400/80">已重试 {task.retryCount} 次</span>}
        <span className="ml-auto flex items-center gap-1">
          {running && (
            <button
              className="px-1.5 py-0.5 rounded border border-neutral-700/60 hover:border-red-500/60 hover:text-red-400 cursor-pointer"
              onClick={() => onCancel(task.id)}
            >
              取消
            </button>
          )}
          {(task.status === "failed" || task.status === "cancelled") && (
            <button
              className="px-1.5 py-0.5 rounded border border-neutral-700/60 hover:border-accent/60 hover:text-accent cursor-pointer"
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
