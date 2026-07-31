// ImageGenPanel.tsx — 绘梦图片生成弹窗（三栏布局 · 深度优化）
import { useState, useCallback, useEffect, useRef } from "react";
import {
  X, Sparkles, Play, Square, Loader2,
  Shuffle, Trash2, Pencil,
  SlidersHorizontal, Dices, Ban, History, LayoutTemplate, ChevronDown, Copy, FolderOpen,
} from "lucide-react";
import { generateFreeImage, getComfyUIStatus, startComfyUI, stopComfyUI, loadImageResults, deleteImageResult, openImageHistoryDir } from "../lib/image";
import { ResultGallery } from "./ResultGallery";
import { PromptBar } from "./imagegen/PromptBar";
import { Lightbox, type LightboxImage } from "./Lightbox";
import {
  TEMPLATES, getAllCategories,
  loadCustomTemplates, saveCustomTemplates, generateTemplateId,
  type Template, type CustomTemplate,
} from "../data/imageTemplates";
import type { ImageGenResult, ComfyUIStatus } from "../lib/types";

const SIZE_OPTIONS = [
  { label: "方形 1:1 (1024)", value: "1024x1024" },
  { label: "风景 4:3", value: "1024x768" },
  { label: "宽屏 16:9", value: "1024x576" },
  { label: "竖屏 9:16", value: "576x1024" },
  { label: "肖像 3:4", value: "768x1024" },
  { label: "超宽 21:9", value: "1280x544" },
];

const MODEL_OPTIONS = [
  { label: "Flux Dev", value: "flux" },
  { label: "Z-Image-Turbo", value: "z-image-turbo" },
];

interface ImageGenPanelProps {
  onClose: () => void;
}

