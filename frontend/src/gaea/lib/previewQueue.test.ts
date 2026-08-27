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

  it("navTo 跳到指定索引,越界/相同无操作", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    st.openFilePreview("b.docx");
    st.openFilePreview("c.xlsx");
    let s = usePreviewStore.getState();
    s.navTo(0);
    s = usePreviewStore.getState();
    expect(s.previewFile).toBe("a.md");
    expect(s.previewIndex).toBe(0);
    s.navTo(2);
    s = usePreviewStore.getState();
    expect(s.previewFile).toBe("c.xlsx");
    s.navTo(9); // 越界
    s.navTo(-1); // 越界
    s = usePreviewStore.getState();
    expect(s.previewFile).toBe("c.xlsx");
  });

  it("closePreviewAt 关闭非当前项保持当前预览", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    st.openFilePreview("b.docx");
    st.openFilePreview("c.xlsx");
    let s = usePreviewStore.getState();
    s.navTo(0); // 当前 a.md
    s = usePreviewStore.getState();
    s.closePreviewAt(2); // 删 c.xlsx（当前之后）
    s = usePreviewStore.getState();
    expect(s.previewList).toEqual(["a.md", "b.docx"]);
    expect(s.previewFile).toBe("a.md");
    expect(s.previewIndex).toBe(0);
  });

  it("closePreviewAt 关闭当前项跳到相邻项（删前一项回退）", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    st.openFilePreview("b.docx");
    st.openFilePreview("c.xlsx");
    let s = usePreviewStore.getState();
    s.navTo(2); // 当前 c.xlsx
    s = usePreviewStore.getState();
    s.closePreviewAt(2); // 删当前（末尾）
    s = usePreviewStore.getState();
    expect(s.previewList).toEqual(["a.md", "b.docx"]);
    expect(s.previewFile).toBe("b.docx"); // 回退前一项
    expect(s.previewIndex).toBe(1);
  });

  it("closePreviewAt 关闭唯一项清空队列关闭预览", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    usePreviewStore.getState().closePreviewAt(0);
    const s = usePreviewStore.getState();
    expect(s.previewFile).toBeNull();
    expect(s.previewList).toEqual([]);
    expect(s.previewIndex).toBe(-1);
  });

  it("closePreviewAt 越界无操作", () => {
    const st = usePreviewStore.getState();
    st.openFilePreview("a.md");
    st.openFilePreview("b.docx");
    usePreviewStore.getState().closePreviewAt(5);
    const s = usePreviewStore.getState();
    expect(s.previewList).toEqual(["a.md", "b.docx"]);
  });

  it("空路径不入队", () => {
    usePreviewStore.getState().openFilePreview("");
    const s = usePreviewStore.getState();
    expect(s.previewList).toEqual([]);
  });
});
