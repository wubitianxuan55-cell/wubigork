// mock/office/methods_dag.ts — 办公多文件 DAG 域（批 30 FE3-15 拆分自
// methods_office.ts，原位搬移零逻辑改动）：模块级单例 mock 态（会话内存、
// 刷新复位，语义对齐 taskMock）+ 单节点跑/推进器 + Dag* 方法 11 件。
// 组合侧 methods_office.ts 以 `...dagMethods` 原位展开（展开点=原 Dag 块
// 位置），键序与运行时绑定与拆分前一致。
import type { DagRunView, DagTemplateView } from "../../types";
import type { OfficeMethods } from "./types";

// ── 6.3 办公多文件 DAG mock 态（模块级单例，语义对齐 taskMock：会话内存、刷新复位）──
// 示例链 = 月度报告流水线：读三份月度 xlsx 报表 → 透视汇总 summary.xlsx →
// 嵌图表出报告 docx → 导 PDF 归档。推进用 setTimeout 逐节点接力（非真实子代理），
// 仅为 ?mock=1 走查 DagPanel 的状态迁移与操作闭环，不落盘。
interface DagMockState {
  runs: DagRunView[];
  timers: Set<ReturnType<typeof setTimeout>>;
  advancing: Set<string>;
}

let dagMock: DagMockState | null = null;

// 模板库 mock 态（与会话内 runs 同生命周期）：预置一条「月度经营报告」模板，
// 让 ?mock=1 首屏即可走查「模板」区的展开/重建/删除。
interface DagTplMockState {
  tpls: DagTemplateView[];
}

let dagTplMock: DagTplMockState | null = null;

function dagTplState(): DagTplMockState {
  if (!dagTplMock) {
    dagTplMock = {
      tpls: [
        {
          id: "dagtpl_mock_monthly",
          name: "月度经营报告",
          goal: "读三份月度报表，出一份带图表的月度经营报告",
          createdAt: new Date(Date.now() - 24 * 3600_000).toISOString(),
          nodes: [
            { id: "n1_read_reports", title: "读三份月度 xlsx 报表", prompt: "读取三份报表并提取核心表。", status: "pending", runCount: 0 },
            { id: "n2_pivot_summary", title: "透视汇总生成 summary.xlsx", prompt: "按月聚合营收/成本/利润。", dependsOn: ["n1_read_reports"], status: "pending", runCount: 0 },
            { id: "n3_chart_report", title: "嵌图表出报告 docx", prompt: "生成柱状/饼图并撰写报告。", dependsOn: ["n2_pivot_summary"], status: "pending", runCount: 0 },
            { id: "n4_export_pdf", title: "导出 PDF 归档", prompt: "无头转换为 PDF 归档。", dependsOn: ["n3_chart_report"], status: "pending", runCount: 0 },
          ],
        },
      ],
    };
  }
  return dagTplMock;
}

function dagState(): DagMockState {
  if (!dagMock) {
    const ago = (ms: number) => new Date(Date.now() - ms).toISOString();
    dagMock = {
      timers: new Set(),
      advancing: new Set(),
      runs: [
        {
          id: "dag_monthly_report",
          goal: "读三份月度报表，出一份带图表的月度经营报告（mock 示例链）",
          createdAt: ago(30 * 60_000),
          updatedAt: ago(30 * 60_000),
          derived: "draft",
          nodes: [
            {
              id: "n1_read_reports",
              title: "读三份月度 xlsx 报表",
              prompt: "读取 docs/报表-7月.xlsx、docs/报表-8月.xlsx、docs/报表-9月.xlsx，提取营收/成本/利润三张核心表。",
              status: "pending",
              runCount: 0,
            },
            {
              id: "n2_pivot_summary",
              title: "透视汇总生成 summary.xlsx",
              prompt: "对三份报表做透视汇总，按月聚合营收/成本/利润，写 docs/月度汇总.xlsx（附汇总公式行）。",
              dependsOn: ["n1_read_reports"],
              status: "pending",
              runCount: 0,
            },
            {
              id: "n3_chart_report",
              title: "嵌图表出报告 docx",
              prompt: "基于月度汇总.xlsx 生成柱状图/饼图，撰写 docs/月度经营报告.docx（图表内嵌）。",
              dependsOn: ["n2_pivot_summary"],
              status: "pending",
              runCount: 0,
            },
            {
              id: "n4_export_pdf",
              title: "导出 PDF 归档",
              prompt: "把 docs/月度经营报告.docx 无头转换为 .gaea/exports/月度经营报告.pdf 归档。",
              dependsOn: ["n3_chart_report"],
              status: "pending",
              runCount: 0,
            },
          ],
        },
      ],
    };
  }
  return dagMock;
}

