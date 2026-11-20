// EngineSection.tsx — 引擎管理主区（编排层）。引擎卡渲染见 EngineCard.tsx，
// 自定义引擎表单见 CustomEngineForm.tsx；本文件负责状态摘要、筛选、本地/云端
// 分组与批量操作。编辑态和会话价目记忆保留在编排层，跨开合不丢失。
import { useState } from 'react'
import { Button, Popconfirm, Segmented, Switch, message } from 'antd'
import {
  ApiOutlined, CloudOutlined, DesktopOutlined, LoginOutlined, LogoutOutlined,
  PlusOutlined, SettingOutlined, ThunderboltOutlined,
} from '@ant-design/icons'
import { EmptyState, KpiTile, SectionHead, StatusChip } from './ui'
import { filterEnginesByEnabled, isLocalEngine } from './utils'
import type { EngineConfig } from '../../api/engines'
import { useModelCenterActions, useModelCenterState } from './context'
import { EngineCard } from './EngineCard'
import { AddCustomEngineForm, type EnginePrices } from './CustomEngineForm'
import { useAppStore } from '../../stores/appStore'

type EngineScope = 'all' | 'local' | 'cloud'

export function EngineSection() {
  const { engines, engineStatuses } = useModelCenterState()
  const { handleBulkToggleEngines, reloadEngines } = useModelCenterActions()
  const loggedIn = useAppStore((state) => state.loggedIn)
  const login = useAppStore((state) => state.login)
  const logout = useAppStore((state) => state.logout)

  const [showEnabledOnly, setShowEnabledOnly] = useState(false)
  const [scope, setScope] = useState<EngineScope>('all')
  const [bulkBusy, setBulkBusy] = useState(false)
  const [loggingIn, setLoggingIn] = useState(false)
  const [showAddForm, setShowAddForm] = useState(false)
  // 编辑表单编排：当前展开编辑的引擎 id + 本会话已保存价目（saveEngine 成功后
  // engines 不刷新，重开表单优先显示已保存值，防回退到列表旧值）。
  const [editingCustomId, setEditingCustomId] = useState<string | null>(null)
  const [savedPrices, setSavedPrices] = useState<Record<string, EnginePrices>>({})

  const enabledCount = engines.filter(engine => engine.enabled).length
  const localCount = engines.filter(isLocalEngine).length
  const cloudCount = engines.length - localCount
  const modelCount = engines.reduce((total, engine) => total + (engine.models?.length || 0), 0)
  const checkedStatuses = engines
    .map(engine => engineStatuses[engine.id] || engine.status)
    .filter(Boolean)
  const connectedCount = checkedStatuses.filter(status => status?.connected).length

  const visibleEngines = filterEnginesByEnabled(engines, showEnabledOnly)
    .filter(engine => scope === 'all' || (scope === 'local' ? isLocalEngine(engine) : !isLocalEngine(engine)))

  const groups = [
    {
      key: 'local' as const,
      title: '本地运行时',
      description: '本机服务与受管推理进程，关注运行状态、地址与模型装载。',
      icon: <DesktopOutlined />,
      engines: visibleEngines.filter(isLocalEngine),
    },
    {
      key: 'cloud' as const,
      title: '云端服务',
      description: '远程 API 与自定义 OpenAI 兼容端点，关注密钥、连通性与模型目录。',
      icon: <CloudOutlined />,
      engines: visibleEngines.filter(engine => !isLocalEngine(engine)),
    },
  ].filter(group => (scope === 'all' || group.key === scope) && group.engines.length > 0)

  const bulk = (enabled: boolean) => {
    setBulkBusy(true)
    handleBulkToggleEngines(enabled).finally(() => setBulkBusy(false))
  }

  const handleXaiLogin = async () => {
    setLoggingIn(true)
    try {
      await login()
      await reloadEngines()
      message.success('xAI 登录成功，已刷新云端引擎')
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : 'xAI 登录失败，请检查浏览器授权是否完成')
    } finally {
      setLoggingIn(false)
    }
  }

  const handleXaiLogout = async () => {
    try {
      await logout()
      await reloadEngines()
      message.success('已退出 xAI')
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '退出 xAI 失败')
    }
  }

  const emptyState = showEnabledOnly
    ? {
      title: scope === 'local' ? '没有已启用的本地引擎' : scope === 'cloud' ? '没有已启用的云端引擎' : '没有已启用的引擎',
      hint: '关闭「仅显示已启用」可查看停用项，或先启用一个引擎再配置功能绑定。',
    }
    : scope === 'local'
      ? { title: '尚未配置本地引擎', hint: '本地服务通常使用 127.0.0.1 地址；如需接入其他本地 OpenAI 兼容服务，请添加自定义引擎。' }
      : scope === 'cloud'
        ? { title: '尚未配置云端引擎', hint: '添加云端或 OpenAI 兼容服务商后，可在这里统一管理密钥、连通性与模型目录。' }
        : { title: '尚未配置引擎', hint: '点击「添加自定义引擎」接入云端或本地模型服务；启用后可继续配置功能绑定。' }

  return (
    <section className="mc-section mc-engine-workspace" data-testid="engine-workspace">
      <SectionHead
        icon={<SettingOutlined />}
        title="引擎工作台"
        desc="按本地与云端分组管理引擎；先看整体状态，再进入单个引擎调整地址、密钥与模型目录。"
        extra={(
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            onClick={() => setShowAddForm(v => !v)}
          >
            添加自定义引擎
          </Button>
        )}
      />

      <div className="mc-engine-overview" aria-label="引擎状态摘要">
        <KpiTile
          icon={<ApiOutlined />}
          label="引擎"
          value={engines.length}
          hint={`${enabledCount} 个已启用`}
        />
        <KpiTile
          icon={<ThunderboltOutlined />}
          label="连接正常"
          value={connectedCount}
          hint={checkedStatuses.length ? `${checkedStatuses.length} 个已有检测结果` : '尚未检测'}
        />
        <KpiTile
          icon={<DesktopOutlined />}
          label="本地"
          value={localCount}
          hint={`${cloudCount} 个云端服务`}
        />
        <KpiTile
          icon={<CloudOutlined />}
          label="已发现模型"
          value={modelCount}
          hint="含语言、语音、图片与检索模型"
        />
      </div>

      <div className={`mc-engine-account${loggedIn ? ' is-connected' : ''}`} data-testid="xai-account">
        <div className="mc-engine-account__identity">
          <span className="mc-engine-account__mark" aria-hidden="true"><CloudOutlined /></span>
          <div className="mc-engine-account__copy">
            <strong>xAI 账号</strong>
            <span>
              {loggedIn
                ? '已连接 xAI，可直接使用 Grok 云端模型'
                : '登录后接入 Grok 云端模型，授权在系统浏览器中完成'}
            </span>
          </div>
        </div>
        <div className="mc-engine-account__actions">
          <StatusChip tone={loggedIn ? 'ok' : 'neutral'} dot>
            {loggedIn ? '已连接' : '未登录'}
          </StatusChip>
          {loggedIn ? (
            <Button size="small" icon={<LogoutOutlined />} onClick={() => void handleXaiLogout()}>
              退出 xAI
            </Button>
          ) : (
            <Button
              type="primary"
              size="small"
              icon={<LoginOutlined />}
              loading={loggingIn}
              onClick={() => void handleXaiLogin()}
            >
              登录 xAI
            </Button>
          )}
        </div>
      </div>

      <div className="mc-engine-toolbar">
        <div className="mc-engine-scope" role="group" aria-label="引擎范围">
          <Segmented
            size="small"
            value={scope}
            onChange={(value) => setScope(value as EngineScope)}
            options={[
              { label: '全部', value: 'all' },
              { label: '本地', value: 'local' },
              { label: '云端', value: 'cloud' },
            ]}
          />
        </div>
        <label className="mc-engine-enabled-filter">
          <Switch
            size="small"
            checked={showEnabledOnly}
            onChange={setShowEnabledOnly}
            aria-label="仅显示已启用引擎"
          />
          <span>仅显示已启用</span>
        </label>
        <span className="mc-engine-visible-count" aria-live="polite">
          {visibleEngines.length} / {engines.length} 个引擎
        </span>
        <div className="mc-engine-toolbar-actions">
          <Button size="small" loading={bulkBusy} disabled={engines.length === 0} onClick={() => bulk(true)}>
            全部启用
          </Button>
          <Popconfirm
            title="禁用全部引擎？"
            description="各功能将回退到无可用模型，建议谨慎操作。"
            onConfirm={() => bulk(false)}
          >
            <Button size="small" danger loading={bulkBusy} disabled={engines.length === 0}>全部禁用</Button>
          </Popconfirm>
        </div>
      </div>

      {showAddForm && <AddCustomEngineForm onClose={() => setShowAddForm(false)} />}

      {visibleEngines.length === 0 && (
        <div data-testid="engine-empty">
          <EmptyState
            icon={<SettingOutlined />}
            title={emptyState.title}
            hint={emptyState.hint}
          />
        </div>
      )}

      <div className="mc-engine-groups">
        {groups.map(group => (
          <section key={group.key} className="mc-engine-cluster" aria-labelledby={`engine-group-${group.key}`}>
            <header className="mc-engine-cluster-head">
              <span className="mc-engine-cluster-icon" aria-hidden="true">{group.icon}</span>
              <div className="mc-engine-cluster-copy">
                <h3 id={`engine-group-${group.key}`}>{group.title}</h3>
                <p>{group.description}</p>
              </div>
              <span className="mc-engine-cluster-count">{group.engines.length} 个</span>
            </header>
            <div className="mc-engine-grid" data-testid={`engine-grid-${group.key}`}>
              {group.engines.map(engine => (
                <EngineCard
                  key={engine.id}
                  engine={engine}
                  editingCustom={editingCustomId === engine.id}
                  savedPrice={savedPrices[engine.id]}
                  onToggleEdit={(e: EngineConfig | null) => setEditingCustomId(e ? e.id : null)}
                  onPriceSaved={(id, prices) => setSavedPrices(prev => ({ ...prev, [id]: prices }))}
                />
              ))}
            </div>
          </section>
        ))}
      </div>
    </section>
  )
}
