import React, { useCallback, useEffect, useState } from 'react'
import { Button, Input, InputNumber, Switch, message } from 'antd'
import { ControlOutlined, FolderOpenOutlined, SettingOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import { saveConfig } from '../../api/settings'
import SettingsSection from './SettingsSection'
import SettingsPanelTabs, { type SettingsPanelTab } from './SettingsPanelTabs'

/**
 * AdvancedSection — 低频高级配置。
 *
 * 12 个键按任务拆为「运行与意图」和「引擎与目录」两组。两组共用同一份草稿、
 * 差异检测与保存动作，切换页签不会丢未保存输入。
 */

interface FieldDef {
  key: string
  label: string
  desc: string
  kind: 'string' | 'int' | 'bool'
  placeholder?: string
}

// 键集与 Go AdvancedConfigGet 单源对齐（新增键两处同批）
const RUNTIME_FIELDS: FieldDef[] = [
  { key: 'model_idle_exit_minutes', label: '外部引擎空闲退出（分钟）', desc: '闲置该时长后自动退出本地/外部引擎释放显存（0=不退出）', kind: 'int' },
  { key: 'http_timeout_seconds', label: 'HTTP 超时（秒）', desc: '云端引擎请求超时上限', kind: 'int' },
  { key: 'intents_llm_fallback', label: '意图识别 LLM 兜底', desc: '指令中枢规则未命中时走 LLM 意图兜底', kind: 'bool' },
  { key: 'intents_llm_timeout_ms', label: '意图兜底超时（毫秒）', desc: 'LLM 意图识别的等待上限', kind: 'int' },
  { key: 'read_screen_summary', label: '读屏摘要', desc: '读屏输出附摘要', kind: 'bool' },
  { key: 'read_screen_keep_last', label: '读屏保留末次', desc: '保留上一次读屏结果', kind: 'bool' },
]

const PATH_FIELDS: FieldDef[] = [
  { key: 'strata_path', label: 'Strata 引擎路径', desc: 'Strata 推理引擎安装目录', kind: 'string', placeholder: 'C:\\AI\\strata' },
  { key: 'whirl_path', label: 'WHIRL 引擎路径', desc: 'WHIRL 推理引擎目录', kind: 'string', placeholder: 'C:\\AI\\whirl' },
  { key: 'breeze_dir', label: 'Breeze TTS 目录', desc: 'Breeze-TTS-2 本地服务目录', kind: 'string', placeholder: 'C:\\AI\\Breeze-TTS-2' },
  { key: 'breeze_port', label: 'Breeze TTS 端口', desc: 'Breeze 本地服务端口', kind: 'int' },
  { key: 'comfyui_path', label: 'ComfyUI 安装目录', desc: 'main.py 所在路径（绘梦本地后端）', kind: 'string', placeholder: 'C:\\AI\\ComfyUI' },
  { key: 'glm_catalog_url', label: 'GLM 目录刷新源', desc: 'GLM 模型价格目录 URL（保存后重启生效）', kind: 'string', placeholder: 'https://…' },
]

type Draft = Record<string, string>

interface AdvancedSectionProps {
  query?: string
  initialTab?: string
  onTabChange?: (tab: string) => void
}

const AdvancedSection: React.FC<AdvancedSectionProps> = ({ query, initialTab, onTabChange }) => {
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

  const dirty = [...RUNTIME_FIELDS, ...PATH_FIELDS]
    .filter((f) => (draft[f.key] ?? '') !== (initial?.[f.key] ?? ''))

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

  const renderFields = (fields: FieldDef[]) => (
    <div className="advanced-field-list">
      {fields.map((field) => (
        <div className="advanced-field-row" key={field.key}>
          <div className="advanced-field-copy">
            <div className="advanced-field-label">{field.label}</div>
            <div className="advanced-field-desc">{field.desc}</div>
          </div>
          <div className="advanced-field-control">
            {field.kind === 'bool' ? (
              <Switch
                size="small"
                aria-label={field.label}
                checked={draft[field.key] === 'true'}
                disabled={!initial}
                onChange={(v) => set(field.key, v ? 'true' : 'false')}
              />
            ) : field.kind === 'int' ? (
              <InputNumber
                size="small"
                style={{ width: 120 }}
                aria-label={field.label}
                value={draft[field.key] === '' || draft[field.key] === undefined ? undefined : Number(draft[field.key])}
                disabled={!initial}
                onChange={(v) => set(field.key, v === null || v === undefined ? '' : String(v))}
              />
            ) : (
              <Input
                size="small"
                aria-label={field.label}
                placeholder={field.placeholder}
                value={draft[field.key] ?? ''}
                disabled={!initial}
                onChange={(e) => set(field.key, e.target.value)}
              />
            )}
          </div>
        </div>
      ))}
    </div>
  )

  const tabs: SettingsPanelTab[] = [
    {
      key: 'runtime',
      label: '运行与意图',
      description: '调整空闲退出、请求超时、意图兜底与读屏行为。',
      icon: <ControlOutlined />,
      keywords: ['运行', '意图', '空闲退出', '超时', '读屏', '兜底'],
      content: (
        <SettingsSection
          icon={<SettingOutlined />}
          title="运行与意图"
          desc="这些键直接影响本地引擎驻留、云端请求等待与意图识别行为。"
        >
          {renderFields(RUNTIME_FIELDS)}
        </SettingsSection>
      ),
    },
    {
      key: 'paths',
      label: '引擎与目录',
      description: '配置本地引擎、语音服务和模型目录的安装位置。',
      icon: <FolderOpenOutlined />,
      keywords: ['路径', '目录', 'strata', 'whirl', 'breeze', 'comfyui', 'glm', '引擎'],
      content: (
        <SettingsSection
          icon={<FolderOpenOutlined />}
          title="引擎与目录"
          desc="路径和端口类键在下次引擎拉起时生效；目录填写到可执行入口所在层级。"
        >
          {renderFields(PATH_FIELDS)}
        </SettingsSection>
      ),
    },
  ]

  return (
    <div className="advanced-settings">
      <SettingsPanelTabs
        ariaLabel="高级设置任务"
        tabs={tabs}
        query={query}
        defaultKey={initialTab}
        onActiveChange={onTabChange}
      />
      <div className="advanced-savebar">
        <span className="advanced-savebar__status">
          {initial ? '即时同步内存的键立即生效；路径/目录类键在下次引擎拉起时生效' : '读取中…'}
        </span>
        <Button
          size="small"
          type="primary"
          loading={saving}
          disabled={!initial || dirty.length === 0}
          onClick={() => void save()}
        >
          保存{dirty.length > 0 ? `（${dirty.length} 项变更）` : ''}
        </Button>
      </div>
    </div>
  )
}

export default AdvancedSection
