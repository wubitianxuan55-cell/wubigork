export interface MarkdownHeading {
  id: string;
  level: number;
  line: number;
  text: string;
}

function cleanHeading(text: string): string {
  return text
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/[*_~]/g, "")
    .trim();
}

/** Extract ATX Markdown headings outside fenced code blocks. */
export function extractMarkdownOutline(text: string): MarkdownHeading[] {
  const out: MarkdownHeading[] = [];
  let fence: "`" | "~" | null = null;
  const lines = text.split(/\r?\n/);
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const marker = line.match(/^\s*(`{3,}|~{3,})/)?.[1]?.[0];
    if (marker === "`" || marker === "~") {
      if (fence === marker) fence = null;
      else if (fence === null) fence = marker;
      continue;
    }
    if (fence !== null) continue;
    const match = line.match(/^\s{0,3}(#{1,6})\s+(.+?)\s*#*\s*$/);
    if (!match) continue;
    const heading = cleanHeading(match[2]);
    if (!heading) continue;
    out.push({
      id: `md-heading-${out.length + 1}`,
      level: match[1].length,
      line: i + 1,
      text: heading,
    });
  }
  return out;
}
