import React, { useState, useEffect } from 'react'
import { Typography, Button, Space, Steps, Modal, Spin, message } from 'antd'
import {
  ThunderboltOutlined, BulbOutlined,
  UserOutlined, UnorderedListOutlined, RocketOutlined,
  ArrowRightOutlined, ArrowLeftOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import { C } from '../utils/theme'
import { handleError } from '../utils/errorHandler'
import {
  Step1Inspiration, Step2Bootstrap,
  Step3CharacterPolish, Step4OutlinePolish, Step5Complete,
} from './StoryBibleSteps'

// ── 步骤定义 ──────────────────────────────────────────────
const stepItems = [
  { title: '灵感', icon: <BulbOutlined /> },
  { title: '一键生成', icon: <ThunderboltOutlined /> },
  { title: '角色优化', icon: <UserOutlined /> },
  { title: '大纲优化', icon: <UnorderedListOutlined /> },
  { title: '完成', icon: <RocketOutlined /> },
]

interface StoryBibleModalProps {
  open: boolean
  onClose: () => void
}

const StoryBibleModal: React.FC<StoryBibleModalProps> = ({ open, onClose }) => {
  const { novelsDir, openProject, loadProjects } = useAppStore()
  const [step, setStep] = useState(0)
  const [loading, setLoading] = useState(false)

  // 表单
  const [title, setTitle] = useState('')
  const [genre, setGenre] = useState('')
  const [style, setStyle] = useState('')
  const [reference, setReference] = useState('')
  const [importName, setImportName] = useState('')
  const [projectDir, setProjectDir] = useState('')

  // 预览数据
  const [brainIdeas, setBrainIdeas] = useState<any[]>([])
  const [worldviewSections, setWorldviewSections] = useState<any[]>([])
  const [characters, setCharacters] = useState<any[]>([])
  const [outlineNodes, setOutlineNodes] = useState<any[]>([])

  const reset = () => {
    setStep(0); setTitle(''); setGenre(''); setStyle(''); setReference('')
    setImportName(''); setProjectDir('')
    setBrainIdeas([]); setWorldviewSections([]); setCharacters([]); setOutlineNodes([])
  }

  useEffect(() => { if (open) reset() }, [open])

  // ── 步骤 0: 脑暴 ──
  const handleBrainstorm = async () => {
    if (!genre.trim()) { message.warning('请输入题材'); return }
    setLoading(true)
    try {
      // @ts-ignore
      const r = await window.go.app.App.BrainstormIdeas(genre.trim())
      setBrainIdeas(r?.ideas || [])
    } catch (err: any) { handleError('脑暴', err) }
    finally { setLoading(false) }
  }

  const adoptIdea = (idea: any) => {
    setTitle(idea.title)
    setGenre(idea.tags?.[0] || genre)
    setStep(1)
  }

  // ── 步骤 1: 创建小说 + 一键生成 ──
  const handleBootstrap = async () => {
    if (!title.trim()) { message.warning('请填写小说标题'); return }
    setLoading(true)
    try {
      const dir = `${novelsDir}\\${title.replace(/[/\\\\:*?"<>|]/g, '_')}`
      setProjectDir(dir)
      // @ts-ignore
      const result = await window.go.app.App.BootstrapProject(dir, title, genre || '未分类', style || '默认', reference)
      if (result?.worldview) {
        try {
          // @ts-ignore
          const wf = await window.go.app.App.GetWorldviewSections()
          if (wf?.sections) setWorldviewSections(wf.sections)
        } catch (_) {}
      }
      try {
        // @ts-ignore
        const cf = await window.go.app.App.GetCharacters()
        setCharacters(cf?.characters || [])
      } catch (_) {}
      try {
        // @ts-ignore
        const of = await window.go.app.App.GetOutlines()
        setOutlineNodes(of?.nodes || [])
      } catch (_) {}
      setStep(4)
    } catch (err: any) { handleError('生成', err) }
    finally { setLoading(false) }
  }

  // ── 步骤 2: 重新生成角色 ──
  const handleRegenerateCharacters = async () => {
    setLoading(true)
    try {
      // @ts-ignore
      const r = await window.go.app.App.GenerateCharacters(5)
      setCharacters(r?.characters || [])
    } catch (err: any) { handleError('生成', err) }
    finally { setLoading(false) }
  }

  // ── 步骤 3: 重新生成大纲 ──
  const handleRegenerateOutline = async () => {
    setLoading(true)
    try {
      // @ts-ignore
      const r = await window.go.app.App.ContinueOutline(5)
      setOutlineNodes(r?.nodes || r?.outlines?.nodes || [])
    } catch (err: any) { handleError('生成', err) }
    finally { setLoading(false) }
  }

  // ── 步骤 4: 完成 ──
  const handleFinish = () => {
    openProject(projectDir, title)
    loadProjects()
    onClose()
  }

  // ── 导入文件 ──
  const handleImportFile = () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.txt,.md,.json'
    input.onchange = (e: any) => {
      const file = e.target.files?.[0]
      if (!file) return
      const reader = new FileReader()
      reader.onload = () => { setReference(reader.result as string); setImportName(file.name) }
      reader.readAsText(file)
    }
    input.click()
  }

  // ── 渲染当前步骤内容 ──
  const renderStepContent = () => {
    const stepCommon = { loading }

    switch (step) {
      case 0:
        return (
          <Step1Inspiration
            {...stepCommon}
            genre={genre} setGenre={setGenre}
            title={title} setTitle={setTitle}
            reference={reference} setReference={setReference}
            importName={importName}
            brainIdeas={brainIdeas}
            handleBrainstorm={handleBrainstorm}
            handleImportFile={handleImportFile}
            adoptIdea={adoptIdea}
            setStep={setStep}
          />
        )
      case 1:
        return (
          <Step2Bootstrap
            {...stepCommon}
            title={title} setTitle={setTitle}
            style={style} setStyle={setStyle}
            reference={reference} setReference={setReference}
            importName={importName}
            handleBootstrap={handleBootstrap}
            handleImportFile={handleImportFile}
          />
        )
      case 2:
        return (
          <Step3CharacterPolish
            {...stepCommon}
            characters={characters}
            handleRegenerateCharacters={handleRegenerateCharacters}
            setStep={setStep}
          />
        )
      case 3:
        return (
          <Step4OutlinePolish
            {...stepCommon}
            outlineNodes={outlineNodes}
            handleRegenerateOutline={handleRegenerateOutline}
            setStep={setStep}
          />
        )
      case 4:
        return (
          <Step5Complete
            title={title}
            genre={genre}
            worldviewSections={worldviewSections}
            characters={characters}
            outlineNodes={outlineNodes}
            handleFinish={handleFinish}
          />
        )
      default:
        return null
    }
  }

  const showFullSpin = loading && !brainIdeas.length && !worldviewSections.length && !characters.length && !outlineNodes.length

  return (
    <Modal
      title={null}
      open={open}
      onCancel={onClose}
      footer={null}
      width={720}
      styles={{ body: { padding: 0 } }}
    >
      {/* 顶部标题 + 步骤条 */}
      <div style={{ padding: '16px 24px 12px', borderBottom: '1px solid ' + C('color-border') }}>
        <Space style={{ marginBottom: 12 }}>
          <ThunderboltOutlined style={{ color: '#f59e0b', fontSize: 18 }} />
          <Typography.Title level={4} style={{ color: C('color-text'), margin: 0 }}>
            Story Bible · 引导式创建
          </Typography.Title>
        </Space>
        <Steps current={step} items={stepItems} size="small" style={{ marginTop: 8 }} />
      </div>

      {/* 步骤内容 */}
      <div style={{ padding: '16px 24px', minHeight: 320, maxHeight: '60vh', overflow: 'auto' }}>
        {showFullSpin ? (
          <div style={{ textAlign: 'center', padding: 60 }}>
            <Spin size="large" />
            <div style={{ color: C('color-text-secondary'), marginTop: 16 }}>
              AI 正在生成{stepItems[step].title}...
            </div>
          </div>
        ) : (
          renderStepContent()
        )}
      </div>

      {/* 底部导航 */}
      <div style={{
        padding: '12px 24px', borderTop: '1px solid ' + C('color-border'),
        display: 'flex', justifyContent: 'space-between',
      }}>
        <Space>
          {step > 0 && (
            <Button size="large" onClick={() => setStep(step - 1)} icon={<ArrowLeftOutlined />}>上一步</Button>
          )}
          {step === 0 && (
            <Button size="large" onClick={() => { reset(); onClose() }}>取消</Button>
          )}
        </Space>
        {step === 0 && (
          <Button size="large" onClick={() => {
            if (title.trim()) {
              setGenre(genre || '未分类'); setStyle(style || '默认'); setStep(1)
              return
            }
            if (genre.trim()) { handleBrainstorm(); return }
            message.warning('请输入题材或标题')
          }} type="primary" style={{ background: '#f59e0b', borderColor: '#f59e0b' }}>
            开始 <ThunderboltOutlined />
          </Button>
        )}
        {step > 0 && step < 4 && (
          <Button size="large" type="primary" onClick={() => setStep(4)}
            style={{ background: C('color-primary'), borderColor: C('color-primary') }}>
            跳过预览 <ArrowRightOutlined />
          </Button>
        )}
      </div>
    </Modal>
  )
}

export default StoryBibleModal
