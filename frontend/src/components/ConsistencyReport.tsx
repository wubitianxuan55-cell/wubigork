import React from 'react'
import { Typography, Button, Space, Tag, Card } from 'antd'
import {
  AlertOutlined, WarningOutlined, InfoCircleOutlined,
  CheckCircleOutlined, BulbOutlined,
} from '@ant-design/icons'
import type { ConsistencyReportData } from '../types'
import { C } from '../utils/theme'

const severityIcons: Record<string, React.ReactNode> = {
  error: <AlertOutlined style={{ color: '#f87171' }} />,
  warning: <WarningOutlined style={{ color: '#f59e0b' }} />,
  info: <InfoCircleOutlined style={{ color: '#60a5fa' }} />,
}
const severityColors: Record<string, string> = {
  error: '#f87171', warning: '#f59e0b', info: '#60a5fa',
}
const severityLabels: Record<string, string> = {
  error: '严重', warning: '警告', info: '提示',
}

interface ConsistencyReportProps {
  report: ConsistencyReportData
  onClose: () => void
  onJumpToSection: (sectionId: string) => void
}

/** ConsistencyReport — 世界观一致性检查报告卡片 */
const ConsistencyReport: React.FC<ConsistencyReportProps> = ({ report, onClose, onJumpToSection }) => (
  <Card size="small" style={{
    marginBottom: 8,
    background: 'var(--bg-glass)',
    backdropFilter: 'blur(8px)',
    borderRadius: 'var(--radius-lg)',
    border: `1px solid ${report.issues.length > 0 ? '#f59e0b' : 'var(--border-subtle)'}`,
  }}
    title={
      <Space>
        <CheckCircleOutlined style={{ color: report.issues.length === 0 ? C('color-primary') : '#f59e0b' }} />
        <span style={{ color: C('color-text') }}>一致性检查报告</span>
        <Button type="text" size="small" onClick={onClose} style={{ color: C('color-text-secondary') }}>✕</Button>
      </Space>
    }
  >
    {report.issues.length === 0 ? (
      <Typography.Text style={{ color: C('color-primary') }}>✅ {report.overall_note || '未发现逻辑矛盾'}</Typography.Text>
    ) : (
      <div>
        <Typography.Text style={{ color: C('color-text-secondary') }}>{report.overall_note}</Typography.Text>
        <Space direction="vertical" size={8} style={{ width: '100%', marginTop: 12 }}>
          {report.issues.map((issue, i) => (
            <div key={i} style={{
              padding: '8px 12px', background: C('color-bg-layout'), borderRadius: 6,
              borderLeft: `3px solid ${severityColors[issue.severity]}`,
            }}>
              <Space>
                {severityIcons[issue.severity]}
                <Tag color={severityColors[issue.severity]} style={{ fontSize: 10 }}>{severityLabels[issue.severity]}</Tag>
                <Tag style={{ fontSize: 10, cursor: 'pointer', background: '#c084fc18' }}
                  onClick={() => onJumpToSection(issue.section)}>
                  {issue.section} 🔧
                </Tag>
              </Space>
              <div style={{ color: C('color-text'), fontSize: 13, marginTop: 6 }}>{issue.description}</div>
              {issue.suggestion && (
                <div style={{ color: C('color-primary'), fontSize: 12, marginTop: 4 }}>
                  <BulbOutlined style={{ marginRight: 4 }} />{issue.suggestion}
                </div>
              )}
            </div>
          ))}
        </Space>
      </div>
    )}
  </Card>
)

export default ConsistencyReport
