// remarkDisableCodeIndented — LLM 输出防御刀（C2，机制对照 deepseek-ai/chat-demo
// parse.ts 的 `{ ...gfm(), disable: { null: ['codeIndented'] } }`，MIT）。
// CommonMark 的四空格缩进代码块语法会把 LLM 常见输出的「前导空格列表」误判为
// 代码块（缩进 4 格以上的 - 列表整段变成 <pre>）；禁用 codeIndented 后按普通
// 列表/段落渲染。代价：真正想用缩进表示代码的输入不再成码块（聊天场景 LLM 一律
// 用围栏 ```，可接受——chat-demo 同款取舍）。
// 机制：micromark 允许任一扩展对象携带 disable 清单（挂 null = 全字符码生效），
// remark-parse 解析器会合并 data.micromarkExtensions——与 chat-demo 同款注入位。
// 只挂聊天线 ChatMarkdown（办公/文件预览渲染链不动，控制爆炸半径）。
import type { Pluggable } from 'unified'

function remarkDisableCodeIndented(this: { data(): Record<string, unknown[]> }) {
  const data = this.data()
  ;(data.micromarkExtensions ??= []).push({ disable: { null: ['codeIndented'] } })
}

export default remarkDisableCodeIndented as unknown as Pluggable
