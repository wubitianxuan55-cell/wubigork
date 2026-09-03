// Composer 拆分产物：底部工具栏（工作区/导入/截图/权限级别/思考深度/快捷键提示，行为零变化，T6-10.1）
import { Camera, ChevronDown, FolderGit2, Gauge, Loader, Paperclip, Zap } from "../../icons";
import { useT } from "../../lib/i18n";
import type { DictKey } from "../../locales/en";

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
  thinkLevel?: string
  onSetThinkLevel?: (level: "fast" | "normal" | "deep") => void
}

// 徽标/悬浮文案走三语字典（zh 原硬编码文案逐字收编）：键映射在模块层，
// 取值在渲染层（useT 后查表），语言切换即时生效。yolo 徽标为字面 YOLO 不需要键。
const PERM_LABELS: Record<string, DictKey> = { ask: "composer.permAsk", auto: "composer.permAuto" }
const PERM_DESCS: Record<string, DictKey> = { ask: "composer.permAskDesc", auto: "composer.permAutoDesc", yolo: "composer.permYoloDesc" }
const THINK_LABELS: Record<string, DictKey> = { fast: "composer.thinkFast", normal: "composer.thinkNormal", deep: "composer.thinkDeep" }
const THINK_DESCS: Record<string, DictKey> = {
  fast: "composer.thinkFastDesc",
  normal: "composer.thinkNormalDesc",
  deep: "composer.thinkDeepDesc",
}

export function ComposerToolbar({
  cwd, workspaceName, workspaceMenuOpen, onToggleWorkspaceMenu, workspaceAnchorRef,
  running, pendingPaste, captureBusy, onPickFiles, onScreenshot,
  permLevel, onSetPermLevel, thinkLevel, onSetThinkLevel,
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
        title={running ? t("common.busyHint") : t("composer.screenshot")}
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
              title={t(PERM_DESCS[level])}
            >
              {level === "yolo" ? (
                <><Zap size={11} className="shrink-0" /><span>YOLO</span></>
              ) : (
                t(PERM_LABELS[level])
              )}
            </button>
          )
        })}
      </div>

      {/* 思考深度选择器：快速 / 标准 / 深度（映射到 SetAgentParams 温度） */}
      <div className="flex gap-[3px]" aria-label={t("composer.thinkLabel")}>
        {(["fast", "normal", "deep"] as const).map((level) => (
          <button key={level} type="button"
            className={`flex items-center gap-1 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
              thinkLevel === level
                ? "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
                : "text-fg-faint border-transparent hover:text-fg hover:bg-bg-soft"
            }`}
            onClick={() => { if (thinkLevel !== level && onSetThinkLevel) void onSetThinkLevel(level) }}
            title={t(THINK_DESCS[level])}
            aria-pressed={thinkLevel === level}
          >
            <Gauge size={11} className="shrink-0" />
            <span>{t(THINK_LABELS[level])}</span>
          </button>
        ))}
      </div>

      {/* 快捷提示 */}
      <span className="ml-auto text-fg-faint/40 text-[10px] select-none hidden sm:inline-flex items-center gap-1.5">
        <span>{t("composer.hintSlash")}</span>
        <span>{t("composer.hintAt")}</span>
        {running && <span className="text-warn/60">{t("composer.hintCorrect")}</span>}
      </span>
    </div>
  )
}
