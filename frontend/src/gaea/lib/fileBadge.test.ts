import { describe, expect, it } from "vitest";
import { extBadge, extOf, fileIconName, fileTypeLabel, BADGE_EXTS } from "./fileBadge";

describe("fileBadge 扩展名展示工具（P0-2 单源）", () => {
  it("extOf 提取小写扩展名", () => {
    expect(extOf("方案.DOCX")).toBe("docx");
    expect(extOf("数据.xlsx")).toBe("xlsx");
    expect(extOf("无扩展名")).toBeNull();
  });

  it("extBadge 对常用办公扩展名返回 badge 文案", () => {
    expect(extBadge("方案.docx")).toBe("docx");
    expect(extBadge("数据.xlsx")).toBe("xlsx");
    expect(extBadge("报告.pdf")).toBe("pdf");
    expect(extBadge("a.png")).toBe("png");
    expect(BADGE_EXTS.has("docx")).toBe(true);
  });

  it("extBadge 对未知扩展名兜底为 file", () => {
    expect(extBadge("a.bin")).toBe("file");
    expect(extBadge("无扩展名")).toBe("file");
  });

  it("fileIconName 按类型返回语义图标", () => {
    expect(fileIconName("a.png")).toBe("FileImage");
    expect(fileIconName("a.xlsx")).toBe("FileSpreadsheet");
    expect(fileIconName("a.csv")).toBe("FileSpreadsheet");
    expect(fileIconName("a.pptx")).toBe("FilePpt");
    expect(fileIconName("a.docx")).toBe("FileText");
    expect(fileIconName("a.bin")).toBe("FileText");
  });

  it("fileTypeLabel 返回可读类型名", () => {
    expect(fileTypeLabel("a.docx")).toBe("Word 文档");
    expect(fileTypeLabel("a.xlsx")).toBe("Excel 表格");
    expect(fileTypeLabel("a.pdf")).toBe("PDF 文档");
    expect(fileTypeLabel("a.bin")).toBe("BIN 文件");
    expect(fileTypeLabel("无扩展名")).toBe("文件");
  });
});
