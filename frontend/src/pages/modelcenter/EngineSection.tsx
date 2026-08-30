import { useState, type CSSProperties } from 'react'
import { Button, Input, Popconfirm, Segmented, Space, Switch } from 'antd'
import { SettingOutlined } from '@ant-design/icons'
import { SectionHead, StatusChip } from './ui'
import { engineColor, engineIcons, engineLabel, filterEnginesByEnabled, glmEndpointFamily, kindOf } from './utils'
import { useModelCenter } from './context'

export function EngineSection() {
  const {
    engines, engineStatuses, activeEngine, testingEngine, savingEngine,
    editingURLs, setEditingURLs,
    deepseekKey, setDeepseekKeyState, deepseekKeyMasked,
    glmKey, setGlmKeyState, glmKeyMasked,
    opencodeGoKey, setOpencodeGoKeyState, opencodeGoKeyMasked,
    opencodeZenKey, setOpencodeZenKeyState, opencodeZenKeyMasked,
    settingGlmEndpoint, handleSetGlmEndpoint,
    handleTestConnection, handleRefreshModels, handleSaveURL, handleToggleEngine,
    handleBulkToggleEngines,
    handleSaveDeepseekKey, handleSaveGlmKey, handleSaveOpencodeGoKey, handleSaveOpencodeZenKey,
    makeModels,
  } = useModelCenter()
  const [showEnabledOnly, setShowEnabledOnly] = useState(false)
  const [bulkBusy, setBulkBusy] = useState(false)
  const visibleEngines = filterEnginesByEnabled(engines, showEnabledOnly)
  const bulk = (enabled: boolean) => {
    setBulkBusy(true)
    handleBulkToggleEngines(enabled).finally(() => setBulkBusy(false))
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
  ]

  return (
    <section className="mc-section">
      <SectionHead
        icon={<SettingOutlined />}
        title="引擎管理"
        desc="配置云端与本地引擎地址、连接状态、API Key 和模型分类"
        extra={(
          <>
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

      {visibleEngines.map(engine => {
        const color = engineColor(engine)
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
                <span className="mc-engine-icon">{engineIcons[engine.id]}</span>
                <div style={{ minWidth: 0 }}>
                  <div className="mc-engine-name">{engine.name}</div>
                  <div className="mc-engine-sub">
                    {engineLabel(engine)} · 默认模型 {engine.default_model || '未设置'}
                  </div>
                </div>
              </div>
              <div className="mc-engine-actions">
                <StatusChip tone={engine.is_local ? 'warn' : 'neutral'}>
                  {engine.is_local ? '本地' : '云端'}
                </StatusChip>
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

            {engine.is_local && (
              <Space.Compact style={{ width: '100%', maxWidth: 640 }}>
                <Input
                  size="small"
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

            {engine.id === 'glm' && (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                <span style={{ fontSize: 12, color: 'var(--mc-muted)' }}>端点</span>
                <Segmented
                  size="small"
                  value={glmEndpointFamily(engine.base_url)}
                  onChange={(v) => handleSetGlmEndpoint(v as 'std' | 'coding')}
                  disabled={!engine.enabled || settingGlmEndpoint}
                  options={[
                    { value: 'std', label: '标准（按量付费）' },
                    { value: 'coding', label: '编码套餐' },
                  ]}
                />
                <span style={{ fontSize: 11, color: 'var(--mc-muted)' }}>
                  编码套餐 Key 须走 /api/coding 端点；标准 Key 走 /api/paas
                </span>
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
