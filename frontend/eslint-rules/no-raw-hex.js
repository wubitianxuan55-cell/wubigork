// no-raw-hex.js — 自定义 eslint 规则：禁组件内 raw hex 色值
// （审计「151 hex token 化」+ MASTER.md「组件零硬编码色值」，图表豁免）。
//
// 豁免两种：
//  1. 行内 `// hex-exempt` 注释（图表调色板 / 品牌识别色 / 模板数据色）；
//  2. 文件级豁免 exemptFiles（按 basename 匹配，图表/数据文件整文件放行）。
// 其余 raw hex（#rgb/#rrggbb/#rrggbbaa）一律报错：改用主题 token
// （var(--md-sys-color-*) / var(--color-*) / var(--gaea-*) 或 C() 辅助）。
const HEX = /^#[0-9a-fA-F]{3,8}$/

export default {
  meta: {
    type: 'suggestion',
    docs: { description: 'Forbid raw hex colors in component code (use theme tokens)' },
    schema: [
      {
        type: 'object',
        properties: {
          exemptFiles: { type: 'array', items: { type: 'string' } },
        },
        additionalProperties: false,
      },
    ],
  },
  create(context) {
    const options = context.options[0] || {}
    const exemptFiles = options.exemptFiles || []
    const filename = context.filename || ''
    const basename = filename.split(/[\\/]/).pop() || ''
    if (exemptFiles.includes(basename)) return {}
    return {
      Literal(node) {
        if (typeof node.value !== 'string') return
        if (!HEX.test(node.value.trim())) return
        const line = context.getSourceCode().lines[node.loc.start.line - 1] || ''
        if (line.includes('hex-exempt')) return
        context.report({
          node,
          message: 'raw hex "{{v}}"：改用主题 token（var(--md-sys-color-*) / var(--color-*) / C()），图表/品牌色可加 // hex-exempt',
          data: { v: node.value },
        })
      },
    }
  },
}
