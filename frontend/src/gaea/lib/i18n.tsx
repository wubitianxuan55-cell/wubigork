/* eslint-disable react-refresh/only-export-components -- i18n 基础设施：Provider + 翻译工具/hook 同文件（Context 模块惯例），非组件热更新场景 */
// i18n is the desktop's localization seam. It mirrors theme.ts's "persist a choice
// and apply it" shape, but UI text must re-render on a switch, so the active locale
// lives in React state behind a context — flipping it re-renders the whole tree
// (App is a child of the provider). A module-level mirror (`currentLocale`) lets
// non-React code (lib/tools.ts) translate too; it stays fresh because the provider
// updates it on every render.
//
// Desktop UI language is intentionally frontend-only. The CLI/kernel may still
// have its own `language` config for prompts and terminal text, but switching the
// desktop setting must not rewrite config or rebuild the model controller.

import { createContext, useCallback, useContext, useEffect, useReducer, useState, useMemo } from "react";
import type { ReactNode } from "react";
import { zh } from "../locales/zh";
import type { DictKey } from "../locales/en";

export type Locale = "en" | "zh" | "zh-TW";
// LangPref is the stored preference: "" means auto-detect from the OS.
export type LangPref = "" | "en" | "zh" | "zh-TW";

// P4-H7：locale 按需加载。DICTS 初始只含静态默认语言 zh（运行时绝大多数会话，
// 首帧零异步零回退）；en / zh-TW 由 loadLocale() 首次需要时动态导入独立 chunk。
// Partial + ?.[key] 保证未加载语言的 translate 安全回退（已加载字典 → 裸键），
// useT/语言切换的同步 API 签名完全不变，只是切换瞬间多一次 chunk 加载。
const DICTS: Partial<Record<Locale, Record<DictKey, string>>> = { zh };
const loadedLocales = new Set<Locale>(["zh"]);

// 按需加载语言字典并注入 DICTS（幂等：已加载语言直接返回）。显式分支而非
// `import('../locales/' + lang)` 模板拼接——让 Vite/Rollup 静态解析出 en 与
// zh-TW 两个独立 chunk（模板拼接会因无法静态解析而构建失败）。
// 导出供测试在断言 en/zh-TW 译文前 await（H7 动态化后 chunk 异步就绪）。
export async function loadLocale(lang: Locale): Promise<void> {
  if (lang === "zh" || loadedLocales.has(lang)) return;
  if (lang === "en") {
    const mod = await import("../locales/en");
    DICTS.en = mod.en;
  } else {
    const mod = await import("../locales/zh-TW");
    DICTS["zh-TW"] = mod.zhTW;
  }
  loadedLocales.add(lang);
}
const STORAGE_KEY = "gaea-lang";

// currentLocale mirrors the active locale for callers outside React (lib/tools.ts).
let currentLocale: Locale = "en";


export function detectLocale(pref: LangPref): Locale {
  if (pref === "en" || pref === "zh" || pref === "zh-TW") return pref;
  const nav = typeof navigator !== "undefined" ? navigator.language.toLowerCase() : "en";
  if (nav === "zh-tw" || nav === "zh-hk") return "zh-TW";
  return nav.startsWith("zh") ? "zh" : "en";
}

function readPref(): LangPref {
  const v = typeof localStorage !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
  return v === "en" || v === "zh" || v === "zh-TW" ? v : "";
}

function writePref(pref: LangPref): void {
  try {
    if (pref) localStorage.setItem(STORAGE_KEY, pref);
    else localStorage.removeItem(STORAGE_KEY);
  } catch {
    /* private mode / no storage — the in-memory state still applies this session */
  }
}

// translate resolves a key for a locale and fills {placeholders}. Missing keys fall
// back to English, then to the raw key, so the UI never renders blank.
function translate(locale: Locale, key: DictKey, vars?: Record<string, string | number>): string {
  // zh 是静态恒可用字典（P4-H7 后 en/zh-TW 按需加载）。非 React 调用方
  // （lib/tools.ts 摘要口述）可能在任何 en chunk 就绪前触达——zh 兜底保证
  // 裸键永不外泄；已加载语言优先取自身译文，其次 zh，最后裸键。
  const s = DICTS[locale]?.[key] ?? DICTS.zh?.[key] ?? DICTS.en?.[key] ?? key;
  if (!vars) return s;
  return s.replace(/\{(\w+)\}/g, (_, k) => (vars[k] !== undefined ? String(vars[k]) : `{${k}}`));
}

// t is the non-reactive translator for code outside React (e.g. lib/tools.ts). It
// reads the module mirror, which the provider keeps in sync.
export function t(key: DictKey, vars?: Record<string, string | number>): string {
  return translate(currentLocale, key, vars);
}

export type Translator = (key: DictKey, vars?: Record<string, string | number>) => string;

interface I18nValue {
  locale: Locale;
  pref: LangPref;
  setPref: (pref: LangPref) => void;
  t: Translator;
}

const I18nContext = createContext<I18nValue | null>(null);

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [pref, setPrefState] = useState<LangPref>(() => readPref());
  const locale = detectLocale(pref);
  currentLocale = locale; // keep the mirror fresh for non-React callers
  const [, forceRender] = useReducer((c: number) => c + 1, 0);

  // P4-H7：动态语言（en/zh-TW）首次生效时字典尚未就绪——先按 detectLocale
  // 渲染（translate 回退到已加载字典/裸键，同步 API 不阻塞渲染），chunk 就绪
  // 后 force re-render 整树引入真译文。zh 为静态，首帧零异步。alive 标志防
  // 语言再切换后旧 chunk 落地时的过期重渲染（加载本身幂等）。
  useEffect(() => {
    if (locale === "zh") return;
    let alive = true;
    void loadLocale(locale).then(() => {
      if (alive) forceRender();
    });
    return () => { alive = false; };
  }, [locale]);

  // setPref updates only the live UI and the browser cache.
  const setPref = useCallback((next: LangPref) => {
    writePref(next);
    setPrefState(next);
  }, []);

  const tt = useCallback<Translator>((key, vars) => translate(detectLocale(pref), key, vars), [pref]);

  // v4.234：context value 必须 memo 化——forceRender（en chunk 就绪）每次
  // 渲染造新对象会把全部 context 消费者拖进重渲染级联；Markdown 成为消费者
  // 后（useTOptional），级联落在流式/弹层的中间态上会撕掉子组件状态
  // （MemCitationChip 弹层即此回归）。deps 变化才换新值。
  const value = useMemo(() => ({ locale, pref, setPref, t: tt }), [locale, pref, setPref, tt]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within a LocaleProvider");
  return ctx;
}

// useT is the common shorthand: just the translator.
export function useT(): Translator {
  return useI18n().t;
}

// useTOptional：无 Provider 时回退 zh 直译而不抛错。供 Markdown/FilePreview
// 这类被裸渲染的共享组件用（测试/mock 面板/独立挂载）；gaea 板块运行态恒有
// Provider，真实三语不受影响。zh 静态字典恒可用（v4.177 回退链先例）。
export function useTOptional(): Translator {
  const ctx = useContext(I18nContext);
  const fallback = useCallback<Translator>((key, vars) => translate("zh", key, vars), []);
  return ctx ? ctx.t : fallback;
}
