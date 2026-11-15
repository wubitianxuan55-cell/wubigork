export const JSONL_RECORD_LIMIT = 500;
export const JSONL_RECORD_PREVIEW_CHARS = 4000;

export interface JsonlRecord {
  line: number;
  raw: string;
  valid: boolean;
  summary: string;
  formatted: string;
  truncated: boolean;
}

function summarize(value: unknown): string {
  if (value === null) return "null";
  if (Array.isArray(value)) return `数组 · ${value.length} 项`;
  if (typeof value !== "object") return String(value);
  const rec = value as Record<string, unknown>;
  const preferred = ["title", "name", "subject", "type", "kind", "role", "status", "id", "event"];
  const parts: string[] = [];
  for (const key of preferred) {
    const v = rec[key];
    if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") {
      parts.push(`${key}: ${String(v)}`);
    }
    if (parts.length >= 3) break;
  }
  if (parts.length < 3) {
    for (const [key, v] of Object.entries(rec)) {
      if (preferred.includes(key)) continue;
      if (typeof v === "string" || typeof v === "number" || typeof v === "boolean") {
        parts.push(`${key}: ${String(v)}`);
      }
      if (parts.length >= 3) break;
    }
  }
  if (parts.length > 0) return parts.join(" · ");
  const keys = Object.keys(rec);
  return keys.length > 0 ? `对象 · ${keys.slice(0, 5).join(", ")}${keys.length > 5 ? "…" : ""}` : "空对象";
}

/** Parse bounded JSONL records without letting one bad line blank the file. */
export function parseJsonlRecords(
  text: string,
  limit = JSONL_RECORD_LIMIT,
  previewChars = JSONL_RECORD_PREVIEW_CHARS,
): JsonlRecord[] {
  const out: JsonlRecord[] = [];
  const max = Math.max(1, Math.min(limit, JSONL_RECORD_LIMIT));
  const lines = text.split(/\r?\n/);
  for (let i = 0; i < lines.length && out.length < max; i++) {
    const rawLine = lines[i];
    if (rawLine.trim() === "") continue;
    try {
      const value: unknown = JSON.parse(rawLine);
      const pretty = JSON.stringify(value, null, 2) ?? String(value);
      out.push({
        line: i + 1,
        raw: rawLine,
        valid: true,
        summary: summarize(value),
        formatted: pretty.slice(0, previewChars),
        truncated: pretty.length > previewChars,
      });
    } catch {
      out.push({
        line: i + 1,
        raw: rawLine.slice(0, previewChars),
        valid: false,
        summary: "无效 JSON",
        formatted: rawLine.slice(0, previewChars),
        truncated: rawLine.length > previewChars,
      });
    }
  }
  return out;
}
