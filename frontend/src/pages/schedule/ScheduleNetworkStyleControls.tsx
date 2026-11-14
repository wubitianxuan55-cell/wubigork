/**
 * ScheduleNetworkStyleControls — ZPert W4 图面参数。
 *
 * 四个显示开关与节点/箭线样式走同一个 `networkStyle` 写入入口，避免功能区、
 * 视图组件各持一份状态。横道双标尺不受这些开关影响；本面板明确标注作用域，
 * 防止用户在横道页误以为“节点时间”会改变任务表。
 */
import React from 'react'
import { InputNumber, Select, Switch } from 'antd'
import type { SchedNetworkStyle } from '../../schedule/types'
import { resolveNetworkDisplay } from '../../schedule/networkDisplay'

interface ScheduleNetworkStyleControlsProps {
  style?: SchedNetworkStyle
  onChange: (patch: Partial<SchedNetworkStyle>) => void
  /** 当前是否在 AOA/PDM 网络页；仅影响提示，不阻断预配置。 */
  networkView: boolean
}

const DISPLAY_ROWS = [
  { key: 'showDotTime', label: '节点时间', detail: 'AOA 节点最早/最迟时间' },
  { key: 'showDuration', label: '工期', detail: '关系箭线上的工期/时距' },
  { key: 'showSixPara', label: '六参数', detail: 'ES · EF · LS · LF · TF · FF' },
  { key: 'showStartStopTime', label: '起止时间', detail: '任务的计划开始/计划完成' },
] as const

export const ScheduleNetworkStyleControls: React.FC<ScheduleNetworkStyleControlsProps> = ({ style, onChange, networkView }) => {
  const flags = resolveNetworkDisplay(style)
  return (
    <div className="sched-network-style-controls" data-testid="sched-network-style-controls">
      <div className="sched-network-style-heading">
        <strong>图上内容</strong>
        <span>{networkView ? '随工程文件保存' : '作用于网络图，切换视图后生效'}</span>
      </div>
      <div className="sched-network-style-grid">
        {DISPLAY_ROWS.map((row) => (
          <label key={row.key} className="sched-network-style-row" title={row.detail}>
            <span>
              <span className="sched-network-style-label">{row.label}</span>
              <span className="sched-network-style-detail">{row.detail}</span>
            </span>
            <Switch
              size="small"
              checked={flags[row.key]}
              onChange={(checked) => onChange({ [row.key]: checked })}
              data-testid={`sched-network-toggle-${row.key}`}
            />
          </label>
        ))}
      </div>

      <div className="sched-network-style-heading sched-network-style-heading-second">
        <strong>节点、箭线与刻度</strong>
        <span>几何与标尺同源，不改 CPM 结果</span>
      </div>
      <div className="sched-network-style-fields">
        <label>
          <span>节点形状</span>
          <Select
            size="small"
            value={style?.dotStyle ?? 'circle'}
            options={[
              { value: 'circle', label: '圆形事件' },
              { value: 'rect', label: '矩形事件' },
            ]}
            onChange={(dotStyle) => onChange({ dotStyle })}
            data-testid="sched-network-style-dot"
          />
        </label>
        <label>
          <span>节点半径</span>
          <InputNumber
            size="small"
            min={10}
            max={40}
            value={style?.nodeRadius ?? 16}
            onChange={(nodeRadius) => nodeRadius != null && onChange({ nodeRadius })}
            suffix="px"
            data-testid="sched-network-style-radius"
          />
        </label>
        <label>
          <span>箭头样式</span>
          <Select
            size="small"
            value={style?.arrowStyle ?? 'solid'}
            options={[
              { value: 'solid', label: '实心' },
              { value: 'hollow', label: '空心' },
            ]}
            onChange={(arrowStyle) => onChange({ arrowStyle })}
            data-testid="sched-network-style-arrow"
          />
        </label>
        <label>
          <span>箭头高度</span>
          <InputNumber
            size="small"
            min={4}
            max={20}
            value={style?.headHeight ?? 9}
            onChange={(headHeight) => headHeight != null && onChange({ headHeight })}
            suffix="px"
            data-testid="sched-network-style-head"
          />
        </label>
        <label>
          <span>线条宽度</span>
          <InputNumber
            size="small"
            min={0.5}
            max={8}
            step={0.5}
            value={style?.lineWidth ?? 1.5}
            onChange={(lineWidth) => lineWidth != null && onChange({ lineWidth })}
            suffix="px"
            data-testid="sched-network-style-line"
          />
        </label>
        <label className="sched-network-style-field-wide">
          <span>名称排布</span>
          <Select
            size="small"
            value={style?.namePos ?? 'auto'}
            options={[
              { value: 'auto', label: '自动' },
              { value: 'horizontal', label: '横向' },
              { value: 'vertical', label: '纵向' },
            ]}
            onChange={(namePos) => onChange({ namePos })}
            data-testid="sched-network-style-name"
          />
        </label>
        <label>
          <span>刻度线</span>
          <Select
            size="small"
            value={style?.tickLines ?? 'small'}
            options={[
              { value: 'none', label: '不画' },
              { value: 'small', label: '小刻度' },
              { value: 'large', label: '大刻度' },
            ]}
            onChange={(tickLines) => onChange({ tickLines })}
            data-testid="sched-network-style-ticks"
          />
        </label>
        <label>
          <span>布图密度</span>
          <Select
            size="small"
            value={style?.density ?? 'normal'}
            options={[
              { value: 'compact', label: '紧凑' },
              { value: 'normal', label: '常规' },
              { value: 'loose', label: '疏松' },
            ]}
            onChange={(density) => onChange({ density })}
            data-testid="sched-network-style-density"
          />
        </label>
        <label>
          <span>层高 1</span>
          <InputNumber
            size="small"
            min={20}
            max={2000}
            step={12}
            value={style?.layerHeight1 ?? 112}
            onChange={(layerHeight1) => layerHeight1 != null && onChange({ layerHeight1 })}
            suffix="px"
            data-testid="sched-network-style-layer1"
          />
        </label>
        <label>
          <span>层高 2</span>
          <InputNumber
            size="small"
            min={20}
            max={2000}
            step={12}
            value={style?.layerHeight2 ?? 60}
            onChange={(layerHeight2) => layerHeight2 != null && onChange({ layerHeight2 })}
            suffix="px"
            data-testid="sched-network-style-layer2"
          />
        </label>
        <label className="sched-network-style-row sched-network-style-field-wide" title="相邻事件纵向错开，减少交叉（仅自动布局）">
          <span>
            <span className="sched-network-style-label">错层排布</span>
            <span className="sched-network-style-detail">仅自动布图；手动 pin 后不生效</span>
          </span>
          <Switch
            size="small"
            checked={style?.stagger === true}
            onChange={(stagger) => onChange({ stagger })}
            data-testid="sched-network-style-stagger"
          />
        </label>
        <label className="sched-network-style-row sched-network-style-field-wide" title="隐藏非关键、非虚工作的关系线">
          <span>
            <span className="sched-network-style-label">隐藏非重要关系线</span>
            <span className="sched-network-style-detail">保留关键线路与虚工作拓扑</span>
          </span>
          <Switch
            size="small"
            checked={style?.hideUnimportantLink === true}
            onChange={(hideUnimportantLink) => onChange({ hideUnimportantLink })}
            data-testid="sched-network-style-hidelink"
          />
        </label>
      </div>
    </div>
  )
}
