import { create } from 'zustand'

// ═══════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════

export interface ProjectInfo { title: string; genre: string; style: string; path: string }
export interface StatsData { totalWords: number; chapterCount: number; avgWordsPerChapter: number; characterCount: number; charAlive: number; foreshadowTotal: number; foreshadowRevealed: number; foreshadowRate: number; plannedChapters?: number }
export interface ProjectCard { title: string; genre: string; style: string; path: string; word_count: number; chapter_count: number; created_at: string; last_opened_at: string }

// 暗夜系列 — 6套精心调色
export type ThemePreset = 'nightJade' | 'nightViolet' | 'nightRose' | 'nightAmber' | 'nightMoss' | 'nightSlate'

// 主题元数据单一数据源（3.0 Wave 2：MainLayout 色点 / AppearancePanel 选择卡从此派生，
// 消除三处重复维护——appStore 色板表 + MainLayout themeDots + AppearancePanel themeOptions）。
export interface ThemePresetMeta {
  key: ThemePreset
  label: string
  desc: string
  /** 暗色主色（色点/预览用；与 getThemeTokens(preset, true).colorPrimary 一致） */
  color: string
}

export const THEME_PRESETS: ThemePresetMeta[] = [
  { key: 'nightJade',   label: '暗夜青', desc: '深海翡翠 · 冷静专注', color: '#2dd4bf' },
  { key: 'nightViolet', label: '暗夜紫', desc: '深靛星云 · 灵感涌动', color: '#a78bfa' },
  { key: 'nightRose',   label: '暗夜玫', desc: '深褐暖调 · 温情创作', color: '#fb7185' },
  { key: 'nightAmber',  label: '暗夜金', desc: '深色暖灯 · 沉浸舒适', color: '#f59e0b' },
  { key: 'nightMoss',   label: '暗夜苔', desc: '深色林间 · 自然舒适', color: '#84cc16' },
  { key: 'nightSlate',  label: '暗夜墨', desc: '中性深灰 · 极简克制', color: '#94a3b8' },
]

/** 主题色点查表（MainLayout 等轻量消费，避免逐次 find） */
export const THEME_PRESET_COLORS: Record<ThemePreset, string> = Object.fromEntries(
  THEME_PRESETS.map((t) => [t.key, t.color]),
) as Record<ThemePreset, string>

export const THEME_PRESET_LABELS: Record<ThemePreset, string> = Object.fromEntries(
  THEME_PRESETS.map((t) => [t.key, t.label]),
) as Record<ThemePreset, string>

export interface ThemeTokens {
  colorPrimary: string; onPrimary: string; primaryContainer: string; onPrimaryContainer: string
  surface: string; onSurface: string; surfaceVariant: string; onSurfaceVariant: string
  surfaceContainer: string; surfaceContainerHigh: string; surfaceContainerHighest: string
  surfaceDim: string; surfaceBright: string
  outline: string; outlineVariant: string
  colorBgContainer: string; colorBgLayout: string
  colorText: string; colorTextSecondary: string; colorBorder: string
  colorSuccess: string; colorWarning: string
  colorDestructive: string  // 破坏性操作红（危险确认/删除/失败态），12 主题统一语义
  elevation1: string; elevation2: string; elevation3: string; elevation4: string; elevation5: string
  radiusSm: string; radiusMd: string; radiusLg: string; radiusXl: string
  transitionFast: string; transitionNormal: string; transitionSlow: string
  accentRgb: string
  // ── 未来感扩展令牌 ──
  glow: string            // 霓虹光晕色（比 primary 更亮，用于 box-shadow / 文字发光）
  glassBg: string         // 玻璃拟态表面（半透明）
  auroraBg: string        // 深空星云渐变背景（完整 background 值）
}

// ═══════════════════════════════════════════════════════════
// Shared
// ═══════════════════════════════════════════════════════════

