import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

// 前端 CI 门禁约定（2026-08-14 长期规划 E1-1）：
// - 硬错误（rules-of-hooks 等会掩盖真实缺陷的规则）必须清零；
// - 存量风格问题（no-explicit-any / no-empty / no-unused-vars / ban-ts-comment /
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
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': 'warn',
      'no-empty': 'warn',
      '@typescript-eslint/ban-ts-comment': 'warn',
      'react-refresh/only-export-components': 'warn',
      '@typescript-eslint/no-namespace': 'warn',
    },
  },
])
