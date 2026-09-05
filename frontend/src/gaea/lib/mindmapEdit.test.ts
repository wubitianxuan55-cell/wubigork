// mindmapEdit.test.ts — M2 导图画布编辑纯函数回归（树操作守卫 + 规范序列化幂等）
import { describe, expect, it } from "vitest";
import { parseMindmapOutline, type MindNode } from "./mindmap";
import {
  findMindNode,
  mindAddChild,
  mindAddSibling,
  mindMaxId,
  mindMove,
  mindRemove,
  mindRename,
  serializeMindmapOutline,
} from "./mindmapEdit";

const outline = `# 项目
## 设计
- 原型
  - 高保真
## 开发
- 后端
`;

const root = () => parseMindmapOutline(outline, "项目").root;

describe("mindmapEdit 树操作", () => {
  it("mindAddChild：追加子节点，id 递增不冲突", () => {
    const r = root();
    const design = findMindNode(r, findIdByText(r, "设计"))!;
    const out = mindAddChild(r, design.id, "评审稿")!;
    const newDesign = findMindNode(out, design.id)!;
    expect(newDesign.children.map((c) => c.text)).toEqual(["原型", "评审稿"]); // 高保真是原型的孩子
    expect(out).not.toBe(r); // 不可变
  });

  it("mindAddSibling：插到目标之后；根不可加兄弟", () => {
    const r = root();
    const proto = findMindNode(r, findIdByText(r, "原型"))!;
    const out = mindAddSibling(r, proto.id, "线框")!;
    const design = findMindNode(out, findIdByText(r, "设计"))!;
    expect(design.children.map((c) => c.text)).toEqual(["原型", "线框"]);
    expect(mindAddSibling(r, r.id, "x")).toBeNull();
  });

  it("mindRename：改名生效；空文本拒绝", () => {
    const r = root();
    const id = findIdByText(r, "原型");
    const out = mindRename(r, id, "低保真原型")!;
    expect(findMindNode(out, id)!.text).toBe("低保真原型");
    expect(mindRename(r, id, "   ")).toBeNull();
  });

  it("mindRemove：删子树；根不可删", () => {
    const r = root();
    const designId = findIdByText(r, "设计");
    const out = mindRemove(r, designId)!;
    expect(findMindNode(out, designId)).toBeNull();
    expect(out.children.map((c) => c.text)).toEqual(["开发"]);
    expect(mindRemove(r, r.id)).toBeNull();
  });

  it("mindMove：挂到目标最后；守卫=不动根/不挂自身/目标不可在拖动子树内", () => {
    const r = root();
    const protoId = findIdByText(r, "原型");
    const devId = findIdByText(r, "开发");
    const out = mindMove(r, protoId, devId)!;
    const dev = findMindNode(out, devId)!;
    expect(dev.children.map((c) => c.text)).toEqual(["后端", "原型"]);
    expect(mindMove(r, r.id, devId)).toBeNull(); // 根不可拖
    expect(mindMove(r, protoId, protoId)).toBeNull(); // 挂自身
    const hiFiId = findIdByText(r, "高保真");
    expect(mindMove(r, protoId, hiFiId)).toBeNull(); // 目标在拖动子树内 → 成环
  });

  it("mindMaxId 取全树最大编号", () => {
    expect(mindMaxId(root())).toBeGreaterThanOrEqual(5);
  });
});

describe("serializeMindmapOutline（规范大纲）", () => {
  it("root→H1、depth1→H2、depth≥2→嵌套列表", () => {
    const text = serializeMindmapOutline(root());
    expect(text.split(/\r?\n/)).toEqual([
      "# 项目",
      "## 设计",
      "- 原型",
      "  - 高保真",
      "## 开发",
      "- 后端",
      "",
    ]);
  });

  it("parse(serialize(tree)) 结构幂等（深度/文本/形状一致）", () => {
    const r = root();
    const once = parseMindmapOutline(serializeMindmapOutline(r), "x").root;
    const shape = (n: MindNode): object => ({
      text: n.text,
      depth: n.depth,
      children: n.children.map(shape),
    });
    expect(JSON.stringify(shape(once))).toBe(JSON.stringify(shape(r)));
    const twice = parseMindmapOutline(serializeMindmapOutline(once), "x").root;
    expect(JSON.stringify(shape(twice))).toBe(JSON.stringify(shape(r)));
  });
});

function findIdByText(root: ReturnType<typeof parseMindmapOutline>["root"], text: string): string {
  const found = searchByText(root, text);
  if (!found) throw new Error(`node not found: ${text}`);
  return found.id;
}

function searchByText(n: ReturnType<typeof parseMindmapOutline>["root"], text: string): { id: string } | null {
  if (n.text === text) return n;
  for (const c of n.children) {
    const hit = searchByText(c, text);
    if (hit) return hit;
  }
  return null;
}
