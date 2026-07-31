import { useCallback, useEffect, useState } from "react";

/** 检测是否为移动端屏幕（宽度 < 768px） */
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(() =>
    typeof window !== "undefined" && window.innerWidth < 768
  );

  useEffect(() => {
    const check = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener("resize", check);
    return () => window.removeEventListener("resize", check);
  }, []);

  return isMobile;
}

/** 返回 safe-area-inset-* 的 CSS 变量值，用于 iPhone X+ 底部横条适配 */
export function useSafeArea(): { bottom: number; top: number } {
  const getInsets = useCallback(() => {
    const style = getComputedStyle(document.documentElement);
    const parse = (name: string) => {
      const val = style.getPropertyValue(name).trim();
      return val.endsWith("px") ? parseInt(val, 10) : 0;
    };
    return {
      bottom: parse("--sat-bottom") || parse("--safe-area-inset-bottom") || 0,
      top: parse("--sat-top") || parse("--safe-area-inset-top") || 0,
    };
  }, []);

  const [insets, setInsets] = useState(getInsets);

  useEffect(() => {
    // initial read
    setInsets(getInsets());
    // env() values may be applied after paint
    const id = setTimeout(() => setInsets(getInsets()), 300);
    return () => clearTimeout(id);
  }, [getInsets]);

  return insets;
}
