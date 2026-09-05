// GenUI 块级交互状态与 action 发射。
import { createContext, useContext } from "react";

export interface QuestionMeta {
  label: string;
  answer: string | number;
  explanation?: string;
}

export interface BlockApi {
  answers: Record<string, string>;
  fields: Record<string, string>;
  meta: Record<string, QuestionMeta>;
  locked: boolean;
  /** 重做轮次：radio/quiz 以此为 key 重挂。 */
  round: number;
  hasAction: boolean;
  setAnswer: (group: string, label: string) => void;
  setField: (id: string, value: string) => void;
  registerMeta: (group: string, meta: QuestionMeta) => void;
  registerSecret: (id: string) => void;
  isSecret: (id: string) => boolean;
  lock: () => void;
  reset: (notifyAction?: string) => void;
  emit: (action: string, payload: Record<string, unknown>) => void;
}

export const BlockApiContext = createContext<BlockApi | null>(null);

export function useBlockApi(): BlockApi | null {
  return useContext(BlockApiContext);
}
