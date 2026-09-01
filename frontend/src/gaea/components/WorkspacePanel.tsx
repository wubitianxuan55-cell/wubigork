import { useCallback } from "react";
import { FolderTree, RefreshCw, X } from "../icons";
import { FileTree, type RevealRequest } from "./FileTree";
import { RecentFilesBar } from "./RecentFilesBar";
import { EditorTabs } from "./EditorTabs";
import { app } from "../lib/bridge";
import { useComposerInsertStore } from "../lib/store";
import { useEditorTabsStore } from "../lib/editorTabs";
import { recordRecentFile } from "../lib/recentFiles";
import { useToast } from "./Toast";

// v4.25 A3 转发导出：产物「树中定位」接线（App / sidebarRegistry）直接从
// WorkspacePanel 取请求形状，无需感知 FileTree 内部。
export type { RevealRequest };

// 右侧面板：文件工作台（Codex 式工作区，v4.25 A3「编辑器 tab 化」）。
// 文件树行点击 / 最近文件 chip 默认在右栏内 EditorTabs 打开（多文件编辑器
// tab，复用 FilePreview 预览/编辑能力）；主区预览 pane 保留（双入口）——
// 文件树右键菜单「预览」仍走 onSelectFile 开主区，本面板不再因打开文件收起。
// 树中定位：revealRequest nonce 变化 → FileTree 展开父链 + 滚动 + 高亮闪烁
// （产物面板「树中定位」经 App 接线汇入）。
// v3「星枢」面板语言：v3-panel-head 细条头部 + 令牌化容器，零硬编码色值。
export function WorkspacePanel({
  cwd,
  selectedFile,
  refreshKey,
  onClose,
  onSelectFile,
  onRefresh,
  revealRequest,
}: {
  cwd?: string;
  selectedFile?: string;
  refreshKey?: number;
  onClose: () => void;
  /** 主区预览入口（双入口保留）：文件树右键菜单「预览」经此开主区 pane。 */
  onSelectFile: (rel: string) => void;
  onRefresh?: () => void;
  /** 树中定位请求（产物面板 → 文件 tab）：nonce 变化触发一次定位。 */
  revealRequest?: RevealRequest | null;
}) {
  const toast = useToast();
  const handleRefresh = useCallback(() => onRefresh?.(), [onRefresh]);

  // 右栏内打开（v4.25 A3）：行点击/最近文件 chip 汇入编辑器 tab 状态机，
  // 不再走 onSelectFile 收面板开主区。store 为模块级 zustand（App 事件侧
  // 也能程序化 openEditorTab），命令式 getState() 即可，无需 hook 订阅。
  const openInEditor = useCallback((rel: string) => {
    if (!rel) return;
    useEditorTabsStore.getState().open(rel);
  }, []);

  // 行内 @ 引用：插入输入框 + 记入最近文件 + toast（与 MaterialsPanel 行为一致）。
  // 空路径（根行 @ 按钮引用工作区根）无引用意义，静默忽略。
  const reference = useCallback((rel: string) => {
    if (!rel) return;
    useComposerInsertStore.getState().requestAt(rel);
    recordRecentFile(rel);
    toast.show(`已引用 @${rel}`, "info");
  }, [toast]);

  const openExternal = useCallback((rel: string) => {
    void app.OpenWorkspacePath(rel).catch(() => {});
  }, []);

  const reveal = useCallback((rel: string) => {
    void app.RevealWorkspacePath(rel).catch(() => {});
  }, []);

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

      {/* 最近文件快捷区（P0-3：@ 引用/预览过的文件一键回到；A3 起开右栏内 tab） */}
      <RecentFilesBar cwd={cwd} onOpenFile={openInEditor} />

      {/* 编辑器 tab 区（A3）：多文件编辑器 + 空态提示；占上部 3/5（120px 下限防挤压） */}
      <div className="flex-[3] min-h-[120px] overflow-hidden" style={{ borderBottom: "var(--v3-split)" }}>
        <EditorTabs />
      </div>

      {/* 文件树：行点击开右栏内编辑器 tab；右键「预览」仍开主区（双入口保留） */}
      <div className="flex-[2] min-h-0 overflow-hidden">
        <FileTree
          key={refreshKey ?? 0}
          cwd={cwd}
          selectedFile={selectedFile}
          onSelect={openInEditor}
          onOpenMainPreview={onSelectFile}
          onReference={reference}
          onOpenExternal={openExternal}
          onReveal={reveal}
          revealRequest={revealRequest}
        />
      </div>
    </div>
  );
}
