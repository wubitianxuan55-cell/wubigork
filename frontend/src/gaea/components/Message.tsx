import { memo, useCallback, useMemo, useRef, useState, useEffect } from "react";
import { ChevronRight, Brain, FileText, Rollback, Wand2 } from "../icons";
import { app } from "../lib/bridge";
import { MemoMarkdown } from "./MemoMarkdown";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import { displayReasoningText } from "../lib/reasoningDisplay";
import { useNow } from "../lib/useNow";
import { usePreviewStore, useTurnStartAt } from "../lib/store";
import type { Item } from "../lib/store";
import { DeliverableCards } from "./DeliverableCards";

type AssistantItem = Extract<Item, { kind: "assistant" }>;

// 行内附件渲染：图片显示缩略图，文件显示图标
function InlineAttachment({ path }: { path: string }) {
  const [dataUrl, setDataUrl] = useState<string | null>(null);
  const [isImage, setIsImage] = useState(false);
  const openFilePreview = usePreviewStore((s) => s.openFilePreview);
  useEffect(() => {
    let live = true;
    if (/\.(png|jpg|jpeg|gif|webp|bmp|svg)$/i.test(path)) {
      setIsImage(true);
      app.AttachmentDataURL(path).then((url) => { if (live) setDataUrl(url); }).catch(() => {});
    } else {
      setIsImage(false);
    }
    return () => { live = false; };
  }, [path]);
  const fileName = path.split("/").pop() ?? path;
  if (isImage && dataUrl) {
    return (
      <button
        type="button"
        onClick={() => openFilePreview(path)}
        title={`点击预览 ${path}`}
        className="block p-0 border-0 bg-transparent cursor-pointer rounded-lg my-1"
      >
        <img src={dataUrl} alt={fileName} className="max-w-[240px] max-h-[180px] rounded-lg border border-border-soft object-cover hover:opacity-85 transition-opacity" loading="lazy" />
      </button>
    );
  }
  if (isImage) {
    return (
      <button type="button" onClick={() => openFilePreview(path)} className="text-accent/60 text-[12px] italic cursor-pointer hover:text-accent">
        [{fileName}]
      </button>
    );
  }
  return (
    <button
      type="button"
      onClick={() => openFilePreview(path)}
      title={`点击预览 ${path}`}
      className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-bg-soft border border-border-soft text-fg-dim text-[11px] font-mono mx-0.5 cursor-pointer hover:border-accent/40 hover:text-fg transition-colors"
    >
      <FileText size={11} className="text-accent shrink-0" />
      {fileName}
    </button>
  );
}

function UserAvatar({ size = 14 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="8" r="4" />
      <path d="M4 22c0-4.4 3.6-8 8-8s8 3.6 8 8" />
    </svg>
  );
}


// ── 推理区 ────────────────────────────────────────────────────────────

// ── UserMessage ───────────────────────────────────────────────────────────

