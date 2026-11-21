// Composer 拆分产物：底部工具栏（工作区/导入/截图/参数/计划/快捷键提示）。
// 简化批（对标 ExportMenu v4.29「新动作只进菜单不加常驻按钮」先例）：原「询问/自动/YOLO
// ×3 + 快速/标准/深度 ×3」六个同形态 chip 常驻收进一个「参数」钮 + 上翻弹出层——
// 弹出层内 chip 的样式/文案/title/onClick 逐字保留（行为零变化，功能零删除），
// 触发钮文案实时显示当前取值（YOLO 激活时红色高亮，风险状态不藏）。
import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { Dropdown } from "antd";
import { Camera, ChevronDown, ClipboardList, FolderGit2, FolderOpen, Gauge, Globe, Image, Inbox, Layers, LineChart, Loader, Paperclip, Square, Wand2, Wrench, Zap } from "../../icons";
import { useT } from "../../lib/i18n";
import type { DictKey } from "../../locales/en";
import type { OfficeRunModeViewB } from "../../lib/bridge/model";

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
  /** 打开工具面板；2026-11-14 起为右栏一级 Tab。按 variant 派生候选（office=CAD/进度计划；wenxue=右栏 pane 全集）。 */
  onOpenTool?: (tool: string) => void
  /** 板块形态：决定「工具」下拉候选（不同面板工具不一样，2026-11-21）。 */
  variant?: "office" | "wenxue"
  permLevel?: string
  onSetPermLevel?: (p: "ask" | "auto" | "yolo") => void
  thinkLevel?: string
  onSetThinkLevel?: (level: "fast" | "normal" | "deep") => void
  /** 办公运行模式（2026-11-21 众神殿终案）：视图快照+选档回调。锁定=只显当前档。 */
  runMode?: OfficeRunModeViewB | null
  onSetRunMode?: (mode: string) => void
  /** 会话定档锁：true=会话已有内容/回合运行中（只显当前档，切档需新建会话）；
   *  false=新会话未发消息（三态可选，选档即定档重建生效）。 */
  runModeLocked?: boolean
  /** 计划模式开/关（v4.420）：由最近一条计划回执推导，点击经 /plan on|off 动词链翻转 */
  planActive?: boolean
  onTogglePlan?: () => void
  /** 上下文占用状态；落在输入卡底部工具栏右侧，不再单独占一行。 */
  contextStatus?: ReactNode
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

// 运行模式三档（2026-11-21 众神殿 pan_1795202763988 终案）：zh 直书（页面内容层
// zh-only 纪律同「计划」钮）；title=档位定位 + 「查看配置」诚实面（该档隐藏的
// 专用工具面 / 追加的技能清单，Go OfficeRunModeView.hide/extras 渲染）。
const RUNMODE_LABELS: Record<string, string> = { standard: "标准", cost: "造价", proposal: "方案" }
const RUNMODE_DESCS: Record<string, string> = {
  standard: "通用办公：造价族/CAD/进度计划/方案族专用工具不在本档（切档后进清单）。",
  cost: "造价+CAD：workcost 结构化测算三件（cost_project 读 / cost_project_apply 写 / cost_audit 机检）+ cad_ask 图纸识别 + 造价技能。",
  proposal: "方案+进度：longdoc_manifest 清单机检 + longdoc_build 按清单组装交付 + schedule_get/apply/analyze 进度计划 + 方案技能。",
}
const RUNMODE_FACE: Record<string, (hide: string[], extras: string[]) => string> = {
  standard: (hide) => `
本档隐藏：${hide.join("、")}`,
  cost: (_hide, extras) => `
本档技能：${extras.join("、")}`,
  proposal: (_hide, extras) => `
本档技能：${extras.join("、")}`,
}

