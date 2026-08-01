import React from 'react'
import {
  MessageOutlined, ReadOutlined, PictureOutlined, HeartOutlined,
  FileTextOutlined, ToolOutlined, ApiOutlined, SettingOutlined,
  ThunderboltOutlined, ArrowRightOutlined, AudioOutlined,
} from '@ant-design/icons'
import { Typography, Button, Tooltip } from 'antd'
import VoiceChatOrb from './VoiceChatOrb'

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

// 左/右卡片列（正中语音入口，卡片分居两侧）
const leftModules = modules.slice(0, 4)
const rightModules = modules.slice(4)

/** 进入轻语板块并自动启动语音对话的跨页信号（WhisperPage 挂载时消费） */
export const VOICE_LAUNCH_FLAG = 'gaea_voice_launch'

interface ModuleLauncherProps {
  onNavigate: (target: LauncherTarget) => void
  /** 当前激活的 AI 模型名（顶栏已加载，传入提升真实感） */
  activeModel?: string
}

/** 单张霓虹玻璃悬浮卡片 */
const LauncherCard: React.FC<{ m: LauncherModule; idx: number; onOpen: () => void }> = ({ m, idx, onOpen }) => (
  <div
    role="button"
    tabIndex={0}
    aria-label={`进入${m.name}模块`}
    className="void-card neon-card launcher-card"
    onClick={onOpen}
    onKeyDown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onOpen()
      }
    }}
    style={{
      animationDelay: `${0.08 + idx * 0.05}s, ${0.7 + (idx % 2) * 0.8}s`,
      background: 'rgba(255, 255, 255, 0.04)',
      WebkitBackdropFilter: 'blur(10px) saturate(120%)',
      backdropFilter: 'blur(10px) saturate(120%)',
      border: '1px solid rgba(255, 255, 255, 0.1)',
    }}
  >
    <div style={{
      padding: 18,
      display: 'flex', flexDirection: 'column', gap: 10,
      height: '100%',
    }}>
      <div style={{
        width: 42, height: 42, borderRadius: 'var(--md-sys-radius-lg)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 21, color: m.accent,
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

/** 卡片列（左右两侧，垂直悬浮排列） */
const CardColumn: React.FC<{ list: LauncherModule[]; onNavigate: (t: LauncherTarget) => void }> = ({ list, onNavigate }) => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 16, justifyContent: 'center' }}>
    {list.map((m, i) => (
      <LauncherCard key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
    ))}
  </div>
)

/**
 * ModuleLauncher — AI 中枢首页（语音入口 + 模块启动器）。
 * 正中 = 语言粒子交互入口（点击进入轻语板块启动语音对话，语音能力归属轻语）；
 * 两侧 = 霓虹玻璃悬浮卡片墙。整体三栏悬浮布局。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel }) => {
  // 进入轻语板块并自动启动语音对话（首页只做入口，语音能力在轻语板块）
  const launchVoice = () => {
    try { sessionStorage.setItem(VOICE_LAUNCH_FLAG, '1') } catch (_) {}
    onNavigate('whisper')
  }

  return (
    <div style={{
      flex: 1,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '16px 24px 28px',
      position: 'relative', zIndex: 1,
      minHeight: 0,
    }}>
      <div style={{
        display: 'flex', flexDirection: 'column', gap: 16,
        width: '100%', maxWidth: 1280, minHeight: 0, flex: 1,
      }}>
        {/* ═══ AI 中枢状态条 ═══ */}
        <div
          className="md-glass scanline-top"
          style={{
            position: 'relative',
            display: 'flex', alignItems: 'center', gap: 14,
            padding: '10px 18px', borderRadius: 'var(--md-sys-radius-lg)',
            animation: 'launcherFadeUp 0.4s cubic-bezier(0.16, 1, 0.3, 1) backwards',
            flexShrink: 0,
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
              正中语音入口 —— 语音对话在轻语板块，模型可在模型中心选择
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
            聊天面板 <ArrowRightOutlined style={{ fontSize: 11 }} />
          </span>
        </div>

        {/* ═══ 主区域：左卡片 | 语言交互入口 | 右卡片 ═══ */}
        <div style={{
          flex: 1, minHeight: 0,
          display: 'grid',
          gridTemplateColumns: '1fr 1.6fr 1fr',
          gap: 20,
          alignItems: 'center',
        }}>
          <CardColumn list={leftModules} onNavigate={onNavigate} />

          {/* ── 正中：语言粒子交互入口 ── */}
          <div
            className="void-card neon-card language-core"
            style={{
              position: 'relative',
              display: 'flex', flexDirection: 'column', alignItems: 'center',
              padding: '18px 22px 20px',
              borderRadius: 'var(--md-sys-radius-xl)',
              animation: 'launcherFadeUp 0.5s cubic-bezier(0.16, 1, 0.3, 1) 0.12s backwards',
              overflow: 'hidden',
            }}
          >
            {/* 标题行 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
              <span className="live-dot" />
              <Typography.Text strong style={{ fontSize: 15, color: 'var(--md-sys-color-text)', letterSpacing: '0.04em' }}>
                语言交互中枢
              </Typography.Text>
              <span style={{
                fontSize: 10, padding: '1px 7px', borderRadius: 9,
                background: 'color-mix(in srgb, var(--gaea-glow) 14%, transparent)',
                color: 'var(--gaea-glow)',
                border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
                fontWeight: 500, letterSpacing: '0.06em',
              }}>
                轻语板块
              </span>
            </div>

            {/* 语言粒子球（入口展示，语音实现在轻语板块） */}
            <VoiceChatOrb
              volume={0}
              listening={false}
              speaking={false}
              aiSpeaking={false}
              transcript=""
              size={400}
            />

            {/* 副标题 */}
            <div style={{ textAlign: 'center', marginTop: 6, marginBottom: 12 }}>
              <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)', fontWeight: 500 }}>
                与平台 AI 助手 gaea 语音对话
              </Typography.Text>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 4 }}>
                语音识别 / 对话 / 合成模型可在「模型中心 → 语音模型」选择
              </div>
            </div>

            {/* 入口按钮 */}
            <Tooltip title="进入轻语板块开启语音对话">
              <Button
                type="primary"
                icon={<AudioOutlined />}
                size="large"
                onClick={launchVoice}
                style={{
                  height: 48, padding: '0 28px', borderRadius: 24,
                  fontSize: 15, fontWeight: 600, letterSpacing: '0.04em',
                  background: 'linear-gradient(135deg, var(--gaea-glow), color-mix(in srgb, var(--gaea-glow) 55%, #8b5cf6))',
                  border: 'none', color: '#042f2e',
                  boxShadow: '0 0 26px color-mix(in srgb, var(--gaea-glow) 45%, transparent)',
                  transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal)',
                }}
                className="launcher-voice-btn"
              >
                进入语音对话
              </Button>
            </Tooltip>

            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 14 }}>
              <span style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
                语音能力由轻语板块提供
              </span>
            </div>
          </div>

          <CardColumn list={rightModules} onNavigate={onNavigate} />
        </div>
      </div>
    </div>
  )
}

export default ModuleLauncher
