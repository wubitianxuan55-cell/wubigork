// officeLiveStore.test.ts — COM 实时编辑状态归约（波3）：office_open/edit/save/
// close 各工具形态、edit step 回执解析、close 清空、5 分钟惰性过期。
// 纯函数 officeLiveReduce 钉死口径（store 只是薄壳），解析依据 =
// internal/office/comhelper/format.go FormatEdit 的回执文本。
import { describe, expect, it, beforeEach, afterEach, vi } from "vitest";
import {
  inferOfficeAppFromPath,
  isOfficeLiveFresh,
  initialOfficeLiveState,
  OFFICE_LIVE_TTL_MS,
  officeLiveReduce,
  parseEditStepFromOutput,
  resetOfficeLiveStoreForTest,
  useOfficeLiveStore,
} from "./officeLiveStore";

const T0 = 1_700_000_000_000;

describe("inferOfficeAppFromPath（扩展名 → 应用推断）", () => {
  it("word 族：doc/docx/docm/wps/rtf", () => {
    expect(inferOfficeAppFromPath("docs/报告.docx")).toBe("word");
    expect(inferOfficeAppFromPath("C:\\ws\\通知.doc")).toBe("word");
    expect(inferOfficeAppFromPath("a/b.DOCM")).toBe("word");
    expect(inferOfficeAppFromPath("模板.wps")).toBe("word");
  });
  it("excel 族：xls/xlsx/xlsm/csv/et", () => {
    expect(inferOfficeAppFromPath("exports/成本测算.xlsx")).toBe("excel");
    expect(inferOfficeAppFromPath("a.csv")).toBe("excel");
    expect(inferOfficeAppFromPath("表.et")).toBe("excel");
  });
  it("ppt 族：ppt/pptx/dps/odp", () => {
    expect(inferOfficeAppFromPath("汇报.pptx")).toBe("ppt");
    expect(inferOfficeAppFromPath("演示.dps")).toBe("ppt");
  });
  it("认不出返回 null", () => {
    expect(inferOfficeAppFromPath("readme.md")).toBeNull();
    expect(inferOfficeAppFromPath("")).toBeNull();
    expect(inferOfficeAppFromPath("noext")).toBeNull();
  });
});

describe("parseEditStepFromOutput（FormatEdit 回执文本解析）", () => {
  it("全成：All N operation(s) applied → done=total=N", () => {
    expect(parseEditStepFromOutput("All 3 operation(s) applied (not saved yet):\n1. xxx")).toEqual({ done: 3, total: 3 });
    expect(parseEditStepFromOutput("All 0 operation(s) applied (not saved yet):")).toEqual({ done: 0, total: 0 });
  });
  it("中止：Stopped at operation i → done=i-1（total 文本没有，回 null 由调用方补）", () => {
    expect(parseEditStepFromOutput("Stopped at operation 2 (SET_TEXT): anchor mismatch")).toEqual({ done: 1, total: null });
    expect(parseEditStepFromOutput("Stopped at operation 1 (BAD_ARGS): bad")).toEqual({ done: 0, total: null });
  });
  it("认不出返回 null（诚实缺省）", () => {
    expect(parseEditStepFromOutput("Saved C:/ws/a.docx (100 bytes).")).toBeNull();
    expect(parseEditStepFromOutput(undefined)).toBeNull();
    expect(parseEditStepFromOutput("")).toBeNull();
  });
});

