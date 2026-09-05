import { memo, useCallback, useMemo, useRef, useState, useEffect } from "react";
import { Bot, Check, ChevronDown, ChevronRight, Brain, Copy, FileText, Rollback, Wand2 } from "../icons";
import { app } from "../lib/bridge";
import { MemoMarkdown } from "./MemoMarkdown";
import { useT } from "../lib/i18n";
import { useCompact } from "../hooks/useCompact";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";
import { displayReasoningText } from "../lib/reasoningDisplay";
import { useNow } from "../lib/useNow";
import { useTurnStartAt } from "../lib/store";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";
import type { Item } from "../lib/store";
import { DeliverableCards } from "./DeliverableCards";

// v4.26 契约：store 的 assistant item 将追加可选字段 subagentRef?: string
// （后端把子代理最终答复以普通 assistant 消息回投主回合时打标）。store.ts
// 本线禁改，这里用交叉类型提前承接——字段缺省（undefined）时渲染行为与
// 现状完全一致，后端/主代理接线后无需再改渲染层。
type AssistantItem = Extract<Item, { kind: "assistant" }> & { subagentRef?: string };

// 行内附件渲染：图片显示缩略图，文件显示图标
function InlineAttachment({ path }: { path: string }) {
  const [dataUrl, setDataUrl] = useState<string | null>(null);
  const [isImage, setIsImage] = useState(false);
  const t = useT();
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
        onClick={() => openPaneFileOrPreview(path)}
        title={t("msg.clickPreview", { path })}
        className="block p-0 border-0 bg-transparent cursor-pointer rounded-lg my-1"
      >
        <img src={dataUrl} alt={fileName} className="max-w-[240px] max-h-[180px] rounded-lg border border-border-soft object-cover hover:opacity-85 transition-opacity" loading="lazy" />
      </button>
    );
  }
  if (isImage) {
    return (
      <button type="button" onClick={() => openPaneFileOrPreview(path)} className="text-accent/60 text-[12px] italic cursor-pointer hover:text-accent">
        [{fileName}]
      </button>
    );
  }
  return (
    <button
      type="button"
      onClick={() => openPaneFileOrPreview(path)}
      title={t("msg.clickPreview", { path })}
      className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-bg-soft border border-border-soft text-fg-dim text-[11px] font-mono mx-0.5 cursor-pointer hover:border-accent/40 hover:text-fg transition-colors"
    >
      <FileText size={11} className="text-accent shrink-0" />
      {fileName}
    </button>
  );
}

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
  // Kimi Work 同款：超长用户消息默认折叠（纯文本超过阈值只显示前几行），
  // 避免长任务描述把对话撑爆；点击「展开/收起」切换。
  const LONG_MSG_CHARS = 240;
  const longText = textParts.some((p) => p.type === "text" && p.value.length > LONG_MSG_CHARS);
  const [msgOpen, setMsgOpen] = useState(false);
  // Codex 式用户消息：无气泡、无头像，右对齐纯文本 + 细标签；
  // 正文与助手回复同宽，视觉上让"对话记录"更线性、更安静。
  return (
    <div className="flex justify-end my-1.5 group" data-entrance={turn != null ? `u${turn}` : undefined}>
      <div className={`flex items-start gap-2 max-w-[85%] ${compact ? "min-w-[120px]" : "min-w-[160px]"}`}>
        <div className="flex-1 min-w-0">
          <div
            className={`${compact ? "text-[13px]" : "text-[14px]"} text-fg leading-relaxed`}
          >
            {textParts.map((part, i) => {
              if (part.type === "text") {
                if (longText && !msgOpen) return <span key={i} className="line-clamp-3">{part.value}</span>;
                return <span key={i}>{part.value}</span>;
              }
              return <InlineAttachment key={i} path={part.value} />;
            })}
            {longText && (
              <button
                type="button"
                className="block mt-1 px-0 py-0.5 border-0 bg-transparent text-fg-faint/60 text-[10.5px] cursor-pointer hover:text-fg transition-colors"
                onClick={() => setMsgOpen((v) => !v)}
                title={msgOpen ? t("msg.collapseLongTitle") : t("msg.expandLongTitle")}
              >
                {msgOpen ? t("msg.collapseLong") : t("msg.expandLong")}
                <ChevronDown size={10} className={`inline-block ml-0.5 -mt-px transition-transform duration-200 ${msgOpen ? "rotate-180" : ""}`} aria-hidden />
              </button>
            )}
          </div>
          {canRewind && (
            <div className="flex justify-end mt-0.5">
              <button
                className="opacity-0 group-hover:opacity-100 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/50 text-[10px] cursor-pointer hover:text-fg transition-opacity"
                onClick={onToggle}
                title={t("rewind.label")}
              >
                <Rollback size={10} className="inline-block -mt-px mr-0.5" aria-hidden />
                {t("msg.rewindText")}
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
      </div>
    </div>
  );
});

