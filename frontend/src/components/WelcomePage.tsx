import React from 'react'
import { Typography, Button } from 'antd'
import { C } from '../utils/theme'

interface WelcomePageProps {
  onLogin: () => void
}

/** WelcomePage — 未登录品牌的欢迎页 */
const WelcomePage: React.FC<WelcomePageProps> = ({ onLogin }) => (
  <div style={{
    display: 'flex', flexDirection: 'column', justifyContent: 'center',
    alignItems: 'center', height: '75vh', gap: 32,
  }}>
    <div style={{ textAlign: 'center' }}>
      <img src="/favicon.svg" alt="gaea" style={{ width: 80, height: 80, marginBottom: 20 }} />
      <Typography.Title level={1} style={{
        color: C('color-text'), margin: '0 0 4px', fontSize: 36,
        fontWeight: 700, letterSpacing: '-0.5px',
      }}>
        gaea
      </Typography.Title>
      <Typography.Text style={{
        color: C('color-primary'), fontSize: 18, fontWeight: 400, letterSpacing: 2,
      }}>
        让灵感成为故事
      </Typography.Text>
    </div>

    <Typography.Paragraph style={{
      color: C('color-text-secondary'), fontSize: 15, textAlign: 'center',
      maxWidth: 400, lineHeight: 1.8, margin: 0,
    }}>
      基于 AI 的桌面端小说创作平台<br />
      导入灵感，AI 为你构建世界观、角色与大纲
    </Typography.Paragraph>

    <Button
      type="primary" size="large" onClick={onLogin}
      style={{
        background: C('color-primary'), borderColor: C('color-primary'),
        padding: '8px 48px', height: 44, fontSize: 16, borderRadius: 8, marginTop: 8,
      }}
    >
      登录 xAI，开始创作
    </Button>

    <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
      需要 xAI 账号 · 安全登录，OAuth 授权
    </Typography.Text>
  </div>
)

export default WelcomePage