describe("officeLiveReduce：office_open", () => {
  it("done：activeDoc=args.path 原样，app 走扩展名推断", () => {
    const next = officeLiveReduce(initialOfficeLiveState, {
      tool: "office_open", args: JSON.stringify({ path: "docs/报告.docx" }), status: "done", now: T0,
    });
    expect(next.activeDoc).toBe("docs/报告.docx");
    expect(next.activeApp).toBe("word");
    expect(next.updatedAt).toBe(T0);
  });
  it("done：app 参数优先于扩展名推断", () => {
    const next = officeLiveReduce(initialOfficeLiveState, {
      tool: "office_open", args: JSON.stringify({ path: "docs/无扩展名", app: "excel" }), status: "done", now: T0,
    });
    expect(next.activeDoc).toBe("docs/无扩展名");
    expect(next.activeApp).toBe("excel");
  });
  it("done 且无 path（新建未保存文档）：保留现 activeDoc，只更新 app", () => {
    const base = officeLiveReduce(initialOfficeLiveState, {
      tool: "office_open", args: JSON.stringify({ path: "a.docx" }), status: "done", now: T0,
    });
    const next = officeLiveReduce(base, {
      tool: "office_open", args: JSON.stringify({ app: "ppt" }), status: "done", now: T0 + 1,
    });
    expect(next.activeDoc).toBe("a.docx");
    expect(next.activeApp).toBe("ppt");
  });
  it("running/error：只刷新心跳，不动 activeDoc", () => {
    const base = officeLiveReduce(initialOfficeLiveState, {
      tool: "office_open", args: JSON.stringify({ path: "a.docx" }), status: "done", now: T0,
    });
    const running = officeLiveReduce(base, { tool: "office_open", args: "{}", status: "running", now: T0 + 2 });
    expect(running.activeDoc).toBe("a.docx");
    expect(running.updatedAt).toBe(T0 + 2);
  });
});

describe("officeLiveReduce：office_edit（step 口径）", () => {
  const open = officeLiveReduce(initialOfficeLiveState, {
    tool: "office_open", args: JSON.stringify({ path: "docs/报告.docx" }), status: "done", now: T0,
  });

  it("running：activeDoc=args.doc，step={done:0,total}（total=ops 数组长度）", () => {
    const next = officeLiveReduce(open, {
      tool: "office_edit",
      args: JSON.stringify({ doc: "docs/报告.docx", ops: [{ op: "set_text" }, { op: "set_text" }, { op: "insert" }] }),
      status: "running", now: T0 + 1,
    });
    expect(next.activeDoc).toBe("docs/报告.docx");
    expect(next.step).toEqual({ done: 0, total: 3 });
  });
  it("running 且缺 doc：保留现 activeDoc", () => {
    const next = officeLiveReduce(open, {
      tool: "office_edit", args: JSON.stringify({ ops: [{ op: "set_text" }] }), status: "running", now: T0 + 1,
    });
    expect(next.activeDoc).toBe("docs/报告.docx");
    expect(next.step).toEqual({ done: 0, total: 1 });
  });
  it("done 且回执全成：done/total 取回执文本（权威于 ops 长度）", () => {
    const next = officeLiveReduce(open, {
      tool: "office_edit",
      args: JSON.stringify({ ops: [{}, {}] }),
      status: "done",
      output: "All 2 operation(s) applied (not saved yet):\n1. 把标题改成…\n2. 修正落款",
      now: T0 + 2,
    });
    expect(next.step).toEqual({ done: 2, total: 2 });
  });
  it("done 且回执中止：done=i-1，total 用 ops 长度补", () => {
    const next = officeLiveReduce(open, {
      tool: "office_edit",
      args: JSON.stringify({ ops: [{}, {}, {}, {}] }),
      status: "done",
      output: "Stopped at operation 3 (SET_TEXT): anchor mismatch\nApplied before it:\n1. …\n2. …",
      now: T0 + 2,
    });
    expect(next.step).toEqual({ done: 2, total: 4 });
  });
  it("done 且中止计数超出 total 时收敛到 total（clamp）", () => {
    const next = officeLiveReduce(open, {
      tool: "office_edit",
      args: JSON.stringify({ ops: [{}, {}] }),
      status: "done",
      output: "All 5 operation(s) applied (not saved yet):",
      now: T0 + 2,
    });
    expect(next.step).toEqual({ done: 2, total: 2 });
  });
  it("done 且解析不出进度：按全成计（total=ops 长度）", () => {
    const next = officeLiveReduce(open, {
      tool: "office_edit", args: JSON.stringify({ ops: [{}, {}] }), status: "done", now: T0 + 2,
    });
    expect(next.step).toEqual({ done: 2, total: 2 });
  });
  it("error 且中止文本在 error 字段：同样解析", () => {
    const next = officeLiveReduce(open, {
      tool: "office_edit",
      args: JSON.stringify({ ops: [{}, {}, {}] }),
      status: "error",
      error: "Stopped at operation 2 (INSERT_TABLE): busy",
      now: T0 + 2,
    });
    expect(next.step).toEqual({ done: 1, total: 3 });
  });
  it("error 且无进度信息：保持现 step（不假装知道进度）", () => {
    const ran = officeLiveReduce(open, {
      tool: "office_edit", args: JSON.stringify({ ops: [{}, {}] }), status: "running", now: T0 + 1,
    });
    const next = officeLiveReduce(ran, {
      tool: "office_edit", args: JSON.stringify({ ops: [{}, {}] }), status: "error", error: "helper 超时", now: T0 + 2,
    });
    expect(next.step).toEqual({ done: 0, total: 2 });
  });
});

