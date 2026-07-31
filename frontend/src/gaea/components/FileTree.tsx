import { useState, useEffect, useCallback } from "react";
import { ChevronRight, ChevronDown, File, Folder, Image, FileText } from "lucide-react";
import { app } from "../lib/bridge";
import type { DirEntry } from "../lib/types";

// 文件图标映射（按扩展名）
function fileIcon(name: string, isDir: boolean) {
  if (isDir) return <Folder size={14} className="text-accent shrink-0" />;
  const ext = name.split(".").pop()?.toLowerCase();
  if (["png", "jpg", "jpeg", "gif", "webp", "bmp", "svg"].includes(ext ?? ""))
    return <Image size={14} className="text-fg-dim shrink-0" />;
  if (["md", "txt", "json", "toml", "yaml", "yml", "csv", "xml", "html", "css", "js", "ts", "tsx", "jsx", "go", "py"].includes(ext ?? ""))
    return <FileText size={14} className="text-fg-dim shrink-0" />;
  return <File size={14} className="text-fg-faint shrink-0" />;
}

export function FileTree({
  cwd,
  onSelect,
  selectedFile,
}: {
  cwd?: string;
  onSelect: (rel: string) => void;
  selectedFile?: string;
}) {
  return (
    <div className="flex flex-col h-full text-[12px]">
      <div className="px-2 py-1.5 text-fg-faint text-[10px] font-semibold uppercase tracking-wider">文件</div>
      <div className="flex-1 overflow-y-auto">
        <DirNode
          relPath=""
          name={cwd ? cwd.split(/[/\\]/).pop() || "工作区" : "工作区"}
          depth={0}
          defaultOpen={true}
          onSelect={onSelect}
          selectedFile={selectedFile}
          isRoot
        />
      </div>
    </div>
  );
}

function DirNode({
  relPath,
  name,
  depth,
  defaultOpen,
  onSelect,
  selectedFile,
  isRoot,
}: {
  relPath: string;
  name: string;
  depth: number;
  defaultOpen?: boolean;
  onSelect: (rel: string) => void;
  selectedFile?: string;
  isRoot?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen ?? false);
  const [entries, setEntries] = useState<DirEntry[] | null>(null);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const es = await app.ListDir(relPath);
      setEntries(es ?? []);
    } catch { setEntries([]); }
    finally { setLoading(false); }
  }, [relPath]);

  useEffect(() => {
    if (open && entries === null) void load();
  }, [open, entries, load]);

  const toggle = () => {
    if (!open) setOpen(true);
    else setOpen(false);
  };

  return (
    <div>
      <button
        className={`w-full flex items-center gap-1 px-2 py-1 border-0 bg-transparent text-left cursor-pointer transition-colors hover:bg-bg-soft ${
          isRoot ? "font-semibold text-fg" : "text-fg-dim"
        }`}
        style={{ paddingLeft: `${8 + depth * 14}px` }}
        onClick={toggle}
      >
        {!isRoot && (
          open ? <ChevronDown size={10} className="shrink-0 text-fg-faint" /> : <ChevronRight size={10} className="shrink-0 text-fg-faint" />
        )}
        {fileIcon(name, true)}
        <span className="truncate flex-1">{name}</span>
        {loading && <span className="text-fg-faint text-[9px]">⋯</span>}
      </button>
      {open && entries && (
        <div>
          {entries
            .filter((e) => !e.name.startsWith("."))
            .map((e) => {
              const childPath = relPath ? `${relPath}/${e.name}` : e.name;
              if (e.isDir) {
                return (
                  <DirNode
                    key={childPath}
                    relPath={childPath}
                    name={e.name}
                    depth={depth + 1}
                    onSelect={onSelect}
                    selectedFile={selectedFile}
                  />
                );
              }
              const isSelected = selectedFile === childPath;
              return (
                <button
                  key={childPath}
                  className={`w-full flex items-center gap-1 px-2 py-0.5 border-0 text-left cursor-pointer transition-colors ${
                    isSelected ? "bg-accent/10 text-accent" : "text-fg-dim hover:bg-bg-soft"
                  }`}
                  style={{ paddingLeft: `${22 + (depth + 1) * 14}px` }}
                  onClick={() => onSelect(childPath)}
                >
                  {fileIcon(e.name, false)}
                  <span className="truncate">{e.name}</span>
                </button>
              );
            })}
          {entries.length === 0 && (
            <div className="text-fg-faint/40 text-[10px] text-center py-2" style={{ paddingLeft: `${22 + (depth + 1) * 14}px` }}>
              空目录
            </div>
          )}
        </div>
      )}
    </div>
  );
}
