import React from 'react'
import { Typography, Button, Space, Input, Modal, Checkbox } from 'antd'
import { C } from '../../utils/theme'

const GENRE_OPTIONS = [
  '玄幻', '奇幻', '武侠', '仙侠', '都市',
  '现实', '历史', '军事', '游戏', '体育',
  '科幻', '悬疑', '灵异', '二次元', '短篇',
]

const STYLE_OPTIONS = [
  '热血', '轻松', '搞笑', '治愈', '暗黑',
  '爽文', '甜宠', '正剧', '无CP', '悬疑',
]

interface CreateNovelModalProps {
  open: boolean
  onClose: () => void
  onCreate: () => Promise<void>
  title: string; onTitleChange: (v: string) => void
  genre: string[]; onGenreChange: (v: string[]) => void
  style: string[]; onStyleChange: (v: string[]) => void
}

const CreateNovelModal: React.FC<CreateNovelModalProps> = ({
  open, onClose, onCreate,
  title, onTitleChange,
  genre, onGenreChange,
  style, onStyleChange,
}) => (
  <Modal
    title={<span style={{ color: C('color-text') }}>新建小说</span>}
    open={open}
    onOk={onCreate}
    onCancel={onClose}
    okText="创建"
    cancelText="取消"
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

export default CreateNovelModal