const dS = { elevation1:'0 1px 2px rgba(0,0,0,0.3), 0 1px 3px rgba(0,0,0,0.15)', elevation2:'0 1px 2px rgba(0,0,0,0.3), 0 2px 6px rgba(0,0,0,0.15)', elevation3:'0 4px 8px rgba(0,0,0,0.35), 0 1px 4px rgba(0,0,0,0.15)', elevation4:'0 6px 12px rgba(0,0,0,0.4), 0 2px 6px rgba(0,0,0,0.15)', elevation5:'0 8px 24px rgba(0,0,0,0.45), 0 4px 12px rgba(0,0,0,0.2)', radiusSm:'8px', radiusMd:'12px', radiusLg:'16px', radiusXl:'28px', transitionFast:'200ms cubic-bezier(0.2,0,0,1)', transitionNormal:'300ms cubic-bezier(0.2,0,0,1)', transitionSlow:'400ms cubic-bezier(0.2,0,0,1)', colorDestructive:'#ef4444' }
const lS = { elevation1:'0 1px 2px rgba(0,0,0,0.08), 0 1px 3px rgba(0,0,0,0.04)', elevation2:'0 1px 2px rgba(0,0,0,0.08), 0 2px 6px rgba(0,0,0,0.04)', elevation3:'0 4px 8px rgba(0,0,0,0.12), 0 1px 4px rgba(0,0,0,0.04)', elevation4:'0 6px 12px rgba(0,0,0,0.15), 0 2px 6px rgba(0,0,0,0.04)', elevation5:'0 8px 24px rgba(0,0,0,0.18), 0 4px 12px rgba(0,0,0,0.08)', radiusSm:'8px', radiusMd:'12px', radiusLg:'16px', radiusXl:'28px', transitionFast:'200ms cubic-bezier(0.2,0,0,1)', transitionNormal:'300ms cubic-bezier(0.2,0,0,1)', transitionSlow:'400ms cubic-bezier(0.2,0,0,1)', colorDestructive:'#dc2626' }

// ═══════════════════════════════════════════════════════════
// 6 Dark Themes
// ═══════════════════════════════════════════════════════════

// 暗夜青 · Night Jade — 深海翡翠
const nightJadeD = (): Partial<ThemeTokens> => ({ surface:'#0a1014',surfaceDim:'#060b0e',surfaceBright:'#1a2b34',surfaceContainer:'#0f1a20',surfaceContainerHigh:'#14242e',surfaceContainerHighest:'#1b2f3a',surfaceVariant:'#152530',colorPrimary:'#2dd4bf',onPrimary:'#042f2e',primaryContainer:'#134e4a',onPrimaryContainer:'#99f6e4',onSurface:'#e2e8f0',onSurfaceVariant:'#94a3b8',colorText:'#e2e8f0',colorTextSecondary:'#94a3b8',outline:'rgba(45,212,191,0.15)',outlineVariant:'rgba(45,212,191,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#34d399',colorWarning:'#f59e0b',colorBgContainer:'#0f1a20',colorBgLayout:'#0a1014',accentRgb:'45,212,191',glow:'#5eead4',glassBg:'rgba(15,26,32,0.62)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(45,212,191,0.14), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(20,184,166,0.12), transparent 55%), radial-gradient(700px 500px at 75% 15%, rgba(14,165,233,0.08), transparent 50%), linear-gradient(160deg, #0a1014 0%, #0e1a22 50%, #081218 100%)',...dS })

// 暗夜紫 · Night Violet — 深靛星云
const nightVioletD = (): Partial<ThemeTokens> => ({ surface:'#0d0e18',surfaceDim:'#06070f',surfaceBright:'#1d1e30',surfaceContainer:'#12131e',surfaceContainerHigh:'#181928',surfaceContainerHighest:'#1f2034',surfaceVariant:'#1a1b2e',colorPrimary:'#a78bfa',onPrimary:'#1e1040',primaryContainer:'#3b2970',onPrimaryContainer:'#ddd6fe',onSurface:'#e2e0f0',onSurfaceVariant:'#9898b8',colorText:'#e2e0f0',colorTextSecondary:'#9898b8',outline:'rgba(167,139,250,0.15)',outlineVariant:'rgba(167,139,250,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#a3e635',colorWarning:'#fbbf24',colorBgContainer:'#12131e',colorBgLayout:'#0d0e18',accentRgb:'167,139,250',glow:'#c4b5fd',glassBg:'rgba(18,19,30,0.62)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(167,139,250,0.16), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(99,102,241,0.13), transparent 55%), radial-gradient(700px 500px at 70% 15%, rgba(217,70,239,0.07), transparent 50%), linear-gradient(160deg, #0d0e18 0%, #131430 50%, #0a0b16 100%)',...dS })

