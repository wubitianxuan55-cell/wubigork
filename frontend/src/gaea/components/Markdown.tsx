import { memo, useCallback, useEffect, useRef, useState } from "react";
import ReactMarkdown, { defaultUrlTransform } from "react-markdown";
import type { Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import mermaid from "mermaid";
import { Check, Copy, FileText, Loader } from "../icons";

import { app, openExternal } from "../lib/bridge";
import { classifyExternalLink } from "../lib/browserPolicy";
import { isLocalFilePath } from "../lib/fileLinks";
import { remarkFileLinks } from "../lib/remarkFileLinks";
import { remarkMemCitations } from "../lib/remarkMemCitations";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";
import { useToast } from "./Toast";
import { FileChip } from "./FileChip";
import { MemCitationChip } from "./MemCitationChip";

// KaTeX CSS 延迟注入：避免非数学对话的 ~23KB CSS 开销。
// 有数学内容时才加载（$$ 或 $ 包裹的公式）。
let katexCssLoaded = false;

function ensureKatexCss() {
  if (katexCssLoaded) return;
  katexCssLoaded = true;
  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = new URL("katex/dist/katex.min.css", import.meta.url).href;
  document.head.appendChild(link);
}

function hasMathContent(text: string): boolean {
  return text.includes("$$") || (text.includes("$") && /\$\S[^$]*\S\$/.test(text));
}

// ── Mermaid 图表渲染（agent 生成的流程图/架构图/思维导图等）─────────────

let mermaidInitedTheme: string | null = null;

// 已自动导出过的 Mermaid 代码（按内容哈希去重，避免历史消息滚动重复生成）。
const autoExportedCodes = new Set<string>();

function codeHash(s: string): string {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) | 0;
  }
  return String(h);
}

function utf8ToB64(s: string): string {
  return btoa(unescape(encodeURIComponent(s)));
}

// standaloneHtmlFromSvg 把渲染后的 Mermaid SVG 包成可独立打开的 HTML 页面。
function standaloneHtmlFromSvg(svg: string): string {
  const bg = mermaidTheme() === "dark" ? "#1e1e2e" : "#ffffff"; // hex-exempt mermaid 渲染底色（主题已分支）
  return `<!doctype html>
<html lang="zh">
<head>
<meta charset="utf-8">
<title>Mermaid 图表</title>
<style>html,body{margin:0;padding:16px;background:${bg};font-family:system-ui,-apple-system,'Segoe UI',sans-serif;} svg{max-width:100%;height:auto;}</style>
</head>
<body>
${svg}
</body>
</html>`;
}

function mermaidTheme(): "default" | "dark" {
  if (typeof window === "undefined" || !window.matchMedia) return "default";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "default";
}

function ensureMermaid(theme: "default" | "dark") {
  if (mermaidInitedTheme === theme) return;
  mermaidInitedTheme = theme;
  mermaid.initialize({
    startOnLoad: false,
    theme,
    securityLevel: "strict",
    fontFamily: "system-ui, -apple-system, 'Segoe UI', sans-serif",
  });
}

