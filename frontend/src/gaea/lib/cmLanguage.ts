// cmLanguage — CodeMirror 语言支持按扩展名映射（3a；纯函数，供编辑器与测试）。
import type { Extension } from "@codemirror/state";
import { markdown } from "@codemirror/lang-markdown";
import { javascript } from "@codemirror/lang-javascript";
import { python } from "@codemirror/lang-python";
import { json } from "@codemirror/lang-json";
import { css as cssLang } from "@codemirror/lang-css";
import { html as htmlLang } from "@codemirror/lang-html";

/** 按扩展名挑语言支持（未知扩展返回空数组=纯文本）。ext 含点、小写。 */
export function cmLanguageFor(ext: string): Extension[] {
  switch (ext) {
    case ".md":
    case ".markdown":
    case ".mdx":
      return [markdown()];
    case ".js":
    case ".jsx":
    case ".mjs":
    case ".cjs":
      return [javascript()];
    case ".ts":
    case ".tsx":
      return [javascript({ typescript: true, jsx: true })];
    case ".py":
      return [python()];
    case ".json":
      return [json()];
    case ".css":
      return [cssLang()];
    case ".html":
    case ".htm":
      return [htmlLang()];
    default:
      return [];
  }
}
