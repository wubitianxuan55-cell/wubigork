/**
 * unsavedGuard.tsx — 小说板块「未保存修改」统一保护原语（v4.421.0 共用件，
 * 规格 `进度计划/gaea-novel-polish-20260928.md` 线1/线3/线4 消费）。
 *
 * 为什么需要：小说五个子页**常驻挂载**（NovelPage 全 pane 同挂 + CSS 隐藏），
 * 而创作页正文、设定页 markdown、提示词工坊表单都是「受控 state + 显式保存钮」，
 * 破坏性动作（切章 / 切项目 / 生成 / 导入覆盖 / 换模板）此前直接覆盖缓冲。
 * 本模块把「先问人」收敛成两种固定形态，避免各处弹窗口径不一：
 *   - `confirmDiscard`     两选：放弃修改 / 取消
 *   - `chooseUnsavedAction` 三选：先保存 / 放弃修改 / 取消
 *
 * 语义纪律：**✕ 与 Esc 一律等于「取消」**，绝不触发破坏性分支——反例即
 * CreatePage 旧实现把「作为分支追加」挂在 `Modal.confirm.onCancel` 上，
 * 用户想退出却触发了一次 AI 生成。
 *
 * 测试纪律（沿用 SinSidePanel.test 记忆坑）：imperative Modal 不随
 * `cleanup()` 卸载，用例 afterEach 必须 `Modal.destroyAll()` + 清
 * `.ant-modal-root` 残留根节点，否则跨用例误报。
 */
import React from 'react'
import { Button, Modal } from 'antd'
import { ExclamationCircleOutlined } from '@ant-design/icons'

export interface DiscardConfirmOptions {
  title?: string
  message?: string
  /** 破坏性确认按钮文案（默认「放弃修改」） */
  discardLabel?: string
  cancelLabel?: string
  onDiscard: () => void
}

/** 两选：有未保存修改时确认「放弃修改并继续 / 取消」。取消（含 ✕/Esc）不做任何事。 */
export function confirmDiscard(opts: DiscardConfirmOptions): void {
  const {
    title = '有未保存的修改',
    message = '继续将丢弃这些修改，且无法恢复。',
    discardLabel = '放弃修改',
    cancelLabel = '取消',
    onDiscard,
  } = opts
  Modal.confirm({
    title,
    icon: <ExclamationCircleOutlined />,
    content: message,
    okText: discardLabel,
    okButtonProps: { danger: true },
    cancelText: cancelLabel,
    onOk: () => onDiscard(),
  })
}

export interface UnsavedChoiceOptions extends DiscardConfirmOptions {
  /** 提供则渲染「先保存」（调用方在保存成功后再执行后续动作） */
  onSave?: () => void
  saveLabel?: string
}

/**
 * 三选：先保存 / 放弃修改 / 取消。`footer: null` 自绘按钮，唯一出口是三个按钮
 * （✕/Esc 走默认 onCancel＝关闭，不做任何事）。
 */
export function chooseUnsavedAction(opts: UnsavedChoiceOptions): void {
  const {
    title = '有未保存的修改',
    message = '先保存，或放弃这些修改？',
    saveLabel = '先保存',
    discardLabel = '放弃修改',
    cancelLabel = '取消',
    onSave,
    onDiscard,
  } = opts

  const inst = Modal.confirm({
    title,
    icon: <ExclamationCircleOutlined />,
    maskClosable: false,
    footer: null,
    content: (
      <div data-testid="unsaved-guard">
        <div style={{ marginBottom: 16 }}>{message}</div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button size="small" onClick={() => inst.destroy()}>{cancelLabel}</Button>
          <Button size="small" danger onClick={() => { inst.destroy(); onDiscard() }}>{discardLabel}</Button>
          {onSave ? (
            <Button size="small" type="primary" onClick={() => { inst.destroy(); onSave() }}>{saveLabel}</Button>
          ) : null}
        </div>
      </div>
    ),
  })
}

export interface ChooseActionOption {
  label: string
  /** primary=主推动作；danger=破坏性动作；缺省为普通动作 */
  tone?: 'default' | 'primary' | 'danger'
  run: () => void
}

export interface ChooseActionOptions {
  title?: string
  message?: string
  /** 2~4 个并列动作，最后一个渲染在最右（惯例＝主推或取消） */
  options: ChooseActionOption[]
  cancelLabel?: string
}

/**
 * 通用并列选择（2~4 项）+ 取消。用于「覆盖下一章 / 作为分支追加 / 取消」这类
 * **多分支都有效**的场景——旧实现把第二分支挂在 `Modal.confirm.onCancel` 上，
 * 结果 ✕/Esc（用户想退出）反而触发了一次 AI 生成。
 */
export function chooseAction(opts: ChooseActionOptions): void {
  const { title, message, options, cancelLabel = '取消' } = opts
  const inst = Modal.confirm({
    title,
    icon: <ExclamationCircleOutlined />,
    maskClosable: false,
    footer: null,
    content: (
      <div data-testid="action-choice">
        {message ? <div style={{ marginBottom: 16 }}>{message}</div> : null}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button size="small" onClick={() => inst.destroy()}>{cancelLabel}</Button>
          {options.map((o) => (
            <Button
              key={o.label}
              size="small"
              type={o.tone === 'primary' ? 'primary' : 'default'}
              danger={o.tone === 'danger'}
              onClick={() => { inst.destroy(); o.run() }}
            >
              {o.label}
            </Button>
          ))}
        </div>
      </div>
    ),
  })
}
