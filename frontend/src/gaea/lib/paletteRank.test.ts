import { describe, expect, it } from "vitest";
import { rankPaletteItems } from "./paletteRank";

// v4.30 命令面板按当前视图重排（Linear 式）——纯函数单测：
// 相关命令置顶、无关项保序（稳定排序）。

const ITEMS = [
  { id: "cmd-new", title: "新建会话" },
  { id: "cmd-memory", title: "记忆" },
  { id: "cmd-files", title: "文件面板" },
  { id: "cmd-deliverables", title: "产物面板" },
  { id: "tpl-1", title: "模板" },
  { id: "sess-1", title: "会话" },
];

describe("rankPaletteItems 按当前视图重排", () => {
  it("无视图上下文时保持原顺序（稳定）", () => {
    expect(rankPaletteItems(ITEMS, {})).toEqual(ITEMS);
  });

  it("右栏激活产物面板 → cmd-deliverables 置顶，其余命令保持相对顺序", () => {
    const out = rankPaletteItems(ITEMS, { rightTab: "deliverables" });
    expect(out[0].id).toBe("cmd-deliverables");
    // 其余项保持原相对顺序（稳定排序）
    const rest = out.slice(1).map((i) => i.id);
    expect(rest).toEqual([
      "cmd-new", "cmd-memory", "cmd-files",
      "tpl-1", "sess-1",
    ]);
  });

  it("右栏激活文件面板 → cmd-files 置顶", () => {
    const out = rankPaletteItems(ITEMS, { chatTab: "chat", rightTab: "files" });
    expect(out[0].id).toBe("cmd-files");
    expect(out[1].id).toBe("cmd-new");
  });

  it("任务模板与会话项不受视图影响，排在命令后保持原序", () => {
    const out = rankPaletteItems(ITEMS, { rightTab: "deliverables" });
    const tpl = out.findIndex((i) => i.id === "tpl-1");
    const sess = out.findIndex((i) => i.id === "sess-1");
    expect(tpl).toBeGreaterThan(-1);
    expect(sess).toBeGreaterThan(tpl);
  });

  it("不修改入参数组（纯函数）", () => {
    const snapshot = [...ITEMS];
    rankPaletteItems(ITEMS, { rightTab: "files" });
    expect(ITEMS).toEqual(snapshot);
  });
});
