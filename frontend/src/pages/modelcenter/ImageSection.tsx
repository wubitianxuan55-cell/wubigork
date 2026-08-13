import { Button, Card, Collapse, Segmented, Select, Space, Tag, Typography, message } from 'antd'
import { FolderOpenOutlined, PictureOutlined, SettingOutlined } from '@ant-design/icons'
import SettingField from '../../components/SettingField'
import { C } from '../../utils/theme'
import { COMFY_IMAGE_MODELS, engineColor, engineLabel, imageModelDefaultFor, imageModelOptionsFor } from './utils'
import { useModelCenter } from './context'
import { openImageSaveDir } from '../../api/image'

const popupContainer = () => document.body

export function ImageSection() {
  const { engines, imageBackend, setImageBackend, comfyStatus, comfyBusy, handleToggleComfy, comfyUIURL, comfyUIPath, comfyUIPythonPath, imageModel, setImageModel, imageModels, imageBackendSaving, handleSaveImageBackend, imageSaveDir, setImageSaveDir } = useModelCenter()

  // 后端切换时同步校正图片模型，避免「xAI 后端 + krea2 模型」的错配
  const handleBackendChange = (v: string) => {
    setImageBackend(v)
    const next = imageModelDefaultFor(v, engines)
    if (next) setImageModel(next)
  }

  const imageModelOptions = imageModelOptionsFor(imageBackend, engines, imageModel)
  return (
            <section className="mc-section">
              <div className="mc-section-head">
                <div>
                  <div className="mc-section-title"><PictureOutlined /> 图片生成</div>
                  <div className="mc-section-desc">选择图片后端、模型和存储目录；ComfyUI 可在本地一键启停</div>
                </div>
              </div>
              {/* 图片后端 + ComfyUI + 模型 */}
              <Card className="mc-panel" style={{ marginBottom: 18 }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                  {/* 后端选择 */}
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, display: 'block', marginBottom: 8 }}>图片生成后端</Typography.Text>
                    <Segmented
                      value={imageBackend}
                      onChange={(v) => handleBackendChange(v as string)}
                      options={[
                        { value: 'xai', label: 'xAI 云端' },
                        { value: 'comfyui', label: 'ComfyUI 本地' },
                        { value: 'herdsman', label: 'Herdsman' },
                        { value: 'ollama', label: 'Ollama' },
                      ]}
                    />
                  </div>

                  {/* ComfyUI 配置（本机默认已写死，仅展示 + 启停） */}
                  {imageBackend === 'comfyui' && (
                    <div style={{ padding: '12px 14px', borderRadius: 10, background: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-subtle)' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                        <span style={{
                          width: 8, height: 8, borderRadius: '50%',
                          background: comfyStatus.running ? '#22c55e' : '#64748b',
                          boxShadow: comfyStatus.running ? '0 0 8px #22c55e' : 'none',
                        }} />
                        <Typography.Text style={{ color: C('color-text'), fontSize: 13, fontWeight: 600 }}>
                          ComfyUI {comfyStatus.running ? `运行中 (端口 ${comfyStatus.port || 8188})` : '未启动'}
                        </Typography.Text>
                        <Button size="small" type={comfyStatus.running ? 'default' : 'primary'} loading={comfyBusy}
                          onClick={handleToggleComfy} style={{ fontSize: 11 }}>
                          {comfyStatus.running ? '⏹ 停止' : '▶ 启动'}
                        </Button>
                      </div>
                      <div style={{ fontSize: 11, color: C('color-text-secondary'), marginTop: 8, lineHeight: 1.8 }}>
                        <div>URL：<span style={{ color: C('color-text') }}>{comfyUIURL}</span></div>
                        <div>启动位置：<span style={{ color: C('color-text') }}>{comfyUIPath || 'C:\\AI\\ComfyUI\\ComfyUI（默认）'}</span></div>
                        <div>Python：<span style={{ color: C('color-text') }}>{comfyUIPythonPath || 'C:\\AI\\ComfyUI\\standalone-env\\python.exe（默认）'}</span></div>
                      </div>
                    </div>
                  )}

                  {/* 图片模型选择 */}
                  <div>
                    <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, display: 'block', marginBottom: 8 }}>图片模型</Typography.Text>
                    <Select
                      size="middle" style={{ width: 320 }} value={imageModel}
                      onChange={setImageModel}
                      getPopupContainer={popupContainer}
                      options={imageModelOptions}
                    />
                  </div>

                  {/* 保存 */}
                  <div>
                    <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>
                      保存图片后端设置
                    </Button>
                  </div>
                </div>
              </Card>

              {/* 发现的图片模型（含 ComfyUI 本地模型 krea2 / z-image-turbo，状态跟随实际运行情况） */}
              {(imageModels.length > 0 || imageBackend === 'comfyui') && (
                <div style={{ marginBottom: 24 }}>
                  <div className="mc-section-title" style={{ marginBottom: 10 }}><PictureOutlined /> 发现的图片模型</div>
                  <div className="mc-grid">
                    {[...COMFY_IMAGE_MODELS, ...imageModels.filter(m => m.engineId !== 'comfyui')].map(m => {
                      const isComfy = m.engineId === 'comfyui'
                      const running = isComfy ? comfyStatus.running : m.status === 'running'
                      return (
                        <Card key={`${m.engineId}:${m.modelId}`} size="small" className="mc-model-card">
                          <div className="mc-model-name">{m.modelName}</div>
                          <div className="mc-model-meta">
                            <Tag color={engineColor(m)} style={{ fontSize: 10, margin: 0 }}>{engineLabel(m)}</Tag>
                            <Tag color="orange" style={{ fontSize: 10, margin: 0 }}>图片</Tag>
                            <Tag color={running ? 'green' : 'default'} style={{ fontSize: 10, margin: 0 }}>{running ? '运行中' : isComfy ? '未启动' : '已停止'}</Tag>
                          </div>
                          {isComfy && !running && (
                            <div className="mc-model-foot">
                              <Button size="small" type="primary" loading={comfyBusy} onClick={handleToggleComfy} style={{ fontSize: 11, marginLeft: 'auto' }}>
                                ▶ 启动 ComfyUI
                              </Button>
                            </div>
                          )}
                        </Card>
                      )
                    })}
                  </div>
                </div>
              )}
              <Collapse ghost size="small" defaultActiveKey={['img-cfg']} items={[{
                key: 'img-cfg', label: <span style={{ color: C('color-text-secondary'), fontSize: 13 }}><SettingOutlined style={{ marginRight: 6 }} />图片存储</span>,
                children: (
                  <Card style={{ background: 'var(--bg-glass)', border: '1px solid var(--border-subtle)', borderRadius: 12 }}>
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <SettingField label="图片保存目录" value={imageSaveDir} onChange={v => setImageSaveDir(v)} placeholder="默认: Pictures/gaea" />
                      <Space>
                        <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving} style={{ borderRadius: 8 }}>保存</Button>
                        <Button icon={<FolderOpenOutlined />} onClick={async () => {
                          try {
                            await openImageSaveDir()
                          } catch (err: any) {
                            message.error(err?.message || '打开目录失败')
                          }
                        }} style={{ borderRadius: 8 }}>打开目录</Button>
                      </Space>
                    </Space>
                  </Card>
                ),
              }]} />
            </section>
  )
}
