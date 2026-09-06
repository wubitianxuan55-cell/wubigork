/**
 * schedule/sample.ts — 示例工程（某办公楼施工进度计划）
 *
 * 覆盖分组行、里程碑、四种搭接（FS/SS/FF/SF）与时距，
 * 用于三视图切换演示与 CPM/双代号转换的自证数据。
 */
import type { SchedProject } from './types'

let seq = 0
const nid = (p: string) => `${p}${++seq}`

export function makeSampleProject(): SchedProject {
  seq = 0
  const g1 = nid('g')
  const g2 = nid('g')
  const g3 = nid('g')
  const g4 = nid('g')
  const g5 = nid('g')
  const t = {
    permit: nid('t'), site: nid('t'), drawing: nid('t'),
    earth: nid('t'), cushion: nid('t'), rebar: nid('t'), foundation: nid('t'),
    main1: nid('t'), main2: nid('t'),
    embed: nid('t'), masonry: nid('t'), finish: nid('t'), mep: nid('t'),
    accept: nid('t'), deliver: nid('t'),
  }
  return {
    name: '示例：办公楼施工进度计划',
    startDate: '2026-09-01',
    tasks: [
      { id: g1, name: '前期准备', duration: 0, level: 0, progress: 0 },
      { id: t.site, name: '场地三通一平', duration: 10, level: 1, progress: 60 },
      { id: t.permit, name: '施工许可办理', duration: 15, level: 1, progress: 40 },
      { id: t.drawing, name: '施工图会审', duration: 5, level: 1, progress: 0 },
      { id: g2, name: '基础工程', duration: 0, level: 0, progress: 0 },
      { id: t.earth, name: '土方开挖', duration: 12, level: 1, progress: 0 },
      { id: t.cushion, name: '垫层浇筑', duration: 3, level: 1, progress: 0 },
      { id: t.rebar, name: '基础钢筋绑扎', duration: 8, level: 1, progress: 0 },
      { id: t.foundation, name: '基础混凝土浇筑', duration: 5, level: 1, progress: 0 },
      { id: g3, name: '主体结构', duration: 0, level: 0, progress: 0 },
      { id: t.main1, name: '主体结构一区', duration: 25, level: 1, progress: 0 },
      { id: t.main2, name: '主体结构二区', duration: 25, level: 1, progress: 0 },
      { id: g4, name: '机电与装修', duration: 0, level: 0, progress: 0 },
      { id: t.embed, name: '机电预埋', duration: 20, level: 1, progress: 0 },
      { id: t.masonry, name: '二次结构', duration: 18, level: 1, progress: 0 },
      { id: t.finish, name: '装饰装修', duration: 30, level: 1, progress: 0 },
      { id: t.mep, name: '机电安装', duration: 25, level: 1, progress: 0 },
      { id: g5, name: '竣工阶段', duration: 0, level: 0, progress: 0 },
      { id: t.accept, name: '竣工验收备案', duration: 5, level: 1, progress: 0 },
      { id: t.deliver, name: '交付使用', duration: 0, level: 1, progress: 0, isMilestone: true },
    ],
    links: [
      { from: t.site, to: t.permit, type: 'SS', lag: 5 },
      { from: t.site, to: t.drawing, type: 'FS', lag: 0 },
      { from: t.drawing, to: t.earth, type: 'FS', lag: 0 },
      { from: t.permit, to: t.earth, type: 'FS', lag: 0 },
      { from: t.earth, to: t.cushion, type: 'FS', lag: 0 },
      { from: t.cushion, to: t.rebar, type: 'FS', lag: 0 },
      { from: t.rebar, to: t.foundation, type: 'FS', lag: 0 },
      { from: t.foundation, to: t.main1, type: 'FS', lag: 0 },
      { from: t.main1, to: t.main2, type: 'SS', lag: 10 },
      { from: t.main1, to: t.main2, type: 'FF', lag: 5 },
      { from: t.main1, to: t.embed, type: 'SS', lag: 5 },
      { from: t.main2, to: t.masonry, type: 'FS', lag: 0 },
      { from: t.masonry, to: t.finish, type: 'FS', lag: 0 },
      { from: t.embed, to: t.mep, type: 'FS', lag: 0 },
      { from: t.finish, to: t.mep, type: 'SS', lag: 10 },
      { from: t.finish, to: t.accept, type: 'FS', lag: 0 },
      { from: t.mep, to: t.accept, type: 'FS', lag: 0 },
      { from: t.accept, to: t.deliver, type: 'FS', lag: 0 },
    ],
  }
}

export function makeEmptyProject(): SchedProject {
  return { name: '未命名工程', startDate: new Date().toISOString().slice(0, 10), tasks: [], links: [] }
}
