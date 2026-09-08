import { describe, expect, it, vi, beforeEach } from "vitest";
import JSZip from "jszip";
import {
  acceptanceOf,
  setAcceptance,
  statusKeyOf,
  type DeliverableStatusMap,
} from "./deliverableStatus";
import type { DiffRow } from "./diff";
import {
  buildTextDiff,
  clampDiffRows,
  compareVersionWithCurrent,
  diffStatOf,
  extOfPath,
  isTextComparable,
} from "./versionCompare";

// A2 结构化对比取数依赖：bridge（ReadFile/Preview/AttachmentDataURL）与 docx
// 段落提取。纯函数用例不触碰，这里整模块替换供 compareVersionWithCurrent 用例消费。
const appMocks = vi.hoisted(() => ({
  ReadFile: vi.fn(),
  Preview: vi.fn(),
  AttachmentDataURL: vi.fn(),
}));
vi.mock("./bridge", () => ({ app: appMocks }));

const docxMocks = vi.hoisted(() => ({
  extractDocxParagraphs: vi.fn(),
}));
vi.mock("./docxText", () => docxMocks);

beforeEach(() => {
  appMocks.ReadFile.mockReset();
  appMocks.Preview.mockReset();
  appMocks.AttachmentDataURL.mockReset();
  docxMocks.extractDocxParagraphs.mockReset();
});

describe("deliverableStatus 验收状态机", () => {
  it("缺省 open；标记后读回同状态", () => {
    let map: DeliverableStatusMap = {};
    const key = statusKeyOf("/mock/sessions/a.jsonl", "out/报告.md");
    expect(acceptanceOf(map, "/mock/sessions/a.jsonl", "out/报告.md")).toBe("open");
    map = setAcceptance(map, "/mock/sessions/a.jsonl", "out/报告.md", "confirmed", 1000, 1754438400);
    expect(map[key].status).toBe("confirmed");
    expect(acceptanceOf(map, "/mock/sessions/a.jsonl", "out/报告.md", 1754438400)).toBe("confirmed");
  });

  it("路径归一：反斜杠/大小写同键", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "/Mock/Sessions/A.jsonl", "OUT\\报告.MD", "redo", 1000, 5);
    expect(acceptanceOf(map, "/mock/sessions/a.jsonl", "out/报告.md", 5)).toBe("redo");
  });

  it("新版本落盘（updatedAt 前进）→ 重置 open；versionAt=0 不误重置", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "s", "a.md", "confirmed", 1000, 100);
    expect(acceptanceOf(map, "s", "a.md", 101)).toBe("open");
    map = setAcceptance(map, "s", "a.md", "confirmed", 1000, 0);
    expect(acceptanceOf(map, "s", "a.md", 999)).toBe("confirmed");
  });

  it("open 等价清除记录", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "s", "a.md", "redo", 1000, 1);
    map = setAcceptance(map, "s", "a.md", "open", 2000, 1);
    expect(acceptanceOf(map, "s", "a.md", 1)).toBe("open");
    expect(Object.keys(map)).toHaveLength(0);
  });

  it("不同会话同路径互不影响", () => {
    let map: DeliverableStatusMap = {};
    map = setAcceptance(map, "s1", "a.md", "confirmed", 1000, 1);
    expect(acceptanceOf(map, "s2", "a.md", 1)).toBe("open");
  });
});

describe("versionCompare 版本对比", () => {
  it("isTextComparable 按扩展名判定", () => {
    expect(isTextComparable("a.md")).toBe(true);
    expect(isTextComparable("b.XLSX")).toBe(false);
    expect(isTextComparable("docs/c.md")).toBe(true);
    expect(isTextComparable("noext")).toBe(false);
    expect(extOfPath("a.DOCX")).toBe(".docx");
  });

  it("buildTextDiff 行级差异与增删计数", () => {
    const r = buildTextDiff("a\nb\nc", "a\nc\nd");
    expect(r.add).toBe(1);
    expect(r.del).toBe(1);
    expect(r.rows.some((x) => x.type === "add" && x.text === "d")).toBe(true);
    expect(r.rows.some((x) => x.type === "del" && x.text === "b")).toBe(true);
  });

  it("diffStatOf 与 contentMissing 透传", () => {
    const rows = buildTextDiff("x", "x\ny").rows;
    expect(diffStatOf(rows)).toEqual({ add: 1, del: 0 });
    expect(buildTextDiff("", "", true).contentMissing).toBe(true);
  });

  it("clampDiffRows 未超限：原样返回且不标记截断", () => {
    const rows: DiffRow[] = [
      { type: "ctx", text: "a" },
      { type: "add", text: "b" },
    ];
    expect(clampDiffRows(rows, 200)).toEqual({ shown: rows, total: 2, truncated: false });
    expect(clampDiffRows([], 200)).toEqual({ shown: [], total: 0, truncated: false });
  });

  it("clampDiffRows 超限：只保留前 max 行并标记截断（长 diff 折叠）", () => {
    const long: DiffRow[] = Array.from({ length: 250 }, (_, i) => ({ type: "ctx", text: `L${i}` }));
    const c = clampDiffRows(long, 200);
    expect(c.truncated).toBe(true);
    expect(c.total).toBe(250);
    expect(c.shown).toHaveLength(200);
    expect(c.shown[0].text).toBe("L0");
    expect(c.shown[199].text).toBe("L199");
  });
});

