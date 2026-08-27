// mock/office.ts — 办公/文件/任务域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
// 检索/索引/测评（SemanticSearch/UnifiedSearch/RetrievalEvalRun/FileIndexRebuild/
// FileSemanticSearch）因行数约束拆分至 retrieval.ts，见该文件头注释。
import type { AppBindings } from "../bridge";
import type { FilePickResult } from "../types";
import {
  delay,
  emit,
  MOCK_DOCX_DATA_URL,
  MOCK_XLSX_BODY,
  mockTaskListeners,
  pinnedMock,
  setPinnedMock,
  taskMock,
} from "./shared";
import type { MakeMockState } from "./state";

type OfficeMethods = Pick<
  AppBindings,
  | "ListDir" | "FileSearch" | "Materials" | "WorkspaceSearch"
  | "PinnedMaterials" | "PinMaterial" | "UnpinMaterial" | "SummarizeFile"
  | "TaskTemplates"
  | "ReadFile" | "Preview" | "OpenWorkspacePath"
  | "OfficeEditText" | "DocxApplyEdit" | "DocxAcceptChanges"
  | "XlsxEdit" | "XlsxSetCell" | "XlsxRecalc" | "XlsxRowOps" | "XlsxColOps"
  | "XlsxChart" | "ZipDeliverables" | "SubagentRuns" | "WriteFile"
  | "ExportDeliverable" | "CrossEmbed" | "RevealWorkspacePath"
  | "SavePastedImage" | "SaveAttachmentFile" | "AttachmentDataURL"
  | "CaptureScreen" | "RecognizeImage" | "OCRText"
  | "HerdsmanDigitalLife" | "HerdsmanOperations"
  | "PickFiles" | "PickDirectory"
  | "TaskList" | "TaskCancel" | "TaskRetry" | "TaskOutput"
>;

