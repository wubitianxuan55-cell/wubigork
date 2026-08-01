import React from 'react'
import {
  MessageOutlined, ReadOutlined, PictureOutlined, HeartOutlined,
  FileTextOutlined, ToolOutlined, ApiOutlined, SettingOutlined,
} from '@ant-design/icons'

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
}

/**
 * ModuleLauncher — 功能模块卡片墙（纯启动器首页）。
 * 单击卡片即切换至对应模块专属界面；支持键盘 Enter/Space 激活。
 * 使用现有 M3 设计令牌：.md-card 抬升 + .md-ripple 涟漪 + launcherFadeUp 入场。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate }) => (
  <div style={{
    flex: 1,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    padding: '24px 16px',
  }}>
    <div style={{
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fill, minmax(196px, 1fr))',
      gap: 16,
      width: '100%',
      maxWidth: 920,
    }}>
      {modules.map((m, idx) => (
        <div
          key={m.key}
          role="button"
          tabIndex={0}
          aria-label={`进入${m.name}模块`}
          className="md-card md-ripple launcher-card"
          onClick={() => onNavigate(m.key)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              onNavigate(m.key)
            }
          }}
          style={{ animationDelay: `${idx * 0.045}s` }}
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
            }}>
              {m.icon}
            </div>
            <div>
              <div style={{
                fontSize: 15, fontWeight: 600,
                color: 'var(--md-sys-color-text)', marginBottom: 2,
              }}>
                {m.name}
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
      ))}
    </div>
  </div>
)

export default ModuleLauncher
