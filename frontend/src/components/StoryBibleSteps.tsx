import React from 'react'
import { Typography, Button, Space, Tag, Input, Card, Row, Col, message } from 'antd'
import {
  ThunderboltOutlined, BulbOutlined,
  UserOutlined, UnorderedListOutlined, RocketOutlined,
  ReloadOutlined, ArrowRightOutlined,
  BookOutlined, GlobalOutlined, TeamOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'

// ── 角色角色类型配置 ──
const roleColors: Record<string, string> = {
  protagonist: '#4ade80', antagonist: '#f87171', supporting: '#60a5fa', minor: C('color-text-secondary'),
}
const roleLabels: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '次要',
}

// ═══════════════════════════════════════════════
// Step 1 灵感
// ═══════════════════════════════════════════════

interface Step1Props {
  genre: string; setGenre: (v: string) => void
  title: string; setTitle: (v: string) => void
  reference: string; setReference: (v: string) => void
  importName: string
  brainIdeas: any[]
  loading: boolean
  handleBrainstorm: () => Promise<void>
  handleImportFile: () => void
  adoptIdea: (idea: any) => void
  setStep: (s: number) => void
}

export const Step1Inspiration: React.FC<Step1Props> = ({
  genre, setGenre, title, setTitle,
  reference, setReference, importName,
  brainIdeas, loading, handleBrainstorm, handleImportFile, adoptIdea, setStep,
}) => (
  <Space direction="vertical" size={16} style={{ width: '100%' }}>
    <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13 }}>
      AI 脑暴会根据题材生成 6 个独特的小说点子，选一个作为起点。
    </Typography.Text>

    <Space.Compact style={{ width: '100%' }}>
      <Input
        placeholder="题材（如：修仙+科幻、都市悬疑...）"
        value={genre} onChange={(e) => setGenre(e.target.value)}
        onPressEnter={handleBrainstorm}
        size="large"
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
      />
      <Button type="primary" size="large" icon={<ThunderboltOutlined />} onClick={handleBrainstorm}
        loading={loading} style={{ background: '#f59e0b', borderColor: '#f59e0b' }}>生成点子</Button>
    </Space.Compact>

    {/* 跳过脑暴——手动输入 */}
    <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
      <Input placeholder="或者直接输入小说标题..." value={title}
        onChange={(e) => setTitle(e.target.value)} size="large"
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }} />
      <Button size="large" onClick={() => { if (title.trim()) setStep(1) }}
        style={{ borderColor: C('color-primary'), color: C('color-primary') }}>下一步</Button>
    </div>

    {brainIdeas.length > 0 && (
      <div>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginBottom: 8 }}>
          选一个喜欢的点子 ↓
        </Typography.Text>
        <Row gutter={[10, 10]}>
          {brainIdeas.map((idea: any) => (
            <Col key={idea.id} xs={24} sm={12}>
              <Card
                hoverable size="small" onClick={() => adoptIdea(idea)}
                style={{ background: C('color-bg-layout'), borderColor: C('color-border'), borderRadius: 8, height: '100%' }}
              >
                <Typography.Text strong style={{ color: '#f59e0b', fontSize: 13 }}>#{idea.id} {idea.title}</Typography.Text>
                <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 4, lineHeight: 1.5 }}>{idea.pitch}</div>
                <Space size={4} style={{ marginTop: 6 }}>
                  <Tag color="purple" style={{ fontSize: 9 }}>{idea.conflict}</Tag>
                  {(idea.tags || []).slice(0, 2).map((t: string) => <Tag key={t} style={{ fontSize: 9 }}>{t}</Tag>)}
                </Space>
              </Card>
            </Col>
          ))}
        </Row>
      </div>
    )}

    {/* 参考素材 */}
    <div>
      <Space style={{ marginBottom: 8 }}>
        <Button size="small" onClick={handleImportFile}>📁 导入参考文件</Button>
        {importName && <span style={{ color: C('color-primary'), fontSize: 12 }}>{importName}</span>}
      </Space>
      <Input.TextArea placeholder="或者粘贴参考素材..." value={reference}
        onChange={(e) => setReference(e.target.value)} rows={3}
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }} />
    </div>
  </Space>
)

// ═══════════════════════════════════════════════
// Step 2 一键生成
// ═══════════════════════════════════════════════

