// useDrawers — 统一管理办公工作台剩余抽屉（历史/能力/知识）。
// v4.73：「记忆」由左侧抽屉迁入主区 tab（App 层管理 memView），不再经抽屉路由。
import { useCallback, useState } from "react";
import type { SessionMeta } from "../lib/types";

export function useDrawers() {
  const [histView, setHistView] = useState<SessionMeta[] | null>(null);
  const [capsOpen, setCapsOpen] = useState(false);
  const [knowledgeOpen, setKnowledgeOpen] = useState(false);

  const anyOpen = histView !== null || capsOpen || knowledgeOpen;

  // Esc 分层关闭：能力 → 历史 → 知识（顺序与既有 App 快捷键一致；
  // 预览层的关闭由调用方先处理，此处只管抽屉）。
  const closeTopmost = useCallback((): boolean => {
    if (capsOpen) { setCapsOpen(false); return true; }
    if (histView !== null) { setHistView(null); return true; }
    if (knowledgeOpen) { setKnowledgeOpen(false); return true; }
    return false;
  }, [capsOpen, histView, knowledgeOpen]);

  return {
    histView, setHistView,
    capsOpen, setCapsOpen,
    knowledgeOpen, setKnowledgeOpen,
    anyOpen, closeTopmost,
  };
}
