/**
 * chatPrefs.test.ts — 板块工作台偏好纯函数用例（刀11；刀C 增横道双栏字段；
 * v4.138 #14 增自定义列缺省收起；W5 增列筛持久化 ganttColFilters）
 */
import { describe, expect, it } from 'vitest'
import { CHAT_PREFS_KEY, clampChatWidth, loadChatPrefs, saveChatPrefs } from './chatPrefs'
import { GANTT_LEFT_W_FULL, GANTT_TABLE_W_MIN } from './ganttCols'
import {
  BUILTIN_GANTT_VIEWS,
  CUSTOM_GANTT_VIEW_ID,
  DEFAULT_GANTT_VIEW_ID,
  hideForBuiltinView,
  widthForBuiltinView,
} from './viewPresets'

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

const PLAN = BUILTIN_GANTT_VIEWS.find((v) => v.id === DEFAULT_GANTT_VIEW_ID)!
/** W1 缺省=计划编制：只露身份/分区/工期/前置/计划日期，表格宽=可见列宽。 */
const DEFAULT_HIDE = hideForBuiltinView(PLAN)
const DEFAULTS = {
  collapsed: true,
  width: 420,
  ganttTableW: widthForBuiltinView(PLAN),
  ganttHide: DEFAULT_HIDE,
  ganttSort: { field: 'none', dir: 'asc' },
  ganttGroup: [],
  ganttViewId: DEFAULT_GANTT_VIEW_ID,
  ganttFilterKind: 'all',
  ganttDisplayPreset: 'normal',
  ganttColFilters: {},
}

describe('clampChatWidth', () => {
  it('钳位 320~680，非有限值回落 420', () => {
    expect(clampChatWidth(200)).toBe(320)
    expect(clampChatWidth(1000)).toBe(680)
    expect(clampChatWidth(500)).toBe(500)
    expect(clampChatWidth(NaN)).toBe(420)
  })
})

describe('loadChatPrefs', () => {
  it('无存储/空值回落缺省', () => {
    expect(loadChatPrefs(fakeStorage())).toEqual(DEFAULTS)
    expect(loadChatPrefs(fakeStorage({ [CHAT_PREFS_KEY]: 'not json' }))).toEqual(DEFAULTS)
  })
  it('字段级容错：坏 width 不拖累 collapsed', () => {
    const s = fakeStorage({ [CHAT_PREFS_KEY]: '{"collapsed":true,"width":"fat"}' })
    expect(loadChatPrefs(s)).toEqual({ ...DEFAULTS, collapsed: true })
  })
  it('刀C：ganttTableW 钳位 [最小宽,全列宽]、ganttHide 过滤非法键（未知键/固定列丢弃）', () => {
    const s = fakeStorage({ [CHAT_PREFS_KEY]: JSON.stringify({ ganttTableW: 99999, ganttHide: ['ls', 'no', 'ghost', 'lf'] }) })
    expect(loadChatPrefs(s).ganttTableW).toBe(GANTT_LEFT_W_FULL)
    expect(loadChatPrefs(s).ganttHide).toEqual(['ls', 'lf'])
    const s2 = fakeStorage({ [CHAT_PREFS_KEY]: '{"ganttTableW":"fat"}' })
    expect(loadChatPrefs(s2).ganttTableW).toBe(DEFAULTS.ganttTableW)
    const s3 = fakeStorage({ [CHAT_PREFS_KEY]: JSON.stringify({ ganttTableW: 5 }) })
    expect(loadChatPrefs(s3).ganttTableW).toBe(GANTT_TABLE_W_MIN)
  })
  it('W1：新建偏好=计划编制；存量 ganttHide（无 viewId）进入自定义视图并原样保留', () => {
    expect(DEFAULT_HIDE).toContain('progress')
    expect(DEFAULT_HIDE).toContain('mode')
    // 存量数据已存 ganttHide 数组时逐项容错保留，不把用户强行迁到新预设
    const s = fakeStorage({ [CHAT_PREFS_KEY]: JSON.stringify({ ganttHide: ['progress'] }) })
    expect(loadChatPrefs(s).ganttHide).toEqual(['progress'])
    expect(loadChatPrefs(s).ganttViewId).toBe(CUSTOM_GANTT_VIEW_ID)
    // 存量数据无 ganttHide 字段 → 回落计划编制
    const s2 = fakeStorage({ [CHAT_PREFS_KEY]: '{"width":420}' })
    expect(loadChatPrefs(s2).ganttHide).toEqual(DEFAULT_HIDE)
    expect(loadChatPrefs(s2).ganttViewId).toBe(DEFAULT_GANTT_VIEW_ID)
  })
})

