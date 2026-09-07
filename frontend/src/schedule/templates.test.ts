/**
 * templates.test.ts — 项目模板纯函数用例（v4.138 #10）
 *
 * 覆盖：listTemplates 空表/坏 JSON/非数组/坏条目逐条容错、saveTemplate
 * 追加与落盘往返、同名覆盖（原位更新不追加）、name trim（空名 no-op、
 * 带空格 trim 命中覆盖）、上限 10 FIFO 淘汰最旧、deleteTemplate 删除与
 * 不存在 no-op、getTemplate 命中/未命中、注入 storage 隔离（写入落到注入实现）。
 */
import { describe, expect, it } from 'vitest'
import {
  TEMPLATES_CAP, TEMPLATES_KEY, deleteTemplate, getTemplate, listTemplates, saveTemplate,
} from './templates'
import type { SchedProject } from './types'
import type { TemplateItem } from './templates'

/** 内存 storage（chatPrefs.test 同款），dump 供断言落盘内容 */
function fakeStorage(initial: Record<string, string> = {}): Storage & { dump: () => Record<string, string> } {
  const bag = { ...initial }
  return {
    getItem: (k: string) => (k in bag ? bag[k] : null),
    setItem: (k: string, v: string) => { bag[k] = v },
    removeItem: (k: string) => { delete bag[k] },
    clear: () => { for (const k of Object.keys(bag)) delete bag[k] },
    key: (i: number) => Object.keys(bag)[i] ?? null,
    get length() { return Object.keys(bag).length },
    dump: () => ({ ...bag }),
  } as Storage & { dump: () => Record<string, string> }
}

/** 最小工程样板：只带名字与一条任务（存模板只看整份快照往返，不看排程） */
function proj(name: string): SchedProject {
  return {
    name,
    startDate: '2026-09-07',
    tasks: [{ id: 'A', name: '挖土', duration: 3, level: 1, progress: 0 }],
    links: [],
  }
}

describe('listTemplates', () => {
  it('空存储/坏 JSON/非数组/全坏条目 → 回落空表', () => {
    expect(listTemplates(fakeStorage())).toEqual([])
    expect(listTemplates(fakeStorage({ [TEMPLATES_KEY]: 'not json' }))).toEqual([])
    expect(listTemplates(fakeStorage({ [TEMPLATES_KEY]: '{"name":"x"}' }))).toEqual([])
    expect(listTemplates(fakeStorage({ [TEMPLATES_KEY]: '["junk", null, 42]' }))).toEqual([])
  })

  it('坏条目逐条容错丢弃：缺名字/空名字/时间非串/project 缺 tasks 的条目丢弃，好条目保留', () => {
    const good = { name: '好模板', savedAt: '2026-09-07 08:00', project: { name: '工程P', startDate: '2026-09-07', tasks: [] } }
    const raw = JSON.stringify([
      good,
      { savedAt: 'x', project: { tasks: [] } }, // 缺名字
      { name: '', savedAt: 'x', project: { tasks: [] } }, // 空名字
      { name: '时间坏', savedAt: 5, project: { tasks: [] } }, // 时间非字符串
      { name: '缺project', savedAt: 'x' }, // 缺 project
      { name: 'project坏', savedAt: 'x', project: { name: 'P' } }, // project 缺 tasks 数组
      'junk',
    ])
    const list = listTemplates(fakeStorage({ [TEMPLATES_KEY]: raw }))
    expect(list).toHaveLength(1)
    expect(list[0].name).toBe('好模板')
    expect(list[0].project.tasks).toEqual([])
  })
})

