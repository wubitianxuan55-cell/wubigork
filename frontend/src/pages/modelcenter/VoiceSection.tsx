import { Button, Card, Tag, Typography } from 'antd'
import { AudioOutlined, SoundOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { engineColor, engineLabel } from './utils'
import { useModelCenter } from './context'

export function VoiceSection() {
  const { voiceCfg, ttsModels, sttModels, handleSetVoiceModel } = useModelCenter()
  return (
            <section className="mc-section">
              {/* 三段激活模型汇总（模型中心 → 语音管道） */}
              <Card className="mc-panel" style={{ marginBottom: 16 }}>
                <div className="mc-section-head" style={{ marginBottom: 0 }}>
                  <div>
                    <div className="mc-section-title"><SoundOutlined /> 语音管道</div>
                    <div className="mc-section-desc">识别、对话、合成三段模型；点击卡片切换并自动持久化</div>
                  </div>
                </div>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', fontSize: 12 }}>
                  <Tag color={voiceCfg.stt.model ? 'blue' : 'default'} style={{ fontSize: 11 }}>
                    <AudioOutlined /> 识别 {voiceCfg.stt.model || '自动'}
                  </Tag>
                  <Tag color={voiceCfg.llm.model ? 'green' : 'default'} style={{ fontSize: 11 }}>
                    对话 {voiceCfg.llm.model || '默认'}
                  </Tag>
                  <Tag color={voiceCfg.tts.model ? 'purple' : 'default'} style={{ fontSize: 11 }}>
                    <SoundOutlined /> 合成 {voiceCfg.tts.model || '自动'}
                  </Tag>
                </div>
              </Card>

              {ttsModels.length === 0 && sttModels.length === 0 ? (
                <div className="mc-empty">
                  <SoundOutlined className="mc-empty-icon" />
                  <Typography.Text style={{ color: C('color-text'), fontSize: 13 }}>未发现语音模型</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 6 }}>
                    请先在「引擎管理」中刷新模型列表（Herdsman 本地引擎可提供 whisper / qwen3-tts 等）
                  </Typography.Text>
                </div>
              ) : (
                <>
                  {ttsModels.length > 0 && (
                    <div style={{ marginBottom: 22 }}>
                      <div className="mc-section-title" style={{ marginBottom: 10 }}><SoundOutlined /> TTS 语音合成</div>
                      <div className="mc-grid">
                        {ttsModels.map(m => {
                          const active = voiceCfg.tts.engine === m.engineId && voiceCfg.tts.model === m.modelId
                          return (
                            <Card key={`${m.engineId}:${m.modelId}`} size="small" className={`mc-model-card${active ? ' is-active' : ''}`}>
                              <div className="mc-model-name">{m.modelName}</div>
                              <div className="mc-model-meta">
                                <Tag color={engineColor(m)} style={{ fontSize: 10 }}>{engineLabel(m)}</Tag>
                                <Tag color="purple" style={{ fontSize: 10 }}>TTS</Tag>
                                <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                              </div>
                              <div className="mc-model-foot">
                                {active && <Tag color="purple" style={{ fontSize: 10 }}>语音合成中</Tag>}
                                <Button size="small" type={active ? 'primary' : 'default'} icon={<SoundOutlined />}
                                  onClick={() => handleSetVoiceModel('tts', m.engineId, m.modelId)}
                                  style={{ fontSize: 11, marginLeft: 'auto' }}>{active ? '已设为语音合成' : '设为语音合成'}</Button>
                              </div>
                            </Card>
                          )
                        })}
                      </div>
                    </div>
                  )}
                  {sttModels.length > 0 && (
                    <div>
                      <div className="mc-section-title" style={{ marginBottom: 10 }}><AudioOutlined /> STT 语音识别</div>
                      <div className="mc-grid">
                        {sttModels.map(m => {
                          const active = voiceCfg.stt.engine === m.engineId && voiceCfg.stt.model === m.modelId
                          return (
                            <Card key={`${m.engineId}:${m.modelId}`} size="small" className={`mc-model-card${active ? ' is-active' : ''}`}>
                              <div className="mc-model-name">{m.modelName}</div>
                              <div className="mc-model-meta">
                                <Tag color={engineColor(m)} style={{ fontSize: 10 }}>{engineLabel(m)}</Tag>
                                <Tag color="blue" style={{ fontSize: 10 }}>STT</Tag>
                                <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                              </div>
                              <div className="mc-model-foot">
                                {active && <Tag color="blue" style={{ fontSize: 10 }}>语音识别中</Tag>}
                                <Button size="small" type={active ? 'primary' : 'default'} icon={<AudioOutlined />}
                                  onClick={() => handleSetVoiceModel('asr', m.engineId, m.modelId)}
                                  style={{ fontSize: 11, marginLeft: 'auto' }}>{active ? '已设为语音识别' : '设为语音识别'}</Button>
                              </div>
                            </Card>
                          )
                        })}
                      </div>
                    </div>
                  )}
                </>
              )}
            </section>
  )
}
