// Composer 拆分产物：工作区切换菜单（行为零变化，T6-10.1）
import { FolderGit2, FolderPlus, Search, Check } from "../../icons";
import { useT } from "../../lib/i18n";
import type { WorkspaceView } from "../../lib/types";

export interface ComposerWorkspaceMenuProps {
  menuRef: React.RefObject<HTMLDivElement>
  query: string
  onQueryChange: (q: string) => void
  workspaces: WorkspaceView[]
  onChoose: (path?: string) => void
  onClose: () => void
}

export function ComposerWorkspaceMenu({ menuRef, query, onQueryChange, workspaces, onChoose, onClose }: ComposerWorkspaceMenuProps) {
  const t = useT()
  return (
    <div
      className="absolute left-2.5 bottom-12 z-40 w-[min(320px,82vw)] p-2.5 border border-border rounded-xl bg-bg-elev anim-menu-in no-drag"
      style={{boxShadow: "var(--ds-shadow-dropdown)"}}
      ref={menuRef}
    >
      <label className="flex items-center gap-[7px] px-2 py-1.5 mb-1 border border-border-soft rounded-md bg-bg-soft focus-within:border-accent transition-colors">
        <Search size={14} className="text-fg-faint" />
        <input autoFocus className="flex-1 border-0 bg-transparent text-fg text-[13px] outline-none placeholder:text-fg-faint"
          value={query} onChange={(e) => onQueryChange(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Escape") onClose() }}
          placeholder={t("composer.searchProjects")} />
      </label>
      <div className="max-h-[280px] overflow-y-auto mb-1">
        {workspaces.map((w) => (
          <button key={w.path}
            className={`flex items-center gap-2.5 w-full px-2 py-1.5 bg-transparent border-0 rounded-lg text-left cursor-pointer transition-colors duration-100 ${w.current ? "text-accent bg-accent-soft font-medium" : "text-fg-dim hover:bg-bg-soft hover:text-fg"}`}
            onClick={() => { if (w.current) { onClose(); return } void onChoose(w.path) }}
            title={w.path}>
            <FolderGit2 size={15} className="shrink-0" />
            <span className="min-w-0 truncate flex-1 text-[13px]">{w.name}</span>
            {w.current && <Check size={15} className="text-accent shrink-0" />}
          </button>
        ))}
        {workspaces.length === 0 && <div className="py-4 text-fg-faint text-xs text-center">{t("composer.noProjectMatches")}</div>}
      </div>
      <div className="pt-1 border-t border-border-soft">
        <button className="flex items-center gap-2.5 w-full px-2 py-1.5 bg-transparent border-0 rounded-lg text-left cursor-pointer text-fg-dim hover:bg-bg-soft hover:text-fg text-[13px] transition-colors" onClick={() => void onChoose()}>
          <FolderPlus size={15} className="shrink-0" />
          <span>{t("composer.addProject")}</span>
        </button>
      </div>
    </div>
  )
}
