import { Folder, FileText } from "../icons";
import { MenuContainer, useMenuScroll } from "./MenuContainer";

// AtEntry 是 @ 菜单的统一条目（目录浏览 / 工作区搜索 / 最近使用文件）。
export interface AtEntry {
  path: string; // 工作区相对路径；目录以 / 结尾
  name: string;
  isDir: boolean;
  size?: number;
}

const BADGE_EXTS = new Set([
  "doc", "docx", "pdf", "xls", "xlsx", "ppt", "pptx", "md", "txt",
  "csv", "png", "jpg", "jpeg", "svg",
]);

function extBadge(name: string): string | null {
  const m = /\.([a-z0-9]+)$/i.exec(name);
  if (!m) return null;
  const ext = m[1].toLowerCase();
  return BADGE_EXTS.has(ext) ? ext : null;
}

export function FileMenu({
  items,
  activeIndex,
  onPick,
  onHover,
}: {
  items: AtEntry[];
  activeIndex: number;
  onPick: (e: AtEntry) => void;
  onHover: (i: number) => void;
}) {
  const activeRef = useMenuScroll(activeIndex);
  return (
    <MenuContainer count={items.length} hint="Tab 进入子目录">
      {items.map((e, i) => (
        <button
          key={(e.isDir ? "d:" : "f:") + e.path}
          ref={i === activeIndex ? activeRef : undefined}
          role="option"
          aria-selected={i === activeIndex}
          className={`flex items-baseline gap-2 w-full px-2 py-1.5 bg-transparent border-0 rounded-md text-inherit text-left cursor-pointer transition-colors duration-100 ${
            i === activeIndex ? "bg-accent-soft border-l-[2px] border-l-accent pl-[6px]" : "border-l-[2px] border-l-transparent pl-[6px]"
          }`}
          onMouseDown={(ev) => { ev.preventDefault(); onPick(e); }}
          onMouseMove={() => onHover(i)}
        >
          {e.isDir ? (
            <Folder size={13} className="text-accent shrink-0" />
          ) : (
            <FileText size={13} className="text-fg-faint shrink-0" />
          )}
          <span className="font-mono text-[13px] text-fg font-normal min-w-0 truncate">
            {e.name}
            {e.isDir ? "/" : ""}
          </span>
          {!e.isDir && (
            <span className="ml-auto shrink-0 text-[9px] uppercase text-fg-faint/60 border border-border-soft/60 rounded px-1 py-px font-mono">
              {extBadge(e.name) ?? "file"}
            </span>
          )}
        </button>
      ))}
    </MenuContainer>
  );
}
