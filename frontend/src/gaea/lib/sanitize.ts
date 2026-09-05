// sanitize — HTML 注入点的统一消毒层（蒸馏规划 3b，DOMPurify）。
//
// 接入点与口径：
//  1. MemoMarkdown 流式尾部 renderPending 的 HTML 串——各分支已逐段转义，
//     消毒层兜底未来回归（新增分支忘转义不致 XSS）。✅ 本层负责。
//  2. Mermaid 渲染产物（Markdown.tsx 经 innerHTML 插入）——**不做外层
//     再消毒**：mermaid v11 在 securityLevel:"strict"（本项目显式配置）
//     下已用内置 DOMPurify 消毒输出；实测（vite 页面配置矩阵实验 +
//     ?mock=1 三轮走查）外层 DOMPurify 的 svg profile 必然剥离
//     foreignObject 内的 html 标签（节点文字全失），ADD_TAGS 无法补救
//     （svg 命名空间校验拒绝 html 标签）。功能性破坏 > 边际防御收益，
//     strict 上游是唯一正解。scripts/事件处理器由 strict 层拦截。
// 纯函数，可单测。
import DOMPurify from "dompurify";
import rehypeSanitize from "rehype-sanitize";
import type { Schema } from "hast-util-sanitize";
import type { Pluggable } from "unified";

/** 流式尾部 renderPending 的 HTML：默认 HTML 白名单（span/button/data-* 保留）。 */
export function sanitizeHtml(html: string): string {
  return DOMPurify.sanitize(html);
}

// ── FilePreview 内嵌 HTML 白名单（蒸馏规划 3b 遗留刀）────────────────
// 范围：Markdown.tsx（文件预览/AskCard/KnowledgePanel 的 md 渲染链）。
// MemoMarkdown（聊天流）不在此刀范围，仍走上方 sanitizeHtml 尾部消毒，
// 两条渲染链并存、互不改动。
//
// 消毒发生在 raw 解析后、React 渲染前（全程不经 dangerouslySetInnerHTML）：
// rehype-raw 把 raw HTML 解析进整棵 hast 后，rehype-sanitize 按显式 schema
// （标签/属性白名单 + 敏感标签连内容 strip）先消毒、再渲染。
//
// 为何树级 rehype-sanitize 而非 DOMPurify：
//  1. 唯一可行的注入点是 hast 树，树级消毒与 md 管线产物同树统一过滤，
//     且能按「值」保住渲染链命脉 class（KaTeX math-* / 围栏代码
//     language-*，值受限正则，见 mdHtmlSchema——raw HTML 自带的任意
//     class 过不了这两个正则）。
//  2. 实测：remark 把行内 HTML 拆成多个 raw 片段（"<b>" / 文本 / "</b>"），
//     逐片段过 DOMPurify 时 parse5 会补全残缺标签（"<b>"→"<b></b>"），
//     rehype-raw 重组后文本被顶出标签——字符串级消毒与 raw 片段语义
//     天然冲突，此路不通。
//  3. 消毒产物由 React createElement 渲染，无 HTML 序列化再解析步骤，
//     没有 sanitizer/serializer 回转的 mXSS 面（svg/math 命名空间混淆
//     类攻击不适用）。
//
// 协议白名单（http/https/mailto/相对路径/#锚点；javascript:/data: 除
// data:image/* 必剥）不在 schema.protocols 落地（显式给空对象）：树级
// safeProtocol 无法区分节点来源，会把 md 管线自产的 mem: 记忆引用、本地
// 路径 href（C:\ 盘符被当作协议）一并剥掉——FileChip/记忆徽标即回归。
// 协议收口在 Markdown.tsx 渲染层：mdUrlTransform 只放行本地路径/mem:/
// data:image/*，其余（javascript:/data:text/html 等）被 defaultUrlTransform
// 剥成空串；点击另有 classifyExternalLink 分流（loopback 拒）兜底。

