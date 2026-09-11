// mock/office/methods_office.ts — office 域杂项方法（P4 结构刀2 拆分自 mock/office.ts）：
// 文件浏览/搜索/预览（ListDir/FileSearch/Materials/WorkspaceSearch/ReadFile/Preview）、
// 收藏/摘要/模板、子代理会话、交付物、附件/截图/OCR、Herdsman、原生文件对话框、
// Git、任务、证据链（GaeaJournal/Verify/Rollback）、WriteFile、数据备份（批次三a）。
// 方法与返回形状零改动；PinnedMaterials/PinMaterial/UnpinMaterial 三件套同文件
// 保证 `this.PinnedMaterials()` 的运行时绑定与拆分前逐字节一致。
import {
  delay,
  emit,
  MOCK_DOCX_DATA_URL,
  mockScenario,
  mockTaskListeners,
  pinnedMock,
  setPinnedMock,
  taskMock,
} from "../shared";
import { makeSampleProject } from "../../../../schedule/sample";
import type { DagRunView, DagTemplateView, FilePickResult } from "../../types";
import { mockFileBodies, mockXlsxState } from "./state";
import { mockScheduleFiles } from "./schedule";
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

export const officeMethods: Partial<Omit<OfficeMethods, "PinnedMaterials">> &
  Pick<OfficeMethods, "PinnedMaterials"> = {
  async ListDir(rel: string) {
    // A tiny fake tree so the @ menu is navigable in browser dev.
    if (rel === "" || rel === "./") {
      return [
        { name: "internal", isDir: true, size: 0 },
        { name: "desktop", isDir: true, size: 0 },
        { name: "README.md", isDir: false, size: 18 },
        { name: "go.mod", isDir: false, size: 42 },
      ];
    }
    if (rel === "internal/") {
      return [
        { name: "control", isDir: true, size: 0 },
        { name: "boot", isDir: true, size: 0 },
        { name: "event.go", isDir: false, size: 9 },
      ];
    }
    // v4.96 走查样例：Verifier 通道 B 逐页缩略图的产物目录（布局对齐
    // internal/app/gaea_verify.go：before/after 子目录 + <prefix>-<N>.png）。
    if (rel.replace(/\/+$/, "") === ".gaea/work/journal/verify/ev_1003") {
      return [
        { name: "before.pdf", isDir: false, size: 1024 },
        { name: "after.pdf", isDir: false, size: 1024 },
        { name: "before", isDir: true, size: 0 },
        { name: "after", isDir: true, size: 0 },
      ];
    }
    {
      const m = rel.replace(/\/+$/, "").match(/^\.gaea\/work\/journal\/verify\/ev_1003\/(before|after)$/);
      if (m) {
        const p = m[1];
        return [1, 2, 3].map((n) => ({ name: `${p}-${n}.png`, isDir: false, size: 2048 }));
      }
    }
    return [{ name: "file.go", isDir: false, size: 4 }];
  },
  async FileSearch(query: string, limit = 30) {
    const tree = [
      { path: "README.md", name: "README.md", isDir: false, size: 18, modTime: 0 },
      { path: "desktop/file.go", name: "file.go", isDir: false, size: 42, modTime: 0 },
      { path: "docs/成本测算.xlsx", name: "成本测算.xlsx", isDir: false, size: 120, modTime: 0 },
      { path: "docs/项目大纲.md", name: "项目大纲.md", isDir: false, size: 96, modTime: 0 },
      { path: "docs/方案.docx", name: "方案.docx", isDir: false, size: 80, modTime: 0 },
      { path: "internal/control", name: "control", isDir: true, size: 0, modTime: 0 },
    ];
    const q = query.toLowerCase();
    return tree.filter((f) => f.name.toLowerCase().includes(q)).slice(0, limit);
  },
  async Materials(limit = 100) {
    const now = Date.now();
    return [
      { path: "docs/成本测算.xlsx", name: "成本测算.xlsx", isDir: false, size: 120, modTime: now },
      { path: "docs/方案.docx", name: "方案.docx", isDir: false, size: 80, modTime: now - 1000 },
      { path: "docs/说明.md", name: "说明.md", isDir: false, size: 40, modTime: now - 2000 },
    ].slice(0, limit);
  },
  async WorkspaceSearch(query: string, limit = 20) {
    const q = query.toLowerCase();
    const corpus = [
      { path: "docs/成本测算.xlsx", name: "成本测算.xlsx", size: 120, body: "成本测算总金额 100 万元，材料费 60 万、人工费 40 万。" },
      { path: "docs/方案.docx", name: "方案.docx", size: 80, body: "市政道路改造方案：背景、目标、实施计划与预算。" },
      { path: "docs/说明.md", name: "说明.md", size: 40, body: "这是固定的项目说明，包含本周进展与下周计划。" },
      { path: "README.md", name: "README.md", size: 18, body: "gaea 办公助手使用说明。" },
    ];
    const now = Date.now();
    return corpus
      .filter((f) => f.name.toLowerCase().includes(q) || f.body.toLowerCase().includes(q))
      .slice(0, limit)
      .map((f, i) => ({
        path: f.path,
        name: f.name,
        size: f.size,
        modTime: now - i * 1000,
        score: 0.9 - i * 0.1,
        snippet: f.body.length > 40 ? `…${f.body.slice(0, 40)}…` : f.body,
      }));
  },
  async PinnedMaterials() {
    return pinnedMock.map((path) => ({
      path,
      name: path.split("/").pop() ?? path,
      isDir: false,
      size: 40,
      modTime: Date.now(),
    }));
  },
  async PinMaterial(rel: string) {
    if (!pinnedMock.includes(rel)) pinnedMock.push(rel);
    return this.PinnedMaterials();
  },
  async UnpinMaterial(rel: string) {
    setPinnedMock(pinnedMock.filter((p) => p !== rel));
    return this.PinnedMaterials();
  },
  async SummarizeFile(rel: string, focus?: string) {
    const name = rel.split("/").pop() ?? rel;
    return {
      path: rel,
      totalPages: 0,
      chars: 120,
      chunks: 1,
      summary: `${name} 的分块摘要（mock）：主题、要点与关键数据概览${focus ? `，侧重「${focus}」` : ""}。`,
    };
  },
  async TaskTemplates() {
    return [
      { name: "weekly-report", title: "周报", description: "结构化周报：进展 / 数据 / 问题 / 下周计划", prompt: "帮我生成一份本周工作周报：按「本周进展 / 关键数据 / 遇到的问题 / 下周计划」四部分撰写，输出 Markdown 并保存到 .gaea/exports/。" },
      { name: "meeting-minutes", title: "会议纪要", description: "纪要模板：议题 / 结论 / 行动项", prompt: "帮我整理一份会议纪要：按「议题与讨论 / 结论 / 行动项」组织，行动项包含负责人和截止时间。" },
      { name: "cost-estimate", title: "成本测算", description: "生成 xlsx 成本测算表：公式 + 图表", prompt: "帮我制作一份成本测算表（.xlsx）：\n1. 先与我对齐测算范围和科目（人工/材料/机械/管理费/税费等）；\n2. 测算前先用 cost_search 查询成本库中的历史单价作为定价依据：命中的科目直接引用并在正文注明依据的条目名称，缺失科目与用户确认或给出合理估价并说明假设；\n3. 用 xlsx 能力创建表格：科目、单位、数量、单价、金额，金额用公式计算（数量×单价），并提供汇总行；\n4. 为费用构成生成原生图表（柱状/饼图）；\n5. 测算完成后用 cost_save 把本次采用的单价沉淀为成本条目（来源标注本次项目/文件，同名覆盖），并在正文汇报新增/更新条数；\n6. 保存到 .gaea/exports/ 并在正文给出可点击的 [文件名](路径)。" },
      { name: "proposal-outline", title: "方案大纲", description: "背景 / 目标 / 方案对比 / 实施 / 预算 / 风险", prompt: "帮我撰写一份方案大纲：按「背景与目标 / 现状分析 / 方案设计 / 实施计划 / 预算 / 风险」组织。" },
      { name: "data-analysis", title: "数据分析", description: "清洗 → 透视 → 图表 → 结论", prompt: "帮我做一份数据分析：清洗数据 → 分类汇总 → 生成图表 → 输出结论。" },
      { name: "document-convert", title: "文档转换", description: "docx / xlsx / pdf 与 Markdown 互转", prompt: "帮我转换这份文档：用 format_convert 转为 Markdown 并保留标题层级与表格。" },
      { name: "report-assemble", title: "报告拼装", description: "多素材合并为完整报告", prompt: "帮我拼装一份完整报告：封面 / 目录 / 正文 / 附录，保留来源标注。" },
      { name: "ppt-deck", title: "演示文稿", description: "大纲 → PPT 成稿（.pptx）", prompt: "帮我生成一份演示文稿（.pptx）：先列 8-12 页大纲再成稿。" },
    ];
  },
  async ReadFile(rel: string) {
    const samples: Record<string, string> = {
      "README.md": "# gaea\n\nBrowser-dev workspace preview.\n\n```mermaid\nflowchart LR\n  A[输入] --> B[处理]\n  B --> C[输出]\n```\n\n- Chat in the center\n- Browse files on the right\n- Keep sessions on the left\n\n内嵌 HTML 白名单样例：<b>加粗</b>、<em>强调</em>、<sub>下标</sub>、<details><summary>点开详情</summary>折叠正文，白名单内标签原样渲染。</details>\n\n<script>alert('xss')</script><img src='x' onerror='alert(1)'> 危险标签应被消毒剥除，仅本行文字可见。\n",
      "go.mod": "module gaea\n\ngo 1.23\n",
      "desktop/file.go": "package desktop\n\nfunc main() {\n\tprintln(\"workspace preview\")\n}\n",
      "internal/event.go": "package internal\n\n// mock file used by the browser dev seam\n",
      // B1 多维表视图配置样例：与 MOCK_XLSX_BODY（docs/成本测算.xlsx 预算 sheet）配套，
      // ?mock=1 打开该 xlsx 即出现「按金额」视图 chips（金额>100 行着色）
      "docs/成本测算.gbase.json": JSON.stringify({
        version: 1,
        views: [
          {
            id: "byAmount",
            name: "按金额",
            type: "grid",
            sheet: "预算",
            sort: [{ column: "金额", dir: "desc" }],
            colorRules: [{ column: "金额", op: "gt", value: 100, color: "rgba(99,102,241,0.10)" }],
          },
          {
            // B2 看板走查样例：阶段列=泳道，拖卡片改阶段单元格
            id: "byStage",
            name: "阶段看板",
            type: "board",
            sheet: "预算",
            groupBy: "阶段",
            cardFields: ["项目", "金额"],
          },
        ],
      }),
      // M2 导图画布编辑走查样例：纯大纲（无段落/代码块）→ 编辑闸放行
      "docs/项目大纲.md": "# 项目大纲\n## 设计\n- 原型\n- 评审\n## 开发\n- 后端\n- 前端\n",
    };
    return {
      path: rel,
      markdown: samples[rel] ?? `// ${rel}\n\nMock file body from browser dev.`,
      size: samples[rel]?.length ?? 42,
    };
  },
  async Preview(rel: string) {
    const samples: Record<string, string> = {
      "README.md": "# gaea\n\nBrowser-dev workspace preview.\n\n```mermaid\nflowchart LR\n  A[输入] --> B[处理]\n  B --> C[输出]\n```\n\n- Chat in the center\n- Browse files on the right\n- Keep sessions on the left\n\n内嵌 HTML 白名单样例：<b>加粗</b>、<em>强调</em>、<sub>下标</sub>、<details><summary>点开详情</summary>折叠正文，白名单内标签原样渲染。</details>\n\n<script>alert('xss')</script><img src='x' onerror='alert(1)'> 危险标签应被消毒剥除，仅本行文字可见。\n",
      "go.mod": "module gaea\n\ngo 1.23\n",
      // M2 导图画布编辑走查样例：纯大纲（编辑闸放行）；保存后优先回读 mockFileBodies
      "docs/项目大纲.md": "# 项目大纲\n## 设计\n- 原型\n- 评审\n## 开发\n- 后端\n- 前端\n",
    };
    const ext = rel.split(".").pop()?.toLowerCase() ?? "";
    if (["png", "jpg", "jpeg", "gif", "webp", "svg"].includes(ext)) {
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: `.${ext}`,
        size: 1024, kind: "image" as const,
        body: "", dataUrl: "data:image/png;base64,iVBORw0KGgo=", error: "",
      };
    }
    if (ext === "docx") {
      // 最小 docx（mock），由 docx-preview 渲染成版式预览；
      // body 附带文本缩略图内容（与后端 GaeaPreview docx 分支一致）。
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".docx",
        size: 1728, kind: "docx" as const,
        body: "# 季度经营总结\n\n本季度经营数据平稳增长，成本结构持续优化。\n\n详见各章节明细。", dataUrl: MOCK_DOCX_DATA_URL, error: "",
      };
    }
    if (ext === "xlsx") {
      // 结构化单元格预览（mock），由 XlsxPreview 渲染。
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".xlsx",
        size: 2048, kind: "xlsx" as const,
        body: JSON.stringify(mockXlsxState), dataUrl: "", error: "",
      };
    }
    if (ext === "html" || ext === "htm") {
      // 1c 走查样例：沙箱 iframe 渲染（脚本受限+无网络，标注条可见）。
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".html",
        size: 512, kind: "html" as const,
        body: "<html><body style=\"font-family: sans-serif; padding: 16px;\"><h1>季度报告（沙箱样例）</h1><p>这段文字由沙箱 iframe 渲染：脚本受限、无网络、与宿主隔离。</p></body></html>",
        dataUrl: "", error: "",
      };
    }
    if (ext === "md") {
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".md",
        size: (mockFileBodies[rel] ?? samples[rel])?.length ?? 0, kind: "markdown" as const,
        body: mockFileBodies[rel] ?? samples[rel] ?? "# Mock\n\n预览内容来自浏览器 mock。", dataUrl: "", error: "",
      };
    }
    if (ext === "pdf") {
      if (rel === "scan.pdf") {
        // 扫描件 PDF：模拟 OCR 逐页进度事件，随后返回识别结果。
        emit({ kind: "preview_progress", path: rel, progress: { path: rel, done: 1, total: 3 } });
        await delay(80);
        emit({ kind: "preview_progress", path: rel, progress: { path: rel, done: 2, total: 3 } });
        await delay(80);
        emit({ kind: "preview_progress", path: rel, progress: { path: rel, done: 3, total: 3 } });
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".pdf",
          size: 2048, kind: "markdown" as const,
          body: "（以下内容由 OCR 识别）\n\n扫描页内容。", dataUrl: "", error: "",
        };
      }
      // 大 PDF 预览截断样例：truncated/totalPages 由后端 GaeaPreview 填充。
      const truncated = rel === "big.pdf";
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".pdf",
        size: truncated ? 2_400_000 : 1024, kind: "markdown" as const,
        body: truncated ? "第 1 页内容\n\n> ⚠️ 预览已截断：PDF 共 1200 页，仅显示前 500 页。" : "# PDF mock",
        dataUrl: "", error: "",
        truncated: truncated || undefined,
        totalPages: truncated ? 1200 : undefined,
      };
    }
    // v4.121 刀12 走查：计划文件预览=真实 mock 计划态（板块未保存过时给最小
    // 样例），FilePreview 据此出摘要卡；坏形状仍可手工构造以验证回落文本视图。
    // v4.139 #15 多工程：任意 .gsched.json 从 Map 按 rel 取内容（未落盘给样例）。
    if (rel.toLowerCase().endsWith(".gsched.json")) {
      const body = mockScheduleFiles.get(rel) ?? JSON.stringify(makeSampleProject());
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".gsched.json",
        size: body.length, kind: "text" as const,
        body, dataUrl: "", error: "",
      };
    }
    return {
      path: rel, name: rel.split("/").pop() ?? rel, ext: `.${ext}`,
      size: samples[rel]?.length ?? 0, kind: "text" as const,
      body: samples[rel] ?? `// ${rel}\n\nMock file body from browser dev.`, dataUrl: "", error: "",
    };
  },
  async OpenWorkspacePath(rel: string) {
    console.info("mock OpenWorkspacePath", rel);
  },
  async ZipDeliverables(paths: string[]) {
    return {
      path: ".gaea/exports/gaea-会话产物-mock.zip",
      name: "gaea-会话产物-mock.zip",
      entries: paths.length,
      bytes: paths.length * 128,
    };
  },
  async SubagentRuns(sessionPath: string) {
    // 左栏「子代理会话入口」按父会话归属展示：不同会话返回各自的子代理，
    // 与任务面板（只轮询当前 c.jsonl）共用同一契约，避免历史会话行串味。
    const now = Date.now();
    const ago = (ms: number) => new Date(now - ms).toISOString();
    if (sessionPath === "/mock/sessions/a.jsonl") {
      return {
        available: true,
        total: 2,
        running: 0,
        runs: [
          {
            ref: "sa_20260903_100000_0000000011_a1a1a1a1",
            status: "completed" as const,
            model: "deepseek-v4-flash",
            task: "核对季度报表财务口径与附注勾稽",
            answer: "口径一致，补充了研发费用归集说明。",
            lastText: "口径一致，补充了研发费用归集说明。",
            lastTool: "edit_file: 季度报表说明.md",
            toolCalls: 4,
            createdAt: ago(2 * 3_600_000),
            updatedAt: ago(3_500_000),
          },
          {
            ref: "sa_20260903_090000_0000000012_b3b3b3b3",
            status: "completed" as const,
            model: "deepseek-v4-pro",
            task: "起草季度经营快报的图表数据摘要",
            answer: "已生成四张核心指标的简明摘要。",
            lastText: "已生成四张核心指标的简明摘要。",
            lastTool: "web_search: 季度经营快报 2026",
            toolCalls: 2,
            createdAt: ago(5 * 3_600_000),
            updatedAt: ago(4 * 3_600_000),
          },
        ],
      };
    }
    if (sessionPath.includes("/annual/r1.jsonl")) {
      return {
        available: true,
        total: 1,
        running: 0,
        runs: [
          {
            ref: "sa_20260903_080000_0000000013_c4c4c4c4",
            status: "completed" as const,
            model: "deepseek-v4-flash",
            task: "复核年度经营数据与去年同期的口径差异",
            answer: "两处口径差异已标注，建议按统一基准重算。",
            lastText: "两处口径差异已标注，建议按统一基准重算。",
            lastTool: "xlsx_read: 年度经营数据.xlsx",
            toolCalls: 5,
            createdAt: ago(8 * 3_600_000),
            updatedAt: ago(7 * 3_600_000),
          },
        ],
      };
    }
    if (sessionPath === "/mock/sessions/d.jsonl") {
      return {
        available: true,
        total: 1,
        running: 0,
        runs: [
          {
            ref: "sa_20260902_180000_0000000014_d5d5d5d5",
            status: "failed" as const,
            model: "deepseek-v4-flash",
            task: "梳理 dsh 插件宿主与 uiConversation 的启动依赖",
            answer: "",
            lastTool: "bash: pnpm web boot",
            toolCalls: 3,
            createdAt: ago(20 * 3_600_000),
            updatedAt: ago(19 * 3_600_000),
          },
        ],
      };
    }
    // calm 场景：无 running 子代理——消除办公冷挂载「子代理任务自动置前」
    // 对文件预览链路走查的干扰（demo 场景的置前噪声，v4.123 走查发现）。
    if (mockScenario() === "calm") {
      return { available: false, total: 0, running: 0, runs: [] };
    }
    if (sessionPath === "" || sessionPath.includes("c.jsonl") || sessionPath.includes("cur.jsonl")) {
      return {
        available: true,
        total: 2,
        running: 1,
        runs: [
          {
            ref: "sa_20260817_110000_0000000002_b2b2b2b2",
            status: "running",
            task: "调研竞品表格 Agent 能力并总结可蒸馏点",
            lastText: "正在比对三家竞品的表格选中→图表链路…",
            lastTool: "web_fetch: https://example.com/table-agent",
            toolCalls: 1,
            createdAt: "2026-08-17T11:00:00+08:00",
            updatedAt: "2026-08-17T11:01:00+08:00",
          },
          {
            ref: "sa_20260817_100000_0000000001_a1a1a1a1",
            status: "completed",
            model: "deepseek-v4-flash",
            toolScope: ["web_search", "web_fetch"],
            task: "收集 2026 年办公 Agent 竞品更新信息",
            answer: "千问办公公测、WorkSwarm 蜂群智能体、QClaw V2 多 Agent。",
            lastText: "千问办公公测、WorkSwarm 蜂群智能体、QClaw V2 多 Agent。",
            lastTool: "web_search: 办公 Agent 竞品 2026",
            toolCalls: 3,
            createdAt: "2026-08-17T10:00:00+08:00",
            updatedAt: "2026-08-17T10:30:00+08:00",
          },
        ],
      };
    }
    return {
      available: false,
      total: 0,
      running: 0,
      runs: [],
    };
  },
  async SubagentTranscript(_sessionPath: string, ref: string) {
    const run = ref.endsWith("b2b2b2b2")
      ? {
          task: "调研竞品表格 Agent 能力并总结可蒸馏点",
          status: "running",
        }
      : {
          task: "收集 2026 年办公 Agent 竞品更新信息",
          status: "completed",
        };
    return {
      ref,
      task: run.task,
      messages: [
        { role: "system" as const, content: "你是子代理，专注完成派发任务。" },
        { role: "user" as const, content: run.task },
        { role: "assistant" as const, reasoning: "先检索竞品资料", content: "开始检索公开信息。" },
        { role: "tool" as const, name: "web_search", content: "千问办公公测、WorkSwarm 蜂群智能体、QClaw V2 多 Agent。" },
        { role: "assistant" as const, content: run.status === "completed" ? "调研完成，已汇总三条可蒸馏点。" : "正在比对三家竞品的表格选中→图表链路…" },
      ],
    };
  },
  async DeliverableRegistry(_sessionPath: string) {
    // 浏览器开发 mock：模拟会话权威产物登记表（写类 + 生成/导出类工具落盘）。
    return {
      available: true,
      total: 3,
      entries: [
        {
          // turn:1 = 演示会话唯一轮（浏览器走查：工具写出但正文未提及路径，
          // 验证消息尾部交付卡与权威登记表的合并渲染 + 未生成缺失态）。
          path: "docs/竞品调研报告.md",
          tool: "write_file",
          turn: 1,
          updatedAt: 1754438400,
          touches: 2,
        },
        {
          path: ".gaea/exports/表格方案-mock.xlsx",
          tool: "format_convert",
          turn: 3,
          updatedAt: 1754439000,
          touches: 1,
        },
        {
          path: "docs/架构图.svg",
          tool: "diagram_gen",
          turn: 4,
          updatedAt: 1754439600,
          touches: 1,
        },
      ],
    };
  },
  async ExportDeliverable(input: { markdown: string; format: string; title?: string }) {      const format = input.format.replace(".", "");
    return {
      path: `.gaea/exports/${input.title || "deliverable"}-mock.${format}`,
      name: `${input.title || "deliverable"}-mock.${format}`,
      format,
      size: input.markdown.length,
    };
  },
  async ConvertToPdf(rel: string) {
    const base = rel.split("/").pop()?.replace(/\.[^.]+$/, "") ?? "document";
    return {
      path: `.gaea/exports/${base}-mock.pdf`,
      name: `${base}-mock.pdf`,
      size: 4096,
      source: rel,
    };
  },
  async CrossEmbed(input: { xlsxRel: string; into: string; title?: string }) {
    const name = `${input.title || "chart"}-mock.${input.into}`;
    return {
      path: `.gaea/exports/${name}`,
      name,
      size: 4096,
      chartPath: `.gaea/exports/${input.title || "chart"}-chart-mock.png`,
    };
  },
  async RevealWorkspacePath(rel: string) {
    console.info("mock RevealWorkspacePath", rel);
  },
  async SavePastedImage(_dataUrl: string) {
    return ".gaea/attachments/mock.png";
  },
  async SaveAttachmentFile(_fileName: string, _base64Data: string) {
    return ".gaea/attachments/mock-file.bin";
  },
  async AttachmentDataURL(_path: string) {
    // 浏览器 mock：返回一张可渲染的 4×3 色块 SVG dataURL（占位缩略图）。
    return (
      "data:image/svg+xml;utf8," +
      encodeURIComponent(
        '<svg xmlns="http://www.w3.org/2000/svg" width="40" height="30"><rect width="40" height="30" fill="#134e4a"/><circle cx="14" cy="12" r="6" fill="#2dd4bf"/><rect x="24" y="16" width="12" height="8" fill="#f59e0b"/></svg>',
      )
    );
  },
  async CaptureScreen() {
    // 1x1 红色 PNG，占位截图
    return "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==";
  },
  async RecognizeImage(_imagePath: string, _prompt: string) {
    return "（开发预览）这是一张模拟识图结果：截图内容为一份通用办公任务清单。";
  },
  async OCRText(_imagePath: string) {
    return "（开发预览）模拟文字提取：项目周报 / 营收 120 万元 / 同比增长 18%。";
  },
  async HerdsmanDigitalLife() {
    return {
      available: false,
      source: "herdsman-digital-life",
      error: "浏览器开发环境无 Herdsman 数字生命库",
      character_count: 0, timeline_events: 0, state_commits: 0, world_events: 0,
      memory_events: 0, memory_summaries: 0, relationships: 0, turn_traces: 0,
      characters: [], recent_timeline: [], recent_world: [],
    };
  },
  async HerdsmanOperations() {
    return { total: 0, items: [], source: "herdsman-operations" };
  },
  async PickFiles(_filters?: string): Promise<FilePickResult[]> {
    // In dev mode there is no native dialog -- return empty.
    return [];
  },
  async ReadFileB64(_path: string): Promise<string> {
    // In dev mode there is no native dialog -- nothing to read.
    return "";
  },
  async SaveFileAs(defaultName: string, _base64Data: string): Promise<string> {
    // In dev mode there is no native dialog -- report the default name as saved.
    return defaultName;
  },
  async OpenLogsDir(): Promise<void> {
    // In dev mode there is no native file manager bridge -- no-op.
  },
  async PickDirectory(): Promise<string> {
    // mock: no native dialog
    return "";
  },
  async GaeaGitStatus() {
    // 2b 走查样例：三分组状态（暂存/未暂存/未跟踪）+ 分支 ahead。
    return {
      isRepo: true,
      branch: "main",
      ahead: 1,
      behind: 0,
      files: [
        { path: "docs/调研结论.md", x: "M", y: " ", staged: true, modified: true },
        { path: "internal/gaea/config/config.go", x: " ", y: "M", modified: true },
        { path: "web/app.ts", x: " ", y: "M", modified: true },
        { path: "reports/季度报告.html", x: "?", y: "?", untracked: true },
      ],
    };
  },
  async GaeaGitDiff(path: string, staged: boolean) {
    void staged;
    return [
      "diff --git a/" + path + " b/" + path,
      "index 111111..222222 100644",
      "--- a/" + path,
      "+++ b/" + path,
      "@@ -1,14 +1,15 @@",
      " 第一行（上下文）",
      " 上下文 2",
      " 上下文 3",
      " 上下文 4",
      " 上下文 5",
      " 上下文 6",
      " 上下文 7",
      " 上下文 8",
      " 上下文 9",
      " 上下文 10",
      " 上下文 11",
      " 上下文 12",
      "-被删除的一行",
      "+新增加的一行",
      "+第二个新增行",
    ].join("\n");
  },
  async GaeaGitStage(_paths: string[]) {},
  async GaeaGitUnstage(_paths: string[]) {},
  async GaeaGitDiscard(_path: string) {},
  async GaeaGitCommit(_message: string) {
    return "abc1234";
  },
  async GaeaGitLog(_limit: number) {
    return [
      { hash: "abc1234", subject: "release: 演示提交（mock）", author: "gaea", ts: 1750000000 },
      { hash: "def5678", subject: "fix: 上一笔演示修复（mock）", author: "gaea", ts: 1749990000 },
    ];
  },
  async TaskList() {
    return [...taskMock];
  },
  async TaskCancel(id: string) {
    const t = taskMock.find((x) => x.id === id);
    if (t && (t.status === "queued" || t.status === "running")) {
      t.status = "cancelled";
      t.finishedAt = Date.now();
      mockTaskListeners.forEach((l) => l(t));
    }
  },
  async TaskKill(id: string) {
    // mock：强杀 = 立即终态（真实实现 = 协作取消 + 进程树击杀钩子）。
    const t = taskMock.find((x) => x.id === id);
    if (t && (t.status === "queued" || t.status === "running" || t.status === "stopping")) {
      t.status = "cancelled";
      t.message = "已强制终止";
      t.finishedAt = Date.now();
      mockTaskListeners.forEach((l) => l(t));
    }
  },
  async TaskRetry(id: string) {
    const t = taskMock.find((x) => x.id === id);
    if (t && (t.status === "failed" || t.status === "cancelled")) {
      t.status = "succeeded";
      t.progress = 100;
      t.error = "";
      t.finishedAt = Date.now();
      mockTaskListeners.forEach((l) => l(t));
    }
  },
  async TaskOutput(id: string) {
    // mock：对已知任务返回样例输出尾（真实实现 = tasks 环形缓冲回放）。
    const t = taskMock.find((x) => x.id === id);
    if (!t) return { tail: "", truncated: false };
    const tail = [
      `[10:00:00] 开始 ${t.label}`,
      `[10:00:01] 处理中…（进度 ${t.progress}%）`,
      t.status === "running" ? "[10:00:02] 正在抓取四川造价信息网…" : `[10:00:03] 完成（${t.error || t.message || "ok"}）`,
    ];
    return { tail: tail.join("\n"), truncated: false };
  },
  // ── v4.8 证据链（Verifier 产品化）：证据卡 / 复核 / 回滚 —— 声明样本与
  // Preview 的 MOCK_XLSX_BODY 对齐（预算!B2=120.50、B4 公式 SUM(B2:B3)），
  // 保证「声明↔实况」diff 在浏览器开发态可演示。 ──
  async GaeaJournalList(limit: number) {
    const now = Date.now();
    const opsJson = JSON.stringify([
      { type: "set_value", sheet: "预算", target: "B2", value: 120.5 },
      { type: "set_formula", sheet: "预算", target: "B4", formula: "SUM(B2:B3)" },
      { type: "replace", sheet: "预算", range: "A1:A3", find: "设备", replace: "机械" },
    ]);
    return [
      {
        id: "ev_1003", sessionId: "mock-session", space: "work", turn: 3,
        tool: "xlsx_apply", target: "docs/成本测算.xlsx",
        beforeSummary: "成本测算表初稿（mock）", afterSummary: "已应用规划操作并重算公式（mock）",
        model: "deepseek-v4-flash", at: now - 60_000, status: "applied",
        baselinePath: ".gaea/snapshots/docs/成本测算.xlsx.snap", opsJson,
      },
      {
        id: "ev_1002", sessionId: "mock-session", space: "work", turn: 2,
        tool: "edit_file", target: "docs/说明.md",
        beforeSummary: "旧说明内容（v1，mock）", afterSummary: "新说明内容（v2，mock）",
        at: now - 120_000, status: "applied",
      },
      {
        id: "ev_1001", sessionId: "mock-session", space: "work", turn: 1,
        tool: "xlsx_apply", target: "docs/旧表.xlsx",
        beforeSummary: "旧表快照——历史卡无 opsJson，仅回放变更摘要（mock）",
        afterSummary: "已应用（mock）",
        at: now - 180_000, status: "applied",
      },
    ].slice(0, limit);
  },
  async VerifyRecord(id: string) {
    // 样本复核结论：ev_1003 通过（携带 v4.16 通道 B 结构化字段——像素差异率/
    // 渲染页数/产物目录，前端渲染「视觉复核」行 + 查看产物按钮）；ev_1001
    // 警告但保持旧形态（无 channelB 结构化字段，向后兼容样本）。
    if (id === "ev_1003") {
      return {
        id,
        status: "verified" as const,
        channelA: "结构完整",
        channelB: "视觉正常",
        note: "（mock）双通道复核通过",
        channelBRatio: 0.013,
        channelBPages: 3,
        channelBArtifacts: ".gaea/work/journal/verify/ev_1003",
        at: Date.now(),
      };
    }
    return {
      id,
      status: (id === "ev_1001" ? "warned" : "verified") as "verified" | "warned" | "failed",
      channelA: "结构完整",
      channelB: "视觉正常",
      note: "（mock）双通道复核通过",
      at: Date.now(),
    };
  },
  async RollbackRecord(_id: string) {
    // 成功路径：mock 不落盘，返回空即可（前端 toast 透出成功文案）。
  },
  async WriteFile(rel: string, content: string) {
    // mock：浏览器开发环境不落盘（真实实现 = GaeaWriteFile 原子写回工作区）。
    // 走查态：会话内记忆（导图编辑 Ctrl+S 后预览回读可见，刷新即复位）。
    mockFileBodies[rel] = content;
  },
  // ── 批次三a legacy 直调转正（Go OfficeB.GaeaDataBackup*，Gaea 前缀经 mappings）──
  async DataBackupInfo() {
    // 未做过备份：中性空态（DataPanel 初始状态；不编造 lastBackup 时间）。
    return { lastBackup: null, status: "idle" };
  },
  async DataBackupCreate(destDir: string) {
    // mock：会话内成功形状（ok:true），不真实落备份文件（path 回显目标目录）。
    return { ok: true, path: destDir };
  },
  async DataBackupRestore(_zipPath: string) {
    // mock：声明成功（ok:true），不真实写盘；回滚语义见 DataBackupRollback。
    return { ok: true };
  },
  async DataBackupCancel() {
    // mock: no-op——无进行中的备份/恢复可取消。
  },
  async DataBackupRollback() {
    // mock：从未真实恢复过 → 无可回滚（false）。
    return false;
  },
  async DataBackupRestoreResult() {
    // 从未执行恢复：status:none（DataPanel 轮询空态）。
    return { status: "none" };
  },
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