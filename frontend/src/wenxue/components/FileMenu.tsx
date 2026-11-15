import { Folder, FileText } from "../icons";
import { MenuContainer, useMenuScroll } from "./MenuContainer";
import { extBadge } from "../lib/fileBadge";
import type { AtEntry } from "../lib/types";

export type { AtEntry };

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
              {extBadge(e.name)}
            </span>
          )}
        </button>
      ))}
    </MenuContainer>
  );
}
