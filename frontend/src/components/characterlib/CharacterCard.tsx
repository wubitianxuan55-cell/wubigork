// CharacterCard.tsx — 角色库「人物档案卡」
// 竖版设定卡：档案眉（编号/类型/可聊天）→ 立绘横幅 → 正文（名称/元数据/弧线/标签）→ 底部（雷达 + 操作）
import React from 'react'
import { Button, Popconfirm, Tag } from 'antd'
import {
  EditOutlined, SwapOutlined, DatabaseOutlined, DeleteOutlined, ReadOutlined,
} from '@ant-design/icons'
import TisorRadar from '../TisorRadar'
import type { LibraryCharacter } from '../../api/characterlib'
import { C } from '../../utils/theme'
import './character-card.css'

const KIND_META: Record<string, { label: string; color: string }> = {
  builtin: { label: '内置', color: 'gold' },
  custom: { label: '自定义', color: 'green' },
  assistant: { label: '助手', color: 'geekblue' },
}

const ROLE_LABELS: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '龙套',
}

const GENDER_LABELS: Record<string, string> = {
  female: '女性', male: '男性', neutral: '中性',
}

const STATUS_LABELS: Record<string, string> = {
  Alive: '存活', Dead: '已故', Missing: '失踪', Transformed: '变身',
}

interface CharacterCardProps {
  character: LibraryCharacter
  index: number
  inProject?: boolean
  isCurrentPersona?: boolean
  hasProject?: boolean
  onClick?: (c: LibraryCharacter) => void
  onEdit: (c: LibraryCharacter) => void
  onSetPersona: (c: LibraryCharacter) => void
  onMemory: (c: LibraryCharacter) => void
  onAssociate: (c: LibraryCharacter) => void
  onDissociate: (c: LibraryCharacter) => void
  onDelete: (c: LibraryCharacter) => void
}

export const CharacterCard: React.FC<CharacterCardProps> = ({
  character: c,
  index,
  inProject = false,
  isCurrentPersona = false,
  hasProject = false,
  onClick,
  onEdit,
  onSetPersona,
  onMemory,
  onAssociate,
  onDissociate,
  onDelete,
}) => {
  const km = KIND_META[c.kind] || KIND_META.custom
  const meta = [
    c.roleType ? ROLE_LABELS[c.roleType] || c.roleType : '',
    c.gender ? GENDER_LABELS[c.gender] || c.gender : '',
    c.age,
    c.status ? STATUS_LABELS[c.status] || c.status : '',
    inProject ? '已加入' : '',
  ].filter(Boolean).join(' · ')

  return (
    <div
      className="ccard"
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onClick={onClick ? () => onClick(c) : undefined}
      onKeyDown={onClick ? (e) => {
        if (e.currentTarget !== e.target) return
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick(c)
        }
      } : undefined}
    >
      {isCurrentPersona && <span className="ccard-current">当前人格</span>}

      <div className="ccard-head">
        <span className="ccard-no">NO.{String(index + 1).padStart(3, '0')}</span>
        <span className="ccard-head-right">
          <Tag color={km.color} className="ccard-kind-tag">{km.label}</Tag>
          {c.chatEnabled && (
            <span className="ccard-chat"><i className="ccard-chat-dot" />可聊天</span>
          )}
        </span>
      </div>

      <div className="ccard-portrait ccard-sheen">
        {c.portraitUrl ? (
          <img className="ccard-portrait-img" src={c.portraitUrl} alt={c.name} />
        ) : (
          <div className="ccard-placeholder"><span>{c.name.slice(0, 1) || '?'}</span></div>
        )}
      </div>

      <div className="ccard-body">
        <h3 className="ccard-name">{c.name}</h3>
        {meta && <p className="ccard-meta">{meta}</p>}
        {c.arc && <p className="ccard-arc">{c.arc}</p>}
        {c.tags && c.tags.length > 0 && (
          <div className="ccard-tags">
            {c.tags.slice(0, 3).map(t => (
              <span key={t} className="ccard-tag">#{t}</span>
            ))}
          </div>
        )}
      </div>

      <div className="ccard-foot">
        {c.chatEnabled && (
          <TisorRadar dims={c.dims} size={52} color="#f472b6" showLabels={false} />
        )}
        <div className="ccard-actions">
          <Button size="small" type="text" icon={<EditOutlined />} title="编辑"
            onClick={e => { e.stopPropagation(); onEdit(c) }} />
          <Button size="small" type="text" icon={<SwapOutlined />} title="设为当前聊天人格"
            disabled={!c.chatEnabled}
            onClick={e => { e.stopPropagation(); onSetPersona(c) }} />
          {c.chatEnabled && (
            <Button size="small" type="text" icon={<DatabaseOutlined />} title="查看状态 / 记忆 / 追踪"
              onClick={e => { e.stopPropagation(); onMemory(c) }} />
          )}
          {hasProject && (
            inProject ? (
              <Button size="small" type="text" icon={<ReadOutlined />} title="从当前项目移除"
                onClick={e => { e.stopPropagation(); onDissociate(c) }}>已加入</Button>
            ) : (
              <Button size="small" type="text" icon={<ReadOutlined />} title="加入当前项目"
                onClick={e => { e.stopPropagation(); onAssociate(c) }}>加入项目</Button>
            )
          )}
          <Popconfirm
            title={c.kind === 'builtin' ? `隐藏「${c.name}」？` : `删除「${c.name}」？删除会同时清理项目引用与聊天通道`}
            okText={c.kind === 'builtin' ? '隐藏' : '删除'} cancelText="取消"
            onConfirm={() => onDelete(c)}>
            <Button size="small" type="text" danger icon={<DeleteOutlined />} title="删除"
              onClick={e => e.stopPropagation()}
              style={{ color: C('color-text-secondary') }} />
          </Popconfirm>
        </div>
      </div>
    </div>
  )
}

export default CharacterCard
