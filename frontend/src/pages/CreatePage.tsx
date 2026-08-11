import React, { useState, useEffect, useCallback } from 'react'
import { Typography, Button, Input, InputNumber, Card, Space, message, Spin, Modal, Popconfirm, Select, Checkbox, Tag } from 'antd'
import {
  BookOutlined,
  EditOutlined, BulbOutlined, RightOutlined, DownOutlined,
  LoadingOutlined, PlusOutlined, SaveOutlined,
  DeleteOutlined, ReloadOutlined, ShareAltOutlined,
  SettingOutlined, FileTextOutlined, ExperimentOutlined, BarChartOutlined, ThunderboltOutlined,
} from '@ant-design/icons'
import { C } from '../utils/theme'
import { useOutlineStore } from '../stores/outlineStore'
import { useAppStore } from '../stores/appStore'
import type { OutlineNode } from '../types'
import { associateToProject, syncProjectCharacters } from '../api/characterlib'
import * as App from '../../wailsjs/go/app/App'

const { TextArea } = Input

interface Branch { title: string; pitch: string }

// 新角色条目（可编辑名称 + 可选择）
interface NewCharEntry {
  original: string   // AI 提取的原始名
  name: string       // 编辑后的名字
  selected: boolean
}

// 角色库已有同名角色（直接关联，不新建）
interface LibMatchEntry {
  id: string
  name: string
  roleType?: string
  portraitUrl?: string
  selected: boolean
}

