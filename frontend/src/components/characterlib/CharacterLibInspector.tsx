// CharacterLibInspector.tsx — 角色档案库「档案详情」inspector（右侧 v3-panel，可折叠）
// 只读速览：立绘 / 身份徽标 / 设定摘要 / 记忆入口；操作复用页面既有处理器（编辑→抽屉弹窗、记忆→状态面板等）。
// 未选中档案时展示优雅空态。
import React from 'react'
import { Button, Popconfirm } from 'antd'
import {
  IdcardOutlined, LeftOutlined, RightOutlined, EditOutlined, SwapOutlined,
  DatabaseOutlined, ReadOutlined, DeleteOutlined, InboxOutlined,
} from '@ant-design/icons'
import type { LibraryCharacter } from '../../api/characterlib'
import { PortraitImg } from './PortraitImg'
import TisorRadar from '../TisorRadar'
import { C } from '../../utils/theme'

const KIND_META: Record<string, string> = {
  builtin: '内置', custom: '自定义', assistant: '助手',
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

interface Props {
  character: LibraryCharacter | null
  index: number
  inProject?: boolean
  isCurrentPersona?: boolean
  hasProject?: boolean
  collapsed: boolean
  onToggleCollapse: () => void
  onEdit: (c: LibraryCharacter) => void
  onSetPersona: (c: LibraryCharacter) => void
  onMemory: (c: LibraryCharacter) => void
  onAssociate: (c: LibraryCharacter) => void
  onDissociate: (c: LibraryCharacter) => void
  onDelete: (c: LibraryCharacter) => void
}

const CharacterLibInspector: React.FC<Props> = ({
  character: c, index, inProject = false, isCurrentPersona = false, hasProject = false,
  collapsed, onToggleCollapse, onEdit, onSetPersona, onMemory, onAssociate, onDissociate, onDelete,
}) => {
  if (collapsed) {
    return (
      <aside className="clib-inspector v3-panel is-collapsed" aria-label="档案详情检查器（已折叠）">
        <button type="button" className="clib-inspector-reopen" onClick={onToggleCollapse}
          title="展开档案详情" aria-label="展开档案详情">
          <LeftOutlined aria-hidden />
          <span className="clib-inspector-reopen-label">档案详情</span>
        </button>
      </aside>
    )
  }

  const kindLabel = c ? KIND_META[c.kind] || KIND_META.custom : ''
  const meta = c
    ? [
        c.roleType ? ROLE_LABELS[c.roleType] || c.roleType : '',
        c.gender ? GENDER_LABELS[c.gender] || c.gender : '',
        c.age,
        c.status ? STATUS_LABELS[c.status] || c.status : '',
      ].filter(Boolean).join(' · ')
    : ''

  const field = (label: string, value?: string) => (
    value && value.trim() ? (
      <div className="clib-insp-field">
        <span className="clib-insp-label">{label}</span>
        <p className="clib-insp-text">{value}</p>
      </div>
    ) : null
  )

  const tags = c?.tags ?? []
  const samples = (c?.dialogueSamples ?? []).filter(Boolean)
  const hasDetail = !!(c && (
    c.personality || c.background || c.arc || c.appearance || c.figure || c.motivation
    || c.voiceGuide || c.behaviorRules || c.emotionLogic || c.notes || tags.length > 0 || samples.length > 0
  ))

  return (
    <aside className="clib-inspector v3-panel" aria-label="档案详情检查器">
      <div className="v3-panel-head clib-inspector-head">
        <IdcardOutlined className="clib-side-head-icon" aria-hidden />
        <span className="v3-panel-title">档案详情</span>
        {c && index >= 0 && <span className="clib-insp-no">NO.{String(index + 1).padStart(3, '0')}</span>}
        <span className="v3-panel-spacer" />
        <button type="button" className="clib-inspector-fold" onClick={onToggleCollapse}
          title="收起详情" aria-label="收起档案详情">
          <RightOutlined aria-hidden />
        </button>
      </div>

      <div className="clib-inspector-body">
        {!c ? (
          <div className="clib-insp-empty">
            <div className="clib-insp-empty-icon"><IdcardOutlined /></div>
            <p className="clib-insp-empty-title">未选中档案</p>
            <p className="clib-insp-empty-desc">点击左侧档案卡，在此查看立绘 / 详情 / 记忆。</p>
          </div>
        ) : (
          <>
            {/* 立绘 + 身份 */}
            <div className="clib-insp-hero">
              {c.portraitUrl ? (
                <PortraitImg className="clib-insp-hero-img" src={c.portraitUrl} alt={c.name} />
              ) : (
                <div className="clib-insp-hero-ph">{c.name.slice(0, 1) || '?'}</div>
              )}
              <div className="clib-insp-hero-info">
                <h3 className="clib-insp-name">{c.name}</h3>
                {meta && <p className="clib-insp-meta">{meta}</p>}
                <div className="clib-insp-badges">
                  {c.chatEnabled && (
                    <span className="clib-insp-badge is-chat"><i className="clib-insp-dot" aria-hidden />可聊天</span>
                  )}
                  {isCurrentPersona && <span className="clib-insp-badge is-persona">当前人格</span>}
                  {inProject && <span className="clib-insp-badge is-project">已加入项目</span>}
                </div>
                <span className="clib-insp-kind">{kindLabel}</span>
              </div>
            </div>

            {/* 快速操作 */}
            <div className="clib-insp-actions">
              <Button size="small" type="primary" ghost icon={<EditOutlined />} onClick={() => onEdit(c)}>编辑</Button>
              <Button size="small" icon={<SwapOutlined />} disabled={!c.chatEnabled}
                onClick={() => onSetPersona(c)}
                title={c.chatEnabled ? '设为当前聊天人格' : '该角色未开启聊天通道'}>设为人格</Button>
              {c.chatEnabled && (
                <Button size="small" icon={<DatabaseOutlined />} onClick={() => onMemory(c)}>状态 / 记忆</Button>
              )}
              {inProject ? (
                hasProject && (
                  <Button size="small" icon={<ReadOutlined />} onClick={() => onDissociate(c)}>移出项目</Button>
                )
              ) : (
                <Button size="small" icon={<ReadOutlined />} onClick={() => onAssociate(c)}>加入项目</Button>
              )}
              <Popconfirm
                title={c.kind === 'builtin' ? `隐藏「${c.name}」？` : `删除「${c.name}」？删除会同时清理项目引用与聊天通道`}
                okText={c.kind === 'builtin' ? '隐藏' : '删除'} cancelText="取消"
                onConfirm={() => onDelete(c)}>
                <Button size="small" icon={<DeleteOutlined />} danger style={{ color: C('color-text-secondary') }}>删除</Button>
              </Popconfirm>
            </div>

            {/* 设定摘要 */}
            <div className="clib-insp-sec">
              <h4 className="clib-insp-sec-title">档案摘要</h4>
              {hasDetail ? (
                <>
                  {field('性格', c.personality)}
                  {field('背景', c.background)}
                  {field('角色弧线', c.arc)}
                  {field('外貌', c.appearance)}
                  {field('身材体型', c.figure)}
                  {field('动机', c.motivation)}
                  {field('口吻指南', c.voiceGuide)}
                  {field('行为规则', c.behaviorRules)}
                  {field('情感逻辑', c.emotionLogic)}
                  {field('备注', c.notes)}
                  {tags.length > 0 && (
                    <div className="clib-insp-field">
                      <span className="clib-insp-label">标签</span>
                      <div className="clib-insp-tags">
                        {tags.slice(0, 8).map(t => <span key={t} className="clib-insp-tag">#{t}</span>)}
                      </div>
                    </div>
                  )}
                  {samples.length > 0 && (
                    <div className="clib-insp-field">
                      <span className="clib-insp-label">对话样本</span>
                      <p className="clib-insp-text clib-insp-quote">{samples.slice(0, 2).join('　')}</p>
                    </div>
                  )}
                </>
              ) : (
                <p className="clib-insp-text clib-insp-mem-hint">
                  <InboxOutlined aria-hidden /> 暂无设定内容，点击「编辑」补全，或使用「一键补齐」。
                </p>
              )}
              {c.chatEnabled && c.dims && (
                <div className="clib-insp-radar">
                  <TisorRadar dims={c.dims} size={124} color="var(--gaea-glow)" />
                </div>
              )}
            </div>

            {/* 记忆入口 */}
            <div className="clib-insp-sec">
              <h4 className="clib-insp-sec-title">记忆</h4>
              {c.chatEnabled ? (
                <p className="clib-insp-text clib-insp-mem-hint">
                  <InboxOutlined aria-hidden /> 聊天状态 / 记忆事实 / 行为追踪请打开「状态 / 记忆」面板查看。
                </p>
              ) : (
                <p className="clib-insp-text clib-insp-mem-hint">
                  <InboxOutlined aria-hidden /> 该角色未开启聊天通道，暂无记忆数据。
                </p>
              )}
            </div>
          </>
        )}
      </div>
    </aside>
  )
}

export default CharacterLibInspector