describe('saveChatPrefs', () => {
  it('部分字段合并落盘，读取往返一致', () => {
    const s = fakeStorage()
    saveChatPrefs({ collapsed: true }, s)
    expect(loadChatPrefs(s)).toEqual({ ...DEFAULTS, collapsed: true })
    saveChatPrefs({ width: 555 }, s)
    expect(loadChatPrefs(s)).toEqual({ ...DEFAULTS, collapsed: true, width: 555 })
  })
  it('落盘前钳位', () => {
    const s = fakeStorage()
    saveChatPrefs({ width: 99 }, s)
    expect(JSON.parse(s.dump()[CHAT_PREFS_KEY]).width).toBe(320)
  })
  it('刀C：横道表格宽度/显隐键往返', () => {
    const s = fakeStorage()
    saveChatPrefs({ ganttTableW: 420, ganttHide: ['ls', 'lf', 'cost'] }, s)
    expect(loadChatPrefs(s).ganttTableW).toBe(420)
    expect(loadChatPrefs(s).ganttHide).toEqual(['ls', 'lf', 'cost'])
    // 只改宽度不丢显隐
    saveChatPrefs({ ganttTableW: 300 }, s)
    expect(loadChatPrefs(s).ganttHide).toEqual(['ls', 'lf', 'cost'])
    expect(loadChatPrefs(s).ganttTableW).toBe(300)
  })
  it('W1：当前视图/筛选档往返；手动改设置保存 custom 不被回写为内置', () => {
    const s = fakeStorage()
    saveChatPrefs({
      ganttViewId: 'control',
      ganttFilterKind: 'critical',
      ganttGroup: ['status', 'owner'],
    }, s)
    expect(loadChatPrefs(s)).toMatchObject({
      ganttViewId: 'control',
      ganttFilterKind: 'critical',
      ganttGroup: ['status', 'owner'],
    })
    saveChatPrefs({ ganttViewId: CUSTOM_GANTT_VIEW_ID, ganttHide: ['progress', 'cost'] }, s)
    expect(loadChatPrefs(s).ganttViewId).toBe(CUSTOM_GANTT_VIEW_ID)
    expect(loadChatPrefs(s).ganttHide).toEqual(['progress', 'cost'])
  })
})

