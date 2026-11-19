import React, { useEffect, useState } from 'react'
import { Button, Input, Modal, Radio, Select, Typography, message } from 'antd'
import { TeamOutlined, SmileOutlined, DatabaseOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'

const PERSONALITY_KEY = 'gaea_whisper_personality'
const LEGACY_PERSONALITY_KEY = 'wubigrok_whisper_personality'
const COMPANION_SETTINGS_KEY = 'gaea_whisper_companion_settings'
const LEGACY_COMPANION_SETTINGS_KEY = 'wubigrok_whisper_companion_settings'

interface Personality {
  id: string
  label: string
  gender: string
  dims: { T: number; I: number; S: number; O: number; R: number }
}

interface CompanionSettings {
  companionName: string
  companionGender: 'male' | 'female'
}

/** ChatVoiceSection（全局 TTS 默认音色）已随设置中枢批 2 拆出归入语音组
 *  （./ChatVoiceSection.tsx），与 VoiceSettingsPanel 同组互补展示。 */

function loadSettings(): CompanionSettings {
  try {
    const r = localStorage.getItem(COMPANION_SETTINGS_KEY) ?? localStorage.getItem(LEGACY_COMPANION_SETTINGS_KEY)
    if (r) {
      const parsed = JSON.parse(r)
      return {
        companionName: parsed?.companionName || '',
        companionGender: parsed?.companionGender === 'male' ? 'male' : 'female',
      }
    }
  } catch (_) { /* 忽略损坏的本地设置 */ }
  return { companionName: '', companionGender: 'female' }
}

function saveSettings(s: CompanionSettings) {
  try { localStorage.setItem(COMPANION_SETTINGS_KEY, JSON.stringify(s)) } catch (_) { /* 忽略 */ }
}

/** ChatPanel — 聊天设置：AI 伴侣（称呼/性别）、默认人格与语音对话核心项 */
const ChatPanel: React.FC = () => {
  const t = useT()
  const [settings, setSettings] = useState<CompanionSettings>(loadSettings)
  const [personalities, setPersonalities] = useState<Personality[]>([])
  const [activePersonality, setActivePersonality] = useState<string>(() => {
    try { return (localStorage.getItem(PERSONALITY_KEY) ?? localStorage.getItem(LEGACY_PERSONALITY_KEY)) || 'gaea' } catch { return 'gaea' }
  })

  useEffect(() => {
    try {
      // v4.362：人格清单读取失败可见化——原下拉空白伪装「没配人格」。
      app.WhisperGetPersonalities().then((p) => setPersonalities(p || [])).catch(() => {
        message.error(t('settings.chat.personalitiesLoadFailed'))
      })
    } catch { /* 未初始化时静默 */ }
  }, [t])

  const updateSettings = (patch: Partial<CompanionSettings>) => {
    const next = { ...settings, ...patch }
    setSettings(next)
    saveSettings(next)
  }

  const handleSwitchPersonality = async (id: string) => {
    try { await app.WhisperClearSession(activePersonality) } catch (_) { /* 忽略 */ }
    setActivePersonality(id)
    try { localStorage.setItem(PERSONALITY_KEY, id) } catch (_) { /* 忽略 */ }
    message.success(t('settings.chat.personaSwitched', { name: personalities.find((p) => p.id === id)?.label || id }))
  }

  const currentPersonality = personalities.find((p) => p.id === activePersonality)

  return (
    <>
      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><TeamOutlined /></span>}
        title={t('settings.chat.companionTitle')}
        desc={t('settings.chat.companionDesc')}
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 14 }}>
          <div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', display: 'block', marginBottom: 4 }}>{t('settings.chat.nameLabel')}</Typography.Text>
            <Input size="small" placeholder={t('settings.chat.namePlaceholder')} value={settings.companionName}
              onChange={(e) => updateSettings({ companionName: e.target.value })} style={{ maxWidth: 260 }} />
          </div>
          <div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', display: 'block', marginBottom: 4 }}>{t('settings.chat.genderLabel')}</Typography.Text>
            <Radio.Group size="small" value={settings.companionGender}
              onChange={(e) => updateSettings({ companionGender: e.target.value })}>
              <Radio.Button value="female" style={{ fontSize: 12 }}>{t('settings.chat.genderFemale')}</Radio.Button>
              <Radio.Button value="male" style={{ fontSize: 12 }}>{t('settings.chat.genderMale')}</Radio.Button>
            </Radio.Group>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><SmileOutlined /></span>}
        title={t('settings.chat.personaTitle')}
        desc={t('settings.chat.personaDesc')}
        instant
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
          <Select size="small" value={activePersonality} style={{ width: 200 }}
            onChange={handleSwitchPersonality}
            options={personalities.map((p) => ({ value: p.id, label: `${p.gender === 'male' ? '♂' : '♀'} ${p.label}` }))} />
          {currentPersonality?.dims && (
            <span style={{ display: 'flex', gap: 10, fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
              {['T', 'I', 'S', 'O', 'R'].map((dim, i) => {
                const vals = [currentPersonality.dims.T, currentPersonality.dims.I, currentPersonality.dims.S, currentPersonality.dims.O, currentPersonality.dims.R]
                return (
                  <span key={dim}>
                    <span style={{ color: 'var(--md-sys-color-primary)' }}>{dim}</span>
                    <span style={{ marginLeft: 2 }}>{vals[i]}</span>
                  </span>
                )
              })}
            </span>
          )}
        </div>
      </SettingsSection>

      <ChatDataSection />
    </>
  )
}

