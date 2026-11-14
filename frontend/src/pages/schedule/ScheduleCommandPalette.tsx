import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Input, Modal } from 'antd'
import { SearchOutlined } from '@ant-design/icons'
import type { RuntimeScheduleCommand, ScheduleCommandGroup } from '../../schedule/commandRegistry'
import { formatShortcut, searchScheduleCommands } from '../../schedule/commandRegistry'

/** 列表区 id：搜索框（combobox）用 aria-controls 指过来，aria-activedescendant 指到活动项 */
const LISTBOX_ID = 'sched-command-list'
const optionId = (commandId: string) => `sched-cmd-${commandId}`
/**
 * 活动项滚动入视：**不引入动画**——`behavior:'auto'`（瞬时）。显式写死 auto 是因为
 * 这一处是列表里唯一的滚动点，宁可写死也不给后续改成 `smooth` 留口子；系统开了
 * 「减少动态效果」（prefers-reduced-motion: reduce）时 auto 同样是正确语义。
 */
const ACTIVE_SCROLL = { block: 'nearest', behavior: 'auto' } as const

interface ScheduleCommandPaletteProps {
  open: boolean
  commands: readonly RuntimeScheduleCommand[]
  onClose: () => void
  onBlocked: (reason: string) => void
}

interface CommandGroup {
  name: ScheduleCommandGroup
  entries: Array<{ command: RuntimeScheduleCommand; index: number }>
}

export const ScheduleCommandPalette: React.FC<ScheduleCommandPaletteProps> = ({
  open,
  commands,
  onClose,
  onBlocked,
}) => {
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const inputRef = useRef<React.ComponentRef<typeof Input>>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const byId = useMemo(() => new Map(commands.map((command) => [command.spec.id, command])), [commands])
  const filtered = useMemo(
    () => searchScheduleCommands(query)
      .map((spec) => byId.get(spec.id))
      .filter((command): command is RuntimeScheduleCommand => command !== undefined),
    [byId, query],
  )
  const groups = useMemo(() => {
    const grouped = new Map<ScheduleCommandGroup, CommandGroup>()
    filtered.forEach((command, index) => {
      const current = grouped.get(command.spec.group) ?? { name: command.spec.group, entries: [] }
      current.entries.push({ command, index })
      grouped.set(command.spec.group, current)
    })
    return [...grouped.values()]
  }, [filtered])

  useEffect(() => {
    if (!open) return
    setQuery('')
    setActiveIndex(0)
    const timer = window.setTimeout(() => inputRef.current?.focus(), 0)
    return () => window.clearTimeout(timer)
  }, [open])

  useEffect(() => {
    setActiveIndex((index) => Math.min(index, Math.max(0, filtered.length - 1)))
  }, [filtered.length])

  useEffect(() => {
    const active = listRef.current?.querySelector<HTMLElement>('[data-command-active="true"]')
    active?.scrollIntoView?.(ACTIVE_SCROLL)
  }, [activeIndex])

  /** 活动项 id（combobox 的 aria-activedescendant；无结果时不下发，避免悬空引用） */
  const activeOptionId = filtered[activeIndex] ? optionId(filtered[activeIndex].spec.id) : undefined

  const execute = (command: RuntimeScheduleCommand) => {
    if (command.disabledReason) {
      onBlocked(command.disabledReason)
      return
    }
    onClose()
    void Promise.resolve(command.run()).catch((error: unknown) => {
      onBlocked(error instanceof Error ? error.message : String(error))
    })
  }

  const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Escape') {
      event.preventDefault()
      onClose()
      return
    }
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      if (filtered.length) setActiveIndex((index) => (index + 1) % filtered.length)
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      if (filtered.length) setActiveIndex((index) => (index - 1 + filtered.length) % filtered.length)
      return
    }
    if (event.key === 'Home') {
      // 列表首项（与 ↑/↓ 同一活动态，不另存一份游标）
      event.preventDefault()
      if (filtered.length) setActiveIndex(0)
      return
    }
    if (event.key === 'End') {
      event.preventDefault()
      if (filtered.length) setActiveIndex(filtered.length - 1)
      return
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      const command = filtered[activeIndex]
      if (command) execute(command)
    }
  }

  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      title={null}
      width={680}
      closable={false}
      destroyOnHidden
      className="sched-command-palette-modal"
      wrapClassName="sched-command-palette-wrap"
      aria-label="进度命令与快捷键"
    >
      <div
        className="sched-command-palette"
        data-testid="sched-command-palette"
        onKeyDown={(event) => {
          // 搜索框自己已处理 Esc（preventDefault）——这里只兜住「焦点落在选项上」的情形
          if (event.defaultPrevented) return
          if (event.key === 'Escape') {
            event.preventDefault()
            onClose()
          }
        }}
      >
        <div className="sched-command-palette-search">
          <SearchOutlined aria-hidden="true" />
          <Input
            ref={inputRef}
            variant="borderless"
            value={query}
            placeholder="搜索命令、功能或快捷键…"
            aria-label="搜索进度命令"
            role="combobox"
            aria-autocomplete="list"
            aria-expanded={open}
            aria-controls={LISTBOX_ID}
            aria-activedescendant={activeOptionId}
            data-testid="sched-command-search"
            onChange={(event) => {
              setQuery(event.target.value)
              setActiveIndex(0)
            }}
            onKeyDown={onKeyDown}
          />
          <kbd>esc</kbd>
        </div>
        <div className="sched-command-palette-list" ref={listRef} id={LISTBOX_ID} role="listbox" aria-label="进度命令">
          {filtered.length === 0 && (
            <div className="sched-command-palette-empty" data-testid="sched-command-empty">
              没有匹配的命令。试试“查找”“预警”或“网络图”。
            </div>
          )}
          {groups.map((group) => (
            <section key={group.name} className="sched-command-palette-group" role="group" aria-label={group.name}>
              <div className="sched-command-palette-group-title">{group.name}</div>
              {group.entries.map(({ command, index }) => {
                const active = index === activeIndex
                return (
                  <button
                    key={command.spec.id}
                    type="button"
                    role="option"
                    id={optionId(command.spec.id)}
                    // 选项不占 Tab 位：焦点始终在搜索框上，由 aria-activedescendant 报活动项
                    tabIndex={-1}
                    aria-selected={active}
                    aria-disabled={!!command.disabledReason}
                    className={`sched-command-palette-item${active ? ' is-active' : ''}${command.disabledReason ? ' is-disabled' : ''}`}
                    data-command-active={active}
                    data-command-id={command.spec.id}
                    onMouseEnter={() => setActiveIndex(index)}
                    onClick={() => execute(command)}
                  >
                    <span className="sched-command-palette-copy">
                      <span className="sched-command-palette-label">{command.spec.label}</span>
                      <span className="sched-command-palette-detail">
                        {command.disabledReason ?? command.spec.detail}
                      </span>
                    </span>
                    <span className="sched-command-palette-shortcuts">
                      {(command.spec.shortcuts ?? []).map((shortcut) => <kbd key={shortcut}>{formatShortcut(shortcut)}</kbd>)}
                    </span>
                  </button>
                )
              })}
            </section>
          ))}
        </div>
        <div className="sched-command-palette-foot">
          <span><kbd>↑</kbd><kbd>↓</kbd> 移动</span>
          <span><kbd>enter</kbd> 执行</span>
          <span><kbd>esc</kbd> 关闭</span>
          <span className="sched-dim">禁用项会说明原因，不静默失败</span>
        </div>
      </div>
    </Modal>
  )
}