export function ComposerToolbar({
  cwd, workspaceName, workspaceMenuOpen, onToggleWorkspaceMenu, workspaceAnchorRef,
  running, pendingPaste, captureBusy, onPickFiles, onScreenshot, onRecordSkill, onOpenTool, variant = "office",
  permLevel, onSetPermLevel, thinkLevel, onSetThinkLevel, runMode, onSetRunMode, runModeLocked, planActive, onTogglePlan, contextStatus,
}: ComposerToolbarProps) {
  const t = useT();
  // 弹层单开控制（2026-11-21 拆分批）：权限/模式/思考各自独立触发钮，
  // 同一时刻只开一个（Esc 关当前，点外部也关）。
  const [openControl, setOpenControl] = useState<null | "perm" | "mode" | "think">(null);
  const toggleControl = (c: "perm" | "mode" | "think") => setOpenControl((cur) => (cur === c ? null : c));

  // 工具下拉候选按板块派生（2026-11-21「不同面板工具不一样」）：office=CAD/进度
  // 计划；wenxue=右栏 pane 全集（wenxue 无 cad/schedule 域，gallery 为其专属面）。
  const toolItems =
    variant === "wenxue"
      ? [
          { key: "gallery", label: "插图", icon: <Image size={13} /> },
          { key: "files", label: "文件", icon: <FolderOpen size={13} /> },
          { key: "deliverables", label: "产物", icon: <Inbox size={13} /> },
          { key: "browser", label: "浏览器", icon: <Globe size={13} /> },
          { key: "git", label: "Git", icon: <FolderGit2 size={13} /> },
          { key: "ui", label: "GenUI", icon: <Square size={13} /> },
        ]
      : [
          { key: "cad", label: "CAD 图纸", icon: <Layers size={13} /> },
          { key: "schedule", label: "进度计划", icon: <LineChart size={13} /> },
        ];

  // Esc 关闭弹出层（ExportMenu 同款）
  useEffect(() => {
    if (!openControl) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenControl(null);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [openControl]);

  // 各触发钮文案 = 自身当前取值（2026-11-21 拆分批：模式选择与思考强度分开
  // 显示，不再并进「参数」弹层）。显式 aria-label 压平可访问名。
  const permText = permLevel === "yolo" ? "YOLO" : permLevel ? t(PERM_LABELS[permLevel]) : null;
  const thinkText = thinkLevel ? t(THINK_LABELS[thinkLevel]) : null;
  const rmLabel = (m: string) => RUNMODE_LABELS[m] ?? m;
  // 模式钮显当前会话实跑档（active）——三档都显（用户拍板：分开显示后模式是
  // 常驻状态钮，标准档不再藏名）。
  const runModeText = runMode ? rmLabel(runMode.active) : null;
  const yoloActive = permLevel === "yolo";
  const popupShell = "absolute bottom-[calc(100%+6px)] left-0 z-20 flex min-w-52 flex-col gap-2 rounded-lg border border-border-soft bg-bg-elev p-2";
  const maskEl = <span className="fixed inset-0 z-10 cursor-default" aria-hidden onClick={() => setOpenControl(null)} />;
  const triggerCls = (active: boolean, danger: boolean) => `office-toolbar-param flex items-center gap-1.5 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer whitespace-nowrap transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${danger ? "text-err bg-err/10 border-err/20 shadow-[0_0_0_1px_var(--err)]" : active ? "text-fg bg-bg-soft border-border-soft" : "text-fg-dim border-border-soft hover:text-fg hover:bg-bg-soft"}`;

  return (
    <div className="office-composer-toolbar flex items-center flex-wrap gap-x-1.5 gap-y-1 min-w-0 px-2.5 py-1.5">
      {cwd && (
        <div className="relative inline-flex min-w-0" ref={workspaceAnchorRef}>
          <button
            className={`office-toolbar-workspace inline-flex items-center gap-1.5 max-w-60 px-2 py-1 border-0 rounded-md bg-transparent text-fg-dim text-xs cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-60 no-drag ${workspaceMenuOpen ? "text-fg bg-bg-soft" : ""}`}
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
        className={`office-toolbar-icon inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0 ${pendingPaste > 0 ? "pointer-events-none opacity-40" : ""}`}
        onClick={onPickFiles}
        disabled={running}
        title={running ? t("common.busyHint") : t("composer.importFile")}
      >
        <Paperclip size={14} />
      </button>

      {/* 截图按钮：整屏捕获后裁剪并附加 */}
      <button
        className={`office-toolbar-icon inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0 ${captureBusy ? "pointer-events-none opacity-40" : ""}`}
        onClick={onScreenshot}
        disabled={running || captureBusy}
        title={running ? t("common.busyHint") : t("composer.screenshot")}
      >
        {captureBusy ? <Loader size={14} className="animate-spin" /> : <Camera size={14} />}
      </button>

      {/* 工具下拉：仅图文板块保留（2026-11-21 用户拍板：办公输入框删 CAD/进度
          工具钮——两族入口由运行模式承担，右栏一级 Tab 与命令面板入口不变）。
          wenxue 候选=右栏 pane 全集（插图/文件/产物/浏览器/Git/GenUI）。 */}
      {variant === "wenxue" && (
        <Dropdown
          trigger={["click"]}
          placement="topLeft"
          menu={{
            items: toolItems,
            onClick: ({ key }) => onOpenTool?.(key),
          }}
        >
          <button
            className="office-toolbar-icon inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0"
            disabled={running || !onOpenTool}
            title={onOpenTool ? `工具：${toolItems.map((it) => it.label).join(" / ")}` : "工具"}
            aria-label="工具"
            data-testid="composer-tools"
          >
            <Wrench size={14} />
          </button>
        </Dropdown>
      )}

      {/* 会话录制技能按钮（阶段七 7.2-1 演示式录制）：蒸馏当前会话为可复用技能 */}
      <button
        className="office-toolbar-icon inline-flex items-center justify-center w-[28px] h-[28px] border-0 rounded-md bg-transparent text-fg-dim cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-bg-soft disabled:cursor-default disabled:opacity-40 shrink-0"
        onClick={() => onRecordSkill?.()}
        disabled={running}
        title={running ? t("common.busyHint") : t("composer.recordSkill")}
        aria-label={t("composer.recordSkill")}
      >
        <Wand2 size={14} />
      </button>

      {/* 权限选择器（独立钮，2026-11-21 拆分批）：YOLO 激活红色高亮（风险状态
          一眼可见）。弹出层向上翻（工具栏贴屏底），透明遮罩点击外部关闭。 */}
      {permLevel && onSetPermLevel && (
        <div className="relative inline-flex shrink-0">
          <button type="button"
            className={triggerCls(openControl === "perm", yoloActive)}
            onClick={() => toggleControl("perm")}
            title={t("composer.permLabel")}
            aria-label={t("composer.permLabel")}
            aria-haspopup="menu"
            aria-expanded={openControl === "perm"}
          >
            <Gauge size={11} className="shrink-0" />
            <span>{permText}</span>
            <ChevronDown size={11} className={openControl === "perm" ? "rotate-180 transition-transform" : "transition-transform"} />
          </button>
          {openControl === "perm" && (
            <>
              {maskEl}
              <span
                role="menu"
                aria-label={t("composer.permLabel")}
                data-testid="composer-params-menu"
                className={popupShell}
                style={{ boxShadow: "var(--ds-shadow-dropdown)" }}
              >
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
              </span>
            </>
          )}
        </div>
      )}

      {/* 运行模式选择器（独立钮，office）：新会话三态可选即定档；会话已有内容/
          回合运行中锁定只显当前档（切档需新建会话——众神殿会话定档语义）。 */}
      {runMode && onSetRunMode && (
        <div className="relative inline-flex shrink-0">
          <button type="button"
            className={triggerCls(openControl === "mode", false)}
            onClick={() => toggleControl("mode")}
            title={runModeLocked
              ? `运行模式：${rmLabel(runMode.active)}档（会话已定档：切档需新建会话）`
              : `运行模式：${rmLabel(runMode.active)}（新会话选档即定档，引擎按新档重建）`}
            aria-label={t("composer.runModeLabel")}
            aria-haspopup="menu"
            aria-expanded={openControl === "mode"}
          >
            <Layers size={11} className="shrink-0" />
            <span>{runModeText}</span>
            <ChevronDown size={11} className={openControl === "mode" ? "rotate-180 transition-transform" : "transition-transform"} />
          </button>
          {openControl === "mode" && (
            <>
              {maskEl}
              <span
                role="menu"
                aria-label={t("composer.runModeLabel")}
                data-testid="composer-runmode-menu"
                className={popupShell}
                style={{ boxShadow: "var(--ds-shadow-dropdown)" }}
              >
                <div role="group" aria-label={t("composer.runModeLabel")}>
                  <div className="px-1 pb-1 text-[10px] text-fg-faint select-none">{t("composer.runModeLabel")}</div>
                  {runModeLocked ? (
                    <div className="flex items-center gap-1.5 px-1 py-1">
                      <span
                        className="flex items-center gap-1 text-[11px] rounded-md px-2 py-0.5"
                        style={{ color: "var(--gaea-glow)", background: "color-mix(in srgb, var(--gaea-glow) 14%, transparent)" }}
                      >
                        <Layers size={11} className="shrink-0" />
                        <span>{rmLabel(runMode.active)}档</span>
                      </span>
                      <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                        会话已定档：切档需新建会话
                      </span>
                    </div>
                  ) : (
                    <>
                      <div className="flex gap-[3px]">
                        {(["standard", "cost", "proposal"] as const).map((mode) => (
                          <button key={mode} type="button"
                            className={`flex items-center gap-1 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer whitespace-nowrap transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
                              runMode.active === mode
                                ? "text-accent bg-accent-soft border-accent/30 shadow-[0_0_0_1px_var(--accent-soft)]"
                                : "text-fg-dim border-border-soft hover:text-fg hover:bg-bg-soft hover:border-fg-faint"
                            }`}
                            onClick={() => { if (runMode.active !== mode) { onSetRunMode(mode); setOpenControl(null) } }}
                            title={RUNMODE_DESCS[mode] + RUNMODE_FACE[mode](runMode.hide, runMode.extras) + "\n选择即定档：本会话按新档重建生效"}
                            aria-pressed={runMode.active === mode}
                          >
                            <Layers size={11} className="shrink-0" />
                            <span>{RUNMODE_LABELS[mode]}</span>
                          </button>
                        ))}
                      </div>
                      <div className="px-1 pt-1 text-[10px] text-fg-faint select-none">新会话选档即定档（引擎按新档重建）；登记档同时成为默认档。</div>
                    </>
                  )}
                </div>
              </span>
            </>
          )}
        </div>
      )}

      {/* 思考深度选择器（独立钮，2026-11-21 拆分批）：与权限/模式同构。 */}
      {thinkLevel && onSetThinkLevel && (
        <div className="relative inline-flex shrink-0">
          <button type="button"
            className={triggerCls(openControl === "think", false)}
            onClick={() => toggleControl("think")}
            title={t("composer.thinkLabel")}
            aria-label={t("composer.thinkLabel")}
            aria-haspopup="menu"
            aria-expanded={openControl === "think"}
          >
            <Gauge size={11} className="shrink-0" />
            <span>{thinkText}</span>
            <ChevronDown size={11} className={openControl === "think" ? "rotate-180 transition-transform" : "transition-transform"} />
          </button>
          {openControl === "think" && (
            <>
              {maskEl}
              <span
                role="menu"
                aria-label={t("composer.thinkLabel")}
                data-testid="composer-think-menu"
                className={popupShell}
                style={{ boxShadow: "var(--ds-shadow-dropdown)" }}
              >
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
          className={`office-toolbar-plan flex items-center gap-1 px-2 py-1 border rounded-md bg-transparent text-xs cursor-pointer whitespace-nowrap transition-[color,background,border,transform] duration-[var(--dur-fast)] active:scale-[0.97] ${
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

      {/* 右侧状态区：上下文占用常驻；快捷提示降为低优先级辅助，窄窗先隐藏。 */}
      <div className="office-toolbar-status ml-auto flex min-w-0 shrink-0 items-center gap-2">
        {contextStatus && <div className="office-toolbar-context">{contextStatus}</div>}
        <span className="office-toolbar-hint text-fg-faint/40 text-[10px] select-none hidden sm:inline-flex items-center gap-1.5">
          <span>{t("composer.hintSlash")}</span>
          <span>{t("composer.hintAt")}</span>
          {running && <span className="text-warn/60">{t("composer.hintCorrect")}</span>}
        </span>
      </div>
    </div>
  )
}

