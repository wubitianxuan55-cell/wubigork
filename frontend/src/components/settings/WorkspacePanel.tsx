import React, { useEffect, useState } from 'react'
import { Button, Input, message, Space } from 'antd'
import { FileMarkdownOutlined, FolderOpenOutlined, PictureOutlined, SaveOutlined } from '@ant-design/icons'
import { useAppStore } from '../../stores/appStore'
import { getConfig, saveConfig } from '../../api/settings'
import SkillModal from '../SkillModal'
import SettingsSection from './SettingsSection'

/** WorkspacePanel — 工作空间：小说目录 + 图片保存目录 + Skill 管理 */
const WorkspacePanel: React.FC = () => {
  const { novelsDir, setNovelsDir } = useAppStore()
  const [wsDir, setWsDir] = useState(novelsDir)
  const [imgDir, setImgDir] = useState('')
  const [wsSaving, setWsSaving] = useState(false)
  const [imgSaving, setImgSaving] = useState(false)
  const [skillOpen, setSkillOpen] = useState(false)

  useEffect(() => { setWsDir(novelsDir) }, [novelsDir])

  // 加载图片保存目录
  useEffect(() => {
    getConfig().then((cfg) => {
      if (cfg.image_save_dir) setImgDir(cfg.image_save_dir)
    }).catch(() => {})
  }, [])

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

  const handleSaveImageDir = async () => {
    const dir = imgDir.trim()
    if (!dir) { message.warning('请输入图片保存目录'); return }
    setImgSaving(true)
    try {
      await saveConfig('image_save_dir', dir)
      message.success('图片保存目录已更新')
    } catch (err: any) { message.error(err?.message || '保存失败') }
    finally { setImgSaving(false) }
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
            placeholder="例如: D:\AI\xiaoshuo"
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

      <SettingsSection
        title={<>图片保存目录</>}
        desc="AI 绘梦生成的图片保存位置。留空表示不自动存盘。"
      >
        <Space.Compact style={{ width: '100%' }}>
          <Input
            prefix={<PictureOutlined style={{ color: 'var(--md-sys-color-text-secondary)' }} />}
            placeholder="留空 = 不保存到磁盘"
            value={imgDir}
            onChange={(e) => setImgDir(e.target.value)}
            style={{
              background: 'var(--md-sys-color-surface-container)',
              border: '1px solid var(--md-sys-color-outline-variant)',
              borderRadius: 'var(--md-sys-radius-md)', color: 'var(--md-sys-color-text)',
            }}
          />
          <Button
            type="primary" icon={<SaveOutlined />} onClick={handleSaveImageDir} loading={imgSaving}
            style={{
              background: 'var(--md-sys-color-primary)', borderColor: 'var(--md-sys-color-primary)',
              boxShadow: '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
              borderRadius: 'var(--md-sys-radius-md)',
            }}
          >保存</Button>
        </Space.Compact>
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
