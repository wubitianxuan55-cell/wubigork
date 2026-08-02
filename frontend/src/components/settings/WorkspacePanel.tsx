import React, { useEffect, useState } from 'react'
import { Button, Input, message, Space, Typography } from 'antd'
import { FileMarkdownOutlined, FolderOpenOutlined, PictureOutlined, SaveOutlined } from '@ant-design/icons'
import { useAppStore } from '../../stores/appStore'
import SkillModal from '../SkillModal'
import SettingsSection from './SettingsSection'

/** WorkspacePanel — 小说：小说存储目录 + 写作风格 Skill 管理 */
const WorkspacePanel: React.FC = () => {
  const { novelsDir, setNovelsDir } = useAppStore()
  const [wsDir, setWsDir] = useState(novelsDir)
  const [wsSaving, setWsSaving] = useState(false)
  const [skillOpen, setSkillOpen] = useState(false)

  useEffect(() => { setWsDir(novelsDir) }, [novelsDir])

  const handleSaveWorkspace = async () => {
    const dir = wsDir.trim()
    if (!dir) { message.warning('请输入工作空间路径'); return }
    setWsSaving(true)
    try {
      await setNovelsDir(dir)
      message.success('工作空间已更新')
    } catch (err: any) { message.error(err?.message || '保存失败') }
    finally { setWsSaving(false) }
  }

  return (
    <>
      <SettingsSection
        title={<>小说存储目录</>}
        desc="所有小说项目的书架根目录。修改后书架将刷新到新路径，当前打开的项目会自动关闭。"
      >
        <Space.Compact style={{ width: '100%' }}>
          <Input
            prefix={<FolderOpenOutlined style={{ color: 'var(--md-sys-color-text-secondary)' }} />}
            placeholder="C:\AI\xiaoshuo"
            value={wsDir}
            onChange={(e) => setWsDir(e.target.value)}
            style={{
              background: 'var(--md-sys-color-surface-container)',
              border: '1px solid var(--md-sys-color-outline-variant)',
              borderRadius: 'var(--md-sys-radius-md)', color: 'var(--md-sys-color-text)',
            }}
          />
          <Button
            type="primary" icon={<SaveOutlined />} onClick={handleSaveWorkspace} loading={wsSaving}
            style={{
              background: 'var(--md-sys-color-primary)', borderColor: 'var(--md-sys-color-primary)',
              boxShadow: '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
              borderRadius: 'var(--md-sys-radius-md)',
            }}
          >保存</Button>
        </Space.Compact>
      </SettingsSection>

      {/* 角色剧照：跟随项目存储，无需配置 */}
      <SettingsSection
        title={<>角色剧照</>}
        desc="角色剧照（AI 生成）自动保存到各小说项目的 portraits/ 子目录，跟随小说目录，无需单独配置。"
      >
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '10px 14px', borderRadius: 10,
          background: 'var(--md-sys-color-surface-container)', border: '1px solid var(--md-sys-color-outline-variant)',
        }}>
          <PictureOutlined style={{ color: 'var(--md-sys-color-primary)' }} />
          <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12 }}>
            生成位置：{wsDir || 'C:\\AI\\xiaoshuo'}\{'{项目名}'}\portraits\
          </Typography.Text>
        </div>
      </SettingsSection>


      

      <SettingsSection
        title={<>写作风格 (Skill)</>}
        desc="Skill 是写作风格指导文件，AI 写作时会注入到 prompt 中影响文风。"
      >
        <Button
          icon={<FileMarkdownOutlined />} onClick={() => setSkillOpen(true)}
          style={{
            background: 'var(--md-sys-color-primary-container)',
            border: '1px solid var(--md-sys-color-outline-variant)',
            color: 'var(--md-sys-color-on-primary-container)',
            borderRadius: 'var(--md-sys-radius-md)',
            boxShadow: '0 0 12px color-mix(in srgb, var(--gaea-glow) 15%, transparent)',
          }}
        >管理 Skill</Button>
      </SettingsSection>

      <SkillModal open={skillOpen} onClose={() => setSkillOpen(false)} />
    </>
  )
}

export default WorkspacePanel
