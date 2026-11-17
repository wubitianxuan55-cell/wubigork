import { lineWidth } from "./textMetrics";

export interface HintItem {
  id: string;
  keys: string[];
  label: string;
}

export const HINT_GAP = 2;

export function hintWidth(item: HintItem): number {
  return lineWidth(item.keys.join("")) + 1 + lineWidth(item.label);
}

export function lineHintsWidth(items: readonly HintItem[], gap = HINT_GAP): number {
  if (items.length === 0) return 0;
  return items.reduce((sum, item) => sum + hintWidth(item), 0) + gap * (items.length - 1);
}

export function firstFittingLine(
  variants: readonly (readonly HintItem[])[],
  maxWidth: number,
  gap = HINT_GAP,
): readonly HintItem[] | null {
  for (const variant of variants) {
    if (lineHintsWidth(variant, gap) <= maxWidth) return variant;
  }
  return null;
}

export function wrapHintRows(
  items: readonly HintItem[],
  maxWidth: number,
  gap = HINT_GAP,
): HintItem[][] {
  const rows: HintItem[][] = [];
  for (const item of items) {
    const current = rows[rows.length - 1];
    if (!current || lineHintsWidth([...current, item], gap) > maxWidth) {
      rows.push([item]);
    } else {
      current.push(item);
    }
  }
  return rows;
}
