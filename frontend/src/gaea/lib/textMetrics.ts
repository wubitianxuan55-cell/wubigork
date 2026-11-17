const graphemeSegmenter = typeof Intl !== "undefined" && typeof Intl.Segmenter === "function"
  ? new Intl.Segmenter(undefined, { granularity: "grapheme" })
  : null;

function graphemes(text: string): string[] {
  if (graphemeSegmenter) return Array.from(graphemeSegmenter.segment(text), (part) => part.segment);
  return Array.from(text);
}

function isZeroWidth(code: number): boolean {
  return code === 0x200d
    || (code >= 0x0300 && code <= 0x036f)
    || (code >= 0x1ab0 && code <= 0x1aff)
    || (code >= 0x1dc0 && code <= 0x1dff)
    || (code >= 0x20d0 && code <= 0x20ff)
    || (code >= 0xfe00 && code <= 0xfe0f)
    || (code >= 0xfe20 && code <= 0xfe2f)
    || (code >= 0xe0100 && code <= 0xe01ef);
}

function codePointWidth(code: number): number {
  if (code === 0 || code < 0x20 || (code >= 0x7f && code < 0xa0) || isZeroWidth(code)) return 0;
  if (code === 0x09) return 4;
  if (
    (code >= 0x1100 && code <= 0x115f)
    || (code >= 0x2329 && code <= 0x232a)
    || (code >= 0x2e80 && code <= 0xa4cf && code !== 0x303f)
    || (code >= 0xac00 && code <= 0xd7a3)
    || (code >= 0xf900 && code <= 0xfaff)
    || (code >= 0xfe10 && code <= 0xfe19)
    || (code >= 0xfe30 && code <= 0xfe6f)
    || (code >= 0xff00 && code <= 0xff60)
    || (code >= 0xffe0 && code <= 0xffe6)
    || (code >= 0x1f000 && code <= 0x1faff)
    || (code >= 0x20000 && code <= 0x3fffd)
  ) return 2;
  return 1;
}

export function graphemeWidth(grapheme: string): number {
  let width = 0;
  for (const char of grapheme) width = Math.max(width, codePointWidth(char.codePointAt(0) ?? 0));
  return width;
}

/** Display columns, not UTF-16 code units. */
export function lineWidth(text: string): number {
  let width = 0;
  for (const grapheme of graphemes(text)) width += graphemeWidth(grapheme);
  return width;
}

export function truncateToWidth(text: string, maxWidth: number, ellipsis = "…"): string {
  if (maxWidth <= 0) return "";
  if (lineWidth(text) <= maxWidth) return text;
  const ellipsisWidth = lineWidth(ellipsis);
  if (ellipsisWidth >= maxWidth) return lineWidth(ellipsis) <= maxWidth ? ellipsis : "";
  const budget = maxWidth - ellipsisWidth;
  let width = 0;
  let out = "";
  for (const grapheme of graphemes(text)) {
    const next = graphemeWidth(grapheme);
    if (width + next > budget) break;
    width += next;
    out += grapheme;
  }
  return out + ellipsis;
}

export function truncateMiddleToWidth(text: string, maxWidth: number, ellipsis = "…"): string {
  if (maxWidth <= 0) return "";
  if (lineWidth(text) <= maxWidth) return text;
  const ellipsisWidth = lineWidth(ellipsis);
  if (ellipsisWidth >= maxWidth) return lineWidth(ellipsis) <= maxWidth ? ellipsis : "";

  const budget = maxWidth - ellipsisWidth;
  const leftBudget = Math.floor(budget / 2);
  const rightBudget = budget - leftBudget;
  const parts = graphemes(text);

  let head = "";
  let width = 0;
  for (const grapheme of parts) {
    const next = graphemeWidth(grapheme);
    if (width + next > leftBudget) break;
    width += next;
    head += grapheme;
  }

  let tail = "";
  width = 0;
  for (let i = parts.length - 1; i >= 0; i--) {
    const grapheme = parts[i];
    const next = graphemeWidth(grapheme);
    if (width + next > rightBudget) break;
    width += next;
    tail = grapheme + tail;
  }
  return head + ellipsis + tail;
}
