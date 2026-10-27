// EngineCard.tsx — 单引擎卡渲染（EngineSection 拆件，2026-10-27 瘦身重组：
// 537 行热点分流）。卡内自持 managedRunning（受管外部引擎 Strata/天璇 的实时
// 探活——停止态不再回显 engines.json 里的旧连接错误）；密钥行、GLM 端点、
// 地址框、自定义引擎编辑表单随卡内聚，编排态（editingCustomId/savedPrices）归父。
import { useState, type CSSProperties } from 'react'
import { Button, Input, Popconfirm, Segmented, Space, Switch } from 'antd'
import { StatusChip } from './ui'
import { engineColor, engineIcon, engineLabel, glmEndpointFamily, isCustomEngine, kindOf } from './utils'
import type { EngineConfig } from '../../api/engines'
import { useModelCenterActions, useModelCenterState } from './context'
import { LocalEngineLifecycle } from './LocalEngineLifecycle'
import { EditCustomEngineForm, type EnginePrices } from './CustomEngineForm'

const isManagedEngine = (id: string) => id === 'strata' || id === 'tianxuan'

export function EngineCard({ engine, editingCustom, savedPrice, onToggleEdit, onPriceSaved }: {
  engine: EngineConfig
  editingCustom: boolean
  savedPrice?: EnginePrices
  onToggleEdit: (engine: EngineConfig | null) => void
  onPriceSaved: (id: string, prices: EnginePrices) => void
}) {
  const {
    engineStatuses,
    activeEngine,
    testingEngine,
    savingEngine,
    editingURLs,
    deepseekKey,
    deepseekKeyMasked,
    glmKey,
    glmKeyMasked,
    opencodeGoKey,
    opencodeGoKeyMasked,
    opencodeZenKey,
    opencodeZenKeyMasked,
    settingGlmEndpoint,
  } = useModelCenterState()
  const {
    setEditingURLs,
    setDeepseekKeyState,
    setGlmKeyState,
    setOpencodeGoKeyState,
    setOpencodeZenKeyState,
    handleSetGlmEndpoint,
    handleTestConnection,
    handleRefreshModels,
    handleSaveURL,
    handleToggleEngine,
    handleSaveDeepseekKey,
    handleSaveGlmKey,
    handleSaveOpencodeGoKey,
    handleSaveOpencodeZenKey,
    handleRemoveCustomEngine,
    makeModels,
  } = useModelCenterActions()

  const [managedRunning, setManagedRunning] = useState<boolean | null>(null)
  const managedStopped = isManagedEngine(engine.id) && managedRunning === false

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

  const keyField = ([
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
  ] as {
    type: string
    value: string
    setter: (v: string) => void
    placeholder: string
    save: () => Promise<void>
  }[]).find(k => k.type === engine.type)

  const managedEngine = isManagedEngine(engine.id)
  const runningManagedModel = (engine.models || []).find(m => m.status === 'running')
  const managedModel = managedEngine
    ? (runningManagedModel?.name || runningManagedModel?.id || '未运行')
    : ''

  return (
    <div
      className={`mc-engine-card${engine.enabled ? ' is-enabled' : ''}`}
      style={{ '--engine-color': color } as CSSProperties}
    >
      <div className="mc-engine-head">
        <div className="mc-engine-id">
          <span className="mc-engine-icon">{engineIcon(engine)}</span>
          <div style={{ minWidth: 0 }}>
            <div className="mc-engine-name">{engine.name}</div>
            <div className="mc-engine-sub">
              {engineLabel(engine)} · {managedEngine ? '当前模型' : '默认模型'} {
                managedEngine
                  ? managedModel
                  : (engine.models || []).find(m => m.id === engine.default_model)?.name
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
            disabled={!engine.enabled || managedStopped}
          >
            测试连接
          </Button>
          {(engine.id === 'tianxuan' || engine.id === 'strata') && (
            <LocalEngineLifecycle
              engineId={engine.id}
              name={engine.id === 'tianxuan' ? '天璇' : 'Strata'}
              onReady={() => handleRefreshModels(engine.id)}
              onRunningChange={(running) => {
                setManagedRunning(prev => prev === running ? prev : running)
              }}
              externalConnected={engineStatuses[engine.id]?.connected}
            />
          )}
          <Button
            size="small"
            onClick={() => handleRefreshModels(engine.id)}
            loading={testingEngine === engine.id}
            disabled={!engine.enabled || managedStopped}
          >
            刷新
          </Button>
          {isCustomEngine(engine) && (
            <>
              <Button
                size="small"
                onClick={() => onToggleEdit(editingCustom ? null : engine)}
              >
                {editingCustom ? '收起' : '编辑'}
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

      {editingCustom && (
        <EditCustomEngineForm
          engine={engine}
          savedPrice={savedPrice}
          onSaved={onPriceSaved}
          onClose={() => onToggleEdit(null)}
        />
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

      {managedEngine ? (
        managedStopped ? (
          <div className="mc-engine-status">
            <span style={{ color: 'var(--mc-muted)' }}>
              引擎未运行 · 在「模型一览」逐模型启动（或用上方快捷启动档位）
            </span>
          </div>
        ) : null
      ) : engineStatuses[engine.id] && (
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
}
