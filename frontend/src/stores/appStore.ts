import { create } from 'zustand'

// ═══════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════

export interface ProjectInfo { title: string; genre: string; style: string; path: string }
export interface StatsData { totalWords: number; chapterCount: number; avgWordsPerChapter: number; characterCount: number; charAlive: number; foreshadowTotal: number; foreshadowRevealed: number; foreshadowRate: number }
export interface ProjectCard { title: string; genre: string; style: string; path: string; word_count: number; chapter_count: number; created_at: string; last_opened_at: string }

// 暗夜系列 — 6套精心调色
export type ThemePreset = 'nightJade' | 'nightViolet' | 'nightRose' | 'nightAmber' | 'nightMoss' | 'nightSlate'

export interface ThemeTokens {
  colorPrimary: string; onPrimary: string; primaryContainer: string; onPrimaryContainer: string
  surface: string; onSurface: string; surfaceVariant: string; onSurfaceVariant: string
  surfaceContainer: string; surfaceContainerHigh: string; surfaceContainerHighest: string
  surfaceDim: string; surfaceBright: string
  outline: string; outlineVariant: string
  colorBgContainer: string; colorBgLayout: string
  colorText: string; colorTextSecondary: string; colorBorder: string
  colorSuccess: string; colorWarning: string
  elevation1: string; elevation2: string; elevation3: string; elevation4: string; elevation5: string
  radiusSm: string; radiusMd: string; radiusLg: string; radiusXl: string
  transitionFast: string; transitionNormal: string; transitionSlow: string
  accentRgb: string
}

// ═══════════════════════════════════════════════════════════
// Shared
// ═══════════════════════════════════════════════════════════

const dS = { elevation1:'0 1px 2px rgba(0,0,0,0.3), 0 1px 3px rgba(0,0,0,0.15)', elevation2:'0 1px 2px rgba(0,0,0,0.3), 0 2px 6px rgba(0,0,0,0.15)', elevation3:'0 4px 8px rgba(0,0,0,0.35), 0 1px 4px rgba(0,0,0,0.15)', elevation4:'0 6px 12px rgba(0,0,0,0.4), 0 2px 6px rgba(0,0,0,0.15)', elevation5:'0 8px 24px rgba(0,0,0,0.45), 0 4px 12px rgba(0,0,0,0.2)', radiusSm:'8px', radiusMd:'12px', radiusLg:'16px', radiusXl:'28px', transitionFast:'200ms cubic-bezier(0.2,0,0,1)', transitionNormal:'300ms cubic-bezier(0.2,0,0,1)', transitionSlow:'400ms cubic-bezier(0.2,0,0,1)' }
const lS = { elevation1:'0 1px 2px rgba(0,0,0,0.08), 0 1px 3px rgba(0,0,0,0.04)', elevation2:'0 1px 2px rgba(0,0,0,0.08), 0 2px 6px rgba(0,0,0,0.04)', elevation3:'0 4px 8px rgba(0,0,0,0.12), 0 1px 4px rgba(0,0,0,0.04)', elevation4:'0 6px 12px rgba(0,0,0,0.15), 0 2px 6px rgba(0,0,0,0.04)', elevation5:'0 8px 24px rgba(0,0,0,0.18), 0 4px 12px rgba(0,0,0,0.08)', radiusSm:'8px', radiusMd:'12px', radiusLg:'16px', radiusXl:'28px', transitionFast:'200ms cubic-bezier(0.2,0,0,1)', transitionNormal:'300ms cubic-bezier(0.2,0,0,1)', transitionSlow:'400ms cubic-bezier(0.2,0,0,1)' }

// ═══════════════════════════════════════════════════════════
// 6 Dark Themes
// ═══════════════════════════════════════════════════════════

