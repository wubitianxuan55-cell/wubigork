// 聊天代码块语法高亮——懒加载缝（对齐同类产品 AI 输出的可见短板）。
//
// 设计口径：
//  - highlight.js 走动态 import（独立 async chunk 不进入口体积；mermaid
//    异步 chunk 同款先例），首个代码块出现时才拉取。
//  - 语言白名单注册（后端/办公/数据/通用域 ~27 个），未注册语言回退纯文本
//    着色——宁缺勿硬造，与 lang.ts「纯语言解析不拉高亮器」的缝互补。
//  - 围栏语言别名归一复用 lang.ts ALIASES（编辑器/工具卡同源表）；toml
//    无官方语法，借 ini 语法着色（节+键结构一致），展示标签不受影响。
//  - hljs 产出仅含转义文本 + hljs-* span（自带转义），插入前再过统一
//    消毒层 DOMPurify 兜底（class 白名单保留，hljs-* 不受伤）。
//  - 明暗调色板色值在 styles.css（色值单一真源）：:root 兜底暗色、
//    :root[data-hl="light"] 浅色覆写；applyHlTheme 由主应用主题 effect
//    调用挂 dataset（darkMode 为 store 派生的实际明暗布尔）。
import type { LanguageFn } from "highlight.js";
import { ALIASES } from "./lang";
import { sanitizeHtml } from "./sanitize";

// 白名单：只注册常用域语言，控制 chunk 体积；语法定义各自独立模块。
const REGISTER: Record<string, () => Promise<{ default: LanguageFn }>> = {
  bash: () => import("highlight.js/lib/languages/bash"),
  c: () => import("highlight.js/lib/languages/c"),
  cpp: () => import("highlight.js/lib/languages/cpp"),
  csharp: () => import("highlight.js/lib/languages/csharp"),
  css: () => import("highlight.js/lib/languages/css"),
  diff: () => import("highlight.js/lib/languages/diff"),
  dockerfile: () => import("highlight.js/lib/languages/dockerfile"),
  go: () => import("highlight.js/lib/languages/go"),
  ini: () => import("highlight.js/lib/languages/ini"),
  java: () => import("highlight.js/lib/languages/java"),
  javascript: () => import("highlight.js/lib/languages/javascript"),
  json: () => import("highlight.js/lib/languages/json"),
  kotlin: () => import("highlight.js/lib/languages/kotlin"),
  lua: () => import("highlight.js/lib/languages/lua"),
  makefile: () => import("highlight.js/lib/languages/makefile"),
  markdown: () => import("highlight.js/lib/languages/markdown"),
  nginx: () => import("highlight.js/lib/languages/nginx"),
  perl: () => import("highlight.js/lib/languages/perl"),
  php: () => import("highlight.js/lib/languages/php"),
  powershell: () => import("highlight.js/lib/languages/powershell"),
  python: () => import("highlight.js/lib/languages/python"),
  ruby: () => import("highlight.js/lib/languages/ruby"),
  rust: () => import("highlight.js/lib/languages/rust"),
  sql: () => import("highlight.js/lib/languages/sql"),
  typescript: () => import("highlight.js/lib/languages/typescript"),
  xml: () => import("highlight.js/lib/languages/xml"),
  yaml: () => import("highlight.js/lib/languages/yaml"),
};

// 无官方 hljs 语法的语言借结构相同者着色（标签仍展示原语言名）。
const REMAP: Record<string, string> = { toml: "ini" };

type HljsCore = typeof import("highlight.js/lib/core")["default"];
let corePromise: Promise<HljsCore | null> | null = null;
const registered = new Set<string>();

function loadCore(): Promise<HljsCore | null> {
  corePromise ??= import("highlight.js/lib/core")
    .then((m) => m.default)
    .catch(() => null); // 拉取失败静默回退纯文本（增强面不挡主功能）
  return corePromise;
}

// 围栏语言 → 白名单语法名。返回 undefined = 不着色（纯文本/未知/明示纯文本）。
export function canonFenceLang(lang: string | undefined): string | undefined {
  const l = lang?.toLowerCase().trim();
  if (!l) return undefined;
  const aliased = ALIASES[l];
  const base = aliased !== undefined ? aliased : l; // ""=明示纯文本（text/txt/plaintext）
  if (base === "") return undefined;
  if (REGISTER[base]) return base;
  return REMAP[base] !== undefined && REGISTER[REMAP[base]] ? REMAP[base] : undefined;
}

// 高亮一段代码：返回消毒后的 HTML；不着色/失败返回 null（调用方回退纯文本）。
export async function highlightCode(code: string, lang: string | undefined): Promise<string | null> {
  const target = canonFenceLang(lang);
  if (!target) return null;
  const hljs = await loadCore();
  if (!hljs) return null;
  try {
    if (!registered.has(target)) {
      hljs.registerLanguage(target, (await REGISTER[target]()).default);
      registered.add(target);
    }
    return sanitizeHtml(hljs.highlight(code, { language: target }).value);
  } catch {
    return null;
  }
}

// 明暗主题挂钩：写 data-hl 供 styles.css 切换代码高亮调色板（暗色=缺省兜底）。
export function applyHlTheme(dark: boolean): void {
  document.documentElement.dataset.hl = dark ? "dark" : "light";
}
