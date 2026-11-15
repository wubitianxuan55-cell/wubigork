import type { Item } from "./store";

export function exportAsMarkdown(items: Item[]): string {
  const lines: string[] = [`# gaea 会话导出\n> ${new Date().toLocaleString()}\n`];
  for (const it of items) {
    switch (it.kind) {
      case "user": lines.push(`### 👤 用户\n\n${it.text}\n`); break;
      case "assistant":
        if (it.reasoning) lines.push(`<details>\n<summary>💭 思考过程</summary>\n\n${it.reasoning}\n\n</details>\n`);
        if (it.text) lines.push(`${it.text}\n`);
        break;
      case "tool":
        lines.push(`### ${it.status==="error"?"❌":it.status==="running"?"⏳":"✅"} ${it.name}\n`);
        if (it.output) lines.push(`\`\`\`\n${it.output}\n\`\`\`\n`);
        if (it.error) lines.push(`> ❌ ${it.error}\n`);
        break;
      case "notice": lines.push(`> ${it.level==="warn"?"⚠️":"ℹ️"} ${it.text}\n`); break;
      case "phase": lines.push(`> 📌 ${it.text}\n`); break;
    }
  }
  return lines.join("\n");
}
