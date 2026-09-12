import { wailsApp } from '../lib/wailsApp';
import React, { useState, useEffect, useRef } from 'react'
import { Typography, Button, Space, Tag, Modal, message } from 'antd'
import {
  ImportOutlined, FileMarkdownOutlined, BulbOutlined,
} from '@ant-design/icons'

import { C } from '../utils/theme'
import V3Empty from './V3Empty'
import { inShellEnv, pickFileAsFile } from '../gaea/lib/pickFile'

interface SkillInfo {
  name: string
  description: string
  appliesTo: string[]
  version: string
}

interface SkillModalProps {
  open: boolean
  onClose: () => void
}

const SkillModal: React.FC<SkillModalProps> = ({ open, onClose }) => {
  const [skills, setSkills] = useState<SkillInfo[]>([])
  const [importName, setImportName] = useState('')

  const mountedRef = useRef(true)
  useEffect(() => { mountedRef.current = true; return () => { mountedRef.current = false } }, [])

  useEffect(() => {
    if (open) loadSkills()
  }, [open])

  const loadSkills = async () => {
    try {
      const s = await wailsApp().ListSkills()
      setSkills(s || [])
    } catch (_) {}
  }

  // 引导性半桩（诚实 parity）：只识别并展示文件名，真导入=手动把 SKILL.md
  // 放入 skills/ 目录（后端无导入绑定，不做落盘——禁止过度设计）。
  const acceptSkillFile = (file: File) => {
    const name = file.name.replace(/\.md$/i, '')
    setImportName(name)
    message.info(`Skill「${name}」已识别。将 SKILL.md 放入 skills/ 目录即可使用。`)
  }

  // 审计刀C-3（v4.162 实证）：壳内动态 input.click() 不弹文件对话框，改走
  // GaeaPickFiles 系统对话框还原 File 后维持同款「仅展示文件名」半桩语义；
  // 浏览器回退原动态 input。扩展名不合法由 pickFileAsFile 后置校验抛错，
  // message 提示（fail-closed）；取消=null 静默返回。
  const handleImport = async () => {
    if (inShellEnv()) {
      try {
        const file = await pickFileAsFile(['md'])
        if (file) acceptSkillFile(file)
      } catch (e) {
        message.error(e instanceof Error ? e.message : String(e))
      }
      return
    }
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.md'
    input.onchange = async (e: Event) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (!file) return
      acceptSkillFile(file)
    }
    input.click()
  }

  const appliesColors: Record<string, string> = {
    chapter: 'var(--color-success)', character: 'var(--color-warning)',
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}><FileMarkdownOutlined style={{ color: '#60a5fa' /* hex-exempt 品牌识别色 */, marginRight: 8 }} />Skill 管理</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      width={640}
      styles={{
        body: { maxHeight: '65vh', overflow: 'auto' },
      }}
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
          <BulbOutlined style={{ marginRight: 4 }} />Skill 是写作风格指导文件（Markdown），AI 写作时会注入到 prompt 中影响文风。支持长篇网文、去AI味等场景。
        </Typography.Text>

        {/* 导入 */}
        <div style={{ background: 'var(--bg-elevated)', borderRadius: 'var(--radius-md)', padding: 12, border: '1px solid var(--border-subtle)' }}>
          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 8 }}>
            导入 Skill
          </Typography.Text>
          <Space>
            <Button icon={<ImportOutlined />} onClick={handleImport}
              style={{ borderColor: '#60a5fa' /* hex-exempt 品牌识别色 */, color: '#60a5fa' /* hex-exempt 品牌识别色 */, borderRadius: 'var(--radius-md)', background: 'var(--bg-elevated)' }}>
              导入 SKILL.md
            </Button>
            {importName && (
              <Tag color="green">{importName}</Tag>
            )}
          </Space>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10, display: 'block', marginTop: 8 }}>
            将 .md 文件放入 skills/ 子目录，重启后生效
          </Typography.Text>
        </div>

        {/* 已安装列表 */}
        {skills.length === 0 ? (
          <V3Empty description="暂无 Skill" />
        ) : (
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
              已安装 ({skills.length})
            </Typography.Text>
            {skills.map((s) => (
              <div
                key={s.name}
                style={{
                  background: 'var(--bg-elevated)', borderRadius: 'var(--radius-md)',
                  padding: '10px 14px', border: '1px solid var(--border-subtle)',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <div style={{ flex: 1 }}>
                    <Space size={4}>
                      <FileMarkdownOutlined style={{ color: '#60a5fa' /* hex-exempt 品牌识别色 */ }} />
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
                        {s.name}
                      </Typography.Text>
                      <Tag style={{ fontSize: 9 }}>v{s.version}</Tag>
                    </Space>
                    <div style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 4 }}>
                      {s.description}
                    </div>
                  </div>
                  <Space size={4}>
                    {(s.appliesTo || []).map((a) => (
                      <Tag key={a} color={appliesColors[a]} style={{ fontSize: 9, margin: 0 }}>
                        {a === 'chapter' ? '写作' : a}
                      </Tag>
                    ))}
                  </Space>
                </div>
              </div>
            ))}
          </Space>
        )}

        {/* 如何创建 Skill */}
        <div style={{
          background: 'var(--bg-elevated)', borderRadius: 'var(--radius-md)', padding: 12,
          border: '1px dashed var(--border-subtle)',
        }}>
          <Typography.Text strong style={{ color: C('color-text'), fontSize: 12, display: 'block', marginBottom: 6 }}>
            🛠️ 如何创建自定义 Skill
          </Typography.Text>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, fontFamily: 'monospace', whiteSpace: 'pre-wrap', lineHeight: 1.8 }}>
{`---
name: my-style
description: 我的专属写作风格
applies_to:
  - chapter
version: "1.0"
---

# 写作指导内容...
（Markdown 格式，写你的风格要求）`}
          </Typography.Text>
        </div>
      </Space>
    </Modal>
  )
}

export default SkillModal
