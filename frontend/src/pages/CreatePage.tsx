import React, { useState, useEffect } from 'react'
import { Typography, Button, Input, Card, Space, message, Spin, Modal, Popconfirm, Select, Checkbox } from 'antd'
import {
  BookOutlined,
  EditOutlined, BulbOutlined, RightOutlined, DownOutlined,
  LoadingOutlined, PlusOutlined, SaveOutlined,
  DeleteOutlined, ReloadOutlined, ShareAltOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import { useOutlineStore } from '../stores/outlineStore'
import type { OutlineNode } from '../types'
import * as App from '../../wailsjs/go/app/App'

const { TextArea } = Input

interface Branch { title: string; pitch: string }

// 新角色条目（可编辑名称 + 可选择）
interface NewCharEntry {
  original: string   // AI 提取的原始名
  name: string       // 编辑后的名字
  selected: boolean
}

// 构建节点树：{ node, children[], depth }
interface TreeNode { node: OutlineNode; children: TreeNode[]; depth: number }

function buildTree(nodes: OutlineNode[]): TreeNode[] {
  const map = new Map<string, TreeNode>()
  const roots: TreeNode[] = []
  for (const n of nodes) {
    map.set(n.id, { node: n, children: [], depth: 0 })
  }
  for (const n of nodes) {
    const tn = map.get(n.id)!
    if (n.parent_id && map.has(n.parent_id)) {
      map.get(n.parent_id)!.children.push(tn)
    } else {
      roots.push(tn)
    }
  }
  // 设置深度
  const setDepth = (list: TreeNode[], d: number) => {
    for (const tn of list) { tn.depth = d; setDepth(tn.children, d + 1) }
  }
  setDepth(roots, 0)
  return roots
}

function flattenTree(roots: TreeNode[]): TreeNode[] {
  const result: TreeNode[] = []
  const walk = (list: TreeNode[]) => {
    for (const tn of list) { result.push(tn); walk(tn.children) }
  }
  walk(roots)
  return result
}

const CreatePage: React.FC = () => {
  const [setting, setSetting] = useState('')
  const outlines = useOutlineStore(s => s.outlines)
  const loadOutlines = useOutlineStore(s => s.loadOutlines)
  const tree = buildTree(outlines)
  const flatNodes = flattenTree(tree)

  const [activeId, setActiveId] = useState<string>('')
  const [content, setContent] = useState('')
  const [chapterLoading, setChapterLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [saving, setSaving] = useState(false)
  const [genPhase, setGenPhase] = useState('')

  const [wizOpen, setWizOpen] = useState(false)
  const [wizStep, setWizStep] = useState<'loading' | 'branches'>('loading')
  const [branches, setBranches] = useState<Branch[]>([])
  const [selectedBranch, setSelectedBranch] = useState<number | null>(null)
  const [userInput, setUserInput] = useState('')
  const [wizPrevChapter, setWizPrevChapter] = useState(0)
  const [wizOverwriteChapter, setWizOverwriteChapter] = useState(0)
  const [wizBranchFromID, setWizBranchFromID] = useState('')
  const [skills, setSkills] = useState<{name: string; description: string}[]>([])
  const [selectedSkill, setSelectedSkill] = useState<string | undefined>(undefined)

  // 新角色发现弹窗
  const [newCharsModal, setNewCharsModal] = useState(false)
  const [newCharsList, setNewCharsList] = useState<NewCharEntry[]>([])
  const [newCharsChapter, setNewCharsChapter] = useState(0)
  useEffect(() => { loadOutlines() }, [])
  useEffect(() => {
    ;(async () => {
      try { setSetting(await App.GetWorldview() || '') } catch (_) { }
    })()
  }, [])

  // 加载可用 Skill 列表
  useEffect(() => {
    (async () => {
      try {
        const list = await App.ListSkills()
        setSkills((list || []) as any)
      } catch (_) {}
    })()
  }, [])

  // 监听新角色发现事件
  useEffect(() => {
    const handler = (event: any) => {
      const data = event.detail || event
      if (data?.characters?.length > 0) {
        setNewCharsList(data.characters.map((name: string) => ({
          original: name, name, selected: true
        })))
        setNewCharsChapter(data.chapterNum || 0)
        setNewCharsModal(true)
      }
    }
    try { window.runtime?.EventsOn?.('new-characters-discovered', handler) } catch (_) {}
    return () => { try { window.runtime?.EventsOff?.('new-characters-discovered') } catch (_) {} }
  }, [])
  const selectChapter = async (node: OutlineNode) => {
    setActiveId(node.id); setChapterLoading(true)
    try {
      const branch = (node as any).branch || ''
      setContent((await App.GetChapterBranch(node.order_index || 1, branch) as any)?.content || '')
    } catch (_) { setContent('') }
    finally { setChapterLoading(false) }
  }

  const getPrevSummary = (upToChapter: number): string => {
    const parts: string[] = []
    for (const n of outlines) {
      const cn = n.order_index || 0
      if (cn > 0 && cn <= upToChapter && n.summary) {
        parts.push(`第${cn}章：${n.summary.slice(0, 100)}`)
      }
    }
    return parts.join('\n\n')
  }

  const openWizard = (prevChapter: number, overwriteChapter: number = 0, branchFromID: string = '') => {
    setWizPrevChapter(prevChapter)
    setWizOverwriteChapter(overwriteChapter)
    setWizBranchFromID(branchFromID)
    setWizOpen(true)
    setWizStep('loading'); setBranches([]); setSelectedBranch(null); setUserInput('')
    fetchBranchesFor(prevChapter)
  }

  const fetchBranchesFor = async (prevChapter: number) => {
      const prevSummary = prevChapter > 0 ? getPrevSummary(prevChapter) : ''
      const res = await App.QuickBrainstormBranches(setting, prevSummary || '')
      const list = (res as any)?.branches || []
      setBranches(list.map((b: any) => ({ title: b.title, pitch: b.summary })))
      setWizStep('branches')
  }

  const confirmGenerate = async () => {
    const chosen = selectedBranch !== null ? branches[selectedBranch] : null
    const plotReq = userInput.trim() || (chosen ? `${chosen.title}：${chosen.pitch}` : '')
    if (!plotReq) { message.warning('请选择分支或输入剧情要求'); return }
    setWizOpen(false); setGenerating(true); setGenPhase('正在生成…')
    setContent('')

    // 注册流式事件监听
    const handler = (event: any) => {
      const data = event.detail || event
      switch (data.type) {
        case 'chunk':
          setContent((prev) => prev + (data.content || ''))
          setGenPhase(`正在生成… ${(data.total || 0).toLocaleString()} 字`)
          break
        case 'done':
          setGenPhase('')
          setGenerating(false)
          const chNum = data.chapterNum || 0
          const branch = data.branch || ''
          const label = branch ? `第${chNum}${branch}章` : `第${chNum}章`
          message.success(`${label} 生成完成（${(data.total || 0).toLocaleString()} 字）`)
          loadOutlines().then(() => {
            const updated = useOutlineStore.getState().outlines
            const ch = updated.find(n => n.order_index === chNum && (n as any).branch === branch)
            if (ch) setActiveId(ch.id)
          })
          try { window.runtime?.EventsOff?.('create-chapter-stream') } catch (_) {}
          break
        case 'error':
          setGenPhase('')
          setGenerating(false)
          message.error(data.error || '生成失败')
          try { window.runtime?.EventsOff?.('create-chapter-stream') } catch (_) {}
          break
      }
    }

    try {
      try { window.runtime?.EventsOff?.('create-chapter-stream') } catch (_) {}
      try { window.runtime?.EventsOn?.('create-chapter-stream', handler) } catch (_) {}
      const result = await App.CreateChapter(setting, '', plotReq, wizOverwriteChapter, wizBranchFromID, selectedSkill || '')
      // 预创建节点已由后端同步完成，立即激活
      const nodeId = (result as any)?.nodeId
      const chapNum = (result as any)?.chapterNum
      if (nodeId) {
        const store = useOutlineStore.getState()
        if (!store.outlines.find(n => n.id === nodeId)) {
          store.setOutlines([...store.outlines, {
            id: nodeId, order_index: chapNum,
            title: `第${chapNum}章`, status: 'generating',
            parent_id: '', summary: '',
          } as any])
        }
        setActiveId(nodeId)
        setContent('')
      }
    } catch (err: any) {
      setGenerating(false)
      setGenPhase('')
      message.error(err?.message || '生成失败')
      try { window.runtime?.EventsOff?.('create-chapter-stream') } catch (_) {}
    }
  }

  const handleDelete = async (node: OutlineNode) => {
    try {
      await App.DeleteOutlineNode(node.id)
      if (activeId === node.id) { setActiveId(''); setContent('') }
      await loadOutlines()
      message.success('已删除')
    } catch (err: any) { message.error(err?.message || '失败') }
  }

  const handleRegenerate = (node: OutlineNode) => {
    const prevChap = (node.order_index || 1) - 1
    openWizard(prevChap, node.order_index || 1)
  }

  const handleSave = async () => {
    const node = outlines.find(n => n.id === activeId)
    if (!node || !content.trim()) return
    setSaving(true)
    try {
      const branch = (node as any).branch || ''
      await App.SaveChapterBranchContent(node.order_index || 1, branch, content)
      message.success('已保存')
    } catch (err: any) { message.error(err?.message || '失败') }
    finally { setSaving(false) }
  }

  const activeNode = outlines.find(n => n.id === activeId)

  return (
    <div style={{ height: 'calc(100vh - 56px)', display: 'flex', flexDirection: 'column', gap: 12 }}>

      <div style={{ flex: 1, display: 'flex', gap: 12, minHeight: 0 }}>

        {/* 左：章节节点树 */}
        <Card size="small" title={<><ShareAltOutlined style={{ marginRight: 6 }} />节点树</>}
          style={{ width: 260, flexShrink: 0, background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)', display: 'flex', flexDirection: 'column' }}
          bodyStyle={{ flex: 1, overflow: 'auto', padding: 8 }}>
          {flatNodes.length === 0 ? (
            <div style={{ textAlign: 'center', padding: 24, color: C('color-text-secondary'), fontSize: 12 }}>
              <BookOutlined style={{ fontSize: 24, opacity: 0.3, display: 'block', marginBottom: 8 }} />暂无章节
            </div>
          ) : flatNodes.map(tn => {
            const n = tn.node
            const isActive = activeId === n.id
            const chapNum = n.order_index || 1
            const isBranch = !!n.parent_id
            const nodeBranch = (n as any).branch || ''
            return (
              <div key={n.id} style={{ marginBottom: 2 }}>
                <div style={{
                  display: 'flex', alignItems: 'center',
                  padding: '4px 6px', paddingLeft: 6 + tn.depth * 16,
                  borderRadius: 6,
                  background: isActive ? 'var(--md-sys-color-primary-container)' : 'transparent',
                  borderLeft: isBranch ? '2px solid var(--md-sys-color-outline-variant)' : undefined,
                }}>
                  <div onClick={() => selectChapter(n)}
                    title={n.summary ? `摘要: ${n.summary.slice(0, 100)}${n.summary.length > 100 ? '...' : ''}` : `ID: ${n.id}`}
                    style={{ flex: 1, cursor: 'pointer', color: isActive ? 'var(--md-sys-color-on-primary-container)' : C('color-text'), fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', minWidth: 0 }}>
                    {isBranch ? <ShareAltOutlined style={{ fontSize: 9, marginRight: 4, opacity: 0.5 }} /> : <RightOutlined style={{ fontSize: 10, marginRight: 4, opacity: 0.4 }} />}
                    {n.title || `第${chapNum}章`}
                  </div>
                  <Space size={0}>
                    <Button type="text" size="small" icon={<ReloadOutlined />}
                      onClick={() => handleRegenerate(n)}
                      style={{ color: C('color-text-secondary'), fontSize: 11, padding: '0 3px', height: 22 }} title="重新生成" />
                    <Popconfirm title="删除此章节？" onConfirm={() => handleDelete(n)} okText="删除" cancelText="取消">
                      <Button type="text" size="small" danger icon={<DeleteOutlined />}
                        style={{ fontSize: 11, padding: '0 3px', height: 22 }} />
                    </Popconfirm>
                    <Button type="text" size="small" icon={<PlusOutlined />}
                      onClick={() => {
                        const next = outlines.find(nx => nx.order_index === chapNum + 1 && !nx.parent_id)
                        if (next) {
                          Modal.confirm({
                            title: `第${chapNum}章后已有「${next.title || `第${chapNum + 1}章`}」`,
                            content: '选择生成方式：',
                            okText: '覆盖下一章',
                            cancelText: '末尾追加',
                            onOk: () => openWizard(chapNum, chapNum + 1),
                            onCancel: () => openWizard(chapNum, 0, n.id),
                          })
                        } else {
                          openWizard(chapNum, 0, '')
                        }
                      }}
                      style={{ color: 'var(--md-sys-color-primary)', fontSize: 11, padding: '0 3px', height: 22 }}
                      title={tn.children.length > 0 ? '添加子分支' : '生成下一章'} />
                  </Space>
                </div>
              </div>
            )
          })}
          <div onClick={() => openWizard(outlines.length)}
            style={{ marginTop: 8, padding: '8px', borderRadius: 6, cursor: 'pointer', border: '1px dashed var(--md-sys-color-primary)', textAlign: 'center', color: 'var(--md-sys-color-primary)', fontSize: 13 }}>
            <PlusOutlined style={{ marginRight: 4 }} />生成第{outlines.length + 1}章
          </div>
        </Card>

        {/* 右：编辑区 */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 8, minWidth: 0 }}>
          {generating && (
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}><LoadingOutlined style={{ marginRight: 4 }} />{genPhase}</Typography.Text>
          )}
          {chapterLoading ? (
            <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
          ) : (activeNode || generating) ? (
            <>
              {skills.length > 0 && (
                <Select
                  allowClear
                  placeholder="选择写作技能（可选）"
                  value={selectedSkill}
                  onChange={(v) => setSelectedSkill(v)}
                  options={skills.map(s => ({ value: s.name, label: s.name }))}
                  size="small"
                  style={{ width: 220, alignSelf: 'flex-end' }}
                />
              )}
              <TextArea value={content} onChange={e => setContent(e.target.value)}
                style={{ flex: 1, resize: 'none', background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', color: C('color-text'), borderRadius: 'var(--radius-md)', fontSize: 14, lineHeight: 1.8, fontFamily: '"Noto Serif SC", "Source Han Serif SC", "SimSun", serif', minHeight: 0 }}
              />
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
                {activeNode && <Button icon={<ReloadOutlined />} onClick={() => handleRegenerate(activeNode)}>重新生成</Button>}
                <Button icon={<SaveOutlined />} onClick={handleSave} loading={saving}>保存</Button>
              </div>
            </>
          ) : (
            <Card style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-glass)', border: '1px dashed var(--border-subtle)', borderRadius: 'var(--radius-lg)' }}>
              <div style={{ textAlign: 'center' }}>
                <EditOutlined style={{ fontSize: 36, opacity: 0.2, marginBottom: 12, color: C('color-text-secondary') }} />
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13, display: 'block' }}>
                  {flatNodes.length === 0 ? '点击底部「生成第1章」开始' : '选择左侧节点查看或编辑'}
                </Typography.Text>
              </div>
            </Card>
          )}
        </div>
      </div>
      {/* 新角色发现弹窗 */}
      <Modal
        title={<>🔍 第{newCharsChapter}章发现了 {newCharsList.length} 个新角色</>}
        open={newCharsModal}
        onOk={async () => {
          const selected = newCharsList.filter(c => c.selected).map(c => c.name)
          if (selected.length === 0) { message.warning('请至少选择一个角色'); return }
          try {
            await App.SaveCharactersBatch(JSON.stringify(selected))
            message.success(`已添加 ${selected.length} 个角色`)
          } catch (err: any) { message.error(err?.message || '添加失败') }
          setNewCharsModal(false)
        }}
        onCancel={() => setNewCharsModal(false)}
        okText={`确认添加 (${newCharsList.filter(c => c.selected).length})`}
        cancelText="稍后处理"
        width={480}
      >
        <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Checkbox
            checked={newCharsList.every(c => c.selected)}
            indeterminate={newCharsList.some(c => c.selected) && !newCharsList.every(c => c.selected)}
            onChange={e => setNewCharsList(prev => prev.map(c => ({ ...c, selected: e.target.checked })))}
          >
            全选
          </Checkbox>
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            💡 可编辑名字合并重复称呼
          </Typography.Text>
        </div>
        <div style={{ maxHeight: 320, overflow: 'auto' }}>
          {newCharsList.map((entry, i) => (
            <div key={i} style={{
              display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0',
              borderBottom: '1px solid var(--border-subtle)'
            }}>
              <Checkbox
                checked={entry.selected}
                onChange={e => {
                  setNewCharsList(prev => prev.map((c, j) => j === i ? { ...c, selected: e.target.checked } : c))
                }}
              />
              <Input
                size="small"
                value={entry.name}
                onChange={e => {
                  setNewCharsList(prev => prev.map((c, j) => j === i ? { ...c, name: e.target.value } : c))
                }}
                style={{
                  flex: 1, background: entry.name !== entry.original ? 'rgba(245,158,11,0.08)' : 'transparent',
                  border: entry.name !== entry.original ? '1px solid rgba(245,158,11,0.4)' : '1px solid transparent',
                  color: 'var(--color-text)', fontSize: 13
                }}
              />
              {entry.name !== entry.original && (
                <Button type="text" size="small"
                  onClick={() => setNewCharsList(prev => prev.map((c, j) => j === i ? { ...c, name: c.original } : c))}
                  style={{ fontSize: 10, padding: '0 2px', height: 20, color: '#f59e0b' }}>
                  还原
                </Button>
              )}
            </div>
          ))}
        </div>
        <Typography.Text type="secondary" style={{ fontSize: 12, marginTop: 8, display: 'block' }}>
          角色将标记为「配角·存活」，可在角色页面补充详情
        </Typography.Text>
      </Modal>

      {/* 向导 */}
      <Modal title={<><BulbOutlined style={{ marginRight: 8 }} />剧情方向</>}
        open={wizOpen} onCancel={() => setWizOpen(false)} footer={null} width={620} destroyOnClose>
        {wizStep === 'loading' && <div style={{ textAlign: 'center', padding: 24 }}><Spin size="large" /><div style={{ marginTop: 8, color: C('color-text-secondary'), fontSize: 12 }}>AI 正在分析设定，构思剧情分支…</div></div>}
        {wizStep === 'branches' && (<>
          {branches.length > 0 && <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginBottom: 12 }}>
            {branches.map((b, i) => (
              <Card key={i} size="small" hoverable onClick={() => { setSelectedBranch(i); setUserInput('') }}
                style={{ cursor: 'pointer', border: selectedBranch === i ? '2px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)', background: selectedBranch === i ? 'var(--md-sys-color-primary-container)' : 'var(--bg-elevated)' }}>
                <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>{i + 1}. {b.title}</Typography.Text>
                <Typography.Paragraph style={{ color: C('color-text-secondary'), fontSize: 12, margin: '4px 0 0' }}>{b.pitch}</Typography.Paragraph>
              </Card>
            ))}
          </div>}
          <Space direction="vertical" style={{ width: '100%' }}>
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>或输入你自己的剧情要求：</Typography.Text>
            <TextArea value={userInput} onChange={e => { setUserInput(e.target.value); setSelectedBranch(null) }} rows={2} placeholder="剧情要求…"
              style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', color: C('color-text'), borderRadius: 'var(--radius-md)' }} />
          </Space>
          <div style={{ marginTop: 12, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => fetchBranchesFor(wizPrevChapter)}>重新构思</Button>
              <Button type="primary" onClick={confirmGenerate} disabled={selectedBranch === null && !userInput.trim()}>
                {wizOverwriteChapter > 0 ? `重新生成第${wizOverwriteChapter}章` : '生成'}
              </Button>
            </Space>
          </div>
        </>)}
      </Modal>
    </div>
  )
}

export default CreatePage
