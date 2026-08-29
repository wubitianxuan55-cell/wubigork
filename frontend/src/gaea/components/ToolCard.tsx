import { memo, useMemo, useRef, useState } from "react";
import {
  Ban,
  Check,
  ChevronRight,
  Loader2,
  X,
} from "../icons";


import { ICONS, mcpOr } from "./tool_icons";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import { boundedOutput, diffsFor, subjectOf, summarize } from "../lib/tools";
import type { Item } from "../lib/store";
import { FileLinkText } from "./FileLinkText";

type ToolItem = Extract<Item, { kind: "tool" }>;

function pretty(json: string): string {
  try {
    return JSON.stringify(JSON.parse(json), null, 2);
  } catch {
    return json;
  }
}

function StatusGlyph({ status, recoverable }: { status: ToolItem["status"]; recoverable?: boolean }) {
  if (status === "running") return <Loader2 className="animate-spin" size={12} />;
  if (status === "error") return <X className={recoverable ? "text-fg-faint/60" : "text-err"} size={12} />;
  if (status === "stopped") return <Ban className="text-fg-faint" size={12} />;
  return <Check className="text-ok" size={12} />;
}

export const ToolCard = memo(function ToolCard({ item, subcalls }: { item: ToolItem; subcalls?: ToolItem[] }) {
  const t = useT();
  const compact = useCompact();
  const diffs = useMemo(() => diffsFor(item.name, item.args), [item.name, item.args]);
  const subject = useMemo(() => subjectOf(item.name, item.args), [item.name, item.args]);
  const Icon = ICONS[item.name] ?? mcpOr(item.name);
  const nested = subcalls ?? [];
  const hasNested = nested.length > 0;

  const summary =
    item.status === "running"
      ? ""
      : hasNested
        ? t(nested.length === 1 ? "tool.stepOne" : "tool.stepOther", { n: nested.length })
        : summarize(item.name, item.args, item.output, item.error);

  const hasArgs = diffs.length > 0 || !!item.args;
  const hasOutput = !!item.output;
  const expandable = hasArgs || hasOutput;

  const [open, setOpen] = useState(false);
  // P2-2 大工具输出有界预览：超长输出折叠为头部 + 展开全部开关
  const bounded = useMemo(() => boundedOutput(item.output), [item.output]);
  const [showFullOutput, setShowFullOutput] = useState(false);

  const bodyRef = useRef<HTMLDivElement>(null);
  useGSAPCollapse(bodyRef, open && expandable);

  const quiet =
    item.readOnly && !hasNested && item.status !== "error" && item.status !== "stopped";

  const outputLines = item.output ? item.output.split("\n").length : 0;

  const rowPy = compact ? "py-0" : "py-0.5";
  const rowPx = compact ? "px-1" : "px-2";
  const chevronSize = compact ? 11 : 12;
  const summarySize = compact ? "text-[11px]" : "text-[12px]";
  const innerPx = compact ? "px-1" : "px-2";
  const innerPb = compact ? "pb-1" : "pb-1.5";

  // Codex 式工具行：无边框无背景块，hover 才高亮；只读工具安静降噪。
  return (
    <div
      className={`rounded-md transition-colors duration-150 ${
        expandable ? "hover:bg-(color:--md-sys-color-surface-container-high)" : ""
      } ${item.status === "stopped" ? "opacity-60" : ""}`}
      data-tone={item.status === "error" && !item.recoverable ? "danger" : item.status === "running" ? "info" : item.status === "done" ? "success" : item.status === "stopped" ? "warning" : undefined}
    >
      <div
        className={`flex items-center gap-1.5 ${rowPx} ${rowPy} select-none ${
          expandable ? "cursor-pointer hover:bg-bg-soft" : ""
        } ${quiet ? "text-fg-faint/60" : "text-fg-dim"}`}
        onClick={expandable ? () => setOpen((v) => !v) : undefined}
      >
        {expandable ? (
          <ChevronRight
            className={`shrink-0 transition-transform duration-200 ${open ? "rotate-90" : ""}`}
            size={chevronSize}
          />
        ) : (
          <ChevronRight className="shrink-0 invisible" size={chevronSize} />
        )}
        <Icon
          className={`shrink-0 ${item.status === "error" && !item.recoverable ? "text-err" : item.status === "error" && item.recoverable ? "text-fg-faint/60" : item.status === "running" ? "text-accent" : "text-fg-faint"}`}
          size={chevronSize + 2}
        />
        <span className={`font-mono font-medium truncate ${item.status === "error" && !item.recoverable ? "text-err" : item.status === "error" && item.recoverable ? "text-fg-dim/60 line-through" : quiet ? "text-fg-faint/70" : "text-fg"} ${compact ? "text-[11px]" : "text-[12px]"}`}>
          {item.name}
        </span>
        {subject && (
          <span className={`text-fg-faint truncate ${summarySize}`}>{subject}</span>
        )}
        {summary && (
          <span className={`text-fg-faint italic ml-0.5 ${summarySize}`}>{summary}</span>
        )}
        <span className="ml-auto shrink-0">
          <StatusGlyph status={item.status} recoverable={item.recoverable} />
        </span>
      </div>

      <div ref={bodyRef} style={{ overflow: "hidden" }}>
        <div>
          {diffs.map((d, i) => (
            <div className={`${innerPx} ${innerPb}`} key={i}>
              {d.label && <div className="text-[10px] text-fg-faint uppercase tracking-wider mb-0.5">{d.label}</div>}
              <pre className="px-3 py-2 font-mono text-[12px] leading-[1.5] overflow-auto whitespace-pre bg-bg-soft border border-border-soft rounded text-fg-dim"><code>{d.original}</code></pre>
            </div>
          ))}

          {hasNested && (
            <div className="pl-3 border-l border-border-soft ml-3">
              {nested.map((c) => (
                <ToolCard key={c.id} item={c} />
              ))}
            </div>
          )}

          {hasArgs && (
            <div className={`${innerPx} ${innerPb}`}>
              {item.args && <pre className="px-3 py-2 font-mono text-[12px] leading-[1.5] overflow-auto whitespace-pre bg-bg-soft border border-border-soft rounded text-fg-dim"><code>{pretty(item.args)}</code></pre>}
            </div>
          )}
          {hasOutput && (
            <div className={`${innerPx} ${innerPb}`}>
              <div className="text-[9px] text-fg-faint/60 uppercase tracking-wider mb-0.5 select-none">输出 · {outputLines}L</div>
              <pre className="px-3 py-2 font-mono text-[12px] leading-[1.5] overflow-auto whitespace-pre bg-bg-soft border border-border-soft rounded text-fg-dim"><code><FileLinkText text={showFullOutput ? bounded.full : bounded.preview} compact /></code></pre>
              {bounded.collapsed && (
                <button
                  type="button"
                  className="mt-1 px-2 py-0.5 border border-border-soft rounded bg-bg-soft text-fg-dim text-[11px] cursor-pointer hover:bg-bg-soft hover:text-fg transition-colors"
                  onClick={() => setShowFullOutput((v) => !v)}
                >
                  {showFullOutput ? "收起输出" : `展开全部 ${bounded.hiddenLines} 行`}
                </button>
              )}
              {item.truncated && (
                <div className="mt-1 px-2 py-0.5 border border-border-soft rounded bg-bg-soft text-fg-dim text-[11px]">
                  {t("tool.truncated")}
                </div>
              )}
            </div>
          )}

          {item.error && !item.recoverable && (
            <div className={`${innerPx} py-1 text-err text-[12px] leading-snug border-t border-err/20`}>
              {item.error}
            </div>
          )}
          {item.error && item.recoverable && (
            <div className={`${innerPx} py-1 text-fg-faint/60 text-[12px] leading-snug border-t border-fg-faint/15`}>
              {item.error}
            </div>
          )}
        </div>
      </div>
    </div>
  );
});
