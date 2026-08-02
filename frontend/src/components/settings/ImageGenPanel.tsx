import React, { useCallback, useEffect, useState } from 'react'
import { Button, Input, Select, Tag, Typography, message } from 'antd'
import { PictureOutlined, SaveOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { getImageBackendInfo, setImageBackend } from '../../api/settings'
import { getEngines } from '../../api/engines'
import SettingsSection from './SettingsSection'

const BUILTIN_BACKENDS = [
  { value: 'xai', label: 'xAI 云端 (grok-imagine)' },
  { value: 'comfyui', label: 'ComfyUI 本地' },
]

/** ImageGenPanel — 绘梦（AI 图像）后端设置 */
const ImageGenPanel: React.FC = () => {
  const [backend, setBackend] = useState('')
  const [model, setModel] = useState('')
  const [comfyURL, setComfyURL] = useState('')
  const [saveDir, setSaveDir] = useState('')
  const [engineOptions, setEngineOptions] = useState<{ value: string; label: string }[]>([])
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    try {
      const info = await getImageBackendInfo()
      if (info?.backend) setBackend(info.backend)
      if (info?.model) setModel(info.model)
      const es = await getEngines()
      setEngineOptions((es || []).filter((e) => e.enabled).map((e) => ({ value: e.id, label: `${e.name} (本地引擎)` })))
    } catch { /* 未初始化静默 */ }
  }, [])

  useEffect(() => { load() }, [load])

  const handleSave = async () => {
    setSaving(true)
    try {
      await setImageBackend(backend, backend === 'comfyui' ? comfyURL : '', model, saveDir)
      message.success('绘梦后端已更新')
    } catch (err: any) { message.error(err?.message || '保存失败') }
    finally { setSaving(false) }
  }

  return (
    <>
      <SettingsSection
        title={<>当前绘梦后端</>}
        desc="AI 图像生成所使用的引擎与模型。"
        instant
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <PictureOutlined style={{ fontSize: 18, color: 'var(--gaea-glow)', filter: 'drop-shadow(0 0 6px var(--gaea-glow))' }} />
          <Tag style={{ fontSize: 12, margin: 0 }} color="blue">{backend || '-'}</Tag>
          <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{model || '未配置模型'}</Typography.Text>
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>后端配置</>}
        desc="选择图像生成后端：云端 xAI 或本地 ComfyUI / 已启用的模型引擎。"
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <Select
            value={backend || undefined}
            placeholder="选择图片后端"
            onChange={setBackend}
            style={{ width: '100%' }}
            options={[...BUILTIN_BACKENDS, ...engineOptions]}
          />
          {backend === 'comfyui' && (
            <Input
              placeholder="ComfyUI 地址（例如 http://127.0.0.1:8188）"
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
            placeholder="图片保存目录（留空 = 不自动存盘）"
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
            >保存绘梦配置</Button>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 12 }}>
          <ThunderboltOutlined style={{ color: 'var(--gaea-glow)' }} />
          <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 11 }}>
            工作流细节与参数请前往「绘梦」模块调整
          </Typography.Text>
        </div>
      </SettingsSection>
    </>
  )
}

export default ImageGenPanel