export const UserMessage = memo(function UserMessage({
  text,
  turn,
  open,
  onToggle,
  onRewind,
}: {
  text: string;
  turn?: number;
  open?: boolean;
  onToggle?: () => void;
  onRewind?: (turn: number, scope: string) => void;
}) {
  const t = useT();
  const compact = useCompact();
  const canRewind = onRewind != null && turn != null;
  const rewind = (scope: string) => onRewind?.(turn as number, scope);
  // 解析 @ 附件引用 → 分段渲染
  const textParts = useMemo(() => {
    const parts: { type: "text" | "attachment"; value: string }[] = [];
    const re = /(@\.gaea\/attachments\/[^\s)]+)/g;
    let last = 0;
    let m: RegExpExecArray | null;
    while ((m = re.exec(text)) !== null) {
      if (m.index > last) parts.push({ type: "text", value: text.slice(last, m.index) });
      parts.push({ type: "attachment", value: m[1].slice(1) });
      last = re.lastIndex;
    }
    if (last < text.length) parts.push({ type: "text", value: text.slice(last) });
    return parts;
  }, [text]);
  return (
    <div className="flex justify-end my-2 group" data-entrance={turn != null ? `u${turn}` : undefined}>
      <div className={`flex items-start gap-2 max-w-[85%] ${compact ? "min-w-[120px]" : "min-w-[160px]"}`}>
        <div className="flex-1">
          <div className={`rounded-2xl rounded-br-md px-3.5 py-2 bg-accent/10 border border-accent/20 shadow-[inset_0_1px_0_color-mix(in_srgb,var(--accent)_14%,transparent)] ${
            compact ? "text-[13px]" : "text-[14px]"
          } text-fg leading-relaxed`}>
            {textParts.map((part, i) => {
              if (part.type === "text") return <span key={i}>{part.value}</span>;
              return <InlineAttachment key={i} path={part.value} />;
            })}
          </div>
          {canRewind && (
            <div className="flex justify-end mt-0.5">
              <button
                className="opacity-0 group-hover:opacity-100 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/50 text-[10px] cursor-pointer hover:text-fg transition-opacity"
                onClick={onToggle}
                title={t("rewind.label")}
              >
                <Rollback size={10} className="inline-block -mt-px mr-0.5" aria-hidden />
                回退
              </button>
              {open && (
                <div className="absolute bottom-full right-0 mb-1 z-30 min-w-[140px] py-1 bg-bg-elev-2 border border-border rounded-lg" style={{boxShadow: "var(--ds-shadow-dropdown)"}}>
                  {(["both","conversation","code","fork","summ-from","summ-upto"] as const).map(scope => {
                    const key = scope === "summ-from" ? "rewind.summFrom" as const : scope === "summ-upto" ? "rewind.summUpto" as const : `rewind.${scope}` as const;
                    return (
                    <button key={scope} className="w-full text-left px-3 py-1.5 border-0 bg-transparent text-fg-dim text-[12px] cursor-pointer hover:bg-bg-soft hover:text-fg" onClick={() => rewind(scope)}>
                      {t(key)}
                    </button>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
        <span className="shrink-0 w-7 h-7 rounded-full bg-accent/15 flex items-center justify-center text-accent mt-0.5 shadow-[0_0_10px_color-mix(in_srgb,var(--accent)_16%,transparent)]">
          <UserAvatar size={14} />
        </span>
      </div>
    </div>
  );
});

// ── AssistantMessage ──────────────────────────────────────────────────────

export const AssistantMessage = memo(function AssistantMessage({
  item,
  onCapture,
}: {
  item: AssistantItem;
  onCollapse?: () => void;
  onCapture?: (solution: string) => void;
}) {
  const t = useT();
  const compact = useCompact();
  const now = useNow();
  const turnStartAt = useTurnStartAt();
  const reasoningBodyRef = useRef<HTMLDivElement>(null);

  const reasoningRunning = !!(item.streaming && !item.text && item.reasoning);
  const [userToggled, setUserToggled] = useState(false);
  const [reasoningOpenState, setReasoningOpenState] = useState(false);
  const reasoningOpen = userToggled ? reasoningOpenState : !!item.streaming;
  useGSAPCollapse(reasoningBodyRef, reasoningOpen);
  const toggleReasoning = useCallback(() => {
    setUserToggled(true);
    setReasoningOpenState((v) => !v);
  }, []);

  const reasoningDisplay = displayReasoningText(item.reasoning ?? "", {
    streaming: item.streaming ?? false,
    truncateStreaming: true,
  });
  const reasoningLines = item.reasoning ? item.reasoning.split("\n").filter(l => l.trim()).length : 0;

  const elapsed = turnStartAt > 0 ? Math.max(0, now - Math.floor(turnStartAt / 1000)) : 0;
  const elapsedStr = elapsed < 60 ? `${elapsed}s` : `${Math.floor(elapsed / 60)}m${elapsed % 60}s`;

  // 流式处理中的纯文本（不渲染 Markdown）
  const streaming = item.streaming ?? false;
  return (
    <div className="flex justify-start my-2" data-entrance={item.id}>
      <div className="flex-1 min-w-0">
          {/* 推理区 */}
          {item.reasoning && (
            <div className="mb-1.5">
              <button
                type="button"
                className={`flex items-center gap-1.5 w-full px-2.5 py-1 rounded-lg border transition-colors ${
                  reasoningOpen
                    ? "border-accent/20 bg-accent/5 shadow-[inset_0_1px_0_color-mix(in_srgb,var(--accent)_10%,transparent)]"
                    : "border-transparent hover:bg-bg-soft"
                } text-fg-faint text-[11px] cursor-pointer`}
                onClick={toggleReasoning}
                aria-expanded={reasoningOpen}
              >
                <Brain size={13} className="flex-shrink-0" />
                {reasoningRunning && <span aria-hidden className="w-1 h-1 rounded-full bg-accent animate-pulse shadow-[0_0_6px_var(--accent)] shrink-0" />}
                <span className="font-medium">{reasoningRunning ? t("msg.thinkingRunning") : t("msg.thinking")}</span>
                <span className="text-fg-faint/50 text-[10px] ml-auto tabular-nums">
                  {reasoningRunning
                    ? elapsedStr
                    : `${reasoningLines} 行 · ${elapsedStr}`}
                </span>
                <ChevronRight
                  className={`transition-transform duration-200 ${reasoningOpen ? "rotate-90" : ""}`}
                  size={11}
                />
              </button>
              <div ref={reasoningBodyRef} style={{ overflow: "hidden" }}>
                <div className={`mt-1 px-2.5 py-1.5 border-l-2 border-accent/25 ml-1 text-fg-dim/80 text-[11px] leading-relaxed whitespace-pre-wrap ${
                  compact ? "max-h-[160px] overflow-y-auto" : ""
                }`}>
                  {reasoningDisplay}
                </div>
              </div>
            </div>
          )}

          {/* 正文区 */}
          {item.text && (
            <div className="min-w-0">
              <MemoMarkdown text={item.text} streaming={streaming} />
              {/* 流式光标：生成中闪烁的细光标（reduced-motion 下全局动画关停 → 静态） */}
              {streaming && (
                <span aria-hidden className="inline-block w-[2px] h-[1.05em] align-text-bottom ml-0.5 rounded-full bg-accent/80 animate-pulse" />
              )}
            </div>
          )}
          {/* 交付物附件卡片：正文中的文件引用渲染成可点击预览卡片 */}
          {item.text && <DeliverableCards text={item.text} />}

          {/* 沉淀为技能：把这次成功对话一键封装为可复用 playbook */}
          {onCapture && !streaming && item.text && (
            <div className="mt-1 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <button
                type="button"
                className="inline-flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/60 text-[10.5px] cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
                onClick={() => onCapture(item.text)}
                title="把这次任务与回答保存为可复用技能（/技能名 调用）"
              >
                <Wand2 size={11} />
                沉淀为技能
              </button>
            </div>
          )}
        </div>
    </div>
  );
});
