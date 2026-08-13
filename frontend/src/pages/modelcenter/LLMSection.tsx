import { useState } from 'react'
import { Button, Card, Input, Space, Tag, Typography } from 'antd'
import { CaretRightOutlined, CheckCircleOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { engineColor, engineIcons, engineLabel } from './utils'
import { useModelCenter } from './context'

export function LLMSection() {
  const { engines, llmModels, engineStatuses, testingEngine, handleTestConnection, handleRefreshModels, handleStartModel, isModelActive } = useModelCenter()
  const [llmSearch, setLlmSearch] = useState('')
  return (
            <section className="mc-section">
              <div className="mc-section-head">
                <div>
                  <div className="mc-section-title"><ThunderboltOutlined /> 语言模型</div>
                  <div className="mc-section-desc">按引擎浏览和切换聊天、办公、写作等使用的 LLM 模型</div>
                </div>
              <Input.Search
                allowClear
                placeholder="搜索模型名称"
                value={llmSearch}
                onChange={e => setLlmSearch(e.target.value)}
                style={{ maxWidth: 320 }}
              />
              </div>
              {llmModels.length === 0 && (
                <div className="mc-empty">
                  <ThunderboltOutlined className="mc-empty-icon" />
                  <Typography.Text style={{ color: C('color-text'), fontSize: 13 }}>未发现语言模型</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, marginTop: 6 }}>请在「引擎管理」中启用引擎并刷新模型。</Typography.Text>
                </div>
              )}
              {llmModels.length > 0 && !llmModels.some(m => !llmSearch || m.modelName.toLowerCase().includes(llmSearch.toLowerCase())) && (
                <div className="mc-empty">
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13 }}>没有匹配「{llmSearch}」的模型</Typography.Text>
                </div>
              )}
              {engines.filter(e => e.enabled).map(engine => {
                const engineModels = llmModels.filter(m => m.engineId === engine.id && (!llmSearch || m.modelName.toLowerCase().includes(llmSearch.toLowerCase())))
                if (engineModels.length === 0) return null
                const color = engineColor(engine)
                return (
                  <div key={engine.id} style={{ marginTop: 18 }}>
                    <div className="mc-panel" style={{ marginBottom: 10, padding: '10px 14px' }}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, flexWrap: 'wrap' }}>
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
                    </div>
                    <div className="mc-grid">
                      {engineModels.map(card => {
                        const active = isModelActive(card)
                        return (
                          <Card key={card.modelId} size="small" className={`mc-model-card${active ? ' is-active' : ''}`} style={{ borderColor: active ? color : undefined }}>
                            <div className="mc-model-name" style={{ color: active ? color : C('color-text') }}>{card.modelName}</div>
                            <div className="mc-model-meta">
                              <Tag color={color} style={{ fontSize: 10, margin: 0 }}>{engineLabel(card)}</Tag>
                              {card.modelId === engine.default_model && (
                                <Tag color="cyan" style={{ fontSize: 10, margin: 0 }}>默认</Tag>
                              )}
                              <Tag color={card.status === 'stopped' ? 'default' : active ? 'green' : 'blue'} style={{ fontSize: 10, margin: 0 }}>{card.status === 'stopped' ? '已停止' : active ? '运行中' : '就绪'}</Tag>
                            </div>
                            <div className="mc-model-foot">
                              <span className="mc-status">
                                <i className={`mc-status-dot ${card.status === 'running' ? 'is-running' : ''}`} />
                                {card.status === 'running' ? '运行中' : card.status === 'stopped' ? '已停止' : '就绪'}
                              </span>
                              <Button type={active ? 'default' : 'primary'} size="small" icon={active ? <CheckCircleOutlined /> : <CaretRightOutlined />} onClick={() => handleStartModel(card)} disabled={active} style={{ borderRadius: 8, fontSize: 11, marginLeft: 'auto' }}>{active ? '已启动' : '启动'}</Button>
                            </div>
                          </Card>
                        )
                      })}
                    </div>
                  </div>
                )
              })}
            </section>
  )
}
