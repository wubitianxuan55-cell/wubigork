// GenUI 宿主作用域：告诉共享内核当前在哪个板块/会话，用于状态 key 隔离。
/* eslint-disable react-refresh/only-export-components -- Provider + hook 同文件导出 */
import { createContext, useContext, type ReactNode } from "react";

export interface GenuiScope {
  scope: "chat" | "office";
  /** 会话/话题稳定 id（隔离状态与面板）。 */
  sessionKey: string;
}

const GenuiScopeContext = createContext<GenuiScope | null>(null);

export function GenuiScopeProvider({ scope, children }: { scope: GenuiScope | null; children: ReactNode }) {
  return <GenuiScopeContext.Provider value={scope}>{children}</GenuiScopeContext.Provider>;
}

export function useGenuiScope(): GenuiScope | null {
  return useContext(GenuiScopeContext);
}