// cloneRun 深拷贝单条 run（数组字段独立拷贝，防 UI 侧改动串入 mock 态）。
function cloneRun(r: DagRunView): DagRunView {
  return {
    ...r,
    nodes: r.nodes.map((n) => ({
      ...n,
      dependsOn: n.dependsOn ? [...n.dependsOn] : undefined,
      outputs: n.outputs ? [...n.outputs] : undefined,
    })),
  };
}

function dagSnapshot(): DagRunView[] {
  return dagState().runs.map(cloneRun);
}

// dagDerive 派生整链态（与契约 derived 语义逐条对应：draft/running/failed/ready/accepted）。
function dagDerive(run: DagRunView): DagRunView["derived"] {
  const st = new Set(run.nodes.map((n) => n.status));
  if (st.has("running")) return "running";
  if (st.has("failed")) return "failed";
  if (run.nodes.length > 0 && run.nodes.every((n) => n.status === "accepted")) return "accepted";
  if (run.nodes.length > 0 && run.nodes.every((n) => n.status === "done" || n.status === "accepted")) return "ready";
  return "draft";
}

// dagOutputsOf 节点演示产物（工作区相对路径；样例族与 Preview mock 的 docs/ 口径一致）。
function dagOutputsOf(nodeId: string): string[] {
  switch (nodeId) {
    case "n1_read_reports":
      return ["docs/报表-7月.xlsx", "docs/报表-8月.xlsx", "docs/报表-9月.xlsx"];
    case "n2_pivot_summary":
      return ["docs/月度汇总.xlsx"];
    case "n3_chart_report":
      return ["docs/月度经营报告.docx"];
    case "n4_export_pdf":
      return [".gaea/exports/月度经营报告.pdf"];
    default:
      return [];
  }
}

const DAG_STEP_MS = 600;

// dagRunNode 单节点跑/重跑：置 running，600ms 后落 done + 产物 + 子代理 ref（mock）。
function dagRunNode(runId: string, nodeId: string) {
  const st = dagState();
  const run = st.runs.find((r) => r.id === runId);
  const node = run?.nodes.find((n) => n.id === nodeId);
  if (!run || !node) return;
  node.status = "running";
  node.error = undefined;
  node.runCount += 1;
  run.derived = "running";
  run.updatedAt = new Date().toISOString();
  const tid = setTimeout(() => {
    st.timers.delete(tid);
    node.status = "done";
    node.ref = `sa_${Date.now()}_${nodeId}`;
    node.outputs = dagOutputsOf(node.id);
    // 推进器在途时保持 running 观感（下一步马上接棒），否则按真实节点态派生。
    run.derived = st.advancing.has(run.id) ? "running" : dagDerive(run);
    run.updatedAt = new Date().toISOString();
  }, DAG_STEP_MS);
  st.timers.add(tid);
}

// dagAdvance 起跑/续跑推进器：每步取一个「依赖已就绪的未完成节点」单跑，完成后
// 接力推进；无可跑节点（全部终态/失败卡链）或被 DagCancel 清定时器后自然停。
function dagAdvance(runId: string) {
  const st = dagState();
  const run = st.runs.find((r) => r.id === runId);
  if (!run) return;
  const ok = new Set(["done", "accepted"]);
  const statusOf = (id: string) => run.nodes.find((x) => x.id === id)?.status ?? "pending";
  const ready = run.nodes.find(
    (n) => !ok.has(n.status) && (n.dependsOn ?? []).every((d) => ok.has(statusOf(d))),
  );
  if (!ready) {
    st.advancing.delete(runId);
    run.derived = dagDerive(run);
    return;
  }
  st.advancing.add(runId);
  dagRunNode(run.id, ready.id);
  const tid = setTimeout(() => {
    st.timers.delete(tid);
    dagAdvance(runId);
  }, DAG_STEP_MS * 2);
  st.timers.add(tid);
}
export const dagMethods: Pick<
  OfficeMethods,
  | "DagList"
  | "DagGet"
  | "DagRun"
  | "DagNodeRun"
  | "DagNodeSteer"
  | "DagNodeAccept"
  | "DagCancel"
  | "DagTemplateList"
  | "DagTemplateSave"
  | "DagTemplateNew"
  | "DagTemplateDelete"
