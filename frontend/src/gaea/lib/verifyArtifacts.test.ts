import { describe, expect, it } from "vitest";
import {
  buildThumbSections,
  classifyDirProbe,
  imagePageNo,
  mapPool,
  orderGroupDirs,
  pairPages,
  splitArtifactEntries,
  verifyArtifactRelPath,
} from "./verifyArtifacts";
import type { VerifyDirEntry, VerifyPageFile } from "./verifyArtifacts";

// 文件名样例全部取自真实链路（internal/app/gaea_verify.go runVisualDiff +
// internal/office/docmd/render.go RenderPDFPages）：pdftoppm 输出
// <prefix>-<N>.png，prefix 取 before/after；≥10 页时可能补零。
const pf = (page: number, name: string, relPath: string): VerifyPageFile => ({ page, name, relPath });

describe("verifyArtifactRelPath 产物目录绝对路径 → 工作区相对路径", () => {
  it("Windows 绝对路径（Go filepath.ToSlash 形态）按固定标记相对化", () => {
    expect(verifyArtifactRelPath("C:/ws/项目/.gaea/work/journal/verify/ev_1003")).toBe(
      ".gaea/work/journal/verify/ev_1003",
    );
  });

  it("Unix 绝对路径同样按标记相对化（与工作区根无关）", () => {
    expect(verifyArtifactRelPath("/home/u/gaea/.gaea/work/journal/verify/s-a1-1720000000")).toBe(
      ".gaea/work/journal/verify/s-a1-1720000000",
    );
  });

  it("尾部斜杠归一去掉；反斜杠（未 ToSlash 的旧数据）归一为 /", () => {
    expect(verifyArtifactRelPath("D:/data/ws/.gaea/work/journal/verify/x-2-123/")).toBe(
      ".gaea/work/journal/verify/x-2-123",
    );
    expect(verifyArtifactRelPath("D:\\data\\ws\\.gaea\\work\\journal\\verify\\x-2-123")).toBe(
      ".gaea/work/journal/verify/x-2-123",
    );
  });

  it("已是相对路径（mock/旧 verdict）原样返回", () => {
    expect(verifyArtifactRelPath(".gaea/work/journal/verify/ev_1003")).toBe(
      ".gaea/work/journal/verify/ev_1003",
    );
  });

  it("绝对路径但不含标记（不在工作区）→ null（诚实降级，不臆测）", () => {
    expect(verifyArtifactRelPath("C:/elsewhere/verify/abc")).toBeNull();
    expect(verifyArtifactRelPath("/opt/other/.gaea/work/journal/x")).toBeNull();
  });

  it("空串 → null", () => {
    expect(verifyArtifactRelPath("")).toBeNull();
    expect(verifyArtifactRelPath("   ")).toBeNull();
  });
});

describe("imagePageNo 页码解析（对齐 render.go 尾数字段规则）", () => {
  it("真实命名规则：before-1 / after-10 / 补零 before-03", () => {
    expect(imagePageNo("before-1.png")).toBe(1);
    expect(imagePageNo("after-10.png")).toBe(10);
    expect(imagePageNo("before-03.png")).toBe(3);
    expect(imagePageNo("AFTER-2.PNG")).toBe(2); // 大小写不敏感
  });

  it("非页面图 / 解析不出页码 → null（过滤）", () => {
    expect(imagePageNo("before.pdf")).toBeNull(); // 审计 PDF
    expect(imagePageNo("after.jsonl")).toBeNull();
    expect(imagePageNo("cover.png")).toBeNull(); // 无 -N 段
    expect(imagePageNo("before-.png")).toBeNull(); // 空数字段
    expect(imagePageNo("第2页.png")).toBeNull(); // 非尾数字段
    expect(imagePageNo("before-0.png")).toBeNull(); // 页码从 1 起
    expect(imagePageNo("before-x.png")).toBeNull();
  });
});

