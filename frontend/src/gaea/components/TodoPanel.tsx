import { useEffect, useRef, useState } from "react";
import { Check, Circle, Loader } from "../icons";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import type { Todo } from "../lib/tools";
import type { Requirement } from "../lib/types";
import { PromptBadge, PromptHeaderAction, PromptShelf } from "./PromptShelf";

const statusIcon = (status: string) => {
  switch (status) {
    case "completed":
      return <Check size={13} className="text-ok shrink-0" />;
    case "in_progress":
      return <Loader size={13} className="text-accent shrink-0 animate-spin" />;
    default:
      return <Circle size={13} className="text-fg-faint shrink-0" />;
  }
};

export function TodoPanel({
  todos,
  onDismiss,
  requirement,
  onToggleRequirementDone,
}: {
  todos: Todo[];
  onDismiss: () => void;
  requirement?: Requirement | null;
  onToggleRequirementDone?: () => void;
}) {
  const t = useT();
  const compact = useCompact();
  const [open, setOpen] = useState(true);
  const listRef = useRef<HTMLUListElement>(null);
  const currentRef = useRef<HTMLLIElement | null>(null);
  useGSAPCollapse(listRef, open);

  // 自动滚动到进行中任务
  useEffect(() => {
    if (open && currentRef.current) {
      currentRef.current.scrollIntoView({ block: "nearest", behavior: "smooth" });
    }
  }, [open]);

  if (todos.length === 0 && !requirement?.text) return null;

  const done = todos.filter((td) => td.status === "completed").length;
  const current = todos.find((td) => td.status === "in_progress");
  const summary = current?.activeForm || current?.content || todos[todos.length - 1]?.content || "";
  const pct = todos.length > 0 ? Math.round((done / todos.length) * 100) : 0;
  const reqDone = !!requirement?.done;

  const itemPy = compact ? "py-[5px]" : "py-[7px]";
  const itemPx = compact ? "px-[7px] pl-[9px]" : "px-[7px] pl-[11px]";
  const itemTextSize = compact ? "text-[11.5px]" : "text-[12.5px]";

  return (
    <PromptShelf
      titleId="todo-shelf-title"
      title={t("todo.title")}
      badges={
        <PromptBadge>
          {done}/{todos.length}
          {pct > 0 && pct < 100 && ` · ${pct}%`}
        </PromptBadge>
      }
      meta={!open ? summary : undefined}
      role="region"
      headerActions={
        <>
          <PromptHeaderAction onClick={() => setOpen((v) => !v)}>
            {open ? t("common.collapse") : t("common.expand")}
          </PromptHeaderAction>
          <PromptHeaderAction onClick={onDismiss}>
            ✕
          </PromptHeaderAction>
        </>
      }
    >
      {/* 任务目标（Kun 从需求到验收）：会话首条消息自动锚定，随会话持久化 */}
      {requirement?.text && (
        <div className={`flex items-start gap-2.5 px-3 ${compact ? "pt-1.5 pb-2" : "pt-2 pb-2.5"} border-b border-border-soft`}>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5 mb-0.5">
              <span className="text-fg-faint text-[10.5px] font-semibold uppercase tracking-[0.02em]">
                任务目标
              </span>
              <span
                className={`inline-flex items-center gap-1 rounded-full px-1.5 py-px text-[10px] font-medium ${
                  reqDone ? "bg-ok/12 text-ok" : "bg-accent/10 text-accent"
                }`}
              >
                {reqDone ? <Check size={10} /> : <Circle size={10} />}
                {reqDone ? "已验收" : "进行中"}
              </span>
            </div>
            <p className={`text-fg leading-relaxed line-clamp-2 ${compact ? "text-[11.5px]" : "text-[12.5px]"}`}>
              {requirement.text}
            </p>
          </div>
          <button
            className={`shrink-0 inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium cursor-pointer transition-[color,background] active:scale-[0.98] ${
              reqDone
                ? "border border-border-soft text-fg-dim hover:text-fg hover:bg-sidebar-hover"
                : "border border-ok/30 text-ok bg-ok/8 hover:bg-ok/15"
            }`}
            onClick={onToggleRequirementDone}
            title={reqDone ? "重新打开任务" : "对照验收标准，标记任务完成"}
          >
            {reqDone ? "重新打开" : "标记验收完成"}
          </button>
        </div>
      )}

      {/* 进度条 — 加高+渐变色+百分比标注 */}
      {todos.length > 0 && (
        <div className="h-[5px] bg-border-soft relative">
          <div
            className={`h-full transition-[width] duration-700 ease-out rounded-r-sm ${
              pct >= 100
                ? "bg-ok"
                : "bg-gradient-to-r from-accent via-accent to-ok/70"
            }`}
            style={{ width: `${pct}%` }}
          />
          {pct >= 100 && (
            <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
              <span className="text-[9px] font-bold text-ok tracking-wider">✓ 全部完成</span>
            </div>
          )}
        </div>
      )}

      {/* 任务列表 */}
      {todos.length > 0 && (
      <ul ref={listRef} className="m-0 p-0 list-none" style={{ overflow: "hidden" }}>
        {todos.map((td, i) => {
          const isPhase = td.level === 0;
          const isSub = td.level != null && td.level > 0;
          const isCurrent = td.status === "in_progress";
          return (
            <li
              key={i}
              ref={isCurrent ? currentRef : undefined}
              className={`relative flex items-center gap-2.5 ${itemPx} ${itemPy} border-b border-border-soft last:border-b-0 transition-colors duration-200 ${
                isCurrent
                  ? "bg-accent-soft/70"
                  : "bg-transparent hover:bg-bg-elev"
              } ${isSub ? (compact ? "pl-8" : "pl-9") : ""}`}
            >
              {/* 左强调条 — 进行中带微动画 */}
              {isCurrent && !isSub && (
                <div className="absolute left-0 top-0 bottom-0 w-[3px] bg-accent rounded-r-sm animate-pulse" />
              )}
              {/* 子任务连接线 */}
              {isSub && (
                <div className="absolute left-[11px] top-0 bottom-0 w-[2px] bg-border-soft" />
              )}

              {statusIcon(td.status)}

              <span
                className={`min-w-0 leading-relaxed ${
                  isPhase ? "font-medium text-fg" : "text-fg-dim"
                } ${
                  td.status === "completed"
                    ? "line-through text-fg-faint/60"
                    : isCurrent
                      ? "text-fg font-semibold"
                      : ""
                } ${itemTextSize}`}
              >
                {isCurrent && td.activeForm ? td.activeForm : td.content}
              </span>
            </li>
          );
        })}
      </ul>
      )}
    </PromptShelf>
  );
}