// README 级务实白名单标签：排版强调 / 结构块 / 媒体 / 表格 / 交互披露。
const MD_HTML_TAGS = [
  "a", "b", "i", "em", "strong", "u", "s", "del", "ins", "sub", "sup",
  "br", "hr", "span", "img", "code", "kbd", "mark", "abbr", "q", "cite",
  "blockquote", "pre", "p", "div", "ul", "ol", "li",
  "table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption",
  "h1", "h2", "h3", "h4", "h5", "h6",
  "details", "summary", "figure", "figcaption", "center", "small",
  // input 是 remark-gfm 任务列表（- [ ] / - [x]）的复选框载体，styles.css
  // 的 .md input[type="checkbox"] 既有样式依赖它——必须放行（红线：不许因
  // 消毒丢既有能力）；属性按值受限（见 attributes.input），raw HTML 的
  // <input type="text|radio|..."> 会被剥成无类型空 input，无交互面。
  "input",
];

// 白名单之外且内容敏感的标签：树级连内容一起丢弃（unwrap 会把脚本体
// 暴露成正文文本）。不在其中的（font/video/dialog 等）unwrap 保留纯文本。
const MD_HTML_STRIP = [
  "script", "style", "noscript", "noembed", "template", "iframe", "frame",
  "frameset", "object", "embed", "applet", "svg", "math", "form",
  "button", "select", "option", "textarea", "datalist", "head", "meta",
  "link", "base", "title", "xmp", "plaintext",
];

// 显式 schema（hast property 名）。要点：
//  - tagNames/strip 见上；on*/style/id/data-* 靠不在 attributes 里剥净
//    （hast 里事件属性只是普通 attribute，schema 不放行即剥）。
//  - className 仅按值放行两个 md 管线自产类域（propertyValueMany 逐
//    token 过滤）：span 的 KaTeX math-inline/math-display、code 的
//    language-*。raw HTML 携带的任意 class 过不了这两个正则，同样被剥。
//  - protocols 刻意省略，理由见本节头注。
//  - protocols 显式给空对象：hast-util-sanitize 内部按 {...defaultSchema,
//    ...options} 浅合并，缺省会把 defaultSchema.protocols（href 仅
//    http/https/irc/mailto/xmpp、src 仅 http/https）带进来——md 管线自产
//    的 mem: 引用、C:\ 本地路径 href（盘符被解析成 "c:" 协议）、
//    data:image/* 都会被它剥掉（FileChip/记忆徽标/内嵌图回归）。协议收口
//    统一走渲染层 mdUrlTransform + classifyExternalLink（见下）。
export const mdHtmlSchema: Schema = {
  tagNames: MD_HTML_TAGS,
  strip: MD_HTML_STRIP,
  protocols: {},
  ancestors: {
    thead: ["table"],
    tbody: ["table"],
    tfoot: ["table"],
    tr: ["table"],
    td: ["table"],
    th: ["table"],
  },
  attributes: {
    a: ["href", "title"],
    img: ["src", "alt", "title", "width", "height"],
    abbr: ["title"],
    q: ["cite"],
    blockquote: ["cite"],
    del: ["cite"],
    ins: ["cite"],
    th: ["colspan", "rowspan", "scope", "align"],
    td: ["colspan", "rowspan", "align"],
    ol: ["start", "reversed", "type"],
    li: ["value"],
    details: ["open"],
    span: [["className", /^math-(inline|display)$/]],
    code: [["className", /^language-./]],
    input: [["type", "checkbox"], "disabled", "checked"],
  },
  required: { img: { alt: "" } },
};

/** 整棵 hast 渲染前消毒（rehype-raw 之后、rehype-katex 之前）。
 *
 * 用法（Pluggable 元组，让 unified 以正确参数调用插件工厂）：
 *   rehypePlugins={[rehypeRaw, [rehypeSanitize, mdHtmlSchema], rehypeKatex]}
 * 切勿把 rehypeSanitize(mdHtmlSchema) 的返回值（transformer）直接塞进
 * rehypePlugins——unified 会把它当 plugin 无参调用（tree=undefined →
 * 静默 no-op，消毒形同虚设）。
 */
export const mdHtmlSchemaSanitize: Pluggable = [rehypeSanitize, mdHtmlSchema];
