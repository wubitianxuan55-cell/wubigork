import { useEffect, useRef, useState, type RefObject } from "react";
import { Check, ChevronDown, ChevronRight, Circle, Loader, X } from "../icons";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import type { Todo } from "../lib/tools";

// 状态图标：completed=对勾（success）、in_progress=旋转（glow）、其余=空心圆。
const statusIcon = (status: string) => {
  switch (status) {
    case "completed":
      return <Check size={13} className="shrink-0" aria-hidden style={{ color: "var(--md-sys-color-success)" }} />;
    case "in_progress":
      return <Loader size={13} className="shrink-0 animate-spin" aria-hidden style={{ color: "var(--gaea-glow)" }} />;
    default:
      return <Circle size={13} className="shrink-0" aria-hidden style={{ color: "var(--md-sys-color-text-secondary)" }} />;
  }
};

// 待办卡（todo_write 提取）：独立于任务目标卡。
// 重设计要点：默认折叠、折叠态显示当前任务摘要、展开后按「未完成在前 / 已完成
// 收尾」分组、阶段行小标题化、当前任务高亮、进度条全圆角。
export function TodoCard({ todos, onDismiss }: { todos: Todo[]; onDismiss: () => void }) {
  const t = useT();
  const compact = useCompact();
  const [open, setOpen] = useState(false); // 默认折叠
  const listRef = useRef<HTMLUListElement>(null);
  const currentRef = useRef<HTMLLIElement>(null);
  useGSAPCollapse(listRef, open);

  const done = todos.filter((td) => td.status === "completed").length;
  const active = todos.filter((td) => td.status !== "completed");
  const finished = todos.filter((td) => td.status === "completed");
  const current = todos.find((td) => td.status === "in_progress");
  const summary = current?.activeForm || current?.content || active[0]?.content || "";
  const pct = todos.length > 0 ? Math.round((done / todos.length) * 100) : 0;
  const allDone = active.length === 0;

  // 展开后自动滚动到进行中任务
  useEffect(() => {
    if (open && currentRef.current) {
      currentRef.current.scrollIntoView?.({ block: "nearest", behavior: "smooth" });
    }
  }, [open]);

  const itemPy = compact ? "py-[5px]" : "py-[7px]";
  const itemPx = compact ? "px-[7px] pl-[9px]" : "px-[7px] pl-[11px]";
  const itemTextSize = compact ? "text-[11.5px]" : "text-[12.5px]";

  const headerBtn =
    "inline-flex items-center justify-center gap-1 border-0 bg-transparent cursor-pointer w-6 h-6 rounded text-fg-faint hover:text-fg hover:bg-bg transition-colors";

  return (
    <section
      aria-label="待办"
      className="max-w-(--maxw) mx-auto mb-2 border border-border rounded-[9px] bg-bg-soft overflow-hidden"
      style={{ boxShadow: "var(--ds-shadow-card)" }}
    >
      {/* 头部：标题 + 进度徽标 + 状态摘要 + 折叠/关闭（单行紧凑） */}
      <div className={`flex items-center gap-1.5 ${compact ? "px-2 py-1" : "px-2.5 py-1.5"}`}>
        <button
          type="button"
          className={headerBtn}
          aria-expanded={open}
          aria-controls="todo-card-body"
          aria-label={open ? "收起待办" : "展开待办"}
          onClick={() => setOpen((v) => !v)}
        >
          {open ? <ChevronDown size={13} aria-hidden /> : <ChevronRight size={13} aria-hidden />}
        </button>
        <span className={`font-semibold shrink-0 ${compact ? "text-[12px]" : "text-[12.5px]"}`}>{t("todo.title")}</span>
        <span
          className="inline-flex items-center gap-1 rounded px-1.5 py-px bg-bg text-fg-faint text-[10px] font-mono tabular-nums shrink-0"
          role="status"
          aria-label={`待办进度 ${done}/${todos.length}`}
        >
          {done}/{todos.length}
          {pct > 0 && pct < 100 && ` · ${pct}%`}
        </span>
        {!open && (
          <span className="text-fg-faint text-[11px] truncate flex-1 min-w-0">
            {allDone
              ? "全部完成"
              : current
                ? `进行中：${summary}`
                : summary
                  ? `待办：${summary}`
                  : `${active.length} 项待办`}
          </span>
        )}
        {open && <span className="flex-1" />}
        <button type="button" className={headerBtn} onClick={onDismiss} aria-label="关闭待办">
          <X size={12} aria-hidden />
        </button>
      </div>

      {/* 展开体 */}
      {open && (
        <div id="todo-card-body" className="border-t border-border-soft">
          {/* 进度条 — 令牌渐变 + 完成态全绿 */}
          {todos.length > 0 && (
            <div className="h-[5px] relative" style={{ background: "var(--md-sys-color-outline-variant)" }}>
              <div
                className="h-full transition-[width] duration-700 ease-out rounded-r-sm"
                style={{
                  width: `${pct}%`,
                  background: allDone
                    ? "var(--md-sys-color-success)"
                    : "linear-gradient(90deg, var(--gaea-glow), color-mix(in srgb, var(--gaea-glow) 62%, var(--md-sys-color-success)))",
                }}
              />
            </div>
          )}

          {/* 任务列表：未完成在前，已完成收尾（原序保留） */}
          {todos.length > 0 && (
            <ul ref={listRef} className="m-0 p-0 list-none" style={{ overflow: "hidden" }}>
              {active.map((td, i) => (
                <TodoRow key={`a-${i}`} td={td} isCurrent={td.status === "in_progress"} currentRef={currentRef} itemPx={itemPx} itemPy={itemPy} itemTextSize={itemTextSize} compact={compact} />
              ))}
              {finished.length > 0 && (
                <li className="flex items-center gap-2 px-3 pt-2 pb-1" aria-hidden="true">
                  <span
                    className="inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-[0.04em]"
                    style={{ color: "var(--md-sys-color-text-secondary)" }}
                  >
                    <Check size={10} aria-hidden />
                    已完成 {finished.length}
                  </span>
                </li>
              )}
              {finished.map((td, i) => (
                <TodoRow
                  key={`d-${i}`}
                  td={td}
                  isCurrent={false}
                  currentRef={currentRef}
                  itemPx={itemPx}
                  itemPy={itemPy}
                  itemTextSize={itemTextSize}
                  compact={compact}
                />
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  );
}

function TodoRow({
  td,
  isCurrent,
  currentRef,
  itemPx,
  itemPy,
  itemTextSize,
  compact,
}: {
  td: Todo;
  isCurrent: boolean;
  currentRef: RefObject<HTMLLIElement>;
  itemPx: string;
  itemPy: string;
  itemTextSize: string;
  compact: boolean;
}) {
  const isPhase = td.level === 0;
  const isSub = td.level != null && td.level > 0;
  const isDone = td.status === "completed";
  return (
    <li
      ref={isCurrent ? currentRef : undefined}
      className={`relative flex items-center gap-2.5 ${itemPx} ${itemPy} border-b last:border-b-0 transition-colors duration-200 ${
        isCurrent ? "" : "hover:bg-(color:--md-sys-color-surface-container-high)"
      } ${isSub ? (compact ? "pl-8" : "pl-9") : ""}`}
      style={{
        borderColor: "color-mix(in srgb, var(--color-border) 75%, transparent)",
        background: isCurrent ? "var(--md-sys-color-primary-container)" : "transparent",
      }}
    >
      {isCurrent && !isSub && (
        <div
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r animate-pulse"
          style={{ background: "var(--gaea-glow)", boxShadow: "0 0 8px var(--gaea-glow)" }}
        />
      )}
      {isSub && <div className="absolute left-[11px] top-0 bottom-0 w-[2px]" style={{ background: "var(--md-sys-color-outline-variant)" }} />}
      {statusIcon(td.status)}
      <span
        className={`min-w-0 leading-relaxed ${isPhase ? "font-semibold" : ""} ${isDone ? "line-through" : isCurrent ? "font-semibold" : ""} ${itemTextSize}`}
        style={{
          color: isDone
            ? "var(--md-sys-color-text-secondary)"
            : isCurrent
              ? "var(--md-sys-color-on-primary-container)"
              : isPhase
                ? "var(--md-sys-color-text)"
                : "var(--md-sys-color-text-secondary)",
        }}
      >
        {isCurrent && td.activeForm ? td.activeForm : td.content}
      </span>
    </li>
  );
}