interface Step2Props {
  title: string; setTitle: (v: string) => void
  style: string; setStyle: (v: string) => void
  reference: string; setReference: (v: string) => void
  importName: string
  loading: boolean
  handleBootstrap: () => Promise<void>
  handleImportFile: () => void
}

export const Step2Bootstrap: React.FC<Step2Props> = ({
  title, setTitle, style, setStyle,
  reference, setReference, importName,
  loading, handleBootstrap, handleImportFile,
}) => (
  <Space direction="vertical" size={16} style={{ width: '100%' }}>
    <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13 }}>
      填写基本信息，AI 一键生成世界观、角色和大纲。导入参考文件会让结果更贴合你的构思。
    </Typography.Text>

    <Input placeholder="小说标题" value={title}
      onChange={(e) => setTitle(e.target.value)} size="large"
      style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }} />
    <div style={{ display: 'flex', gap: 8 }}>
      <Input placeholder="文风（如：热血战斗、细腻温情...）" value={style}
        onChange={(e) => setStyle(e.target.value)}
        style={{ flex: 1, background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }} />
    </div>

    <div>
      <Space style={{ marginBottom: 8 }}>
        <Button size="small" onClick={handleImportFile}>📁 导入参考文件</Button>
        {importName && <Tag color="green">{importName}</Tag>}
      </Space>
      <Input.TextArea
        placeholder="或者粘贴参考素材（可选）——已有设定、灵感片段..."
        value={reference} onChange={(e) => setReference(e.target.value)} rows={3}
        style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }} />
    </div>

    <div style={{ textAlign: 'center', paddingTop: 16 }}>
      <Button type="primary" size="large" icon={<ThunderboltOutlined />}
        onClick={handleBootstrap} loading={loading}
        style={{ background: C('color-primary'), borderColor: C('color-primary'), padding: '8px 48px', height: 48 }}>
        一键生成全部
      </Button>
    </div>
  </Space>
)

// ═══════════════════════════════════════════════
// Step 3 角色优化
// ═══════════════════════════════════════════════

interface Step3Props {
  characters: any[]
  loading: boolean
  handleRegenerateCharacters: () => Promise<void>
  setStep: (s: number) => void
}

export const Step3CharacterPolish: React.FC<Step3Props> = ({
  characters, loading, handleRegenerateCharacters, setStep,
}) => {
  if (characters.length === 0) {
    return (
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <div style={{ textAlign: 'center', padding: 40 }}>
          <UserOutlined style={{ fontSize: 40, color: C('color-text-secondary'), marginBottom: 12 }} />
          <Typography.Paragraph style={{ color: C('color-text-secondary') }}>先完成一键生成</Typography.Paragraph>
        </div>
      </Space>
    )
  }

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Space>
        <Button size="small" icon={<ReloadOutlined />} onClick={handleRegenerateCharacters} loading={loading}
          style={{ borderColor: '#f59e0b', color: '#f59e0b' }}>重新生成角色</Button>
        <Tag color="green">{characters.length} 个角色</Tag>
      </Space>
      <Row gutter={[10, 10]}>
        {characters.map((ch: any) => (
          <Col key={ch.id} xs={24} sm={12} md={8}>
            <Card size="small" hoverable
              style={{ background: C('color-bg-layout'), borderColor: C('color-border'), borderRadius: 8, height: '100%' }}>
              <Space direction="vertical" size={4}>
                <Space size={4}>
                  <Tag color={roleColors[ch.role_type]} style={{ fontSize: 10 }}>{roleLabels[ch.role_type]}</Tag>
                  <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>{ch.name}</Typography.Text>
                </Space>
                {ch.personality && (
                  <div style={{ color: C('color-text-secondary'), fontSize: 11, lineHeight: 1.5 }}>
                    {ch.personality.slice(0, 80)}{ch.personality.length > 80 ? '…' : ''}
                  </div>
                )}
                {ch.gender && <Tag style={{ fontSize: 9 }}>{ch.gender} · {ch.age || '?'}岁</Tag>}
              </Space>
            </Card>
          </Col>
        ))}
      </Row>
      <div style={{ textAlign: 'right' }}>
        <Button size="large" type="primary" onClick={() => setStep(3)}
          style={{ background: C('color-primary'), borderColor: C('color-primary') }}>
          下一步 <ArrowRightOutlined />
        </Button>
      </div>
    </Space>
  )
}

