// pptxTextDiff.test.ts — pptx 结构化对比纯函数测试（刀3：页对齐/形状段落 LCS/
// 降级 unsupported 口径）。zip fixture 手工 JSZip 构造（docxOutline.test.ts 同套路）。
import { describe, expect, it } from "vitest";
import JSZip from "jszip";
import {
  diffPptxBytes,
  diffPptxSlideTexts,
  parsePptxSlides,
  pptxBytesFromDataUrl,
  type PptxSlideTexts,
} from "./pptxTextDiff";

const A_NS = 'xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"';
const P_NS = 'xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"';

// 构造单页 slide XML：每段一个 a:p，段内单 run a:t（"-" 开头的段视为空段；
// "raw:" 前缀的段原样嵌入，供跨 run 等手工构造）。
function slideXml(paras: string[]): string {
  const body = paras
    .map((t) => {
      if (t === "-") return `<a:p><a:endParaRPr lang="zh-CN"/></a:p>`;
      if (t.startsWith("raw:")) return t.slice(4);
      return `<a:p><a:r><a:t>${t}</a:t></a:r></a:p>`;
    })
    .join("");
  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><p:sld ${A_NS} ${P_NS}><p:cSld><p:spTree><p:sp><p:txBody>${body}</p:txBody></p:sp></p:spTree></p:cSld></p:sld>`;
}

// 构造 pptx 字节：entries = [文件名, 段落列表]（文件名任意序写入，验证自然序）。
async function pptxBytes(entries: Array<[string, string[]]>): Promise<Uint8Array> {
  const zip = new JSZip();
  for (const [name, paras] of entries) zip.file(name, slideXml(paras));
  return zip.generateAsync({ type: "uint8array" });
}

const slide = (page: number, texts: string[]): PptxSlideTexts => ({ page, texts });

describe("parsePptxSlides 解包与防御", () => {
  it("逐页抽 a:p 聚合 a:t：跨 run 拼接、空段保留、页码=文件名自然序", async () => {
    const bytes = await pptxBytes([
      ["ppt/slides/slide10.xml", ["第十页"]],
      ["ppt/slides/slide1.xml", ["标题", "raw:<a:p><a:r><a:t>跨</a:t></a:r><a:r><a:t>run</a:t></a:r></a:p>", "-"]],
    ]);
    // 故意再插一页 slide2，验证 slide2 排在 slide10 前（数值序而非字典序）。
    const zip = await JSZip.loadAsync(bytes);
    zip.file("ppt/slides/slide2.xml", slideXml(["第二页"]));
    const out = zip.generateAsync({ type: "uint8array" });
    const r = await parsePptxSlides(new Uint8Array(await out));
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    // 页码 = 排序后的展示序（1 起连续）；关键是 slide10 依数值序排在 slide2 之后
    //（字典序会把 "slide10" 排在 "slide2" 前，这里必须不发生）。
    expect(r.slides.map((s) => s.page)).toEqual([1, 2, 3]);
    expect(r.slides[0].texts).toEqual(["标题", "跨run", ""]);
    expect(r.slides[1].texts).toEqual(["第二页"]);
    expect(r.slides[2].texts).toEqual(["第十页"]);
  });

  it("坏 zip（非 zip 字节）→ 结构化失败不抛错", async () => {
    const r = await parsePptxSlides(new Uint8Array([1, 2, 3, 4]));
    expect(r.ok).toBe(false);
    if (r.ok) return;
    expect(r.reason).toContain("zip 解包失败");
  });

  it("无 slides 目录（非 pptx）→ 结构化失败", async () => {
    const zip = new JSZip();
    zip.file("word/document.xml", "<w/>");
    const r = await parsePptxSlides(await zip.generateAsync({ type: "uint8array" }));
    expect(r.ok).toBe(false);
    if (r.ok) return;
    expect(r.reason).toContain("无幻灯片");
  });

  it("页 XML 解析失败 → 结构化失败（宁漏勿误）", async () => {
    const zip = new JSZip();
    zip.file("ppt/slides/slide1.xml", "<p:sld><未闭合>");
    const r = await parsePptxSlides(await zip.generateAsync({ type: "uint8array" }));
    expect(r.ok).toBe(false);
    if (r.ok) return;
    expect(r.reason).toContain("slide1.xml");
  });

  it("pptxBytesFromDataUrl：取逗号后负载还原字节（兼容裸 base64）", () => {
    const bytes = pptxBytesFromDataUrl("data:application/octet-stream;base64," + btoa("abc"));
    expect(Array.from(bytes)).toEqual([97, 98, 99]);
    expect(Array.from(pptxBytesFromDataUrl(btoa("ab")))).toEqual([97, 98]);
  });
});

describe("diffPptxSlideTexts 页对齐与段落 LCS", () => {
  it("完全一致：全 equal 页 + equal 行（层2 全量口径）", () => {
    const r = diffPptxSlideTexts([slide(1, ["一", "二"])], [slide(1, ["一", "二"])]);
    expect(r.summary).toEqual({ pagesBase: 1, pagesCur: 1, added: 0, removed: 0, changed: 0 });
    expect(r.pages).toEqual([{ page: 1, state: "equal", rowCount: 2 }]);
    expect(r.rows).toEqual([
      { type: "equal", page: 1, index: 1, text: "一" },
      { type: "equal", page: 1, index: 2, text: "二" },
    ]);
  });

  it("页增：当前侧尾部新页 → add 页（整页 add 行，页码取当前侧）", () => {
    const r = diffPptxSlideTexts([slide(1, ["一"])], [slide(1, ["一"]), slide(2, ["新页"])]);
    expect(r.summary).toEqual({ pagesBase: 1, pagesCur: 2, added: 1, removed: 0, changed: 0 });
    expect(r.pages.map((p) => p.state)).toEqual(["equal", "add"]);
    expect(r.rows.filter((x) => x.type === "add")).toEqual([
      { type: "add", page: 2, index: 1, text: "新页" },
    ]);
  });

  it("页删：基线侧中间页移除 → del 页（页码取基线侧）", () => {
    const r = diffPptxSlideTexts(
      [slide(1, ["一"]), slide(2, ["删页"]), slide(3, ["三"])],
      [slide(1, ["一"]), slide(2, ["三"])],
    );
    expect(r.summary).toEqual({ pagesBase: 3, pagesCur: 2, added: 0, removed: 1, changed: 0 });
    expect(r.pages.map((p) => p.state)).toEqual(["equal", "del", "equal"]);
    expect(r.rows.filter((x) => x.type === "del")).toEqual([
      { type: "del", page: 2, index: 1, text: "删页" },
    ]);
  });

  it("页内段落改：changed 页内相邻 del+add 对（diffDocxParagraphs 复用）", () => {
    const r = diffPptxSlideTexts(
      [slide(1, ["标题", "旧要点"])],
      [slide(1, ["标题", "新要点"])],
    );
    expect(r.summary).toEqual({ pagesBase: 1, pagesCur: 1, added: 0, removed: 0, changed: 1 });
    expect(r.pages).toEqual([{ page: 1, state: "changed", rowCount: 3 }]);
    expect(r.rows).toEqual([
      { type: "equal", page: 1, index: 1, text: "标题" },
      { type: "del", page: 1, index: 2, text: "旧要点" },
      { type: "add", page: 1, index: 2, text: "新要点" },
    ]);
  });

  it("页序错位：无公共锚时双侧余页按位置配对成 changed（不误报整本重写）", () => {
    const r = diffPptxSlideTexts(
      [slide(1, ["甲"]), slide(2, ["乙"])],
      [slide(1, ["丙"]), slide(2, ["丁"])],
    );
    expect(r.summary).toEqual({ pagesBase: 2, pagesCur: 2, added: 0, removed: 0, changed: 2 });
    expect(r.pages.map((p) => [p.page, p.state])).toEqual([
      [1, "changed"],
      [2, "changed"],
    ]);
    // 每个配对页内：整段替换 = del+add 对
    expect(r.rows.filter((x) => x.type !== "equal")).toEqual([
      { type: "del", page: 1, index: 1, text: "甲" },
      { type: "add", page: 1, index: 1, text: "丙" },
      { type: "del", page: 2, index: 1, text: "乙" },
      { type: "add", page: 2, index: 1, text: "丁" },
    ]);
  });

  it("混合场景：锚点间增删改并存，rows 按页连续切分（rowCount 游标口径）", () => {
    const r = diffPptxSlideTexts(
      [slide(1, ["一"]), slide(2, ["旧"]), slide(3, ["删"])],
      [slide(1, ["一"]), slide(2, ["新"]), slide(3, ["增"]), slide(4, ["尾"])],
    );
    expect(r.summary).toEqual({ pagesBase: 3, pagesCur: 4, added: 1, removed: 0, changed: 2 });
    expect(r.pages.map((p) => [p.page, p.state])).toEqual([
      [1, "equal"],
      [2, "changed"],
      [3, "changed"],
      [4, "add"],
    ]);
    expect(r.rows.filter((x) => x.type !== "equal")).toEqual([
      { type: "del", page: 2, index: 1, text: "旧" },
      { type: "add", page: 2, index: 1, text: "新" },
      { type: "del", page: 3, index: 1, text: "删" },
      { type: "add", page: 3, index: 1, text: "增" },
      { type: "add", page: 4, index: 1, text: "尾" },
    ]);
  });

  it("空侧 ↔ 有内容：整本 add/del（空页也是页，计数不吞）", () => {
    const add = diffPptxSlideTexts([], [slide(1, []), slide(2, ["一"])]);
    expect(add.summary).toEqual({ pagesBase: 0, pagesCur: 2, added: 2, removed: 0, changed: 0 });
    expect(add.pages.map((p) => p.state)).toEqual(["add", "add"]);
    const del = diffPptxSlideTexts([slide(1, ["一"])], []);
    expect(del.summary.removed).toBe(1);
    expect(del.rows).toEqual([{ type: "del", page: 1, index: 1, text: "一" }]);
  });
});

describe("diffPptxBytes 端到端（字节进出）", () => {
  it("两侧字节 → 页对齐 + 段级 diff", async () => {
    const base = await pptxBytes([
      ["ppt/slides/slide1.xml", ["标题一", "要点"]],
      ["ppt/slides/slide2.xml", ["第二页"]],
    ]);
    const cur = await pptxBytes([
      ["ppt/slides/slide1.xml", ["标题一", "要点改"]],
      ["ppt/slides/slide2.xml", ["第二页"]],
      ["ppt/slides/slide3.xml", ["新增页"]],
    ]);
    const r = await diffPptxBytes(base, cur);
    expect(r.ok).toBe(true);
    if (!r.ok) return;
    expect(r.summary).toEqual({ pagesBase: 2, pagesCur: 3, added: 1, removed: 0, changed: 1 });
    expect(r.rows.some((x) => x.type === "add" && x.text === "要点改")).toBe(true);
  });

  it("任一侧不可解析 → {ok:false, reason}（宿主降级 unsupported）", async () => {
    const good = await pptxBytes([["ppt/slides/slide1.xml", ["一"]]]);
    const r = await diffPptxBytes(good, new Uint8Array([9, 9, 9]));
    expect(r.ok).toBe(false);
    if (r.ok) return;
    expect(r.reason.length).toBeGreaterThan(0);
  });
});
