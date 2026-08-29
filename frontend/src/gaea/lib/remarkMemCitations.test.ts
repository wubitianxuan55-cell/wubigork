import { describe, expect, it } from "vitest";
import { findMemCitations } from "./memCitations";
import { remarkMemCitations } from "./remarkMemCitations";

interface MdNode {
  type: string;
  value?: string;
  url?: string;
  children?: MdNode[];
}

function run(tree: MdNode): MdNode {
  remarkMemCitations()(tree);
  return tree;
}

function text(value: string): MdNode {
  return { type: "text", value };
}

describe("findMemCitations（引用键识别）", () => {
  it("提取 MEM 引用键（大小写不敏感、name 归一小写）", () => {
    expect(findMemCitations("按 [MEM:Cost-Rule] 汇总")).toEqual([
      { start: 2, end: 17, name: "cost-rule" },
    ]);
    expect(findMemCitations("[mem:a] 与 [MEM:b]")).toHaveLength(2);
  });

  it("普通方括号文本与 kebab 之外的键不匹配", () => {
    expect(findMemCitations("普通 [链接](x) 文本")).toEqual([]);
    expect(findMemCitations("[MEM:x-y_z] 下划线不合 kebab")).toEqual([]);
    expect(findMemCitations("[MEM:] 空键")).toEqual([]);
  });
});

describe("remarkMemCitations（mdast 层记忆引用识别）", () => {
  it("把文本节点中的 [MEM:name] 包成 mem: link 节点", () => {
    const tree = run({
      type: "root",
      children: [
        {
          type: "paragraph",
          children: [text("依据 [MEM:cost-rule] 汇总，金额用公式。")],
        },
      ],
    });
    const para = tree.children?.[0] as MdNode;
    const links = para.children?.filter((n) => n.type === "link") as MdNode[];
    expect(links).toHaveLength(1);
    expect(links[0].url).toBe("mem:cost-rule");
    expect(links[0].children?.[0].value).toBe("cost-rule");
    // 前后缀文本保留
    expect(para.children?.[0].value).toBe("依据 ");
    expect(para.children?.[para.children.length - 1].value).toBe(" 汇总，金额用公式。");
  });

  it("不进入代码块 / 行内代码 / 已有链接", () => {
    const tree = run({
      type: "root",
      children: [
        { type: "code", value: "[MEM:leak]" },
        { type: "paragraph", children: [{ type: "inlineCode", value: "[MEM:leak]" }] },
        {
          type: "paragraph",
          children: [{ type: "link", url: "https://x", children: [text("[MEM:leak]")] }],
        },
        { type: "paragraph", children: [text("正常 [MEM:hit] 引用")] },
      ],
    });
    const para = tree.children?.[3] as MdNode;
    const links = para.children?.filter((n) => n.type === "link") as MdNode[];
    expect(links).toHaveLength(1);
    expect(links[0].url).toBe("mem:hit");
  });
});
