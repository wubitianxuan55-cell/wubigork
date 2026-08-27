import React from 'react'
import { Card, Input, Space, Tag, Typography } from 'antd'
import { FileTextOutlined } from '@ant-design/icons'

const { Text } = Typography

export interface SrcRef { fileId?: string; fileName?: string; page?: number; snippet?: string }
export interface BidItem { name?: string; content?: string; sources?: SrcRef[] }

export interface BidSummaryShape {
  overview?: string
  duration?: string
  techScoring?: { name?: string; maxScore?: string; requirement?: string; sources?: SrcRef[] }[]
  keyRequirements?: string[]
  redLines?: string[]
  redLineItems?: BidItem[]
  qualification?: BidItem[]
  format?: BidItem[]
  darkRules?: BidItem[]
  overviewSources?: SrcRef[]
  durationSources?: SrcRef[]
  parseStatus?: string
  rawFiles?: { name?: string; markdown?: string; path?: string; size?: number }[]
}

function SrcChips({ sources, onJump }: { sources?: SrcRef[]; onJump: (s: SrcRef) => void }) {
  if (!sources?.length) return null
  return (
    <Space size={4} wrap style={{ marginTop: 4 }}>
      {sources.map((s, i) => (
        <Tag key={i} icon={<FileTextOutlined />} color="blue" style={{ cursor: 'pointer', fontSize: 11 }} onClick={() => onJump(s)}>
          {s.fileName || s.fileId || '来源'}{s.page ? ` · 第${s.page}页` : ''}
        </Tag>
      ))}
    </Space>
  )
}

function ItemList({ items, onJump }: { items?: BidItem[]; onJump: (s: SrcRef) => void }) {
  if (!items?.length) return null
  return (
    <div>
      {items.map((it, i) => (
        <div key={i} style={{ marginBottom: 8 }}>
          <Text strong>{it.name}</Text>
          {it.content ? <div style={{ fontSize: 13 }}>{it.content}</div> : null}
          <SrcChips sources={it.sources} onJump={onJump} />
        </div>
      ))}
    </div>
  )
}

export const ParseSummaryCards: React.FC<{
  bs: BidSummaryShape
  onChange: (bs: BidSummaryShape) => void
  onJump: (s: SrcRef) => void
}> = ({ bs, onChange, onJump }) => {
  const set = (patch: Partial<BidSummaryShape>) => onChange({ ...bs, ...patch })
  return (
    <div>
      <Card size="small" title="项目概况" style={{ marginBottom: 12 }}>
        <Input.TextArea value={bs.overview || ''} rows={3} onChange={e => set({ overview: e.target.value })} />
        <SrcChips sources={bs.overviewSources} onJump={onJump} />
      </Card>
      <Card size="small" title="工期要求" style={{ marginBottom: 12 }}>
        <Input value={bs.duration || ''} onChange={e => set({ duration: e.target.value })} />
        <SrcChips sources={bs.durationSources} onJump={onJump} />
      </Card>
      <Card size="small" title="评分标准" style={{ marginBottom: 12 }}>
        {(bs.techScoring || []).map((sc, i) => (
          <div key={i} style={{ marginBottom: 8 }}>
            <Space style={{ width: '100%' }}>
              <Input style={{ width: 180 }} value={sc.name || ''} placeholder="评分项" />
              <Input style={{ width: 80 }} value={sc.maxScore || ''} placeholder="分值" />
            </Space>
            <Input.TextArea value={sc.requirement || ''} rows={2} style={{ marginTop: 4 }} />
            <SrcChips sources={sc.sources} onJump={onJump} />
          </div>
        ))}
      </Card>
      <Card size="small" title="核心要求" style={{ marginBottom: 12 }}>
        {(bs.keyRequirements || []).map((kr, i) => (
          <div key={i} style={{ marginBottom: 4 }}>• {kr}</div>
        ))}
      </Card>
      <Card size="small" title="废标条款" style={{ marginBottom: 12 }}>
        <ItemList items={bs.redLineItems} onJump={onJump} />
        {!bs.redLineItems?.length && (bs.redLines || []).map((r, i) => <div key={i} style={{ marginBottom: 4 }}>• {r}</div>)}
      </Card>
      <Card size="small" title="资质要求" style={{ marginBottom: 12 }}>
        <ItemList items={bs.qualification} onJump={onJump} />
      </Card>
      <Card size="small" title="格式要求" style={{ marginBottom: 12 }}>
        <ItemList items={bs.format} onJump={onJump} />
      </Card>
      <Card size="small" title="暗标要求" style={{ marginBottom: 12 }}>
        <ItemList items={bs.darkRules} onJump={onJump} />
      </Card>
    </div>
  )
}
