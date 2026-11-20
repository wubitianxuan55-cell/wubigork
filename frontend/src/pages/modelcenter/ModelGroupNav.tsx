import { AppstoreOutlined, PictureOutlined, SoundOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { COMFY_IMAGE_MODELS } from './utils'
import { useModelCenterState } from './context'

const MODEL_GROUPS = [
  { id: 'mc-model-group-llm', label: '语言', icon: <ThunderboltOutlined /> },
  { id: 'mc-model-group-image', label: '图像', icon: <PictureOutlined /> },
  { id: 'mc-model-group-voice', label: '语音', icon: <SoundOutlined /> },
  { id: 'mc-model-group-specialty', label: '专业', icon: <AppstoreOutlined /> },
] as const

/** 全模型目录的组级导航：先给出每类模型数量，再锚定到对应清单。 */
export function ModelGroupNav() {
  const { llmModels, imageModels, ttsModels, sttModels, specialtyModels } = useModelCenterState()
  const counts: Record<(typeof MODEL_GROUPS)[number]['id'], number> = {
    'mc-model-group-llm': llmModels?.length ?? 0,
    'mc-model-group-image': COMFY_IMAGE_MODELS.length
      + (imageModels ?? []).filter(model => model.engineId !== 'comfyui').length,
    'mc-model-group-voice': (ttsModels?.length ?? 0) + (sttModels?.length ?? 0),
    'mc-model-group-specialty': specialtyModels?.length ?? 0,
  }

  const jumpTo = (id: string) => {
    const target = document.getElementById(id)
    if (!target) return
    try {
      target.scrollIntoView({ behavior: 'smooth', block: 'start' })
    } catch {
      target.scrollIntoView()
    }
  }

  return (
    <nav className="mc-model-group-nav" aria-label="模型分类">
      <div className="mc-model-group-nav__head">
        <span>模型目录</span>
        <small>按类型跳转</small>
      </div>
      <div className="mc-model-group-nav__items">
        {MODEL_GROUPS.map(group => (
          <button
            key={group.id}
            type="button"
            className="mc-model-group-nav__item"
            onClick={() => jumpTo(group.id)}
          >
            <span className="mc-model-group-nav__icon" aria-hidden="true">{group.icon}</span>
            <span>{group.label}</span>
            <strong>{counts[group.id]}</strong>
          </button>
        ))}
      </div>
    </nav>
  )
}
