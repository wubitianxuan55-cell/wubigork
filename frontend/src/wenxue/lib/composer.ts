// 输入框斜杠命令解析：只有以 "/" 开头且中间无空格时才进入命令补全。
export function slashQueryOf(text: string): string | null {
  if (!text.startsWith("/") || /\s/.test(text)) return null;
  return text.slice(1).toLowerCase();
}

export interface AtMention {
  raw: string;
  dir: string;
  frag: string;
}

// 解析输入框末尾的 @ 文件引用，拆出目录与文件名片段。
export function atMentionOf(text: string): AtMention | null {
  const m = /(?:^|\s)@([^\s]*)$/.exec(text);
  if (!m) return null;
  const raw = m[1];
  const slash = raw.lastIndexOf("/");
  return slash >= 0
    ? { raw, dir: raw.slice(0, slash + 1), frag: raw.slice(slash + 1).toLowerCase() }
    : { raw, dir: "", frag: raw.toLowerCase() };
}
