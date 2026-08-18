import { useCallback } from "react";
import { FolderTree, RefreshCw, X } from "../icons";
import { FileTree } from "./FileTree";
import { RecentFilesBar } from "./RecentFilesBar";

// 右侧面板：增强文件树（Codex 式工作区）。
// 点击文件后由 App 收起本面板，并在主区域展开可拖宽的预览。
// v3「星枢」面板语言：v3-panel-head 细条头部 + 令牌化容器，零硬编码色值。
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

  const headActionBtn =
    "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--color-text-secondary) cursor-pointer hover:text-(color:--color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

  return (
    <div className="flex flex-col h-full min-h-0 text-(color:--md-sys-color-text-secondary) text-xs">
      {/* 面板头部 — v3 细条（标题 + 操作区） */}
      <div className="v3-panel-head">
        <FolderTree size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">工作区</span>
        <span className="v3-panel-spacer" />
        <button
          className={headActionBtn}
          onClick={handleRefresh}
          title="刷新文件列表"
          aria-label="刷新文件列表"
        >
          <RefreshCw size={12} />
        </button>
        <button
          className={headActionBtn}
          onClick={onClose}
          title="收起面板"
          aria-label="收起面板"
        >
          <X size={14} />
        </button>
      </div>

      {/* 当前工作目录 */}
      {cwd && (
        <div
          className="px-3 py-1.5 font-mono text-[10px] truncate"
          title={cwd}
          style={{ borderBottom: "var(--v3-split)", color: "var(--md-sys-color-text-secondary)" }}
        >
          {cwd}
        </div>
      )}

      {/* 最近文件快捷区（P0-3：@ 引用/预览过的文件一键回到） */}
      <RecentFilesBar cwd={cwd} onOpenFile={onSelectFile} />

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
