import { useState, useCallback } from "react";
import { X, RefreshCw, ChevronRight, Home } from "lucide-react";
import { FileTree } from "./FileTree";
import { FilePreview } from "./FilePreview";

export function WorkspacePanel({
  cwd,
  onClose,
}: {
  open?: boolean;
  cwd?: string;
  maximized?: boolean;
  panelWidth?: number;
  onClose: () => void;
  onToggleMaximized?: () => void;
  onPreviewModeChange?: (v: boolean) => void;
  initialViewMode?: string;
}) {
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [treeWidth, setTreeWidth] = useState(220);
  const [resizing, setResizing] = useState(false);

  const onSelectFile = useCallback((rel: string) => {
    setSelectedFile(rel);
  }, []);

  // 拖拽调整文件树宽度
  const onDividerMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setResizing(true);
    const startX = e.clientX;
    const startW = treeWidth;
    const onMove = (ev: MouseEvent) => {
      const newW = Math.max(120, Math.min(450, startW + ev.clientX - startX));
      setTreeWidth(newW);
    };
    const onUp = () => {
      setResizing(false);
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
    };
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  }, [treeWidth]);

  // 面包屑路径
  const pathParts = selectedFile ? selectedFile.split("/") : [];
  const breadcrumbs = pathParts.map((part, i) => ({
    name: part,
    path: pathParts.slice(0, i + 1).join("/"),
  }));

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      {/* 标题栏 */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="font-semibold text-fg text-sm">工作区</span>
        <div className="flex items-center gap-1">
          <button
            className="flex items-center gap-1 px-2 py-1 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded text-[10px]"
            onClick={() => { setSelectedFile(null); }}
            title="回到根目录"
          >
            <Home size={11} />
          </button>
          <button
            className="flex items-center gap-1 px-2 py-1 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded text-[10px]"
            onClick={() => { /* trigger refresh by re-render */ }}
            title="刷新"
          >
            <RefreshCw size={11} />
          </button>
          <button className="border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg p-1" onClick={onClose}>
            <X size={14} />
          </button>
        </div>
      </div>

      {/* 当前路径 */}
      {cwd && (
        <div className="px-3 py-1.5 text-fg-faint font-mono text-[10px] truncate border-b border-border-soft" title={cwd}>
          {cwd}
        </div>
      )}

      {/* 文件树 + 预览 */}
      <div className="flex flex-1 overflow-hidden">
        {/* 左侧：文件树 */}
        <div style={{ width: treeWidth }} className="border-r border-border-soft overflow-hidden">
          <FileTree cwd={cwd} onSelect={onSelectFile} selectedFile={selectedFile ?? undefined} />
        </div>

        {/* 拖拽调整把手 */}
        <div
          className={`w-[3px] cursor-col-resize shrink-0 transition-colors hover:bg-accent/30 ${resizing ? "bg-accent/50" : "bg-transparent"}`}
          onMouseDown={onDividerMouseDown}
        />

        {/* 右侧：文件预览 */}
        <div className="flex-1 overflow-hidden">
          {/* 面包屑导航 */}
          {selectedFile && breadcrumbs.length > 0 && (
            <div className="flex items-center gap-0.5 px-3 py-1 border-b border-border-soft text-[10px] text-fg-faint overflow-x-auto">
              {breadcrumbs.map((crumb, i) => (
                <span key={crumb.path} className="flex items-center gap-0.5 whitespace-nowrap">
                  {i > 0 && <ChevronRight size={8} className="shrink-0" />}
                  <button
                    className="border-0 bg-transparent px-1 py-0.5 rounded cursor-pointer hover:bg-bg-soft hover:text-fg-dim transition-colors"
                    onClick={() => onSelectFile(crumb.path)}
                  >
                    {crumb.name}
                  </button>
                </span>
              ))}
            </div>
          )}
          <div className="flex-1 overflow-hidden">
            <FilePreview relPath={selectedFile} onClose={() => setSelectedFile(null)} />
          </div>
        </div>
      </div>
    </div>
  );
}