export function ImageGenPanel({ onClose }: ImageGenPanelProps) {
  // 输入状态
  const [prompt, setPrompt] = useState("");
  const [negative, setNegative] = useState("");
  const [size, setSize] = useState("1024x1024");
  const [model, setModel] = useState("flux");
  const [seed, setSeed] = useState(0);
  const [count, setCount] = useState(1);

  // 模板
  const [templateCat, setTemplateCat] = useState<string | undefined>();
  const [customTemplates, setCustomTemplates] = useState<CustomTemplate[]>(() => loadCustomTemplates());
  const [showCustomEditor, setShowCustomEditor] = useState(false);
  const [editingCustom, setEditingCustom] = useState<CustomTemplate | null>(null);
  const [customLabel, setCustomLabel] = useState("");
  const [customPrompt, setCustomPrompt] = useState("");
  const [customNegative, setCustomNegative] = useState("");

  // 生成状态
  const [generating, setGenerating] = useState(false);
  const [comfyStatus, setComfyStatus] = useState<ComfyUIStatus>({ running: false, url: "" });
  const [comfyStarting, setComfyStarting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [elapsed, setElapsed] = useState(0);

  // 结果
  const [results, setResults] = useState<ImageGenResult[]>([]);
  const [history, setHistory] = useState<ImageGenResult[]>([]);

  // Lightbox 状态
  const [lightboxImages, setLightboxImages] = useState<LightboxImage[]>([]);
  const [lightboxIndex, setLightboxIndex] = useState(-1);

  // 焦点管理
  const resultsContainerRef = useRef<HTMLDivElement>(null);
  const prevGeneratingRef = useRef(false);

  // 定时器
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const generatingRef = useRef(false);

  // 加载 ComfyUI 状态
  const refreshStatus = useCallback(async () => {
    try {
      const s = await getComfyUIStatus();
      setComfyStatus(s);
    } catch { /* ignore */ }
  }, []);

  useEffect(() => {
    refreshStatus();
    const interval = comfyStarting ? 2000 : 10000;
    const iv = setInterval(refreshStatus, interval);
    return () => clearInterval(iv);
  }, [refreshStatus, comfyStarting]);

  useEffect(() => {
    if (comfyStatus.running && comfyStarting) {
      setComfyStarting(false);
    }
  }, [comfyStatus.running, comfyStarting]);

  useEffect(() => {
    void (async () => {
      try {
        const h = await loadImageResults();
        if (h && h.length > 0) {
          const items = h as unknown as ImageGenResult[];
          setHistory(items);
        }
      } catch { /* ignore */ }
    })();
  }, []);

  // 生成计时
  useEffect(() => {
    if (!generating) { setElapsed(0); return; }
    const start = Date.now();
    timerRef.current = setInterval(() => setElapsed(Math.round((Date.now() - start) / 1000)), 1000);
    return () => { if (timerRef.current) clearInterval(timerRef.current); };
  }, [generating]);

  // 生成完成后聚焦结果
  useEffect(() => {
    if (prevGeneratingRef.current && !generating && results.length > 0) {
      requestAnimationFrame(() => {
        const firstCard = resultsContainerRef.current?.querySelector("[data-result-card]") as HTMLElement | null;
        firstCard?.focus({ preventScroll: true });
      });
    }
    prevGeneratingRef.current = generating;
  }, [generating, results.length]);

  const generate = async () => {
    if (!prompt.trim()) return;
    generatingRef.current = true;
    setError(null);
    setGenerating(true);
    setResults([]);
    try {
      const resp = await generateFreeImage(prompt, negative, size, model, seed, count);
      if (resp.error) {
        setError(resp.error);
      } else if (resp.images) {
        const newItems = resp.images as ImageGenResult[];
        setResults(newItems);
        setHistory((prev) => [...newItems, ...prev]);
      }
    } catch (e: unknown) {
      setError(typeof e === "string" ? e : (e instanceof Error ? e.message : String(e ?? "生成失败")));
    } finally {
      setGenerating(false);
      generatingRef.current = false;
    }
  };

  const handleStart = async () => {
    setComfyStarting(true);
    try { await startComfyUI(); }
    catch (e: unknown) {
      setError(typeof e === "string" ? e : (e instanceof Error ? e.message : String(e ?? "启动失败")));
      setComfyStarting(false);
    }
  };
  const handleStop = async () => {
    try { await stopComfyUI(); setComfyStatus({ running: false, url: "" }); }
    catch (e: unknown) { setError(typeof e === "string" ? e : (e instanceof Error ? e.message : String(e ?? "停止失败"))); }
  };

  const reuseResult = (r: ImageGenResult) => {
    setPrompt(r.prompt);
    if (r.model) setModel(r.model);
    if (r.size) setSize(r.size);
    if (r.seed) setSeed(r.seed);
  };

  const handleDelete = (index: number) => {
    const r = results[index];
    if (!r) return;
    setResults((prev) => prev.filter((_, i) => i !== index));
    setHistory((prev) => prev.filter((h) => !(h.seed === r.seed && h.prompt === r.prompt && h.time === r.time)));
    if (lightboxIndex === index) setLightboxIndex(-1);
    else if (lightboxIndex > index) setLightboxIndex((li) => li - 1);
  };

  const handleDownload = (index: number) => {
    const r = results[index];
    if (!r?.image) return;
    const a = document.createElement("a");
    a.href = r.image;
    a.download = `gaea-${r.seed || Date.now()}.png`;
    a.click();
  };

  const openLightboxFromResults = (i: number) => {
    setLightboxImages(results.map((r) => ({
      dataUrl: r.image, prompt: r.prompt, seed: r.seed,
    })));
    setLightboxIndex(i);
  };

  const openLightboxFromHistory = (i: number) => {
    setLightboxImages(history.map((h) => ({
      dataUrl: h.image, prompt: h.prompt, seed: h.seed,
    })));
    setLightboxIndex(i);
  };

  /** 删除历史中某条记录 */
  const handleHistoryDelete = async (i: number) => {
    try {
      await deleteImageResult(i);
      setHistory((prev) => prev.filter((_, idx) => idx !== i));
      if (lightboxIndex === i) setLightboxIndex(-1);
      else if (lightboxIndex > i) setLightboxIndex((li) => li - 1);
    } catch (e) {
      console.warn("删除历史记录失败", e);
    }
  };

  /** 复制历史图片到剪贴板 */
  const handleHistoryCopy = async (dataUrl: string) => {
    try {
      const resp = await fetch(dataUrl);
      const blob = await resp.blob();
      await navigator.clipboard.write([
        new ClipboardItem({ [blob.type]: blob }),
      ]);
    } catch (e) {
      console.warn("复制图片失败", e);
    }
  };

  const openCustomAdd = () => {
    setEditingCustom(null); setCustomLabel(""); setCustomPrompt(""); setCustomNegative("");
    setShowCustomEditor(true);
  };
  const openCustomEdit = (t: CustomTemplate) => {
    setEditingCustom(t); setCustomLabel(t.label); setCustomPrompt(t.prompt); setCustomNegative(t.negative);
    setShowCustomEditor(true);
  };
  const saveCustom = () => {
    if (!customLabel.trim() || !customPrompt.trim()) return;
    let updated: CustomTemplate[];
    if (editingCustom) {
      updated = customTemplates.map((t) => t.id === editingCustom.id ? { ...t, label: customLabel, prompt: customPrompt, negative: customNegative } : t);
    } else {
      updated = [...customTemplates, { id: generateTemplateId(), label: customLabel, prompt: customPrompt, negative: customNegative }];
    }
    setCustomTemplates(updated);
    saveCustomTemplates(updated);
    setShowCustomEditor(false);
  };
  const deleteCustom = (id: string) => {
    const updated = customTemplates.filter((t) => t.id !== id);
    setCustomTemplates(updated);
    saveCustomTemplates(updated);
  };

  const applyTemplate = (t: Template) => {
    setPrompt((p) => p ? p + "，" + t.prompt : t.prompt);
    if (t.negative) setNegative((n) => n ? n + ", " + t.negative : t.negative);
  };

  const allCats = getAllCategories(customTemplates.length);
  const currentTemplates: (Template & { _id?: string })[] = templateCat
    ? (templateCat === "⭐ 自定义" ? customTemplates.map((t) => ({ ...t, _id: t.id })) : TEMPLATES[templateCat] || [])
    : [];

  return (
    <div
      className="fixed inset-0 z-40 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4 animate-[overlay-in_0.18s_ease-out]"
      onClick={onClose}
    >
      <div
        className="w-full max-w-6xl h-[90vh] flex flex-col rounded-2xl bg-bg-elev shadow-2xl border border-border overflow-hidden animate-[panel-in_0.34s_ease-out]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* ═══ 顶栏 ═══ */}
        <div className="flex items-center justify-between px-5 py-2 border-b border-border/50 shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
              <Sparkles className="w-4 h-4 text-white" />
            </div>
            <h2 className="text-lg font-bold text-fg">绘梦</h2>
          </div>
          <button onClick={onClose} className="rounded-lg p-2 text-fg-faint hover:text-fg hover:bg-bg-soft transition">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* ═══ 三栏主体 ═══ */}
        <div className="flex-1 flex gap-0 overflow-hidden">
          {/* ── 左栏：参数（卡片式 + 右分隔线） ── */}
          <div className="w-52 shrink-0 flex flex-col overflow-y-auto bg-bg-soft/30 rounded-l-xl border-r border-border p-3 gap-2.5">
            {/* 顶部分组标题 */}
            <div className="flex items-center gap-1.5 mb-1">
              <SlidersHorizontal className="w-3.5 h-3.5 text-fg-dim" />
              <span className="text-xs font-semibold text-fg-dim uppercase tracking-wider">参数</span>
            </div>
            <div className="h-px bg-border/50 -mx-3 mb-2" />

            {/* 错误提示 */}
            {error && (
              <div className="rounded-lg bg-err/10 border border-err/30 px-2.5 py-2 text-sm text-err/90">
                {error}
                <button onClick={() => setError(null)} className="ml-1 underline"><X className="w-3 h-3 inline" /></button>
              </div>
            )}

            {/* ① 模板 */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="dream-label inline-flex items-center gap-1"><LayoutTemplate className="w-3 h-3" />快速模板</span>
                <button onClick={openCustomAdd} className="text-xs text-accent hover:brightness-110">+ 自定义</button>
              </div>
              <select
                value={templateCat || ""}
                onChange={(e) => setTemplateCat(e.target.value || undefined)}
                className="dream-input mb-1.5"
              >
                <option value="">选择模板类别…</option>
                {allCats.map((c) => (<option key={c} value={c}>{c}</option>))}
              </select>
              {currentTemplates.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {currentTemplates.map((t, i) => {
                    const isCustom = templateCat === "⭐ 自定义";
                    return (
                      <span
                        key={isCustom ? (t._id || i) : i}
                        onClick={() => !isCustom && applyTemplate(t)}
                        className={`inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs cursor-pointer transition border ${
                          isCustom
                            ? "border-accent bg-accent-soft text-accent"
                            : "border-border bg-bg-soft text-fg-dim hover:border-fg-faint"
                        }`}
                      >
                        {isCustom && (
                          <span onClick={(e) => { e.stopPropagation(); openCustomEdit(t as CustomTemplate); }} className="cursor-pointer">
                            <Pencil className="w-2 h-2 mr-0.5 inline" />
                          </span>
                        )}
                        {t.label}
                        {isCustom && (
                          <button onClick={(e) => { e.stopPropagation(); deleteCustom((t as CustomTemplate).id); }} className="ml-0.5 hover:text-err">
                            <X className="w-2 h-2" />
                          </button>
                        )}
                      </span>
                    );
                  })}
                </div>
              )}
            </div>

            {/* ② 负向提示词 */}
            <div>
              <span className="dream-label inline-flex items-center gap-1 mb-2"><Ban className="w-3 h-3" />不想出现</span>
              <textarea
                placeholder="模糊, 低质量, 畸形手指..."
                value={negative}
                onChange={(e) => setNegative(e.target.value)}
                rows={2}
                className="dream-input"
              />
            </div>

            {/* ③ 图片参数 — 水平网格 */}
            <div>
              <span className="dream-label inline-flex items-center gap-1 mb-2"><SlidersHorizontal className="w-3 h-3" />图片参数</span>
              <div className="grid grid-cols-2 gap-1.5 mb-1.5">
                <select value={size} onChange={(e) => setSize(e.target.value)} className="dream-input">
                  {SIZE_OPTIONS.map((o) => (<option key={o.value} value={o.value}>{o.label}</option>))}
                </select>
                <select value={model} onChange={(e) => setModel(e.target.value)} className="dream-input">
                  {MODEL_OPTIONS.map((o) => (<option key={o.value} value={o.value}>{o.label}</option>))}
                </select>
              </div>
              <div>
                <span className="dream-label">生成数量</span>
                <select value={count} onChange={(e) => setCount(Number(e.target.value))} className="dream-input">
                  {[1, 2, 3, 4].map((n) => (<option key={n} value={n}>{n}</option>))}
                </select>
              </div>
            </div>

            {/* ④ 种子 — progressive-disclosure 默认折叠 */}
            <details className="group" open={seed !== 0}>
              <summary className="dream-label inline-flex items-center gap-1 mb-2 cursor-pointer hover:bg-bg-soft rounded py-0.5 -mx-1 px-1 transition select-none">
                <Dices className="w-3 h-3" />
                种子 {seed !== 0 && <span className="text-accent ml-0.5">({seed})</span>}
                <ChevronDown className="w-3 h-3 ml-auto transition-transform group-open:rotate-180" />
              </summary>
              <div className="flex gap-1">
                <input
                  type="number"
                  value={seed || ""}
                  onChange={(e) => setSeed(Number(e.target.value) || 0)}
                  placeholder="随机"
                  min={1}
                  className="dream-input flex-1"
                />
                <button onClick={() => setSeed(0)} className="p-1.5 min-w-[28px] rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition" title="随机">
                  <Shuffle className="w-3 h-3" />
                </button>
              </div>
            </details>

            {/* ⑤ 自定义模板编辑器 */}
            {showCustomEditor && (
              <div className="rounded-lg border border-accent-soft bg-accent-soft/30 p-3 space-y-1.5">
                <input value={customLabel} onChange={(e) => setCustomLabel(e.target.value)} placeholder="模板名称" className="dream-input text-sm" />
                <textarea value={customPrompt} onChange={(e) => setCustomPrompt(e.target.value)} placeholder="提示词" rows={2} className="dream-input text-sm" />
                <textarea value={customNegative} onChange={(e) => setCustomNegative(e.target.value)} placeholder="负向（可选）" rows={1} className="dream-input text-sm" />
                <div className="flex gap-1.5">
                  <button onClick={saveCustom} disabled={!customLabel.trim() || !customPrompt.trim()} className="flex-1 rounded bg-accent px-2 py-1 text-sm text-accent-fg hover:brightness-110 disabled:opacity-50 transition">保存</button>
                  <button onClick={() => setShowCustomEditor(false)} className="rounded border border-border px-2 py-1 text-sm text-fg-dim hover:bg-bg-soft transition">取消</button>
                </div>
              </div>
            )}

            {/* ComfyUI 状态 — 卡片式启停控制 */}
            <div className="mt-auto pt-2 border-t border-border/50">
              <div className="rounded-lg bg-bg-soft/50 border border-border/50 p-2.5 space-y-1.5">
                <div className="flex items-center gap-1.5 text-xs text-fg-dim">
                  <div className="flex items-center gap-1">
                    <span className={`w-1.5 h-1.5 rounded-full ${comfyStatus.running ? "bg-ok animate-pulse" : "bg-fg-faint"}`} />
                    <span className="text-fg-faint">ComfyUI</span>
                  </div>
                  {comfyStarting ? (
                    <span className="inline-flex items-center gap-1 text-accent">
                      <Loader2 className="w-2.5 h-2.5 animate-spin" /> 启动中
                    </span>
                  ) : (
                    <span className={comfyStatus.running ? "text-ok" : "text-fg-faint"}>
                      {comfyStatus.running ? "已连接" : "未连接"}
                    </span>
                  )}
                </div>
                <div className="flex gap-1.5">
                  {!comfyStarting && (comfyStatus.running ? (
                    <button onClick={handleStop} className="flex-1 flex items-center justify-center gap-1 rounded-md bg-err/10 border border-err/20 px-2 py-1 text-xs text-err hover:bg-err/20 transition">
                      <Square className="w-3 h-3" /> 停止
                    </button>
                  ) : (
                    <button onClick={handleStart} className="flex-1 flex items-center justify-center gap-1 rounded-md bg-ok/10 border border-ok/20 px-2 py-1 text-xs text-ok hover:bg-ok/20 transition">
                      <Play className="w-3 h-3" /> 启动
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>

          {/* ── 中栏：画布 + PromptBar（左右分隔线） ── */}
          <div className="flex-1 flex flex-col min-w-0 overflow-hidden border-r border-border relative">
            {/* 标题头 */}
            <div className="shrink-0 flex items-center gap-1.5 px-3 py-2 border-b border-border/50">
              <Sparkles className="w-3.5 h-3.5 text-fg-dim" />
              <span className="text-xs font-semibold text-fg-dim uppercase tracking-wider">画布</span>
              {generating && (
                <span className="ml-auto inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md bg-accent-soft text-accent text-xs font-medium">
                  <Loader2 className="w-2.5 h-2.5 animate-spin" />
                  {elapsed}s
                </span>
              )}
            </div>
            <div className="flex-1 overflow-y-auto pb-28" ref={resultsContainerRef}>
              <ResultGallery
                results={results}
                generating={generating}
                onPreview={openLightboxFromResults}
                onDownload={handleDownload}
                onReuse={reuseResult}
                onDelete={handleDelete}
              />
            </div>
            <PromptBar
              prompt={prompt}
              onPromptChange={setPrompt}
              generating={generating}
              elapsed={elapsed}
              onGenerate={generate}
            />
          </div>

          {/* ── 右栏：历史（卡片式 + 左分隔线） ── */}
          <div className="w-52 shrink-0 flex flex-col overflow-y-auto bg-bg-soft/30 rounded-r-xl border-l border-border p-3 gap-2">
            <div className="flex items-center justify-between shrink-0">
              <div className="flex items-center gap-1.5">
                <span className="dream-label inline-flex items-center gap-1"><History className="w-3 h-3" />历史 ({history.length})</span>
                <div className="w-px h-3 bg-border" />
              </div>
                <button onClick={openImageHistoryDir} className="text-xs text-fg-faint hover:text-fg px-1 transition" title="打开文件夹">
                  <FolderOpen className="w-3 h-3" />
                </button>
                <button onClick={() => setHistory([])} className="text-xs text-fg-faint hover:text-err px-1 transition" title="清空">
                  <Trash2 className="w-3 h-3" />
                </button>
            </div>
            <div className="flex-1 overflow-y-auto flex flex-col gap-1.5">
              {history.length === 0 ? (
                <div className="flex flex-col items-center justify-center gap-2 mt-8 text-fg-faint">
                  <Sparkles className="w-5 h-5 text-fg-faint/60" />
                  <span className="text-xs">生成第一张图吧 ✨</span>
                </div>
              ) : (
                history.map((h, i) => (
                  <div
                    key={i}
                    className="group relative rounded-xl overflow-hidden cursor-pointer shrink-0 transition border-2"
                    onClick={() => openLightboxFromHistory(i)}
                  >
                    <div className={`absolute inset-0 pointer-events-none z-0 border-2 rounded-lg ${
                      lightboxIndex === i ? "border-accent" : "border-transparent group-hover:border-border"
                    }`} />
                    <img src={h.image} alt="" className="w-full block object-cover aspect-square" loading="lazy" />
                    {/* hover 遮罩操作层 */}
                    <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity duration-200 ease-out flex items-center justify-center gap-2 z-10">
                      <button
                        onClick={(e) => { e.stopPropagation(); handleHistoryCopy(h.image); }}
                        className="w-7 h-7 rounded-md bg-white/20 hover:bg-white/40 flex items-center justify-center transition"
                        title="复制图片"
                      >
                        <Copy className="w-3.5 h-3.5 text-white" />
                      </button>
                      <button
                        onClick={(e) => { e.stopPropagation(); handleHistoryDelete(i); }}
                        className="w-7 h-7 rounded-md bg-white/20 hover:bg-red-500/60 flex items-center justify-center transition"
                        title="删除"
                      >
                        <Trash2 className="w-3.5 h-3.5 text-white" />
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Lightbox */}
      {lightboxIndex >= 0 && (
        <Lightbox
          images={lightboxImages}
          index={lightboxIndex}
          onClose={() => setLightboxIndex(-1)}
          onNavigate={(i) => setLightboxIndex(i)}
        />
      )}
    </div>
  );
}
