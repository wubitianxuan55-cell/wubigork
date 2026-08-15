// Composer（T6-10.1 巨型组件拆分后的编排层，行为零变化）
// 文本/排队/键盘/高度状态 + 跨 hook 装配；菜单/附件/工作区逻辑见 gaea/hooks/，
// 纯展示组件见 components/composer/。
import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, ClipboardEvent, DragEvent, KeyboardEvent, PointerEvent as ReactPointerEvent } from "react";
import { useT } from "../lib/i18n";
import { clearLayoutSize, loadOptionalLayoutSize, saveLayoutSize } from "../lib/layoutPreferences";
import { applyTableConversion, detectTableBlock } from "../lib/tableData";
import { useComposerInsertStore } from "../lib/store";
import { SlashMenu } from "./SlashMenu";
import { ArgMenu } from "./ArgMenu";
import { FileMenu } from "./FileMenu";
import { usePasteBlocks, PasteBlocksUI } from "./PasteManager";
import { ScreenCropOverlay } from "./ScreenCropOverlay";
import { useComposerMenus } from "../hooks/useComposerMenus";
import { useComposerAttachments } from "../hooks/useComposerAttachments";
import { useComposerWorkspace } from "../hooks/useComposerWorkspace";
import { ComposerDragOverlay } from "./composer/ComposerDragOverlay";
import { ComposerWorkspaceMenu } from "./composer/ComposerWorkspaceMenu";
import { ComposerAttachmentBar } from "./composer/ComposerAttachmentBar";
import { ComposerQueueList } from "./composer/ComposerQueueList";
import { ComposerTableBanner } from "./composer/ComposerTableBanner";
import { ComposerInputRow } from "./composer/ComposerInputRow";
import { ComposerToolbar } from "./composer/ComposerToolbar";

const COMPOSER_MIN_HEIGHT = 86;
const COMPOSER_MAX_HEIGHT = 360;
const COMPOSER_MAX_VIEWPORT_RATIO = 0.4;
const INPUT_HISTORY_KEY = "gaea.inputHistory";
const MAX_INPUT_HISTORY = 50;

function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => { const t = setTimeout(() => setDebounced(value), delay); return () => clearTimeout(t); }, [value, delay]);
  return debounced;
}

function composerMaxHeight(): number {
  if (typeof window === "undefined") return COMPOSER_MAX_HEIGHT;
  return Math.max(COMPOSER_MIN_HEIGHT, Math.min(COMPOSER_MAX_HEIGHT, Math.floor(window.innerHeight * COMPOSER_MAX_VIEWPORT_RATIO)));
}
function clampComposerHeight(h: number): number {
  return Math.min(Math.max(Math.round(h), COMPOSER_MIN_HEIGHT), composerMaxHeight());
}
function loadComposerHeight(): number | null {
  return loadOptionalLayoutSize("composerHeight", clampComposerHeight);
}

