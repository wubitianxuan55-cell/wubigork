// chatBoardAnchor.test.ts — 轻语浮层列锚守卫（v4.511 源级断言）
//
// Why：.chat-composer-wrap（输入岛，absolute left:0 right:0 bottom:0）与
// .chat-scroll-bottom（回到底部，absolute left:50%）都挂在 .chat-main 内。
// .chat-main 一旦失去 position:relative，包含块会一路锚到整板 .chat-board
// （含左右两栏）——默认双栏展开只偏 ~20px 不显眼，单侧折叠组合下输入岛与
// 消息列横向错位达 ±114px（侧栏 52/inspector 280 或 240/40）。此缺陷自
// v3.0.0 存在，CSS 特异度/渲染测试均不可见，只有列锚本身是可守卫的不变量
// （同族：v4.510 办公板 Transcript 浮层锚根修）。
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const read = (...p: string[]) => readFileSync(resolve(__dirname, ...p), 'utf8')

describe('轻语浮层列锚（v4.511）', () => {
  const css = read('chat-board.css')

  it('.chat-main 持有 position:relative——输入岛/回底/粒子/声波的包含块锚定消息列', () => {
    const block = css.slice(css.indexOf('.chat-main {'), css.indexOf('}', css.indexOf('.chat-main {')))
    expect(block).toContain('position: relative')
  })

  it('浮层仍声明 absolute（它们依赖列锚存在，重构时守卫提醒锚语义）', () => {
    const composer = css.slice(css.indexOf('.chat-composer-wrap {'), css.indexOf('}', css.indexOf('.chat-composer-wrap {')))
    expect(composer).toContain('position: absolute')
    const scrollBottom = css.slice(css.indexOf('.chat-scroll-bottom {'), css.indexOf('}', css.indexOf('.chat-scroll-bottom {')))
    expect(scrollBottom).toContain('position: absolute')
  })
})