// ═══════════════════════════════════════════════
// Step 4 大纲优化
// ═══════════════════════════════════════════════

interface Step4Props {
  outlineNodes: any[]
  loading: boolean
  handleRegenerateOutline: () => Promise<void>
  setStep: (s: number) => void
}

export const Step4OutlinePolish: React.FC<Step4Props> = ({
  outlineNodes, loading, handleRegenerateOutline, setStep,
}) => {
  if (outlineNodes.length === 0) {
    return (
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <div style={{ textAlign: 'center', padding: 40 }}>
          <UnorderedListOutlined style={{ fontSize: 40, color: C('color-text-secondary'), marginBottom: 12 }} />
          <Typography.Paragraph style={{ color: C('color-text-secondary') }}>先完成一键生成</Typography.Paragraph>
        </div>
      </Space>
    )
  }

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Space>
        <Button size="small" icon={<ReloadOutlined />} onClick={handleRegenerateOutline} loading={loading}
          style={{ borderColor: '#f59e0b', color: '#f59e0b' }}>重新生成大纲</Button>
        <Tag color="green">{outlineNodes.length} 个节点</Tag>
      </Space>
      <div style={{
        maxHeight: 300, overflow: 'auto',
        background: C('color-bg-layout'), borderRadius: 8, border: '1px solid ' + C('color-border'),
        padding: 8,
      }}>
        {outlineNodes.map((n: any, i: number) => {
          const isVolume = !n.parent_id
          return (
            <div key={n.id} style={{
              padding: isVolume ? '6px 10px' : '4px 10px 4px 28px',
              borderLeft: isVolume ? '3px solid #c084fc' : '2px solid ' + C('color-border'),
              marginBottom: 2,
            }}>
              <Space size={4}>
                <span style={{ color: '#c084fc', fontSize: 10, fontWeight: 600 }}>{i + 1}</span>
                <span style={{ color: C('color-text'), fontSize: 12 }}>{n.title}</span>
                {n.summary && <span style={{ color: C('color-text-secondary'), fontSize: 10 }}>{n.summary.slice(0, 30)}</span>}
              </Space>
            </div>
          )
        })}
      </div>
      <div style={{ textAlign: 'right' }}>
        <Button size="large" type="primary" onClick={() => setStep(4)}
          style={{ background: C('color-primary'), borderColor: C('color-primary') }}>
          下一步 <ArrowRightOutlined />
        </Button>
      </div>
    </Space>
  )
}

// ═══════════════════════════════════════════════
// Step 5 完成
// ═══════════════════════════════════════════════

interface Step5Props {
  title: string
  genre: string
  worldviewSections: any[]
  characters: any[]
  outlineNodes: any[]
  handleFinish: () => void
}

export const Step5Complete: React.FC<Step5Props> = ({
  title, genre, worldviewSections, characters, outlineNodes, handleFinish,
}) => (
  <div style={{ textAlign: 'center', padding: '40px 0' }}>
    <RocketOutlined style={{ fontSize: 56, color: C('color-primary'), marginBottom: 16 }} />
    <Typography.Title level={4} style={{ color: C('color-text') }}>一切就绪！</Typography.Title>
    <Space direction="vertical" size={8} style={{ marginBottom: 24 }}>
      <Typography.Text style={{ color: C('color-text-secondary') }}>
        <BookOutlined style={{ marginRight: 6 }} />《{title}》 · {genre}
      </Typography.Text>
      <Typography.Text style={{ color: C('color-text-secondary') }}>
        <GlobalOutlined style={{ marginRight: 6 }} />{worldviewSections.length} 个世界观维度
      </Typography.Text>
      <Typography.Text style={{ color: C('color-text-secondary') }}>
        <TeamOutlined style={{ marginRight: 6 }} />{characters.length} 个角色
      </Typography.Text>
      <Typography.Text style={{ color: C('color-text-secondary') }}>
        <UnorderedListOutlined style={{ marginRight: 6 }} />{outlineNodes.length} 个大纲节点
      </Typography.Text>
    </Space>
    <Button
      type="primary" size="large" icon={<RocketOutlined />}
      onClick={handleFinish}
      style={{ background: C('color-primary'), borderColor: C('color-primary'), padding: '8px 48px', height: 44 }}>
      开始写作！
    </Button>
  </div>
)
