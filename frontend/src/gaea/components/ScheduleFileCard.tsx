/**
 * ScheduleFileCard — 办公文件预览中的进度计划摘要卡（v4.121.0 刀12 办公联动）
 *
 * 办公板块预览 .gsched.json 时不再裸吐 JSON 文本：解析为摘要（总工期/关键/
 * 搭接/基线漂移/倒排校核），并提供联动动作——「在进度计划中打开」跳板块、
 * 「AI 进度周报」把周报指令发进共享工作线程（与进度计划板块左栏同一条线程）。
 * v4.139 #15 多工程（设计 §3.5）：「打开」从仅当前计划扩展为任意计划——非当前
 * 计划点击先切指针（GaeaScheduleProjectOpen，原「复制内容为当前计划」语义废弃：
 * 复制会制造两份同源文件漂移无从收拾）+ 通知板块重水合再跳转；isCurrentPlan 按
 * rel 与当前指针（store.currentPath，缺省回落常量）全等判定。周报仍仅当前计划
 * （schedule_get 缺省读当前指针）。
 * 解析失败时 FilePreview 端回落原始文本视图，本组件不渲染。
 */
import { useState } from "react";
import { ArrowDown, ArrowUp, ExternalLink, LineChart, Sparkles } from "../icons";
import { useT } from "../lib/i18n";
import { app } from "../lib/bridge";
import { useStore } from "../lib/store";
import { emitFrontendEvent, FRONTEND_EVENTS } from "../../events";
import { useToast } from "./Toast";
import { isScheduleFilePath, SCHEDULE_FILE_PATH, type GschedSummary } from "../../schedule/gschedSummary";
import { notifyScheduleFileChanged, useScheduleStore } from "../../schedule/store";

const HEAD_BTN =
  "flex items-center gap-1 px-1.5 py-0.5 border-0 rounded bg-transparent text-fg-dim text-[10px] cursor-pointer hover:bg-bg-soft";

function StatChip({ label, value, tone }: { label: string; value: string | number; tone?: "crit" | "good" }) {
  return (
    <span className="inline-flex items-baseline gap-1 px-2 py-0.5 rounded-md border border-border-soft bg-bg text-[11px]">
      <span className="text-fg-faint">{label}</span>
      <b className={tone === "crit" ? "text-err" : tone === "good" ? "text-emerald-500" : "text-fg"}>{value}</b>
    </span>
  );
}

