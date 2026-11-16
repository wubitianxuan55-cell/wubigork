// 文学三 Hub 的纯展示解析层。所有输入均为现有 Markdown 文本 / 文件名，
// 不产生新的持久化格式，也不改变 agent 工具契约。

export function countWords(text: string): number {
  let n = 0;
  for (const ch of text) if (!/\s/.test(ch)) n++;
  return n;
}

export function initialsOf(name: string): string {
  const compact = name.replace(/\s+/g, "").trim();
  if (!compact) return "文";
  return [...compact].slice(0, 2).join("");
}

export function readingMinutes(words: number): number {
  return Math.max(1, Math.ceil(words / 420));
}

export function formatPercent(value: number, total: number): string {
  if (total <= 0) return "0%";
  return `${Math.round((value / total) * 100)}%`;
}

export interface MarkdownHeading {
  level: number;
  title: string;
}

export function markdownHeadings(text: string): MarkdownHeading[] {
  return text
    .split("\n")
    .map((line) => /^(#{1,3})\s+(.+?)\s*$/.exec(line.trim()))
    .filter((m): m is RegExpExecArray => !!m)
    .map((m) => ({ level: m[1].length, title: m[2].replace(/\s+#+$/, "").trim() }))
    .filter((h) => h.title.length > 0);
}

export function withoutLeadingH1(text: string): string {
  return text.replace(/^\s*#\s+[^\n]*(?:\r?\n)+/, "");
}

export function firstParagraph(text: string, max = 120): string {
  const body = text
    .split("\n")
    .map((line) => line.trim())
    .find((line) => !!line && !line.startsWith("#") && !line.startsWith("-"));
  const chars = [...(body ?? "").replace(/\s+/g, " ")];
  return chars.length > max ? `${chars.slice(0, max).join("")}…` : chars.join("");
}

export function inferKeywordValue(text: string, keys: string[]): string {
  for (const line of text.split("\n")) {
    const t = line.trim().replace(/^[-*•]\s*/, "");
    for (const key of keys) {
      const m = new RegExp(`^${key}\\s*[：:]\\s*(.+)$`).exec(t);
      if (m?.[1]?.trim()) return m[1].trim();
    }
  }
  return "";
}

export type SettingCategory = "世界" | "体系" | "规则" | "势力" | "人物" | "资料";

const CATEGORY_RULES: Array<[SettingCategory, RegExp]> = [
  ["体系", /力量|能力|修炼|魔法|科技|体系|等级|技能/i],
  ["规则", /规则|禁忌|律法|制度|底线|契约/i],
  ["势力", /势力|组织|家族|宗门|阵营|王朝|联盟/i],
  ["人物", /人物|角色|群像|关系/i],
  ["世界", /世界|大陆|地理|城市|区域|历史|时代|宇宙/i],
];

export function settingCategory(name: string, text = ""): SettingCategory {
  const corpus = `${name}\n${text.slice(0, 600)}`;
  return CATEGORY_RULES.find(([, pattern]) => pattern.test(corpus))?.[0] ?? "资料";
}
