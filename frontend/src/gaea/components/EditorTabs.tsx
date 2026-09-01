import { useCallback } from "react";
import { File, X } from "../icons";
import { useEditorTabsStore } from "../lib/editorTabs";
import { FilePreview } from "./FilePreview";

// EditorTabs — 右栏「文件工作台」多文件编辑器 tab 区（v4.25 A3）。
//
// Why: A3 规划要求文件树点开 → 右栏内多文件编辑器 tab，复用现有预览/编辑
// 能力；主区预览 pane 保留（双入口）。tab 状态在 lib/editorTabs 模块级
// zustand store（App 侧 sidebar_open 工具事件可程序化开 tab），本组件只做
// 渲染消费——框架与状态解耦，学 v4.23 sidebarRegistry 的分层。
//
// How to apply: WorkspacePanel 直接挂 <EditorTabs />；打开/关闭走 store
// （文件树行点击、最近文件 chip、App 命令式 openEditorTab 均汇入同一状态）。
// 内容区复用主区 FilePreview（embedded 收窄头部，docx/xlsx/md/图片/预览内
// 编辑能力原样保留——红线「换壳不换芯」）。tab 条样式语言对齐 ChatTabs
// （激活态 accent + 下划线），全部令牌化零硬编码色值。
export function EditorTabs() {
  const tabs = useEditorTabsStore((s) => s.tabs);
  const active = useEditorTabsStore((s) => s.active);
  const activate = useEditorTabsStore((s) => s.activate);
  const close = useEditorTabsStore((s) => s.close);

  // 关闭当前预览文件 = 关该 tab（FilePreview 头部 X 与 tab 条 X 同源）
  const closeActive = useCallback(() => {
    const a = useEditorTabsStore.getState().active;
    if (a) useEditorTabsStore.getState().close(a);
  }, []);

  // 空态：尚未从文件树打开任何文件
  if (tabs.length === 0) {
    return (
      <div
        className="flex flex-col items-center justify-center h-full gap-2 px-6 text-center"
        data-testid="editor-tabs-empty"
      >
        <File size={22} aria-hidden className="opacity-30 text-fg-faint" />
        <span className="text-[11px] leading-relaxed text-fg-faint">从左侧文件树打开文件</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-h-0" data-testid="editor-tabs">
      {/* tab 条：文件名 + 关闭 X + 激活态（横向滚动容纳多 tab） */}
      <div
        className="flex items-stretch gap-0.5 px-1.5 pt-1 overflow-x-auto shrink-0 border-b border-border-soft"
        role="tablist"
        aria-label="编辑器标签页"
      >
        {tabs.map((p) => {
          const name = p.split(/[\\/]/).pop() ?? p;
          const isActive = p === active;
          return (
            <div
              key={p}
              role="tab"
              aria-selected={isActive}
              tabIndex={0}
              title={p}
              className={`relative flex items-center gap-1 pl-2 pr-1 py-1 rounded-t-md border-0 bg-transparent cursor-pointer text-[11px] select-none shrink-0 transition-colors ${
                isActive ? "text-accent bg-accent/10" : "text-fg-dim hover:bg-bg-soft"
              }`}
              onClick={() => activate(p)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  activate(p);
                }
              }}
            >
              <span className="truncate max-w-[120px]">{name}</span>
              {isActive && <span className="absolute inset-x-1 bottom-0 h-0.5 rounded-full bg-accent" />}
              <button
                type="button"
                aria-label={`关闭 ${name}`}
                title={`关闭 ${name}`}
                className="flex items-center justify-center w-4 h-4 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors shrink-0"
                onClick={(e) => {
                  e.stopPropagation();
                  close(p);
                }}
              >
                <X size={9} />
              </button>
            </div>
          );
        })}
      </div>

      {/* 内容区：复用主区 FilePreview（编辑/OCR/docx/xlsx 能力原样保留） */}
      {active && (
        <div className="flex-1 min-h-0">
          <FilePreview relPath={active} onClose={closeActive} embedded />
        </div>
      )}
    </div>
  );
}
