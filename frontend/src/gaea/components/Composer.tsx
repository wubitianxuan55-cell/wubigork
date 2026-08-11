import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, ClipboardEvent, DragEvent, KeyboardEvent, PointerEvent as ReactPointerEvent } from "react";
import { Modal } from "antd";
import { ArrowUp, Camera, Check, ChevronDown, Eye, FileText, FolderGit2, FolderPlus, Loader, Paperclip, Search, Square, Table, X, Zap } from "../icons";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import { clearLayoutSize, loadOptionalLayoutSize, saveLayoutSize } from "../lib/layoutPreferences";
import { applyTableConversion, detectTableBlock } from "../lib/tableData";
import type { CommandInfo, DirEntry, FileSearchHit, SlashArgItem, SlashArgsResult, WorkspaceView } from "../lib/types";
import { useToast } from "./Toast";
import { useComposerInsertStore } from "../lib/store";
import { SlashMenu } from "./SlashMenu";
import { ArgMenu } from "./ArgMenu";
import { FileMenu, type AtEntry } from "./FileMenu";
import { usePasteBlocks, PasteBlocksUI } from "./PasteManager";
import { ScreenCropOverlay } from "./ScreenCropOverlay";

interface Attachment { path: string; previewUrl: string; type: "image" | "file"; }


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
  const toast = useToast();
  const [text, setText] = useState("");
  // 粘贴表格即数据：识别 CSV/TSV 表格块，发送时转 Markdown 表格（可关）
  const tableInfo = useMemo(() => detectTableBlock(text), [text]);
  const [tableMode, setTableMode] = useState(true);
  const debouncedText = useDebounce(text, 80);
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [pendingPaste, setPendingPaste] = useState(0);
  const [cropSrc, setCropSrc] = useState<string | null>(null);
  const [captureBusy, setCaptureBusy] = useState(false);
  const [recognizingPath, setRecognizingPath] = useState<string | null>(null);
  const [ocrPath, setOcrPath] = useState<string | null>(null);
  const [active, setActive] = useState(0);
  const [dismissed, setDismissed] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const [workspaceMenuOpen, setWorkspaceMenuOpen] = useState(false);
  const [workspaceQuery, setWorkspaceQuery] = useState("");
  const [workspaces, setWorkspaces] = useState<WorkspaceView[]>([]);
  const [composerHeight, setComposerHeight] = useState<number | null>(loadComposerHeight);
  const [composerResizing, setComposerResizing] = useState(false);
  const taRef = useRef<HTMLTextAreaElement>(null);
  const paste = usePasteBlocks(text, setText, taRef);
  const composerCardRef = useRef<HTMLDivElement>(null);
  const workspaceAnchorRef = useRef<HTMLDivElement>(null);
  const workspaceMenuRef = useRef<HTMLDivElement>(null);
  const wasRunning = useRef(running);

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

  // ── / 命令 ──
  const [commands, setCommands] = useState<CommandInfo[]>([]);
  useEffect(() => { app.Commands().then(setCommands).catch(() => {}); }, []);
  const slashQuery = useMemo(() => (!text.startsWith("/") || /\s/.test(text) ? null : text.slice(1).toLowerCase()), [text]);
  const slashMatches = useMemo(() => (slashQuery === null ? [] : commands.filter((c) => c.name.toLowerCase().includes(slashQuery)).slice(0, 8)), [slashQuery, commands]);

  // ── 命令参数 ──
  const [argRes, setArgRes] = useState<SlashArgsResult | null>(null);
  useEffect(() => {
    if (!text.startsWith("/") || !/\s/.test(text)) { setArgRes(null); return; }
    let live = true;
    app.SlashArgs(text).then((r) => {
      if (!live) return;
      const useful = (r.items ?? []).filter((it) => text.slice(0, r.from) + it.insert !== text);
      setArgRes(useful.length > 0 ? { items: useful, from: r.from } : null); setActive(0);
    }).catch(() => {});
    return () => { live = false; };
  }, [text]);

  // ── @ 文件引用 ──
  const atRaw = useMemo(() => { const m = /(?:^|\s)@([^\s]*)$/.exec(debouncedText); return m ? m[1] : null; }, [debouncedText]);
  const atDir = useMemo(() => { if (atRaw === null) return ""; const s = atRaw.lastIndexOf("/"); return s >= 0 ? atRaw.slice(0, s + 1) : ""; }, [atRaw]);
  const atFrag = useMemo(() => { if (atRaw === null) return ""; const s = atRaw.lastIndexOf("/"); return (s >= 0 ? atRaw.slice(s + 1) : atRaw).toLowerCase(); }, [atRaw]);
  const [entries, setEntries] = useState<DirEntry[]>([]);
  const dirCache = useRef<Record<string, DirEntry[]>>({});
  useEffect(() => {
    if (atRaw === null) return;
    const cached = dirCache.current[atDir];
    if (cached) { setEntries(cached); return; }
    let live = true;
    app.ListDir(atDir).then((es) => { const list = es ?? []; dirCache.current[atDir] = list; if (live) setEntries(list); }).catch(() => {});
    return () => { live = false; };
  }, [atRaw === null, atDir]);
  // 工作区跨目录搜索（@ 引用增强：搜一下定位资料）
  const [atHits, setAtHits] = useState<FileSearchHit[]>([]);
  useEffect(() => {
    if (atRaw === null) { setAtHits([]); return; }
    let live = true;
    app.FileSearch(atFrag, 30).then((h) => { if (live) setAtHits(h ?? []); }).catch(() => {});
    return () => { live = false; };
  }, [atRaw, atFrag]);
  // 最近使用文件（@ 选择过的文件，本地持久化）
  const RECENT_AT_KEY = "gaea.atRecentFiles";
  const [recent, setRecent] = useState<AtEntry[]>(() => {
    try { return JSON.parse(localStorage.getItem(RECENT_AT_KEY) || "[]") as AtEntry[]; } catch { return []; }
  });
  useEffect(() => {
    try { localStorage.setItem(RECENT_AT_KEY, JSON.stringify(recent.slice(0, 20))); } catch {}
  }, [recent]);
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
  // 统一 @ 条目：目录内浏览（路径前缀）或 最近使用 + 工作区搜索 + 当前目录
  const atItems: AtEntry[] = useMemo(() => {
    if (atRaw === null) return [];
    const out: AtEntry[] = [];
    const seen = new Set<string>();
    const push = (e: AtEntry) => { if (!seen.has(e.path) && out.length < 12) { seen.add(e.path); out.push(e); } };
    if (atDir !== "") {
      for (const e of entries) {
        if (!e.name.toLowerCase().includes(atFrag)) continue;
        push({ path: atDir + e.name + (e.isDir ? "/" : ""), name: e.name, isDir: e.isDir });
      }
      return out;
    }
    for (const r of recent) if (r.name.toLowerCase().includes(atFrag)) push(r);
    for (const h of atHits) {
      if (!h.name.toLowerCase().includes(atFrag)) continue;
      push({ path: h.isDir ? h.path + "/" : h.path, name: h.name, isDir: h.isDir, size: h.size });
    }
    for (const e of entries) {
      if (!e.name.toLowerCase().includes(atFrag)) continue;
      push({ path: e.name + (e.isDir ? "/" : ""), name: e.name, isDir: e.isDir });
    }
    return out;
  }, [atRaw, atDir, atFrag, entries, atHits, recent]);

  // ── 菜单状态 ──
  const menuMode: "slash" | "slasharg" | "at" | null =
    slashMatches.length > 0 && !dismissed ? "slash"
    : argRes && argRes.items.length > 0 && !dismissed ? "slasharg"
    : atItems.length > 0 && !dismissed ? "at"
    : null;
  const menuCount = menuMode === "slash" ? slashMatches.length : menuMode === "slasharg" ? argRes!.items.length : menuMode === "at" ? atItems.length : 0;
  useEffect(() => { setActive(0); setDismissed(false); }, [slashQuery, atRaw]);

  const setTextCaretEnd = (next: string) => {
    setText(next);
    requestAnimationFrame(() => { const ta = taRef.current; if (ta) { ta.focus(); ta.selectionStart = ta.selectionEnd = next.length; } });
  };

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

  // 附加文件：图片走已有流程，非图片用 base64 上传
  const attachDroppedFiles = async (files: File[]) => {
    const images: File[] = [];
    const others: File[] = [];
    for (const f of files) {
      if (f.type.startsWith("image/")) images.push(f);
      else others.push(f);
    }
    // 大文件提示
    const bigFiles = files.filter((f) => f.size > 10 * 1024 * 1024);
    if (bigFiles.length > 0) {
      const names = bigFiles.map((f) => f.name).join(", ");
      // 原生 confirm 会同步阻塞 WebView2 主线程导致界面卡死，改用异步弹窗。
      const ok = await new Promise<boolean>((resolve) => {
        Modal.confirm({
          title: "大文件提示",
          content: `以下文件超过 10MB，可能上传较慢：\n${names}\n\n确定要继续吗？`,
          okText: "继续上传",
          cancelText: "取消",
          onOk: () => resolve(true),
          onCancel: () => resolve(false),
        });
      });
      if (!ok) return;
    }
    // 处理图片
    for (const file of images) {
      setPendingPaste((n) => n + 1);
      try {
        const dataUrl = await new Promise<string>((res, rej) => { const r = new FileReader(); r.onload = () => res(String(r.result)); r.onerror = () => rej(r.error); r.readAsDataURL(file); });
        const path = await app.SavePastedImage(dataUrl);
        const previewUrl = await app.AttachmentDataURL(path);
        setAttachments((prev) => [...prev, { path, previewUrl, type: "image" }]);
      } catch {} finally { setPendingPaste((n) => Math.max(0, n - 1)); }
    }
    // 处理非图片文件
    for (const file of others) {
      setPendingPaste((n) => n + 1);
      try {
        const buf = await new Promise<ArrayBuffer>((res, rej) => { const r = new FileReader(); r.onload = () => res(r.result as ArrayBuffer); r.onerror = () => rej(r.error); r.readAsArrayBuffer(file); });
        const bytes = new Uint8Array(buf);
        let bin = "";
        for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
        const b64 = btoa(bin);
        const path = await app.SaveAttachmentFile(file.name, b64);
        const atRef = `@${path}`;
        setText((prev) => prev + (prev.endsWith(" ") || prev === "" ? "" : " ") + atRef + " ");
        if (taRef.current) { taRef.current.focus(); taRef.current.selectionStart = taRef.current.selectionEnd = (text + " " + atRef + " ").length; }
      } catch {} finally { setPendingPaste((n) => Math.max(0, n - 1)); }
    }
  };

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

  // 导入文件：通过原生对话框选择文件
  const handlePickFiles = async () => {
    try {
      const files = await app.PickFiles();
      if (!files || files.length === 0) return;
      for (const f of files) {
        if (f.type === "image") {
          setAttachments((prev) => [...prev, { path: f.path, previewUrl: f.previewUrl ?? "", type: "image" as const }]);
        } else {
          const atRef = `@${f.path}`;
          setText((prev) => prev + (prev.endsWith(" ") || prev === "" ? "" : " ") + atRef + " ");
          if (taRef.current) { taRef.current.focus(); taRef.current.selectionStart = taRef.current.selectionEnd = (text + " " + atRef + " ").length; }
        }
      }
    } catch {
      // 静默处理（旧后端不支持）
    }
  };

  // ── 截图：整屏捕获 → 裁剪浮层 → 复用图片附件流程 ──
  const handleScreenshot = async () => {
    if (running || captureBusy) return;
    setCaptureBusy(true);
    try {
      const dataUrl = await app.CaptureScreen();
      setCropSrc(dataUrl);
    } catch (e: any) {
      toast.show(String(e?.message ?? e), "warn");
    } finally {
      setCaptureBusy(false);
    }
  };

  const handleCropConfirm = async (dataUrl: string) => {
    setCropSrc(null);
    setPendingPaste((n) => n + 1);
    try {
      const path = await app.SavePastedImage(dataUrl);
      const previewUrl = await app.AttachmentDataURL(path);
      setAttachments((prev) => [...prev, { path, previewUrl, type: "image" }]);
    } catch (e: any) {
      toast.show(String(e?.message ?? e), "warn");
    } finally {
      setPendingPaste((n) => Math.max(0, n - 1));
    }
  };

  // ── 识图：本地视觉模型识别附件图片，把结果作为用户消息发给助手 ──
  const handleRecognize = async (att: Attachment) => {
    if (running || recognizingPath) return;
    setRecognizingPath(att.path);
    try {
      const desc = await app.RecognizeImage(
        att.path,
        "请详细描述这张图片的内容，包括所有可见文字、布局和关键细节。",
      );
      const name = att.path.split(/[/\\]/).pop() || att.path;
      const msg = `【图片识图：${name}】\n${desc}`;
      setText("");
      setAttachments((prev) => prev.filter((x) => x.path !== att.path));
      onSendRef.current(msg, msg);
      toast.show("识别完成，已发送给助手");
    } catch (e: any) {
      toast.show(String(e?.message ?? e), "warn");
    } finally {
      setRecognizingPath(null);
    }
  };

  // ── 提取文字：本地 OvisOCR2 常驻服务识别图片中的文字，结果作为用户消息发给助手 ──
  const handleOCRText = async (att: Attachment) => {
    if (running || ocrPath) return;
    setOcrPath(att.path);
    try {
      const text = await app.OCRText(att.path);
      const name = att.path.split(/[/\\]/).pop() || att.path;
      const msg = `【图片文字提取：${name}】\n${text}`;
      setText("");
      setAttachments((prev) => prev.filter((x) => x.path !== att.path));
      onSendRef.current(msg, msg);
      toast.show("文字提取完成，已发送给助手");
    } catch (e: any) {
      toast.show(String(e?.message ?? e), "warn");
    } finally {
      setOcrPath(null);
    }
  };

  const pickCommand = (c: CommandInfo) => setTextCaretEnd("/" + c.name + " ");
  const pickEntry = (e: AtEntry) => {
    const atPos = text.length - (atRaw?.length ?? 0) - 1;
    const prefix = text.slice(0, atPos);
    if (e.isDir) {
      setTextCaretEnd(prefix + "@" + e.path);
      return;
    }
    setRecent((prev) => [{ path: e.path, name: e.name, isDir: false, size: e.size }, ...prev.filter((r) => r.path !== e.path)].slice(0, 20));
    setTextCaretEnd(prefix + "@" + e.path + " ");
  };
  const pickArg = (it: SlashArgItem) => { if (!argRes) return; setTextCaretEnd(text.slice(0, argRes.from) + it.insert); };
  const pickActive = () => {
    if (menuMode === "slash") pickCommand(slashMatches[active]);
    else if (menuMode === "slasharg" && argRes) pickArg(argRes.items[active]);
    else if (menuMode === "at") pickEntry(atItems[active]);
  };

  // ── 工作区菜单 ──
  const workspaceName = useMemo(() => { if (!cwd) return ""; const parts = cwd.split(/[/\\]/).filter(Boolean); return parts.length > 0 ? parts[parts.length - 1] : cwd; }, [cwd]);
  const loadWorkspaces = () => { app.ListWorkspaces().then(setWorkspaces).catch(() => setWorkspaces([])); };
  useEffect(() => { if (workspaceMenuOpen) loadWorkspaces(); }, [workspaceMenuOpen, cwd]);
  useEffect(() => {
    if (!workspaceMenuOpen) return;
    const close = (e: MouseEvent) => { const tgt = e.target as Node; if (workspaceAnchorRef.current?.contains(tgt) || workspaceMenuRef.current?.contains(tgt)) return; setWorkspaceMenuOpen(false); };
    document.addEventListener("mousedown", close); return () => document.removeEventListener("mousedown", close);
  }, [workspaceMenuOpen]);
  const filteredWorkspaces = useMemo(() => { const q = workspaceQuery.trim().toLowerCase(); if (!q) return workspaces; return workspaces.filter((w) => `${w.name} ${w.path}`.toLowerCase().includes(q)); }, [workspaceQuery, workspaces]);
  const chooseWorkspace = async (path?: string) => { const next = await onPickFolder(path); if (next) { setWorkspaceMenuOpen(false); setWorkspaceQuery(""); } };

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
      {dragOver && (
        <div className="absolute inset-0 z-50 flex items-center justify-center rounded-2xl bg-accent/10 border-2 border-dashed border-accent/40 backdrop-blur-[2px] pointer-events-none animate-[fadeIn_0.15s_ease-out]">
          <div className="flex flex-col items-center gap-2 text-accent">
            <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" x2="12" y1="15" y2="3" />
            </svg>
            <span className="text-sm font-medium">释放以添加文件</span>
          </div>
        </div>
      )}
      {/* ── 工作区切换菜单 ── */}
      {workspaceMenuOpen && cwd && (
        <div
          className="absolute left-2.5 bottom-12 z-40 w-[min(320px,82vw)] p-2.5 border border-border rounded-xl bg-bg-elev anim-menu-in no-drag"
          style={{boxShadow: "var(--ds-shadow-dropdown)"}}
          ref={workspaceMenuRef}
        >
          <label className="flex items-center gap-[7px] px-2 py-1.5 mb-1 border border-border-soft rounded-md bg-bg-soft focus-within:border-accent transition-colors">
            <Search size={14} className="text-fg-faint" />
            <input autoFocus className="flex-1 border-0 bg-transparent text-fg text-[13px] outline-none placeholder:text-fg-faint"
              value={workspaceQuery} onChange={(e) => setWorkspaceQuery(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Escape") setWorkspaceMenuOpen(false); }}
              placeholder={t("composer.searchProjects")} />
          </label>
          <div className="max-h-[280px] overflow-y-auto mb-1">
            {filteredWorkspaces.map((w) => (
              <button key={w.path}
                className={`flex items-center gap-2.5 w-full px-2 py-1.5 bg-transparent border-0 rounded-lg text-left cursor-pointer transition-colors duration-100 ${w.current ? "text-accent bg-accent-soft font-medium" : "text-fg-dim hover:bg-bg-soft hover:text-fg"}`}
                onClick={() => { if (w.current) { setWorkspaceMenuOpen(false); return; } void chooseWorkspace(w.path); }}
                title={w.path}>
                <FolderGit2 size={15} className="shrink-0" />
                <span className="min-w-0 truncate flex-1 text-[13px]">{w.name}</span>
                {w.current && <Check size={15} className="text-accent shrink-0" />}
              </button>
            ))}
            {filteredWorkspaces.length === 0 && <div className="py-4 text-fg-faint text-xs text-center">{t("composer.noProjectMatches")}</div>}
          </div>
          <div className="pt-1 border-t border-border-soft">
            <button className="flex items-center gap-2.5 w-full px-2 py-1.5 bg-transparent border-0 rounded-lg text-left cursor-pointer text-fg-dim hover:bg-bg-soft hover:text-fg text-[13px] transition-colors" onClick={() => void chooseWorkspace()}>
              <FolderPlus size={15} className="shrink-0" />
              <span>{t("composer.addProject")}</span>
            </button>
          </div>
        </div>
      )}

      {/* ── 菜单（命令/参数/文件）── */}
      {menuMode === "slash" && <SlashMenu items={slashMatches} activeIndex={active} onPick={pickCommand} onHover={setActive} />}
      {menuMode === "slasharg" && argRes && <ArgMenu items={argRes.items} activeIndex={active} onPick={pickArg} onHover={setActive} />}
      {menuMode === "at" && <FileMenu items={atItems} activeIndex={active} onPick={pickEntry} onHover={setActive} />}

      {/* ── 附件预览 ── */}
      {attachments.length > 0 && (
        <div className="flex flex-wrap gap-1.5 px-1 pb-1.5">
          {attachments.map((a) => (
            <div className="flex items-center gap-1.5 pl-1.5 pr-1 py-1 bg-bg-elev-2 border border-border-soft rounded-lg text-xs" key={a.path}>
              {a.type === "image" ? (
                <img src={a.previewUrl} alt="" className="w-8 h-8 rounded object-cover" />
              ) : (
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-accent shrink-0">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                  <polyline points="14 2 14 8 20 8" />
                </svg>
              )}
              <span className="max-w-[120px] truncate text-fg-dim font-mono text-[11px]">{a.path.split("/").pop()}</span>
              {a.type === "image" && (
                <button
                  type="button"
                  className="flex items-center justify-center w-5 h-5 bg-transparent border-0 rounded text-fg-dim cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
                  title={running ? "助手回复中，稍后再试" : "识图：本地视觉模型识别图片内容"}
                  disabled={running || !!recognizingPath}
                  onClick={() => void handleRecognize(a)}
                >
                  {recognizingPath === a.path ? <Loader size={12} className="animate-spin" /> : <Eye size={12} />}
                </button>
              )}
              {a.type === "image" && (
                <button
                  type="button"
                  className="flex items-center justify-center w-5 h-5 bg-transparent border-0 rounded text-fg-dim cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
                  title={running ? "助手回复中，稍后再试" : "提取文字：本地 OvisOCR2 识别图中文字"}
                  disabled={running || !!ocrPath}
                  onClick={() => void handleOCRText(a)}
                >
                  {ocrPath === a.path ? <Loader size={12} className="animate-spin" /> : <FileText size={12} />}
                </button>
              )}
              <button type="button" className="flex items-center justify-center w-5 h-5 bg-transparent border-0 rounded text-fg-faint cursor-pointer hover:text-err hover:bg-bg-soft transition-colors" title="移除" onClick={() => setAttachments((prev) => prev.filter((x) => x.path !== a.path))}><X size={13} /></button>
            </div>
          ))}
        </div>
      )}

      {/* ── 粘贴块 ── */}
      <PasteBlocksUI
        blocks={paste.activePastedBlocks}
        openLabels={paste.openPastedLabels}
        onTogglePreview={paste.togglePreview}
        onExpand={paste.expandBlock}
        onRemove={paste.removeBlock}
      />

      {/* ── 排队列表 ── */}
      {running && queueDisplay.length > 0 && (
        <div className="mb-2 max-h-[120px] overflow-y-auto rounded-xl border border-border-soft bg-bg-elev px-2 py-1.5">
          <div className="text-fg-faint/50 text-[10px] font-medium px-2 pb-1 select-none">排队中 ({queueDisplay.length})</div>
          {queueDisplay.map((item, i) => (
            <div key={i} className="flex items-center gap-2 py-1 px-2 rounded-md hover:bg-bg-soft group transition-colors duration-100">
              <span className="text-xs text-fg-dim flex-1 truncate">{item.slice(0, 80)}</span>
              <button
                className="opacity-0 group-hover:opacity-100 inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-fg-faint hover:text-err hover:bg-err/10 cursor-pointer transition-all duration-150"
                onClick={() => cancelQueueItem(i)}
                title="取消排队"
              >
                <X size={12} />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* ── 输入卡片 ── */}
      <div
        className={`relative border border-border-soft bg-bg-elev rounded-2xl overflow-hidden transition-[border-color,box-shadow] duration-[var(--dur-base)] focus-within:border-accent/30 focus-within:shadow-[0_0_0_1px_var(--accent-soft),var(--ds-shadow-composer)] ${composerHeight !== null ? "flex flex-col" : ""} ${composerResizing ? "cursor-ns-resize" : ""}`}
        style={{ ...(composerHeight !== null ? { height: "var(--composer-height)" } : {}), ...composerCardStyle }}
        ref={composerCardRef}
      >
        {/* 拖拽调整大小把手 */}
        <div
          className="absolute top-0 left-[14px] right-[14px] z-[5] h-2 cursor-ns-resize no-drag touch-none"
          onPointerDown={onComposerResizeStart}
          onDoubleClick={resetComposerHeight}
        />

        {/* 粘贴表格即数据：检测到表格块时提示，可关 */}
        {tableInfo && !disabled && (
          <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border-soft/60 bg-accent/[0.03] text-[11px]">
            <Table size={12} className="text-accent shrink-0" />
            <span className="text-fg-dim">
              已识别表格数据：{tableInfo.rows} 行 × {tableInfo.cols} 列
            </span>
            <label className="flex items-center gap-1 ml-auto cursor-pointer select-none text-fg-faint hover:text-fg transition-colors">
              <input
                type="checkbox"
                checked={tableMode}
                onChange={(e) => setTableMode(e.target.checked)}
                className="accent-accent"
              />
              发送时转为 Markdown 表格
            </label>
          </div>
        )}

        {/* 主输入行 */}
        <div
          className={`flex gap-2 items-center shrink-0 min-h-0 bg-transparent border-0 border-b border-border-soft rounded-none px-[13px] py-2.5 ${composerHeight !== null ? "flex-1 items-start" : ""} ${dragOver ? "outline outline-1 outline-dashed outline-accent outline-offset-[-4px] bg-accent-[0.02]" : ""} ${disabled ? "opacity-50 pointer-events-none" : ""}`}
          onDrop={onDrop} onDragOver={onDragOver} onDragLeave={onDragLeave}
        >
          <span className="text-accent font-mono font-semibold text-lg leading-[1.55] shrink-0 select-none">›</span>
          <textarea
            ref={taRef}
            className={`flex-1 resize-none border-0 bg-transparent text-fg leading-[1.55] max-h-[200px] outline-none placeholder:text-fg-faint ${composerHeight !== null ? "h-full max-h-none overflow-y-auto" : ""}`}
            style={{ fieldSizing: "content" }}
            value={text} onChange={(e) => setText(e.target.value)}
            onPaste={onPaste} onKeyDown={onKeyDown}
            placeholder={placeholderText}
            rows={1} disabled={disabled}
          />
          {running && (
            <button className="inline-flex items-center justify-center w-[30px] h-[30px] border-0 rounded-md cursor-pointer shrink-0 transition-all duration-[var(--dur-fast)] bg-bg-elev-2 text-err hover:bg-err hover:text-white active:scale-95" onClick={handleCancel} title={t("composer.stop")}>
              <Square size={14} fill="currentColor" />
            </button>
          )}
          <button
            className={`inline-flex items-center justify-center w-[32px] h-[32px] border-0 rounded-full cursor-pointer shrink-0 transition-all duration-[var(--dur-fast)] active:scale-95 ${running ? (shiftHeld ? "bg-warn/20 text-warn hover:bg-warn hover:text-white shadow-[0_0_8px_var(--warn)]" : "bg-bg-elev-2 text-fg-dim hover:bg-accent hover:text-accent-fg hover:scale-105") : "bg-accent text-accent-fg hover:brightness-110"} disabled:bg-bg-elev-2 disabled:text-fg-faint disabled:cursor-default disabled:hover:scale-100 disabled:active:scale-100 disabled:shadow-none`}
            style={!running && !disabled ? {boxShadow: "var(--ds-shadow-accent-btn)"} : undefined}
            onClick={submit}
            disabled={disabled || pendingPaste > 0 || (!text.trim() && attachments.length === 0 && (!running || queueLen === 0))}
            title={running ? (shiftHeld ? "纠正发送（Shift+Enter）" : queueLen > 0 ? `排队发送 (${queueLen})` : t("composer.queue")) : t("composer.send")}
          >
            {running && shiftHeld ? (
              <Zap size={16} />
            ) : running && queueLen > 0 ? (
              <span className="text-xs font-semibold leading-none">{queueLen}</span>
            ) : (
              <ArrowUp size={16} />
            )}
          </button>
        </div>

        {/* 底部工具栏 */}
        <div className="flex items-center gap-1.5 min-w-0 px-2.5 py-1.5">
          {cwd && (
            <div className="relative inline-flex min-w-0" ref={workspaceAnchorRef}>
              <button
                className={`inline-flex items-center gap-1.5 max-w-60 px-2 py-1 border-0 rounded-md bg-transparent text-fg-dim text-xs cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-60 no-drag ${workspaceMenuOpen ? "text-fg bg-bg-soft" : ""}`}
                onClick={() => { if (!running) setWorkspaceMenuOpen((o) => !o); }}
                disabled={running}
                title={running ? t("common.busyHint") : t("status.switchFolder", { cwd })}
              >
                <FolderGit2 size={13} />
                <span className="min-w-0 truncate">{workspaceName}</span>
                <ChevronDown size={12} />
              </button>
            </div>
          )}

          {/* 导入文件按钮 */}
          <button
            className={`inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0 ${pendingPaste > 0 ? "pointer-events-none opacity-40" : ""}`}
            onClick={() => void handlePickFiles()}
            disabled={running}
            title={running ? t("common.busyHint") : t("composer.importFile")}
          >
            <Paperclip size={14} />
          </button>

          {/* 截图按钮：整屏捕获后裁剪并附加 */}
          <button
            className={`inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0 ${captureBusy ? "pointer-events-none opacity-40" : ""}`}
            onClick={() => void handleScreenshot()}
            disabled={running || captureBusy}
            title={running ? t("common.busyHint") : "截图：捕获屏幕并裁剪附加"}
          >
            {captureBusy ? <Loader size={14} className="animate-spin" /> : <Camera size={14} />}
          </button>

          {/* 权限级别选择器：询问 / 自动 / YOLO */}
          <div className="flex gap-[3px]">
            {(["ask", "auto", "yolo"] as const).map((level) => {
              const labels: Record<string, string> = { ask: "询问", auto: "自动", yolo: "⚡ YOLO" };
              const descs: Record<string, string> = { ask: "写入前需确认（默认）", auto: "写入无需确认，deny 规则仍生效", yolo: "跳过所有确认提示" };
              const isYolo = level === "yolo";
              return (
                <button key={level} type="button"
                  className={`flex items-center gap-1.5 px-2.5 py-1 border rounded-md bg-transparent text-xs cursor-pointer transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
                    permLevel === level
                      ? isYolo ? "text-err bg-err/10 border-err/20 shadow-[0_0_0_1px_var(--err)]" : "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
                      : "text-fg-dim border-border-soft hover:text-fg hover:bg-bg-soft hover:border-fg-faint"
                  }`}
                  onClick={() => { if (permLevel !== level && onSetPermLevel) onSetPermLevel(level); }}
                  title={descs[level]}
                >
                  {labels[level]}
                </button>
              );
            })}
          </div>

{/* 快捷提示 */}
          <span className="ml-auto text-fg-faint/40 text-[10px] select-none hidden sm:inline-flex items-center gap-1.5">
            <span>/ 命令</span>
            <span>@ 文件</span>
            {running && <span className="text-warn/60">Shift+Enter 纠正</span>}
          </span>
        </div>
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
