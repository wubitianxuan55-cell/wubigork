import type { ReactNode } from 'react'
import { Button, Card, Tag, Typography } from 'antd'
import { CheckCircleOutlined, FileTextOutlined, NumberOutlined, SearchOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { engineColor, engineLabel, kindOf } from './utils'
import { useModelCenter } from './context'

const KIND_META: Record<string, { label: string; icon: ReactNode }> = {
  embedding: { label: 'Embedding', icon: <NumberOutlined /> },
  rerank: { label: 'Rerank', icon: <SearchOutlined /> },
  ocr: { label: 'OCR', icon: <FileTextOutlined /> },
}

export function SpecialtySection() {
  const { specialtyModels, ocrCfg, handleSetOCRModel } = useModelCenter()

  if (specialtyModels.length === 0) {
    return (
      <div className="mc-empty">
        <SearchOutlined className="mc-empty-icon" />
        <Typography.Text style={{ color: C('color-text'), fontSize: 13 }}>暂无 Embedding / Rerank / OCR 模型</Typography.Text>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 6 }}>
          在「引擎管理」中刷新本地 Herdsman 模型后，这里会显示 bge-m3、bge-reranker-v2-m3、PaddleOCR 与 MinerU
        </Typography.Text>
      </div>
    )
  }

  return (
    <section className="mc-section">
      <div className="mc-section-head">
        <div>
          <div className="mc-section-title"><FileTextOutlined /> 专业模型</div>
          <div className="mc-section-desc">Embedding / Rerank 用于本地检索，OCR 模型可设为办公「提取文字」的默认通道</div>
        </div>
        {ocrCfg.model && (
          <Button size="small" onClick={() => handleSetOCRModel('', '')}>恢复自动选择</Button>
        )}
      </div>
      <div className="mc-grid">
        {specialtyModels.map(m => {
          const kind = kindOf(m)
          const meta = KIND_META[kind] || { label: kind, icon: <FileTextOutlined /> }
          const color = engineColor(m)
          const isOCR = kind === 'ocr'
          const activeOCR = isOCR && ocrCfg.engine === m.engineId && ocrCfg.model === m.modelId
          return (
            <Card key={`${m.engineId}:${m.modelId}`} size="small" className={`mc-model-card${activeOCR ? ' is-active' : ''}`}>
              <div className="mc-model-name">{m.modelName}</div>
              <div className="mc-model-meta">
                <Tag color={color} style={{ fontSize: 10, margin: 0 }}>{engineLabel(m)}</Tag>
                <Tag color={kind === 'ocr' ? 'gold' : kind === 'rerank' ? 'geekblue' : 'cyan'} style={{ fontSize: 10, margin: 0 }}>
                  {meta.icon} {meta.label}
                </Tag>
              </div>
              <div className="mc-model-foot">
                <span className="mc-status">
                  <i className={`mc-status-dot ${m.status === 'running' ? 'is-running' : ''}`} />
                  {m.status === 'running' ? '运行中' : m.status === 'stopped' ? '已停止' : '就绪'}
                </span>
                {isOCR && (
                  <Button
                    size="small"
                    type={activeOCR ? 'primary' : 'default'}
                    icon={activeOCR ? <CheckCircleOutlined /> : <FileTextOutlined />}
                    onClick={() => handleSetOCRModel(activeOCR ? '' : m.engineId, activeOCR ? '' : m.modelId)}
                    style={{ fontSize: 11, marginLeft: 'auto' }}
                  >
                    {activeOCR ? '当前 OCR' : '设为 OCR'}
                  </Button>
                )}
              </div>
            </Card>
          )
        })}
      </div>
    </section>
  )
}
