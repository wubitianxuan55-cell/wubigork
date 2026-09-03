// ReadingPrefsPanel — 阅读排版面板（字号/行距/版宽/主题/亮度/滚屏）。
// 自 ChapterPage 阅读偏好 Popover 的 content 原样搬移：纯受控展示组件，
// 偏好值与变更回调经 props 传入（持久化仍由 ChapterPage 的 patchReadPrefs 负责），
// 选项文案/档位/钳制边界零变更。
import React from 'react'
import { Button, Segmented, Slider } from 'antd'
import {
  clampFontSize, clampBrightness, clampAutoScrollSpeed,
  type ReadingColumn, type ReadingSettings, type ReadingTheme,
} from '../../utils/readingSettings'

interface ReadingPrefsPanelProps {
  prefs: ReadingSettings
  /** 局部变更（与 ChapterPage patchReadPrefs 同口径：传增量 partial） */
  onChange: (p: Partial<ReadingSettings>) => void
}

const ReadingPrefsPanel: React.FC<ReadingPrefsPanelProps> = ({ prefs, onChange }) => (
  <div className="novel-read-prefs">
    <div className="novel-read-prefs-row">
      <span className="novel-read-prefs-label">字号</span>
      <Button size="small" onClick={() => onChange({ fontSize: clampFontSize(prefs.fontSize - 1) })} disabled={prefs.fontSize <= 14} aria-label="减小字号">A−</Button>
      <span className="novel-read-prefs-val">{prefs.fontSize}</span>
      <Button size="small" onClick={() => onChange({ fontSize: clampFontSize(prefs.fontSize + 1) })} disabled={prefs.fontSize >= 24} aria-label="增大字号">A+</Button>
    </div>
    <div className="novel-read-prefs-row">
      <span className="novel-read-prefs-label">行距</span>
      <Segmented
        size="small"
        value={String(prefs.lineHeight)}
        options={[
          { label: '紧凑', value: '1.8' },
          { label: '标准', value: '2' },
          { label: '宽松', value: '2.3' },
        ]}
        onChange={(v) => onChange({ lineHeight: Number(v) as ReadingSettings['lineHeight'] })}
      />
    </div>
    <div className="novel-read-prefs-row">
      <span className="novel-read-prefs-label">版宽</span>
      <Segmented
        size="small"
        value={prefs.column}
        options={[
          { label: '窄', value: 'narrow' },
          { label: '标准', value: 'standard' },
          { label: '铺满', value: 'wide' },
        ]}
        onChange={(v) => onChange({ column: v as ReadingColumn })}
      />
    </div>
    <div className="novel-read-prefs-row">
      <span className="novel-read-prefs-label">主题</span>
      <Segmented
        size="small"
        value={prefs.theme}
        options={[
          { label: '跟随', value: 'auto' },
          { label: '米黄', value: 'sepia' },
          { label: '护眼绿', value: 'green' },
          { label: '夜间', value: 'dark' },
        ]}
        onChange={(v) => onChange({ theme: v as ReadingTheme })}
      />
    </div>
    <div className="novel-read-prefs-row">
      <span className="novel-read-prefs-label">亮度</span>
      <Slider
        min={70}
        max={120}
        step={5}
        value={prefs.brightness}
        onChange={(v) => onChange({ brightness: clampBrightness(Number(v)) })}
        style={{ flex: 1, margin: '0 6px' }}
        tooltip={{ formatter: (v?: number) => `${v}%` }}
      />
      <span className="novel-read-prefs-val">{prefs.brightness}%</span>
    </div>
    <div className="novel-read-prefs-row">
      <span className="novel-read-prefs-label">滚屏</span>
      <Segmented
        size="small"
        value={String(prefs.autoScrollSpeed)}
        options={[
          { label: '慢', value: '1' },
          { label: '2', value: '2' },
          { label: '3', value: '3' },
          { label: '4', value: '4' },
          { label: '快', value: '5' },
        ]}
        onChange={(v) => onChange({ autoScrollSpeed: clampAutoScrollSpeed(Number(v)) })}
      />
    </div>
  </div>
)

export default ReadingPrefsPanel
