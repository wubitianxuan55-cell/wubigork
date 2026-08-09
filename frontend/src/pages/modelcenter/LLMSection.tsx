import { useState } from 'react'
import { Button, Card, Input, Space, Tag, Typography } from 'antd'
import { CaretRightOutlined, CheckCircleOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { engineColor, engineIcons, engineLabel } from './utils'
import { useModelCenter } from './context'

export function LLMSection() {
  const { engines, llmModels, engineStatuses, testingEngine, handleTestConnection, handleRefreshModels, handleStartModel, isModelActive } = useModelCenter()
  const [llmSearch, setLlmSearch] = useState('')
  return (
            <>
              <Input.Search
                allowClear
                placeholder="搜索模型名称"
                value={llmSearch}
                onChange={e => setLlmSearch(e.target.value)}
                style={{ maxWidth: 320, marginBottom: 16 }}
              />
              {llmModels.length === 0 && (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 40, marginBottom: 16 }}>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14 }}>未发现语言模型。请在「引擎管理」中启用引擎并刷新模型。</Typography.Text>
                </Card>
              )}
              {llmModels.length > 0 && !llmModels.some(m => !llmSearch || m.modelName.toLowerCase().includes(llmSearch.toLowerCase())) && (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 24, marginBottom: 16 }}>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13 }}>没有匹配「{llmSearch}」的模型</Typography.Text>
                </Card>
              )}
              {engines.filter(e => e.enabled).map(engine => {
                const engineModels = llmModels.filter(m => m.engineId === engine.id && (!llmSearch || m.modelName.toLowerCase().includes(llmSearch.toLowerCase())))
                if (engineModels.length === 0) return null
                const color = engineColor(engine)
                return (
                  <div key={engine.id} style={{ marginBottom: 24 }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10, paddingBottom: 8, borderBottom: `1px solid ${color}30` }}>
                      <Space size={8}>
                        <span style={{ fontSize: 18, color }}>{engineIcons[engine.id]}</span>
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 15 }}>{engine.name}</Typography.Text>
                        <Tag color={color} style={{ fontSize: 10 }}>{engineLabel(engine)}</Tag>
                        <Tag style={{ fontSize: 10 }}>{engineModels.length} 个</Tag>
                        {engineStatuses[engine.id] && (
                          <Tag color={engineStatuses[engine.id].connected ? 'green' : 'red'} style={{ fontSize: 10 }}>
                            {engineStatuses[engine.id].connected ? '● 已连接' : '✗ 连接失败'}
                          </Tag>
                        )}
                      </Space>
                      <Space size={4}>
                        <Button size="small" onClick={() => handleTestConnection(engine.id)} loading={testingEngine === engine.id} style={{ fontSize: 11 }}>测试连接</Button>
                        <Button size="small" onClick={() => handleRefreshModels(engine.id)} loading={testingEngine === engine.id} style={{ fontSize: 11 }}>刷新模型</Button>
                      </Space>
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
                      {engineModels.map(card => {
                        const active = isModelActive(card)
                        return (
                          <Card key={card.modelId} size="small" style={{ background: active ? `linear-gradient(135deg, ${color}18, ${color}08)` : 'var(--bg-glass)', border: active ? `2px solid ${color}` : '1px solid var(--border-subtle)', borderRadius: 10 }}>
                            <Typography.Text strong style={{ color: active ? color : C('color-text'), fontSize: 13, display: 'block', marginBottom: 6, wordBreak: 'break-all' }}>{card.modelName}</Typography.Text>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                              <Tag color={color} style={{ fontSize: 10, margin: 0 }}>{engineLabel(card)}</Tag>
                              {active ? (
                                <Tag color="green" style={{ fontSize: 10, margin: 0 }}>● 运行中</Tag>
                              ) : (
                                <Tag color={card.status === 'stopped' ? 'default' : 'blue'} style={{ fontSize: 10, margin: 0 }}>{card.status === 'stopped' ? '○ 已停止' : '○ 就绪'}</Tag>
                              )}
                              <Button type={active ? 'default' : 'primary'} size="small" icon={active ? <CheckCircleOutlined /> : <CaretRightOutlined />} onClick={() => handleStartModel(card)} disabled={active} style={{ borderRadius: 8, fontSize: 11, marginLeft: 'auto' }}>{active ? '已启动' : '启动'}</Button>
                            </div>
                          </Card>
                        )
                      })}
                    </div>
                  </div>
                )
              })}
            </>
  )
}