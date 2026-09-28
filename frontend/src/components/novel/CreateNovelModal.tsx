import React, { useState } from 'react'
import { Typography, Space, Input, Modal, Checkbox, message } from 'antd'
import { C } from '../../utils/theme'
import { GENRE_OPTIONS, STYLE_OPTIONS } from './novelOptions'

interface CreateNovelModalProps {
  open: boolean
  onClose: () => void
  onCreate: () => Promise<void>
  title: string; onTitleChange: (v: string) => void
  genre: string[]; onGenreChange: (v: string[]) => void
  style: string[]; onStyleChange: (v: string[]) => void
}

/**
 * 新建小说 Modal（标题/题材/文风是父层 HomePage 的受控 state——本组件只收
 * props 不接管）。
 * v4.421.0 收口两处体验缺陷（对齐 ImportNovelModal 的正确做法）：
 *  ① 空标题此前静默无反应（HomePage.handleCreate 直接 return）：OK 按钮按
 *     `!title.trim()` 禁用，模态内再加一道提交前兜底提示；
 *  ② `onCreate()` 在途可重复点击（antd Modal 的 onOk 不管 loading，非受控
 *     返回值不会自动置 loading）→ 本地 submitting 门闩 + confirmLoading，
 *     在途重复点击直接吞掉（confirmLoading 同时挡住 ✕/Esc 误关）。
 */
const CreateNovelModal: React.FC<CreateNovelModalProps> = ({
  open, onClose, onCreate,
  title, onTitleChange,
  genre, onGenreChange,
  style, onStyleChange,
}) => {
  const [submitting, setSubmitting] = useState(false)
  const emptyTitle = !title.trim()

  const handleOk = async () => {
    if (submitting) return // 门闩：在途重复点击不再起第二次创建
    if (emptyTitle) {
      // 兜底（OK 已禁用，正常点不到）：不静默、也不关窗
      message.warning('请填写书名')
      return
    }
    setSubmitting(true)
    try {
      await onCreate()
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}>新建小说</span>}
      open={open}
      onOk={() => void handleOk()}
      onCancel={onClose}
      okText={submitting ? '创建中…' : '创建'}
      cancelText="取消"
      confirmLoading={submitting}
      okButtonProps={{ disabled: emptyTitle || submitting }}
      width={520}
      // WebView2 冻结 rAF 时退出动画不结束会残留遮罩卡死界面：关闭即卸载。
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      styles={{
        body: { background: 'transparent' },
        header: { background: 'transparent' },
      }}
    >
      <Space direction="vertical" size={14} style={{ width: '100%' }}>
        <Input
          placeholder="小说标题（必填）"
          value={title}
          onChange={(e) => onTitleChange(e.target.value)}
          style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
        />

        {/* 题材 */}
        <div>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginBottom: 6, display: 'block' }}>
            题材（可多选）
          </Typography.Text>
          <Checkbox.Group
            options={GENRE_OPTIONS}
            value={genre}
            onChange={(v) => onGenreChange(v as string[])}
            style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
          />
        </div>

        {/* 文风 */}
        <div>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginBottom: 6, display: 'block' }}>
            文风（可多选）
          </Typography.Text>
          <Checkbox.Group
            options={STYLE_OPTIONS}
            value={style}
            onChange={(v) => onStyleChange(v as string[])}
            style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
          />
        </div>
      </Space>
    </Modal>
  )
}

export default CreateNovelModal
