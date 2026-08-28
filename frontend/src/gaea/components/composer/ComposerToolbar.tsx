// Composer 拆分产物：底部工具栏（工作区/导入/截图/权限级别/思考深度/快捷键提示，行为零变化，T6-10.1）
import { Camera, ChevronDown, FolderGit2, Gauge, Loader, Paperclip, Zap } from "../../icons";
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
  sessionMode?: string
  onSetSessionMode?: (m: "default" | "plan") => void
  thinkLevel?: string
  onSetThinkLevel?: (level: "fast" | "normal" | "deep") => void
}

const PERM_LABELS: Record<string, string> = { ask: "询问", auto: "自动", yolo: "YOLO" }
const PERM_DESCS: Record<string, string> = { ask: "写入前需确认（默认）", auto: "写入无需确认，deny 规则仍生效", yolo: "跳过所有确认提示" }
const MODE_LABELS: Record<string, string> = { default: "默认", plan: "方案" }
const MODE_DESCS: Record<string, string> = {
  default: "默认模式 — 复杂任务自动规划",
  plan: "方案模式 — 每轮先出开工计划，确认后执行",
}
const THINK_LABELS: Record<string, string> = { fast: "快速", normal: "标准", deep: "深度" }
const THINK_DESCS: Record<string, string> = {
  fast: "思考温度 0.1 — 快而直接，适合简单任务",
  normal: "思考温度 0.3 — 平衡质量与速度（默认）",
  deep: "思考温度 0.7 — 更发散，适合复杂方案",
}

export function ComposerToolbar({
  cwd, workspaceName, workspaceMenuOpen, onToggleWorkspaceMenu, workspaceAnchorRef,
  running, pendingPaste, captureBusy, onPickFiles, onScreenshot,
  permLevel, onSetPermLevel, sessionMode, onSetSessionMode, thinkLevel, onSetThinkLevel,
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
              {level === "yolo" ? (
                <><Zap size={11} className="shrink-0" /><span>YOLO</span></>
              ) : (
                PERM_LABELS[level]
              )}
            </button>
          )
        })}
      </div>

      {/* 会话模式选择器（蒸馏自 codex ModeKind）：默认 / 方案 */}
      <div className="flex gap-[3px]" aria-label="会话模式">
        {(["default", "plan"] as const).map((mode) => (
          <button key={mode} type="button"
            className={`flex items-center gap-1 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
              sessionMode === mode
                ? "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
                : "text-fg-faint border-transparent hover:text-fg hover:bg-bg-soft"
            }`}
            onClick={() => { if (sessionMode !== mode && onSetSessionMode) onSetSessionMode(mode) }}
            title={MODE_DESCS[mode]}
            aria-pressed={sessionMode === mode}
          >
            <span>{MODE_LABELS[mode]}</span>
          </button>
        ))}
      </div>

      {/* 思考深度选择器：快速 / 标准 / 深度（映射到 SetAgentParams 温度） */}
      <div className="flex gap-[3px]" aria-label="思考深度">
        {(["fast", "normal", "deep"] as const).map((level) => (
          <button key={level} type="button"
            className={`flex items-center gap-1 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
              thinkLevel === level
                ? "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
                : "text-fg-faint border-transparent hover:text-fg hover:bg-bg-soft"
            }`}
            onClick={() => { if (thinkLevel !== level && onSetThinkLevel) void onSetThinkLevel(level) }}
            title={THINK_DESCS[level]}
            aria-pressed={thinkLevel === level}
          >
            <Gauge size={11} className="shrink-0" />
            <span>{THINK_LABELS[level]}</span>
          </button>
        ))}
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
