// ScheduleTool — 进度计划工具面板（办公 ToolSheet 的内容件）。
//
// 边界（docs/gaea-ai-workbench-ui-2026-10.md §7.4）：进度计划不是工作台，是
// 办公对话里调用的工具——工具栏 + 甘特/网络图 + 质量体检 + 任务检查器（what-if
// 复用）+ 状态栏 +「问这个计划」回对话通道。与 SchedulePage 共享同一
// useScheduleStore（initScheduleSync 模块级幂等，同一计划文件、防抖自动保存、
// agent schedule_* 写入即时可见）；完整编辑面（导入导出/工程管理/资源/模板/
// 挣值模拟）仍在 SchedulePage 深链接，头部「完整工作台」一键直达。
// 零新绑定：视图与检查器全部复用 schedule/ 既有件。
import { useEffect, useMemo, useState } from "react";
import { Alert, Button, Segmented } from "antd";
import { ExternalLink } from "../../icons";
import { emitFrontendEvent, FRONTEND_EVENTS } from "../../../events";
import { useScheduleStore, initScheduleSync } from "../../../schedule/store";
import { computeCpm } from "../../../schedule/cpm";
import { buildAoa } from "../../../schedule/aoa";
import { checkDeadline, planFinishOf } from "../../../schedule/deadline";
import { computeBaselineDrift } from "../../../schedule/baseline";
import { depFreeSlacks, drivingLinkKeys, linkKey } from "../../../schedule/pathDriver";
import type { SchedLink } from "../../../schedule/types";
import { wbsOf } from "../../../schedule/ganttGroup";
import { GanttView } from "../../../schedule/GanttView";
import { PdmView } from "../../../schedule/PdmView";
import { AoaView } from "../../../schedule/AoaView";
import { DcmaView } from "../../../schedule/DcmaView";
import { TaskInspector, type InspectorDepRow } from "../../../schedule/TaskInspector";
import { WhatIfPanel } from "../../../schedule/WhatIfPanel";
import "../../../schedule/schedule.css";

/** 工具面板内的视图档位：完整工作台七视图的检查子集（编制向视图留深链接）。 */
const TOOL_VIEWS = ["gantt", "pdm", "aoa", "dcma"] as const;
type ToolView = (typeof TOOL_VIEWS)[number];

const VIEW_LABELS: Record<ToolView, string> = {
  gantt: "横道图",
  pdm: "单代号",
  aoa: "双代号",
  dcma: "质量体检",
};

