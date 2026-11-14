import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Input, Modal } from 'antd'
import { SearchOutlined } from '@ant-design/icons'
import type { RuntimeScheduleCommand, ScheduleCommandGroup } from '../../schedule/commandRegistry'
import { formatShortcut, searchScheduleCommands } from '../../schedule/commandRegistry'

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
    active?.scrollIntoView?.({ block: 'nearest' })
  }, [activeIndex])

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
      <div className="sched-command-palette" data-testid="sched-command-palette">
        <div className="sched-command-palette-search">
          <SearchOutlined aria-hidden="true" />
          <Input
            ref={inputRef}
            variant="borderless"
            value={query}
            placeholder="搜索命令、功能或快捷键…"
            aria-label="搜索进度命令"
            data-testid="sched-command-search"
            onChange={(event) => {
              setQuery(event.target.value)
              setActiveIndex(0)
            }}
            onKeyDown={onKeyDown}
          />
          <kbd>esc</kbd>
        </div>
        <div className="sched-command-palette-list" ref={listRef} role="listbox" aria-label="进度命令">
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
