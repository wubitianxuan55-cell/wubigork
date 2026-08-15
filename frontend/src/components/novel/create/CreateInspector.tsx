import React, { useEffect, useState } from 'react'
import { Button, Input, InputNumber, Select, Tag } from 'antd'
import {
  BulbOutlined, FileTextOutlined, ReloadOutlined, ExperimentOutlined,
  SettingOutlined, ThunderboltOutlined, BarChartOutlined,
  MenuFoldOutlined, MenuUnfoldOutlined,
} from '@ant-design/icons'
import { C } from '../../../utils/theme'
import * as App from '../../../../src/wailsjsCompat'

interface Skill { name: string; description: string; appliesTo?: string[] }

interface CreateInspectorProps {
  setting: string
  onRefreshSetting: () => void
  selectedSkill?: string
  onSelectSkill: (value?: string) => void
  minWords: number
  onMinWordsChange: (value: number) => void
  temperature: number
  onTemperatureChange: (value: number) => void
  directPlot: string
  onDirectPlotChange: (value: string) => void
  onDirectGenerate: () => void
  /** 构思剧情方向的起点章节（当前选中或末章） */
  prevChapterHint: number
  onOpenWizard: (prevChapter: number) => void
  stats: { totalWords: number; chapterCount: number } | null
  chapterCount: number
}

/** 右侧创作设置面板（T6-7.5 从 CreatePage 拆分）：设定预览 / 技能 / 字数温度 / 剧情方向 / 统计 */
const CreateInspector: React.FC<CreateInspectorProps> = ({
  setting, onRefreshSetting,
  selectedSkill, onSelectSkill,
  minWords, onMinWordsChange,
  temperature, onTemperatureChange,
  directPlot, onDirectPlotChange, onDirectGenerate, prevChapterHint,
  onOpenWizard, stats, chapterCount,
}) => {
  const [skills, setSkills] = useState<Skill[]>([])
  const [collapsed, setCollapsed] = useState(false)

  // 加载可用 Skill 列表
  useEffect(() => {
    (async () => {
      try {
        const list = await App.ListSkills()
        setSkills((list || []) as Skill[])
      } catch {
        // 技能列表加载失败：右侧仅展示空技能提示，不阻塞创作
      }
    })()
  }, [])

  const selectedSkillDesc = skills.find(s => s.name === selectedSkill)?.description
  const selectedSkillApplies = skills.find(s => s.name === selectedSkill)?.appliesTo || []

  return (
    <aside className={`novel-panel novel-workspace-col novel-inspector${collapsed ? ' is-collapsed' : ''}`}>
      <div className="novel-panel-head">
        <span className="novel-panel-title"><SettingOutlined />创作设置</span>
        <div style={{ flex: 1 }} />
        <Button type="text" size="small"
          icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          onClick={() => setCollapsed((p) => !p)}
          aria-label={collapsed ? '展开创作设置' : '折叠创作设置'}
          title={collapsed ? '展开创作设置' : '折叠创作设置'}
          style={{ color: 'var(--color-text-secondary)', fontSize: 11, padding: 0 }} />
      </div>
      {!collapsed && (
      <div className="novel-inspector-body">

        <section className="novel-inspector-section">
          <div className="novel-inspector-section-title">
            <FileTextOutlined />小说设定
            <div style={{ flex: 1 }} />
            <Button type="text" size="small" icon={<ReloadOutlined />} title="刷新设定" onClick={onRefreshSetting} />
          </div>
          {setting ? (
            <div className="novel-setting-preview-box" title={setting}>
              {setting.slice(0, 240)}{setting.length > 240 ? '…' : ''}
            </div>
          ) : (
            <div className="novel-inspector-hint">设定为空，请先在「设定」页填写世界观，创作提示词会注入最新设定。</div>
          )}
        </section>

        <section className="novel-inspector-section">
          <div className="novel-inspector-section-title"><ExperimentOutlined />写作技能</div>
          <Select
            allowClear
            placeholder="选择写作技能（可选）"
            value={selectedSkill}
            onChange={(v) => onSelectSkill(v)}
            options={skills.map(s => ({ value: s.name, label: s.name }))}
            size="small"
            style={{ width: '100%' }}
          />
          {selectedSkill && selectedSkillDesc && (
            <div className="novel-skill-desc">{selectedSkillDesc}</div>
          )}
          {selectedSkill && selectedSkillApplies.length > 0 && (
            <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
              {selectedSkillApplies.map(a => <Tag key={a} style={{ marginInlineEnd: 0, fontSize: 10 }}>适用·{a}</Tag>)}
            </div>
          )}
          {skills.length === 0 && (
            <div className="novel-inspector-hint">暂无可用写作技能（将技能放入 skills 目录即可自动加载）</div>
          )}
        </section>

        <section className="novel-inspector-section">
          <div className="novel-inspector-section-title"><SettingOutlined />生成设置</div>
          <div className="novel-inspector-row">
            <span>目标字数</span>
            <InputNumber
              min={1000} max={20000} step={500} size="small"
              value={minWords}
              onChange={(v) => onMinWordsChange((v as number) || 5000)}
              style={{ width: 120 }}
            />
          </div>
          <div className="novel-inspector-row">
            <span>生成温度</span>
            <Select
              size="small"
              value={temperature}
              onChange={(v) => onTemperatureChange(v as number)}
              options={[
                { value: 0, label: '默认' },
                { value: 0.7, label: '0.7 · 稳妥' },
                { value: 0.8, label: '0.8 · 平衡' },
                { value: 0.9, label: '0.9 · 灵动' },
                { value: 1.0, label: '1.0 · 大胆' },
              ]}
              style={{ width: 120 }}
            />
          </div>
          <div className="novel-inspector-hint" style={{ fontSize: 11 }}>
            目标字数不足时自动续写；温度越高文风越奔放。
          </div>
        </section>

        <section className="novel-inspector-section">
          <div className="novel-inspector-section-title"><BulbOutlined />剧情方向</div>
          <Button block icon={<BulbOutlined />} onClick={() => onOpenWizard(prevChapterHint)}>
            构思剧情方向
          </Button>
          <Input.TextArea
            rows={3}
            value={directPlot}
            onChange={e => onDirectPlotChange(e.target.value)}
            placeholder="或直接输入剧情要求，跳过分支构思…"
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', color: C('color-text'), borderRadius: 'var(--radius-md)', fontSize: 12 }}
          />
          <Button block type="primary" icon={<ThunderboltOutlined />} onClick={onDirectGenerate} disabled={!directPlot.trim()}>
            按剧情要求直接生成
          </Button>
        </section>

        <section className="novel-inspector-section">
          <div className="novel-inspector-section-title"><BarChartOutlined />创作统计</div>
          <div className="novel-inspector-row"><span>章节数</span><b>{stats?.chapterCount ?? chapterCount}</b></div>
          <div className="novel-inspector-row"><span>累计字数</span><b>{(stats?.totalWords ?? 0).toLocaleString()} 字</b></div>
        </section>

      </div>
      )}
    </aside>
  )
}

export default CreateInspector
