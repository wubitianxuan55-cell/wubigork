// T7-4（v2.37.0）store 写路径错误可见性 + reconcileFinalAnswer 完整文本比较。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { initialState, isFinalAnswerRendered, useController, useStore } from "./store";
import type { Item } from "./store";

// 假 Wails 门面：让 realApp() 命中 CoreB；LogFrontendError 用 vi.fn 断言。
function facade(): Record<string, unknown> {
  return {
    GaeaLogFrontendError: vi.fn(async () => {}),
    GaeaMeta: async () => ({ version: "test", app: "gaea" }),
    GaeaContext: async () => ({ used: 0, window: 0 }),
    GaeaHistory: async () => [],
    GaeaBalance: async () => ({ available: true, display: "CNY 0.00" }),
    GaeaJobs: async () => [],
    GaeaFactBase: async () => ({ facts: [], markdown: "", count: 0, path: "" }),
    GaeaTCCAReport: async () => '{"ok":true}',
  };
}

type Facade = ReturnType<typeof facade>;
type G = { go?: { app?: Record<string, unknown> } };

function inject(f: Facade): void {
  (window as unknown as G).go = { app: { CoreB: f } };
}
function clearInject(): void {
  delete (window as unknown as G).go;
}
function lfeOf(f: Facade): ReturnType<typeof vi.fn> {
  return f.GaeaLogFrontendError as ReturnType<typeof vi.fn>;
}
async function flush(): Promise<void> {
  await act(async () => { await new Promise((r) => setTimeout(r, 0)); });
}
function notices(): Item[] {
  return useStore.getState().items.filter((it) => it.kind === "notice");
}
function noticeText(): string {
  const n = notices().find((it) => it.kind === "notice");
  return n?.kind === "notice" ? n.text : "";
}

describe("isFinalAnswerRendered（T7-4 完整文本比较）", () => {
  it("只渲染了前缀（流式丢尾）→ 未渲染，需要补发", () => {
    const final = "这是最终回答的完整正文内容，一共超过一百二十个字会触发旧启发式的前缀命中。";
    const rendered = "这是最终回答的完整正文内容，一共超过一百二十个字"; // 前缀，缺尾巴
    expect(isFinalAnswerRendered(rendered, final)).toBe(false);
  });

  it("渲染文本以完整正文结尾 → 已渲染", () => {
    const final = "你好，有什么可以帮你？";
    expect(isFinalAnswerRendered("第一轮回答\n你好，有什么可以帮你？", final)).toBe(true);
    expect(isFinalAnswerRendered("你好，有什么可以帮你？", final)).toBe(true);
  });

  it("完整正文出现在中间但结尾不同（更早的相同回答）→ 未渲染", () => {
    const final = "最终答案 ABC";
    expect(isFinalAnswerRendered("最终答案 ABC\n上一轮残留文字", final)).toBe(false);
  });

  it("无正文（纯推理/纯工具轮）→ 视为已渲染，不误补", () => {
    expect(isFinalAnswerRendered("", "")).toBe(true);
    expect(isFinalAnswerRendered("anything", "   ")).toBe(true);
  });

  it("渲染为空但正文非空 → 未渲染", () => {
    expect(isFinalAnswerRendered("", "答案")).toBe(false);
  });

  it("末尾空白差异不影响判定（trim 归一）", () => {
    expect(isFinalAnswerRendered("回答内容  ", "回答内容")).toBe(true);
  });
});

