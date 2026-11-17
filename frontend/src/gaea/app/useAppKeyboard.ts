// useAppKeyboard — 全局快捷键（新建/命令面板/历史/知识库/侧栏/工作台/专注）。
// 单一 keydown effect 按原 deps 数组原样迁自 App.tsx，事件监听/清理不变。
import { useEffect } from "react";
import type { ControllerState } from "../lib/store";
import { escOwner } from "../lib/escLadder";

interface UseAppKeyboardParams {
  state: Pick<ControllerState, "running" | "approval" | "ask">;
  closeTopmost: () => boolean;
  anyOpen: boolean;
  cancel: () => string | undefined;
  workspacePanelOpen: boolean;
  previewFile: string | null;
  toggleFocus: () => void;
  closeFilePreview: () => void;
  newSessionAndReset: () => Promise<void>;
  openHistory: () => Promise<void>;
  openKnowledge?: () => void;
  toggleSidebar: () => void;
  toggleWorkspacePanel: () => void;
  setPaletteOpen: (open: boolean) => void;
}

export function useAppKeyboard(params: UseAppKeyboardParams) {
  const {
    state, closeTopmost, workspacePanelOpen, previewFile, toggleFocus,
    closeFilePreview, newSessionAndReset, openHistory, openKnowledge,
    toggleSidebar, toggleWorkspacePanel, setPaletteOpen, anyOpen, cancel,
  } = params;

  // 全局快捷键
  useEffect(() => {
    const onKey = (e: Event) => {
      const ke = e as globalThis.KeyboardEvent;
      const mod = ke.ctrlKey || ke.metaKey, t = ke.target as HTMLElement;
      const inInput = t.tagName === "INPUT" || t.tagName === "TEXTAREA" || t.isContentEditable;
      if (ke.key === "Escape") {
        const owner = escOwner({
          modalOpen: state.approval != null || state.ask != null,
          previewOpen: previewFile !== null,
          overlayOpen: anyOpen,
          inInput,
          running: state.running,
          composerEmpty: true,
        });
        if (owner === "preview") {
          ke.preventDefault();
          closeFilePreview();
          return;
        }
        if (owner === "overlay" && closeTopmost()) {
          ke.preventDefault();
          return;
        }
        if (owner === "stop") {
          ke.preventDefault();
          cancel();
          return;
        }
        if (owner === "backtrack") {
          ke.preventDefault();
          return;
        }
        if (owner === "modal" || owner === "warnings" || owner === "questions" || owner === "banner" || owner === "composer") {
          return;
        }
      }
      if (!mod) return;
      if (ke.key === "n" && !state.running) { ke.preventDefault(); void newSessionAndReset(); return; }
      if (ke.key === "k") { ke.preventDefault(); setPaletteOpen(true); return; }
      if (ke.key === "H" && ke.shiftKey) { ke.preventDefault(); void openHistory(); return; }
      if (ke.key === "K" && ke.shiftKey) { ke.preventDefault(); openKnowledge?.(); return; }
      if (ke.key === "b") { ke.preventDefault(); toggleSidebar(); return; }
      if (ke.key === "j") { ke.preventDefault(); toggleWorkspacePanel(); return; }
      if (ke.key === "F" && ke.shiftKey) { ke.preventDefault(); toggleFocus(); return; }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setPaletteOpen 为 React dispatch（稳定引用），原 deps 数组语义保留
  }, [state.running, state.approval, state.ask, anyOpen, cancel, closeTopmost, workspacePanelOpen, previewFile, toggleFocus, closeFilePreview, newSessionAndReset, openHistory, openKnowledge, toggleSidebar, toggleWorkspacePanel]);
}
