import { visit, SKIP } from "unist-util-visit";
import { findMemCitations } from "./memCitations";

// remarkMemCitations — Markdown AST 层的记忆引用识别（与 remarkFileLinks 同一
// 架构）：遍历 mdast 文本节点，把 [MEM:<name>] 引用键包成 mem: 链接节点，交给
// Markdown 组件的 `a` 渲染器渲染成引用徽标。
//
// 复用 link 类型（而非自造节点类型）是因为 remark-rehype 会丢弃未知 mdast
// 节点；url 走 `mem:` 自定义协议，由 Markdown 的 urlTransform 放行、a 渲染器
// 拦截。天然跳过：围栏/行内代码、已有链接、数学公式、HTML、图片。

interface MdNode {
  type: string;
  value?: string;
  url?: string;
  children?: MdNode[];
}

const SKIP_TYPES = new Set([
  "code",
  "inlineCode",
  "link",
  "image",
  "definition",
  "footnoteDefinition",
  "footnoteReference",
  "html",
  "math",
  "inlineMath",
]);

export function remarkMemCitations(): (tree: MdNode) => void {
  return (tree) => {
    visit(tree, (node: MdNode, index: number | undefined, parent: MdNode | undefined) => {
      if (SKIP_TYPES.has(node.type)) return SKIP;
      if (node.type !== "text" || !parent || index == null) return;

      const value = node.value ?? "";
      const citations = findMemCitations(value);
      if (citations.length === 0) return;

      const children: MdNode[] = [];
      let last = 0;
      for (const c of citations) {
        if (c.start > last) children.push({ type: "text", value: value.slice(last, c.start) });
        children.push({
          type: "link",
          url: `mem:${c.name}`,
          children: [{ type: "text", value: c.name }],
        });
        last = c.end;
      }
      if (last < value.length) children.push({ type: "text", value: value.slice(last) });

      parent.children?.splice(index, 1, ...children);
      // 跳过已插入节点（前缀/链接/后缀均不再包含可匹配文本），
      // 从原下一个兄弟节点继续遍历。
      return index + children.length;
    });
  };
}