describe("useController 写路径失败可见化（T7-4）", () => {
  beforeEach(() => {
    // 重置共享 store，避免用例间状态污染（保留工作 _dispatch）
    useStore.setState({ ...initialState, _dispatch: useStore.getState()._dispatch });
  });
  afterEach(() => {
    clearInject();
  });

  it("send 失败：提示发送消息失败（含错误原因），不再静默", async () => {
    const f = facade();
    f.GaeaSend = vi.fn(async () => { throw new Error("network down"); });
    inject(f);
    const lfe = lfeOf(f);
    const { result } = renderHook(() => useController());
    await flush();

    act(() => { result.current.send("你好"); });
    await flush();

    expect(noticeText()).toContain("发送消息失败");
    expect(noticeText()).toContain("network down");
    expect(lfe).toHaveBeenCalledWith(expect.stringContaining("发送消息"));
  });

  it("send 失败：乐观上屏的用户消息保留（可复制重发）", async () => {
    const f = facade();
    f.GaeaSend = vi.fn(async () => { throw new Error("boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    act(() => { result.current.send("保留我"); });
    await flush();

    const user = useStore.getState().items.find((it) => it.kind === "user");
    expect(user?.kind === "user" && user.text).toBe("保留我");
  });

  it("approve 失败：保留审批弹窗（approval 不清除）并提示重试", async () => {
    const f = facade();
    f.GaeaApprove = vi.fn(async () => { throw new Error("approve boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    act(() => {
      useStore.getState()._dispatch({ type: "event", e: { kind: "approval_request", approval: { id: "a1", tool: "edit_file", subject: "x.md" } } });
    });
    expect(useStore.getState().approval?.id).toBe("a1");

    act(() => { result.current.approve("a1", "deny"); });
    await flush();

    expect(useStore.getState().approval?.id).toBe("a1"); // 弹窗保留
    expect(noticeText()).toContain("审批提交失败");
  });

  it("approve 成功：审批弹窗正常关闭", async () => {
    const f = facade();
    f.GaeaApprove = vi.fn(async () => {});
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    act(() => {
      useStore.getState()._dispatch({ type: "event", e: { kind: "approval_request", approval: { id: "a2", tool: "edit_file", subject: "y.md" } } });
    });
    act(() => { result.current.approve("a2", "allow_once"); });
    await flush();

    expect(useStore.getState().approval).toBeUndefined();
  });

  it("answerQuestion 失败：保留提问弹窗（ask 不清除）并提示", async () => {
    const f = facade();
    f.GaeaAnswer = vi.fn(async () => { throw new Error("ask boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    act(() => {
      useStore.getState()._dispatch({
        type: "event",
        e: { kind: "ask_request", ask: { id: "q1", questions: [{ id: "qq", prompt: "选哪个？", options: [{ label: "A" }] }] } },
      });
    });
    expect(useStore.getState().ask?.id).toBe("q1");

    act(() => { result.current.answerQuestion("q1", [{ questionId: "qq", selected: ["A"] }]); });
    await flush();

    expect(useStore.getState().ask?.id).toBe("q1");
    expect(noticeText()).toContain("回答提交失败");
  });

  it("cancel 失败：取消失败可见化", async () => {
    const f = facade();
    f.GaeaCancel = vi.fn(async () => { throw new Error("cancel boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    act(() => { result.current.cancel(); });
    await flush();

    expect(noticeText()).toContain("取消失败");
  });

  it("setPermLevel 失败：可见化", async () => {
    const f = facade();
    f.GaeaSetPermLevel = vi.fn(async () => { throw new Error("perm boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    act(() => { result.current.setPermLevel("auto"); });
    await flush();

    expect(noticeText()).toContain("切换权限级别失败");
  });

  it("newSession 失败：提示且不重置界面", async () => {
    const f = facade();
    f.GaeaNewSession = vi.fn(async () => { throw new Error("session boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();
    const nonceBefore = useStore.getState().sessionNonce;

    await act(async () => { await result.current.newSession(); });

    expect(noticeText()).toContain("新建会话失败");
    expect(useStore.getState().sessionNonce).toBe(nonceBefore); // 未触发 reset
  });

  it("saveDoc 失败：可见化", async () => {
    const f = facade();
    f.GaeaSaveDoc = vi.fn(async () => { throw new Error("write boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    await act(async () => { await result.current.saveDoc("a.md", "body"); });

    expect(noticeText()).toContain("保存文档失败");
    expect(noticeText()).toContain("write boom");
  });

  it("deleteSession 失败：可见化", async () => {
    const f = facade();
    f.GaeaDeleteSession = vi.fn(async () => { throw new Error("del boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    await act(async () => { await result.current.deleteSession("s1"); });

    expect(noticeText()).toContain("删除会话失败");
  });

  it("rewind 失败：可见化", async () => {
    const f = facade();
    f.GaeaRewind = vi.fn(async () => { throw new Error("rewind boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    await act(async () => { await result.current.rewind(1, "code"); });

    expect(noticeText()).toContain("回退对话失败");
  });

  it("listSessions 失败：预期降级返回 []，仅记日志不弹窗", async () => {
    const f = facade();
    f.GaeaListSessions = vi.fn(async () => { throw new Error("list boom"); });
    inject(f);
    const lfe = lfeOf(f);
    const { result } = renderHook(() => useController());
    await flush();

    const list = await result.current.listSessions();

    expect(list).toEqual([]);
    expect(lfe).toHaveBeenCalledWith(expect.stringContaining("listSessions"));
    expect(notices()).toHaveLength(0); // 降级路径不弹窗
  });

  it("remember 失败：可见化", async () => {
    const f = facade();
    f.GaeaRemember = vi.fn(async () => { throw new Error("mem boom"); });
    inject(f);
    const { result } = renderHook(() => useController());
    await flush();

    await act(async () => { await result.current.remember("work", "note"); });

    expect(noticeText()).toContain("保存记忆失败");
  });
});
