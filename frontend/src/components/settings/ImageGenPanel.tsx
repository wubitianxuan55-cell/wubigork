import React, { useCallback, useEffect, useState } from 'react'
import { Button, Input, Select, Typography, message } from 'antd'
import { SaveOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { getImageBackendInfo, setImageBackend } from '../../api/settings'
import { getEngines } from '../../api/engines'
import SettingsSection from './SettingsSection'
import { BACKEND_OPTIONS, isLocalBackend } from '../imagegen/meta'
import { useT } from '../../gaea/lib/i18n'

// 固定后端（非引擎型，恒可选）：xai / comfyui——标签与绘梦页同源（meta.ts）。
// herdsman/ollama/glm 等为引擎型，由下方「已启用引擎」列表按启用状态呈现。
const FIXED_BACKEND_VALUES = ['xai', 'comfyui']

/** ImageGenPanel — 绘梦（AI 图像）后端设置 */
const ImageGenPanel: React.FC = () => {
  const t = useT()
  const [backend, setBackend] = useState('')
  const [model, setModel] = useState('')
  const [comfyURL, setComfyURL] = useState('')
  const [saveDir, setSaveDir] = useState('')
  const [engineOptions, setEngineOptions] = useState<{ value: string; label: string }[]>([])
  const [saving, setSaving] = useState(false)

  // 固定后端标签带「本地/云端」字尾，需经 i18n —— 在组件内派生（原为模块级 BUILTIN_BACKENDS）
  const builtinBackends = BACKEND_OPTIONS
    .filter((o) => FIXED_BACKEND_VALUES.includes(o.value))
    .map((o) => ({
      value: o.value,
      label: t('settings.imgen.builtinLabel', {
        name: o.label,
        kind: isLocalBackend(o.value) ? t('settings.imgen.kindLocal') : t('settings.imgen.kindCloud'),
      }),
    }))

  const load = useCallback(async () => {
    try {
      const info = await getImageBackendInfo()
      if (info?.backend) setBackend(info.backend)
      if (info?.model) setModel(info.model)
      // 已存地址/目录一并回填：否则 comfyui 后端下直接保存会把 URL 清空
      if (info?.comfyui_url) setComfyURL(info.comfyui_url)
      if (info?.image_save_dir) setSaveDir(info.image_save_dir)
      const es = await getEngines()
      // 引擎本地/云端属性决定标签（此前一律标「本地引擎」，云端引擎被误标）；
      // 固定后端（xai/comfyui）已在 builtinBackends 呈现，引擎列表过滤掉以免
      // 下拉重复（xai 也是引擎 id）。
      setEngineOptions((es || [])
        .filter((e) => e.enabled && !FIXED_BACKEND_VALUES.includes(e.id))
        .map((e) => ({
          value: e.id,
          label: t('settings.imgen.engineLabel', {
            name: e.name,
            kind: e.is_local ? t('settings.imgen.kindLocalEngine') : t('settings.imgen.kindCloud'),
          }),
        })))
    } catch { /* 未初始化静默 */ }
  }, [t])

  useEffect(() => { load() }, [load])

  const handleSave = async () => {
    setSaving(true)
    try {
      await setImageBackend(
        backend,
        backend === 'comfyui' ? comfyURL : '',
        model,
        saveDir,
      )
      message.success(t('settings.imgen.saved'))
    } catch (err: unknown) { message.error(err instanceof Error ? err.message : t('settings.saveFailed')) }
    finally { setSaving(false) }
  }

  return (
    <>
      <SettingsSection
        title={t('settings.imgen.backendTitle')}
        desc={t('settings.imgen.backendDesc')}
        instant
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <Select
            value={backend || undefined}
            placeholder={t('settings.imgen.backendPlaceholder')}
            onChange={setBackend}
            style={{ width: '100%' }}
            options={[...builtinBackends, ...engineOptions]}
          />
          {backend === 'comfyui' && (
            <Input
              placeholder={t('settings.imgen.comfyURLPlaceholder')}
              value={comfyURL}
              onChange={(e) => setComfyURL(e.target.value)}
              style={{
                background: 'var(--md-sys-color-surface-container)',
                border: '1px solid var(--md-sys-color-outline-variant)',
                color: 'var(--md-sys-color-text)',
              }}
            />
          )}
          <Input
            placeholder={t('settings.imgen.saveDirPlaceholder')}
            value={saveDir}
            onChange={(e) => setSaveDir(e.target.value)}
            style={{
              background: 'var(--md-sys-color-surface-container)',
              border: '1px solid var(--md-sys-color-outline-variant)',
              color: 'var(--md-sys-color-text)',
            }}
          />
          <div>
            <Button
              type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving}
              style={{
                background: 'var(--md-sys-color-primary)', borderColor: 'var(--md-sys-color-primary)',
                boxShadow: '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
                borderRadius: 'var(--md-sys-radius-md)',
              }}
            >{t('settings.imgen.save')}</Button>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 12 }}>
          <ThunderboltOutlined style={{ color: 'var(--gaea-glow)' }} />
          <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 11 }}>
            {t('settings.imgen.goImagegen')}
          </Typography.Text>
        </div>
      </SettingsSection>
    </>
  )
}

export default ImageGenPanel
