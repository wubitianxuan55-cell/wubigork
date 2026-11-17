import { createContext, useContext } from "react";

export interface SharedI18nValue {
  locale: string;
  pref: string;
  setPref: (pref: string) => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
}

export const SharedI18nContext = createContext<SharedI18nValue | null>(null);

export function useSharedI18nOptional(): SharedI18nValue | null {
  return useContext(SharedI18nContext);
}

export function useSharedI18n(): SharedI18nValue {
  const value = useSharedI18nOptional();
  if (!value) throw new Error("useI18n must be used within a LocaleProvider");
  return value;
}
