// Composer 拆分产物：底部工具栏（工作区/导入/截图/参数/计划/快捷键提示）。
// 简化批（对标 ExportMenu v4.29「新动作只进菜单不加常驻按钮」先例）：原「询问/自动/YOLO
// ×3 + 快速/标准/深度 ×3」六个同形态 chip 常驻收进一个「参数」钮 + 上翻弹出层——
// 弹出层内 chip 的样式/文案/title/onClick 逐字保留（行为零变化，功能零删除），
// 触发钮文案实时显示当前取值（YOLO 激活时红色高亮，风险状态不藏）。
import { useEffect, useState } from "react";
import { Camera, ChevronDown, ClipboardList, FolderGit2, Gauge, Loader, Paperclip, Wand2, Zap } from "../../icons";
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
  onRecordSkill?: () => void
  permLevel?: string
  onSetPermLevel?: (p: "ask" | "auto" | "yolo") => void
  thinkLevel?: string
  onSetThinkLevel?: (level: "fast" | "normal" | "deep") => void
  /** 计划模式开/关（v4.420）：由最近一条计划回执推导，点击经 /plan on|off 动词链翻转 */
  planActive?: boolean
  onTogglePlan?: () => void
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
  running, pendingPaste, captureBusy, onPickFiles, onScreenshot, onRecordSkill,
  permLevel, onSetPermLevel, thinkLevel, onSetThinkLevel, planActive, onTogglePlan,
}: ComposerToolbarProps) {
  const t = useT();
  const [paramsOpen, setParamsOpen] = useState(false);

  // Esc 关闭弹出层（ExportMenu 同款）
  useEffect(() => {
    if (!paramsOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setParamsOpen(false);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [paramsOpen]);

  // 触发钮文案 = 当前取值摘要；显式 aria-label 压平可访问名（防图标 aria-label 混入）
  const permText = permLevel === "yolo" ? "YOLO" : permLevel ? t(PERM_LABELS[permLevel]) : null;
  const thinkText = thinkLevel ? t(THINK_LABELS[thinkLevel]) : null;
  const summary = [permText, thinkText].filter(Boolean).join(" · ");
  const yoloActive = permLevel === "yolo";

  return (
    <div className="flex items-center flex-wrap gap-x-1.5 gap-y-1 min-w-0 px-2.5 py-1.5">
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

      {/* 会话录制技能按钮（阶段七 7.2-1 演示式录制）：蒸馏当前会话为可复用技能 */}
      <button
        className="inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0"
        onClick={() => onRecordSkill?.()}
        disabled={running}
        title={running ? t("common.busyHint") : t("composer.recordSkill")}
        aria-label={t("composer.recordSkill")}
      >
        <Wand2 size={14} />
      </button>

      {/* 参数选择器（简化批）：权限级别 + 思考深度收进一个弹出层；YOLO 激活时触发钮
          红色高亮（风险状态一眼可见，不因收拢而隐身）。弹出层向上翻（工具栏贴屏底）。 */}
      {(permLevel || thinkLevel) && (
        <div className="relative inline-flex shrink-0">
          <button type="button"
            className={`flex items-center gap-1.5 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer whitespace-nowrap transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
              yoloActive
                ? "text-err bg-err/10 border-err/20 shadow-[0_0_0_1px_var(--err)]"
                : paramsOpen
                  ? "text-fg bg-bg-soft border-border-soft"
                  : "text-fg-dim border-border-soft hover:text-fg hover:bg-bg-soft"
            }`}
            onClick={() => setParamsOpen((o) => !o)}
            title={t("composer.params")}
            aria-label={t("composer.params")}
            aria-haspopup="menu"
            aria-expanded={paramsOpen}
          >
            <Gauge size={11} className="shrink-0" />
            <span>{summary}</span>
            <ChevronDown size={11} className={paramsOpen ? "rotate-180 transition-transform" : "transition-transform"} />
          </button>
          {paramsOpen && (
            <>
              {/* 透明遮罩：点击弹出层外部即关闭（ExportMenu 同款交互） */}
              <span className="fixed inset-0 z-10 cursor-default" aria-hidden onClick={() => setParamsOpen(false)} />
              <span
                role="menu"
                aria-label={t("composer.params")}
                data-testid="composer-params-menu"
                className="absolute bottom-[calc(100%+6px)] left-0 z-20 flex min-w-52 flex-col gap-2 rounded-lg border border-border-soft bg-bg-elev p-2"
                style={{ boxShadow: "var(--ds-shadow-dropdown)" }}
              >
                {permLevel && onSetPermLevel && (
                  <div role="group" aria-label={t("composer.permLabel")}>
                    <div className="px-1 pb-1 text-[10px] text-fg-faint select-none">{t("composer.permLabel")}</div>
                    <div className="flex gap-[3px]">
                      {(["ask", "auto", "yolo"] as const).map((level) => {
                        const isYolo = level === "yolo"
                        return (
                          <button key={level} type="button"
                            className={`flex items-center gap-1.5 px-2.5 py-1 border rounded-md bg-transparent text-xs cursor-pointer whitespace-nowrap transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
                              permLevel === level
                                ? isYolo ? "text-err bg-err/10 border-err/20 shadow-[0_0_0_1px_var(--err)]" : "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
                                : "text-fg-dim border-border-soft hover:text-fg hover:bg-bg-soft hover:border-fg-faint"
                            }`}
                            onClick={() => { if (permLevel !== level) onSetPermLevel(level) }}
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
                  </div>
                )}
                {thinkLevel && onSetThinkLevel && (
                  <div role="group" aria-label={t("composer.thinkLabel")}>
                    <div className="px-1 pb-1 text-[10px] text-fg-faint select-none">{t("composer.thinkLabel")}</div>
                    <div className="flex gap-[3px]">
                      {(["fast", "normal", "deep"] as const).map((level) => (
                        <button key={level} type="button"
                          className={`flex items-center gap-1 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer whitespace-nowrap transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
                            thinkLevel === level
                              ? "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
                              : "text-fg-faint border-transparent hover:text-fg hover:bg-bg-soft"
                          }`}
                          onClick={() => { if (thinkLevel !== level) void onSetThinkLevel(level) }}
                          title={t(THINK_DESCS[level])}
                          aria-pressed={thinkLevel === level}
                        >
                          <Gauge size={11} className="shrink-0" />
                          <span>{t(THINK_LABELS[level])}</span>
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </span>
            </>
          )}
        </div>
      )}

      {/* 计划模式选择器（v4.420）：非默认、手动选择——点击经 /plan on|off 动词链
          翻转，状态由最近一条计划回执推导；回合进行中引擎拒绝切换（置灰）。
          保持独立直钮：模式忘关会改变模型行为，须常驻可见。 */}
      <div className="flex gap-[3px] shrink-0">
        <button type="button"
          className={`flex items-center gap-1 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer whitespace-nowrap transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
            planActive
              ? "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
              : "text-fg-faint border-transparent hover:text-fg hover:bg-bg-soft"
          } ${running ? "opacity-50 cursor-not-allowed" : ""}`}
          onClick={() => { if (!running && onTogglePlan) onTogglePlan() }}
          title={planActive
            ? "计划模式已开启：点击退出（模型回到正常执行）"
            : "计划模式：模型只做研究和方案设计，提交计划审批后再执行"}
          aria-pressed={!!planActive}
        >
          <ClipboardList size={11} className="shrink-0" />
          <span>计划</span>
        </button>
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
