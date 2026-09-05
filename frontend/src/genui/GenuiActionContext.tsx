/* eslint-disable react-refresh/only-export-components -- provider + hook 同文件导出是 GenUI 约定 */
import { createContext, useContext, type ReactNode } from "react";

/** 宿主注入的 action 回调：组件动作 + 收集到的载荷。 */
export type GenuiActionHandler = (action: string, payload: Record<string, unknown>) => void;

const GenuiActionContext = createContext<GenuiActionHandler | undefined>(undefined);

/** 在宿主消息区包一层；缺省时 GenUI 组件进入纯展示态（action 按钮禁用）。 */
export function GenuiActionProvider({
  onAction,
  children,
}: {
  onAction?: GenuiActionHandler;
  children: ReactNode;
}) {
  return (
    <GenuiActionContext.Provider value={onAction}>
      {children}
    </GenuiActionContext.Provider>
  );
}

export function useGenuiAction(): GenuiActionHandler | undefined {
  return useContext(GenuiActionContext);
}
