import React from 'react'
import { Typography, Button } from 'antd'
import { ThunderboltOutlined, ArrowRightOutlined } from '@ant-design/icons'

interface WelcomePageProps {
  onLogin: () => void
}

/** WelcomePage — 未来感品牌欢迎页（深空星云 + 玻璃卡片 + 霓虹） */
const WelcomePage: React.FC<WelcomePageProps> = ({ onLogin }) => (
  <div style={{
    position: 'relative', zIndex: 1,
    display: 'flex', flexDirection: 'column', justifyContent: 'center',
    alignItems: 'center', height: '100%', minHeight: '70vh', gap: 0,
  }}>
    {/* 玻璃品牌卡 */}
    <div className="md-glass" style={{
      position: 'relative',
      display: 'flex', flexDirection: 'column', alignItems: 'center',
      padding: '48px 64px 40px', borderRadius: 'var(--md-sys-radius-xl)',
      boxShadow: '0 0 0 1px var(--gaea-glow), 0 24px 80px rgba(0,0,0,0.35)',
      maxWidth: 440, width: 'calc(100% - 48px)',
    }}>
      <div style={{ position: 'relative', marginBottom: 24 }}>
        <img src="/favicon.svg" alt="gaea" style={{
          width: 88, height: 88,
          filter: 'drop-shadow(0 0 18px var(--gaea-glow)) drop-shadow(0 0 48px var(--gaea-glow))',
        }} />
        <span className="live-dot" style={{ position: 'absolute', top: 6, right: 2 }} />
      </div>
      <Typography.Title level={1} className="neon-glow-text" style={{
        color: 'var(--md-sys-color-text)', margin: '0 0 6px', fontSize: 40,
        fontWeight: 700, letterSpacing: '-0.5px',
      }}>
        gaea
      </Typography.Title>
      <Typography.Text style={{
        color: 'var(--gaea-glow)', fontSize: 17, fontWeight: 500, letterSpacing: 3,
        textShadow: '0 0 14px var(--gaea-glow)',
        marginBottom: 28,
      }}>
        让灵感成为故事
      </Typography.Text>

      <Typography.Paragraph style={{
        color: 'var(--md-sys-color-text-secondary)', fontSize: 14, textAlign: 'center',
        maxWidth: 360, lineHeight: 1.9, margin: '0 0 32px',
      }}>
        基于 AI 的多功能创作平台<br />
        导入灵感，AI 为你构建世界观、角色与大纲
      </Typography.Paragraph>

      <Button
        type="primary" size="large" onClick={onLogin}
        style={{
          background: 'var(--md-sys-color-primary)', borderColor: 'var(--md-sys-color-primary)',
          padding: '8px 40px', height: 46, fontSize: 15, borderRadius: 12,
          boxShadow: '0 0 22px color-mix(in srgb, var(--gaea-glow) 45%, transparent)',
          display: 'flex', alignItems: 'center', gap: 8,
          transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal)',
        }}
        className="welcome-login-btn"
      >
        <ThunderboltOutlined /> 登录 xAI，开始创作 <ArrowRightOutlined style={{ fontSize: 12 }} />
      </Button>

      <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 11, marginTop: 18 }}>
        需要 xAI 账号 · 安全登录，OAuth 授权
      </Typography.Text>
    </div>
  </div>
)

export default WelcomePage