// 暗夜玫 · Night Rose — 深褐暖调
const nightRoseD = (): Partial<ThemeTokens> => ({ surface:'#120c0d',surfaceDim:'#080506',surfaceBright:'#24181a',surfaceContainer:'#181012',surfaceContainerHigh:'#1f1417',surfaceContainerHighest:'#271a1d',surfaceVariant:'#1c1417',colorPrimary:'#fb7185',onPrimary:'#400c17',primaryContainer:'#64203b',onPrimaryContainer:'#fecdd3',onSurface:'#ede0e2',onSurfaceVariant:'#b8989c',colorText:'#ede0e2',colorTextSecondary:'#b8989c',outline:'rgba(251,113,133,0.15)',outlineVariant:'rgba(251,113,133,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#34d399',colorWarning:'#fbbf24',colorBgContainer:'#181012',colorBgLayout:'#120c0d',accentRgb:'251,113,133',glow:'#fda4af',glassBg:'rgba(24,16,18,0.62)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(251,113,133,0.14), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(244,63,94,0.12), transparent 55%), radial-gradient(700px 500px at 70% 15%, rgba(236,72,153,0.07), transparent 50%), linear-gradient(160deg, #120c0d 0%, #1d1016 50%, #0e090b 100%)',...dS })

// 暗夜金 · Night Amber — 深色暖灯
const nightAmberD = (): Partial<ThemeTokens> => ({ surface:'#100f0a',surfaceDim:'#070603',surfaceBright:'#222015',surfaceContainer:'#15140e',surfaceContainerHigh:'#1c1a12',surfaceContainerHighest:'#242118',surfaceVariant:'#191812',colorPrimary:'#f59e0b',onPrimary:'#331c00',primaryContainer:'#594015',onPrimaryContainer:'#fde68a',onSurface:'#e6e0d4',onSurfaceVariant:'#b8b098',colorText:'#e6e0d4',colorTextSecondary:'#b8b098',outline:'rgba(245,158,11,0.15)',outlineVariant:'rgba(245,158,11,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#a3e635',colorWarning:'#f59e0b',colorBgContainer:'#15140e',colorBgLayout:'#100f0a',accentRgb:'245,158,11',glow:'#fcd34d',glassBg:'rgba(21,20,14,0.62)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(245,158,11,0.13), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(249,115,22,0.10), transparent 55%), radial-gradient(700px 500px at 70% 15%, rgba(250,204,21,0.06), transparent 50%), linear-gradient(160deg, #100f0a 0%, #1a1608 50%, #0c0b07 100%)',...dS })

// 暗夜苔 · Night Moss — 深色林间
const nightMossD = (): Partial<ThemeTokens> => ({ surface:'#0a100f',surfaceDim:'#040807',surfaceBright:'#182420',surfaceContainer:'#0e1513',surfaceContainerHigh:'#131b18',surfaceContainerHighest:'#19221f',surfaceVariant:'#121a17',colorPrimary:'#84cc16',onPrimary:'#0a2000',primaryContainer:'#27500a',onPrimaryContainer:'#d9f99d',onSurface:'#dce4e0',onSurfaceVariant:'#9cb0a4',colorText:'#dce4e0',colorTextSecondary:'#9cb0a4',outline:'rgba(132,204,22,0.15)',outlineVariant:'rgba(132,204,22,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#84cc16',colorWarning:'#fbbf24',colorBgContainer:'#0e1513',colorBgLayout:'#0a100f',accentRgb:'132,204,22',glow:'#a3e635',glassBg:'rgba(14,21,19,0.62)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(132,204,22,0.13), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(101,163,13,0.11), transparent 55%), radial-gradient(700px 500px at 70% 15%, rgba(163,230,53,0.06), transparent 50%), linear-gradient(160deg, #0a100f 0%, #10180e 50%, #070c0a 100%)',...dS })