// ── AssistantMessage ──────────────────────────────────────────────────────

export const AssistantMessage = memo(function AssistantMessage({
  item,
  onCapture,
  turnNo,
  deliverTail,
}: {
  item: AssistantItem;
  onCollapse?: () => void;
  onCapture?: (solution: string) => void;
  /** 本条消息所属轮次（Transcript 的 0-based 用户消息序号）；轮外缺省。
   *  原样下传 DeliverableCards 做登记表「本轮」条目匹配。 */
  turnNo?: number;
  /** 轮尾段才合并登记-only 交付卡（同轮去重）；缺省 true 保持独立渲染语义。 */
  deliverTail?: boolean;
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
  // 复制正文反馈（Codex 式消息操作；复制成功后短暂显示「已复制」）
  const [copied, setCopied] = useState(false);
  useEffect(() => { setCopied(false); }, [item.id]);
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
  // v4.26：带 subagentRef 的 assistant 消息 = 子代理最终答复回投主回合，
  // 加「子代理」小徽标区分来源（Codex "Report sub-agent activity on parent
  // turns"）；ref 全文放 title。字段缺省不渲染，行为与现状一致。
  const subagentRef = item.subagentRef;
  return (
    <div className="flex justify-start my-2" data-entrance={item.id}>
      <div className="flex-1 min-w-0">
          {/* 子代理来源徽标 */}
          {subagentRef && (
            <div className="mb-1">
              <span
                data-testid="subagent-badge"
                title={t("msg.subagentBadgeTitle", { ref: subagentRef })}
                className="inline-flex items-center gap-1 rounded-full border border-accent/25 bg-accent/10 px-1.5 py-px text-[10px] font-medium text-accent align-middle"
              >
                <Bot size={10} className="shrink-0" aria-hidden />
                {t("msg.subagentBadge")}
              </span>
            </div>
          )}
          {/* 推理区 */}
          {item.reasoning && (
            <div className="mb-1.5">
              <button
                type="button"
                className={`flex items-center gap-1.5 w-full px-2.5 py-1 rounded-lg transition-colors ${
                  reasoningOpen ? "bg-accent/[0.04]" : "hover:bg-(color:--md-sys-color-surface-container-high)"
                } text-fg-faint text-[11px] cursor-pointer`}
                onClick={toggleReasoning}
                aria-expanded={reasoningOpen}
              >
                <span className={`shrink-0 w-0.5 self-stretch rounded-full transition-colors ${reasoningRunning ? "bg-accent animate-pulse" : "bg-transparent"}`} />
                <Brain size={12} className="flex-shrink-0" />
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
              <MemoMarkdown text={item.text} streaming={streaming} genuiKey={item.id} />
              {/* 流式光标：生成中闪烁的细光标（reduced-motion 下全局动画关停 → 静态） */}
              {streaming && (
                <span aria-hidden className="inline-block w-[2px] h-[1.05em] align-text-bottom ml-0.5 rounded-full bg-accent/80 animate-pulse" />
              )}
            </div>
          )}
          {/* 交付物附件卡片：正文中的文件引用 + 权威登记表本轮条目，渲染成可点击预览卡片 */}
          {item.text && <DeliverableCards text={item.text} turnNo={turnNo} mergeRegistry={deliverTail} />}

          {/* 消息操作：复制正文（常驻，Codex 式）+ 沉淀为技能（成功对话可复用） */}
          {item.text && !streaming && (
            <div className="mt-1 flex items-center gap-1">
              <button
                type="button"
                className="inline-flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/50 text-[10.5px] cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                onClick={() => {
                  void navigator.clipboard.writeText(item.text);
                  setCopied(true);
                  window.setTimeout(() => setCopied(false), 1500);
                }}
                title={t("msg.copy")}
              >
                {copied ? <Check size={11} className="text-ok" /> : <Copy size={11} />}
                {copied ? t("msg.copied") : t("msg.copy")}
              </button>
              {onCapture && (
                <button
                  type="button"
                  className="inline-flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/50 text-[10.5px] cursor-pointer hover:text-accent hover:bg-bg-soft hover:text-fg transition-colors"
                  onClick={() => onCapture(item.text)}
                  title={t("msg.captureSkillTitle")}
                >
                  <Wand2 size={11} />
                  {t("msg.captureSkill")}
                </button>
              )}
            </div>
          )}
        </div>
    </div>
  );
});
