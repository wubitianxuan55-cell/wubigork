/**
 * 统一 z-index 层级管理
 *
 * 层级分配：
 *   0-49:    普通内容
 *   50-99:   Sticky 导航 (AppBar)
 *   100-149: TabBar / FAB 浮动元素
 *   200-299: Sheet / Drawer 覆盖层
 *   300-399: 桌面端侧边栏控制台
 *   1000+:   Modal / Lightbox
 *   2000+:   右键菜单 / 全屏覆盖
 */
export const Z_INDEX = {
  /** 基础内容层 */
  BASE: 0,

  /* ─── 导航层 (50-99) ─── */
  /** AppBar sticky 顶栏 */
  APP_BAR: 99,

  /* ─── 浮动元素层 (100-149) ─── */
  /** 底部 TabBar 导航 */
  TAB_BAR: 100,
  /** 浮动操作按钮 (FAB) */
  FAB: 100,
  /** AI 控制台展开/收起按钮 */
  CONSOLE_BUTTON: 100,

  /* ─── 覆盖层 (200-299) ─── */
  /** MobileSheet 蒙版 */
  SHEET_BACKDROP: 200,
  /** MobileSheet 内容面板 */
  SHEET_CONTENT: 210,

  /* ─── 侧边栏 ─── */
  /** 桌面端 AI 控制台（280px sidebar） */
  CONSOLE_SIDEBAR: 10,

  /* ─── 模态框 (1000-1099) ─── */
  MODAL: 1000,
  /** 图片 Lightbox */
  LIGHTBOX: 1050,

  /* ─── 顶层覆盖 (2000+) ─── */
  /** 右键菜单 */
  CONTEXT_MENU: 2000,
  /** 全屏遮罩 */
  FULLSCREEN_OVERLAY: 2000,
} as const
