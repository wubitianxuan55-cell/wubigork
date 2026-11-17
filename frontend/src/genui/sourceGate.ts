import { OUTPUT_BUDGET } from "../gaea/lib/outputBudget";

export const GENUI_FRAGMENT_BYTE_LIMIT = OUTPUT_BUDGET.frontend.genuiFragmentByteLimit;
export const GENUI_PLACEHOLDER_ORIGIN = OUTPUT_BUDGET.frontend.genuiPlaceholderOrigin;

const utf8 = new TextEncoder();

export function genuiSourceExceedsLimit(text: string): boolean {
  if (text.length > GENUI_FRAGMENT_BYTE_LIMIT) return true;
  return utf8.encode(text).byteLength > GENUI_FRAGMENT_BYTE_LIMIT;
}

export function genuiPlaceholderUrl(key: string): string {
  try {
    return new URL(encodeURIComponent(key), GENUI_PLACEHOLDER_ORIGIN).toString();
  } catch {
    return GENUI_PLACEHOLDER_ORIGIN;
  }
}