/** ChatDataSection — E 档数据管理：历史 ZIP 导出/导入（导入前自动备份、
 *  原子替换；成功后广播 gaea-chat-history-replaced 让聊天页即时刷新）。 */
const ChatDataSection: React.FC = () => {
  const [busy, setBusy] = useState<'export' | 'import' | null>(null)

  const handleExport = async () => {
    setBusy('export')
    try {
      const path = await app.ChatExportHistory()
      message.success(`已导出聊天历史：${path}`)
    } catch (err: unknown) {
      message.error(`导出失败：${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setBusy(null)
    }
  }

  const handleImport = async () => {
    try {
      const picked = await app.PickFiles('ZIP|*.zip')
      const path = picked?.[0]?.path
      if (!path) return
      Modal.confirm({
        title: '导入将覆盖全部本地会话',
        content: '导入前会自动导出一份当前历史作为备份。确定继续？',
        okText: '导入并覆盖',
        cancelText: '取消',
        onOk: async () => {
          setBusy('import')
          try {
            const n = await app.ChatImportHistory(path)
            message.success(`已导入 ${n} 个会话`)
            window.dispatchEvent(new CustomEvent('gaea-chat-history-replaced'))
          } catch (err: unknown) {
            message.error(`导入失败（原数据未动）：${err instanceof Error ? err.message : String(err)}`)
          } finally {
            setBusy(null)
          }
        },
      })
    } catch (err: unknown) {
      message.error(`选择文件失败：${err instanceof Error ? err.message : String(err)}`)
    }
  }

  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><DatabaseOutlined /></span>}
      title="数据管理"
      desc="聊天历史打包备份与恢复（ZIP，含会话与消息原文）"
      instant
    >
      <div style={{ display: 'flex', gap: 10 }}>
        <Button size="small" loading={busy === 'export'} onClick={handleExport}
          style={{ borderRadius: 8 }}>导出全部历史</Button>
        <Button size="small" danger loading={busy === 'import'} onClick={handleImport}
          style={{ borderRadius: 8 }}>从 ZIP 导入</Button>
      </div>
      <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 8 }}>
        导入会覆盖现有全部会话；导入前自动备份当前历史到导出目录。
      </div>
    </SettingsSection>
  )
}

export default ChatPanel
