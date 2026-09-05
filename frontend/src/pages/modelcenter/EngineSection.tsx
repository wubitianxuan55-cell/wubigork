import { useState, type CSSProperties } from 'react'
import { Button, Input, InputNumber, message, Popconfirm, Segmented, Space, Switch } from 'antd'
import { PlusOutlined, SettingOutlined } from '@ant-design/icons'
import { SectionHead, StatusChip } from './ui'
import { engineColor, engineIcon, engineLabel, filterEnginesByEnabled, glmEndpointFamily, isCustomEngine, isValidBaseURL, kindOf } from './utils'
import type { EngineConfig } from '../../api/engines'
import { saveEngine } from '../../api/engines'
import { useModelCenter } from './context'

export function EngineSection() {
  const {
    engines, engineStatuses, activeEngine, testingEngine, savingEngine,
    editingURLs, setEditingURLs,
    deepseekKey, setDeepseekKeyState, deepseekKeyMasked,
    glmKey, setGlmKeyState, glmKeyMasked,
    opencodeGoKey, setOpencodeGoKeyState, opencodeGoKeyMasked,
    opencodeZenKey, setOpencodeZenKeyState, opencodeZenKeyMasked,
    modelHubKey, setModelHubKeyState, modelHubKeyMasked,
    settingGlmEndpoint, handleSetGlmEndpoint,
    handleTestConnection, handleRefreshModels, handleSaveURL, handleToggleEngine,
    handleBulkToggleEngines,
    handleSaveDeepseekKey, handleSaveGlmKey, handleSaveOpencodeGoKey, handleSaveOpencodeZenKey,
    handleSaveModelHubKey,
    handleAddCustomEngine, handleUpdateCustomEngine, handleRemoveCustomEngine,
    makeModels,
  } = useModelCenter()
  const [showEnabledOnly, setShowEnabledOnly] = useState(false)
  const [bulkBusy, setBulkBusy] = useState(false)

  // ── 自定义引擎（A 刀）：添加/编辑行内表单状态 ─────────────────
  const [showAddForm, setShowAddForm] = useState(false)
  const [addName, setAddName] = useState('')
  const [addURL, setAddURL] = useState('')
  const [addKey, setAddKey] = useState('')
  const [addBusy, setAddBusy] = useState(false)
  const [editingCustomId, setEditingCustomId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [editURL, setEditURL] = useState('')
  const [editKey, setEditKey] = useState('')
  const [editBusy, setEditBusy] = useState(false)
  // ── 价目 v1（自定义引擎用户价目）：引擎级统一价，每百万 tokens，CNY ──
  // null = 输入框留空 = 清除（后端语义 0=未填=不计价）。经 SaveEngine 指针
  // 三态落库：数字=设置、0=清除；本会话已保存值优先于 context 引擎列表
  // （saveEngine 成功后 engines 不刷新，避免重开表单显示旧值）。
  const [editPriceIn, setEditPriceIn] = useState<number | null>(null)
  const [editPriceOut, setEditPriceOut] = useState<number | null>(null)
  const [savedPrices, setSavedPrices] = useState<Record<string, { in: number | null; out: number | null }>>({})

  const visibleEngines = filterEnginesByEnabled(engines, showEnabledOnly)
  const bulk = (enabled: boolean) => {
    setBulkBusy(true)
    handleBulkToggleEngines(enabled).finally(() => setBulkBusy(false))
  }

  const closeAddForm = () => { setShowAddForm(false); setAddName(''); setAddURL(''); setAddKey('') }
  const closeEditCustom = () => { setEditingCustomId(null); setEditKey('') }

  // 表单校验（名称非空 + 地址 http(s) 前缀，与后端 validBaseURL 同口径双保险，
  // 防 API Key 粘进地址框）。通过返回 null，不通过返回提示文案。
  const customFormError = (name: string, url: string): string | null => {
    if (!name) return '请输入引擎名称'
    if (!isValidBaseURL(url)) return 'API 地址无效：必须以 http:// 或 https:// 开头'
    return null
  }

  const submitAdd = async () => {
    const name = addName.trim()
    const url = addURL.trim()
    const invalid = customFormError(name, url)
    if (invalid) { message.warning(invalid); return }
    setAddBusy(true)
    try {
      if (await handleAddCustomEngine(name, url, addKey.trim())) closeAddForm()
    } finally { setAddBusy(false) }
  }

  const openEditCustom = (engine: EngineConfig) => {
    setEditingCustomId(engine.id)
    setEditName(engine.name)
    setEditURL(engine.base_url || '')
    setEditKey('')
    // 价目预填：本会话已保存值优先（context 引擎列表在 saveEngine 后未刷新）
    const saved = savedPrices[engine.id]
    setEditPriceIn(saved ? saved.in : engine.user_price_in ?? null)
    setEditPriceOut(saved ? saved.out : engine.user_price_out ?? null)
  }

  const submitEditCustom = async () => {
    if (!editingCustomId) return
    const name = editName.trim()
    const url = editURL.trim()
    const invalid = customFormError(name, url)
    if (invalid) { message.warning(invalid); return }
    if (editPriceIn !== null && (!Number.isFinite(editPriceIn) || editPriceIn < 0)) {
      message.warning('输入价必须是不小于 0 的数字')
      return
    }
    if (editPriceOut !== null && (!Number.isFinite(editPriceOut) || editPriceOut < 0)) {
      message.warning('输出价必须是不小于 0 的数字')
      return
    }
    setEditBusy(true)
    try {
      // Key 留空 = 不修改（后端 UpdateCustomEngine 契约）
      if (!(await handleUpdateCustomEngine(editingCustomId, name, url, editKey.trim()))) return
      // 价目 v1：经既有 SaveEngine 通道落库（数字=设置、留空=0=清除；
      // 费用折算优先消费用户价目，留空维持现状不计价）
      await saveEngine({
        id: editingCustomId,
        user_price_in: editPriceIn ?? 0,
        user_price_out: editPriceOut ?? 0,
      } as EngineConfig)
      setSavedPrices(prev => ({ ...prev, [editingCustomId]: { in: editPriceIn, out: editPriceOut } }))
      closeEditCustom()
    } catch {
      message.error('价目保存失败，请重试')
    } finally {
      setEditBusy(false)
    }
  }

  const keyFields: {
    type: string
    value: string
    setter: (v: string) => void
    placeholder: string
    save: () => Promise<void>
  }[] = [
    {
      type: 'deepseek',
      value: deepseekKey,
      setter: setDeepseekKeyState,
      placeholder: deepseekKeyMasked || 'sk-...',
      save: handleSaveDeepseekKey,
    },
    {
      type: 'glm',
      value: glmKey,
      setter: setGlmKeyState,
      placeholder: glmKeyMasked || '智谱 API Key（open.bigmodel.cn 获取）',
      save: handleSaveGlmKey,
    },
    {
      type: 'opencode-go',
      value: opencodeGoKey,
      setter: setOpencodeGoKeyState,
      placeholder: opencodeGoKeyMasked || 'oc-...（opencode.ai 订阅获取）',
      save: handleSaveOpencodeGoKey,
    },
    {
      type: 'opencode-zen',
      value: opencodeZenKey,
      setter: setOpencodeZenKeyState,
      placeholder: opencodeZenKeyMasked || 'zen-...（opencode.ai/auth 获取）',
      save: handleSaveOpencodeZenKey,
    },
    {
      type: 'modelhub',
      value: modelHubKey,
      setter: setModelHubKeyState,
      placeholder: modelHubKeyMasked || 'sk-unsloth-...（Unsloth 设置 → API 创建）',
      save: handleSaveModelHubKey,
    },
  ]

  return (
    <section className="mc-section">
      <SectionHead
        icon={<SettingOutlined />}
        title="引擎管理"
        desc="配置云端与本地引擎地址、连接状态、API Key 和模型分类"
        extra={(
          <>
            <Button
              size="small"
              icon={<PlusOutlined />}
              onClick={() => (showAddForm ? closeAddForm() : setShowAddForm(true))}
            >
              添加自定义引擎
            </Button>
            <Button size="small" loading={bulkBusy} onClick={() => bulk(true)}>全部启用</Button>
            <Popconfirm
              title="禁用全部引擎？"
              description="各功能将回退到无可用模型，建议谨慎操作。"
              onConfirm={() => bulk(false)}
            >
              <Button size="small" danger loading={bulkBusy}>全部禁用</Button>
            </Popconfirm>
            <Segmented
              size="small"
              value={showEnabledOnly ? 'enabled' : 'all'}
              onChange={(v) => setShowEnabledOnly(v === 'enabled')}
              options={[
                { label: '全部', value: 'all' },
                { label: '仅已启用', value: 'enabled' },
              ]}
            />
          </>
        )}
      />

      {showAddForm && (
        <div
          style={{
            border: '1px solid var(--mc-border)',
            borderRadius: 'var(--mc-radius-md)',
            padding: '10px 12px',
            display: 'grid',
            gap: 8,
            maxWidth: 640,
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600 }}>添加自定义引擎（OpenAI 兼容）</div>
          <Input
            size="small"
            aria-label="自定义引擎名称"
            placeholder="名称（如：硅基流动 / OneAPI 中转）"
            value={addName}
            onChange={e => setAddName(e.target.value)}
          />
          <Input
            size="small"
            aria-label="自定义引擎 API 地址"
            placeholder="API 地址（https://api.example.com/v1）"
            value={addURL}
            onChange={e => setAddURL(e.target.value)}
          />
          <Input
            size="small"
            aria-label="自定义引擎 API Key"
            placeholder="API Key（可选，服务商控制台获取）"
            value={addKey}
            onChange={e => setAddKey(e.target.value)}
          />
          <Space>
            <Button size="small" type="primary" loading={addBusy} onClick={submitAdd}>保存</Button>
            <Button size="small" onClick={closeAddForm}>取消</Button>
          </Space>
        </div>
      )}

      {visibleEngines.map(engine => {
        const color = engineColor(engine)
        const glmFamily = glmEndpointFamily(engine.base_url)
        const em = makeModels(engine)
        const mc = {
          llm: em.filter(m => kindOf(m) === 'llm').length,
          tts: em.filter(m => kindOf(m) === 'tts').length,
          stt: em.filter(m => kindOf(m) === 'stt').length,
          image: em.filter(m => kindOf(m) === 'image').length,
          embedding: em.filter(m => kindOf(m) === 'embedding').length,
          rerank: em.filter(m => kindOf(m) === 'rerank').length,
          ocr: em.filter(m => kindOf(m) === 'ocr').length,
        }
        const keyField = keyFields.find(k => k.type === engine.type)
        return (
          <div
            key={engine.id}
            className={`mc-engine-card${engine.enabled ? ' is-enabled' : ''}`}
            style={{ '--engine-color': color } as CSSProperties}
          >
            <div className="mc-engine-head">
              <div className="mc-engine-id">
                <span className="mc-engine-icon">{engineIcon(engine)}</span>
                <div style={{ minWidth: 0 }}>
                  <div className="mc-engine-name">{engine.name}</div>
                  <div className="mc-engine-sub">
                    {engineLabel(engine)} · 默认模型 {
                      (engine.models || []).find(m => m.id === engine.default_model)?.name
                      || engine.default_model || '未设置'
                    }
                  </div>
                </div>
              </div>
              <div className="mc-engine-actions">
                <StatusChip tone={engine.is_local ? 'warn' : 'neutral'}>
                  {engine.is_local ? '本地' : '云端'}
                </StatusChip>
                {isCustomEngine(engine) && <StatusChip tone="accent">自定义</StatusChip>}
                {activeEngine === engine.id && (
                  <StatusChip tone="ok" dot>当前活跃</StatusChip>
                )}
                <Switch
                  size="small"
                  checked={engine.enabled}
                  onChange={(v) => handleToggleEngine(engine, v)}
                />
                <Button
                  size="small"
                  onClick={() => handleTestConnection(engine.id)}
                  loading={testingEngine === engine.id}
                  disabled={!engine.enabled}
                >
                  测试连接
                </Button>
                <Button
                  size="small"
                  onClick={() => handleRefreshModels(engine.id)}
                  loading={testingEngine === engine.id}
                  disabled={!engine.enabled}
                >
                  刷新
                </Button>
                {isCustomEngine(engine) && (
                  <>
                    <Button
                      size="small"
                      onClick={() => (editingCustomId === engine.id ? closeEditCustom() : openEditCustom(engine))}
                    >
                      {editingCustomId === engine.id ? '收起' : '编辑'}
                    </Button>
                    <Popconfirm
                      title="删除该自定义引擎？"
                      description="将同时移除其功能绑定能力，已绑定功能将回退到其他可用模型。"
                      onConfirm={() => handleRemoveCustomEngine(engine.id)}
                    >
                      <Button size="small" danger>删除</Button>
                    </Popconfirm>
                  </>
                )}
              </div>
            </div>

            <div className="mc-engine-stats">
              {mc.llm > 0 && <StatusChip>{mc.llm} 语言</StatusChip>}
              {mc.tts > 0 && <StatusChip tone="warn">{mc.tts} TTS</StatusChip>}
              {mc.stt > 0 && <StatusChip tone="accent">{mc.stt} STT</StatusChip>}
              {mc.image > 0 && <StatusChip tone="warn">{mc.image} 图片</StatusChip>}
              {mc.embedding > 0 && <StatusChip tone="accent">{mc.embedding} Embedding</StatusChip>}
              {mc.rerank > 0 && <StatusChip tone="warn">{mc.rerank} Rerank</StatusChip>}
              {mc.ocr > 0 && <StatusChip tone="ok">{mc.ocr} OCR</StatusChip>}
              {em.length === 0 && <StatusChip>暂无模型</StatusChip>}
            </div>

            {/* 本地引擎 + 自定义引擎显示地址框；内置云端引擎不露（v4.9.1 防线：
                曾有用户把 API Key 粘进地址框导致 base_url=Key 本体） */}
            {(engine.is_local || isCustomEngine(engine)) && (
              <Space.Compact style={{ width: '100%', maxWidth: 640 }}>
                <Input
                  size="small"
                  aria-label="引擎服务地址"
                  value={editingURLs[engine.id] || ''}
                  onChange={e => setEditingURLs(prev => ({ ...prev, [engine.id]: e.target.value }))}
                  disabled={!engine.enabled}
                  style={{ fontSize: 12 }}
                />
                <Button
                  size="small"
                  onClick={() => handleSaveURL(engine)}
                  loading={savingEngine === engine.id}
                  disabled={!engine.enabled}
                >
                  保存
                </Button>
              </Space.Compact>
            )}

            {keyField && (
              <Space.Compact style={{ width: '100%', maxWidth: 640 }}>
                <Input
                  size="small"
                  value={keyField.value}
                  onChange={e => keyField.setter(e.target.value)}
                  placeholder={keyField.placeholder}
                  disabled={!engine.enabled}
                  style={{ fontSize: 12 }}
                />
                <Button
                  size="small"
                  onClick={keyField.save}
                  loading={savingEngine === engine.id}
                  disabled={!engine.enabled}
                >
                  保存 Key
                </Button>
              </Space.Compact>
            )}

            {editingCustomId === engine.id && (
              <div
                style={{
                  border: '1px dashed var(--mc-border)',
                  borderRadius: 'var(--mc-radius-sm)',
                  padding: '8px 10px',
                  display: 'grid',
                  gap: 8,
                  maxWidth: 640,
                }}
              >
                <div style={{ fontSize: 12, color: 'var(--mc-muted)' }}>编辑自定义引擎（Key 留空 = 不修改）</div>
                <Input
                  size="small"
                  aria-label="编辑引擎名称"
                  placeholder="名称"
                  value={editName}
                  onChange={e => setEditName(e.target.value)}
                />
                <Input
                  size="small"
                  aria-label="编辑引擎 API 地址"
                  placeholder="API 地址（http:// 或 https:// 开头）"
                  value={editURL}
                  onChange={e => setEditURL(e.target.value)}
                />
                <Input
                  size="small"
                  aria-label="编辑引擎 API Key"
                  placeholder="API Key（留空 = 不修改）"
                  value={editKey}
                  onChange={e => setEditKey(e.target.value)}
                />
                {/* 价目 v1：引擎级统一价（每百万 tokens，CNY）。留空 = 清除 =
                    不计价（费用统计维持现状）；仅影响费用估算，不影响调用。 */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                  <InputNumber
                    size="small"
                    style={{ width: '100%' }}
                    min={0}
                    step={0.1}
                    aria-label="输入价（每百万 tokens，CNY）"
                    placeholder="输入价（每百万 tokens，CNY）"
                    value={editPriceIn}
                    onChange={v => setEditPriceIn(typeof v === 'number' && Number.isFinite(v) ? v : null)}
                  />
                  <InputNumber
                    size="small"
                    style={{ width: '100%' }}
                    min={0}
                    step={0.1}
                    aria-label="输出价（每百万 tokens，CNY）"
                    placeholder="输出价（每百万 tokens，CNY）"
                    value={editPriceOut}
                    onChange={v => setEditPriceOut(typeof v === 'number' && Number.isFinite(v) ? v : null)}
                  />
                </div>
                <div style={{ fontSize: 11, color: 'var(--mc-muted)' }}>
                  价目为引擎级统一价（¥/百万 tokens），留空 = 不计价；填了价目的引擎，用量统计的费用估算将按此价折算
                </div>
                <Space>
                  <Button size="small" type="primary" loading={editBusy} onClick={submitEditCustom}>保存</Button>
                  <Button size="small" onClick={closeEditCustom}>取消</Button>
                </Space>
              </div>
            )}

            {engine.id === 'glm' && (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                <span style={{ fontSize: 12, color: 'var(--mc-muted)' }}>端点</span>
                <Segmented
                  size="small"
                  value={glmFamily}
                  onChange={(v) => handleSetGlmEndpoint(v as 'std' | 'coding')}
                  disabled={!engine.enabled || settingGlmEndpoint}
                  options={[
                    { value: 'std', label: '标准（按量付费）' },
                    { value: 'coding', label: '编码套餐' },
                  ]}
                />
                <span style={{ fontSize: 11, color: 'var(--mc-muted)' }}>
                  编码套餐 Key 须走 /api/coding 端点；标准 Key 走 /api/paas
                  {glmFamily === 'coding' && '；编码套餐=积分制计费，下方费用估算不含该端点用量'}
                </span>
              </div>
            )}

            {engine.id === 'modelhub' && (
              <div style={{ fontSize: 11, color: 'var(--mc-muted)' }}>
                先在本机 Unsloth Studio「Model hub」加载模型；刷新后只列出已加载模型。
                Key 在 Unsloth 左下角头像 → 设置 → API 创建（sk-unsloth- 开头）；
                地址保持 Studio 固定入口 127.0.0.1:8888/v1 即可（llama 内部端口每次加载会变，8888/v1 自动转发当前模型）。
              </div>
            )}

            {engineStatuses[engine.id] && (
              <div className="mc-engine-status">
                {engineStatuses[engine.id].connected ? (
                  <span style={{ color: 'var(--mc-ok)' }}>
                    连接正常 · {engineStatuses[engine.id].model_count} 个模型
                    {engineStatuses[engine.id].latency_ms ? ` · ${engineStatuses[engine.id].latency_ms}ms` : ''}
                  </span>
                ) : (
                  <span style={{ color: 'var(--mc-danger)' }}>
                    连接失败 · {engineStatuses[engine.id].error}
                  </span>
                )}
                {engineStatuses[engine.id].last_checked && (
                  <span style={{ color: 'var(--mc-muted)', marginLeft: 8 }}>
                    上次检查 {engineStatuses[engine.id].last_checked}
                  </span>
                )}
              </div>
            )}
          </div>
        )
      })}
    </section>
  )
}
