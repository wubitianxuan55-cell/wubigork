// mock/office/methods_xlsx.ts — 办公文档真编辑域（P4 结构刀2 拆分自 mock/office.ts）：
// OfficeEditText/Docx*/Pptx*/Xlsx* 方法。走查态分别来自 state.ts（pptx seed /
// mockXlsxState）与 schedule.ts 无关；方法与返回形状零改动。
import { MOCK_DOCX_DATA_URL, MOCK_XLSX_BODY } from "../shared";
import { mockPptxSeed, MOCK_PPTX_PAGE_PNG, mockXlsxState } from "./state";
import type { OfficeMethods } from "./types";

export const xlsxMethods = {
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
  // ── pptx 真编辑刀2（编辑面）走查桩：GaeaPptxSlideText（每页段落全文，
  // 两页四段演示数据）+ GaeaPptxApplyEdit（会话内存改数据——PptxSlideText
  // 回读即所见，与 XlsxSetCell 的走查态先例同构；不落盘，刷新即复位）。
  // target 未命中按后端同语义拒绝（宁拒不误改），前端透出原样错误。 ──
  async PptxSlideText(rel: string) {
    return mockPptxSeed(rel).map((s) => ({ index: s.index, paragraphs: [...s.paragraphs] }));
  },
  async PptxApplyEdit(rel: string, slideIdx: number, target: string, replacement: string) {
    const slides = mockPptxSeed(rel);
    const slide = slides.find((s) => s.index === slideIdx);
    const at = slide?.paragraphs.findIndex((p) => p.includes(target)) ?? -1;
    if (!slide || at < 0) {
      throw new Error(`未在第 ${slideIdx} 页找到目标文本（宁拒不误改）`);
    }
    slide.paragraphs[at] = slide.paragraphs[at].replace(target, replacement);
    // 返回新预览（与真机同契约：写盘后预览缓存自动失效）；mock 页缩略用
    // 占位 PNG（真实壳 = soffice→PDF→poppler 逐页缩略）。
    return {
      path: rel, name: rel.split("/").pop() ?? rel, ext: ".pptx",
      size: 4096, kind: "pdf" as const,
      body: "", dataUrl: "", error: "",
      hint: "outline",
      pages: slides.map((s) => ({ page: s.index, dataUrl: MOCK_PPTX_PAGE_PNG })),
    };
  },
  async XlsxPlanEdit(_rel: string, sheet: string, instruction: string, selection: string) {
    return {
      ops: JSON.stringify([{ type: "set_formula", sheet, target: "B5", formula: "SUM(B2:B4)" }]),
      summary: `（mock）已在 ${sheet} 规划操作：${instruction}（选区 ${selection}）`,
      changes: [
        { sheet, cell: "B5", before: "", after: "600", formula: "SUM(B2:B4)" },
        { sheet, cell: "B2", before: "100", after: "120" },
      ],
      total: 2,
      truncated: false,
    };
  },
  async XlsxApplyEdit(_rel: string, _ops: string) {
    return {
      preview: MOCK_XLSX_BODY,
      summary: "（mock）已应用规划操作并重算公式",
      applied: 1,
    };
  },
  async XlsxSetCell(_rel: string, sheet: string, ref: string, value: string) {
    // 走查态：真实写内存工作簿（看板拖卡片后泳道即时移动）
    const sh = mockXlsxState.sheets.find((x) => x.name === sheet);
    const row = sh?.rows.find((r) => r.some((c) => c.ref === ref));
    const cell = row?.find((c) => c.ref === ref);
    if (cell) cell.value = value;
    return {
      preview: JSON.stringify(mockXlsxState),
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
  async XlsxChart(input: { rel: string; chartType?: string; refs?: string; sheet?: string }) {
    // 原生图表已嵌入工作簿本身：path 即 xlsx 文件，附迷你预览数据
    const name = input.rel.split("/").pop() ?? "sheet";
    const n = input.refs ? 3 : 2;
    return {
      path: input.rel,
      name,
      sheet: input.sheet ?? "Sheet1",
      anchor: "D1",
      labels: n,
      labelList: ["一月", "二月", "三月"].slice(0, n),
      values: [120, 260, 90].slice(0, n),
      chartType: input.chartType ?? "bar",
      title: name,
    };
  },
} satisfies Partial<OfficeMethods>;