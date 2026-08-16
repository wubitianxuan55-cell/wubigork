// useDrawers — 统一管理办公工作台四个懒加载抽屉（记忆/历史/能力/知识）。
// 收敛 App.tsx 手管的 4 个独立状态（memView/histView/capsOpen/knowledgeOpen，
// 见 03-office-frontend.md §6 缺陷 9），提供打开/关闭 + Esc 分层关闭路由。
import { useCallback, useState } from "react";
import type { MemoryView, SessionMeta } from "../lib/types";

export function useDrawers() {
  const [memView, setMemView] = useState<MemoryView | null>(null);
  const [histView, setHistView] = useState<SessionMeta[] | null>(null);
  const [capsOpen, setCapsOpen] = useState(false);
  const [knowledgeOpen, setKnowledgeOpen] = useState(false);

  const anyOpen = memView !== null || histView !== null || capsOpen || knowledgeOpen;

  // Esc 分层关闭：能力 → 记忆 → 历史 → 知识（顺序与既有 App 快捷键一致；
  // 预览层的关闭由调用方先处理，此处只管抽屉）。
  const closeTopmost = useCallback((): boolean => {
    if (capsOpen) { setCapsOpen(false); return true; }
    if (memView !== null) { setMemView(null); return true; }
    if (histView !== null) { setHistView(null); return true; }
    if (knowledgeOpen) { setKnowledgeOpen(false); return true; }
    return false;
  }, [capsOpen, memView, histView, knowledgeOpen]);

  return {
    memView, setMemView,
    histView, setHistView,
    capsOpen, setCapsOpen,
    knowledgeOpen, setKnowledgeOpen,
    anyOpen, closeTopmost,
  };
}
