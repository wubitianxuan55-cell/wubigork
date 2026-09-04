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

/** 流式尾部 renderPending 的 HTML：默认 HTML 白名单（span/button/data-* 保留）。 */
export function sanitizeHtml(html: string): string {
  return DOMPurify.sanitize(html);
}