const roleLabels: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '次要',
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
  const projectPath = useAppStore(s => s.projectPath)
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
  const [skills, setSkills] = useState<{name: string; description: string; appliesTo?: string[]}[]>([])
  const [selectedSkill, setSelectedSkill] = useState<string | undefined>(undefined)
  // 生成设置
  const [minWords, setMinWords] = useState(5000)
  const [temperature, setTemperature] = useState<number>(0)
  const [directPlot, setDirectPlot] = useState('')
  const [stats, setStats] = useState<{ totalWords: number; chapterCount: number } | null>(null)

  // 新角色发现弹窗
  const [newCharsModal, setNewCharsModal] = useState(false)
  const [newCharsList, setNewCharsList] = useState<NewCharEntry[]>([])
  const [newCharsChapter, setNewCharsChapter] = useState(0)
  const [libMatches, setLibMatches] = useState<LibMatchEntry[]>([])
  const [adding, setAdding] = useState(false)

  // 拉取最新小说设定；projectPath 变化（切换小说）时也会重新拉取，
  // 避免页面常驻导致创作提示词使用陈旧/空设定
  const refreshSetting = useCallback(async () => {
    let fresh = ''
    try { fresh = await App.GetWorldview() || '' } catch (_) { }
    setSetting(fresh)
    return fresh
  }, [])

  // 创作统计（章节数 / 总字数）
  const refreshStats = useCallback(async () => {
    try {
      const s: any = await App.GetStats()
      if (s) setStats(s)
    } catch (_) {}
  }, [])

  useEffect(() => {
    ;(async () => {
      loadOutlines()
      await refreshSetting()
      refreshStats()
    })()
  }, [projectPath, loadOutlines, refreshSetting, refreshStats])

  // 切换小说时清空编辑区与选中节点，避免展示上一个项目的内容
  useEffect(() => {
    setActiveId('')
    setContent('')
  }, [projectPath])

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
      if (data?.characters?.length > 0 || data?.libraryMatches?.length > 0) {
        setNewCharsList((data.characters || []).map((name: string) => ({
          original: name, name, selected: true
        })))
        setLibMatches((data.libraryMatches || []).map((m: any) => ({
          id: m.id, name: m.name, roleType: m.roleType || '', portraitUrl: m.portraitUrl || '', selected: true,
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
    // 每次打开向导都重新读取设定，确保分支构思基于最新设定
    const freshSetting = await refreshSetting()
    const prevSummary = prevChapter > 0 ? getPrevSummary(prevChapter) : ''
    const res = await App.QuickBrainstormBranches(freshSetting, prevSummary || '')
    const list = (res as any)?.branches || []
    setBranches(list.map((b: any) => ({ title: b.title, pitch: b.summary })))
    setWizStep('branches')
  }

  // 直接开始生成：注册流式监听并调用后端（带目标字数/温度/技能设置）
  const startGeneration = async (plotReq: string, overwriteChapter: number = 0, branchFromID: string = '') => {
    if (!plotReq.trim()) { message.warning('请选择分支或输入剧情要求'); return }
    setWizOpen(false); setGenerating(true); setGenPhase('正在生成…')
    setContent('')

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
          refreshStats()
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
      // 生成前再读一次最新设定，确保正文提示词注入当前小说设定
      const freshSetting = await refreshSetting()
      const result = await App.CreateChapter(freshSetting, '', plotReq, overwriteChapter, branchFromID, selectedSkill || '', minWords, temperature)
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

  const confirmGenerate = async () => {
    const chosen = selectedBranch !== null ? branches[selectedBranch] : null
    const plotReq = userInput.trim() || (chosen ? `${chosen.title}：${chosen.pitch}` : '')
    await startGeneration(plotReq, wizOverwriteChapter, wizBranchFromID)
  }

  // 直接生成下一章（跳过分支向导）
  const handleDirectGenerate = () => {
    if (!directPlot.trim()) { message.warning('请输入剧情要求'); return }
    const chapNum = activeNode?.order_index || 0
    const next = outlines.find(nx => nx.order_index === chapNum + 1 && !nx.parent_id)
    if (next) {
      Modal.confirm({
        title: `第${chapNum + 1}章已存在「${next.title || `第${chapNum + 1}章`}」`,
        content: '选择生成方式：',
        okText: '覆盖下一章',
        cancelText: '作为分支追加',
        onOk: () => startGeneration(directPlot, chapNum + 1, ''),
        onCancel: () => startGeneration(directPlot, 0, next.id),
      })
    } else {
      startGeneration(directPlot, 0, '')
    }
  }

  const handleDelete = async (node: OutlineNode) => {
    try {
      await App.DeleteOutlineNode(node.id)
      if (activeId === node.id) { setActiveId(''); setContent('') }
      await loadOutlines()
      refreshStats()
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
  const selectedSkillDesc = skills.find(s => s.name === selectedSkill)?.description
  const selectedSkillApplies = skills.find(s => s.name === selectedSkill)?.appliesTo || []

  return (
    <div className="novel-workspace">

      {/* ── 左：章节树 ── */}
      <aside className="novel-panel novel-workspace-col novel-tree-col">
        <div className="novel-panel-head">
          <span className="novel-panel-title"><ShareAltOutlined />章节</span>
          <div style={{ flex: 1 }} />
          <span className="novel-setting-meta">{flatNodes.length} 章</span>
        </div>
        <div className="novel-tree" style={{ flex: 1, overflow: 'auto' }}>
          {flatNodes.length === 0 ? (
            <div className="novel-tree-empty">
              <BookOutlined />
              <span>还没有章节，从「生成第 1 章」开始</span>
            </div>
          ) : flatNodes.map(tn => {
            const n = tn.node
            const isActive = activeId === n.id
            const chapNum = n.order_index || 1
            const isBranch = !!n.parent_id
            const statusClass = n.status === 'generating' ? 'is-writing' : n.status === 'written' ? 'is-done' : 'is-todo'
            return (
              <div
                key={n.id}
                className={`novel-tree-item${isActive ? ' is-active' : ''}`}
                style={{ paddingLeft: 8 + tn.depth * 14, borderLeft: isBranch ? '2px solid var(--md-sys-color-outline-variant)' : undefined }}
              >
                <span className={`novel-tree-status ${statusClass}`} title={n.status || '未写'} />
                <div onClick={() => selectChapter(n)}
                  title={n.summary ? `摘要: ${n.summary.slice(0, 100)}${n.summary.length > 100 ? '...' : ''}` : `ID: ${n.id}`}
                  style={{ flex: 1, cursor: 'pointer', fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', minWidth: 0 }}>
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
            )
          })}
        </div>
        <div className="novel-tree-footer">
          <Button block type="dashed" icon={<PlusOutlined />} onClick={() => openWizard(outlines.length)}>
            生成第{outlines.length + 1}章
          </Button>
        </div>
      </aside>

      {/* ── 中：编辑器 ── */}
      <section className="novel-panel novel-workspace-col novel-editor-col">
        <div className="novel-panel-head">
          <span className="novel-panel-title">
            {activeNode ? (activeNode.title || `第${activeNode.order_index}章`) : '正文编辑'}
          </span>
          {activeNode?.status === 'generating' && <Tag color="warning" style={{ marginInlineEnd: 0, fontSize: 11 }}>生成中</Tag>}
          {activeNode?.status === 'written' && <Tag color="success" style={{ marginInlineEnd: 0, fontSize: 11 }}>已写入</Tag>}
          <div style={{ flex: 1 }} />
          <span className="novel-setting-meta">{content.length.toLocaleString()} 字</span>
          {activeNode && (
            <Button size="small" icon={<ReloadOutlined />} onClick={() => handleRegenerate(activeNode)}>重写</Button>
          )}
          {activeNode && (
            <Button size="small" type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving}>保存</Button>
          )}
        </div>

        <div className="novel-editor-body">
          {chapterLoading ? (
            <div className="novel-editor-loading"><Spin /></div>
          ) : (activeNode || generating) ? (
            <TextArea className="novel-editor" value={content} onChange={e => setContent(e.target.value)}
              placeholder="AI 将在此流式呈现正文；也可直接手写后保存…"
            />
          ) : (
            <div className="novel-editor-empty">
              <EditOutlined className="novel-editor-empty-icon" />
              {flatNodes.length === 0 ? (
                <>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14 }}>
                    开始创作你的第一部章节
                  </Typography.Text>
                  <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => openWizard(0)}>
                    生成第 1 章
                  </Button>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    可先在右侧选择写作技能、设置目标字数与温度，再构思剧情方向
                  </Typography.Text>
                </>
              ) : (
                <>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14 }}>
                    选择左侧章节查看，或生成下一章
                  </Typography.Text>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => openWizard(outlines.length)}>
                    生成第 {outlines.length + 1} 章
                  </Button>
                </>
              )}
            </div>
          )}
        </div>

        {generating && (
          <div className="novel-gen-status">
            <LoadingOutlined />
            {genPhase}
          </div>
        )}
      </section>

      {/* ── 右：创作设置 ── */}
      <aside className="novel-panel novel-workspace-col novel-inspector">
        <div className="novel-panel-head">
          <span className="novel-panel-title"><SettingOutlined />创作设置</span>
        </div>
        <div className="novel-inspector-body">

          <section className="novel-inspector-section">
            <div className="novel-inspector-section-title">
              <FileTextOutlined />小说设定
              <div style={{ flex: 1 }} />
              <Button type="text" size="small" icon={<ReloadOutlined />} title="刷新设定" onClick={() => refreshSetting()} />
            </div>
            {setting ? (
              <div className="novel-setting-preview-box" title={setting}>
                {setting.slice(0, 240)}{setting.length > 240 ? '…' : ''}
              </div>
            ) : (
              <div className="novel-inspector-hint">设定为空，请先在「设定」页填写世界观，创作提示词会注入最新设定。</div>
            )}
          </section>

          <section className="novel-inspector-section">
            <div className="novel-inspector-section-title"><ExperimentOutlined />写作技能</div>
            <Select
              allowClear
              placeholder="选择写作技能（可选）"
              value={selectedSkill}
              onChange={(v) => setSelectedSkill(v)}
              options={skills.map(s => ({ value: s.name, label: s.name }))}
              size="small"
              style={{ width: '100%' }}
            />
            {selectedSkill && selectedSkillDesc && (
              <div className="novel-skill-desc">{selectedSkillDesc}</div>
            )}
            {selectedSkill && selectedSkillApplies.length > 0 && (
              <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                {selectedSkillApplies.map(a => <Tag key={a} style={{ marginInlineEnd: 0, fontSize: 10 }}>适用·{a}</Tag>)}
              </div>
            )}
            {skills.length === 0 && (
              <div className="novel-inspector-hint">暂无可用写作技能（将技能放入 skills 目录即可自动加载）</div>
            )}
          </section>

          <section className="novel-inspector-section">
            <div className="novel-inspector-section-title"><SettingOutlined />生成设置</div>
            <div className="novel-inspector-row">
              <span>目标字数</span>
              <InputNumber
                min={1000} max={20000} step={500} size="small"
                value={minWords}
                onChange={(v) => setMinWords((v as number) || 5000)}
                style={{ width: 120 }}
              />
            </div>
            <div className="novel-inspector-row">
              <span>生成温度</span>
              <Select
                size="small"
                value={temperature}
                onChange={(v) => setTemperature(v as number)}
                options={[
                  { value: 0, label: '默认' },
                  { value: 0.7, label: '0.7 · 稳妥' },
                  { value: 0.8, label: '0.8 · 平衡' },
                  { value: 0.9, label: '0.9 · 灵动' },
                  { value: 1.0, label: '1.0 · 大胆' },
                ]}
                style={{ width: 120 }}
              />
            </div>
            <div className="novel-inspector-hint" style={{ fontSize: 11 }}>
              目标字数不足时自动续写；温度越高文风越奔放。
            </div>
          </section>

          <section className="novel-inspector-section">
            <div className="novel-inspector-section-title"><BulbOutlined />剧情方向</div>
            <Button block icon={<BulbOutlined />} onClick={() => openWizard(activeNode?.order_index || outlines.length)}>
              构思剧情方向
            </Button>
            <TextArea
              rows={3}
              value={directPlot}
              onChange={e => setDirectPlot(e.target.value)}
              placeholder="或直接输入剧情要求，跳过分支构思…"
              style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', color: C('color-text'), borderRadius: 'var(--radius-md)', fontSize: 12 }}
            />
            <Button block type="primary" icon={<ThunderboltOutlined />} onClick={handleDirectGenerate} disabled={!directPlot.trim()}>
              按剧情要求直接生成
            </Button>
          </section>

          <section className="novel-inspector-section">
            <div className="novel-inspector-section-title"><BarChartOutlined />创作统计</div>
            <div className="novel-inspector-row"><span>章节数</span><b>{stats?.chapterCount ?? flatNodes.length}</b></div>
            <div className="novel-inspector-row"><span>累计字数</span><b>{(stats?.totalWords ?? 0).toLocaleString()} 字</b></div>
          </section>

        </div>
      </aside>
      {/* 新角色发现弹窗 */}
      <Modal
        title={<>🔍 第{newCharsChapter}章发现了 {newCharsList.length + libMatches.length} 个新角色</>}
        open={newCharsModal}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        confirmLoading={adding}
        onOk={async () => {
          const selected = newCharsList.filter(c => c.selected).map(c => c.name)
          const libSelected = libMatches.filter(m => m.selected)
          if (selected.length === 0 && libSelected.length === 0) {
            message.warning('请至少选择一个角色')
            return
          }
          setAdding(true)
          try {
            if (selected.length > 0) {
              await App.SaveCharactersBatch(JSON.stringify(selected))
              message.success(`已添加 ${selected.length} 个新角色（含 AI 完整档案）`)
            }
            if (libSelected.length > 0) {
              for (const m of libSelected) {
                await associateToProject(m.id, m.roleType || 'supporting')
              }
              await syncProjectCharacters()
              message.success(`已关联 ${libSelected.length} 个角色库已有角色`)
            }
          } catch (err: any) {
            message.error(err?.message || '添加失败')
          } finally {
            setAdding(false)
            setNewCharsModal(false)
            // 角色面板常驻挂载：通知其重新读取全局库与项目引用
            try { window.dispatchEvent(new CustomEvent('gaea-project-chars-changed')) } catch (_) {}
          }
        }}
        onCancel={() => setNewCharsModal(false)}
        okText={adding ? 'AI 生成档案中…' : `确认添加 (${newCharsList.filter(c => c.selected).length + libMatches.filter(m => m.selected).length})`}
        cancelText="稍后处理"
        width={480}
      >
        {libMatches.length > 0 && (
          <>
            <div style={{ marginBottom: 4, fontSize: 12, fontWeight: 600, color: '#60a5fa' }}>
              角色库已有同名角色（直接关联，不新建）
            </div>
            <div style={{
              maxHeight: 180, overflow: 'auto', border: '1px solid rgba(96,165,250,0.25)',
              borderRadius: 8, padding: '2px 8px', marginBottom: 10,
            }}>
              {libMatches.map((m, i) => (
                <div key={m.id} style={{
                  display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0',
                  borderBottom: i < libMatches.length - 1 ? '1px solid var(--border-subtle)' : 'none',
                }}>
                  <Checkbox
                    checked={m.selected}
                    onChange={e => setLibMatches(prev => prev.map((x, j) => j === i ? { ...x, selected: e.target.checked } : x))}
                  />
                  <span style={{ fontSize: 13, color: 'var(--color-text)' }}>{m.name}</span>
                  {m.roleType && (
                    <Tag color="blue" style={{ marginInlineEnd: 0, fontSize: 11 }}>{roleLabels[m.roleType] || m.roleType}</Tag>
                  )}
                  <div style={{ flex: 1 }} />
                  <Typography.Text type="secondary" style={{ fontSize: 11 }}>库内角色</Typography.Text>
                </div>
              ))}
            </div>
          </>
        )}
        <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Checkbox
            disabled={newCharsList.length === 0}
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
        {newCharsList.length > 0 && (
          <div style={{ maxHeight: 260, overflow: 'auto' }}>
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
        )}
        <Typography.Text type="secondary" style={{ fontSize: 12, marginTop: 8, display: 'block' }}>
          新角色将 AI 生成完整档案（性格/背景/外貌）并标记为「配角·存活」；角色库已有角色直接关联，不新建。
        </Typography.Text>
      </Modal>

      {/* 向导 */}
      <Modal title={<><BulbOutlined style={{ marginRight: 8 }} />剧情方向</>}
        open={wizOpen} onCancel={() => setWizOpen(false)} footer={null} width={620}
        destroyOnHidden transitionName="" maskTransitionName="">
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
