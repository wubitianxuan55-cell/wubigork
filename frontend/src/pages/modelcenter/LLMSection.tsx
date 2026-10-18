import { useState } from 'react'
import { Button, Input } from 'antd'
import { CaretRightOutlined, CheckCircleOutlined, LoadingOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { EmptyState, ModelCard, SectionHead, StatusChip } from './ui'
import { capLabels, engineColor, engineLabel, filterModelsBySearch, formatCtx, formatPrice, glmAliasNote, modelAvailability, sortModelsPinnedFirst } from './utils'
import { useModelCenterActions, useModelCenterState } from './context'
import { usePinnedModels } from './modelPrefs'

export function LLMSection() {
  const {
    engines,
    llmModels,
    engineStatuses,
    testingEngine,
    savingEngine,
  } = useModelCenterState()

  const {
    handleTestConnection,
    handleRefreshModels,
    handleStartModel,
    isModelActive,
  } = useModelCenterActions()
  const [llmSearch, setLlmSearch] = useState('')
  const [pinned, togglePin] = usePinnedModels()

  const visibleEngines = engines.filter(e => e.enabled)
  const hasAny = llmModels.length > 0
  const hasMatch = filterModelsBySearch(llmModels, llmSearch).length > 0

  return (
    <section className="mc-section">
      <SectionHead
        icon={<ThunderboltOutlined />}
        title="语言模型"
        desc="按引擎浏览和切换聊天、办公、写作等使用的 LLM 模型"
        extra={(
          <Input.Search
            allowClear
            placeholder="搜索模型名称"
            value={llmSearch}
            onChange={e => setLlmSearch(e.target.value)}
            style={{ maxWidth: 320 }}
          />
        )}
      />

      {!hasAny ? (
        <EmptyState
          icon={<ThunderboltOutlined />}
          title="未发现语言模型"
          hint="请在「引擎管理」中启用引擎并刷新模型。"
        />
      ) : !hasMatch ? (
        <EmptyState
          compact
          title={`没有匹配「${llmSearch}」的模型`}
          hint="换个关键词试试"
        />
      ) : (
        visibleEngines.map(engine => {
          const baseModels = filterModelsBySearch(llmModels.filter(m => m.engineId === engine.id), llmSearch)
          const engineModels = sortModelsPinnedFirst(baseModels, pinned)
          if (engineModels.length === 0) return null
          const color = engineColor(engine)
          const status = engineStatuses[engine.id]
          const managedEngine = engine.id === 'strata' || engine.id === 'tianxuan'
          return (
            <div key={engine.id} className="mc-engine-group">
              <div className="mc-engine-group-head">
                <span className="mc-engine-group-name">
                  <i className="mc-engine-mark" style={{ background: color }} />
                  {engine.name}
                </span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                  <StatusChip>{engineModels.length} 个</StatusChip>
                  {status && (
                    managedEngine ? (
                      <StatusChip tone={status.connected ? 'ok' : 'neutral'} dot>
                        {status.connected ? '引擎运行中' : '引擎未运行'}
                      </StatusChip>
                    ) : (
                      <StatusChip tone={status.connected ? 'ok' : 'danger'} dot>
                        {status.connected ? '已连接' : '连接失败'}
                      </StatusChip>
                    )
                  )}
                  <Button
                    size="small"
                    onClick={() => handleTestConnection(engine.id)}
                    loading={testingEngine === engine.id}
                    style={{ fontSize: 11 }}
                  >
                    测试连接
                  </Button>
                  <Button
                    size="small"
                    onClick={() => handleRefreshModels(engine.id)}
                    loading={testingEngine === engine.id}
                    style={{ fontSize: 11 }}
                  >
                    刷新模型
                  </Button>
                </span>
              </div>

              <div className="mc-grid">
                {engineModels.map(card => {
                  const active = isModelActive(card)
                  // 天璇/Strata 的模型目录在引擎未运行时依然可启动（启动模型会
                  // 拉起/切换引擎），不能像云端引擎那样按 disconnected 全禁用。
                  const avail = managedEngine
                    ? (engine.enabled ? (card.status === 'running' ? 'ready' : 'stopped') : 'disabled')
                    : modelAvailability(card, engine.enabled, status?.connected)
                  const blocked = avail === 'disabled' || (!managedEngine && avail === 'disconnected')
                  const managedStarting = managedEngine && savingEngine === engine.id
                  const loading = managedStarting
                  const loadingText = managedStarting ? '启动中…' : ''
                  // coding 端点套餐旧名自动切换注记（后端 alias_of，std 家族为空）
                  const aliasNote = glmAliasNote((engine.models || []).find(m => m.id === card.modelId))
                  // B 刀：模型元数据徽标（上下文/能力/价格；meta 缺失时不渲染不占位）
                  const meta = card.meta
                  const ctxText = formatCtx(meta?.context_length)
                  const priceText = formatPrice(meta)
                  const statusText = loading
                    ? loadingText
                    : active
                      ? '运行中'
                      : avail === 'disconnected'
                        ? '未连接'
                        : card.status === 'running'
                          ? '运行中'
                          : card.status === 'stopped'
                            ? '未启动'
                            : '就绪'
                  return (
                    <ModelCard
                      key={card.modelId}
                      name={card.modelName}
                      engineId={card.engineId}
                      kindChip={<StatusChip tone={active ? 'ok' : 'neutral'}>{engineLabel(card)}</StatusChip>}
                      chips={[
                        card.modelId === engine.default_model
                          ? <StatusChip key="default" tone="accent">默认</StatusChip>
                          : null,
                        pinned.includes(card.modelId)
                          ? <StatusChip key="pin" tone="warn">置顶</StatusChip>
                          : null,
                        aliasNote
                          ? <StatusChip key="alias" tone="accent" title={aliasNote}>自动切换</StatusChip>
                          : null,
                        avail === 'disconnected'
                          ? <StatusChip key="off" tone="danger">未连接</StatusChip>
                          : avail === 'stopped'
                            ? <StatusChip key="stop" tone="warn">未启动</StatusChip>
                            : null,
                        ctxText
                          ? <StatusChip key="ctx" title="上下文长度">{ctxText}</StatusChip>
                          : null,
                        ...(meta?.caps || []).map(cap => (
                          <StatusChip key={`cap-${cap}`} title={`能力：${capLabels[cap] || cap}`}>
                            {capLabels[cap] || cap}
                          </StatusChip>
                        )),
                        priceText
                          ? (
                            <StatusChip
                              key="price"
                              tone={meta?.free ? 'ok' : 'neutral'}
                              title={meta?.price_note || undefined}
                            >
                              {priceText}
                            </StatusChip>
                          )
                          : null,
                      ].filter(Boolean)}
                      active={active}
                      dimmed={blocked}
                      pinned={pinned.includes(card.modelId)}
                      onTogglePin={() => togglePin(card.modelId)}
                      status={{
                        tone: loading
                          ? 'accent'
                          : active || card.status === 'running'
                            ? 'ok'
                            : avail === 'disconnected'
                              ? 'danger'
                              : card.status === 'stopped'
                                ? 'warn'
                                : 'neutral',
                        text: statusText,
                      }}
                      action={(
                        <Button
                          type={active ? 'default' : 'primary'}
                          size="small"
                          icon={loading ? <LoadingOutlined /> : active ? <CheckCircleOutlined /> : <CaretRightOutlined />}
                          onClick={() => handleStartModel(card)}
                          disabled={active || blocked || loading}
                        >
                          {loading ? loadingText : active ? '已启动' : '启动'}
                        </Button>
                      )}
                    />
                  )
                })}
              </div>
            </div>
          )
        })
      )}
    </section>
  )
}
