// mock/office/methods_xlsx.ts — 办公文档真编辑域（P4 结构刀2 拆分自 mock/office.ts）。
// 图文板块（2026-11-15 刀0b）：pptx/xlsx 编辑面已撤（预览链收缩），仅保留
// docx 编辑三件（DocxPreview 消费）。方法与返回形状与拆分前一致。
import { MOCK_DOCX_DATA_URL } from "../shared";
import type { OfficeMethods } from "./types";

export const xlsxMethods = {
  async OfficeEditText(selectedText: string, instruction: string) {
    return { edited: `${selectedText}（mock 编辑：${instruction}）` };
  },
  // 形参对齐 Go 门面签名（X1-11 契约测试）：mock 固定返回样例文档，编辑实参只收不用。
  async DocxApplyEdit(rel: string, _selectedText: string, _replacement: string) {
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
} satisfies Partial<OfficeMethods>;