describe('ganttColFilters（W5 列筛本机偏好）', () => {
  it('缺省=`{}`；存量 JSON 无该键 → `{}`，其余字段照旧', () => {
    expect(loadChatPrefs(fakeStorage()).ganttColFilters).toEqual({})
    const s = fakeStorage({
      [CHAT_PREFS_KEY]: '{"collapsed":false,"width":500,"ganttHide":["ls"]}',
    })
    const got = loadChatPrefs(s)
    expect(got.ganttColFilters).toEqual({})
    expect(got.collapsed).toBe(false)
    expect(got.width).toBe(500)
    expect(got.ganttHide).toEqual(['ls'])
    expect(got.ganttViewId).toBe(CUSTOM_GANTT_VIEW_ID) // 存量 ganttHide 仍是自定义视图
  })

  it('坏值回落 `{}`：标量 / 未知列键 / 类型冲突 / 空条件都不留下', () => {
    const scalar = fakeStorage({ [CHAT_PREFS_KEY]: '{"ganttColFilters":"broken"}' })
    expect(loadChatPrefs(scalar).ganttColFilters).toEqual({})
    const junk = fakeStorage({
      [CHAT_PREFS_KEY]: JSON.stringify({
        ganttColFilters: {
          ghost: { kind: 'text', text: 'x' }, // 未登记列键
          name: { kind: 'number', num: 3 }, // kind 与登记类型冲突
          dur: { kind: 'number', num: NaN }, // 非有限阈值
          start: { kind: 'date', from: '2026/01/01' }, // 非法日期
          mode: { kind: 'enum', set: [] }, // 空集
        },
      }),
    })
    expect(loadChatPrefs(junk).ganttColFilters).toEqual({})
  })

  it('写入后读回一致（五类条件共存）', () => {
    const s = fakeStorage()
    const filters = {
      name: { kind: 'text' as const, text: '土方' },
      dur: { kind: 'number' as const, op: 'le' as const, num: 5 },
      start: { kind: 'date' as const, from: '2026-01-01', to: '2026-03-31' },
      mode: { kind: 'enum' as const, set: ['manual'] },
      progress: { kind: 'number' as const, op: 'ge' as const, num: 0, empty: 'nonEmpty' as const },
    }
    saveChatPrefs({ ganttColFilters: filters }, s)
    expect(loadChatPrefs(s).ganttColFilters).toEqual(filters)
    expect(JSON.parse(s.dump()[CHAT_PREFS_KEY]).ganttColFilters.dur).toEqual({
      kind: 'number',
      op: 'le',
      num: 5,
    })
  })

  it('落盘剔除空条件：空对象就写 `{}`，不留垃圾键', () => {
    const s = fakeStorage()
    saveChatPrefs(
      {
        ganttColFilters: {
          name: { kind: 'text', text: '   ' },
          dur: { kind: 'number', op: 'ge', num: NaN },
        },
      },
      s,
    )
    expect(JSON.parse(s.dump()[CHAT_PREFS_KEY]).ganttColFilters).toEqual({})
    expect(loadChatPrefs(s).ganttColFilters).toEqual({})
  })

  it('字段级隔离：坏列筛不拖累 ganttHide / ganttViewId', () => {
    const s = fakeStorage({
      [CHAT_PREFS_KEY]: JSON.stringify({
        ganttHide: ['progress'],
        ganttViewId: 'control',
        ganttColFilters: 'broken',
      }),
    })
    const got = loadChatPrefs(s)
    expect(got.ganttColFilters).toEqual({})
    expect(got.ganttHide).toEqual(['progress'])
    expect(got.ganttViewId).toBe('control')
  })

  it('部分保存不覆盖已有列筛（与其它字段的合并语义一致）', () => {
    const s = fakeStorage()
    saveChatPrefs({ ganttColFilters: { dur: { kind: 'number', op: 'eq', num: 2 } } }, s)
    saveChatPrefs({ width: 480 }, s)
    expect(loadChatPrefs(s).ganttColFilters).toEqual({ dur: { kind: 'number', op: 'eq', num: 2 } })
    // 显式传空对象=清空列筛（清空是用户动作，不是缺省回落）
    saveChatPrefs({ ganttColFilters: {} }, s)
    expect(loadChatPrefs(s).ganttColFilters).toEqual({})
  })
})

describe('显示预设（W6 护眼/黑白）偏好', () => {
  it('缺省 normal；写入后读回；脏值回落 normal', () => {
    const s = fakeStorage()
    expect(loadChatPrefs(s).ganttDisplayPreset).toBe('normal')
    saveChatPrefs({ ganttDisplayPreset: 'eyeCare' }, s)
    expect(loadChatPrefs(s).ganttDisplayPreset).toBe('eyeCare')
    saveChatPrefs({ ganttDisplayPreset: 'rainbow' as never }, s)
    expect(loadChatPrefs(s).ganttDisplayPreset).toBe('normal')
  })

  it('旧 JSON 无该键 → normal，且不拖累其它字段', () => {
    const s = fakeStorage()
    s.setItem(CHAT_PREFS_KEY, JSON.stringify({ collapsed: false, width: 500, ganttHide: ['dur'] }))
    const p = loadChatPrefs(s)
    expect(p.ganttDisplayPreset).toBe('normal')
    expect(p.width).toBe(500)
    expect(p.ganttHide).toContain('dur')
  })
})
