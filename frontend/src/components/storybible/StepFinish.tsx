import React from 'react'
import { Button, Typography, Space } from 'antd'
import { RocketOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'

interface StepFinishProps {
  projectDir: string
  title: string
  onFinish: () => void
}

const StepFinish: React.FC<StepFinishProps> = ({ projectDir, title, onFinish }) => (
  <div style={{ textAlign: 'center', padding: '20px 0' }}>
    <RocketOutlined style={{ fontSize: 48, color: '#c084fc', marginBottom: 16 }} />
    <Typography.Title level={4} style={{ color: C('color-text'), margin: 0, marginBottom: 8 }}>
      小说「{title}」创建完成！
    </Typography.Title>
    <Typography.Text style={{ color: C('color-text-secondary'), display: 'block', marginBottom: 8 }}>
      项目目录：{projectDir}
    </Typography.Text>
    <Typography.Text style={{ color: C('color-text-secondary'), display: 'block', marginBottom: 16, fontSize: 12 }}>
      已自动生成世界观设定、角色卡片和大纲框架，你可以在各个页面中进一步编辑和优化。
    </Typography.Text>
    <Button type="primary" size="large" icon={<RocketOutlined />} onClick={onFinish}>
      开始创作
    </Button>
  </div>
)

export default StepFinish
