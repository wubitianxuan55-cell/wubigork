import { describe, expect, it } from "vitest";
import { app } from "./bridge";

describe("框选即改 mock 契约", () => {
  it("OfficeEditText 返回替换文本", async () => {
    const r = await app.OfficeEditText("原始文本", "润色");
    expect(r.edited).toContain("原始文本");
  });

  it("DocxApplyEdit 返回 docx 预览负载", async () => {
    const r = await app.DocxApplyEdit("dir/合同.docx", "原始文本", "新文本");
    expect(r.kind).toBe("docx");
    expect(r.dataUrl).toContain("application/vnd.openxmlformats");
    expect(r.path).toBe("dir/合同.docx");
  });

  it("ExportDeliverable 返回交付结果", async () => {
    const r = await app.ExportDeliverable({
      markdown: "# 报告",
      format: "docx",
      title: "事实底座报告",
      cover: true,
    });
    expect(r.format).toBe("docx");
    expect(r.name).toContain("事实底座报告");
    expect(r.size).toBeGreaterThan(0);
  });

  it("CrossEmbed 返回图表嵌入结果", async () => {
    const r = await app.CrossEmbed({ xlsxRel: "dir/预算.xlsx", sheet: "预算", into: "docx", title: "预算构成" });
    expect(r.name).toContain("预算构成");
    expect(r.name).toContain(".docx");
    expect(r.chartPath).toContain(".png");
  });
});
