// 输入框斜杠命令分类（App.handleSend 的前置解析）：
// - "/model <ref>"   → 切换模型
// - "/memory"        → 打开记忆抽屉
// - 其它             → 交给办公引擎提交
export type ComposerCommand =
  | { type: "model"; ref: string }
  | { type: "memory" }
  | { type: "submit" };

export function classifyComposerCommand(input: string): ComposerCommand {
  const text = input.trim();
  const model = /^\/model\s+(\S+)$/.exec(text);
  if (model) return { type: "model", ref: model[1] };
  if (text === "/memory") return { type: "memory" };
  return { type: "submit" };
}
