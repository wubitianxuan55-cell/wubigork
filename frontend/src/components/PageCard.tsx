import React from 'react'
import { Card } from 'antd'
import type { CardProps } from 'antd'

/**
 * PageCard — 统一页面卡片，自动应用品牌 CSS 变量
 */
export const PageCard: React.FC<CardProps> = ({ style, ...rest }) => (
  <Card
    style={{
      background: 'var(--color-bg-container)',
      borderColor: 'var(--color-border)',
      ...style,
    }}
    {...rest}
  />
)
