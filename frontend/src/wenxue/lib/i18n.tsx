/* eslint-disable react-refresh/only-export-components -- i18n 基础设施：Provider + 翻译工具/hook 同文件（Context 模块惯例），非组件热更新场景 */
// Wenxue keeps its own dictionaries and DictKey surface, but shares the React
// context with the office shell so single-source components resolve the same t().
import { useCallback, useEffect, useReducer, useState, useMemo } from "react";
import type { ReactNode } from "react";
import { zh } from "../locales/zh";
import type { DictKey } from "../locales/en";
import {
  SharedI18nContext, useSharedI18n, useSharedI18nOptional, type SharedI18nValue,
} from "../../lib/i18nContext";

export type Locale = "en" | "zh" | "zh-TW";
export type LangPref = "" | "en" | "zh" | "zh-TW";

const DICTS: Partial<Record<Locale, Record<DictKey, string>>> = { zh };
const loadedLocales = new Set<Locale>(["zh"]);

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

function translate(locale: Locale, key: DictKey, vars?: Record<string, string | number>): string {
  const s = DICTS[locale]?.[key] ?? DICTS.zh?.[key] ?? DICTS.en?.[key] ?? key;
  if (!vars) return s;
  return s.replace(/\{(\w+)\}/g, (_, k) => (vars[k] !== undefined ? String(vars[k]) : `{${k}}`));
}

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

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [pref, setPrefState] = useState<LangPref>(() => readPref());
  const locale = detectLocale(pref);
  currentLocale = locale;
  const [, forceRender] = useReducer((c: number) => c + 1, 0);

  useEffect(() => {
    if (locale === "zh") return;
    let alive = true;
    void loadLocale(locale).then(() => {
      if (alive) forceRender();
    });
    return () => { alive = false; };
  }, [locale]);

  const setPref = useCallback((next: LangPref) => {
    writePref(next);
    setPrefState(next);
  }, []);

  const tt = useCallback<Translator>((key, vars) => translate(detectLocale(pref), key, vars), [pref]);
  const value = useMemo<SharedI18nValue>(() => ({
    locale,
    pref,
    setPref: (next) => setPref(next as LangPref),
    t: tt as SharedI18nValue["t"],
  }), [locale, pref, setPref, tt]);

  return <SharedI18nContext.Provider value={value}>{children}</SharedI18nContext.Provider>;
}

export function useI18n(): I18nValue {
  const ctx = useSharedI18n();
  return {
    locale: ctx.locale as Locale,
    pref: ctx.pref as LangPref,
    setPref: ctx.setPref as (pref: LangPref) => void,
    t: ctx.t as Translator,
  };
}

export function useT(): Translator {
  return useI18n().t;
}

export function useTOptional(): Translator {
  const ctx = useSharedI18nOptional();
  const fallback = useCallback<Translator>((key, vars) => translate("zh", key, vars), []);
  return ctx ? ctx.t as Translator : fallback;
}
