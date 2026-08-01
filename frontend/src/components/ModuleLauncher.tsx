import React from 'react'
import {
  MessageOutlined, ReadOutlined, PictureOutlined, HeartOutlined,
  FileTextOutlined, ToolOutlined, ApiOutlined, SettingOutlined,
  ThunderboltOutlined, ArrowRightOutlined,
} from '@ant-design/icons'
import { Typography } from 'antd'

/** 启动器可跳转的目标页（与 MainLayout 的 Page 类型保持一致的子集） */
export type LauncherTarget =
  | 'chat' | 'novel' | 'imagegen' | 'whisper' | 'office' | 'gaea' | 'modelcenter' | 'settings'

interface LauncherModule {
  key: LauncherTarget
  name: string
  desc: string
  icon: React.ReactNode
  accent: string
}

// 每张卡片独立主题色（中亮度色，亮/暗主题下均保证对比度）
const modules: LauncherModule[] = [
  { key: 'chat', name: '聊天', desc: '与 AI 对话，激发灵感', icon: <MessageOutlined />, accent: '#60a5fa' },
  { key: 'novel', name: '小说', desc: '世界观、角色与大纲创作', icon: <ReadOutlined />, accent: '#a78bfa' },
  { key: 'imagegen', name: '绘梦', desc: 'AI 图像生成工作台', icon: <PictureOutlined />, accent: '#f472b6' },
  { key: 'whisper', name: '轻语', desc: '陪伴式 AI 心灵对话', icon: <HeartOutlined />, accent: '#fb7185' },
  { key: 'office', name: '方案编写', desc: '投标方案六标签页编写', icon: <FileTextOutlined />, accent: '#f59e0b' },
  { key: 'gaea', name: '办公', desc: 'gaea 办公套件与文档', icon: <ToolOutlined />, accent: '#2dd4bf' },
  { key: 'modelcenter', name: '模型中心', desc: '模型引擎管理与配置', icon: <ApiOutlined />, accent: '#34d399' },
  { key: 'settings', name: '设置', desc: '应用偏好与主题外观', icon: <SettingOutlined />, accent: '#94a3b8' },
]

interface ModuleLauncherProps {
  onNavigate: (target: LauncherTarget) => void
  /** 当前激活的 AI 模型名（顶栏已加载，传入提升真实感） */
  activeModel?: string
}

/** 单张霓虹玻璃卡片 */
const LauncherCard: React.FC<{ m: LauncherModule; idx: number; onOpen: () => void }> = ({ m, idx, onOpen }) => (
  <div
    role="button"
    tabIndex={0}
    aria-label={`进入${m.name}模块`}
    className="neon-card launcher-card"
    onClick={onOpen}
    onKeyDown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onOpen()
      }
    }}
    style={{
      animationDelay: `${0.08 + idx * 0.05}s`,
      background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
      WebkitBackdropFilter: 'blur(14px) saturate(130%)',
      backdropFilter: 'blur(14px) saturate(130%)',
      border: '1px solid var(--md-sys-color-outline-variant)',
    }}
  >
    <div style={{
      padding: 20,
      display: 'flex', flexDirection: 'column', gap: 12,
      height: '100%',
    }}>
      <div style={{
        width: 44, height: 44, borderRadius: 'var(--md-sys-radius-lg)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 22, color: m.accent,
        background: `color-mix(in srgb, ${m.accent} 14%, transparent)`,
        boxShadow: `0 0 16px color-mix(in srgb, ${m.accent} 35%, transparent)`,
        transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal)',
      }} className="launcher-icon">
        {m.icon}
      </div>
      <div>
        <div style={{
          fontSize: 15, fontWeight: 600,
          color: 'var(--md-sys-color-text)', marginBottom: 2,
          display: 'flex', alignItems: 'center', gap: 6,
        }}>
          {m.name}
          <ArrowRightOutlined
            style={{ fontSize: 10, opacity: 0, transform: 'translateX(-4px)', color: m.accent,
              transition: 'opacity 0.2s, transform 0.2s' }}
            className="launcher-arrow"
          />
        </div>
        <div style={{
          fontSize: 12, lineHeight: 1.5,
          color: 'var(--md-sys-color-text-secondary)',
        }}>
          {m.desc}
        </div>
      </div>
    </div>
  </div>
)

/**
 * ModuleLauncher — AI 中枢首页（未来感）。
 * 顶部 AI 状态条 + 霓虹玻璃卡片墙；单击卡片切换至对应模块。
 * 使用全局未来感令牌：.neon-card 渐变描边、.launcher-card 入场、
 * .live-dot 发光状态点、.md-glass 玻璃拟态。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel }) => (
  <div style={{
    flex: 1,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    padding: '20px 24px 32px',
    position: 'relative', zIndex: 1,
  }}>
    <div style={{
      display: 'flex', flexDirection: 'column', gap: 18,
      width: '100%', maxWidth: 920,
    }}>
      {/* ═══ AI 中枢状态条 ═══ */}
      <div
        className="md-glass scanline-top"
        style={{
          position: 'relative',
          display: 'flex', alignItems: 'center', gap: 14,
          padding: '12px 18px', borderRadius: 'var(--md-sys-radius-lg)',
          animation: 'launcherFadeUp 0.4s cubic-bezier(0.16, 1, 0.3, 1) backwards',
        }}
      >
        <ThunderboltOutlined className="neon-glow-icon" style={{ fontSize: 18, color: 'var(--gaea-glow)' }} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span className="live-dot" />
            <Typography.Text strong style={{ fontSize: 14, color: 'var(--md-sys-color-text)' }}>
              AI 中枢在线
            </Typography.Text>
            <span style={{
              fontSize: 11, padding: '1px 8px', borderRadius: 10,
              background: 'var(--md-sys-color-primary-container)',
              color: 'var(--md-sys-color-on-primary-container)',
              fontWeight: 500, letterSpacing: '0.04em',
            }}>
              GAEA CORE
            </span>
            {activeModel && (
              <span style={{
                fontSize: 11, padding: '1px 8px', borderRadius: 10,
                background: 'color-mix(in srgb, var(--gaea-glow) 14%, transparent)',
                color: 'var(--gaea-glow)',
                fontWeight: 500,
                border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
                maxWidth: 140, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
              }}>
                {activeModel}
              </span>
            )}
          </div>
          <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
            选择模块开始创作 —— 灵感已就绪，随时待命
          </Typography.Text>
        </div>
        <span
          role="button" tabIndex={0}
          onClick={() => onNavigate('chat')}
          onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onNavigate('chat') } }}
          style={{
            display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer',
            fontSize: 13, fontWeight: 500, whiteSpace: 'nowrap',
            color: 'var(--gaea-glow)',
            padding: '6px 14px', borderRadius: 'var(--md-sys-radius-md)',
            border: '1px solid var(--md-sys-color-outline-variant)',
            background: 'var(--md-sys-color-primary-container)',
            boxShadow: '0 0 14px color-mix(in srgb, var(--gaea-glow) 22%, transparent)',
            transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal)',
          }}
        >
          立即对话 <ArrowRightOutlined style={{ fontSize: 11 }} />
        </span>
      </div>

      {/* ═══ 模块卡片墙 ═══ */}
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(196px, 1fr))',
        gap: 16,
      }}>
        {modules.map((m, idx) => (
          <LauncherCard key={m.key} m={m} idx={idx} onOpen={() => onNavigate(m.key)} />
        ))}
      </div>
    </div>
  </div>
)

export default ModuleLauncher