describe('saveTemplate', () => {
  it('追加保存 + 读取往返一致（注入 storage 落盘到 gaea.schedule.templates）', () => {
    const s = fakeStorage()
    const list = saveTemplate('住宅楼模板', proj('工程A'), s, '2026-09-07 08:00')
    expect(list.map((t) => t.name)).toEqual(['住宅楼模板'])
    expect(list[0].savedAt).toBe('2026-09-07 08:00')
    expect(list[0].project.name).toBe('工程A')
    // 注入 storage 收到写入；再读往返一致
    expect(JSON.parse(s.dump()[TEMPLATES_KEY])).toHaveLength(1)
    expect(getTemplate('住宅楼模板', s)?.project.name).toBe('工程A')
    // 再存一个 → 追加不覆盖
    saveTemplate('厂房模板', proj('工程B'), s, '2026-09-07 09:00')
    expect(listTemplates(s).map((t) => t.name)).toEqual(['住宅楼模板', '厂房模板'])
  })

  it('同名覆盖 = 原位更新：长度不变、顺序不变、快照与保存时间更新；带空格名字 trim 后命中', () => {
    const s = fakeStorage()
    saveTemplate('住宅楼模板', proj('工程A'), s, '2026-09-07 08:00')
    saveTemplate('厂房模板', proj('工程B'), s, '2026-09-07 09:00')
    const list = saveTemplate(' 住宅楼模板 ', proj('工程A2'), s, '2026-09-07 10:00')
    expect(list.map((t) => t.name)).toEqual(['住宅楼模板', '厂房模板']) // 顺序不变、名字已 trim
    expect(list).toHaveLength(2)
    expect(getTemplate('住宅楼模板', s)?.project.name).toBe('工程A2')
    expect(getTemplate('住宅楼模板', s)?.savedAt).toBe('2026-09-07 10:00')
  })

  it('name trim 后为空 → 无害 no-op：原表原样返回、落盘不动', () => {
    const s = fakeStorage()
    saveTemplate('住宅楼模板', proj('工程A'), s, '2026-09-07 08:00')
    const list = saveTemplate('   ', proj('工程B'), s, '2026-09-07 09:00')
    expect(list.map((t) => t.name)).toEqual(['住宅楼模板'])
    // 落盘未动：仍是早前那一条，没有写入工程B
    expect(JSON.parse(s.dump()[TEMPLATES_KEY]).map((t: TemplateItem) => t.name)).toEqual(['住宅楼模板'])
    expect(listTemplates(s)).toHaveLength(1)
  })

  it(`上限 ${TEMPLATES_CAP} 个：超出 FIFO 淘汰最旧（头部先走）`, () => {
    const s = fakeStorage()
    for (let i = 1; i <= TEMPLATES_CAP + 1; i++) saveTemplate(`模板${i}`, proj(`工程${i}`), s, `2026-09-07 08:${String(i).padStart(2, '0')}`)
    const list = listTemplates(s)
    expect(list).toHaveLength(TEMPLATES_CAP)
    expect(list[0].name).toBe('模板2') // 最旧的 模板1 被淘汰
    expect(list[list.length - 1].name).toBe(`模板${TEMPLATES_CAP + 1}`)
  })
})

describe('deleteTemplate / getTemplate', () => {
  it('删除按名精确移除；名字不存在 = no-op；删空后列表为空', () => {
    const s = fakeStorage()
    saveTemplate('住宅楼模板', proj('工程A'), s, '2026-09-07 08:00')
    saveTemplate('厂房模板', proj('工程B'), s, '2026-09-07 09:00')
    expect(deleteTemplate('不存在的模板', s).map((t) => t.name)).toEqual(['住宅楼模板', '厂房模板'])
    expect(deleteTemplate('住宅楼模板', s).map((t) => t.name)).toEqual(['厂房模板'])
    expect(getTemplate('住宅楼模板', s)).toBeNull()
    deleteTemplate('厂房模板', s)
    expect(listTemplates(s)).toEqual([])
  })

  it('getTemplate 命中返回整条（含工程快照），未命中返回 null', () => {
    const s = fakeStorage()
    expect(getTemplate('啥都没有', s)).toBeNull()
    saveTemplate('住宅楼模板', proj('工程A'), s, '2026-09-07 08:00')
    const hit = getTemplate('住宅楼模板', s)
    expect(hit).not.toBeNull()
    expect(hit!.name).toBe('住宅楼模板')
    expect(hit!.savedAt).toBe('2026-09-07 08:00')
    expect(hit!.project.tasks[0].id).toBe('A')
  })
})