const MermaidBlock = memo(function MermaidBlock({ code, autoExport = true }: { code: string; autoExport?: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  const idRef = useRef(0);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [exportedPath, setExportedPath] = useState<string | null>(null);
  const [svgExporting, setSvgExporting] = useState(false);
  const [view, setView] = useState<"chart" | "code" | "html">("chart");
  const [zoom, setZoom] = useState(1);
  const htmlDocRef = useRef<string | null>(null);
  const openFilePreview = openPaneFileOrPreview;
  const toast = useToast();

  const setZoomClamped = (next: number) => setZoom(Math.min(4, Math.max(0.5, Math.round(next * 100) / 100)));

  // 把已渲染的 SVG 转成 PNG data URL（SVG 图片异步加载，返回 Promise）。
  const toPngDataUrl = useCallback((): Promise<string | null> => {
    return new Promise((resolve) => {
      const el = ref.current?.querySelector("svg");
      if (!el) {
        resolve(null);
        return;
      }
      const vb = el.getAttribute("viewBox")?.split(/\s+/).map(Number);
      const rect = el.getBoundingClientRect();
      const w = vb?.[2] || rect.width || 800;
      const h = vb?.[3] || rect.height || 600;
      const scale = 2;
      const canvas = document.createElement("canvas");
      canvas.width = Math.max(1, Math.round(w * scale));
      canvas.height = Math.max(1, Math.round(h * scale));
      const ctx = canvas.getContext("2d");
      if (!ctx) {
        resolve(null);
        return;
      }
      const bg = getComputedStyle(el).backgroundColor;
      ctx.fillStyle = bg && bg !== "rgba(0, 0, 0, 0)" ? bg : mermaidTheme() === "dark" ? "#1e1e2e" : "#ffffff"; // hex-exempt mermaid 渲染底色（主题已分支）
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      const xml = new XMLSerializer().serializeToString(el);
      const img = new Image();
      img.onload = () => {
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
        resolve(canvas.toDataURL("image/png"));
      };
      img.onerror = () => resolve(null);
      img.src = "data:image/svg+xml;charset=utf-8," + encodeURIComponent(xml);
    });
  }, []);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    ensureMermaid(mermaidTheme());
    const id = `gm-${Date.now()}-${idRef.current++}`;
    mermaid
      .render(id, code)
      .then(({ svg }) => {
        if (cancelled || !ref.current) return;
        htmlDocRef.current = standaloneHtmlFromSvg(svg);
        ref.current.innerHTML = svg;
        const svgEl = ref.current.querySelector("svg");
        if (svgEl) {
          svgEl.style.background = "transparent";
          svgEl.style.width = "100%";
          svgEl.style.height = "auto";
          svgEl.style.maxWidth = "none";
        }
        // 自动导出 PNG 到工作区（每个唯一图表只导出一次）。
        if (autoExport) {
          const key = codeHash(code);
          if (!autoExportedCodes.has(key)) {
            autoExportedCodes.add(key);
            void toPngDataUrl().then((dataUrl) => {
              if (!dataUrl || cancelled) return;
              return app.SavePastedImage(dataUrl)
                .then((path) => { if (!cancelled) setExportedPath(path); })
                .catch(() => {});
            });
          }
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(String((e as Error)?.message ?? e));
      });
    return () => { cancelled = true; };
  }, [code, autoExport, toPngDataUrl]);

  // 手动导出图片：存到工作区并打开预览。
  const exportPng = useCallback(async () => {
    if (exporting) return;
    setExporting(true);
    try {
      const dataUrl = await toPngDataUrl();
      if (!dataUrl) {
        toast.show("导出图片失败：图表尚未渲染完成", "warn");
        return;
      }
      const path = await app.SavePastedImage(dataUrl);
      setExportedPath(path);
      toast.show(`已导出图片：${path}`);
      openFilePreview(path);
    } finally {
      setExporting(false);
    }
  }, [exporting, toPngDataUrl, openFilePreview, toast]);

  // 导出 SVG：序列化渲染后的 SVG，保存到工作区并打开预览。
  const exportSvg = useCallback(async () => {
    const el = ref.current?.querySelector("svg");
    if (!el || svgExporting) return;
    setSvgExporting(true);
    try {
      const xml = new XMLSerializer().serializeToString(el);
      const name = `diagram-${Date.now()}.svg`;
      const path = await app.SaveAttachmentFile(name, utf8ToB64(xml));
      setExportedPath(path);
      toast.show(`已导出 SVG：${path}`);
      openFilePreview(path);
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn");
    } finally {
      setSvgExporting(false);
    }
  }, [svgExporting, openFilePreview, toast]);

  // 导出 HTML：保存独立页面到工作区并打开预览。
  const exportHtml = useCallback(async () => {
    const html = htmlDocRef.current;
    if (!html) return;
    try {
      const name = `diagram-${Date.now()}.html`;
      const path = await app.SaveAttachmentFile(name, utf8ToB64(html));
      setExportedPath(path);
      toast.show(`已导出 HTML：${path}`);
      openFilePreview(path);
    } catch (e: unknown) {
      toast.show(String((e as Error)?.message ?? e), "warn");
    }
  }, [openFilePreview, toast]);

  if (error) {
    return (
      <div className="my-3 rounded-lg border border-warning/30 bg-warning/5 overflow-hidden">
        <div className="px-3 py-1.5 border-b border-warning/20 text-[11px] text-warning">Mermaid 渲染失败，显示源码</div>
        <pre className="px-3 py-2.5 font-mono text-[12.5px] leading-[1.55] overflow-auto whitespace-pre text-fg">{code}</pre>
      </div>
    );
  }
  return (
    <div className="my-3 rounded-lg border border-border-soft overflow-hidden">
      <div className="flex items-center gap-1 px-2.5 py-1 bg-bg-soft/80 border-b border-border-soft/50 text-[10px] select-none">
        <span className="text-fg-faint/60 font-mono font-medium uppercase tracking-wider mr-1.5">Mermaid</span>
        {/* 图表 / 代码切换 */}
        <div className="flex rounded-md overflow-hidden border border-border-soft/60">
          {(["chart", "code", "html"] as const).map((v) => (
            <button
              key={v}
              type="button"
              className={`px-2 py-0.5 cursor-pointer transition-colors ${view === v ? "bg-accent/15 text-accent" : "bg-transparent text-fg-faint hover:text-fg"}`}
              onClick={() => setView(v)}
            >
              {v === "chart" ? "图表" : v === "code" ? "代码" : "HTML"}
            </button>
          ))}
        </div>
        {view === "chart" && (
          <div className="flex items-center gap-0.5 ml-1 text-fg-faint">
            <button type="button" className="px-1 py-0.5 cursor-pointer hover:text-fg" onClick={() => setZoomClamped(zoom / 1.25)} title="缩小">−</button>
            <span className="w-7 text-center text-fg-dim">{Math.round(zoom * 100)}%</span>
            <button type="button" className="px-1 py-0.5 cursor-pointer hover:text-fg" onClick={() => setZoomClamped(zoom * 1.25)} title="放大">＋</button>
            <button type="button" className="px-1 py-0.5 cursor-pointer hover:text-fg" onClick={() => setZoom(1)} title="适应窗口宽度">适应</button>
          </div>
        )}
        <div className="ml-auto flex items-center gap-1">
          {view === "html" && (
            <button
              type="button"
              className="inline-flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/60 cursor-pointer hover:text-fg transition-colors"
              onClick={() => void exportHtml()}
              title="导出为独立 HTML 页面，保存到工作区"
            >
              <FileText size={10} />
              HTML
            </button>
          )}
          {view === "chart" && (
            <button
              type="button"
              className="inline-flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/60 cursor-pointer hover:text-fg transition-colors disabled:opacity-50 disabled:cursor-default"
              onClick={() => void exportSvg()}
              disabled={svgExporting}
              title="导出为 SVG 矢量图，保存到工作区"
            >
              {svgExporting ? <Loader size={10} className="animate-spin" /> : <FileText size={10} />}
              SVG
            </button>
          )}
          <button
            type="button"
            className="inline-flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/60 cursor-pointer hover:text-fg transition-colors disabled:opacity-50 disabled:cursor-default"
            onClick={() => void exportPng()}
            disabled={exporting}
            title="导出为 PNG 图片，保存到工作区"
          >
            {exporting ? <Loader size={10} className="animate-spin" /> : <FileText size={10} />}
            PNG
          </button>
        </div>
      </div>
      <div
        className="overflow-auto p-3 bg-bg-elev-2/40"
        aria-label="Mermaid 图表"
        style={{ display: view === "chart" ? undefined : "none" }}
      >
        <div style={{ zoom }} className="w-full">
          <div ref={ref} />
        </div>
      </div>
      {view === "code" && (
        <div className="relative bg-bg">
          <pre className="px-3 py-2.5 font-mono text-[12px] leading-[1.55] overflow-auto whitespace-pre text-fg max-h-80">{code}</pre>
          <button
            type="button"
            className="absolute top-2 right-2 inline-flex items-center gap-1 px-2 py-1 rounded-md border border-border-soft bg-bg-elev-2 text-fg-dim text-[10.5px] cursor-pointer hover:text-fg"
            onClick={() => { void navigator.clipboard.writeText(code); toast.show("Mermaid 源码已复制"); }}
          >
            <Copy size={10} /> 复制
          </button>
        </div>
      )}
      {view === "html" && (
        <div className="bg-bg">
          {htmlDocRef.current ? (
            <iframe
              title="Mermaid HTML 查看器"
              srcDoc={htmlDocRef.current}
              className="w-full h-80 border-0 bg-white"
              sandbox=""
            />
          ) : (
            <div className="py-8 text-center text-fg-faint text-xs">HTML 尚未生成，请稍候</div>
          )}
        </div>
      )}
      {exportedPath && (
        <div className="px-3 py-1.5 border-t border-border-soft/50 bg-bg-soft/40 text-[11px]">
          <button
            type="button"
            className="inline-flex items-center gap-1 text-accent cursor-pointer hover:underline"
            onClick={() => openFilePreview(exportedPath)}
            title="点击预览导出的图片文件"
          >
            <FileText size={11} />
            已导出：{exportedPath.split("/").pop()}
          </button>
        </div>
      )}
    </div>
  );
});

