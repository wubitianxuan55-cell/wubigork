import React, { useState } from 'react'
import { Typography, Button, Card, Space, Tag } from 'antd'
import { FileTextOutlined, FileMarkdownOutlined, BookOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

const ExportPage: React.FC = () => {
  const [exporting, setExporting] = useState(false)
  const [results, setResults] = useState<Record<string, string>>({})

  const handleExport = async () => {
    setExporting(true)
    try {
      // @ts-ignore
      const res = await window.go.app.App.ExportAll()
      setResults(res || {})
    } catch (err: any) {
      console.error(err)
    } finally {
      setExporting(false)
    }
  }

  return (
    <div>

      <Card style={{ background: 'var(--bg-glass)', backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-sm)', padding: 24, textAlign: 'center', marginBottom: 24 }}>
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
      </Card>

      {Object.keys(results).length > 0 && (
        <Card style={{ background: 'var(--bg-glass)', backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-sm)' }}>
          <Typography.Title level={5} style={{ color: C('color-text') }}>导出结果</Typography.Title>
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            {Object.entries(results).map(([ext, path]) => (
              <div key={ext} style={{ display: 'flex', flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', gap: 0 }}>
                <Space>
                  {ext === '.epub' ? <BookOutlined style={{ color: C('color-primary') }} /> :
                   ext === '.md' ? <FileMarkdownOutlined style={{ color: '#60a5fa' }} /> :
                   <FileTextOutlined style={{ color: C('color-text-secondary') }} />}
                  <Tag>{ext.toUpperCase()}</Tag>
                </Space>
                <Typography.Text style={{
                  color: path.startsWith('失败') ? '#f87171' : C('color-primary'),
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
        </Card>
      )}
    </div>
  )
}

export default ExportPage
