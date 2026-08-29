import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'
import localRules from './eslint-rules/index.js'

// 前端 CI 门禁约定（2026-08-14 长期规划 E1-1）：
// - 硬错误（rules-of-hooks 等会掩盖真实缺陷的规则）必须清零；
// - 存量风格问题（no-empty / no-unused-vars / ban-ts-comment /
//   react-refresh / no-namespace）降为 warn，随迭代逐步清理，不阻塞门禁。
export default defineConfig([
  globalIgnores([
    'dist',
    'node_modules',
    'wailsjs', // Wails 生成代码，不进 lint
    '.eslint-out.json',
  ]),
  {
    files: ['**/*.{ts,tsx}'],
    plugins: {
      local: localRules,
    },
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs['recommended-latest'],
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      globals: globals.browser,
    },
    rules: {
      // S2.2 151 hex token 化（审计）：组件内 raw hex 报错；图表/数据文件
      // 整文件豁免（basename 名单），品牌识别色/图表调色板行内 // hex-exempt。
      'local/no-raw-hex': ['error', {
        exemptFiles: [
          // 图表/图谱（审计「图表豁免」）：调色板是数据不是 UI chrome
          'RelationGraph.tsx',
          'EmotionStarMap.tsx',
          'TisorRadar.tsx',
          'WhisperTracePanel.tsx',
          'WhisperEmotionPanel.tsx',
          'AgentNetworkCard.tsx',
          'GraphView.tsx',
          'domainColors.ts',
          // 主题令牌源（12 套主题色板即 token 定义本身）+ 其测试夹具
          'appStore.ts',
          'appStore.test.ts',
          'domainColors.test.ts',
          // 模板/常量数据（herdsman 提示词模板、绘梦模板、聊天人格色）
          'herdsmanTemplates.ts',
          'imageTemplates.ts',
          'constants.ts',
          // 图表工具：mermaid 渲染配色
          'mermaidPng.ts',
        ],
      }],
      // T6-10.2（v2.33.0）：any 已全仓清零，升为硬错误进 CI 门禁——
      // 新增任何显式 any 都会让 lint 失败。
      '@typescript-eslint/no-explicit-any': 'error',
      // 质量收敛（v3.3.0）：`^_` 前缀参数/变量/catch 参数 = 显式声明「故意不用」，
      // 属社区标准约定，配置放行；真正的死代码（无 `_` 前缀）仍报警告待清理。
      '@typescript-eslint/no-unused-vars': ['warn', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrorsIgnorePattern: '^_',
      }],
      // 空 catch 块（localStorage/绑定降级吞错）为有意为之，放行；空 if/for/函数体仍报警告。
      'no-empty': ['warn', { allowEmptyCatch: true }],
      '@typescript-eslint/ban-ts-comment': 'warn',
      // react-refresh：放行纯常量导出（Vite 官方模板默认项）——组件文件内
      // const 常量（如 QUICK_REPLIES / FALLBACK_TEMPLATES）不影响 Fast Refresh。
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
      '@typescript-eslint/no-namespace': 'warn',
    },
  },
])
