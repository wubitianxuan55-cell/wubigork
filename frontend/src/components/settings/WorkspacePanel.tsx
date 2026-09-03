import React, { useEffect, useState } from 'react'
import { Button, Input, message, Space } from 'antd'
import { FileMarkdownOutlined, FolderOpenOutlined, SaveOutlined } from '@ant-design/icons'
import { useAppStore } from '../../stores/appStore'
import SkillModal from '../SkillModal'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'

/** WorkspacePanel — 小说：小说存储目录 + 写作风格 Skill 管理 */
const WorkspacePanel: React.FC = () => {
  const t = useT()
  const { novelsDir, setNovelsDir } = useAppStore()
  const [wsDir, setWsDir] = useState(novelsDir)
  const [wsSaving, setWsSaving] = useState(false)
  const [skillOpen, setSkillOpen] = useState(false)

  useEffect(() => { setWsDir(novelsDir) }, [novelsDir])

  const handleSaveWorkspace = async () => {
    const dir = wsDir.trim()
    if (!dir) { message.warning(t('settings.novelWs.dirRequired')); return }
    setWsSaving(true)
    try {
      await setNovelsDir(dir)
      message.success(t('settings.novelWs.saved'))
    } catch (err: unknown) { message.error(err instanceof Error ? err.message : t('settings.saveFailed')) }
    finally { setWsSaving(false) }
  }

  return (
    <>
      <SettingsSection
        title={t('settings.novelWs.dirTitle')}
        desc={t('settings.novelWs.dirDesc')}
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
          >{t('common.save')}</Button>
        </Space.Compact>
      </SettingsSection>

      <SettingsSection
        title={t('settings.novelWs.styleTitle')}
        desc={t('settings.novelWs.styleDesc')}
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
        >{t('settings.novelWs.manageSkills')}</Button>
      </SettingsSection>

      <SkillModal open={skillOpen} onClose={() => setSkillOpen(false)} />
    </>
  )
}

export default WorkspacePanel
