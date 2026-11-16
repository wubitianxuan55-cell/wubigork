// mermaidRender — mermaid 模块加载与主题初始化单源（原办公线 Markdown.tsx 内联，
// C 批蒸馏 chat-demo 时抽共享：聊天线 ChatMermaid 与办公线 MermaidBlock 共用同一
// 模块缓存与初始化路径，避免两条宿主各自 initialize 打架——mermaid.initialize
// 是模块级全局配置，两套配置互写会让后渲染方继承先渲染方的字体/主题）。

export type MermaidTheme = "default" | "dark";

let mermaidInitedTheme: string | null = null;

export function mermaidTheme(): MermaidTheme {
  // v4.350：跟应用主题而非操作系统——App.tsx 把实际明暗挂在 <html data-hl>；
  // 此前读 prefers-color-scheme，应用选亮色而 OS 为暗色时深底图表嵌进浅色文档。
  if (typeof document !== "undefined" && document.documentElement.dataset.hl) {
    return document.documentElement.dataset.hl === "dark" ? "dark" : "default";
  }
  if (typeof window === "undefined" || !window.matchMedia) return "default";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "default";
}

// P4-H1：mermaid 由静态 import 改为运行时动态 import——只在首次实际渲染图表时
// 才加载模块（Vite 将 mermaid.core 拆为独立 async chunk，不再进入页面急切包）。
// 初始化配置对象与原本一致（startOnLoad/theme/securityLevel/fontFamily 原样保留）。
let mermaidMod: typeof import("mermaid") | null = null;

export async function ensureMermaid(theme: MermaidTheme) {
  if (mermaidInitedTheme === theme && mermaidMod) return mermaidMod.default;
  mermaidMod = mermaidMod ?? (await import("mermaid"));
  const mm = mermaidMod.default;
  mm.initialize({
    startOnLoad: false,
    theme,
    securityLevel: "strict",
    fontFamily: "system-ui, -apple-system, 'Segoe UI', sans-serif",
  });
  mermaidInitedTheme = theme;
  return mm;
}
