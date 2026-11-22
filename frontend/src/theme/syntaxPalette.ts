/* eslint-disable local/no-raw-hex -- source-layer syntax data palette */
/**
 * Layer 1 data palette for source-code syntax highlighting.
 * These colours encode syntax categories, not UI chrome.
 */
export const SYNTAX_PALETTE = {
  dark: {
    keyword: '#C084FC',
    string: '#22C55E',
    number: '#FB923C',
    comment: '#64748B',
    func: '#60A5FA',
    type: '#FBBF24',
    builtin: '#38BDF8',
    meta: '#94A3B8',
  },
  light: {
    keyword: '#7C3AED',
    string: '#047857',
    number: '#C2410C',
    comment: '#64748B',
    func: '#1D4ED8',
    type: '#A16207',
    builtin: '#0369A1',
    meta: '#4B5563',
  },
} as const

export type SyntaxScheme = keyof typeof SYNTAX_PALETTE
export type SyntaxToken = keyof typeof SYNTAX_PALETTE.dark
