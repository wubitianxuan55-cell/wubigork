// 输入框斜杠命令分类（App.handleSend 的前置解析）：
// - "/model <ref>"   → 切换模型
// - "/memory"        → 打开记忆抽屉
// - "/context"       → 切到上下文看板标签
// - 其它             → 交给办公引擎提交
export type ComposerCommand =
  | { type: "model"; ref: string }
  | { type: "memory" }
  | { type: "context" }
  | { type: "panel"; action: "open" | "clear"; instruction?: string }
  | { type: "submit" };

export function classifyComposerCommand(input: string): ComposerCommand {
  const text = input.trim();
  const model = /^\/model\s+(\S+)$/.exec(text);
  if (model) return { type: "model", ref: model[1] };
  if (text === "/memory") return { type: "memory" };
  if (text === "/context") return { type: "context" };
  if (text === "/panel" || text === "/panel open") return { type: "panel", action: "open" };
  if (text === "/panel clear") return { type: "panel", action: "clear" };
  const panelCustom = /^\/panel\s+(.+)$/s.exec(text);
  if (panelCustom) return { type: "panel", action: "open", instruction: panelCustom[1].trim() };
  return { type: "submit" };
}
