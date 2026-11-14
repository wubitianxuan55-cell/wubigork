/**
 * schedule/gantt/InplaceEditField.tsx — 统一就地编辑弹层（W5，斑马 `ZF*` 族对标）
 *
 * 一个构造函数盖住四类值（date / dateRange / number / text）：网格里双击单元格
 * 弹的都是这一族，而不是每个单元格自造控件——一致性来自单一实现，不是逐格复写。
 *
 * 手感按 §15.2 的 6 步固定流程落地（RE 证据：`ZQtInplaceEdit` + `editingFinished`）：
 *  1. padding:0（inplace.css，保证不撑高行）；
 *  2. 继承单元格字体（font: inherit，12px）；
 *  3. 左对齐 + 垂直居中；
 *  4. 打开即全选（`select()`，直接覆盖输入）；
 *  5. Enter 提交 / Esc 取消 / 失焦提交（≈ editingFinished）；
 *  6. 提交前一律过 `inplaceEdit.parseInplace` 校验：非法**原位提示且不写回**。
 *
 * 组件只管草稿与焦点；值语义（夹取/截断/清空/区间开闭）全在 inplaceEdit.ts。
 */
import React, { useEffect, useRef, useState } from 'react'
import { DatePicker, Input, InputNumber, Popover } from 'antd'
import dayjs from 'dayjs'
import './inplace.css'
import {
  parseInplace,
  type InplaceDateRangeValue,
  type InplaceKind,
  type InplaceRaw,
  type InplaceSpec,
} from '../inplaceEdit'

/** 弹层提交值（已规范化：null=清空；dateRange 用 from/to，单侧 null=开区间） */
export type InplaceCommitValue = string | number | null | InplaceDateRangeValue

export interface InplaceEditFieldProps {
  /** 值种类（列类型决定，调用方从 inplaceKindOfCol 取） */
  kind: InplaceKind
  /** 当前值（date/text/number=显示文本；dateRange={from,to}） */
  value: string | InplaceDateRangeValue
  /** 列规格（min/max/step/maxLen/emptyAs/required） */
  spec?: InplaceSpec
  /** 校验通过后写回（value 已规范化；拒绝写入时不会调用） */
  onSubmit: (value: InplaceCommitValue) => void
  /** 取消（Esc / 点外部）；不写回 */
  onCancel?: () => void
  /** 打开方式（表格单元格=双击；表头筛选用 click 或受控 open） */
  trigger?: 'dblclick' | 'click'
  /** 受控开关（不传=内部自管） */
  open?: boolean
  onOpenChange?: (open: boolean) => void
  /** 打开即全选（缺省 true，§15.2 第 4 步） */
  autoSelect?: boolean
  /** 锚点占满父容器（数值/文本控件原本 width:100% 的单元格用，避免弹层锚点收窄布局） */
  fullWidth?: boolean
  /** 测试/埋点用 testid 前缀（弹层容器=前缀，提示=前缀-error） */
  testId?: string
  /** 触发元素（单元格内容） */
  children: React.ReactElement
}

interface Draft {
  text: string
  from: string
  to: string
}

function draftOf(kind: InplaceKind, value: string | InplaceDateRangeValue): Draft {
  if (kind === 'dateRange') {
    const v = typeof value === 'string' ? { from: value, to: '' } : value
    return { text: '', from: v.from ?? '', to: v.to ?? '' }
  }
  return { text: typeof value === 'string' ? value : '', from: '', to: '' }
}

/** 草稿指纹：受控父组件每次 render 新建对象也不会把 effect 打成死循环 */
function valueKeyOf(kind: InplaceKind, value: string | InplaceDateRangeValue): string {
  if (kind === 'dateRange') {
    const v = typeof value === 'string' ? { from: value, to: '' } : value
    return `r:${v.from ?? ''}|${v.to ?? ''}`
  }
  return `v:${typeof value === 'string' ? value : ''}`
}

