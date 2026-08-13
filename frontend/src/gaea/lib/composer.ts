// 输入框斜杠命令解析：只有以 "/" 开头且中间无空格时才进入命令补全。
export function slashQueryOf(text: string): string | null {
  if (!text.startsWith("/") || /\s/.test(text)) return null;
  return text.slice(1).toLowerCase();
}
