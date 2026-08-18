import { describe, expect, it, beforeEach } from "vitest";
import { usePreviewStore } from "./store";

describe("usePreviewStore 多文件预览队列（P1-1）", () => {
  beforeEach(() => {
    usePreviewStore.setState({ previewFile: null, previewList: [], previewIndex: -1 });
  });

  it("openFilePreview 入队并设为当前", () => {
    usePreviewStore.getState().openFilePreview("a.md");
    const s = usePreviewStore.getState();
    expect(s.previewFile).toBe("a.md");
    expect(s.previewList).toEqual(["a.md"]);
    expect(s.previewIndex).toBe(0);
  });

  it("连续打开不同文件追加到队列末尾", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    st.openFilePreview("b.docx");
    st.openFilePreview("c.xlsx");
    const s = usePreviewStore.getState();
    expect(s.previewList).toEqual(["a.md", "b.docx", "c.xlsx"]);
    expect(s.previewIndex).toBe(2);
    expect(s.previewFile).toBe("c.xlsx");
  });

  it("重复打开已在队列的文件只移动索引不重复入列", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    st.openFilePreview("b.docx");
    st.openFilePreview("a.md"); // 已在队列 → 跳回
    const s = usePreviewStore.getState();
    expect(s.previewList).toEqual(["a.md", "b.docx"]);
    expect(s.previewIndex).toBe(0);
    expect(s.previewFile).toBe("a.md");
  });

  it("navPreview 前后切换,越界无操作", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    st.openFilePreview("b.docx");
    st.openFilePreview("c.xlsx");
    let s = usePreviewStore.getState();
    s.navPreview(-1);
    s = usePreviewStore.getState();
    expect(s.previewFile).toBe("b.docx");
    expect(s.previewIndex).toBe(1);
    s.navPreview(-1);
    s.navPreview(-1); // 已到 0,越界不变
    s = usePreviewStore.getState();
    expect(s.previewFile).toBe("a.md");
    s.navPreview(1);
    s.navPreview(1);
    s.navPreview(1); // 已到末尾,越界不变
    s = usePreviewStore.getState();
    expect(s.previewFile).toBe("c.xlsx");
  });

  it("队列空时 navPreview 无操作", () => {
    const s = usePreviewStore.getState();
    s.navPreview(1);
    s.navPreview(-1);
    expect(usePreviewStore.getState().previewFile).toBeNull();
  });

  it("closeFilePreview 清空队列", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    st.openFilePreview("b.docx");
    usePreviewStore.getState().closeFilePreview();
    const s = usePreviewStore.getState();
    expect(s.previewFile).toBeNull();
    expect(s.previewList).toEqual([]);
    expect(s.previewIndex).toBe(-1);
  });

  it("空路径不入队", () => {
    usePreviewStore.getState().openFilePreview("");
    const s = usePreviewStore.getState();
    expect(s.previewList).toEqual([]);
  });
});
