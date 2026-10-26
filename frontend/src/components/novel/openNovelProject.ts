// openNovelProject — 打开一本小说书的完整链路（共享件，2026-10-26 抽自
// pages/HomePage.tsx handleOpen）。
//
// Why: 该链路（后端 OpenProject → 前端 store 落位 → 可选直达阅读 tab →
// 最近对象记账）原先只存在于书架页闭包内；首页「继续」区续做书（方向一二期）
// 需要从组件外发起同一条链路。抽取后书架与首页共用一份真相，打开即记账。
//
// 注意：切书前的未保存闸门（guardNovelSwitch）不在此函数内——闸门属于调用方
// 的决策层（书架 handleOpen / 首页续做行各自包），本函数只做"真正打开"。
import { wailsApp } from '../../lib/wailsApp'
import { useAppStore } from '../../stores/appStore'
import { recordRecentObject } from '../../gaea/lib/recentObjects'

/**
 * 打开（切换到）一本小说书。
 *
 * @param path 项目目录（身份键）
 * @param title 书名（store 落位与记账展示名）
 * @param opts.goRead 打开后直达阅读 tab（书架"继续阅读"入口用；NovelPage
 *   常驻挂载时事件可达，冷挂载场景调用方应先确保小说板块已挂载）
 */
export async function openNovelProject(path: string, title: string, opts?: { goRead?: boolean }): Promise<void> {
  await wailsApp().OpenProject(path)
  useAppStore.getState().openProject(path, title)
  if (opts?.goRead) {
    window.dispatchEvent(new CustomEvent('novel:goto-tab', { detail: { tab: 'chapter' } }))
  }
  recordRecentObject('novel', path, title)
}