> = {
  // ── 6.3 办公多文件 DAG（Go OfficeB.GaeaDag*）：会话内走查态 + 示例 4 节点链。
  // 链路：读三份月度 xlsx 报表 → 透视汇总 summary.xlsx → 嵌图表出报告 docx →
  // 导 PDF。GaeaDagRun 起跑后用定时器逐节点 pending→running→done 并写 outputs，
  // GaeaDagCancel 清定时器停推进；?mock=1 打开任务视图即可完整走查交互闭环。──
  async DagList() {
    return dagSnapshot();
  },
  async DagGet(id: string) {
    const run = dagState().runs.find((r) => r.id === id);
    if (!run) throw new Error(`dag run not found: ${id}`);
    return cloneRun(run);
  },
  async DagRun(id: string) {
    const run = dagState().runs.find((r) => r.id === id);
    if (!run) throw new Error(`dag run not found: ${id}`);
    run.updatedAt = new Date().toISOString();
    dagAdvance(run.id);
    return `流水线已受理：${run.goal}（共 ${run.nodes.length} 个节点，按依赖顺序推进）`;
  },
  async DagNodeRun(id: string, nodeId: string) {
    const run = dagState().runs.find((r) => r.id === id);
    const node = run?.nodes.find((n) => n.id === nodeId);
    if (!run || !node) throw new Error(`dag node not found: ${id}/${nodeId}`);
    run.updatedAt = new Date().toISOString();
    dagRunNode(run.id, nodeId);
    return `节点「${node.title}」已受理单跑（mock 约 600ms 后完成）`;
  },
  async DagNodeSteer(id: string, nodeId: string, prompt: string) {
    const run = dagState().runs.find((r) => r.id === id);
    const node = run?.nodes.find((n) => n.id === nodeId);
    if (!run || !node) throw new Error(`dag node not found: ${id}/${nodeId}`);
    node.steerCount = (node.steerCount ?? 0) + 1;
    run.updatedAt = new Date().toISOString();
    return `已受理改向指令（第 ${node.steerCount} 次）：${prompt.slice(0, 40)}`;
  },
  async DagNodeAccept(id: string, nodeId: string) {
    const run = dagState().runs.find((r) => r.id === id);
    const node = run?.nodes.find((n) => n.id === nodeId);
    if (!run || !node) throw new Error(`dag node not found: ${id}/${nodeId}`);
    node.status = "accepted";
    node.acceptedAt = new Date().toISOString();
    run.updatedAt = node.acceptedAt;
    run.derived = dagDerive(run);
    return `节点「${node.title}」已验收，产物回流记忆（mock 不落盘）`;
  },
  async DagCancel(id: string) {
    const st = dagState();
    const run = st.runs.find((r) => r.id === id);
    if (!run) throw new Error(`dag run not found: ${id}`);
    for (const tid of st.timers) clearTimeout(tid);
    st.timers.clear();
    st.advancing.delete(id);
    let stopped = 0;
    for (const n of run.nodes) {
      if (n.status === "running") {
        n.status = "skipped";
        n.error = "已由用户终止（mock）";
        stopped++;
      }
    }
    run.derived = dagDerive(run);
    run.updatedAt = new Date().toISOString();
    return stopped > 0
      ? `已终止：${stopped} 个运行中节点置为跳过，未跑节点不再起跑`
      : "已终止：流水线无运行中节点";
  },
  // ── 6.3 余项：流水线模板库（存模板一键重建）——会话内走查态，不落盘。──
  async DagTemplateList() {
    return dagTplState().tpls.map((t) => ({ ...t, nodes: t.nodes.map((n) => ({ ...n })) }));
  },
  async DagTemplateSave(runId: string, name: string) {
    const run = dagState().runs.find((r) => r.id === runId);
    if (!run) throw new Error(`dag run not found: ${runId}`);
    const tpl: DagTemplateView = {
      id: `dagtpl_mock_${dagTplState().tpls.length + 1}`,
      name: name.trim() || run.goal.slice(0, 24),
      goal: run.goal,
      nodes: run.nodes.map((n) => ({
        id: n.id, title: n.title, prompt: n.prompt,
        dependsOn: n.dependsOn ? [...n.dependsOn] : undefined,
        status: "pending" as const, runCount: 0,
      })),
      createdAt: new Date().toISOString(),
      sourceRunId: run.id,
    };
    dagTplState().tpls.unshift(tpl);
    return `已存为模板「${tpl.name}」（${tpl.nodes.length} 个节点）（mock 不落盘）`;
  },
  async DagTemplateNew(templateId: string) {
    const st = dagTplState();
    const tpl = st.tpls.find((t) => t.id === templateId);
    if (!tpl) throw new Error(`dag template not found: ${templateId}`);
    const run: DagRunView = {
      id: `dag_rebuild_${st.tpls.length}_${Date.now() % 10_000}`,
      goal: tpl.goal,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      derived: "draft",
      nodes: tpl.nodes.map((n) => ({
        id: n.id, title: n.title, prompt: n.prompt,
        dependsOn: n.dependsOn ? [...n.dependsOn] : undefined,
        status: "pending" as const, runCount: 0,
      })),
    };
    dagState().runs.unshift(run);
    return `已从模板「${tpl.name}」重建流水线（未起跑，mock 可直接点起跑/续跑）`;
  },
  async DagTemplateDelete(templateId: string) {
    const st = dagTplState();
    const i = st.tpls.findIndex((t) => t.id === templateId);
    if (i < 0) throw new Error(`dag template not found: ${templateId}`);
    st.tpls.splice(i, 1);
    return "模板已删除（mock 不落盘）";
  },
};
