// EngineSection.tsx — 引擎管理主区（2026-10-27 瘦身重组拆件：单卡渲染外迁
// EngineCard.tsx、自定义引擎表单外迁 CustomEngineForm.tsx，本文件只留编排——
// SectionHead 工具行 + 引擎卡网格映射 + 批量启停 + 编辑表单的展开编排
// （editingCustomId/savedPrices 会话级价目记忆留在编排层，跨开合持久）。
import { useState } from 'react'
import { Button, Popconfirm, Segmented } from 'antd'
import { PlusOutlined, SettingOutlined } from '@ant-design/icons'
import { SectionHead } from './ui'
import { filterEnginesByEnabled } from './utils'
import type { EngineConfig } from '../../api/engines'
import { useModelCenterActions, useModelCenterState } from './context'
import { EngineCard } from './EngineCard'
import { AddCustomEngineForm, type EnginePrices } from './CustomEngineForm'

export function EngineSection() {
  const { engines } = useModelCenterState()
  const { handleBulkToggleEngines } = useModelCenterActions()

  const [showEnabledOnly, setShowEnabledOnly] = useState(false)
  const [bulkBusy, setBulkBusy] = useState(false)
  const [showAddForm, setShowAddForm] = useState(false)
  // 编辑表单编排：当前展开编辑的引擎 id + 本会话已保存价目（saveEngine 成功后
  // engines 不刷新，重开表单优先显示已保存值，防回退到列表旧值）。
  const [editingCustomId, setEditingCustomId] = useState<string | null>(null)
  const [savedPrices, setSavedPrices] = useState<Record<string, EnginePrices>>({})

  const visibleEngines = filterEnginesByEnabled(engines, showEnabledOnly)
  const bulk = (enabled: boolean) => {
    setBulkBusy(true)
    handleBulkToggleEngines(enabled).finally(() => setBulkBusy(false))
  }

  return (
    <section className="mc-section">
      <SectionHead
        icon={<SettingOutlined />}
        title="引擎管理"
        desc="配置云端与本地引擎地址、连接状态、API Key 和模型分类"
        extra={(
          <>
            <Button
              size="small"
              icon={<PlusOutlined />}
              onClick={() => setShowAddForm(v => !v)}
            >
              添加自定义引擎
            </Button>
            <Button size="small" loading={bulkBusy} onClick={() => bulk(true)}>全部启用</Button>
            <Popconfirm
              title="禁用全部引擎？"
              description="各功能将回退到无可用模型，建议谨慎操作。"
              onConfirm={() => bulk(false)}
            >
              <Button size="small" danger loading={bulkBusy}>全部禁用</Button>
            </Popconfirm>
            <Segmented
              size="small"
              value={showEnabledOnly ? 'enabled' : 'all'}
              onChange={(v) => setShowEnabledOnly(v === 'enabled')}
              options={[
                { label: '全部', value: 'all' },
                { label: '仅已启用', value: 'enabled' },
              ]}
            />
          </>
        )}
      />

      {showAddForm && <AddCustomEngineForm onClose={() => setShowAddForm(false)} />}

      {visibleEngines.map(engine => (
        <EngineCard
          key={engine.id}
          engine={engine}
          editingCustom={editingCustomId === engine.id}
          savedPrice={savedPrices[engine.id]}
          onToggleEdit={(e: EngineConfig | null) => setEditingCustomId(e ? e.id : null)}
          onPriceSaved={(id, prices) => setSavedPrices(prev => ({ ...prev, [id]: prices }))}
        />
      ))}
    </section>
  )
}
