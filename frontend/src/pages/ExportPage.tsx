import React, { useState } from 'react'
import { Typography, Button, Space, Tag, Empty, message } from 'antd'
import { FileTextOutlined, FileMarkdownOutlined, BookOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'
import * as App from '../../src/wailsjsCompat'

const ExportPage: React.FC = () => {
  const [exporting, setExporting] = useState(false)
  const [results, setResults] = useState<Record<string, string>>({})
  const [exported, setExported] = useState(false)

  const handleExport = async () => {
    setExporting(true)
    setExported(false)
    try {
      const res = await App.ExportAll()
      const next = res || {}
      setResults(next)
      setExported(true)
      if (Object.keys(next).length === 0) {
        message.warning('导出完成，但没有生成文件。请先确认项目中有已写章节。')
      } else {
        message.success('导出完成')
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '导出失败')
    } finally {
      setExporting(false)
    }
  }

  return (
    <div>

      <div className="novel-panel" style={{ padding: 24, textAlign: 'center', marginBottom: 24 }}>
        <Typography.Paragraph style={{ color: C('color-text-secondary') }}>
          一键导出全部格式到小说目录下的 export/ 文件夹。
        </Typography.Paragraph>
        <Button
          type="primary"
          size="large"
          icon={<BookOutlined />}
          onClick={handleExport}
          loading={exporting}
          block
          style={{ background: 'var(--color-primary)', borderColor: 'var(--color-primary)', boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)' }}
        >
          导出全部格式 (TXT + Markdown + EPUB)
        </Button>
      </div>

      {exported && Object.keys(results).length === 0 && (
        <div className="novel-panel" style={{ padding: 32, textAlign: 'center' }}>
          <Empty description="没有可导出的章节内容" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        </div>
      )}

      {Object.keys(results).length > 0 && (
        <div className="novel-panel" style={{ overflow: 'hidden' }}>
          <div className="novel-panel-head">
            <span className="novel-panel-title"><FileTextOutlined />导出结果</span>
          </div>
          <div style={{ padding: 14 }}>
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              {Object.entries(results).map(([ext, path]) => (
                <div key={ext} style={{ display: 'flex', flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', gap: 0 }}>
                  <Space>
                    {ext === '.epub' ? <BookOutlined style={{ color: C('color-primary') }} /> :
                     ext === '.md' ? <FileMarkdownOutlined style={{ color: 'var(--color-primary)' }} /> :
                     <FileTextOutlined style={{ color: C('color-text-secondary') }} />}
                    <Tag>{ext.toUpperCase()}</Tag>
                  </Space>
                  <Typography.Text style={{
                    color: path.startsWith('失败') ? 'var(--color-destructive)' : C('color-primary'),
                    fontSize: 12,
                    maxWidth: 400,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    wordBreak: 'break-all',
                  }}>
                    {path}
                  </Typography.Text>
                </div>
              ))}
            </Space>
          </div>
        </div>
      )}
    </div>
  )
}

export default ExportPage
