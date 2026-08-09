import { useCallback } from "react";
import { FolderTree, RefreshCw, X } from "../icons";
import { FileTree } from "./FileTree";

// 右侧面板：增强文件树（Codex 式工作区）。
// 点击文件后由 App 收起本面板，并在主区域展开可拖宽的预览。
export function WorkspacePanel({
  cwd,
  selectedFile,
  refreshKey,
  onClose,
  onSelectFile,
  onRefresh,
}: {
  cwd?: string;
  selectedFile?: string;
  refreshKey?: number;
  onClose: () => void;
  onSelectFile: (rel: string) => void;
  onRefresh?: () => void;
}) {
  const handleRefresh = useCallback(() => onRefresh?.(), [onRefresh]);

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      {/* 面板头部 */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <FolderTree size={13} className="text-accent" />
          工作区
        </span>
        <div className="flex items-center gap-1">
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={handleRefresh}
            title="刷新文件列表"
          >
            <RefreshCw size={12} />
          </button>
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={onClose}
            title="收起面板"
          >
            <X size={14} />
          </button>
        </div>
      </div>

      {/* 当前工作目录 */}
      {cwd && (
        <div className="px-3 py-1.5 text-fg-faint font-mono text-[10px] truncate border-b border-border-soft" title={cwd}>
          {cwd}
        </div>
      )}

      {/* 文件树：点击文件后自动收起面板，在主区域展开预览 */}
      <div className="flex-1 min-h-0 overflow-hidden">
        <FileTree
          key={refreshKey ?? 0}
          cwd={cwd}
          selectedFile={selectedFile}
          onSelect={onSelectFile}
        />
      </div>
    </div>
  );
}
