import { Button, Card, Space, Tag, Typography } from 'antd'
import { AudioOutlined, SoundOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { engineColor, engineLabel } from './utils'
import { useModelCenter } from './context'

export function VoiceSection() {
  const { voiceCfg, ttsModels, sttModels, handleSetVoiceModel } = useModelCenter()
  return (
            <>
              {/* 三段激活模型汇总（模型中心 → 语音管道） */}
              <Card style={{ marginBottom: 16, background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', fontSize: 12 }}>
                  <span style={{ color: C('color-text-secondary'), fontWeight: 600, marginRight: 4 }}>语音管道：</span>
                  <Tag color={voiceCfg.stt.model ? 'blue' : 'default'} style={{ fontSize: 11 }}>
                    🎙️ 识别 {voiceCfg.stt.model || '自动'}
                  </Tag>
                  <Tag color={voiceCfg.llm.model ? 'green' : 'default'} style={{ fontSize: 11 }}>
                    💬 对话 {voiceCfg.llm.model || '默认'}
                  </Tag>
                  <Tag color={voiceCfg.tts.model ? 'purple' : 'default'} style={{ fontSize: 11 }}>
                    🔊 合成 {voiceCfg.tts.model || '自动'}
                  </Tag>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, marginLeft: 'auto' }}>
                    点击下方卡片可切换识别/合成模型（自动持久化，重启保留）
                  </Typography.Text>
                </div>
              </Card>

              {ttsModels.length === 0 && sttModels.length === 0 ? (
                <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12, textAlign: 'center', padding: 40, marginBottom: 16 }}>
                  <SoundOutlined style={{ fontSize: 32, color: C('color-text-secondary'), marginBottom: 12 }} />
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 14, display: 'block' }}>未发现语音模型</Typography.Text>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginTop: 6 }}>
                    请先在「引擎管理」中刷新模型列表（Herdsman 本地引擎可提供 whisper / qwen3-tts 等）
                  </Typography.Text>
                </Card>
              ) : (
                <>
                  {ttsModels.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>🔊 TTS 语音合成</Typography.Text>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))', gap: 10 }}>
                        {ttsModels.map(m => {
                          const active = voiceCfg.tts.engine === m.engineId && voiceCfg.tts.model === m.modelId
                          return (
                            <Card key={`${m.engineId}:${m.modelId}`} size="small" style={{
                              background: 'var(--bg-glass)',
                              border: active ? '1px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)',
                              borderRadius: 10,
                              boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)' : 'none',
                              transition: 'box-shadow 0.2s, border-color 0.2s',
                            }}>
                              <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6 }}>{m.modelName}</Typography.Text>
                              <Space>
                                <Tag color={engineColor(m)} style={{ fontSize: 10 }}>{engineLabel(m)}</Tag>
                                <Tag color="purple" style={{ fontSize: 10 }}>TTS</Tag>
                                <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                              </Space>
                              {active && <Tag color="purple" style={{ marginTop: 6, fontSize: 10 }}>● 语音合成中</Tag>}
                              <div style={{ marginTop: 8 }}>
                                <Button size="small" type={active ? 'primary' : 'default'} icon={<SoundOutlined />}
                                  onClick={() => handleSetVoiceModel('tts', m.engineId, m.modelId)}
                                  style={{ fontSize: 11 }}>{active ? '已设为语音合成' : '设为语音合成'}</Button>
                              </div>
                            </Card>
                          )
                        })}
                      </div>
                    </div>
                  )}
                  {sttModels.length > 0 && (
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text strong style={{ color: C('color-text'), fontSize: 15, display: 'block', marginBottom: 10 }}>🎙️ STT 语音识别</Typography.Text>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(230px, 1fr))', gap: 10 }}>
                        {sttModels.map(m => {
                          const active = voiceCfg.stt.engine === m.engineId && voiceCfg.stt.model === m.modelId
                          return (
                            <Card key={`${m.engineId}:${m.modelId}`} size="small" style={{
                              background: 'var(--bg-glass)',
                              border: active ? '1px solid var(--md-sys-color-primary)' : '1px solid var(--border-subtle)',
                              borderRadius: 10,
                              boxShadow: active ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 30%, transparent)' : 'none',
                              transition: 'box-shadow 0.2s, border-color 0.2s',
                            }}>
                              <Typography.Text strong style={{ color: C('color-text'), fontSize: 13, display: 'block', marginBottom: 6 }}>{m.modelName}</Typography.Text>
                              <Space>
                                <Tag color={engineColor(m)} style={{ fontSize: 10 }}>{engineLabel(m)}</Tag>
                                <Tag color="blue" style={{ fontSize: 10 }}>STT</Tag>
                                <Tag color={m.status === 'running' ? 'green' : 'default'} style={{ fontSize: 10 }}>{m.status === 'running' ? '● 运行中' : '○ 已停止'}</Tag>
                              </Space>
                              {active && <Tag color="blue" style={{ marginTop: 6, fontSize: 10 }}>● 语音识别中</Tag>}
                              <div style={{ marginTop: 8 }}>
                                <Button size="small" type={active ? 'primary' : 'default'} icon={<AudioOutlined />}
                                  onClick={() => handleSetVoiceModel('asr', m.engineId, m.modelId)}
                                  style={{ fontSize: 11 }}>{active ? '已设为语音识别' : '设为语音识别'}</Button>
                              </div>
                            </Card>
                          )
                        })}
                      </div>
                    </div>
                  )}
                </>
              )}
            </>
  )
}