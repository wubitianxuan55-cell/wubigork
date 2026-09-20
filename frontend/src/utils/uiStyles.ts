// uiStyles.ts — 跨面板复用的内联文本样式常量。
// v4.361 令牌卫生：softTextStyle 此前在 7 个文件逐字重复定义（改一处必漏六处），
// 收敛到单一来源；有差异的变体（如 labelTextStyle 各文件 marginBottom 不同）
// 属有意差异，不在此收敛。
import type React from 'react'

/** 12px 次级说明文字（v3-fg-soft） */
export const softTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)' }
