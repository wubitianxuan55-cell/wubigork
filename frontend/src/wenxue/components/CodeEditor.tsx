// CodeEditor — CodeMirror 6 语法高亮编辑器（蒸馏规划 3a；懒加载 chunk，
// FilePreview 经 React.lazy 引入，加载失败/未就绪时回落纯 textarea）。
//
// 范围（3a 最小集）：行号 + 撤销历史 + 语法高亮（按扩展名选语言：md/js、
// ts/jsx/tsx/py/json/css/html），其余能力（搜索面板、补全、lint）不进本刀。
// 配色用中性透明底 + defaultHighlightStyle——不绑定应用明暗主题（无主题
// 检测 seam），令牌色相在明暗两套主题下均可读。
import { useEffect, useRef } from "react";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, drawSelection } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { defaultHighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { cmLanguageFor } from "../lib/cmLanguage";

export function CodeEditor({ value, path, onChange, onViewReady }: {
  value: string;
  path: string;
  onChange: (next: string) => void;
  /** 测试/高级用法：编辑器实例就绪后回调（可 dispatch 事务）。 */
  onViewReady?: (view: EditorView) => void;
}) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const ext = (path.match(/\.[^.]+$/)?.[0] ?? "").toLowerCase();
    const view = new EditorView({
      parent: host,
      state: EditorState.create({
        doc: value,
        extensions: [
          lineNumbers(),
          history(),
          drawSelection(),
          keymap.of([...defaultKeymap, ...historyKeymap]),
          syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
          ...cmLanguageFor(ext),
          EditorView.theme({
            "&": { background: "transparent", color: "inherit", fontSize: "12px" },
            ".cm-gutters": { background: "transparent", border: "none", opacity: 0.7 },
            ".cm-content": { caretColor: "inherit" },
            ".cm-activeLine": { background: "transparent" },
            ".cm-activeLineGutter": { background: "transparent" },
          }),
          EditorView.updateListener.of((u) => {
            if (u.docChanged) onChangeRef.current(u.state.doc.toString());
          }),
        ],
      }),
    });
    viewRef.current = view;
    onViewReady?.(view);
    return () => {
      view.destroy();
      viewRef.current = null;
    };
    // 仅在挂载/换文件时重建；value 由 CodeMirror 内部维护，避免光标回跳。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [path]);

  return (
    <div
      ref={hostRef}
      data-testid="code-editor"
      className="h-full w-full overflow-auto text-[12px]"
    />
  );
}
