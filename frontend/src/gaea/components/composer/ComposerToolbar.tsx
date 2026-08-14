// Composer 拆分产物：底部工具栏（工作区/导入/截图/权限级别/快捷键提示，行为零变化，T6-10.1）
import { Camera, ChevronDown, FolderGit2, Loader, Paperclip } from "../../icons";
import { useT } from "../../lib/i18n";

export interface ComposerToolbarProps {
  cwd?: string
  workspaceName: string
  workspaceMenuOpen: boolean
  onToggleWorkspaceMenu: () => void
  workspaceAnchorRef: React.RefObject<HTMLDivElement>
  running: boolean
  pendingPaste: number
  captureBusy: boolean
  onPickFiles: () => void
  onScreenshot: () => void
  permLevel?: string
  onSetPermLevel?: (p: "ask" | "auto" | "yolo") => void
}

const PERM_LABELS: Record<string, string> = { ask: "询问", auto: "自动", yolo: "⚡ YOLO" }
const PERM_DESCS: Record<string, string> = { ask: "写入前需确认（默认）", auto: "写入无需确认，deny 规则仍生效", yolo: "跳过所有确认提示" }

export function ComposerToolbar({
  cwd, workspaceName, workspaceMenuOpen, onToggleWorkspaceMenu, workspaceAnchorRef,
  running, pendingPaste, captureBusy, onPickFiles, onScreenshot,
  permLevel, onSetPermLevel,
}: ComposerToolbarProps) {
  const t = useT()
  return (
    <div className="flex items-center gap-1.5 min-w-0 px-2.5 py-1.5">
      {cwd && (
        <div className="relative inline-flex min-w-0" ref={workspaceAnchorRef}>
          <button
            className={`inline-flex items-center gap-1.5 max-w-60 px-2 py-1 border-0 rounded-md bg-transparent text-fg-dim text-xs cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-60 no-drag ${workspaceMenuOpen ? "text-fg bg-bg-soft" : ""}`}
            onClick={onToggleWorkspaceMenu}
            disabled={running}
            title={running ? t("common.busyHint") : t("status.switchFolder", { cwd })}
          >
            <FolderGit2 size={13} />
            <span className="min-w-0 truncate">{workspaceName}</span>
            <ChevronDown size={12} />
          </button>
        </div>
      )}

      {/* 导入文件按钮 */}
      <button
        className={`inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0 ${pendingPaste > 0 ? "pointer-events-none opacity-40" : ""}`}
        onClick={onPickFiles}
        disabled={running}
        title={running ? t("common.busyHint") : t("composer.importFile")}
      >
        <Paperclip size={14} />
      </button>

      {/* 截图按钮：整屏捕获后裁剪并附加 */}
      <button
        className={`inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0 ${captureBusy ? "pointer-events-none opacity-40" : ""}`}
        onClick={onScreenshot}
        disabled={running || captureBusy}
        title={running ? t("common.busyHint") : "截图：捕获屏幕并裁剪附加"}
      >
        {captureBusy ? <Loader size={14} className="animate-spin" /> : <Camera size={14} />}
      </button>

      {/* 权限级别选择器：询问 / 自动 / YOLO */}
      <div className="flex gap-[3px]">
        {(["ask", "auto", "yolo"] as const).map((level) => {
          const isYolo = level === "yolo"
          return (
            <button key={level} type="button"
              className={`flex items-center gap-1.5 px-2.5 py-1 border rounded-md bg-transparent text-xs cursor-pointer transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
                permLevel === level
                  ? isYolo ? "text-err bg-err/10 border-err/20 shadow-[0_0_0_1px_var(--err)]" : "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
                  : "text-fg-dim border-border-soft hover:text-fg hover:bg-bg-soft hover:border-fg-faint"
              }`}
              onClick={() => { if (permLevel !== level && onSetPermLevel) onSetPermLevel(level) }}
              title={PERM_DESCS[level]}
            >
              {PERM_LABELS[level]}
            </button>
          )
        })}
      </div>

      {/* 快捷提示 */}
      <span className="ml-auto text-fg-faint/40 text-[10px] select-none hidden sm:inline-flex items-center gap-1.5">
        <span>/ 命令</span>
        <span>@ 文件</span>
        {running && <span className="text-warn/60">Shift+Enter 纠正</span>}
      </span>
    </div>
  )
}