export function buildOffice(_s: MakeMockState): OfficeMethods {
  return {
    async ListDir(rel: string) {
      // A tiny fake tree so the @ menu is navigable in browser dev.
      if (rel === "" || rel === "./") {
        return [
          { name: "internal", isDir: true },
          { name: "desktop", isDir: true },
          { name: "README.md", isDir: false },
          { name: "go.mod", isDir: false },
        ];
      }
      if (rel === "internal/") {
        return [
          { name: "control", isDir: true },
          { name: "boot", isDir: true },
          { name: "event.go", isDir: false },
        ];
      }
      return [{ name: "file.go", isDir: false }];
    },
    async FileSearch(query: string, limit = 30) {
      const tree = [
        { path: "README.md", name: "README.md", isDir: false, size: 18 },
        { path: "desktop/file.go", name: "file.go", isDir: false, size: 42 },
        { path: "docs/成本测算.xlsx", name: "成本测算.xlsx", isDir: false, size: 120 },
        { path: "docs/方案.docx", name: "方案.docx", isDir: false, size: 80 },
        { path: "internal/control", name: "control", isDir: true },
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
        "README.md": "# gaea\n\nBrowser-dev workspace preview.\n\n- Chat in the center\n- Browse files on the right\n- Keep sessions on the left\n",
        "go.mod": "module gaea\n\ngo 1.23\n",
        "desktop/file.go": "package desktop\n\nfunc main() {\n\tprintln(\"workspace preview\")\n}\n",
        "internal/event.go": "package internal\n\n// mock file used by the browser dev seam\n",
      };
      return {
        path: rel,
        body: samples[rel] ?? `// ${rel}\n\nMock file body from browser dev.`,
        size: samples[rel]?.length ?? 42,
        truncated: false,
        binary: false,
      };
    },
    async Preview(rel: string) {
      const samples: Record<string, string> = {
        "README.md": "# gaea\n\nBrowser-dev workspace preview.\n\n- Chat in the center\n- Browse files on the right\n- Keep sessions on the left\n",
        "go.mod": "module gaea\n\ngo 1.23\n",
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
        // 最小 docx（mock），由 docx-preview 渲染成版式预览。
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".docx",
          size: 1728, kind: "docx" as const,
          body: "", dataUrl: MOCK_DOCX_DATA_URL, error: "",
        };
      }
      if (ext === "xlsx") {
        // 结构化单元格预览（mock），由 XlsxPreview 渲染。
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".xlsx",
          size: 2048, kind: "xlsx" as const,
          body: MOCK_XLSX_BODY, dataUrl: "", error: "",
        };
      }
      if (ext === "md") {
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".md",
          size: samples[rel]?.length ?? 0, kind: "markdown" as const,
          body: samples[rel] ?? "# Mock\n\n预览内容来自浏览器 mock。", dataUrl: "", error: "",
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
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: `.${ext}`,
        size: samples[rel]?.length ?? 0, kind: "text" as const,
        body: samples[rel] ?? `// ${rel}\n\nMock file body from browser dev.`, dataUrl: "", error: "",
      };
    },
    async OpenWorkspacePath(rel: string) {
      console.info("mock OpenWorkspacePath", rel);
    },
    async OfficeEditText(selectedText: string, instruction: string) {
      return { edited: `${selectedText}（mock 编辑：${instruction}）` };
    },
    async DocxApplyEdit(rel: string) {
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".docx",
        size: 1728, kind: "docx" as const,
        body: "", dataUrl: MOCK_DOCX_DATA_URL, error: "",
      };
    },
    async DocxAcceptChanges(rel: string, _accept: boolean) {
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".docx",
        size: 1728, kind: "docx" as const,
        body: "", dataUrl: MOCK_DOCX_DATA_URL, error: "",
      };
    },
    async XlsxEdit(_rel: string, sheet: string, instruction: string, selection: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: `（mock）已在 ${sheet} 应用操作：${instruction}（选区 ${selection}）`,
        applied: 1,
      };
    },
    async XlsxSetCell(_rel: string, sheet: string, ref: string, value: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: `（mock）已更新 ${sheet}!${ref} = ${value}`,
        applied: 1,
      };
    },
    async XlsxRecalc(_rel: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: "（mock）已重算公式",
        applied: 1,
      };
    },
    async XlsxRowOps(_rel: string, sheet: string, action: string, ref: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: `（mock）已在 ${sheet} 执行行操作 ${action}@${ref}`,
        applied: 1,
      };
    },
    async XlsxColOps(_rel: string, sheet: string, action: string, ref: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: `（mock）已在 ${sheet} 执行列操作 ${action}@${ref}`,
        applied: 1,
      };
    },
    async XlsxChart(input: { rel: string; chartType?: string; refs?: string }) {
      const name = `${input.rel.split("/").pop() ?? "sheet"}-chart-mock.png`;
      return {
        path: `.gaea/exports/${name}`,
        name,
        dataUrl:
          "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
        labels: input.refs ? 3 : 2,
        chartType: input.chartType ?? "bar",
      };
    },
    async ZipDeliverables(paths: string[]) {
      return {
        path: ".gaea/exports/gaea-会话产物-mock.zip",
        name: "gaea-会话产物-mock.zip",
        entries: paths.length,
        bytes: paths.length * 128,
      };
    },
    async SubagentRuns(_sessionPath: string) {
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
    },
    async ExportDeliverable(input: { markdown: string; format: string; title?: string }) {
      const format = input.format.replace(".", "");
      return {
        path: `.gaea/exports/${input.title || "deliverable"}-mock.${format}`,
        name: `${input.title || "deliverable"}-mock.${format}`,
        format,
        size: input.markdown.length,
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
      return "data:image/png;base64,iVBORw0KGgo=";
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
    async PickFiles(): Promise<FilePickResult[]> {
      // In dev mode there is no native dialog -- return empty.
      return [];
    },
    async PickDirectory(): Promise<string> {
      // mock: no native dialog
      return "";
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
    async WriteFile(_rel: string, _content: string) {
      // mock：浏览器开发环境不落盘（真实实现 = GaeaWriteFile 原子写回工作区）。
    },
  };
}