// ── A2 结构化对比：compareVersionWithCurrent 分派与降级 ──────────

describe("compareVersionWithCurrent docx/xlsx 分派", () => {
  it(".docx：两侧 Preview 段落提取 → 段级 diff", async () => {
    appMocks.Preview.mockImplementation(async (rel: string) =>
      rel === "base.docx"
        ? { kind: "docx", size: 10, dataUrl: "data:x" }
        : { kind: "docx", size: 12, dataUrl: "data:y" },
    );
    docxMocks.extractDocxParagraphs.mockImplementation(async (d: string) =>
      d === "data:x" ? ["一", "旧段"] : ["一", "新段", "尾段"],
    );
    const r = await compareVersionWithCurrent("base.docx", "cur.docx");
    expect(r.kind).toBe("docx");
    if (r.kind === "docx") {
      expect(r.add).toBe(2);
      expect(r.del).toBe(1);
      expect(r.contentMissing).toBe(false);
      expect(r.rows.some((x) => x.type === "add" && x.text === "新段")).toBe(true);
    }
  });

  it(".docx：一侧 size 0 → 空内容照常 diff 并标 contentMissing（宁漏勿误）", async () => {
    appMocks.Preview.mockImplementation(async (rel: string) =>
      rel === "base.docx"
        ? { kind: "docx", size: 0, dataUrl: "" }
        : { kind: "docx", size: 9, dataUrl: "data:y" },
    );
    docxMocks.extractDocxParagraphs.mockResolvedValue(["一"]);
    const r = await compareVersionWithCurrent("base.docx", "cur.docx");
    expect(r.kind).toBe("docx");
    if (r.kind === "docx") expect(r.contentMissing).toBe(true);
  });

  it(".docx：Preview 抛错 / kind 不符 → 整体降级 unsupported（结构不可信）", async () => {
    appMocks.Preview.mockRejectedValue(new Error("read boom"));
    expect((await compareVersionWithCurrent("base.docx", "cur.docx")).kind).toBe("unsupported");

    appMocks.Preview.mockReset();
    appMocks.Preview.mockResolvedValue({ kind: "error", size: 5, error: "x" });
    expect((await compareVersionWithCurrent("base.docx", "cur.docx")).kind).toBe("unsupported");
  });

  it(".xlsx：body 结构化 JSON → sheet/单元格 diff", async () => {
    appMocks.Preview.mockImplementation(async (rel: string) => ({
      kind: "xlsx",
      size: 8,
      body: JSON.stringify({
        sheets: [
          { name: "销量", rows: [[{ ref: "A1", value: rel === "base.xlsx" ? "100" : "200" }]] },
        ],
      }),
    }));
    const r = await compareVersionWithCurrent("base.xlsx", "cur.xlsx");
    expect(r.kind).toBe("xlsx");
    if (r.kind === "xlsx") {
      expect(r.sheets).toHaveLength(1);
      expect(r.change).toBe(1);
    }
  });

  it(".xlsx：body 非 JSON → 降级 unsupported（不抛错）", async () => {
    appMocks.Preview.mockResolvedValue({ kind: "xlsx", size: 8, body: "not-json" });
    expect((await compareVersionWithCurrent("base.xlsx", "cur.xlsx")).kind).toBe("unsupported");
  });

  it("其余类型（.pdf）直接 unsupported，不发起取数", async () => {
    const r = await compareVersionWithCurrent("base.pdf", "cur.pdf");
    expect(r).toEqual({ kind: "unsupported", ext: ".pdf" });
    expect(appMocks.Preview).not.toHaveBeenCalled();
    expect(appMocks.ReadFile).not.toHaveBeenCalled();
  });

  it("文本类仍走 ReadFile 行级 diff（既有口径不回归）", async () => {
    appMocks.ReadFile.mockImplementation(async (rel: string) => ({
      markdown: rel === "base.md" ? "a\nb" : "a\nc",
      size: 3,
    }));
    const r = await compareVersionWithCurrent("base.md", "cur.md");
    expect(r.kind).toBe("text");
    if (r.kind === "text") {
      expect(r.add).toBe(1);
      expect(r.del).toBe(1);
    }
  });
});

