// ImageGenPage 拆分产物：自定义模板弹窗状态机（行为零变化，T6-10.1）
import { useCallback, useState } from 'react'
import { message } from 'antd'
import {
  loadCustomTemplates, saveCustomTemplates, generateTemplateId,
  type CustomTemplate,
} from '../data/imageTemplates'

export function useCustomTemplates() {
  const [customTemplates, setCustomTemplates] = useState<CustomTemplate[]>(() => loadCustomTemplates())
  const [customModalOpen, setCustomModalOpen] = useState(false)
  const [editingCustom, setEditingCustom] = useState<CustomTemplate | null>(null)
  const [customLabel, setCustomLabel] = useState('')
  const [customDescription, setCustomDescription] = useState('')
  const [customSize, setCustomSize] = useState('')
  const [customPrompt, setCustomPrompt] = useState('')
  const [customNegative, setCustomNegative] = useState('')

  const openCustomAdd = useCallback(() => {
    setEditingCustom(null)
    setCustomLabel('')
    setCustomDescription('')
    setCustomSize('')
    setCustomPrompt('')
    setCustomNegative('')
    setCustomModalOpen(true)
  }, [])

  const openCustomEdit = useCallback((t: CustomTemplate) => {
    setEditingCustom(t)
    setCustomLabel(t.label)
    setCustomDescription(t.description || '')
    setCustomSize(t.size || '')
    setCustomPrompt(t.prompt)
    setCustomNegative(t.negative || '')
    setCustomModalOpen(true)
  }, [])

  const saveCustom = useCallback(() => {
    if (!customLabel.trim() || !customPrompt.trim()) {
      message.warning('标签和 Prompt 不能为空')
      return
    }
    if (editingCustom) {
      const updated = customTemplates.map((t) =>
        t.id === editingCustom.id
          ? {
              ...t,
              label: customLabel, description: customDescription, size: customSize,
              prompt: customPrompt, negative: customNegative,
            }
          : t,
      )
      setCustomTemplates(updated)
      saveCustomTemplates(updated)
    } else {
      const created: CustomTemplate = {
        id: generateTemplateId(), label: customLabel, description: customDescription, size: customSize,
        prompt: customPrompt, negative: customNegative,
      }
      const updated = [...customTemplates, created]
      setCustomTemplates(updated)
      saveCustomTemplates(updated)
    }
    setCustomModalOpen(false)
    message.success(editingCustom ? '模板已更新' : '模板已添加')
  }, [customTemplates, editingCustom, customLabel, customDescription, customSize, customPrompt, customNegative])

  const deleteCustom = useCallback((id: string) => {
    const updated = customTemplates.filter((t) => t.id !== id)
    setCustomTemplates(updated)
    saveCustomTemplates(updated)
    message.success('已删除')
  }, [customTemplates])

  return {
    customTemplates, setCustomTemplates, customModalOpen, setCustomModalOpen,
    editingCustom, setEditingCustom,
    customLabel, setCustomLabel, customDescription, setCustomDescription,
    customSize, setCustomSize, customPrompt, setCustomPrompt,
    customNegative, setCustomNegative,
    openCustomAdd, openCustomEdit, saveCustom, deleteCustom,
  }
}