// 暗夜墨 · Night Slate — 极简灰
const nightSlateD = (): Partial<ThemeTokens> => ({ surface:'#0c0d10',surfaceDim:'#060608',surfaceBright:'#1e1f24',surfaceContainer:'#111216',surfaceContainerHigh:'#16171c',surfaceContainerHighest:'#1d1e23',surfaceVariant:'#15161b',colorPrimary:'#94a3b8',onPrimary:'#0f172a',primaryContainer:'#334155',onPrimaryContainer:'#cbd5e1',onSurface:'#e0e2e6',onSurfaceVariant:'#9ca3af',colorText:'#e0e2e6',colorTextSecondary:'#9ca3af',outline:'rgba(148,163,184,0.15)',outlineVariant:'rgba(148,163,184,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#86efac',colorWarning:'#fbbf24',colorBgContainer:'#111216',colorBgLayout:'#0c0d10',accentRgb:'148,163,184',glow:'#cbd5e1',glassBg:'rgba(17,18,22,0.62)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(148,163,184,0.13), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(100,116,139,0.11), transparent 55%), radial-gradient(700px 500px at 70% 15%, rgba(56,189,248,0.05), transparent 50%), linear-gradient(160deg, #0c0d10 0%, #13151b 50%, #090a0d 100%)',...dS })

// ═══════════════════════════════════════════════════════════
// 6 Light Themes
// ═══════════════════════════════════════════════════════════

