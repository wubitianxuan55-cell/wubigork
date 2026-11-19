import React, { useCallback, useEffect, useState } from 'react'
import { Button, Input, InputNumber, Switch, message } from 'antd'
import { SettingOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import { saveConfig } from '../../api/settings'
import SettingsSection from './SettingsSection'

/**
 * AdvancedSection — 高级配置键（设置中枢尾批收编）：此前仅能手改
 * ~/.gaea_config.json 的 13 键统一入设置页。读走 AdvancedConfigGet（Go 侧
 * 单源对齐的键集），写复用 SaveConfig 宽写面；只保存发生变更的键。
 * Realtime 三键不入组（provider/model 孤儿键、api_key 密文不可回显）。
 * 注：glm_catalog_url 等落盘键保存后需重启生效。
 */

interface FieldDef {
  key: string
  label: string
  desc: string
  kind: 'string' | 'int' | 'bool'
  placeholder?: string
}

// 键集与 Go AdvancedConfigGet 单源对齐（新增键两处同批）
const FIELDS: FieldDef[] = [
  { key: 'model_idle_exit_minutes', label: '外部引擎空闲退出（分钟）', desc: '闲置该时长后自动退出本地/外部引擎释放显存（0=不退出）', kind: 'int' },
  { key: 'http_timeout_seconds', label: 'HTTP 超时（秒）', desc: '云端引擎请求超时上限', kind: 'int' },
  { key: 'intents_llm_fallback', label: '意图识别 LLM 兜底', desc: '指令中枢规则未命中时走 LLM 意图兜底', kind: 'bool' },
  { key: 'intents_llm_timeout_ms', label: '意图兜底超时（毫秒）', desc: 'LLM 意图识别的等待上限', kind: 'int' },
  { key: 'read_screen_summary', label: '读屏摘要', desc: '读屏输出附摘要', kind: 'bool' },
  { key: 'read_screen_keep_last', label: '读屏保留末次', desc: '保留上一次读屏结果', kind: 'bool' },
  { key: 'strata_path', label: 'Strata 引擎路径', desc: 'Strata 推理引擎安装目录', kind: 'string', placeholder: 'C:\\AI\\strata' },
  { key: 'whirl_path', label: 'WHIRL 引擎路径', desc: 'WHIRL 推理引擎目录', kind: 'string', placeholder: 'C:\\AI\\whirl' },
  { key: 'breeze_dir', label: 'Breeze TTS 目录', desc: 'Breeze-TTS-2 本地服务目录', kind: 'string', placeholder: 'C:\\AI\\Breeze-TTS-2' },
  { key: 'breeze_port', label: 'Breeze TTS 端口', desc: 'Breeze 本地服务端口', kind: 'int' },
  { key: 'comfyui_path', label: 'ComfyUI 安装目录', desc: 'main.py 所在路径（绘梦本地后端）', kind: 'string', placeholder: 'C:\\AI\\ComfyUI' },
  { key: 'glm_catalog_url', label: 'GLM 目录刷新源', desc: 'GLM 模型价格目录 URL（保存后重启生效）', kind: 'string', placeholder: 'https://…' },
]

type Draft = Record<string, string>

const AdvancedSection: React.FC = () => {
  const [initial, setInitial] = useState<Draft | null>(null)
  const [draft, setDraft] = useState<Draft>({})
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    app.AdvancedConfigGet()
      .then((v) => {
        setInitial(v || {})
        setDraft(v || {})
      })
      .catch(() => setInitial({}))
  }, [])

  const set = useCallback((key: string, value: string) => {
    setDraft((prev) => ({ ...prev, [key]: value }))
  }, [])

  const dirty = FIELDS.filter((f) => (draft[f.key] ?? '') !== ((initial as Draft | null)?.[f.key] ?? ''))

  const save = async () => {
    if (dirty.length === 0) return
    setSaving(true)
    try {
      for (const f of dirty) {
        await saveConfig(f.key, draft[f.key] ?? '')
      }
      const v = await app.AdvancedConfigGet().catch(() => null)
      if (v) setInitial(v)
      message.success(`已保存 ${dirty.length} 项（部分键重启后生效）`)
    } catch (e: unknown) {
      message.error(`保存失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setSaving(false)
    }
  }

  const rowStyle: React.CSSProperties = {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, padding: '7px 0',
  }

  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><SettingOutlined /></span>}
      title="高级配置"
      desc="此前仅能手改配置文件的低频键：引擎路径/空闲退出/读屏/意图兜底等（写入 ~/.gaea_config.json）"
    >
      <div style={{ display: 'flex', flexDirection: 'column' }}>
        {FIELDS.map((f) => (
          <div key={f.key} style={rowStyle}>
            <div style={{ minWidth: 0 }}>
              <div style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{f.label}</div>
              <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>{f.desc}</div>
            </div>
            {f.kind === 'bool' ? (
              <Switch
                size="small"
                aria-label={f.label}
                checked={draft[f.key] === 'true'}
                disabled={!initial}
                onChange={(v) => set(f.key, v ? 'true' : 'false')}
              />
            ) : f.kind === 'int' ? (
              <InputNumber
                size="small"
                style={{ width: 120 }}
                aria-label={f.label}
                value={draft[f.key] === '' || draft[f.key] === undefined ? undefined : Number(draft[f.key])}
                disabled={!initial}
                onChange={(v) => set(f.key, v === null || v === undefined ? '' : String(v))}
              />
            ) : (
              <Input
                size="small"
                style={{ width: 260 }}
                aria-label={f.label}
                placeholder={f.placeholder}
                value={draft[f.key] ?? ''}
                disabled={!initial}
                onChange={(e) => set(f.key, e.target.value)}
              />
            )}
          </div>
        ))}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, paddingTop: 8 }}>
          <Button size="small" type="primary" loading={saving} disabled={dirty.length === 0} onClick={() => void save()}>
            保存{dirty.length > 0 ? `（${dirty.length} 项变更）` : ''}
          </Button>
          <span style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
            {initial ? '实时同步内存的键立即生效；路径/目录类键在下次引擎拉起时生效' : '读取中…'}
          </span>
        </div>
      </div>
    </SettingsSection>
  )
}

export default AdvancedSection
