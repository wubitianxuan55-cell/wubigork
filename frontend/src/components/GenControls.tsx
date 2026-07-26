import React from 'react'
import { Typography, Select, InputNumber, Button } from 'antd'
import { ShakeOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

interface Props {
  size: string
  model: string
  style: string
  seed: number
  count: number
  showModel: boolean
  onSizeChange: (v: string) => void
  onModelChange: (v: string) => void
  onStyleChange: (v: string) => void
  onSeedChange: (v: number) => void
  onCountChange: (v: number) => void
}

const SIZE_OPTIONS = [
  { label: '🟦 方形 1024×1024', value: '1024x1024' },
  { label: '🎬 宽屏 1024×576', value: '1024x576' },
  { label: '📱 竖屏 576×1024', value: '576x1024' },
]

const STYLE_PRESETS = [
  { label: '🎨 数字油画', value: '数字油画风格，电影级光影，高细节，8K' },
  { label: '📸 写实摄影', value: '写实摄影风格，自然光，超高分辨率，逼真质感' },
  { label: '🖊️ 线稿插画', value: '精致线稿风格，干净利落的线条，扁平色彩，插画风' },
  { label: '🌌 概念艺术', value: '概念艺术风格，史诗级场景，戏剧性光影，氛围感强' },
  { label: '🎭 中国水墨', value: '中国水墨画风格，写意，留白，淡雅色调，传统笔触' },
  { label: '🎪 动漫风格', value: '日系动漫风格，鲜艳色彩，精致角色，明亮光影' },
  { label: '无', value: '' },
]

const GenControls: React.FC<Props> = (props) => {
  return (
    <div>
      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 8 }}>
        尺寸
      </Typography.Text>
      <Select
        value={props.size}
        onChange={props.onSizeChange}
        options={SIZE_OPTIONS}
        style={{ width: '100%', marginBottom: 12 }}
        popupMatchSelectWidth={false}
      />

      {props.showModel && (
        <>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 8 }}>
            模型
          </Typography.Text>
          <Select
            value={props.model}
            onChange={props.onModelChange}
            style={{ width: '100%', marginBottom: 12 }}
            options={[
              { label: '🌊 Flux Dev (20步)', value: 'flux' },
              { label: '⚡ Z-Image-Turbo (8步)', value: 'z-image-turbo' },
            ]}
          />
        </>
      )}

      <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 8 }}>
        风格
      </Typography.Text>
      <Select
        value={props.style}
        onChange={props.onStyleChange}
        options={STYLE_PRESETS}
        style={{ width: '100%', marginBottom: 12 }}
        popupMatchSelectWidth={false}
      />

      <div style={{ display: 'flex', gap: 12 }}>
        <div style={{ flex: 1 }}>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>
            🎲 种子
          </Typography.Text>
          <InputNumber
            value={props.seed || undefined}
            onChange={(v) => props.onSeedChange(v || 0)}
            placeholder="随机"
            min={1}
            max={2147483647}
            style={{ width: '100%' }}
            addonAfter={
              <Button
                type="text"
                size="small"
                icon={<ShakeOutlined />}
                onClick={() => props.onSeedChange(0)}
                style={{ padding: 0, height: 20 }}
              />
            }
          />
        </div>
        <div>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>
            数量
          </Typography.Text>
          <Select
            value={props.count}
            onChange={props.onCountChange}
            style={{ width: 70 }}
            options={[
              { label: '1', value: 1 },
              { label: '2', value: 2 },
              { label: '3', value: 3 },
              { label: '4', value: 4 },
            ]}
          />
        </div>
      </div>
    </div>
  )
}

export default GenControls