const nightJadeL = (): Partial<ThemeTokens> => ({ surface:'#f0fdf9',surfaceDim:'#d9f2e8',surfaceBright:'#f0fdf9',surfaceContainer:'#e6f8f2',surfaceContainerHigh:'#daf2ea',surfaceContainerHighest:'#ccede0',surfaceVariant:'#d2ebe0',colorPrimary:'#0d9488',onPrimary:'#fff',primaryContainer:'#ccfbf1',onPrimaryContainer:'#134e4a',onSurface:'#0f1a20',onSurfaceVariant:'#3d5a50',colorText:'#0f1a20',colorTextSecondary:'#3d5a50',outline:'rgba(13,148,136,0.25)',outlineVariant:'rgba(13,148,136,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#f0fdf9',accentRgb:'13,148,136',glow:'#0d9488',glassBg:'rgba(255,255,255,0.55)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(45,212,191,0.16), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(20,184,166,0.12), transparent 55%), linear-gradient(160deg, #f0fdf9 0%, #e9faf4 50%, #f5fdfb 100%)',...lS })
const nightVioletL = (): Partial<ThemeTokens> => ({ surface:'#f5f3ff',surfaceDim:'#e8e2f8',surfaceBright:'#f5f3ff',surfaceContainer:'#eeebf8',surfaceContainerHigh:'#e6e0f2',surfaceContainerHighest:'#dcd4ea',surfaceVariant:'#e0daf0',colorPrimary:'#7c3aed',onPrimary:'#fff',primaryContainer:'#ede9fe',onPrimaryContainer:'#2e1065',onSurface:'#1a1522',onSurfaceVariant:'#4a4060',colorText:'#1a1522',colorTextSecondary:'#4a4060',outline:'rgba(124,58,237,0.25)',outlineVariant:'rgba(124,58,237,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#f5f3ff',accentRgb:'124,58,237',glow:'#7c3aed',glassBg:'rgba(255,255,255,0.55)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(167,139,250,0.18), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(124,58,237,0.12), transparent 55%), linear-gradient(160deg, #f5f3ff 0%, #ece7fb 50%, #faf9ff 100%)',...lS })
const nightRoseL = (): Partial<ThemeTokens> => ({ surface:'#fff5f6',surfaceDim:'#fde8ec',surfaceBright:'#fff5f6',surfaceContainer:'#fbf0f2',surfaceContainerHigh:'#f5e4e8',surfaceContainerHighest:'#eed8de',surfaceVariant:'#f0e0e4',colorPrimary:'#e11d48',onPrimary:'#fff',primaryContainer:'#ffe4e6',onPrimaryContainer:'#5c1024',onSurface:'#1f1215',onSurfaceVariant:'#5c3a40',colorText:'#1f1215',colorTextSecondary:'#5c3a40',outline:'rgba(225,29,72,0.25)',outlineVariant:'rgba(225,29,72,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#fff5f6',accentRgb:'225,29,72',glow:'#e11d48',glassBg:'rgba(255,255,255,0.55)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(251,113,133,0.16), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(225,29,72,0.10), transparent 55%), linear-gradient(160deg, #fff5f6 0%, #fdeef0 50%, #fffafa 100%)',...lS })
const nightAmberL = (): Partial<ThemeTokens> => ({ surface:'#fffdf5',surfaceDim:'#fef3e0',surfaceBright:'#fffdf5',surfaceContainer:'#faf5e8',surfaceContainerHigh:'#f2e8d4',surfaceContainerHighest:'#e8dcc0',surfaceVariant:'#eedcc0',colorPrimary:'#d97706',onPrimary:'#fff',primaryContainer:'#fef3c7',onPrimaryContainer:'#5c3800',onSurface:'#1c1810',onSurfaceVariant:'#5c4c30',colorText:'#1c1810',colorTextSecondary:'#5c4c30',outline:'rgba(217,119,6,0.25)',outlineVariant:'rgba(217,119,6,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#fffdf5',accentRgb:'217,119,6',glow:'#d97706',glassBg:'rgba(255,255,255,0.55)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(245,158,11,0.16), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(217,119,6,0.10), transparent 55%), linear-gradient(160deg, #fffdf5 0%, #fcf6e3 50%, #fffffa 100%)',...lS })
const nightMossL = (): Partial<ThemeTokens> => ({ surface:'#f5fdf7',surfaceDim:'#ddf2e2',surfaceBright:'#f5fdf7',surfaceContainer:'#ebf8ef',surfaceContainerHigh:'#ddf2e4',surfaceContainerHighest:'#cde8d6',surfaceVariant:'#d8eadc',colorPrimary:'#4d7c0f',onPrimary:'#fff',primaryContainer:'#ecfccb',onPrimaryContainer:'#1a2e05',onSurface:'#101a15',onSurfaceVariant:'#3a4c3c',colorText:'#101a15',colorTextSecondary:'#3a4c3c',outline:'rgba(77,124,15,0.25)',outlineVariant:'rgba(77,124,15,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#4d7c0f',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#f5fdf7',accentRgb:'77,124,15',glow:'#4d7c0f',glassBg:'rgba(255,255,255,0.55)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(132,204,22,0.16), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(77,124,15,0.10), transparent 55%), linear-gradient(160deg, #f5fdf7 0%, #ecf9ee 50%, #fbfefb 100%)',...lS })
const nightSlateL = (): Partial<ThemeTokens> => ({ surface:'#f8fafc',surfaceDim:'#e2e8f0',surfaceBright:'#f8fafc',surfaceContainer:'#f1f5f9',surfaceContainerHigh:'#e6ecf2',surfaceContainerHighest:'#d9e2ea',surfaceVariant:'#e0e6ec',colorPrimary:'#475569',onPrimary:'#fff',primaryContainer:'#e2e8f0',onPrimaryContainer:'#1e293b',onSurface:'#0f172a',onSurfaceVariant:'#475569',colorText:'#0f172a',colorTextSecondary:'#475569',outline:'rgba(71,85,105,0.25)',outlineVariant:'rgba(71,85,105,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#f8fafc',accentRgb:'71,85,105',glow:'#475569',glassBg:'rgba(255,255,255,0.55)',auroraBg:'radial-gradient(1100px 700px at 12% -5%, rgba(148,163,184,0.18), transparent 60%), radial-gradient(900px 600px at 88% 110%, rgba(100,116,139,0.12), transparent 55%), linear-gradient(160deg, #f8fafc 0%, #eef2f6 50%, #fcfdfe 100%)',...lS })

// ═══════════════════════════════════════════════════════════
// Registry
// ═══════════════════════════════════════════════════════════

const darkFn: Record<ThemePreset, () => Partial<ThemeTokens>> = { nightJade:nightJadeD, nightViolet:nightVioletD, nightRose:nightRoseD, nightAmber:nightAmberD, nightMoss:nightMossD, nightSlate:nightSlateD }
const lightFn: Record<ThemePreset, () => Partial<ThemeTokens>> = { nightJade:nightJadeL, nightViolet:nightVioletL, nightRose:nightRoseL, nightAmber:nightAmberL, nightMoss:nightMossL, nightSlate:nightSlateL }

export function getThemeTokens(base: ThemePreset, darkMode: boolean): ThemeTokens { const fn = darkMode ? darkFn[base] : lightFn[base]; return fn() as ThemeTokens }

const THEME_KEY = 'gaea-theme'
const LEGACY_THEME_KEY = 'wubigork-theme'
const DARK_KEY = 'gaea-dark'
const LEGACY_DARK_KEY = 'wubigork-dark'
const MODE_KEY = 'gaea-display-mode'
const DENSITY_KEY = 'gaea-density'
const MOTION_KEY = 'gaea-motion'
const ACCENT_KEY = 'gaea-accent'
const FONT_KEY = 'gaea-font-family'
const FONT_SIZE_KEY = 'gaea-font-size'

/** 预置界面字体（key → 完整 font-family 值） */
export const FONT_OPTIONS: { key: string; label: string; value: string }[] = [
  { key: 'system',   label: '系统默认', value: "system-ui, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif" },
  { key: 'yahei',    label: '微软雅黑', value: "'Microsoft YaHei', 'PingFang SC', 'Segoe UI', sans-serif" },
  { key: 'noto',     label: '思源黑体', value: "'Noto Sans SC', 'Source Han Sans SC', 'Microsoft YaHei', sans-serif" },
  { key: 'songti',   label: '宋体衬线', value: "'SimSun', 'Songti SC', 'Noto Serif SC', serif" },
  { key: 'mono',     label: '等宽字体', value: "'Cascadia Code', 'Consolas', 'JetBrains Mono', monospace" },
]

function loadFontFamily(): string {
  try {
    const v = localStorage.getItem(FONT_KEY)
    if (v && FONT_OPTIONS.some((o) => o.key === v)) return v
  } catch (_) {}
  return 'system'
}
function loadFontSize(): number {
  try {
    const v = parseInt(localStorage.getItem(FONT_SIZE_KEY) || '', 10)
    if (v >= 12 && v <= 20) return v
  } catch (_) {}
  return 14
}

/** 界面密度：standard 标准 / compact 紧凑 */
export type Density = 'standard' | 'compact'
/** 动效强度：full 完整 / reduced 减弱（可访问性） */
export type MotionPref = 'full' | 'reduced'

function loadDensity(): Density {
  try { const v = localStorage.getItem(DENSITY_KEY); if (v === 'compact' || v === 'standard') return v } catch (_) {}
  return 'standard'
}
function loadMotion(): MotionPref {
  try { const v = localStorage.getItem(MOTION_KEY); if (v === 'reduced' || v === 'full') return v } catch (_) {}
  return 'full'
}
function loadAccent(): string {
  try { return localStorage.getItem(ACCENT_KEY) || '' } catch (_) { return '' }
}

/** 显示模式：light/dark/system（system = 跟随操作系统明暗） */
export type DisplayMode = 'light' | 'dark' | 'system'

function resolveDark(mode: DisplayMode, systemDark: boolean): boolean {
  return mode === 'system' ? systemDark : mode === 'dark'
}

function systemPrefersDark(): boolean {
  try { return window.matchMedia('(prefers-color-scheme: dark)').matches } catch (_) { return true }
}

function loadMode(): DisplayMode {
  try {
    const v = localStorage.getItem(MODE_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
    // 兼容旧版 boolean（'1'=暗 '0'=亮）
    const legacy = localStorage.getItem(DARK_KEY) ?? localStorage.getItem(LEGACY_DARK_KEY)
    if (legacy === '0') return 'light'
  } catch (_) {}
  return 'dark'
}

function loadBase(): ThemePreset {
  try {
    const v = localStorage.getItem(THEME_KEY) ?? localStorage.getItem(LEGACY_THEME_KEY)
    if (v && v in darkFn) return v as ThemePreset
  } catch (_) {}
  return 'nightJade'
}

interface AppState {
  loggedIn: boolean
  projectOpen: boolean
  projectPath: string
  projectTitle: string
  novelsDir: string
  projects: ProjectCard[]
  baseTheme: ThemePreset      // 色系（暗色名）
  mode: DisplayMode            // light/dark/system
  systemDark: boolean          // 操作系统当前明暗（matchMedia）
  darkMode: boolean            // 派生实际明暗：system 时跟随 systemDark
  density: Density             // 界面密度 standard/compact
  motion: MotionPref           // 动效强度 full/reduced
  accentColor: string          // 强调色自定义（'' = 跟随主题）
  fontFamily: string           // 界面字体预设 key（FONT_OPTIONS）
  fontSize: number             // 界面字号 12-20
  projectInfo: ProjectInfo | null
  stats: StatsData | null
  login: () => Promise<void>
  logout: () => Promise<void>
  checkLogin: () => Promise<void>
  setLoggedIn: (v: boolean) => void
  openProject: (path: string, title: string) => void
  closeProject: () => void
  loadProjects: () => Promise<void>
  loadNovelsDir: () => Promise<void>
  deleteProject: (path: string) => Promise<void>
  setTheme: (base: ThemePreset) => void
  setMode: (m: DisplayMode) => void
  toggleDarkMode: () => void
  setDensity: (d: Density) => void
  setMotion: (m: MotionPref) => void
  setAccentColor: (c: string) => void
  setFontFamily: (f: string) => void
  setFontSize: (n: number) => void
  loadProjectInfo: () => Promise<void>
  loadStats: () => Promise<void>
  setNovelsDir: (dir: string) => Promise<void>
}

export const useAppStore = create<AppState>((set, get) => ({
  loggedIn: false,
  projectOpen: false,
  projectPath: '',
  projectTitle: '',
  novelsDir: 'C:\\AI\\xiaoshuo',
  projects: [],
  baseTheme: loadBase(),
  mode: loadMode(),
  systemDark: systemPrefersDark(),
  darkMode: resolveDark(loadMode(), systemPrefersDark()),
  density: loadDensity(),
  motion: loadMotion(),
  accentColor: loadAccent(),
  fontFamily: loadFontFamily(),
  fontSize: loadFontSize(),
  projectInfo: null,
  stats: null,

  setTheme: (base: ThemePreset) => {
    set({ baseTheme: base })
    try { localStorage.setItem(THEME_KEY, base) } catch (_) {}
  },

  toggleDarkMode: () => {
    const next = !get().darkMode
    set({ mode: next ? 'dark' : 'light', darkMode: next })
    try { localStorage.setItem(MODE_KEY, next ? 'dark' : 'light'); localStorage.setItem(DARK_KEY, next ? '1' : '0') } catch (_) {}
  },

  setMode: (m: DisplayMode) => {
    set({ mode: m, darkMode: resolveDark(m, get().systemDark) })
    try { localStorage.setItem(MODE_KEY, m) } catch (_) {}
  },

  setDensity: (d: Density) => {
    set({ density: d })
    try { localStorage.setItem(DENSITY_KEY, d) } catch (_) {}
  },

  setMotion: (m: MotionPref) => {
    set({ motion: m })
    try { localStorage.setItem(MOTION_KEY, m) } catch (_) {}
  },

  setAccentColor: (c: string) => {
    set({ accentColor: c })
    try { if (c) localStorage.setItem(ACCENT_KEY, c); else localStorage.removeItem(ACCENT_KEY) } catch (_) {}
  },

  setFontFamily: (f: string) => {
    set({ fontFamily: f })
    try { localStorage.setItem(FONT_KEY, f) } catch (_) {}
  },

  setFontSize: (n: number) => {
    set({ fontSize: n })
    try { localStorage.setItem(FONT_SIZE_KEY, String(n)) } catch (_) {}
  },

  login: async () => {
    try {
      // Login() 现在是异步的：立即返回，OAuth 在后台进行
      await window.go.app.App.Login()
      // 轮询等待登录完成（最多 5 分钟）
      for (let i = 0; i < 75; i++) {
        await new Promise((r) => setTimeout(r, 4000))
        try {
          const status = await window.go.app.App.GetLoginStatus()
          if (status) {
            set({ loggedIn: true })
            return
          }
        } catch (_) {}
      }
      throw new Error('登录超时：请检查浏览器是否完成了 xAI 授权')
    } catch (err: unknown) {
      console.error('login 失败:', err)
      throw err
    }
  },

  checkLogin: async () => {
    try {
      const status = await window.go.app.App.GetLoginStatus()
      set({ loggedIn: status })
    } catch (_) {
      // Go 绑定未就绪时静默忽略
    }
  },
  logout: async () => {
    try {
      await window.go.app.App.Logout()
    } catch (_) {}
    set({ loggedIn: false })
  },

  setLoggedIn: (v: boolean) => set({ loggedIn: v }),

  openProject: (path: string, title: string) =>
    set({ projectOpen: true, projectPath: path, projectTitle: title }),

  closeProject: () =>
    set({ projectOpen: false, projectPath: '', projectTitle: '', projectInfo: null, stats: null }),

  loadProjectInfo: async () => {
    try {
      const info = await window.go.app.App.GetProjectInfo()
      if (info) set({ projectInfo: info as ProjectInfo })
    } catch (_) {}
  },

  loadStats: async () => {
    try {
      const s = await window.go.app.App.GetStats()
      if (s) set({ stats: s as StatsData })
    } catch (_) {}
  },

  loadNovelsDir: async () => {
    try {
      const dir: string = await window.go.app.App.GetNovelsDir()
      set({ novelsDir: dir })
    } catch (_) {
      // 保持默认值
    }
  },

  loadProjects: async () => {
    try {
      const cards: ProjectCard[] = await window.go.app.App.ListProjects()
      set({ projects: cards || [] })
    } catch (err) {
      console.error('loadProjects failed:', err)
    }
  },

  deleteProject: async (path: string) => {
    try {
      await window.go.app.App.DeleteProject(path)
      const projects = get().projects.filter((p) => p.path !== path)
      set({ projects })
    } catch (err: unknown) {
      console.error('deleteProject failed:', err)
      throw err
    }
  },

  setNovelsDir: async (dir: string) => {
    await window.go.app.App.SaveConfig('novels_dir', dir)
    set({ novelsDir: dir, projectOpen: false, projectPath: '', projectTitle: '' })
    // 刷新书架
    try {
      const cards: ProjectCard[] = await window.go.app.App.ListProjects()
      set({ projects: cards || [] })
    } catch (err) {
      console.error('loadProjects failed after setNovelsDir:', err)
    }
  },
}))

// system 显示模式：监听操作系统明暗变化，实时派生 darkMode
if (typeof window !== 'undefined' && window.matchMedia) {
  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  const handler = (e: MediaQueryListEvent) => {
    useAppStore.setState((s) => ({
      systemDark: e.matches,
      darkMode: s.mode === 'system' ? e.matches : s.darkMode,
    }))
  }
  if (typeof mq.addEventListener === 'function') mq.addEventListener('change', handler)
}