export function ScheduleFileCard({ relPath, summary, raw }: { relPath: string; summary: GschedSummary; raw: string }) {
  const t = useT();
  const toast = useToast();
  const [showRaw, setShowRaw] = useState(false);
  const approval = useStore((s) => s.approval);
  const metaReady = useStore((s) => s.meta?.ready);
  // 审批挂起/内核未就绪时禁止发起（与 Composer disabled 同语义）
  const sendBlocked = approval != null || metaReady === false;
  // 当前指针（v4.139 #15）：store.currentPath 缺省回落常量（旧 store 形态=老行为）；
  // isCurrentPlan 按 rel 全等判定，不再按文件名后缀猜。
  const currentPath = useScheduleStore((s) => s.currentPath);
  const isCurrentPlan = isScheduleFilePath(relPath) &&
    relPath.replaceAll("\\", "/") === (currentPath ?? SCHEDULE_FILE_PATH);
  const p = summary.project;
  const drift = summary.baseline?.drift ?? 0;

  /** 周报指令发进共享工作线程：运行中走 Steer 插话（不落气泡），空闲走新回合 */
  const sendReport = () => {
    const prompt = t("schedCard.reportPrompt");
    const st = useStore.getState();
    if (st.running) {
      app.Steer(prompt).catch(() => toast.show(t("schedCard.sendFail"), "error"));
    } else {
      // 与 useController.send 同构：先落本地用户气泡，再提交后端
      st._dispatch({ type: "user", text: prompt });
      app.Submit(prompt).catch(() => toast.show(t("schedCard.sendFail"), "error"));
    }
    toast.show(t("schedCard.sent"), "info");
  };

  const jumpToSchedule = () => {
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: "schedule" });
  };

  /** 在进度计划中打开（v4.139 #15 切指针语义）：当前计划直接跳板块；非当前计划
   *  先 GaeaScheduleProjectOpen 切当前指针 + 通知板块重水合（notifyScheduleFileChanged
   *  既有事件复用）再跳。原「复制内容为当前计划」（saveScheduleFile 覆盖写）废弃。
   *  循环依赖计划也开放打开：指针切换不落盘，修正搭接在板块内完成。 */
  const openThisPlan = () => {
    if (isCurrentPlan) {
      jumpToSchedule();
      return;
    }
    app.ScheduleProjectOpen(relPath)
      .then(() => {
        notifyScheduleFileChanged();
        jumpToSchedule();
        toast.show(t("schedCard.copied"), "info"); // 「已切换为当前计划」
      })
      .catch((err) => toast.show(`${t("schedCard.copyFail")}：${err instanceof Error ? err.message : String(err)}`, "error"));
  };

  return (
    <div className="px-4 py-4 max-w-[860px] mx-auto" data-testid="sched-file-card">
      <div className="rounded-lg border border-border-soft bg-bg-soft/40 p-4 flex flex-col gap-3">
        <div className="flex items-center gap-2">
          <LineChart size={15} className="text-accent shrink-0" />
          <span className="font-medium text-fg text-[13px] truncate flex-1" title={relPath}>
            {p.name || t("schedCard.untitled")}
          </span>
          <span className="shrink-0 px-1.5 py-0.5 rounded-full border border-accent/30 bg-accent/10 text-accent text-[10px]">
            {t("schedCard.tag")}
          </span>
          <button className={HEAD_BTN} onClick={() => setShowRaw((v) => !v)} title={t("schedCard.rawTip")}>
            {showRaw ? <ArrowUp size={10} /> : <ArrowDown size={10} />}
            {t("schedCard.raw")}
          </button>
        </div>

        {!summary.ok ? (
          <div className="text-amber-500 text-[11px] leading-relaxed">{t("schedCard.cycle")}</div>
        ) : summary.taskCount === 0 ? (
          <div className="text-fg-dim text-[11px]">{t("schedCard.empty")}</div>
        ) : (
          <>
            <div className="flex flex-wrap gap-1.5">
              <StatChip label={t("schedCard.duration")} value={t("schedCard.days", { n: summary.duration })} />
              <StatChip label={t("schedCard.work")} value={summary.taskCount} />
              <StatChip label={t("schedCard.critical")} value={summary.criticalCount} tone="crit" />
              <StatChip label={t("schedCard.links")} value={summary.linkCount} />
              {summary.milestoneCount > 0 && (
                <StatChip label={t("schedCard.milestones")} value={summary.milestoneCount} />
              )}
            </div>
            <div className="text-fg-dim text-[11px]">
              {t("schedCard.range", { start: p.startDate, finish: summary.finishDate ?? "?" })}
            </div>
          </>
        )}

        {summary.baseline && (
          <div className="text-[11px] text-fg-dim">
            {t("schedCard.baseline", {
              name: summary.baseline.name,
              savedAt: summary.baseline.savedAt,
              base: summary.baseline.duration,
              now: summary.duration,
            })}
            {drift !== 0 && (
              <b className={drift > 0 ? "text-err" : "text-emerald-500"}>
                {t("schedCard.drift", { n: drift > 0 ? `+${drift}` : `${drift}` })}
              </b>
            )}
          </div>
        )}
        {summary.deadline && (
          <div className="text-[11px] text-fg-dim">
            {t("schedCard.deadline", { date: summary.deadline.date, target: summary.deadline.targetWorkdays })}
            {!summary.deadline.feasible && (
              <b className="text-err">{t("schedCard.deadlineOver", { n: summary.deadline.overrun })}</b>
            )}
            {summary.deadline.feasible && summary.deadline.overrun < 0 && (
              <b className="text-emerald-500">{t("schedCard.deadlineSlack", { n: -summary.deadline.overrun })}</b>
            )}
            {summary.deadline.feasible && summary.deadline.overrun === 0 && (
              <b className="text-emerald-500">{t("schedCard.deadlineExact")}</b>
            )}
          </div>
        )}

        {/* 联动动作行（任意计划）：非当前计划附说明文案（「打开」=切为当前工程+进板块） */}
        <div className="flex items-center gap-2 flex-wrap">
          {!isCurrentPlan && (
            <span className="text-fg-faint text-[10px] flex-1">{t("schedCard.notCurrent", { path: relPath })}</span>
          )}
          <button
            className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[11px] font-medium cursor-pointer border-0 ${isCurrentPlan ? "bg-accent text-bg hover:opacity-90" : "border border-accent/40 text-accent bg-transparent hover:bg-accent/10"}`}
            onClick={openThisPlan}
            title={isCurrentPlan ? undefined : t("schedCard.setAsCurrentTip")}
          >
            <ExternalLink size={11} />
            {t("schedCard.openInSchedule")}
          </button>
          {isCurrentPlan && (
            <button
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md border border-accent/40 text-accent bg-transparent text-[11px] cursor-pointer hover:bg-accent/10 disabled:opacity-50"
              onClick={sendReport}
              disabled={sendBlocked}
              title={t("schedCard.weeklyReportTip")}
            >
              <Sparkles size={11} />
              {t("schedCard.weeklyReport")}
            </button>
          )}
        </div>
      </div>

      {showRaw && (
        <pre className="mt-3 p-3 text-[12px] text-fg-dim font-mono leading-relaxed whitespace-pre-wrap overflow-x-auto rounded-lg border border-border-soft">{raw}</pre>
      )}
    </div>
  );
}