// ── 代码块复制按钮 ──────────────────────────────────────────────────

function CodeBlockHeader({ language, text }: { language?: string; text: string }) {
  const [copied, setCopied] = useState(false);
  const copy = useCallback(async () => {
    try { await navigator.clipboard.writeText(text); } catch { /* noop */ }
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [text]);
  return (
    <div className="flex items-center justify-between px-3 py-1 bg-bg-soft/80 border-b border-border-soft/50 rounded-t-lg text-[10px] select-none">
      <span className="text-fg-faint/60 font-mono font-medium uppercase tracking-wider">
        {language || "text"}
      </span>
      <button
        className="inline-flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-faint/40 cursor-pointer hover:text-fg transition-colors"
        onClick={copy}
        title="复制代码"
      >
        {copied ? <Check size={10} className="text-ok" /> : <Copy size={10} />}
        {copied ? "已复制" : "复制"}
      </button>
    </div>
  );
}

// ── Markdown 组件 ────────────────────────────────────────────────────

function buildComponents(onOpenFile: (rel: string) => void, autoExportMermaid = true): Components {
  return {
    pre: ({ children }) => <>{children}</>,
    code: ({ className, children }) => {
      const text = String(children ?? "").replace(/\n$/, "");
      const match = /language-([\w-]+)/.exec(className ?? "");
      const lang = match?.[1];
      const isBlock = match !== null || text.includes("\n");
      if (lang === "mermaid") {
        return <MermaidBlock code={text} autoExport={autoExportMermaid} />;
      }
      if (isBlock) {
        return (
          <div className="my-3 rounded-lg border border-border-soft overflow-hidden">
            <CodeBlockHeader language={lang} text={text} />
            <pre className="px-3 py-2.5 font-mono text-[12.5px] leading-[1.55] overflow-auto whitespace-pre text-fg"><code>{text}</code></pre>
          </div>
        );
      }
      return <code className="px-1 py-0.5 rounded bg-bg-soft text-fg text-[0.9em] font-mono border border-border-soft/50">{children}</code>;
    },
    a: ({ href, children }) => {
      // 记忆引用键（remarkMemCitations 产出 mem:<name> 链接）：渲染成可点击
      // 溯源徽标（C2 记忆引用可追溯）。
      if (href?.startsWith("mem:")) {
        const name = decodeURIComponent(href.slice(4));
        return <MemCitationChip name={name} />;
      }
      if (href && isLocalFilePath(href)) {
        const rel = decodeURIComponent(href.replace(/^\.{0,2}\//, ""));
        return (
          <FileChip
            path={rel}
            label={typeof children === "string" ? children : undefined}
            onOpen={onOpenFile}
            title={`点击预览 ${rel}`}
          />
        );
      }
      return (
        <a
          href={href}
          onClick={(e) => {
            e.preventDefault();
            // 1c 外链协议分流：http(s)→系统浏览器（loopback 拒）、mailto/tel
            // →系统处理器；javascript:/file:/相对路径等一律不交给系统。
            if (!href) return;
            const decision = classifyExternalLink(href);
            if (decision.kind === "open") openExternal(decision.url);
          }}
          className="text-accent hover:underline"
        >
          {children}
        </a>
      );
    },
    table: ({ children }) => (
      <div className="my-3 overflow-x-auto rounded-lg border border-border-soft">
        <table className="min-w-full text-[13px]">{children}</table>
      </div>
    ),
    th: ({ children }) => (
      <th className="px-3 py-2 text-left text-[11px] font-semibold text-fg-dim bg-bg-soft border-b border-border-soft">
        {children}
      </th>
    ),
    td: ({ children }) => (
      <td className="px-3 py-2 border-b border-border-soft/50 text-fg">{children}</td>
    ),
    blockquote: ({ children }) => (
      <blockquote className="my-2 pl-3 border-l-[3px] border-accent/30 text-fg-dim/80 italic">
        {children}
      </blockquote>
    ),
    hr: () => <hr className="my-4 border-border-soft" />,
    ol: ({ children }) => <ol className="my-2 pl-5 list-decimal text-fg space-y-0.5">{children}</ol>,
    ul: ({ children }) => <ul className="my-2 pl-5 list-disc text-fg space-y-0.5">{children}</ul>,
    h1: ({ children }) => <h1 className="mt-5 mb-2 text-[18px] font-bold text-fg">{children}</h1>,
    h2: ({ children }) => <h2 className="mt-4 mb-1.5 text-[16px] font-bold text-fg">{children}</h2>,
    h3: ({ children }) => <h3 className="mt-3 mb-1 text-[14px] font-semibold text-fg">{children}</h3>,
    p: ({ children }) => <p className="my-1.5 leading-relaxed text-fg">{children}</p>,
  };
}

// ── 数学公式标准化 ──────────────────────────────────────────────────

function normalizeMath(s: string): string {
  const lb = "\x00LB\x00";
  let r = s.replace(/\\\\\[/g, lb);
  r = r
    .replace(/\\\[/g, () => "$$")
    .replace(/\\\]/g, () => "$$")
    .replace(/\\\(/g, () => "$")
    .replace(/\\\)/g, () => "$");
  // 用字面量字符串恢复哨兵（等价于全局替换，避免在正则中书写 \x00 控制字符）
  r = r.split(lb).join("\\\\[");
  const vert = (m: string) => m.replace(/\|/g, "\\vert ");
  r = r.replace(/\$\$([\s\S]*?)\$\$/g, (_m, m) => `$$${vert(m)}$$`);
  r = r.replace(/\$([^$\n]+)\$/g, (_m, m) => `$${vert(m)}$`);
  return r;
}

export const Markdown = memo(function Markdown({ text, autoExportMermaid = true }: { text: string; autoExportMermaid?: boolean }) {
  if (hasMathContent(text)) ensureKatexCss();
  const openFilePreview = openPaneFileOrPreview;
  return (
    <div className="md text-[14px] leading-relaxed">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath, remarkFileLinks, remarkMemCitations]}
        rehypePlugins={[rehypeKatex]}
        components={buildComponents(openFilePreview, autoExportMermaid)}
        urlTransform={(url) => (isLocalFilePath(url) || url.startsWith("mem:") ? url : defaultUrlTransform(url))}
      >
        {normalizeMath(text)}
      </ReactMarkdown>
    </div>
  );
});
