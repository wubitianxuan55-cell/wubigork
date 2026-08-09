import { visit, SKIP } from "unist-util-visit";
import { findFileMentions } from "./fileLinks";

// remarkFileLinks — Markdown AST 层的本地文件链接识别（与 openclaw / llama.cpp
// 的 rehype/remark 文件链接插件同一架构）：遍历 mdast 文本节点，把其中的
// 本地文件引用包成 link 节点，交给 Markdown 组件的 `a` 渲染器打开预览。
//
// 相比在源字符串上做正则替换 + 占位符保护，AST 方案天然跳过：
//   - 围栏代码块 / 行内代码（code / inlineCode 是独立节点）
//   - 已存在的 markdown 链接（link 节点不深入子节点）
//   - 数学公式 / HTML / 图片 alt / 链接引用定义

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

export function remarkFileLinks(): (tree: MdNode) => void {
  return (tree) => {
    visit(tree, (node: MdNode, index: number | undefined, parent: MdNode | undefined) => {
      if (SKIP_TYPES.has(node.type)) return SKIP;
      if (node.type !== "text" || !parent || index == null) return;

      const value = node.value ?? "";
      const mentions = findFileMentions(value);
      if (mentions.length === 0) return;

      const children: MdNode[] = [];
      let last = 0;
      for (const m of mentions) {
        if (m.start > last) children.push({ type: "text", value: value.slice(last, m.start) });
        children.push({
          type: "link",
          url: m.path,
          children: [{ type: "text", value: m.label }],
        });
        last = m.end;
      }
      if (last < value.length) children.push({ type: "text", value: value.slice(last) });

      parent.children.splice(index, 1, ...children);
      // 跳过已插入节点（前缀/链接/后缀均不再包含可匹配文本），
      // 从原下一个兄弟节点继续遍历。
      return index + children.length;
    });
  };
}
