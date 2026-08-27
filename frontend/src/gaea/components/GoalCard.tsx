import { useState } from "react";
import { Check, ChevronRight, Circle, Pencil, Plus, X } from "../icons";
import { useCompact } from "../hooks/useCompact";
import type { Requirement } from "../lib/types";

// 任务目标卡（从需求到验收）：会话级锚点，独立于待办卡。
// 2026-08-26 优化（用户：太占地方）：默认折叠——折叠态只占一行
// （标题 + 状态徽标 + 进度 + 目标文本截断 + 展开箭头），点击展开
// 才显示验收清单 / 添加 / 操作 / 自动追踪开关；折叠/展开带 GSAP 动画。
export function GoalCard({
  requirement,
  onToggleRequirementDone,
  onAddRequirementItem,
  onSetRequirementItem,
  onSetRequirementItemDone,
  onRemoveRequirementItem,
  onToggleRequirementAutoPursue,
}: {
  requirement: Requirement;
  onToggleRequirementDone?: () => void;
  onAddRequirementItem?: (text: string) => void;
  onSetRequirementItem?: (index: number, text: string) => void;
  onSetRequirementItemDone?: (index: number, done: boolean) => void;
  onRemoveRequirementItem?: (index: number) => void;
  onToggleRequirementAutoPursue?: () => void;
}) {
  const compact = useCompact();
  const [open, setOpen] = useState(false); // 默认折叠
  const [newItem, setNewItem] = useState("");
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editingText, setEditingText] = useState("");

  const reqItems = requirement.items ?? [];
  const reqDone = reqItems.length > 0 ? reqItems.every((it) => it.done) : !!requirement.done;
  const reqDoneCount = reqItems.filter((it) => it.done).length;
  const reqAutoPursue = !!requirement.autoPursue;

  const statusBadge = (
    <span
      className="inline-flex items-center gap-1 rounded-full px-1.5 py-px text-[10px] font-medium shrink-0"
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
  );

  const progressBadge = reqItems.length > 0 ? (
    <span
      className="inline-flex items-center rounded-full px-1.5 py-px text-[10px] font-medium tabular-nums shrink-0"
      style={{
        color: "var(--md-sys-color-text-secondary)",
        background: "color-mix(in srgb, var(--md-sys-color-text-secondary) 8%, transparent)",
        border: "1px solid var(--md-sys-color-outline-variant)",
      }}
    >
      {reqDoneCount}/{reqItems.length}
    </span>
  ) : null;

  return (
    <section
      aria-label="任务目标"
      className="max-w-(--maxw) mx-auto mb-2 border border-border rounded-[9px] bg-bg-soft overflow-hidden"
      style={{ boxShadow: "var(--ds-shadow-card)" }}
    >
      {/* 折叠/展开头部：整行可点击，只占一行 */}
      <button
        type="button"
        aria-label="任务目标"
        className={`flex items-center gap-1.5 w-full px-3 ${compact ? "py-1.5" : "py-2"} text-left cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high) transition-colors`}
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-controls="goal-card-body"
      >
        <ChevronRight
          size={12}
          aria-hidden
          className={`shrink-0 text-fg-faint transition-transform duration-200 ${open ? "rotate-90" : ""}`}
        />
        <span className="text-[10.5px] font-semibold uppercase tracking-[0.02em] shrink-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          任务目标
        </span>
        {statusBadge}
        {progressBadge}
        <span className="flex-1 min-w-0 truncate leading-snug" style={{ color: "var(--md-sys-color-text-secondary)" }} title={requirement.text}>
          {requirement.text}
        </span>
      </button>

      {/* 展开体：验收清单 / 添加 / 操作 / 自动追踪（条件渲染，折叠时不占 DOM） */}
      {open && (
        <div id="goal-card-body" className={`px-3 ${compact ? "pt-1 pb-2" : "pt-1.5 pb-2.5"} border-t border-border-soft`}>
        <div className={`px-3 ${compact ? "pt-1 pb-2" : "pt-1.5 pb-2.5"} border-t border-border-soft`}>
          {/* 完整目标文本 */}
          <p className={`leading-relaxed ${compact ? "text-[11.5px]" : "text-[12.5px]"}`} style={{ color: "var(--md-sys-color-text)" }}>
            {requirement.text}
          </p>

          {/* 验收清单：勾选 / 双击编辑 / 删除 */}
          {reqItems.length > 0 && (
            <ul className="m-0 mt-1.5 p-0 list-none flex flex-col gap-px">
              {reqItems.map((it, i) => (
                <li key={i} className="group flex items-center gap-2 rounded-md px-1.5 py-[3px] hover:bg-(color:--md-sys-color-surface-container-high)">
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked={it.done}
                    aria-label={`${it.done ? "取消勾选" : "勾选"}验收项：${it.text}`}
                    className="shrink-0 inline-flex items-center justify-center w-[18px] h-[18px] rounded-[5px] cursor-pointer border transition-colors active:scale-[0.94]"
                    style={
                      it.done
                        ? {
                            background: "var(--md-sys-color-success)",
                            borderColor: "transparent",
                            color: "var(--md-sys-color-on-primary)",
                          }
                        : {
                            background: "transparent",
                            borderColor: "var(--md-sys-color-outline-variant)",
                            color: "transparent",
                          }
                    }
                    onClick={() => onSetRequirementItemDone?.(i, !it.done)}
                    title={it.done ? "取消勾选" : "勾选此项"}
                  >
                    <Check size={11} aria-hidden />
                  </button>
                  {editingIndex === i ? (
                    <input
                      autoFocus
                      value={editingText}
                      onChange={(e) => setEditingText(e.target.value)}
                      onBlur={() => {
                        if (editingText.trim()) onSetRequirementItem?.(i, editingText.trim());
                        setEditingIndex(null);
                      }}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") (e.target as HTMLInputElement).blur();
                        if (e.key === "Escape") setEditingIndex(null);
                      }}
                      className="min-w-0 flex-1 bg-transparent outline-none rounded px-1 py-px text-[11.5px]"
                      style={{ color: "var(--md-sys-color-text)", border: "1px solid var(--md-sys-color-outline-variant)" }}
                    />
                  ) : (
                    <span
                      className={`min-w-0 flex-1 truncate leading-snug ${compact ? "text-[11px]" : "text-[11.5px]"} ${
                        it.done ? "line-through" : ""
                      }`}
                      style={{
                        color: it.done ? "var(--md-sys-color-text-secondary)" : "var(--md-sys-color-text)",
                        cursor: "text",
                      }}
                      title="双击编辑验收项"
                      onDoubleClick={() => {
                        setEditingIndex(i);
                        setEditingText(it.text);
                      }}
                    >
                      {it.text}
                    </span>
                  )}
                  <button
                    type="button"
                    aria-label={`删除验收项：${it.text}`}
                    className="shrink-0 inline-flex items-center justify-center w-[18px] h-[18px] rounded-[5px] cursor-pointer opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity hover:bg-(color:--md-sys-color-error-container)"
                    style={{ color: "var(--md-sys-color-text-secondary)" }}
                    onClick={() => onRemoveRequirementItem?.(i)}
                    title="删除此项"
                  >
                    <X size={11} aria-hidden />
                  </button>
                </li>
              ))}
            </ul>
          )}

          {/* 添加验收项 */}
          {onAddRequirementItem && (
            <div className="mt-1.5 flex items-center gap-1.5">
              <input
                value={newItem}
                onChange={(e) => setNewItem(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && newItem.trim()) {
                    onAddRequirementItem(newItem.trim());
                    setNewItem("");
                  }
                }}
                placeholder="添加验收标准，回车确认…"
                className="min-w-0 flex-1 bg-transparent outline-none rounded-md px-2 py-1 text-[11.5px] placeholder:text-(color:--md-sys-color-text-secondary)"
                style={{ border: "1px dashed var(--md-sys-color-outline-variant)", color: "var(--md-sys-color-text)" }}
                aria-label="添加验收标准"
              />
              <button
                type="button"
                aria-label="添加验收标准"
                className="shrink-0 inline-flex items-center justify-center w-[22px] h-[22px] rounded-md cursor-pointer hover:bg-(color:--md-sys-color-primary-container)"
                style={{ color: "var(--gaea-glow)" }}
                onClick={() => {
                  if (!newItem.trim()) return;
                  onAddRequirementItem(newItem.trim());
                  setNewItem("");
                }}
                title="添加验收标准"
              >
                <Plus size={13} aria-hidden />
              </button>
            </div>
          )}

          {/* 操作行：标记验收 / 重新打开 + 自动追踪开关 */}
          <div className="mt-2 flex items-center gap-2 justify-end">
            {onToggleRequirementAutoPursue && (
              <button
                type="button"
                role="switch"
                aria-checked={reqAutoPursue}
                className={`inline-flex items-center gap-1 rounded-md px-1.5 py-1 text-[10.5px] font-medium cursor-pointer transition-colors active:scale-[0.98] ${
                  reqAutoPursue ? "" : "hover:brightness-110"
                }`}
                style={
                  reqAutoPursue
                    ? {
                        color: "var(--gaea-glow)",
                        background: "color-mix(in srgb, var(--gaea-glow) 12%, transparent)",
                        border: "1px solid color-mix(in srgb, var(--gaea-glow) 32%, transparent)",
                      }
                    : {
                        color: "var(--md-sys-color-text-secondary)",
                        background: "transparent",
                        border: "1px solid var(--md-sys-color-outline-variant)",
                      }
                }
                onClick={onToggleRequirementAutoPursue}
                title="开启后：任务目标写入 agent 目标循环，未达成验收标准会自动继续工作（回合结束由模型判定）"
              >
                {reqAutoPursue ? <Check size={10} aria-hidden /> : <Circle size={10} aria-hidden />}
                自动追踪
              </button>
            )}
            <button
              className={`inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium cursor-pointer transition-[color,background,border-color] active:scale-[0.98] ${
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
              {reqDone ? <Pencil size={11} aria-hidden /> : <Check size={11} aria-hidden />}
              {reqDone ? "重新打开" : "标记验收完成"}
            </button>
          </div>
        </div>
      </div>
      )}
    </section>
  );
}
