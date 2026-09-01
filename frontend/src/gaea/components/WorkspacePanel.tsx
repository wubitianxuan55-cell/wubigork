import { useCallback, useEffect, useRef, useState } from "react";
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
// 文件树行点击 / 最近文件 chip 在右栏内 EditorTabs 打开（多文件编辑器 tab，
// 复用 FilePreview 预览/编辑能力）；主区预览 pane 保留（双入口）——文件树
// 右键菜单「预览」仍走 onSelectFile 开主区。
//
// v4.27 右侧面板体验优化（对标 Codex 右栏）：点文件后编辑器**占满整个右栏**，
// 不再被下方的文件树挤压成顶部 3/5 小窗；文件树收敛为头部「文件」按钮切换的
// 侧栏（默认收起，打开第一个文件时自动收起），需要浏览时点一下即展开。
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
  onAutoWiden,
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
  /** 打开第一个文件时自动加宽右栏（App 侧接线；测试可缺省）。 */
  onAutoWiden?: () => void;
}) {
  const toast = useToast();
  // 编辑器 tab 状态（订阅：tabs 空 ↔ 非空决定「树全屏」/「编辑器全屏」两态）
  const tabs = useEditorTabsStore((s) => s.tabs);
  const activeFile = useEditorTabsStore((s) => s.active);
  // 编辑器态下的文件树侧栏开关：默认收起，点头部「文件」按钮展开
  const [treeOpen, setTreeOpen] = useState(false);
  const prevTabsLen = useRef(tabs.length);
  const handleRefresh = useCallback(() => onRefresh?.(), [onRefresh]);

  // 打开第一个文件时自动收起文件树，让预览/编辑器占满右栏（Codex 式）；
  // 关闭全部 tab 回到树全屏态时同样复位。
  useEffect(() => {
    if (tabs.length === 0 || prevTabsLen.current === 0) {
      setTreeOpen(false);
      if (tabs.length > 0) onAutoWiden?.();
    }
    prevTabsLen.current = tabs.length;
  }, [tabs.length, onAutoWiden]);

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
  const hasTabs = tabs.length > 0;
  const treeToggleBtn =
    "flex items-center gap-1.5 h-6 px-2 rounded-md border-0 bg-transparent cursor-pointer text-[11px] transition-colors " +
    (treeOpen
      ? "text-(color:--md-sys-color-primary) bg-(color:--md-sys-color-surface-container-high)"
      : "text-(color:--md-sys-color-text-secondary) hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high)");
  const fileTree = (
    <FileTree
      key={refreshKey ?? 0}
      cwd={cwd}
      // 高亮当前编辑文件（Codex 式 explorer 联动）：编辑器激活 tab 优先，
      // 无右栏 tab 时回退主区预览文件
      selectedFile={activeFile ?? selectedFile}
      onSelect={openInEditor}
      onOpenMainPreview={onSelectFile}
      onReference={reference}
      onOpenExternal={openExternal}
      onReveal={reveal}
      revealRequest={revealRequest}
    />
  );

  return (
    <div className="flex flex-col h-full min-h-0 text-(color:--md-sys-color-text-secondary) text-xs">
      {/* 面板头部 — v3 细条（标题 + 操作区） */}
      <div className="v3-panel-head">
        {hasTabs ? (
          <button
            type="button"
            className={treeToggleBtn}
            onClick={() => setTreeOpen((o) => !o)}
            aria-pressed={treeOpen}
            title={treeOpen ? "隐藏文件树" : "显示文件树"}
          >
            <FolderTree size={13} aria-hidden style={{ color: treeOpen ? "var(--gaea-glow)" : undefined }} />
            <span>文件</span>
          </button>
        ) : (
          <>
            <FolderTree size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
            <span className="v3-panel-title">工作区</span>
          </>
        )}
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

      {hasTabs ? (
        /* 编辑器态：默认编辑器占满右栏；「文件」展开后左侧 260px 树侧栏并排 */
        <div className="flex flex-1 min-h-0">
          {treeOpen && (
            <div className="flex flex-col min-h-0 shrink-0 border-r border-border-soft" style={{ width: 260 }}>
              <RecentFilesBar cwd={cwd} onOpenFile={openInEditor} />
              <div className="flex-1 min-h-0 overflow-hidden">{fileTree}</div>
            </div>
          )}
          <div className="flex-1 min-h-0 overflow-hidden">
            <EditorTabs />
          </div>
        </div>
      ) : (
        /* 树全屏态：无打开文件时整个右栏就是文件树（含最近文件快捷区） */
        <>
          <RecentFilesBar cwd={cwd} onOpenFile={openInEditor} />
          <div className="flex-1 min-h-0 overflow-hidden">{fileTree}</div>
        </>
      )}
    </div>
  );
}