describe("splitArtifactEntries 目录条目 → 子目录 + 已排序页面图", () => {
  it("真实布局一层条目：过滤 PDF，页面按页码升序，relPath 用 / 拼接", () => {
    const entries: VerifyDirEntry[] = [
      { name: "before.pdf", isDir: false, size: 1024 },
      { name: "after.pdf", isDir: false, size: 1024 },
      { name: "before", isDir: true, size: 0 },
      { name: "after", isDir: true, size: 0 },
    ];
    expect(splitArtifactEntries(entries, ".gaea/work/journal/verify/ev_1")).toEqual({
      dirs: ["before", "after"],
      pages: [],
    });
  });

  it("before/ 子目录一层：乱序输入 → 页码升序 + relPath 带子目录前缀", () => {
    const entries: VerifyDirEntry[] = [
      { name: "before-2.png", isDir: false, size: 1 },
      { name: "before-10.png", isDir: false, size: 1 },
      { name: "before-1.png", isDir: false, size: 1 },
      { name: "notes.txt", isDir: false, size: 1 },
    ];
    const out = splitArtifactEntries(entries, ".gaea/work/journal/verify/ev_1/before");
    expect(out.dirs).toEqual([]);
    expect(out.pages.map((p) => p.page)).toEqual([1, 2, 10]); // 数值序，非字典序
    expect(out.pages[0].relPath).toBe(".gaea/work/journal/verify/ev_1/before/before-1.png");
  });

  it("undefined / 空数组 / 脏条目防御", () => {
    expect(splitArtifactEntries(undefined, "x")).toEqual({ dirs: [], pages: [] });
    expect(
      splitArtifactEntries(
        [{ name: "", isDir: true }, { name: "ok-1.png", isDir: false }] as VerifyDirEntry[],
        "x/",
      ),
    ).toEqual({
      dirs: [],
      pages: [pf(1, "ok-1.png", "x/ok-1.png")], // parentRel 尾斜杠归一
    });
  });
});

describe("orderGroupDirs 分组排序：before → after → 其余按名", () => {
  it("真实布局保持改前在先", () => {
    expect(orderGroupDirs(["after", "before"])).toEqual(["before", "after"]);
  });
  it("未知目录排后并按名稳定序；大小写归一识别 before/after", () => {
    expect(orderGroupDirs(["extra", "Before", "After", "annot"])).toEqual([
      "Before",
      "After",
      "annot",
      "extra",
    ]);
  });
});

describe("pairPages before/after 按页码并集配对（升序，缺页诚实留空）", () => {
  it("齐备 / 单侧缺页 / 页数变化（after 多一页）", () => {
    const before = [pf(1, "before-1.png", "b/1"), pf(2, "before-2.png", "b/2")];
    const after = [pf(1, "after-1.png", "a/1"), pf(2, "after-2.png", "a/2"), pf(3, "after-3.png", "a/3")];
    expect(pairPages(before, after)).toEqual([
      { page: 1, before: before[0], after: after[0] },
      { page: 2, before: before[1], after: after[1] },
      { page: 3, after: after[2] }, // 新增页：before 缺
    ]);
  });

  it("只有一侧时整列缺另一侧", () => {
    expect(pairPages([], [pf(1, "after-1.png", "a/1")])).toEqual([{ page: 1, after: pf(1, "after-1.png", "a/1") }]);
  });
});

describe("buildThumbSections 渲染区组装", () => {
  it("before/after 折成 pair 区（成对行 + side 标注），其余分组独立成区", () => {
    const sections = buildThumbSections([
      { key: "before", pages: [pf(1, "before-1.png", "b/before-1.png")] },
      { key: "after", pages: [pf(1, "after-1.png", "a/after-1.png"), pf(2, "after-2.png", "a/after-2.png")] },
      { key: "flat", pages: [pf(1, "page-1.png", "r/page-1.png")] },
    ]);
    expect(sections.map((s) => s.key)).toEqual(["pair", "flat"]);
    expect(sections[0].rows).toHaveLength(2);
    expect(sections[0].rows[0].cells.map((c) => c.side)).toEqual(["before", "after"]);
    expect(sections[0].rows[1].cells.map((c) => c.side)).toEqual(["after"]); // 第 2 页单侧
    expect(sections[1].rows[0].cells[0].side).toBe("single");
  });

  it("空组不成区；全空 → 空数组", () => {
    expect(buildThumbSections([{ key: "before", pages: [] }])).toEqual([]);
  });
});

describe("classifyDirProbe 目录探测结果 → 降级原因（对齐 gaea_preview.go 语义）", () => {
  it("目录不存在 → missing；目录在（无法预览）而列为空 → empty", () => {
    expect(classifyDirProbe("error", "文件不存在")).toBe("missing");
    expect(classifyDirProbe("error", "目录无法预览")).toBe("empty");
  });
  it("非 error 负载（mock/异常）→ noPages", () => {
    expect(classifyDirProbe("text", "")).toBe("noPages");
  });
});

describe("mapPool 定并发映射", () => {
  it("保持输入顺序返回，且同时在飞不超过 limit", async () => {
    let inflight = 0;
    let peak = 0;
    const out = await mapPool([1, 2, 3, 4, 5, 6, 7], 3, async (n) => {
      inflight++;
      peak = Math.max(peak, inflight);
      await new Promise((r) => setTimeout(r, 1));
      inflight--;
      return n * 10;
    });
    expect(peak).toBeLessThanOrEqual(3);
    expect(out).toEqual([10, 20, 30, 40, 50, 60, 70]);
  });

  it("空输入与单元素", async () => {
    expect(await mapPool([], 4, async (n) => n)).toEqual([]);
    expect(await mapPool(["a"], 4, async (n) => n + "!")).toEqual(["a!"]);
  });
});
