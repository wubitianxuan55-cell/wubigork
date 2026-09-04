// CodeEditor 单测（3a）：CodeMirror 6 在 jsdom 下挂载 + 内容/变更回调 +
// 语言映射。懒加载包装（React.lazy）由 FilePreview 承担，此处直挂组件本体。
import { describe, expect, it, vi } from "vitest";
import { render } from "@testing-library/react";
import type { EditorView } from "@codemirror/view";
import { CodeEditor } from "./CodeEditor";
import { cmLanguageFor } from "../lib/cmLanguage";

describe("CodeEditor 3a 语法高亮编辑器", () => {
  it("挂载 CodeMirror 实例并带入初值", () => {
    const { container } = render(
      <CodeEditor value={"# 标题\n正文"} path="notes/a.md" onChange={() => {}} />,
    );
    expect(container.querySelector(".cm-editor")).toBeTruthy();
    expect(container.querySelector(".cm-content")?.textContent).toContain("# 标题");
    // 行号槽 + 历史/撤销扩展就绪的间接证据：cm-gutters 存在
    expect(container.querySelector(".cm-gutters")).toBeTruthy();
  });

  it("编辑器内变更经 onChange 回传（dispatch 事务）", () => {
    const onChange = vi.fn();
    let view: EditorView | null = null;
    render(
      <CodeEditor
        value="abc"
        path="a.txt"
        onChange={onChange}
        onViewReady={(v) => (view = v)}
      />,
    );
    expect(view).toBeTruthy();
    view!.dispatch({
      changes: { from: 3, insert: "def" },
    });
    expect(onChange).toHaveBeenCalledWith("abcdef");
  });

  it("路径变化重建编辑器（key 语义由父层控制，此处验证不抛错）", () => {
    const { rerender, container } = render(
      <CodeEditor value="one" path="a.md" onChange={() => {}} />,
    );
    rerender(<CodeEditor value="two" path="b.py" onChange={() => {}} />);
    expect(container.querySelector(".cm-editor")).toBeTruthy();
  });
});

describe("cmLanguageFor 语言映射", () => {
  it("常见扩展名映射到对应语言（返回非空扩展）", () => {
    for (const ext of [".md", ".markdown", ".js", ".ts", ".tsx", ".py", ".json", ".css", ".html"]) {
      expect(cmLanguageFor(ext).length).toBeGreaterThan(0);
    }
  });
  it("未知扩展回退纯文本（空扩展数组）", () => {
    expect(cmLanguageFor(".xyz")).toEqual([]);
    expect(cmLanguageFor("")).toEqual([]);
  });
});
