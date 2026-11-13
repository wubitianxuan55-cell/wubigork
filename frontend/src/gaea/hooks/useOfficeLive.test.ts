// useOfficeLive.test.ts — 喂入纯逻辑（波3）：工具卡扫描（office_* 过滤/状态
// 跃迁/基线期只登记）与 office_file_saved 事件解析（kind/event 两代形状 +
// Wails 顶层数组包装容错）。单例接线（useStore.subscribe + onEvent）本身薄，
// 不在本文件重复测。
import { describe, expect, it } from "vitest";
import {
  officeLiveScanItems,
  parseOfficeFileSavedEvent,
  type OfficeToolItemLike,
} from "./useOfficeLive";

const T0 = 1_700_000_000_000;

function tool(id: string, name: string, status: string, extra?: Partial<OfficeToolItemLike>): OfficeToolItemLike {
  return { kind: "tool", id, name, args: "{}", status, ...extra };
}

describe("officeLiveScanItems（工具卡扫描）", () => {
  it("只产 office_* 工具卡的活动；非 office 卡被跳过", () => {
    const items: OfficeToolItemLike[] = [
      tool("t1", "bash", "done"),
      tool("t2", "office_open", "done"),
      { kind: "assistant", id: "a1" }, // 非 tool 卡：跳过
      tool("t3", "office_generate", "running"),
    ];
    const scan = officeLiveScanItems(items, new Map(), false, T0);
    expect(scan.notes.map((n) => n.tool)).toEqual(["office_open", "office_generate"]);
  });

  it("基线期（首扫）：只登记不触发——恢复会话的历史 office 卡不点亮", () => {
    const items = [tool("t1", "office_open", "done"), tool("t2", "office_edit", "done")];
    const scan = officeLiveScanItems(items, new Map(), true, T0);
    expect(scan.notes).toEqual([]);
    expect(scan.baseline).toBe(false);
    expect(scan.seen.get("t1")).toBe("done");
    // 基线期后的新跃迁正常触发
    const scan2 = officeLiveScanItems(
      [tool("t1", "office_open", "done"), tool("t2", "office_edit", "running")],
      scan.seen, scan.baseline, T0 + 1,
    );
    expect(scan2.notes).toEqual([
      { tool: "office_edit", args: "{}", status: "running", output: undefined, error: undefined, now: T0 + 1 },
    ]);
  });

  it("状态未变的卡不重复喂（running 卡随流式刷新反复出现）", () => {
    const first = officeLiveScanItems([tool("t1", "office_edit", "running")], new Map(), false, T0);
    const again = officeLiveScanItems([tool("t1", "office_edit", "running")], first.seen, first.baseline, T0 + 1);
    expect(again.notes).toEqual([]);
  });

  it("running→done 跃迁产出带 output 的回执活动", () => {
    const items = [tool("t1", "office_edit", "done", {
      args: JSON.stringify({ ops: [{}, {}] }),
      output: "All 2 operation(s) applied (not saved yet):",
    })];
    const first = officeLiveScanItems([tool("t1", "office_edit", "running")], new Map(), false, T0);
    const second = officeLiveScanItems(items, first.seen, first.baseline, T0 + 5);
    expect(second.notes).toHaveLength(1);
    expect(second.notes[0].status).toBe("done");
    expect(second.notes[0].output).toContain("All 2 operation(s) applied");
    expect(second.notes[0].now).toBe(T0 + 5);
  });

  it("同一次扫描共享注入的 now（纯函数可测）", () => {
    const items = [tool("t1", "office_open", "running"), tool("t2", "office_save", "done")];
    const scan = officeLiveScanItems(items, new Map(), false, 42);
    expect(scan.notes.every((n) => n.now === 42)).toBe(true);
  });

  it("坏形状宽容：缺 id / 非 tool kind / 无 name 的条目跳过", () => {
    const items: OfficeToolItemLike[] = [
      { kind: "tool", name: "office_open", status: "done" }, // 缺 id
      { kind: "tool", id: "t9", status: "done" }, // 缺 name
    ];
    const scan = officeLiveScanItems(items, new Map(), false, T0);
    expect(scan.notes).toEqual([]);
  });
});

describe("parseOfficeFileSavedEvent（office_file_saved 负载解析）", () => {
  it("kind 键形状（WireEvent 同式，usePreviewProgress 口径）", () => {
    expect(parseOfficeFileSavedEvent({ kind: "office_file_saved", path: "C:/ws/a.docx", rel: "a.docx" }))
      .toEqual({ path: "C:/ws/a.docx", rel: "a.docx" });
  });
  it("event 键形状（Go emit 侧约定备选）", () => {
    expect(parseOfficeFileSavedEvent({ event: "office_file_saved", rel: "exports/b.xlsx" }))
      .toEqual({ path: "", rel: "exports/b.xlsx" });
  });
  it("Wails 顶层数组包装 → 取首个元素", () => {
    expect(parseOfficeFileSavedEvent([{ kind: "office_file_saved", path: "p.docx" }]))
      .toEqual({ path: "p.docx", rel: "" });
  });
  it("其它事件名 / 坏形状 / path 与 rel 双缺 → null", () => {
    expect(parseOfficeFileSavedEvent({ kind: "preview_progress" })).toBeNull();
    expect(parseOfficeFileSavedEvent(null)).toBeNull();
    expect(parseOfficeFileSavedEvent("office_file_saved")).toBeNull();
    expect(parseOfficeFileSavedEvent([{ kind: "office_file_saved" }])).toBeNull();
    expect(parseOfficeFileSavedEvent({ kind: "office_file_saved" })).toBeNull();
  });
});
