import type { ReactNode } from 'react'
import { useState } from 'react'
import { Button, Input } from 'antd'
import { CheckCircleOutlined, FileTextOutlined, NumberOutlined, SearchOutlined } from '@ant-design/icons'
import { EmptyState, ModelCard, SectionHead, StatusChip, type StatusTone } from './ui'
import { engineLabel, filterModelsBySearch, formatCtx, formatPrice, kindOf, modelAvailability, sortModelsPinnedFirst } from './utils'
import { useModelCenter } from './context'
import { usePinnedModels } from './modelPrefs'

const KIND_META: Record<string, { label: string; icon: ReactNode; tone: StatusTone }> = {
  embedding: { label: 'Embedding', icon: <NumberOutlined />, tone: 'accent' },
  rerank: { label: 'Rerank', icon: <SearchOutlined />, tone: 'warn' },
  ocr: { label: 'OCR', icon: <FileTextOutlined />, tone: 'ok' },
}

export function SpecialtySection() {
  const { specialtyModels, ocrCfg, handleSetOCRModel, engines, engineStatuses } = useModelCenter()
  const [search, setSearch] = useState('')
  const [pinned, togglePin] = usePinnedModels()
  const models = sortModelsPinnedFirst(filterModelsBySearch(specialtyModels, search), pinned)

  return (
    <section className="mc-section">
      <SectionHead
        icon={<FileTextOutlined />}
        title="专业模型"
        desc="Embedding / Rerank 用于本地检索，OCR 模型可设为办公「提取文字」的默认通道"
        extra={ocrCfg.model && (
          <Button size="small" onClick={() => handleSetOCRModel('', '')}>恢复自动选择</Button>
        )}
      />

      {specialtyModels.length === 0 ? (
        <EmptyState
          icon={<SearchOutlined />}
          title="暂无 Embedding / Rerank / OCR 模型"
          hint="在「引擎管理」中刷新本地 Herdsman 模型后，这里会显示 bge-m3、bge-reranker-v2-m3、PaddleOCR 与 MinerU"
        />
      ) : (
        <>
          <Input.Search
            allowClear
            placeholder="搜索专业模型"
            value={search}
            onChange={e => setSearch(e.target.value)}
            style={{ maxWidth: 320 }}
          />
          {models.length === 0 ? (
            <EmptyState compact title={`没有匹配「${search}」的模型`} hint="换个关键词试试" />
          ) : (
            <div className="mc-grid">
              {models.map(m => {
                const kind = kindOf(m)
                const meta = KIND_META[kind] || { label: kind, icon: <FileTextOutlined />, tone: 'neutral' as StatusTone }
                const isOCR = kind === 'ocr'
                const activeOCR = isOCR && ocrCfg.engine === m.engineId && ocrCfg.model === m.modelId
                const eng = engines.find(e => e.id === m.engineId)
                const avail = modelAvailability(m, eng?.enabled ?? true, engineStatuses[m.engineId]?.connected)
                const blocked = avail === 'disconnected' || avail === 'disabled'
                const statusText = activeOCR
                  ? '当前 OCR'
                  : avail === 'disconnected'
                    ? '未连接'
                    : m.status === 'running'
                      ? '运行中'
                      : m.status === 'stopped'
                        ? '未启动'
                        : '就绪'
                // B 刀：模型元数据徽标（上下文/价格；embedding/rerank/ocr 目录有官方价，meta 缺失时不占位）
                const ctxText = formatCtx(m.meta?.context_length)
                const priceText = formatPrice(m.meta)
                return (
                  <ModelCard
                    key={`${m.engineId}:${m.modelId}`}
                    name={m.modelName}
                    engineId={m.engineId}
                    engineName={engineLabel(m)}
                    kindChip={(
                      <StatusChip tone={meta.tone}>
                        {meta.icon} {meta.label}
                      </StatusChip>
                    )}
                    chips={[
                      pinned.includes(m.modelId)
                        ? <StatusChip key="pin" tone="warn">置顶</StatusChip>
                        : null,
                      avail === 'disconnected'
                        ? <StatusChip key="off" tone="danger">未连接</StatusChip>
                        : avail === 'stopped'
                          ? <StatusChip key="stop" tone="warn">未启动</StatusChip>
                          : null,
                      ctxText
                        ? <StatusChip key="ctx" title="上下文长度">{ctxText}</StatusChip>
                        : null,
                      priceText
                        ? (
                          <StatusChip
                            key="price"
                            tone={m.meta?.free ? 'ok' : 'neutral'}
                            title={m.meta?.price_note || undefined}
                          >
                            {priceText}
                          </StatusChip>
                        )
                        : null,
                    ].filter(Boolean)}
                    active={activeOCR}
                    dimmed={blocked}
                    pinned={pinned.includes(m.modelId)}
                    onTogglePin={() => togglePin(m.modelId)}
                    status={{
                      tone: activeOCR || m.status === 'running'
                        ? 'ok'
                        : avail === 'disconnected'
                          ? 'danger'
                          : m.status === 'stopped'
                            ? 'warn'
                            : 'neutral',
                      text: statusText,
                    }}
                    action={isOCR && (
                      <Button
                        size="small"
                        type={activeOCR ? 'primary' : 'default'}
                        icon={activeOCR ? <CheckCircleOutlined /> : <FileTextOutlined />}
                        onClick={() => handleSetOCRModel(activeOCR ? '' : m.engineId, activeOCR ? '' : m.modelId)}
                        disabled={blocked && !activeOCR}
                      >
                        {activeOCR ? '当前 OCR' : '设为 OCR'}
                      </Button>
                    )}
                  />
                )
              })}
            </div>
          )}
        </>
      )}
    </section>
  )
}
