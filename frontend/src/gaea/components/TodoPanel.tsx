import { useEffect, useRef, useState } from "react";
import { Check, Circle, Loader, X } from "../icons";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import type { Todo } from "../lib/tools";
import type { Requirement } from "../lib/types";
import { PromptBadge, PromptHeaderAction, PromptShelf } from "./PromptShelf";

// v3「星枢」面板语言：状态图标/进度条/当前任务全部令牌化
// （--md-sys-color-success / --gaea-glow / --md-sys-color-primary-container），零硬编码色值。
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
            <X size={11} aria-hidden />
          </PromptHeaderAction>
        </>
      }
    >
      {/* 任务目标（Kun 从需求到验收）：会话首条消息自动锚定，随会话持久化 */}
      {requirement?.text && (
        <div className={`flex items-start gap-2.5 px-3 ${compact ? "pt-1.5 pb-2" : "pt-2 pb-2.5"} border-b`} style={{ borderBottom: "var(--v3-split)" }}>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5 mb-0.5">
              <span className="text-[10.5px] font-semibold uppercase tracking-[0.02em]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                任务目标
              </span>
              <span
                className="inline-flex items-center gap-1 rounded-full px-1.5 py-px text-[10px] font-medium"
                style={
                  reqDone
                    ? {
                        background: "color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)",
                        color: "var(--md-sys-color-success)",
                        border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 30%, transparent)",
                      }
                    : {
                        background: "color-mix(in srgb, var(--md-sys-color-primary-container) 55%, transparent)",
                        color: "var(--gaea-glow)",
                        border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
                      }
                }
              >
                {reqDone ? <Check size={10} aria-hidden /> : <Circle size={10} aria-hidden />}
                {reqDone ? "已验收" : "进行中"}
              </span>
            </div>
            <p className={`leading-relaxed line-clamp-2 ${compact ? "text-[11.5px]" : "text-[12.5px]"}`} style={{ color: "var(--md-sys-color-text)" }}>
              {requirement.text}
            </p>
          </div>
          <button
            className={`shrink-0 inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium cursor-pointer transition-[color,background,border-color] active:scale-[0.98] ${
              reqDone ? "" : "hover:brightness-110"
            }`}
            style={
              reqDone
                ? {
                    border: "1px solid var(--md-sys-color-outline-variant)",
                    color: "var(--md-sys-color-text-secondary)",
                    background: "transparent",
                  }
                : {
                    border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 36%, transparent)",
                    color: "var(--md-sys-color-success)",
                    background: "color-mix(in srgb, var(--md-sys-color-success) 9%, transparent)",
                  }
            }
            onClick={onToggleRequirementDone}
            title={reqDone ? "重新打开任务" : "对照验收标准，标记任务完成"}
          >
            {reqDone ? "重新打开" : "标记验收完成"}
          </button>
        </div>
      )}

      {/* 进度条 — 令牌渐变 + 百分比标注 */}
      {todos.length > 0 && (
        <div className="h-[5px] relative" style={{ background: "var(--md-sys-color-outline-variant)" }}>
          <div
            className="h-full transition-[width] duration-700 ease-out rounded-r-sm"
            style={{
              width: `${pct}%`,
              background:
                pct >= 100
                  ? "var(--md-sys-color-success)"
                  : "linear-gradient(90deg, var(--gaea-glow), color-mix(in srgb, var(--gaea-glow) 62%, var(--md-sys-color-success)))",
            }}
          />
          {pct >= 100 && (
            <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
              <span
                className="inline-flex items-center gap-0.5 text-[9px] font-bold tracking-wider"
                style={{ color: "var(--md-sys-color-success)" }}
              >
                <Check size={9} aria-hidden />
                全部完成
              </span>
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
              className={`relative flex items-center gap-2.5 ${itemPx} ${itemPy} border-b last:border-b-0 transition-colors duration-200 ${
                isCurrent
                  ? ""
                  : "hover:bg-(color:--md-sys-color-surface-container-high)"
              } ${isSub ? (compact ? "pl-8" : "pl-9") : ""}`}
              style={{
                borderColor: "color-mix(in srgb, var(--color-border) 75%, transparent)",
                background: isCurrent ? "var(--md-sys-color-primary-container)" : "transparent",
              }}
            >
              {/* 左强调条 — 进行中带微动画（reduced-motion 下全局关停） */}
              {isCurrent && !isSub && (
                <div
                  className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r animate-pulse"
                  style={{ background: "var(--gaea-glow)", boxShadow: "0 0 8px var(--gaea-glow)" }}
                />
              )}
              {/* 子任务连接线 */}
              {isSub && (
                <div className="absolute left-[11px] top-0 bottom-0 w-[2px]" style={{ background: "var(--md-sys-color-outline-variant)" }} />
              )}

              {statusIcon(td.status)}

              <span
                className={`min-w-0 leading-relaxed ${
                  isPhase ? "font-medium" : ""
                } ${
                  td.status === "completed"
                    ? "line-through"
                    : isCurrent
                      ? "font-semibold"
                      : ""
                } ${itemTextSize}`}
                style={{
                  color:
                    td.status === "completed"
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
        })}
      </ul>
      )}
    </PromptShelf>
  );
}