export function ScheduleTool({ onAsk }: { onAsk?: (text: string) => void }) {
  const project = useScheduleStore((s) => s.project);
  const view = useScheduleStore((s) => s.view);
  const setView = useScheduleStore((s) => s.setView);
  const select = useScheduleStore((s) => s.select);
  const addTask = useScheduleStore((s) => s.addTask);
  const undo = useScheduleStore((s) => s.undo);
  const redo = useScheduleStore((s) => s.redo);
  const past = useScheduleStore((s) => s.past);
  const future = useScheduleStore((s) => s.future);
  const loadSample = useScheduleStore((s) => s.loadSample);
  const hydrated = useScheduleStore((s) => s.hydrated);
  const sync = useScheduleStore((s) => s.sync);
  const savedAt = useScheduleStore((s) => s.savedAt);
  const syncError = useScheduleStore((s) => s.syncError);
  /** 任务检查器（复用 SchedulePage 同款：右键/检查器入口 → 抽屉，what-if 挂 slot） */
  const [inspectId, setInspectId] = useState<string | null>(null);
  const [question, setQuestion] = useState("");

  useEffect(() => { void initScheduleSync() }, []);

  const cpm = useMemo(() => computeCpm(project.tasks, project.links, { planFinish: planFinishOf(project), calendar: project.calendar, startDate: project.startDate }), [project]);
  const aoa = useMemo(() => buildAoa(project.tasks, project.links, { planFinish: planFinishOf(project) }), [project]);
  const drift = useMemo(() => computeBaselineDrift(project, cpm), [project, cpm]);
  const dl = useMemo(() => checkDeadline(project, cpm), [project, cpm]);

  // 工具面板只认四档：store 视图停在编制档（usage/monte/evm）时回落横道图展示
  const activeView: ToolView = (TOOL_VIEWS as readonly string[]).includes(view) ? (view as ToolView) : "gantt";

  const leafCount = project.tasks.filter((t) => t.level > 0).length;
  const criticalCount = project.tasks.filter((t) => t.level > 0 && cpm.rows[t.id]?.critical).length;

  // 任务检查器数据（与 SchedulePage 同口径：逐依赖自由时差 + 驱动标记）
  const inspectTask = inspectId ? project.tasks.find((t) => t.id === inspectId) ?? null : null;
  const inspectorData = useMemo(() => {
    if (!inspectTask || inspectTask.level === 0) return null;
    const slacks = depFreeSlacks(project.tasks, project.links, cpm);
    const driving = drivingLinkKeys(project.tasks, project.links, cpm);
    const mkRow = (l: SchedLink, otherId: string): InspectorDepRow => ({
      link: l,
      otherId,
      otherName: project.tasks.find((t) => t.id === otherId)?.name ?? otherId,
      freeSlack: slacks[linkKey(l)] ?? 0,
      driving: driving.has(linkKey(l)),
    });
    return {
      predecessors: project.links.filter((l) => l.to === inspectTask.id).map((l) => mkRow(l, l.from)),
      successors: project.links.filter((l) => l.from === inspectTask.id).map((l) => mkRow(l, l.to)),
    };
  }, [inspectTask, project.tasks, project.links, cpm]);

  const ask = () => {
    const q = question.trim();
    if (!onAsk || !q) return;
    onAsk(`【进度计划：${project.name || "未命名工程"}】\n${q}`);
    setQuestion("");
  };

  const openFullWorkbench = () => {
    // 深链接回完整工作台（旧路由仍在：板块清单 inMenu:false，仅显式跳转可达）
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: "schedule" });
  };

  const syncText = !hydrated
    ? "读取计划…"
    : sync === "dirty"
      ? "改动待保存…"
      : sync === "saving"
        ? "保存中…"
        : sync === "error"
          ? syncError ?? "保存失败"
          : savedAt
            ? `已保存 ${savedAt}`
            : "已同步";

  return (
    <div className="flex h-full min-h-0 flex-col gap-2 p-2 text-[12px]" data-testid="sched-tool">
      {/* 工具栏：工程名 + 视图四档 + 撤销/重做 + 完整工作台深链接 */}
      <div className="flex shrink-0 items-center gap-2 overflow-x-auto">
        <b className="max-w-[180px] truncate text-[12px] font-medium" title={project.name}>
          {project.name || "未命名工程"}
        </b>
        <Segmented
          size="small"
          value={activeView}
          onChange={(v) => setView(v as ToolView)}
          options={TOOL_VIEWS.map((v) => ({ value: v, label: VIEW_LABELS[v] }))}
          data-testid="sched-tool-views"
        />
        <Button size="small" disabled={past.length === 0} onClick={() => undo()} aria-label="撤销" data-testid="sched-tool-undo">
          撤销
        </Button>
        <Button size="small" disabled={future.length === 0} onClick={() => redo()} aria-label="重做" data-testid="sched-tool-redo">
          重做
        </Button>
        <Button size="small" onClick={() => addTask()} data-testid="sched-tool-add">
          添加任务
        </Button>
        <span className="flex-1" />
        <Button size="small" icon={<ExternalLink size={11} />} onClick={openFullWorkbench} data-testid="sched-tool-full" title="打开完整编制工作台（导入导出/工程管理/资源/模板）">
          完整工作台
        </Button>
      </div>

      {!cpm.ok && cpm.error && (
        <Alert type="error" showIcon message={cpm.error} description="请修正搭接关系后重试；网络图视图在循环解除前不可用。" />
      )}

      {hydrated && leafCount === 0 ? (
        <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 rounded-lg border border-border-soft bg-bg-soft/30 p-6 text-center" data-testid="sched-tool-empty">
          <div className="text-[12px] text-fg-dim">还没有计划数据</div>
          <div className="text-[11px] text-fg-faint">在对话里让 AI 编排计划，或载入示例工程开始检查</div>
          <Button size="small" onClick={() => loadSample()} data-testid="sched-tool-empty-sample">
            载入示例工程
          </Button>
        </div>
      ) : (
        <div className="min-h-0 flex-1 overflow-hidden">
          {activeView === "gantt" && <GanttView project={project} cpm={cpm} onInspect={setInspectId} />}
          {activeView === "pdm" && <PdmView project={project} cpm={cpm} />}
          {activeView === "aoa" && <AoaView graph={aoa} tasks={project.tasks} />}
          {activeView === "dcma" && <DcmaView project={project} cpm={cpm} />}
        </div>
      )}

      {/* 任务检查器：与 SchedulePage 同款（what-if 推演挂 slot，计算全在 WhatIfPanel） */}
      <TaskInspector
        open={!!inspectTask && inspectTask.level > 0 && !!inspectorData}
        task={inspectTask && inspectTask.level > 0 ? inspectTask : null}
        row={inspectTask ? cpm.rows[inspectTask.id] ?? null : null}
        wbs={inspectTask ? wbsOf(project.tasks)[project.tasks.findIndex((t) => t.id === inspectTask.id)] : undefined}
        predecessors={inspectorData?.predecessors ?? []}
        successors={inspectorData?.successors ?? []}
        startDate={project.startDate}
        calendar={project.calendar}
        whatIf={inspectTask && inspectTask.level > 0 ? (
          <WhatIfPanel project={project} taskId={inspectTask.id} />
        ) : undefined}
        onClose={() => setInspectId(null)}
        onNavigate={(id) => { setInspectId(id); select(id) }}
      />

      {/* 状态栏 + 问计划：结论常驻一行；问题发回对话（面板关闭，回答在对话里继续） */}
      <div className="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-t border-border-soft pt-1.5 text-[11px] text-fg-dim" data-testid="sched-tool-statusbar">
        <span>共 <b className="font-mono tabular-nums text-fg">{leafCount}</b> 项工作</span>
        <span>总工期 <b className="font-mono tabular-nums text-fg">{cpm.duration}</b> 天</span>
        <span>关键 <b className="font-mono tabular-nums text-err">{criticalCount}</b> 项</span>
        {drift && drift.durationDrift !== 0 && (
          <span title={`基线「${drift.baselineName}」保存于 ${drift.baselineSavedAt}`}>
            基线漂移 <b className={`font-mono tabular-nums ${drift.durationDrift > 0 ? "text-err" : "text-emerald-500"}`}>{drift.durationDrift > 0 ? `+${drift.durationDrift}` : drift.durationDrift}</b> 天
          </span>
        )}
        {dl && !dl.feasible && (
          <span>目标竣工超 <b className="font-mono tabular-nums text-err">{dl.overrun}</b> 天</span>
        )}
        <span className="text-fg-faint" title={syncError ?? undefined}>{syncText}</span>
        <span className="ml-auto flex items-center gap-1.5">
          <input
            className="h-6 w-44 rounded-md border border-border-soft bg-bg px-2 text-[11px] text-fg outline-none focus:border-accent/50"
            placeholder="问这个计划…（发回对话）"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); ask(); } }}
            data-testid="sched-tool-ask-input"
            disabled={!onAsk}
          />
          <Button size="small" type="primary" disabled={!onAsk || !question.trim()} onClick={ask} data-testid="sched-tool-ask-send">
            问计划
          </Button>
        </span>
      </div>
    </div>
  );
}