describe("officeLiveReduce：office_save / office_close", () => {
  const open = officeLiveReduce(initialOfficeLiveState, {
    tool: "office_open", args: JSON.stringify({ path: "a.docx" }), status: "done", now: T0,
  });
  const edited = officeLiveReduce(open, {
    tool: "office_edit", args: JSON.stringify({ ops: [{}, {}] }), status: "running", now: T0 + 1,
  });

  it("save done：step 清空（无进行中批次），activeDoc 保留（原地保存）", () => {
    const next = officeLiveReduce(edited, {
      tool: "office_save", args: JSON.stringify({}), status: "done",
      output: "Saved C:/ws/a.docx (100 bytes).", now: T0 + 3,
    });
    expect(next.activeDoc).toBe("a.docx");
    expect(next.step).toBeNull();
    expect(next.updatedAt).toBe(T0 + 3);
  });
  it("save done 且给 path（另存为）：activeDoc 切到落点", () => {
    const next = officeLiveReduce(edited, {
      tool: "office_save", args: JSON.stringify({ path: "exports/最终.docx" }), status: "done", now: T0 + 3,
    });
    expect(next.activeDoc).toBe("exports/最终.docx");
    expect(next.activeApp).toBe("word");
  });
  it("save error：只心跳", () => {
    const next = officeLiveReduce(edited, {
      tool: "office_save", args: JSON.stringify({}), status: "error", error: "文档被占用", now: T0 + 3,
    });
    expect(next.activeDoc).toBe("a.docx");
    expect(next.step).toEqual({ done: 0, total: 2 });
  });
  it("close done：整组清空（编辑会话结束）", () => {
    const next = officeLiveReduce(edited, {
      tool: "office_close", args: JSON.stringify({}), status: "done", now: T0 + 4,
    });
    expect(next.activeDoc).toBeNull();
    expect(next.activeApp).toBeNull();
    expect(next.step).toBeNull();
    expect(next.updatedAt).toBe(T0 + 4);
  });
  it("close error：不清（关闭失败文档还开着）", () => {
    const next = officeLiveReduce(edited, {
      tool: "office_close", args: JSON.stringify({}), status: "error", error: "保存被拒绝", now: T0 + 4,
    });
    expect(next.activeDoc).toBe("a.docx");
  });
});