export const InplaceEditField: React.FC<InplaceEditFieldProps> = ({
  kind,
  value,
  spec,
  onSubmit,
  onCancel,
  trigger = 'dblclick',
  open: openProp,
  onOpenChange,
  autoSelect = true,
  fullWidth = false,
  testId = 'sched-inplace',
  children,
}) => {
  const [innerOpen, setInnerOpen] = useState(false)
  const controlled = openProp !== undefined
  const shown = controlled ? openProp : innerOpen
  const rootRef = useRef<HTMLDivElement>(null)
  const valueKey = valueKeyOf(kind, value)
  const [draft, setDraft] = useState<Draft>(() => draftOf(kind, value))
  const [error, setError] = useState('')

  const requestOpen = (next: boolean) => {
    if (!controlled) setInnerOpen(next)
    onOpenChange?.(next)
  }

  // 打开时把草稿重置为当前值（不沿用上次未提交的输入），并清掉上次的报错。
  // value 与 valueKey 同时入依赖：key 只用来挡住「父组件每轮新建对象」造成的空转
  const openedKeyRef = useRef('')
  const openedKey = `${shown ? 'open' : 'closed'}:${kind}:${valueKey}`
  useEffect(() => {
    if (!shown) return
    if (openedKeyRef.current === openedKey) return
    openedKeyRef.current = openedKey
    setDraft(draftOf(kind, value))
    setError('')
  }, [shown, kind, valueKey, value, openedKey])

  // 打开即聚焦 + 全选（§15.2 第 4 步）
  useEffect(() => {
    if (!shown) return
    const el = rootRef.current?.querySelector('input') as HTMLInputElement | null
    if (!el) return
    el.focus()
    if (!autoSelect) return
    try {
      el.select()
    } catch {
      // 非文本输入（如无选区能力的控件）忽略
    }
  }, [shown, autoSelect, valueKey])

  const commit = () => {
    let raw: InplaceRaw
    if (kind === 'dateRange') raw = { kind: 'dateRange', from: draft.from, to: draft.to }
    else if (kind === 'date') raw = { kind: 'date', text: draft.text }
    else if (kind === 'number') raw = { kind: 'number', text: draft.text }
    else raw = { kind: 'text', text: draft.text }
    const r = parseInplace(raw, spec)
    if (!r.ok) {
      // 非法：原位提示，不写回、不关闭（用户可改）
      setError(r.message)
      return
    }
    setError('')
    onSubmit(r.value)
    requestOpen(false)
  }

  const cancel = () => {
    setError('')
    requestOpen(false)
    onCancel?.()
  }

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      e.stopPropagation()
      commit()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      cancel()
    }
  }

  /** 失焦提交（≈ editingFinished）：焦点落到日历/下拉 portal 或弹层内部时不算离开 */
  const onBlur = (e: React.FocusEvent) => {
    const next = e.relatedTarget as HTMLElement | null
    if (next && rootRef.current?.contains(next)) return
    if (next && typeof next.closest === 'function' && next.closest('.ant-picker-dropdown, .ant-select-dropdown, .ant-dropdown')) return
    commit()
  }

  const editor = (() => {
    if (kind === 'date') {
      return (
        <DatePicker
          size="small"
          format="YYYY-MM-DD"
          value={draft.text ? dayjs(draft.text) : null}
          onChange={(d) => setDraft((p) => ({ ...p, text: d ? d.format('YYYY-MM-DD') : '' }))}
          allowClear
          placeholder="YYYY-MM-DD"
          data-testid={`${testId}-input`}
        />
      )
    }
    if (kind === 'dateRange') {
      return (
        <DatePicker.RangePicker
          size="small"
          format="YYYY-MM-DD"
          value={[draft.from ? dayjs(draft.from) : null, draft.to ? dayjs(draft.to) : null]}
          onChange={(v) =>
            setDraft((p) => ({
              ...p,
              from: v?.[0] ? v[0].format('YYYY-MM-DD') : '',
              to: v?.[1] ? v[1].format('YYYY-MM-DD') : '',
            }))
          }
          allowEmpty={[true, true]}
          data-testid={`${testId}-input`}
        />
      )
    }
    if (kind === 'number') {
      const n = draft.text.trim() === '' ? null : Number(draft.text)
      return (
        <InputNumber
          size="small"
          value={n !== null && Number.isFinite(n) ? n : null}
          onChange={(v) => setDraft((p) => ({ ...p, text: v === null ? '' : String(v) }))}
          min={spec?.min}
          max={spec?.max}
          step={spec?.step}
          style={{ width: '100%' }}
          data-testid={`${testId}-input`}
        />
      )
    }
    return (
      <Input
        size="small"
        value={draft.text}
        onChange={(e) => setDraft((p) => ({ ...p, text: e.target.value }))}
        maxLength={spec?.maxLen}
        data-testid={`${testId}-input`}
      />
    )
  })()

  const content = (
    <div className="sched-inplace" ref={rootRef} onKeyDown={onKeyDown} onBlur={onBlur} data-testid={testId}>
      <div className="sched-inplace-row">{editor}</div>
      {error !== '' && (
        <div className="sched-inplace-error" role="alert" data-testid={`${testId}-error`}>
          {error}
        </div>
      )}
      <div className="sched-inplace-hint">
        Enter 提交 · Esc 取消{kind === 'number' ? ' · 空=清空' : ''}
      </div>
    </div>
  )

  return (
    <Popover
      open={shown}
      onOpenChange={(next) => {
        // 受控：antd 只在 trigger 动作/点外时回调；面板自身的开合由下面锚点接管
        if (!next && shown) requestOpen(false)
      }}
      trigger={[]}
      content={content}
      placement="bottomLeft"
      arrow={false}
    >
      {/* 锚点：双击（表格单元格口径）/单击（筛选按钮口径）由这里接管——
          antd Tooltip/Popover 的 trigger 只认 hover/focus/click/contextMenu，不认 dblclick */}
      <span
        className={`sched-inplace-anchor${fullWidth ? ' sched-inplace-anchor-fw' : ''}`}
        data-testid={`${testId}-anchor`}
        onDoubleClick={(e) => {
          if (trigger !== 'dblclick') return
          e.stopPropagation()
          requestOpen(!shown)
        }}
        onClick={() => {
          if (trigger !== 'click') return
          requestOpen(!shown)
        }}
      >
        {children}
      </span>
    </Popover>
  )
}
