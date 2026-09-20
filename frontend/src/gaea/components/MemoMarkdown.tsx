import { memo, useRef, useState, useEffect, useMemo } from "react";
import { Markdown } from "./Markdown";
import { escapeHtml, htmlFileLinks } from "../lib/fileLinks";
import { sanitizeHtml } from "../lib/sanitize";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";

interface MemoMarkdownProps {
  text: string;
  streaming: boolean;
  /** 消息级源 key（透传 Markdown → genui 状态 key）。 */
  genuiKey?: string;
}

/**
 * findStableCut — 找到"稳定"切分点。
 *
 * 规则（按优先级）：
 *   1. 若候选切点落在未闭合的代码围栏内（fence body 含空行时原 suffix 扫描
 *      的盲区——计数偶数误判可切，把代码块拦腰切断），切点回退到该围栏打开
 *      行之前（v4.368：改为全量行扫描 fence 状态）。
 *   2. 否则在最后一个 \n\n 处切分。
 *   3. 没有 \n\n 则全部视为不稳定。
 *
 * 返回 [stablePrefix, unstableSuffix]。导出供聊天流式行（ChatRow）复用同一
 * 切分语义（v4.368 流式分段渲染）。
 */
export function findStableCut(text: string): [string, string] {
  const lastGap = text.lastIndexOf("\n\n");
  if (lastGap < 0) return ["", text];

  // v4.368：全量行扫描候选切点之前的行首 ``` 标记，跟踪 fence 状态——
  // 原实现只扫切点之后的 suffix，fence body 含空行时计数为偶数误判「可切」，
  // 把代码块拦腰切断（盲区）。现若切点落在未闭合 fence 内，回退切点到该
  // fence 打开行之前（切分只会更保守，稳定段永不含悬挂 fence）。导出供
  // 聊天流式行（ChatRow）复用同一切分语义（v4.368 流式分段渲染）。
  const FENCE = "```";
  let inFence = false;
  let fenceOpenStart = -1;
  let pos = 0;
  while (pos <= lastGap) {
    const nl = text.indexOf("\n", pos);
    const lineEnd = nl < 0 ? text.length : nl;
    const line = text.slice(pos, lineEnd);
    if (line.trimStart().startsWith(FENCE)) {
      inFence = !inFence;
      fenceOpenStart = inFence ? pos : -1;
    }
    if (nl < 0 || nl >= lastGap) break;
    pos = nl + 1;
  }
  let cut = lastGap + 2;
  if (inFence && fenceOpenStart >= 0) {
    const preFenceNL = text.lastIndexOf("\n\n", fenceOpenStart - 1);
    cut = preFenceNL >= 0 ? preFenceNL + 2 : 0;
  }
  return [text.slice(0, cut), text.slice(cut)];
}

/**
 * renderPending — 对不稳定尾部做简单 HTML 渲染。
 *
 * 处理标题、列表、引用、代码块，与旧版类似但只用于最后的不稳定部分。
 */
function renderPending(text: string): string {
  const lines = text.split("\n");
  let inFence = false;
  const out: string[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const isLast = i === lines.length - 1;

    if (line.startsWith("```")) {
      inFence = !inFence;
      out.push(`<span class="text-fg-faint font-mono text-[90%]">${escapeHtml(line)}</span>`);
    } else if (inFence) {
      out.push(`<span class="font-mono text-[90%]">${escapeHtml(line)}</span>`);
    } else if (/^#{1,4}\s/.test(line)) {
      out.push(`<span class="font-bold">${htmlFileLinks(line)}</span>`);
    } else if (/^[-*+]\s/.test(line)) {
      out.push(`<span class="text-fg-dim">  · ${htmlFileLinks(line.slice(2))}</span>`);
    } else if (/^\d+\.\s/.test(line)) {
      out.push(`<span class="text-fg-dim">  ${htmlFileLinks(line)}</span>`);
    } else if (/^>\s/.test(line)) {
      out.push(`<span class="text-fg-faint">│ ${htmlFileLinks(line.slice(2))}</span>`);
    } else if (line.trim() === "" && !isLast) {
      out.push("");
    } else {
      out.push(htmlFileLinks(line));
    }
  }

  return out.join("\n");
}

/**
 * useProgressiveMarkdown — 渐进式 Markdown。
 *
 * 找到稳定切分点后，前缀用 Markdown 渲染，后缀用简单 HTML。
 */
function useProgressiveMarkdown(text: string): { stable: string; pending: string } {
  return useMemo(() => {
    const [s, p] = findStableCut(text);
    return { stable: s, pending: p };
  }, [text]);
}

/**
 * MemoMarkdown — 流式友好的 Markdown 渲染器。
 *
 * 流式期间：找到稳定段落边界（\n\n），前缀用完整 Markdown 渲染，
 * 未完成尾部用简单样式。流式结束后全量 Markdown 渲染。
 */
export const MemoMarkdown = memo(function MemoMarkdown({ text, streaming, genuiKey }: MemoMarkdownProps) {
  const openFilePreview = openPaneFileOrPreview;
  // RAF 节流：每帧最多更新一次
  const [visible, setVisible] = useState(text);
  const rafRef = useRef(0);

  useEffect(() => {
    if (!streaming) {
      setVisible(text);
      return;
    }
    cancelAnimationFrame(rafRef.current);
    rafRef.current = requestAnimationFrame(() => setVisible(text));
    return () => cancelAnimationFrame(rafRef.current);
  }, [text, streaming]);

  const { stable, pending } = useProgressiveMarkdown(visible);

  // 流式结束：全量 Markdown
  if (!streaming) {
    return (
      <div className="break-words overflow-wrap-break-word">
        <Markdown text={text || ""} genuiKey={genuiKey} />
      </div>
    );
  }

  // 流式中：稳定部分 Markdown + 不稳定部分简单样式 + 闪烁光标
  return (
    <div className="break-words overflow-wrap-break-word">
      {stable && (
        <div className="md text-[14px] leading-relaxed">
          <Markdown text={stable} genuiKey={genuiKey} />
        </div>
      )}
      {pending && (
        <div
          className="!font-sans whitespace-pre-wrap !bg-transparent !p-0 !m-0 !text-[inherit] !border-0 leading-relaxed text-[14px]"
          dangerouslySetInnerHTML={{ __html: sanitizeHtml(renderPending(pending)) }}
          onClick={(e) => {
            const btn = (e.target as HTMLElement).closest?.("button[data-file-preview]");
            if (btn instanceof HTMLElement) {
              const path = btn.getAttribute("data-file-preview");
              if (path) openFilePreview(path);
            }
          }}
        />
      )}
      <span
        className="inline-block w-[2px] h-[1em] bg-accent align-middle ml-px animate-pulse"
        aria-hidden
      />
    </div>
  );
}, (prev, next) => prev.text === next.text && prev.streaming === next.streaming);
