import { describe, expect, it } from "vitest";
import { remarkFileLinks } from "./remarkFileLinks";

interface MdNode {
  type: string;
  value?: string;
  url?: string;
  children?: MdNode[];
}

function run(tree: MdNode): MdNode {
  remarkFileLinks()(tree);
  return tree;
}

function text(value: string): MdNode {
  return { type: "text", value };
}

describe("remarkFileLinks（mdast 层文件链接识别）", () => {
  it("把文本节点中的文件引用包成 link 节点", () => {
    const tree = run({
      type: "root",
      children: [
        {
          type: "paragraph",
          children: [text("已生成方案：exports/方案.docx，输出文件：成本测算.xlsx")],
        },
      ],
    });
    const para = tree.children?.[0] as MdNode;
    expect(para.children?.filter((n) => n.type === "link").map((n) => n.url)).toEqual([
      "exports/方案.docx",
      "成本测算.xlsx",
    ]);
  });

  it("不进入代码块 / 行内代码 / 已有链接", () => {
    const tree = run({
      type: "root",
      children: [
        { type: "code", value: "exports/内部.xlsx" },
        {
          type: "paragraph",
          children: [
            text("运行 "),
            { type: "inlineCode", value: "C:\\AI\\fix.bat" },
            text(" 即可，见 "),
            {
              type: "link",
              url: "reports/汇总.md",
              children: [text("报告")],
            },
          ],
        },
      ],
    });
    const links: string[] = [];
    const walk = (n: MdNode) => {
      if (n.type === "link") links.push(n.url ?? "");
      n.children?.forEach(walk);
    };
    walk(tree);
    expect(links).toEqual(["reports/汇总.md"]);
  });

  it("URL 与域名式路径不误判", () => {
    const tree = run({
      type: "root",
      children: [
        {
          type: "paragraph",
          children: [text("参考 https://example.com/a.pdf 与 docs.example.com/b.docx")],
        },
      ],
    });
    const para = tree.children?.[0] as MdNode;
    expect(para.children?.filter((n) => n.type === "link")).toHaveLength(0);
  });
});
