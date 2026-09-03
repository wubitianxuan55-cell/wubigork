import { useCallback } from "react";
import { FolderTree } from "../icons";
import { FileTree, type RevealRequest } from "./FileTree";
import { RecentFilesBar } from "./RecentFilesBar";
import { app } from "../lib/bridge";
import { useComposerInsertStore } from "../lib/store";
import { recordRecentFile } from "../lib/recentFiles";
import { useToast } from "./Toast";

export type { RevealRequest };

// 资源管理器视图 tab（对标 better-sidebar 的 Files 首页 tab）。
// 行点击 → 新增/激活 pane 文件 tab（openFileTab 由 WorkspacePane 注入
// usePaneTabsStore.openFile）；右键「预览」保留主区双入口。
export function ExplorerView({
  cwd,
  refreshKey,
  revealRequest,
  openFileTab,
  openMainPreview,
}: {
  cwd?: string;
  refreshKey?: number;
  revealRequest?: RevealRequest | null;
  openFileTab: (rel: string) => void;
  openMainPreview?: (rel: string) => void;
}) {
  const toast = useToast();

  const reference = useCallback(
    (rel: string) => {
      if (!rel) return;
      useComposerInsertStore.getState().requestAt(rel);
      recordRecentFile(rel);
      toast.show(`已引用 @${rel}`, "info");
    },
    [toast],
  );

  const openExternal = useCallback((rel: string) => {
    void app.OpenWorkspacePath(rel).catch(() => {});
  }, []);

  const reveal = useCallback((rel: string) => {
    void app.RevealWorkspacePath(rel).catch(() => {});
  }, []);

  return (
    <div className="flex flex-col h-full min-h-0 text-(color:--md-sys-color-text-secondary) text-xs">
      <div className="v3-panel-head">
        <FolderTree size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">资源管理器</span>
      </div>
      {cwd && (
        <div
          className="px-3 py-1.5 font-mono text-[10px] truncate"
          title={cwd}
          style={{ borderBottom: "var(--v3-split)", color: "var(--md-sys-color-text-secondary)" }}
        >
          {cwd}
        </div>
      )}
      <RecentFilesBar cwd={cwd} onOpenFile={openFileTab} />
      <div className="flex-1 min-h-0 overflow-hidden">
        <FileTree
          key={refreshKey ?? 0}
          cwd={cwd}
          selectedFile={undefined}
          onSelect={openFileTab}
          onOpenMainPreview={openMainPreview}
          onReference={reference}
          onOpenExternal={openExternal}
          onReveal={reveal}
          revealRequest={revealRequest}
        />
      </div>
    </div>
  );
}
