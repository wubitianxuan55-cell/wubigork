/* eslint-disable react-refresh/only-export-components -- fence 辅助函数与组件同文件 */
// react-markdown code 适配：三个渲染缝共用的 genui 围栏分支。
// 用法（Markdown.tsx / ChatMarkdown.tsx / MarkdownContent 覆盖件）：
//   const lang = /language-([\w-]+)/.exec(className ?? "")?.[1];
//   if (lang && isGenuiFenceLang(lang)) return <GenuiMarkdownFence code={text} stateKey={...} />;

import { GENUI_FENCE_LANGS, type GenuiSpec } from "./spec";
import { parseGenuiFenceBody } from "./parse";
import { GenuiBlock } from "./GenuiBlock";
import { genuiStateKey } from "./fingerprint";
import type { GenuiScope } from "./scope";

export function isGenuiFenceLang(lang: string): boolean {
  return GENUI_FENCE_LANGS.has(lang);
}

/** 供缝代码快速尝试解析；失败返回 null（渲染器退化为普通代码块）。 */
export function tryParseFence(code: string): GenuiSpec | null {
  return parseGenuiFenceBody(code);
}

export function GenuiMarkdownFence({
  code,
  stateKey,
  panelRender = "block",
}: {
  code: string;
  stateKey?: string;
  /** office 对 panel:true 规格只渲染占位 chip（发布动作由宿主面板层负责）。 */
  panelRender?: "block" | "chip";
}) {
  const spec = parseGenuiFenceBody(code);
  if (spec === null) return null;
  if (spec.panel === true && panelRender === "chip") {
    return <div className="gui-panel-chip">已更新 UI 面板</div>;
  }
  return <GenuiBlock spec={spec} stateKey={stateKey} />;
}

/** 由宿主作用域 + 消息源 key + 围栏体构造稳定状态 key。 */
export function genuiFenceStateKey(
  scope: GenuiScope | null,
  sourceKey: string | undefined,
  body: string,
): string | undefined {
  if (scope === null || sourceKey === undefined) return undefined;
  return genuiStateKey(scope.scope, scope.sessionKey, sourceKey, body);
}