describe("officeLiveReduce：心跳 / 过期 / 非 office 工具", () => {
  it("office_read/status/render/help：纯心跳（保留 activeDoc/step）", () => {
    const open = officeLiveReduce(initialOfficeLiveState, {
      tool: "office_open", args: JSON.stringify({ path: "a.xlsx" }), status: "done", now: T0,
    });
    for (const tool of ["office_read", "office_status", "office_render", "office_help"]) {
      const next = officeLiveReduce(open, { tool, args: "{}", status: "done", now: T0 + 10 });
      expect(next.activeDoc).toBe("a.xlsx");
      expect(next.updatedAt).toBe(T0 + 10);
    }
  });
  it("非 office_* 工具：原样返回（不刷心跳）", () => {
    const next = officeLiveReduce(initialOfficeLiveState, { tool: "bash", args: "{}", status: "done", now: T0 });
    expect(next).toBe(initialOfficeLiveState);
  });
  it("5 分钟过期：下一次活动从清空态开始（惰性，无定时器）", () => {
    const open = officeLiveReduce(initialOfficeLiveState, {
      tool: "office_open", args: JSON.stringify({ path: "a.docx" }), status: "done", now: T0,
    });
    // TTL 内：另存为不会丢当前文档语义（save path 分支正常）
    const within = officeLiveReduce(open, {
      tool: "office_save", args: JSON.stringify({ path: "b.docx" }), status: "done", now: T0 + OFFICE_LIVE_TTL_MS - 1,
    });
    expect(within.activeDoc).toBe("b.docx");
    // 超过 TTL：旧活动文档清空，新 open 从零开始
    const expired = officeLiveReduce(open, {
      tool: "office_open", args: JSON.stringify({ path: "c.docx" }), status: "done", now: T0 + OFFICE_LIVE_TTL_MS + 1,
    });
    expect(expired.activeDoc).toBe("c.docx");
    expect(expired.updatedAt).toBe(T0 + OFFICE_LIVE_TTL_MS + 1);
    // 且旧 activeDoc 不是被「保留」下来的——先清空再应用才得到 c.docx
    const heartbeatOnly = officeLiveReduce(open, {
      tool: "office_read", args: "{}", status: "done", now: T0 + OFFICE_LIVE_TTL_MS + 2,
    });
    expect(heartbeatOnly.activeDoc).toBeNull();
  });
  it("isOfficeLiveFresh：有文档 + TTL 内 = 新鲜；超时/无文档 = 否", () => {
    const open = officeLiveReduce(initialOfficeLiveState, {
      tool: "office_open", args: JSON.stringify({ path: "a.docx" }), status: "done", now: T0,
    });
    expect(isOfficeLiveFresh(open, T0 + 1000)).toBe(true);
    expect(isOfficeLiveFresh(open, T0 + OFFICE_LIVE_TTL_MS)).toBe(true); // 边界内
    expect(isOfficeLiveFresh(open, T0 + OFFICE_LIVE_TTL_MS + 1)).toBe(false);
    expect(isOfficeLiveFresh(initialOfficeLiveState, T0)).toBe(false);
  });
});

describe("useOfficeLiveStore（zustand 薄壳）", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(T0);
    resetOfficeLiveStoreForTest();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("noteActivity 更新状态；clear 复位", () => {
    useOfficeLiveStore.getState().noteActivity("office_open", JSON.stringify({ path: "docs/x.xlsx" }), "done");
    expect(useOfficeLiveStore.getState().activeDoc).toBe("docs/x.xlsx");
    expect(useOfficeLiveStore.getState().activeApp).toBe("excel");
    useOfficeLiveStore.getState().clear();
    expect(useOfficeLiveStore.getState().activeDoc).toBeNull();
    expect(useOfficeLiveStore.getState().updatedAt).toBe(0);
  });
  it("expireIfStale：TTL 内不清；越过 TTL 就地清空", () => {
    useOfficeLiveStore.getState().noteActivity("office_open", JSON.stringify({ path: "a.docx" }), "done");
    vi.advanceTimersByTime(OFFICE_LIVE_TTL_MS - 1000);
    useOfficeLiveStore.getState().expireIfStale();
    expect(useOfficeLiveStore.getState().activeDoc).toBe("a.docx");
    vi.advanceTimersByTime(2000);
    useOfficeLiveStore.getState().expireIfStale();
    expect(useOfficeLiveStore.getState().activeDoc).toBeNull();
    expect(useOfficeLiveStore.getState().updatedAt).toBe(0);
  });
  it("expireIfStale：本就无活动文档时不动（含 updatedAt=0 初值）", () => {
    useOfficeLiveStore.getState().expireIfStale();
    expect(useOfficeLiveStore.getState().updatedAt).toBe(0);
  });
});