export function Composer({
  running, cwd, onSend, onCancel, permLevel, onSetPermLevel, onPickFolder, disabled,
}: {
  running: boolean; cwd?: string;
  onSend: (displayText: string, submitText?: string) => void;
  onCancel: () => string | undefined; permLevel?: string; onSetPermLevel?: (p: "ask" | "auto" | "yolo") => void;
  onPickFolder: (path?: string) => Promise<string>; disabled?: boolean;
}) {
  const t = useT();
  const [text, setText] = useState("");
  // 粘贴表格即数据：识别 CSV/TSV 表格块，发送时转 Markdown 表格（可关）
  const tableInfo = useMemo(() => detectTableBlock(text), [text]);
  const [tableMode, setTableMode] = useState(true);
  const debouncedText = useDebounce(text, 80);
  const taRef = useRef<HTMLTextAreaElement>(null);
  const paste = usePasteBlocks(text, setText, taRef);
  const composerCardRef = useRef<HTMLDivElement>(null);
  const wasRunning = useRef(running);

  // ── 附件/粘贴/截图/识图/OCR（useComposerAttachments） ──
  const att = useComposerAttachments({ text, setText, taRef, running, onSend });
  const {
    attachments, setAttachments, pendingPaste, cropSrc, setCropSrc, captureBusy,
    recognizingPath, ocrPath, attachDroppedFiles, handlePickFiles,
    handleScreenshot, handleCropConfirm, handleRecognize, handleOCRText,
  } = att;

  // ── 工作区切换菜单（useComposerWorkspace） ──
  const ws = useComposerWorkspace({ cwd, onPickFolder });
  const {
    workspaceName, workspaceMenuOpen, setWorkspaceMenuOpen,
    workspaceQuery, setWorkspaceQuery, filteredWorkspaces,
    workspaceAnchorRef, workspaceMenuRef, chooseWorkspace,
  } = ws;

  // ── / 命令、/ 参数、@ 文件菜单（useComposerMenus） ──
  const setTextCaretEnd = (next: string) => {
    setText(next);
    requestAnimationFrame(() => { const ta = taRef.current; if (ta) { ta.focus(); ta.selectionStart = ta.selectionEnd = next.length; } });
  };
  const menus = useComposerMenus({ text, debouncedText, setTextCaretEnd });
  const {
    slashMatches, argRes, atItems, menuMode, menuCount,
    active, setActive, setDismissed, pickCommand, pickArg, pickEntry, pickActive,
  } = menus;

  // 排队
  const queueRef = useRef<string[]>([]);
  const [queueLen, setQueueLen] = useState(0);
  const [queueDisplay, setQueueDisplay] = useState<string[]>([]); // 可视化队列列表
  const correctionRef = useRef<string | null>(null);               // 纠正模式待发送文本
  const onSendRef = useRef(onSend);
  onSendRef.current = onSend;
  // Shift 键追踪（用于发送按钮提示 / 纠正发送）
  const [shiftHeld, setShiftHeld] = useState(false);
  useEffect(() => {
    const onDown = (e: globalThis.KeyboardEvent) => { if (e.key === "Shift") setShiftHeld(true); };
    const onUp = (e: globalThis.KeyboardEvent) => { if (e.key === "Shift") setShiftHeld(false); };
    window.addEventListener("keydown", onDown);
    window.addEventListener("keyup", onUp);
    return () => { window.removeEventListener("keydown", onDown); window.removeEventListener("keyup", onUp); };
  }, []);
  useEffect(() => {
    // 纠正模式优先：取消后 running→false，立即发送纠正文本
    if (!running && correctionRef.current) {
      const correction = correctionRef.current;
      correctionRef.current = null;
      onSendRef.current(correction, correction);
      return;
    }
    if (!running && queueRef.current.length > 0) {
      const timer = setTimeout(() => {
        const next = queueRef.current.shift()!;
        setQueueLen(queueRef.current.length);
        setQueueDisplay([...queueRef.current]);
        onSendRef.current(next, next);
      }, 50);
      return () => clearTimeout(timer);
    }
  }, [running]);

  useEffect(() => {
    if (wasRunning.current && !running && text.trim() === "") {
      paste.clearBlocks();
    }
    wasRunning.current = running;
  }, [running, text]);

  // 右侧资料面板「一键 @ 引用」→ 插入输入框
  const pendingAt = useComposerInsertStore((s) => s.pendingAt);
  useEffect(() => {
    if (!pendingAt) return;
    const at = useComposerInsertStore.getState().consumeAt();
    if (!at) return;
    setText((prev) => prev + (prev && !prev.endsWith(" ") ? " " : "") + "@" + at + " ");
    requestAnimationFrame(() => { taRef.current?.focus(); });
  }, [pendingAt]);
  // 资料面板「摘要后引用」：把摘要文本插入输入框（可编辑后再发送）。
  const pendingText = useComposerInsertStore((s) => s.pendingText);
  useEffect(() => {
    if (!pendingText) return;
    const text = useComposerInsertStore.getState().consumeText();
    if (!text) return;
    setText((prev) => prev + (prev ? "\n\n" : "") + text);
    requestAnimationFrame(() => { taRef.current?.focus(); });
  }, [pendingText]);

  // ── 拖放 ──
  const [dragOver, setDragOver] = useState(false);

  // 粘贴处理
  const onPaste = (e: ClipboardEvent<HTMLTextAreaElement>) => {
    const files = Array.from(e.clipboardData.files);
    if (files.length > 0) { e.preventDefault(); void attachDroppedFiles(files); return; }
    // 剪贴板图片（如截图/复制的网页图片）：转为图片附件上下文，而不是静默丢弃
    const imageItem = Array.from(e.clipboardData.items).find(
      (it) => it.type.startsWith("image/")
    );
    if (imageItem) {
      e.preventDefault();
      const file = imageItem.getAsFile();
      if (file) void attachDroppedFiles([file]);
      return;
    }
    paste.onPaste(e);
  };

  const onDrop = (e: DragEvent<HTMLDivElement>) => {
    const files = Array.from(e.dataTransfer.files);
    if (files.length === 0) return;
    e.preventDefault(); setDragOver(false); void attachDroppedFiles(files);
  };
  const onDragOver = (e: DragEvent<HTMLDivElement>) => {
    if (!Array.from(e.dataTransfer.items).some((it) => it.kind === "file")) return;
    e.preventDefault(); setDragOver(true);
  };
  const onDragLeave = () => setDragOver(false);

  const submit = () => {
    if (disabled) return;
    const converted = tableMode ? applyTableConversion(text, true) : text;
    const tTrim = converted.trim();
    if ((!tTrim && attachments.length === 0) || pendingPaste > 0) return;
    const refs = attachments.map((a) => `@${a.path}`).join(" ");
    const displayText = [tTrim, refs].filter(Boolean).join(tTrim && refs ? " " : "");
    const submitText = [paste.expandBlocks(tTrim), refs].filter(Boolean).join(tTrim && refs ? " " : "");
    if (displayText.trim()) {
      try {
        const history = JSON.parse(sessionStorage.getItem(INPUT_HISTORY_KEY) || "[]") as string[];
        history.unshift(displayText); sessionStorage.setItem(INPUT_HISTORY_KEY, JSON.stringify(history.slice(0, MAX_INPUT_HISTORY)));
      } catch {}
    }
    setHistoryIndex(-1);
    if (running) {
      queueRef.current.push(submitText);
      setQueueLen(queueRef.current.length);
      setQueueDisplay([...queueRef.current]);
      setText("");
      setAttachments([]);
      return;
    }
    onSend(displayText, submitText); setText(""); setAttachments([]);
  };

  const handleCancel = () => {
    queueRef.current = [];
    setQueueLen(0);
    setQueueDisplay([]);
    const restored = onCancel();
    if (typeof restored === "string") setTextCaretEnd(restored);
  };

  // 逐条取消排队
  const cancelQueueItem = (index: number) => {
    queueRef.current.splice(index, 1);
    setQueueLen(queueRef.current.length);
    setQueueDisplay([...queueRef.current]);
  };

  // ── 高度调整 ──
  const [composerHeight, setComposerHeight] = useState<number | null>(loadComposerHeight);
  const [composerResizing, setComposerResizing] = useState(false);
  useEffect(() => {
    const onResize = () => setComposerHeight((h) => (h === null ? null : clampComposerHeight(h)));
    window.addEventListener("resize", onResize); return () => window.removeEventListener("resize", onResize);
  }, []);
  const saveComposerHeight = (h: number) => saveLayoutSize("composerHeight", h, clampComposerHeight);
  const resetComposerHeight = () => { setComposerHeight(null); clearLayoutSize("composerHeight"); };
  const onComposerResizeStart = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (e.button !== 0) return;
    const card = composerCardRef.current; if (!card) return;
    e.preventDefault();
    const startY = e.clientY;
    const startHeight = composerHeight ?? card.getBoundingClientRect().height;
    let nextHeight = clampComposerHeight(startHeight);
    let moved = false;
    setComposerResizing(true); document.body.classList.add("composer-resizing");
    const onMove = (ev: PointerEvent) => { moved = true; nextHeight = clampComposerHeight(startHeight + startY - ev.clientY); setComposerHeight(nextHeight); };
    const onUp = () => { setComposerResizing(false); document.body.classList.remove("composer-resizing"); if (moved) saveComposerHeight(nextHeight); document.removeEventListener("pointermove", onMove); document.removeEventListener("pointerup", onUp); document.removeEventListener("pointercancel", onUp); };
    document.addEventListener("pointermove", onMove); document.addEventListener("pointerup", onUp); document.addEventListener("pointercancel", onUp);
  };

  // ── 历史 ──
  const [historyIndex, setHistoryIndex] = useState(-1);
  const historyDraft = useRef("");
  const navigateHistory = (dir: 1 | -1) => {
    try {
      const history: string[] = JSON.parse(sessionStorage.getItem(INPUT_HISTORY_KEY) || "[]");
      if (history.length === 0) return;
      if (historyIndex === -1) historyDraft.current = text;
      const next = Math.max(-1, Math.min(history.length - 1, historyIndex + dir));
      setHistoryIndex(next); setText(next === -1 ? historyDraft.current : history[next] || "");
    } catch {}
  };

  // ── 键盘处理 ──
  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    const composing = e.nativeEvent.isComposing;
    if (menuMode && !composing) {
      if (e.key === "ArrowDown") { e.preventDefault(); setActive((i) => (i + 1) % menuCount); return; }
      if (e.key === "ArrowUp") { e.preventDefault(); setActive((i) => (i - 1 + menuCount) % menuCount); return; }
      if (e.key === "Enter" || e.key === "Tab") { e.preventDefault(); pickActive(); return; }
      if (e.key === "Escape") { e.preventDefault(); setDismissed(true); return; }
    }
    if (!menuMode && !composing) {
      if (e.key === "ArrowUp" && text === "") { e.preventDefault(); navigateHistory(1); return; }
      if (e.key === "ArrowDown" && historyIndex >= 0) { e.preventDefault(); navigateHistory(-1); return; }
      if (e.key !== "ArrowUp" && e.key !== "ArrowDown" && historyIndex >= 0) setHistoryIndex(-1);
    }
    if (e.key === "Enter" && e.shiftKey && !composing) {
      // 纠正模式：Shift+Enter → 清空队列 + 取消当前轮次 + 立即发送新文本
      e.preventDefault();
      if (disabled) return;
      const converted = tableMode ? applyTableConversion(text, true) : text;
      const tTrim = converted.trim();
      if ((!tTrim && attachments.length === 0) || pendingPaste > 0) return;
      const refs = attachments.map((a) => `@${a.path}`).join(" ");
      const displayText = [tTrim, refs].filter(Boolean).join(tTrim && refs ? " " : "");
      const submitText = [paste.expandBlocks(tTrim), refs].filter(Boolean).join(tTrim && refs ? " " : "");
      if (displayText.trim()) {
        try {
          const history = JSON.parse(sessionStorage.getItem(INPUT_HISTORY_KEY) || "[]") as string[];
          history.unshift(displayText);
          sessionStorage.setItem(INPUT_HISTORY_KEY, JSON.stringify(history.slice(0, MAX_INPUT_HISTORY)));
        } catch {}
      }
      setHistoryIndex(-1);
      queueRef.current = [];
      setQueueLen(0);
      setQueueDisplay([]);
      onCancel();
      correctionRef.current = submitText;
      setText("");
      setAttachments([]);
      return;
    }
    if (e.key === "Enter" && !e.shiftKey && !composing) { e.preventDefault(); submit(); }
    if (e.key === "Escape" && running) { e.preventDefault(); handleCancel(); }
  };

  const composerCardStyle = composerHeight === null ? undefined : ({ "--composer-height": `${composerHeight}px` } as CSSProperties);

  // ── 项目感知 placeholder ──
  const placeholderText = useMemo(() => {
    if (disabled) return t("common.loading");
    if (running && queueLen > 0) return `排队中 (${queueLen})…`;
    if (running) return t("composer.placeholderRunning");
    if (cwd && workspaceName) return `在 ${workspaceName}/ 中提问…`;
    return t("composer.placeholder");
  }, [disabled, running, queueLen, cwd, workspaceName, t]);

  return (
    <div className="relative max-w-[--maxw] mx-auto">
      {/* ── 拖放指示器 ── */}
      <ComposerDragOverlay show={dragOver} />

      {/* ── 工作区切换菜单 ── */}
      {workspaceMenuOpen && cwd && (
        <ComposerWorkspaceMenu
          menuRef={workspaceMenuRef}
          query={workspaceQuery}
          onQueryChange={setWorkspaceQuery}
          workspaces={filteredWorkspaces}
          onChoose={chooseWorkspace}
          onClose={() => setWorkspaceMenuOpen(false)}
        />
      )}

      {/* ── 菜单（命令/参数/文件）── */}
      {menuMode === "slash" && <SlashMenu items={slashMatches} activeIndex={active} onPick={pickCommand} onHover={setActive} />}
      {menuMode === "slasharg" && argRes && <ArgMenu items={argRes.items} activeIndex={active} onPick={pickArg} onHover={setActive} />}
      {menuMode === "at" && <FileMenu items={atItems} activeIndex={active} onPick={pickEntry} onHover={setActive} />}

      {/* ── 附件预览 ── */}
      <ComposerAttachmentBar
        attachments={attachments}
        running={running}
        recognizingPath={recognizingPath}
        ocrPath={ocrPath}
        onRecognize={handleRecognize}
        onOCRText={handleOCRText}
        onRemove={(path) => setAttachments((prev) => prev.filter((x) => x.path !== path))}
      />

      {/* ── 粘贴块 ── */}
      <PasteBlocksUI
        blocks={paste.activePastedBlocks}
        openLabels={paste.openPastedLabels}
        onTogglePreview={paste.togglePreview}
        onExpand={paste.expandBlock}
        onRemove={paste.removeBlock}
      />

      {/* ── 排队列表 ── */}
      {running && <ComposerQueueList queueDisplay={queueDisplay} onCancelItem={cancelQueueItem} />}

      {/* ── 输入卡片（Luminous Glass：玻璃底 + 顶部 1px 高光线 + 聚焦收敛光晕） ── */}
      <div
        className={`relative border border-border-soft/80 bg-bg-elev/70 backdrop-blur-xl backdrop-saturate-150 rounded-2xl overflow-hidden shadow-[inset_0_1px_0_color-mix(in_srgb,var(--fg)_6%,transparent)] transition-[border-color,box-shadow,background] duration-[var(--dur-base)] focus-within:border-accent/30 focus-within:shadow-[inset_0_1px_0_color-mix(in_srgb,var(--fg)_6%,transparent),0_0_0_1px_var(--accent-soft),0_0_22px_color-mix(in_srgb,var(--accent)_10%,transparent),var(--ds-shadow-composer)] ${composerHeight !== null ? "flex flex-col" : ""} ${composerResizing ? "cursor-ns-resize" : ""}`}
        style={{ ...(composerHeight !== null ? { height: "var(--composer-height)" } : {}), ...composerCardStyle }}
        ref={composerCardRef}
      >
        {/* 顶部 1px 高光线（v3-edge 渐变） */}
        <div aria-hidden className="pointer-events-none absolute top-0 left-0 right-0 h-px z-[4]" style={{ background: "var(--v3-edge)" }} />
        {/* 拖拽调整大小把手 */}
        <div
          className="absolute top-0 left-[14px] right-[14px] z-[5] h-2 cursor-ns-resize no-drag touch-none"
          onPointerDown={onComposerResizeStart}
          onDoubleClick={resetComposerHeight}
        />

        {/* 粘贴表格即数据：检测到表格块时提示，可关 */}
        {tableInfo && !disabled && (
          <ComposerTableBanner
            rows={tableInfo.rows}
            cols={tableInfo.cols}
            tableMode={tableMode}
            onTableModeChange={setTableMode}
          />
        )}

        {/* 主输入行 */}
        <ComposerInputRow
          taRef={taRef}
          text={text}
          onTextChange={setText}
          onPaste={onPaste}
          onKeyDown={onKeyDown}
          placeholder={placeholderText}
          disabled={disabled}
          running={running}
          composerHeightFixed={composerHeight !== null}
          dragOver={dragOver}
          shiftHeld={shiftHeld}
          queueLen={queueLen}
          pendingPaste={pendingPaste}
          attachmentsCount={attachments.length}
          onDrop={onDrop}
          onDragOver={onDragOver}
          onDragLeave={onDragLeave}
          onStop={handleCancel}
          onSubmit={submit}
        />

        {/* 底部工具栏 */}
        <ComposerToolbar
          cwd={cwd}
          workspaceName={workspaceName}
          workspaceMenuOpen={workspaceMenuOpen}
          onToggleWorkspaceMenu={() => { if (!running) setWorkspaceMenuOpen((o) => !o) }}
          workspaceAnchorRef={workspaceAnchorRef}
          running={running}
          pendingPaste={pendingPaste}
          captureBusy={captureBusy}
          onPickFiles={() => void handlePickFiles()}
          onScreenshot={() => void handleScreenshot()}
          permLevel={permLevel}
          onSetPermLevel={onSetPermLevel}
        />
      </div>

      {/* 截图裁剪浮层 */}
      {cropSrc && (
        <ScreenCropOverlay
          src={cropSrc}
          onCancel={() => setCropSrc(null)}
          onConfirm={(d) => void handleCropConfirm(d)}
        />
      )}
    </div>
  );
}
