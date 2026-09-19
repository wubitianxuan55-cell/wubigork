import { memo, useSyncExternalStore } from "react";
import { Clock } from "../icons";
import { loadRecentFiles, recordRecentFile, subscribeRecentFiles } from "../lib/recentFiles";
import type { AtEntry } from "../lib/types";
import { FileChip } from "./FileChip";

// RecentFilesBar — 最近文件快捷区（调研 2026-08-16 P0-3）。
// 复用 @ 引用菜单同一份 localStorage 最近文件（lib/recentFiles 单源），
// 一键回到刚看过的文件；点击 chip 打开预览并再次置顶该文件。
// 空态不渲染。v4.349：改为订阅单源（useSyncExternalStore）——原先只在 cwd 变化时
// 读一次，「办公里新看过的文件」在本条里不会出现，直到切工作区才补上。
export const RecentFilesBar = memo(function RecentFilesBar({
  cwd,
  onOpenFile,
}: {
  cwd?: string;
  onOpenFile: (path: string) => void;
}) {
  // cwd 仍在契约里（调用方照旧传）：v4.349 起改为订阅单源，不再靠 cwd 变化重读。
  void cwd;
  const recent: AtEntry[] = useSyncExternalStore(subscribeRecentFiles, loadRecentFiles, () => []);

  if (recent.length === 0) return null;

  return (
    <div className="flex items-center gap-1 px-3 py-1.5 overflow-x-auto">
      <Clock
        size={11}
        aria-hidden
        className="shrink-0 text-(color:--md-sys-color-text-secondary)"
      />
      <span className="shrink-0 text-[10px] text-(color:--md-sys-color-text-secondary)">最近</span>
      <div className="flex items-center gap-1 min-w-0">
        {recent.map((e) => (
          <FileChip
            key={e.path}
            path={e.path}
            onOpen={(p) => { recordRecentFile(p); onOpenFile(p); }}
          />
        ))}
      </div>
    </div>
  );
});