// 暗夜青 · Night Jade — 深海翡翠
const nightJadeD = (): Partial<ThemeTokens> => ({ surface:'#0a1014',surfaceDim:'#060b0e',surfaceBright:'#1a2b34',surfaceContainer:'#0f1a20',surfaceContainerHigh:'#14242e',surfaceContainerHighest:'#1b2f3a',surfaceVariant:'#152530',colorPrimary:'#2dd4bf',onPrimary:'#042f2e',primaryContainer:'#134e4a',onPrimaryContainer:'#99f6e4',onSurface:'#e2e8f0',onSurfaceVariant:'#94a3b8',colorText:'#e2e8f0',colorTextSecondary:'#94a3b8',outline:'rgba(45,212,191,0.15)',outlineVariant:'rgba(45,212,191,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#34d399',colorWarning:'#f59e0b',colorBgContainer:'#0f1a20',colorBgLayout:'#0a1014',accentRgb:'45,212,191',...dS })

// 暗夜紫 · Night Violet — 深靛星云
const nightVioletD = (): Partial<ThemeTokens> => ({ surface:'#0d0e18',surfaceDim:'#06070f',surfaceBright:'#1d1e30',surfaceContainer:'#12131e',surfaceContainerHigh:'#181928',surfaceContainerHighest:'#1f2034',surfaceVariant:'#1a1b2e',colorPrimary:'#a78bfa',onPrimary:'#1e1040',primaryContainer:'#3b2970',onPrimaryContainer:'#ddd6fe',onSurface:'#e2e0f0',onSurfaceVariant:'#9898b8',colorText:'#e2e0f0',colorTextSecondary:'#9898b8',outline:'rgba(167,139,250,0.15)',outlineVariant:'rgba(167,139,250,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#a3e635',colorWarning:'#fbbf24',colorBgContainer:'#12131e',colorBgLayout:'#0d0e18',accentRgb:'167,139,250',...dS })

// 暗夜玫 · Night Rose — 深褐暖调
const nightRoseD = (): Partial<ThemeTokens> => ({ surface:'#120c0d',surfaceDim:'#080506',surfaceBright:'#24181a',surfaceContainer:'#181012',surfaceContainerHigh:'#1f1417',surfaceContainerHighest:'#271a1d',surfaceVariant:'#1c1417',colorPrimary:'#fb7185',onPrimary:'#400c17',primaryContainer:'#64203b',onPrimaryContainer:'#fecdd3',onSurface:'#ede0e2',onSurfaceVariant:'#b8989c',colorText:'#ede0e2',colorTextSecondary:'#b8989c',outline:'rgba(251,113,133,0.15)',outlineVariant:'rgba(251,113,133,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#34d399',colorWarning:'#fbbf24',colorBgContainer:'#181012',colorBgLayout:'#120c0d',accentRgb:'251,113,133',...dS })

// 暗夜金 · Night Amber — 深色暖灯
const nightAmberD = (): Partial<ThemeTokens> => ({ surface:'#100f0a',surfaceDim:'#070603',surfaceBright:'#222015',surfaceContainer:'#15140e',surfaceContainerHigh:'#1c1a12',surfaceContainerHighest:'#242118',surfaceVariant:'#191812',colorPrimary:'#f59e0b',onPrimary:'#331c00',primaryContainer:'#594015',onPrimaryContainer:'#fde68a',onSurface:'#e6e0d4',onSurfaceVariant:'#b8b098',colorText:'#e6e0d4',colorTextSecondary:'#b8b098',outline:'rgba(245,158,11,0.15)',outlineVariant:'rgba(245,158,11,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#a3e635',colorWarning:'#f59e0b',colorBgContainer:'#15140e',colorBgLayout:'#100f0a',accentRgb:'245,158,11',...dS })

// 暗夜苔 · Night Moss — 深色林间
const nightMossD = (): Partial<ThemeTokens> => ({ surface:'#0a100f',surfaceDim:'#040807',surfaceBright:'#182420',surfaceContainer:'#0e1513',surfaceContainerHigh:'#131b18',surfaceContainerHighest:'#19221f',surfaceVariant:'#121a17',colorPrimary:'#84cc16',onPrimary:'#0a2000',primaryContainer:'#27500a',onPrimaryContainer:'#d9f99d',onSurface:'#dce4e0',onSurfaceVariant:'#9cb0a4',colorText:'#dce4e0',colorTextSecondary:'#9cb0a4',outline:'rgba(132,204,22,0.15)',outlineVariant:'rgba(132,204,22,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#84cc16',colorWarning:'#fbbf24',colorBgContainer:'#0e1513',colorBgLayout:'#0a100f',accentRgb:'132,204,22',...dS })

