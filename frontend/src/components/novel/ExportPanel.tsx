import React, { useState } from 'react'
import { Typography, Button, Space, Tag, Empty, Checkbox, message } from 'antd'
import { FileTextOutlined, FileMarkdownOutlined, BookOutlined, FileWordOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { app } from '../../gaea/lib/bridge'

/**
 * 小说导出面板（原 ExportPage 内容，合并进阅读面板后复用）：
 * 一键导出全部格式到小说目录下的 export/ 文件夹；可选仅导出主线章节。
 */
const ExportPanel: React.FC = () => {
  const [exporting, setExporting] = useState(false)
  const [results, setResults] = useState<Record<string, string>>({})
  const [exported, setExported] = useState(false)
  // 默认不勾选 = 含分支章节（与后端 ExportAll 默认行为一致）
  const [onlyMainline, setOnlyMainline] = useState(false)

  const handleExport = async () => {
    setExporting(true)
    setExported(false)
    try {
      const res = await app.ExportAll(onlyMainline)
      const next = res || {}
      setResults(next)
      setExported(true)
      const entries = Object.entries(next)
      if (entries.length === 0) {
        message.warning('导出完成，但没有生成文件。请先确认项目中有已写章节。')
        return
      }
      // v4.425：后端把逐格式失败写进 value（"失败: …"）而不是返回 error，故此前
      // 「四种格式全失败」也走 Object.keys 非空分支 → 绿色「导出完成」。这里按同一前缀
      // 口径统计失败项（渲染处 :96 已认这个前缀），全失败报 error、部分失败报 warning
      // 并带上实数，不再出现「全失败但成功提示」。
      const failed = entries.filter(([, v]) => String(v ?? '').startsWith('失败'))
      if (failed.length === entries.length) {
        message.error(`导出失败：${failed.length}/${entries.length} 个格式全部失败（${failed.map(([ext]) => ext.toUpperCase()).join('、')}）`)
      } else if (failed.length > 0) {
        message.warning(`导出完成，但 ${failed.length}/${entries.length} 个格式失败（${failed.map(([ext]) => ext.toUpperCase()).join('、')}）`)
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
          style={{
            background: 'var(--color-primary)',
            borderColor: 'var(--color-primary)',
            boxShadow: 'var(--shadow-glow)',
            borderRadius: 'var(--radius-md)',
          }}
        >
          导出全部格式 (TXT + Markdown + EPUB + DOCX)
        </Button>
        <div style={{ marginTop: 14 }}>
          <Checkbox
            checked={onlyMainline}
            onChange={(e) => setOnlyMainline(e.target.checked)}
            disabled={exporting}
          >
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13 }}>
              仅导出主线章节（跳过分支章节）
            </Typography.Text>
          </Checkbox>
        </div>
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
                      ext === '.docx' ? <FileWordOutlined style={{ color: 'var(--color-primary)' }} /> :
                        ext === '.md' ? <FileMarkdownOutlined style={{ color: 'var(--color-primary)' }} /> :
                          <FileTextOutlined style={{ color: C('color-text-secondary') }} />}
                    <Tag>{ext.toUpperCase()}</Tag>
                  </Space>
                  <Typography.Text style={{
                    color: path.startsWith('失败') ? 'var(--color-destructive)' : C('color-primary'),
                    fontSize: 12,
                    maxWidth: 360,
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

export default ExportPanel
