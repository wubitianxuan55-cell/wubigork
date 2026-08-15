// T7-4（v2.37.0）outlineStore 加载三态 {data, loading, error}。
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useOutlineStore } from "./outlineStore";
import { loadOutlines } from "../components/novel/api/outlines";
import type { OutlineNode } from "../types";

vi.mock("../components/novel/api/outlines", () => ({
  loadOutlines: vi.fn(),
}));

const mocked = vi.mocked(loadOutlines);

const node = (id: string, order: number): OutlineNode => ({
  id, title: `章节 ${id}`, summary: "", status: "draft", order_index: order,
});

describe("outlineStore 加载三态（T7-4）", () => {
  beforeEach(() => {
    mocked.mockReset();
    useOutlineStore.setState({ outlines: [], storyThread: "", loading: false, error: null });
  });

  it("成功：写入 outlines/storyThread，loading 结束且 error 为空", async () => {
    mocked.mockResolvedValue({
      nodes: [node("n2", 2), node("n1", 1)],
      story_thread: "主线",
    });
    await useOutlineStore.getState().loadOutlines();
    const st = useOutlineStore.getState();
    expect(st.loading).toBe(false);
    expect(st.error).toBeNull();
    expect(st.outlines.map((n) => n.id)).toEqual(["n1", "n2"]); // 已按 order_index 排序
    expect(st.storyThread).toBe("主线");
  });

  it("失败：置 error 并结束 loading（不再无限 loading）", async () => {
    mocked.mockRejectedValue(new Error("backend down"));
    await useOutlineStore.getState().loadOutlines();
    const st = useOutlineStore.getState();
    expect(st.loading).toBe(false);
    expect(st.error).toContain("backend down");
  });

  it("API 返回 null（内部吞错）：同样视为失败，置 error", async () => {
    mocked.mockResolvedValue(null);
    await useOutlineStore.getState().loadOutlines();
    const st = useOutlineStore.getState();
    expect(st.loading).toBe(false);
    expect(st.error).toContain("加载大纲失败");
    expect(st.outlines).toEqual([]);
  });

  it("失败后重试成功：error 清除、数据恢复（三态可恢复）", async () => {
    mocked.mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce({ nodes: [node("n1", 1)] });
    await useOutlineStore.getState().loadOutlines();
    expect(useOutlineStore.getState().error).toContain("boom");

    await useOutlineStore.getState().loadOutlines(); // 重试
    const st = useOutlineStore.getState();
    expect(st.error).toBeNull();
    expect(st.outlines.map((n) => n.id)).toEqual(["n1"]);
  });

  it("加载中 loading=true 且 error 被重置", async () => {
    let resolveFn: (v: { nodes: OutlineNode[] } | null) => void = () => {};
    mocked.mockImplementation(() => new Promise((res) => { resolveFn = res; }));
    const p = useOutlineStore.getState().loadOutlines();
    const mid = useOutlineStore.getState();
    expect(mid.loading).toBe(true);
    expect(mid.error).toBeNull();
    resolveFn({ nodes: [node("n1", 1)] });
    await p;
    expect(useOutlineStore.getState().loading).toBe(false);
  });
});