// ── 刀3 结构化对比：compareVersionWithCurrent 的 .pptx 分派 ──────────
//
// 取数通道 = AttachmentDataURL（真实字节经 JSZip 造 pptx，解包解析走真实
// pptxTextDiff——不 mock，页对齐/段落 LCS/降级口径全链路可验）。

const A_NS = 'xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"';
const P_NS = 'xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"';

// 造一份最小 pptx 的 dataUrl：entries = [页码, 段落列表]。
async function pptxDataUrl(pages: Array<[number, string[]]>): Promise<string> {
  const zip = new JSZip();
  for (const [no, paras] of pages) {
    const body = paras.map((t) => `<a:p><a:r><a:t>${t}</a:t></a:r></a:p>`).join("");
    zip.file(
      `ppt/slides/slide${no}.xml`,
      `<?xml version="1.0"?><p:sld ${A_NS} ${P_NS}><p:cSld><p:spTree><p:txBody>${body}</p:txBody></p:spTree></p:cSld></p:sld>`,
    );
  }
  const bytes = await zip.generateAsync({ type: "uint8array" });
  let bin = "";
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
  return `data:application/octet-stream;base64,${btoa(bin)}`;
}

describe("compareVersionWithCurrent pptx 分派", () => {
  it(".pptx：两侧 AttachmentDataURL 字节 → 页对齐 + 段级 diff（真实解包）", async () => {
    const baseUrl = await pptxDataUrl([
      [1, ["标题一", "要点"]],
      [2, ["第二页"]],
    ]);
    const curUrl = await pptxDataUrl([
      [1, ["标题一", "要点改"]],
      [2, ["第二页"]],
      [3, ["新增页"]],
    ]);
    appMocks.AttachmentDataURL.mockImplementation(async (rel: string) =>
      rel === "base.pptx" ? baseUrl : curUrl,
    );
    const r = await compareVersionWithCurrent("base.pptx", "cur.pptx");
    expect(r.kind).toBe("pptx");
    if (r.kind !== "pptx") return;
    // 层1 页摘要：2 页 → 3 页，新页 1、改页 1；层2 只含有差异页
    expect(r.summary).toEqual({ pagesBase: 2, pagesCur: 3, added: 1, removed: 0, changed: 1 });
    expect(r.slides.map((s) => [s.page, s.state])).toEqual([
      [1, "changed"],
      [3, "add"],
    ]);
    // 汇总计数：改页 del+add 一对 + 新页（1 段 + 整页 1）
    expect(r.add).toBe(3);
    expect(r.del).toBe(1);
    expect(r.change).toBe(1);
    expect(r.contentMissing).toBe(false);
  });

  it(".pptx：AttachmentDataURL 抛错 / 坏字节 → 整体降级 unsupported（不抛错）", async () => {
    appMocks.AttachmentDataURL.mockRejectedValue(new Error("read boom"));
    expect((await compareVersionWithCurrent("base.pptx", "cur.pptx")).kind).toBe("unsupported");

    appMocks.AttachmentDataURL.mockResolvedValue("data:application/octet-stream;base64,////");
    expect((await compareVersionWithCurrent("base.pptx", "cur.pptx")).kind).toBe("unsupported");
  });

  it(".pptx：一侧空文件（空负载）→ 空内容照常 diff 并标 contentMissing", async () => {
    const curUrl = await pptxDataUrl([[1, ["仅存页"]]]);
    appMocks.AttachmentDataURL.mockImplementation(async (rel: string) =>
      rel === "base.pptx" ? "data:application/octet-stream;base64," : curUrl,
    );
    const r = await compareVersionWithCurrent("base.pptx", "cur.pptx");
    expect(r.kind).toBe("pptx");
    if (r.kind !== "pptx") return;
    expect(r.contentMissing).toBe(true);
    expect(r.summary.pagesBase).toBe(0);
    expect(r.add).toBe(2); // 1 段 + 整页 1
  });

  it(".ppt / .pdf 等其余类型 unsupported，不发起取数", async () => {
    expect((await compareVersionWithCurrent("base.ppt", "cur.ppt")).kind).toBe("unsupported");
    expect(appMocks.AttachmentDataURL).not.toHaveBeenCalled();
    expect(appMocks.Preview).not.toHaveBeenCalled();
    expect(appMocks.ReadFile).not.toHaveBeenCalled();
  });
});
