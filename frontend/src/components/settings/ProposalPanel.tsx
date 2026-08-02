import React from 'react'
import { Typography } from 'antd'
import { LinkOutlined, FileTextOutlined } from '@ant-design/icons'
import FeatureModelBar from '../FeatureModelBar'
import SettingsSection from './SettingsSection'

/** ProposalPanel — 方案设置：方案编写模型绑定 + 生成说明 */
const ProposalPanel: React.FC = () => {
  return (
    <>
      <SettingsSection
        title={<>方案编写模型</>}
        desc="投标方案编写（大纲生成 / 文本编制 / 图表制作 / 汇总导出）使用的模型。绑定到「模型中心 → 功能绑定」统一管理，此处仅显示状态。"
      >
        <FeatureModelBar feature="office" label="方案" />
        <Typography.Paragraph type="secondary" style={{ fontSize: 12, margin: '10px 0 0' }}>
          <LinkOutlined style={{ marginRight: 4 }} />
          切换引擎 / 模型、启停绑定引擎，请到「模型中心 → 功能绑定」标签页操作，或在方案窗口左下角模型卡一键启停。
        </Typography.Paragraph>
      </SettingsSection>

      <SettingsSection
        title={<>方案生成</>}
        desc="方案板块的生成行为说明。"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {[
            { icon: '📄', t: '招标解析', d: '上传招标文件 → 转换 Markdown → 解析需求与评分项' },
            { icon: '🧭', t: '大纲生成', d: '基于需求自动生成方案章节大纲（AI 生成，可重试）' },
            { icon: '✍️', t: '文本编制', d: '逐章节编写 / 续写 / 润色，结果自动保存' },
            { icon: '📊', t: '图表制作', d: 'Mermaid 流程图 / 表格 / 架构图生成' },
            { icon: '📤', t: '汇总导出', d: '全部章节汇总为完整方案文档' },
          ].map((row) => (
            <div key={row.t} style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
              <span style={{ fontSize: 15, width: 22, textAlign: 'center', flexShrink: 0 }}>{row.icon}</span>
              <div style={{ minWidth: 0 }}>
                <Typography.Text style={{ fontSize: 12.5, color: 'var(--md-sys-color-text)', fontWeight: 500, display: 'block' }}>
                  {row.t}
                </Typography.Text>
                <Typography.Text style={{ fontSize: 11.5, color: 'var(--md-sys-color-text-secondary)' }}>
                  {row.d}
                </Typography.Text>
              </div>
            </div>
          ))}
        </div>
        <div style={{ marginTop: 12, padding: '10px 12px', borderRadius: 10, background: 'var(--md-sys-color-surface-container)', border: '1px solid var(--md-sys-color-outline-variant)', display: 'flex', alignItems: 'center', gap: 8 }}>
          <FileTextOutlined style={{ color: 'var(--md-sys-color-primary)' }} />
          <Typography.Text style={{ fontSize: 11.5, color: 'var(--md-sys-color-text-secondary)' }}>
            方案数据存储于当前小说目录下的方案项目中，随小说目录迁移。
          </Typography.Text>
        </div>
      </SettingsSection>
    </>
  )
}

export default ProposalPanel
