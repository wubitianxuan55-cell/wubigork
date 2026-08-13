import { useState } from 'react'
import { Button, Card, Input, Popconfirm, Segmented, Space, Switch, Tag, Typography } from 'antd'
import { SettingOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { engineColor, engineIcons, engineLabel, filterEnginesByEnabled, kindOf } from './utils'
import { useModelCenter } from './context'

export function EngineSection() {
  const { engines, engineStatuses, activeEngine, testingEngine, savingEngine, editingURLs, setEditingURLs, deepseekKey, setDeepseekKeyState, deepseekKeyMasked, opencodeGoKey, setOpencodeGoKeyState, opencodeGoKeyMasked, opencodeZenKey, setOpencodeZenKeyState, opencodeZenKeyMasked, handleTestConnection, handleRefreshModels, handleSaveURL, handleToggleEngine, handleBulkToggleEngines, handleSaveDeepseekKey, handleSaveOpencodeGoKey, handleSaveOpencodeZenKey, makeModels } = useModelCenter()
  const [showEnabledOnly, setShowEnabledOnly] = useState(false)
  const [bulkBusy, setBulkBusy] = useState(false)
  const visibleEngines = filterEnginesByEnabled(engines, showEnabledOnly)
  const bulk = (enabled: boolean) => {
    setBulkBusy(true)
    handleBulkToggleEngines(enabled).finally(() => setBulkBusy(false))
  }
  return (
            <section className="mc-section" style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <div className="mc-section-head">
                <div>
                  <div className="mc-section-title"><SettingOutlined /> 引擎管理</div>
                  <div className="mc-section-desc">配置云端与本地引擎地址、连接状态、API Key 和模型分类</div>
                </div>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10, flexWrap: 'wrap', marginBottom: 12 }}>
                <Space size={8}>
                  <Button size="small" loading={bulkBusy} onClick={() => bulk(true)}>全部启用</Button>
                  <Popconfirm title="禁用全部引擎？" description="各功能将回退到无可用模型，建议谨慎操作。" onConfirm={() => bulk(false)}>
                    <Button size="small" danger loading={bulkBusy}>全部禁用</Button>
                  </Popconfirm>
                </Space>
                <Segmented size="small" value={showEnabledOnly ? 'enabled' : 'all'} onChange={(v) => setShowEnabledOnly(v === 'enabled')} options={[{ label: '全部', value: 'all' }, { label: '仅已启用', value: 'enabled' }]} />
              </div>
              {visibleEngines.map(engine => {
                const color = engineColor(engine)
                const em = makeModels(engine)
                const mc = { llm: em.filter(m => kindOf(m) === 'llm').length, tts: em.filter(m => kindOf(m) === 'tts').length, stt: em.filter(m => kindOf(m) === 'stt').length, image: em.filter(m => kindOf(m) === 'image').length, embedding: em.filter(m => kindOf(m) === 'embedding').length, rerank: em.filter(m => kindOf(m) === 'rerank').length, ocr: em.filter(m => kindOf(m) === 'ocr').length }
                return (
                  <Card key={engine.id} size="small" className="mc-panel" style={{ borderColor: engine.enabled ? `color-mix(in srgb, ${color} 28%, transparent)` : undefined }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, gap: 8, flexWrap: 'wrap' }}>
                      <Space size={8}>
                        <span style={{ fontSize: 20, color }}>{engineIcons[engine.id]}</span>
                        <div>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 14 }}>{engine.name}</Typography.Text>
                          <div style={{ marginTop: 2 }}>
                            <Tag color={color} style={{ fontSize: 10 }}>{engineLabel(engine)}</Tag>
                            {engine.is_local && <Tag color="orange" style={{ fontSize: 10 }}>本地</Tag>}
                            <Switch size="small" checked={engine.enabled} onChange={(v) => handleToggleEngine(engine, v)} />
                            {activeEngine === engine.id && (
                              <Tag color="green" style={{ fontSize: 10 }}>当前活跃</Tag>
                            )}
                          </div>
                          <div style={{ marginTop: 4, fontSize: 11, color: C('color-text-secondary') }}>
                            默认模型：{engine.default_model || '未设置'}
                          </div>
                        </div>
                      </Space>
                      <Space size={4}>
                        <Button size="small" onClick={() => handleTestConnection(engine.id)} loading={testingEngine === engine.id} disabled={!engine.enabled} style={{ fontSize: 11 }}>测试连接</Button>
                        <Button size="small" onClick={() => handleRefreshModels(engine.id)} loading={testingEngine === engine.id} disabled={!engine.enabled} style={{ fontSize: 11 }}>刷新</Button>
                      </Space>
                    </div>
                    <div style={{ display: 'flex', gap: 8, marginBottom: 8, flexWrap: 'wrap' }}>
                      {mc.llm > 0 && <Tag style={{ fontSize: 10 }}>{mc.llm} 语言</Tag>}
                      {mc.tts > 0 && <Tag color="purple" style={{ fontSize: 10 }}>{mc.tts} TTS</Tag>}
                      {mc.stt > 0 && <Tag color="blue" style={{ fontSize: 10 }}>{mc.stt} STT</Tag>}
                      {mc.image > 0 && <Tag color="orange" style={{ fontSize: 10 }}>{mc.image} 图片</Tag>}
                      {mc.embedding > 0 && <Tag color="cyan" style={{ fontSize: 10 }}>{mc.embedding} Embedding</Tag>}
                      {mc.rerank > 0 && <Tag color="geekblue" style={{ fontSize: 10 }}>{mc.rerank} Rerank</Tag>}
                      {mc.ocr > 0 && <Tag color="gold" style={{ fontSize: 10 }}>{mc.ocr} OCR</Tag>}
                      {em.length === 0 && <Tag style={{ fontSize: 10 }}>暂无模型</Tag>}
                    </div>
                    {engine.type !== 'xai' && engine.type !== 'deepseek' && engine.type !== 'opencode-go' && engine.type !== 'opencode-zen' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={editingURLs[engine.id] || ''} onChange={e => setEditingURLs(prev => ({ ...prev, [engine.id]: e.target.value }))} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={() => handleSaveURL(engine)} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存</Button>
                      </Space.Compact>
                    )}
                    {engine.type === 'deepseek' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={deepseekKey} onChange={e => setDeepseekKeyState(e.target.value)} placeholder={deepseekKeyMasked || 'sk-...'} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={handleSaveDeepseekKey} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存 Key</Button>
                      </Space.Compact>
                    )}
                    {engine.type === 'opencode-go' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={opencodeGoKey} onChange={e => setOpencodeGoKeyState(e.target.value)} placeholder={opencodeGoKeyMasked || 'oc-...（opencode.ai 订阅获取）'} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={handleSaveOpencodeGoKey} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存 Key</Button>
                      </Space.Compact>
                    )}
                    {engine.type === 'opencode-zen' && (
                      <Space.Compact style={{ width: '100%' }}>
                        <Input size="small" value={opencodeZenKey} onChange={e => setOpencodeZenKeyState(e.target.value)} placeholder={opencodeZenKeyMasked || 'zen-...（opencode.ai/auth 获取）'} disabled={!engine.enabled} style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border-subtle)', color: C('color-text'), fontSize: 12 }} />
                        <Button size="small" onClick={handleSaveOpencodeZenKey} loading={savingEngine === engine.id} disabled={!engine.enabled}>保存 Key</Button>
                      </Space.Compact>
                    )}
                    {engineStatuses[engine.id] && (
                      <div style={{ marginTop: 6, fontSize: 11 }}>
                        {engineStatuses[engine.id].connected
                          ? <span style={{ color: '#34d399' }}>✓ 已连接（{engineStatuses[engine.id].model_count} 个模型{engineStatuses[engine.id].latency_ms ? `，${engineStatuses[engine.id].latency_ms}ms` : ''}）</span>
                          : <span style={{ color: '#fb7185' }}>✗ {engineStatuses[engine.id].error}</span>}
                        {engineStatuses[engine.id].last_checked && (
                          <span style={{ color: 'var(--md-sys-color-text-secondary)', marginLeft: 8 }}>上次检查 {engineStatuses[engine.id].last_checked}</span>
                        )}
                      </div>
                    )}
                  </Card>
                )
              })}
            </section>
  )
}
