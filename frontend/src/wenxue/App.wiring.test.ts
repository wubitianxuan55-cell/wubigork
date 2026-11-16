import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * App.wiring.test.ts — 图文 App 装配面源级锁（App 巨组件不整页渲染，
 * gaea App.palette.test.ts 同先例；2026-11-17 审查批15 补 App 级自动化层）。
 * 锁的都是真机出过事故或审查定罪的接线：
 *  - tab 持久化校验单源消费（白名单漏名=tab 被当子代理 tab 吃掉，三犯）
 *  - 板挂幂等 EngineInit（选书不开会话四 tab 空白实弹修 965d1ada）
 *  - 办公域命令拒绝守卫（二次改造守卫本体）
 *  - workspace-changed 广播三处（三 Hub/树/Composer 刷新链）
 *  - 任务模板单源=图文九条（批2 刀1：OfficeB 远端不再消费）
 */
const src = readFileSync(join(process.cwd(), 'src/wenxue/App.tsx'), 'utf8')

describe('图文 App 装配面（源级锁）', () => {
  it('tab 持久化校验走 CHAT_VIEW_TAB_IDS 单源（漏名=被当子代理 tab 吃掉）', () => {
    expect(src).toContain('(CHAT_VIEW_TAB_IDS as string[]).includes(saved)')
    expect(src).toMatch(/import \{[^}]*CHAT_VIEW_TAB_IDS[^}]*\} from "\.\/components\/ChatTabs"/)
  })

  it('板挂幂等 EngineInit 认领（965d1ada 实弹修：四 tab 与 Meta.ready 依赖它）', () => {
    expect(src).toMatch(/void app\.EngineInit\(\)\.catch\(\(\) => \{\}\)/)
    expect(src).toContain('图文引擎板挂即初始化（幂等）')
  })

  it('办公域命令拒绝守卫在位（文学不提供知识库/CAD/进度命令）', () => {
    expect(src).toContain('文学板块不提供知识库、CAD 或进度计划命令')
  })

  it('workspace-changed 广播恰三处（新建/选中/删除小说；多广播=Hub 重复重载）', () => {
    const n = (src.match(/dispatchEvent\(new CustomEvent\("wenxue:workspace-changed"/g) ?? []).length
    expect(n).toBe(3)
  })

  it('任务模板单源=图文九条（批2 刀1：不走 OfficeB 远端）', () => {
    expect(src).toContain('loadTemplates')
    expect(src).toContain('FALLBACK_TEMPLATES')
    expect(src).not.toContain('app.TaskTemplates()')
  })
})
