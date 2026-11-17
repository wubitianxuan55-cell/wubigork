import { describe, expect, it } from "vitest";
import type { AgentNode } from "../../lib/types";
import { FOLD_DONE_MIN, FOLD_IDLE_MIN, buildFleetModel, foldGroupId } from "./fleetModel";

function leaf(id: string, status: AgentNode["status"], children?: AgentNode[]): AgentNode {
  return { id, name: id, kind: "subagent", status, toolCalls: 1, errors: 0, tokens: 100, children };
}

function root(children: AgentNode[]): AgentNode {
  return { id: "root", name: "Lead", kind: "root", status: "completed", toolCalls: 0, errors: 0, tokens: 0, children };
}

describe("buildFleetModel 折叠聚合", () => {
  it("done 组 ≥FOLD_DONE_MIN(4) 折叠：成员隐藏、聚合卡带计数与预览", () => {
    const model = buildFleetModel(root([
      leaf("sa_1", "completed"),
      leaf("sa_2", "completed"),
      leaf("sa_3", "completed"),
      leaf("sa_4", "completed"),
    ]));
    const ids = model.map((entry) => entry.node.id);
    expect(ids).toEqual(["root", "done:root"]);
    const fold = model[1];
    expect(fold.fold).toMatchObject({ kind: "done", parentId: "root", count: 4, errorCount: 0, expanded: false });
    expect(fold.parentId).toBe("root");
    expect(fold.fold?.previews).toEqual(["sa_1", "sa_2"]);
  });

  it("小会话已结算卡不折（gaea 偏差：卡片信息密度取舍），error 计入聚合卡", () => {
    const three = buildFleetModel(root([
      leaf("sa_1", "completed"),
      leaf("sa_2", "completed"),
      leaf("sa_3", "completed"),
    ]));
    expect(three.map((entry) => entry.node.id)).toEqual(["root", "sa_1", "sa_2", "sa_3"]);
    const model = buildFleetModel(root([
      leaf("sa_1", "error"),
      leaf("sa_2", "error"),
      leaf("sa_3", "error"),
      leaf("sa_4", "completed"),
    ]));
    expect(model[1].fold).toMatchObject({ count: FOLD_DONE_MIN, errorCount: 3 });
  });

  it("idle 组按 FOLD_IDLE_MIN=3 收；queued 少于阈值保持原样", () => {
    const two = buildFleetModel(root([leaf("q1", "queued"), leaf("q2", "queued")]));
    expect(two.map((entry) => entry.node.id)).toEqual(["root", "q1", "q2"]);
    const three = buildFleetModel(root([leaf("q1", "queued"), leaf("q2", "queued"), leaf("q3", "queued")]));
    expect(three.map((entry) => entry.node.id)).toEqual(["root", "idle:root"]);
    expect(three[1].fold).toMatchObject({ kind: "idle", count: FOLD_IDLE_MIN });
  });

  it("running 永不折叠，且与已结算兄弟共存时只折 settled 组", () => {
    const model = buildFleetModel(root([
      leaf("run_1", "running"),
      leaf("d1", "completed"),
      leaf("d2", "completed"),
      leaf("d3", "completed"),
      leaf("d4", "completed"),
    ]));
    expect(model.map((entry) => entry.node.id)).toEqual(["root", "run_1", "done:root"]);
  });

  it("分支里有活跃后代（running 子代）则整支不折", () => {
    const branch = leaf("br", "completed", [leaf("br_child", "running")]);
    const model = buildFleetModel(root([
      branch,
      leaf("d1", "completed"),
      leaf("d2", "completed"),
      leaf("d3", "completed"),
      leaf("d4", "completed"),
    ]));
    expect(model.map((entry) => entry.node.id)).toEqual(["root", "br", "br_child", "done:root"]);
  });

  it("点聚合卡只展开该组：展开组成员回归，其它组保持折叠", () => {
    const tree = root([
      leaf("d1", "completed"),
      leaf("d2", "completed"),
      leaf("q1", "queued"),
      leaf("q2", "queued"),
      leaf("q3", "queued"),
    ]);
    const expanded = buildFleetModel(tree, { expandedGroups: new Set([foldGroupId("done", "root")]) });
    // 展开组：成员回归 + 聚合卡保留作收起入口；idle 组（3 个）保持折叠。
    expect(expanded.map((entry) => entry.node.id)).toEqual(["root", "d1", "d2", "done:root", "idle:root"]);
    const doneCard = expanded.find((entry) => entry.node.id === "done:root");
    // 展开态聚合卡仍渲染（收起入口），成员已可见。
    expect(doneCard?.fold).toMatchObject({ expanded: true });
  });

  it("手折优先于组展开：手折后 chevron 不失效", () => {
    const tree = root([leaf("d1", "completed"), leaf("d2", "completed")]);
    const group = foldGroupId("done", "root");
    const both = buildFleetModel(tree, {
      expandedGroups: new Set([group]),
      foldedIds: new Set([group]),
    });
    expect(both.map((entry) => entry.node.id)).toEqual(["root", group]);
    expect(both[1].fold?.expanded).toBe(false);
  });

  it("50 节点树折叠后模型收敛到可渲染规模", () => {
    const children: AgentNode[] = [];
    for (let i = 0; i < 48; i += 1) children.push(leaf(`sa_${i}`, "completed"));
    children.push(leaf("run_1", "running"), leaf("q1", "queued"));
    const model = buildFleetModel(root(children));
    // root + 1 running + 1 queued（不足 3 不成组）+ done 聚合卡 = 4。
    expect(model).toHaveLength(4);
    expect(model.filter((entry) => entry.node.kind === "fold")).toHaveLength(1);
  });
});
