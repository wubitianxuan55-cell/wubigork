import React, { useEffect, useState } from 'react'
import { Button, Input, message } from 'antd'
import { FileTextOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import SettingsSection from './SettingsSection'

/**
 * WenxuePromptSection — 文学分板身份提示词（设置中枢尾批收编）：wenxue_system_
 * prompt / wenxue_system_prompt_file 双键此前仅能手改 ga.cfg。file 键优先
 * （档位文件内容即人格），双空 = 板块代码默认——文学引擎绝不继承办公 system_
 * prompt（2026-11-16 身份分板定局）。保存经 WenxuePromptSet 落盘并跟随重建
 * 图文引擎（下次会话新人格生效）。
 */
const WenxuePromptSection: React.FC = () => {
  const [prompt, setPrompt] = useState('')
  const [file, setFile] = useState('')
  const [effective, setEffective] = useState('')
  const [loaded, setLoaded] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    app.WenxuePromptGet()
      .then((v) => {
        setPrompt(v?.prompt ?? '')
        setFile(v?.file ?? '')
        setEffective(v?.effective ?? '')
        setLoaded(true)
      })
      .catch(() => setLoaded(true))
  }, [])

  const save = async () => {
    setSaving(true)
    try {
      await app.WenxuePromptSet(prompt, file)
      const v = await app.WenxuePromptGet().catch(() => null)
      if (v) setEffective(v.effective ?? '')
      message.success('文学身份提示词已保存，图文引擎已随配置重建（下次会话新人格生效）')
    } catch (e: unknown) {
      message.error(`保存失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setSaving(false)
    }
  }

  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><FileTextOutlined /></span>}
      title="身份提示词"
      desc="文学板块独立人格（不继承办公 system_prompt）：内联文本或档位文件二选一，文件优先，双空 = 板块默认"
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        <div>
          <div style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', marginBottom: 4 }}>
            wenxue_system_prompt_file（档位文件绝对路径，优先于此处内联文本）
          </div>
          <Input
            size="small"
            value={file}
            placeholder="例如 D:\prompts\wenxue-persona.md（留空=用内联文本）"
            aria-label="wenxue_system_prompt_file"
            onChange={(e) => setFile(e.target.value)}
          />
        </div>
        <div>
          <div style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', marginBottom: 4 }}>
            wenxue_system_prompt（内联身份提示词）
          </div>
          <Input.TextArea
            rows={5}
            value={prompt}
            placeholder="留空且未配档位文件 = 板块默认人格"
            aria-label="wenxue_system_prompt"
            onChange={(e) => setPrompt(e.target.value)}
          />
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Button size="small" type="primary" loading={saving} disabled={!loaded} onClick={() => void save()}>
            保存并重建
          </Button>
          {loaded && (
            <span style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
              当前生效：{effective ? `${effective.slice(0, 60)}${effective.length > 60 ? '…' : ''}` : '板块默认'}
            </span>
          )}
        </div>
      </div>
    </SettingsSection>
  )
}

export default WenxuePromptSection
