/* eslint-disable react-refresh/only-export-components -- ToastProvider 组件 + useToast hook 同文件（Context 模块） */
import { createContext, useContext, useState, useCallback, useMemo, type ReactNode } from "react";

type ToastKind = "info" | "warn" | "error";

interface ToastItem {
  id: number;
  text: string;
  kind: ToastKind;
}

interface ToastCtx {
  show: (text: string, kind?: ToastKind) => void;
}

const ToastContext = createContext<ToastCtx>({ show: () => {} });

let nextId = 1;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);

  const show = useCallback((text: string, kind: ToastKind = "info") => {
    const id = nextId++;
    setToasts((prev) => [...prev, { id, text, kind }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 2000);
  }, []);

  // 上下文值稳定化：消费方把 toast 放进 useEffect 依赖时，
  // 弹出一条 toast 不应导致其 effect 反复重跑（否则解析失败会触发
  // 无限重解析循环，UI 卡死、确认按钮始终无反应）。
  const ctx = useMemo(() => ({ show }), [show]);

  return (
    <ToastContext.Provider value={ctx}>
      {children}
      <div className="fixed top-3 left-1/2 -translate-x-1/2 z-[100] flex flex-col gap-1.5 pointer-events-none">
        {toasts.map((t) => (
          <div
            key={t.id}
            // T7-4：role 属性只加在 error 档（role=alert，失败需读屏干预），
            // info/warn 是短暂提示、不设 role——避免与页面内持久状态框的
            // role=status/alert 双匹配（CostCompareModal / CostImportModal
            // 等存量测试依赖单匹配）。
            role={t.kind === "error" ? "alert" : undefined}
            className={`px-4 py-1.5 rounded-lg text-xs border border-border animate-[toast-in_0.2s_ease-out] ${
              t.kind === "error"
                ? "bg-red-500/10 text-red-400 border-l-[3px] border-l-red-500"
                : t.kind === "warn"
                  ? "bg-bg-elev-2 text-fg-dim border-l-[3px] border-l-warning"
                  : "bg-bg-elev-2 text-fg-dim border-l-[3px] border-l-info"
            }`}
            style={{boxShadow: "var(--ds-shadow-dropdown)"}}
          >
            {t.text}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastCtx {
  return useContext(ToastContext);
}