// 暗夜墨 · Night Slate — 极简灰
const nightSlateD = (): Partial<ThemeTokens> => ({ surface:'#0c0d10',surfaceDim:'#060608',surfaceBright:'#1e1f24',surfaceContainer:'#111216',surfaceContainerHigh:'#16171c',surfaceContainerHighest:'#1d1e23',surfaceVariant:'#15161b',colorPrimary:'#94a3b8',onPrimary:'#0f172a',primaryContainer:'#334155',onPrimaryContainer:'#cbd5e1',onSurface:'#e0e2e6',onSurfaceVariant:'#9ca3af',colorText:'#e0e2e6',colorTextSecondary:'#9ca3af',outline:'rgba(148,163,184,0.15)',outlineVariant:'rgba(148,163,184,0.06)',colorBorder:'rgba(255,255,255,0.06)',colorSuccess:'#86efac',colorWarning:'#fbbf24',colorBgContainer:'#111216',colorBgLayout:'#0c0d10',accentRgb:'148,163,184',...dS })

// ═══════════════════════════════════════════════════════════
// 6 Light Themes
// ═══════════════════════════════════════════════════════════

const nightJadeL = (): Partial<ThemeTokens> => ({ surface:'#f0fdf9',surfaceDim:'#d9f2e8',surfaceBright:'#f0fdf9',surfaceContainer:'#e6f8f2',surfaceContainerHigh:'#daf2ea',surfaceContainerHighest:'#ccede0',surfaceVariant:'#d2ebe0',colorPrimary:'#0d9488',onPrimary:'#fff',primaryContainer:'#ccfbf1',onPrimaryContainer:'#134e4a',onSurface:'#0f1a20',onSurfaceVariant:'#3d5a50',colorText:'#0f1a20',colorTextSecondary:'#3d5a50',outline:'rgba(13,148,136,0.25)',outlineVariant:'rgba(13,148,136,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#f0fdf9',accentRgb:'13,148,136',...lS })
const nightVioletL = (): Partial<ThemeTokens> => ({ surface:'#f5f3ff',surfaceDim:'#e8e2f8',surfaceBright:'#f5f3ff',surfaceContainer:'#eeebf8',surfaceContainerHigh:'#e6e0f2',surfaceContainerHighest:'#dcd4ea',surfaceVariant:'#e0daf0',colorPrimary:'#7c3aed',onPrimary:'#fff',primaryContainer:'#ede9fe',onPrimaryContainer:'#2e1065',onSurface:'#1a1522',onSurfaceVariant:'#4a4060',colorText:'#1a1522',colorTextSecondary:'#4a4060',outline:'rgba(124,58,237,0.25)',outlineVariant:'rgba(124,58,237,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#f5f3ff',accentRgb:'124,58,237',...lS })
const nightRoseL = (): Partial<ThemeTokens> => ({ surface:'#fff5f6',surfaceDim:'#fde8ec',surfaceBright:'#fff5f6',surfaceContainer:'#fbf0f2',surfaceContainerHigh:'#f5e4e8',surfaceContainerHighest:'#eed8de',surfaceVariant:'#f0e0e4',colorPrimary:'#e11d48',onPrimary:'#fff',primaryContainer:'#ffe4e6',onPrimaryContainer:'#5c1024',onSurface:'#1f1215',onSurfaceVariant:'#5c3a40',colorText:'#1f1215',colorTextSecondary:'#5c3a40',outline:'rgba(225,29,72,0.25)',outlineVariant:'rgba(225,29,72,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#fff5f6',accentRgb:'225,29,72',...lS })
const nightAmberL = (): Partial<ThemeTokens> => ({ surface:'#fffdf5',surfaceDim:'#fef3e0',surfaceBright:'#fffdf5',surfaceContainer:'#faf5e8',surfaceContainerHigh:'#f2e8d4',surfaceContainerHighest:'#e8dcc0',surfaceVariant:'#eedcc0',colorPrimary:'#d97706',onPrimary:'#fff',primaryContainer:'#fef3c7',onPrimaryContainer:'#5c3800',onSurface:'#1c1810',onSurfaceVariant:'#5c4c30',colorText:'#1c1810',colorTextSecondary:'#5c4c30',outline:'rgba(217,119,6,0.25)',outlineVariant:'rgba(217,119,6,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#fffdf5',accentRgb:'217,119,6',...lS })
const nightMossL = (): Partial<ThemeTokens> => ({ surface:'#f5fdf7',surfaceDim:'#ddf2e2',surfaceBright:'#f5fdf7',surfaceContainer:'#ebf8ef',surfaceContainerHigh:'#ddf2e4',surfaceContainerHighest:'#cde8d6',surfaceVariant:'#d8eadc',colorPrimary:'#4d7c0f',onPrimary:'#fff',primaryContainer:'#ecfccb',onPrimaryContainer:'#1a2e05',onSurface:'#101a15',onSurfaceVariant:'#3a4c3c',colorText:'#101a15',colorTextSecondary:'#3a4c3c',outline:'rgba(77,124,15,0.25)',outlineVariant:'rgba(77,124,15,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#4d7c0f',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#f5fdf7',accentRgb:'77,124,15',...lS })
const nightSlateL = (): Partial<ThemeTokens> => ({ surface:'#f8fafc',surfaceDim:'#e2e8f0',surfaceBright:'#f8fafc',surfaceContainer:'#f1f5f9',surfaceContainerHigh:'#e6ecf2',surfaceContainerHighest:'#d9e2ea',surfaceVariant:'#e0e6ec',colorPrimary:'#475569',onPrimary:'#fff',primaryContainer:'#e2e8f0',onPrimaryContainer:'#1e293b',onSurface:'#0f172a',onSurfaceVariant:'#475569',colorText:'#0f172a',colorTextSecondary:'#475569',outline:'rgba(71,85,105,0.25)',outlineVariant:'rgba(71,85,105,0.10)',colorBorder:'rgba(0,0,0,0.06)',colorSuccess:'#059669',colorWarning:'#d97706',colorBgContainer:'#fff',colorBgLayout:'#f8fafc',accentRgb:'71,85,105',...lS })

// ═══════════════════════════════════════════════════════════
// Registry
// ═══════════════════════════════════════════════════════════

const darkFn: Record<ThemePreset, () => Partial<ThemeTokens>> = { nightJade:nightJadeD, nightViolet:nightVioletD, nightRose:nightRoseD, nightAmber:nightAmberD, nightMoss:nightMossD, nightSlate:nightSlateD }
const lightFn: Record<ThemePreset, () => Partial<ThemeTokens>> = { nightJade:nightJadeL, nightViolet:nightVioletL, nightRose:nightRoseL, nightAmber:nightAmberL, nightMoss:nightMossL, nightSlate:nightSlateL }

export function getThemeTokens(base: ThemePreset, darkMode: boolean): ThemeTokens { const fn = darkMode ? darkFn[base] : lightFn[base]; return fn() as ThemeTokens }

const THEME_KEY = 'wubigork-theme'
const DARK_KEY = 'wubigork-dark'

function loadBase(): ThemePreset {
  try {
    const v = localStorage.getItem(THEME_KEY)
    if (v && v in darkFn) return v as ThemePreset
  } catch (_) {}
  return 'nightJade'
}

function loadDark(): boolean {
  try {
    const v = localStorage.getItem(DARK_KEY)
    if (v === '0') return false
  } catch (_) {}
  return true // 默认暗色
}

interface AppState {
  loggedIn: boolean
  projectOpen: boolean
  projectPath: string
  projectTitle: string
  novelsDir: string
  projects: ProjectCard[]
  baseTheme: ThemePreset      // 色系（暗色名）
  darkMode: boolean            // true=暗 false=亮
  projectInfo: ProjectInfo | null
  stats: StatsData | null
  login: () => Promise<void>
  checkLogin: () => Promise<void>
  setLoggedIn: (v: boolean) => void
  openProject: (path: string, title: string) => void
  closeProject: () => void
  loadProjects: () => Promise<void>
  loadNovelsDir: () => Promise<void>
  deleteProject: (path: string) => Promise<void>
  setTheme: (base: ThemePreset) => void
  toggleDarkMode: () => void
  loadProjectInfo: () => Promise<void>
  loadStats: () => Promise<void>
  setNovelsDir: (dir: string) => Promise<void>
}

export const useAppStore = create<AppState>((set, get) => ({
  loggedIn: false,
  projectOpen: false,
  projectPath: '',
  projectTitle: '',
  novelsDir: 'D:\\AI\\xiaoshuo',
  projects: [],
  baseTheme: loadBase(),
  darkMode: loadDark(),
  projectInfo: null,
  stats: null,

  setTheme: (base: ThemePreset) => {
    set({ baseTheme: base })
    try { localStorage.setItem(THEME_KEY, base) } catch (_) {}
  },

  toggleDarkMode: () => {
    const next = !get().darkMode
    set({ darkMode: next })
    try { localStorage.setItem(DARK_KEY, next ? '1' : '0') } catch (_) {}
  },

  login: async () => {
    try {
      // @ts-ignore — Wails Go binding
      const result = await window.go.app.App.Login()
      // login 可能触发 OAuth 流程（桌面端打开浏览器），等待登录状态确认
      set({ loggedIn: true })
    } catch (err: any) {
      // 移动端 RPC bridge 下，Login 可能因 OAuth 超时抛出异常，
      // 但实际 token 可能已被保存，轮询确认
      console.warn('login 调用异常，尝试轮询登录状态:', err)
      // 轮询最多 60 秒
      for (let i = 0; i < 30; i++) {
        await new Promise((r) => setTimeout(r, 2000))
        try {
          // @ts-ignore
          const status = await window.go.app.App.GetLoginStatus()
          if (status) {
            set({ loggedIn: true })
            return
          }
        } catch (_) {}
      }
      throw err
    }
  },

  checkLogin: async () => {
    try {
      // @ts-ignore
      const status = await window.go.app.App.GetLoginStatus()
      set({ loggedIn: status })
    } catch (_) {
      // Go 绑定未就绪时静默忽略
    }
  },

  setLoggedIn: (v: boolean) => set({ loggedIn: v }),

  openProject: (path: string, title: string) =>
    set({ projectOpen: true, projectPath: path, projectTitle: title }),

  closeProject: () =>
    set({ projectOpen: false, projectPath: '', projectTitle: '', projectInfo: null, stats: null }),

  loadProjectInfo: async () => {
    try {
      // @ts-ignore
      const info: any = await window.go.app.App.GetProjectInfo()
      if (info) set({ projectInfo: info as ProjectInfo })
    } catch (_) {}
  },

  loadStats: async () => {
    try {
      // @ts-ignore
      const s: any = await window.go.app.App.GetStats()
      if (s) set({ stats: s as StatsData })
    } catch (_) {}
  },

  loadNovelsDir: async () => {
    try {
      // @ts-ignore
      const dir: string = await window.go.app.App.GetNovelsDir()
      set({ novelsDir: dir })
    } catch (_) {
      // 保持默认值
    }
  },

  loadProjects: async () => {
    try {
      // @ts-ignore
      const cards: ProjectCard[] = await window.go.app.App.ListProjects()
      set({ projects: cards || [] })
    } catch (err) {
      console.error('loadProjects failed:', err)
    }
  },

  deleteProject: async (path: string) => {
    try {
      // @ts-ignore
      await window.go.app.App.DeleteProject(path)
      const projects = get().projects.filter((p) => p.path !== path)
      set({ projects })
    } catch (err: any) {
      console.error('deleteProject failed:', err)
      throw err
    }
  },

  setNovelsDir: async (dir: string) => {
    // @ts-ignore
    await window.go.app.App.SaveConfig('novels_dir', dir)
    set({ novelsDir: dir, projectOpen: false, projectPath: '', projectTitle: '' })
    // 刷新书架
    try {
      // @ts-ignore
      const cards: ProjectCard[] = await window.go.app.App.ListProjects()
      set({ projects: cards || [] })
    } catch (err) {
      console.error('loadProjects failed after setNovelsDir:', err)
    }
  },
}))
