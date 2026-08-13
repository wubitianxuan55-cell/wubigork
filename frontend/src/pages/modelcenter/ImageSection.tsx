import { Button, Segmented, Select } from 'antd'
import { FolderOpenOutlined, PictureOutlined } from '@ant-design/icons'
import SettingField from '../../components/SettingField'
import { ModelCard, SectionHead, StatusChip } from './ui'
import { COMFY_IMAGE_MODELS, engineLabel, imageModelDefaultFor, imageModelOptionsFor } from './utils'
import { useModelCenter } from './context'
import { openImageSaveDir } from '../../api/image'

const popupContainer = () => document.body

export function ImageSection() {
  const {
    engines, imageBackend, setImageBackend, comfyStatus, comfyBusy, handleToggleComfy,
    comfyUIURL, comfyUIPath, comfyUIPythonPath, imageModel, setImageModel, imageModels,
    imageBackendSaving, handleSaveImageBackend, imageSaveDir, setImageSaveDir,
  } = useModelCenter()

  // 后端切换时同步校正图片模型，避免「xAI 后端 + krea2 模型」的错配
  const handleBackendChange = (v: string) => {
    setImageBackend(v)
    const next = imageModelDefaultFor(v, engines)
    if (next) setImageModel(next)
  }

  const imageModelOptions = imageModelOptionsFor(imageBackend, engines, imageModel)

  return (
    <section className="mc-section">
      <SectionHead
        icon={<PictureOutlined />}
        title="图片生成"
        desc="选择图片后端、模型和存储目录；ComfyUI 可在本地一键启停"
      />

      <div className="mc-panel">
        <div className="mc-panel-body">
          <div className="mc-panel-title"><PictureOutlined /> 图片后端</div>

          <div>
            <div className="mc-field-label">生成后端</div>
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

          {imageBackend === 'comfyui' && (
            <div className="mc-field-box">
              <div className="mc-field-row">
                <span className="mc-live-title">
                  <i
                    className={`mc-status-dot ${comfyStatus.running ? 'is-ok' : ''}`}
                    style={comfyStatus.running ? undefined : { background: 'var(--mc-muted)' }}
                  />
                  ComfyUI {comfyStatus.running ? `运行中（端口 ${comfyStatus.port || 8188}）` : '未启动'}
                </span>
                <Button
                  size="small"
                  type={comfyStatus.running ? 'default' : 'primary'}
                  loading={comfyBusy}
                  onClick={handleToggleComfy}
                >
                  {comfyStatus.running ? '停止' : '启动'}
                </Button>
              </div>
              <div className="mc-field-line">
                URL：<span style={{ color: 'var(--mc-text)' }}>{comfyUIURL}</span><br />
                启动位置：<span style={{ color: 'var(--mc-text)' }}>{comfyUIPath || 'C:\\AI\\ComfyUI\\ComfyUI（默认）'}</span><br />
                Python：<span style={{ color: 'var(--mc-text)' }}>{comfyUIPythonPath || 'C:\\AI\\ComfyUI\\standalone-env\\python.exe（默认）'}</span>
              </div>
            </div>
          )}

          <div>
            <div className="mc-field-label">图片模型</div>
            <Select
              size="middle"
              style={{ width: 320, maxWidth: '100%' }}
              value={imageModel}
              onChange={setImageModel}
              getPopupContainer={popupContainer}
              options={imageModelOptions}
            />
          </div>

          <div>
            <div className="mc-field-label">图片保存目录</div>
            <div className="mc-field-row" style={{ alignItems: 'flex-start' }}>
              <div style={{ flex: 1, minWidth: 220 }}>
                <SettingField
                  label="目录（默认 Pictures/gaea）"
                  value={imageSaveDir}
                  onChange={v => setImageSaveDir(v)}
                  placeholder="默认: Pictures/gaea"
                />
              </div>
              <Button
                icon={<FolderOpenOutlined />}
                onClick={async () => {
                  try {
                    await openImageSaveDir()
                  } catch {
                    // 打开失败静默，交由桌面端处理
                  }
                }}
              >
                打开目录
              </Button>
            </div>
          </div>

          <div className="mc-field-row">
            <Button type="primary" onClick={handleSaveImageBackend} loading={imageBackendSaving}>
              保存图片后端设置
            </Button>
          </div>
        </div>
      </div>

      {(imageModels.length > 0 || imageBackend === 'comfyui') && (
        <div className="mc-engine-group">
          <div className="mc-group-title"><PictureOutlined /> 发现的图片模型</div>
          <div className="mc-grid">
            {[...COMFY_IMAGE_MODELS, ...imageModels.filter(m => m.engineId !== 'comfyui')].map(m => {
              const isComfy = m.engineId === 'comfyui'
              const running = isComfy ? comfyStatus.running : m.status === 'running'
              return (
                <ModelCard
                  key={`${m.engineId}:${m.modelId}`}
                  name={m.modelName}
                  engineId={m.engineId}
                  engineName={engineLabel(m)}
                  kindChip={<StatusChip tone="warn">图片</StatusChip>}
                  status={{
                    tone: running ? 'ok' : 'neutral',
                    text: running ? '运行中' : isComfy ? '未启动' : '已停止',
                  }}
                  action={isComfy && !running && (
                    <Button size="small" type="primary" loading={comfyBusy} onClick={handleToggleComfy}>
                      启动 ComfyUI
                    </Button>
                  )}
                />
              )
            })}
          </div>
        </div>
      )}
    </section>
  )
}
