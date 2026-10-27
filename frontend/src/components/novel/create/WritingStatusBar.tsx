import React, { useState } from 'react'
import { Alert, Button } from 'antd'
import type { AiTasteResult } from './chapterStreamTypes'
import type { DeclGateState } from '../../../pages/create/useCreateBanners'

// WritingStatusBar — 创作页「写作状态条」（2026-10-27 大道至简：四条横幅
// 收敛为一条）。契约：
// 1. **全部条目恒在 DOM**（data-testid 与原横幅逐字保留——CreatePage.test 的
//    行为锁零改动；jsdom 不渲染 CSS，display:none 不影响查询）。
// 2. 视觉默认只显**最高优先级**一条：门禁未过(error) > 阶段收尾(CTA) >
//    AI 味检测 > 门禁已过 > 采样提示；error 级恒居首不被折叠吞掉。
// 3. 条目 >1 时显示「N 条反馈」展开/收起全部。
interface WritingStatusBarProps {
  samplerNote: string
  stageClosing: { stage: number } | null
  declGate: DeclGateState | null
  aiTaste: AiTasteResult | null
  onRunHealth: () => void
}

export function WritingStatusBar({ samplerNote, stageClosing, declGate, aiTaste, onRunHealth }: WritingStatusBarProps) {
  const [expanded, setExpanded] = useState(false)

  const items: { id: string; node: React.ReactNode }[] = []
  if (declGate && declGate.declared && !declGate.accepted) {
    items.push({
      id: 'decl-gate-banner',
      node: (
        <Alert
          type="error"
          showIcon
          data-testid="decl-gate-banner"
          message="变更声明未过章级门禁——声明的状态/伏笔更新未落库"
          description={(
            <div>
              {declGate.issues.map((it, i) => <div key={i}>{it}</div>)}
              <div style={{ marginTop: 4, opacity: 0.8 }}>正文已正常保存；可重新生成，或在定稿时手动修正后确认。</div>
            </div>
          )}
        />
      ),
    })
  }
  if (stageClosing) {
    items.push({
      id: 'stage-closing-banner',
      node: (
        <Alert
          type="info"
          showIcon
          data-testid="stage-closing-banner"
          message={`第 ${stageClosing.stage} 阶段（每 10 章一阶段）收尾——建议跑一次全书体检`}
          description="体检零 LLM 纯确定性：逐章质量/AI 味/伏笔 Lint 一份报告，帮你确认本阶段没有留下悬空伏笔或质量塌方。"
          action={
            <Button size="small" type="primary" data-testid="stage-closing-run" onClick={onRunHealth}>
              立即体检
            </Button>
          }
        />
      ),
    })
  }
  if (aiTaste) {
    items.push({
      id: 'ai-taste',
      node: (
        <Alert
          type={aiTaste.score >= 60 ? 'warning' : aiTaste.score >= 35 ? 'info' : 'success'}
          showIcon
          data-testid="ai-taste"
          message={`AI 味检测 ${aiTaste.score} 分${aiTaste.score >= 60 ? '（建议去味）' : ''}`}
          description={aiTaste.deSlop?.beforeScore != null && aiTaste.deSlop.afterScore != null
            ? `已去味 ${(aiTaste.deSlop.changes ?? []).length} 处，分数 ${aiTaste.deSlop.beforeScore}→${aiTaste.deSlop.afterScore}`
            : aiTaste.issues.length > 0
              ? [...new Set(aiTaste.issues.map(i => i.reason))].slice(0, 3).join('；')
              : '未命中明显 AI 套路'}
        />
      ),
    })
  }
  if (declGate && declGate.declared && declGate.accepted) {
    items.push({
      id: 'decl-gate-ok',
      node: (
        <Alert type="success" showIcon data-testid="decl-gate-ok"
          message="变更声明已过章级门禁——定稿时将零成本预填三件套草案" />
      ),
    })
  }
  if (samplerNote) {
    items.push({
      id: 'sampler-note',
      node: <Alert type="success" showIcon data-testid="sampler-note" message={samplerNote} />,
    })
  }

  if (items.length === 0) return null

  return (
    <div className="novel-create-statusbar" data-count={items.length} data-expanded={expanded ? '1' : '0'}>
      {items.map((it, i) => (
        <div
          key={it.id}
          className={`novel-create-statusbar-item${i === 0 || expanded ? ' is-visible' : ''}`}
          style={i === 0 || expanded ? undefined : { display: 'none' }}
        >
          {it.node}
        </div>
      ))}
      {items.length > 1 && (
        <Button
          size="small"
          type="text"
          className="novel-create-statusbar-toggle"
          aria-expanded={expanded}
          onClick={() => setExpanded(v => !v)}
        >
          {expanded ? '收起反馈' : `${items.length} 条反馈`}
        </Button>
      )}
    </div>
  )
}
