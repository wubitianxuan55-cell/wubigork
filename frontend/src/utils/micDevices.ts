// micDevices.ts — 麦克风设备选择（FE6-02 终局配套；原 hooks/useVoiceChat 内联段，
// V1 语音链退役后迁出为独立 util——设备选择/信号检测是采集硬件关注点，
// 与语音对话版本无关）。
// 实弹实证：本机默认录音设备可能是零信号虚拟设备（ToDesk Virtual Audio），
// getUserMedia 默认拿到的就是它——没有设备选择，真麦插上也听不见。
// deviceId 是 WebView2 按源的持久指纹，纯前端采集关注点，localStorage 持久化
// （不进后端 config——后端不做采集，为浏览器指纹 ID 开五文件配账不值）。

const MIC_DEVICE_KEY = 'gaea.voice.micDeviceId'

/** 读取已选麦克风设备 ID（空 = 跟随系统默认） */
export function loadMicDeviceId(): string {
  try { return localStorage.getItem(MIC_DEVICE_KEY) || '' } catch { return '' }
}

/** 保存麦克风设备 ID（空串 = 清除，恢复跟随系统默认） */
export function saveMicDeviceId(id: string): void {
  try {
    if (id) localStorage.setItem(MIC_DEVICE_KEY, id)
    else localStorage.removeItem(MIC_DEVICE_KEY)
  } catch { /* localStorage 不可用（隐私模式）：仅本会话内生效 */ }
}

export interface MicDeviceInfo {
  deviceId: string
  label: string
  isDefault: boolean
}

/** 列出录音输入设备。权限未授予时 label 为空——借一次瞬时采集换取可读标签
 *  （失败静默，设备仍可列只是名字叫「麦克风」）。 */
export async function listMicDevices(): Promise<MicDeviceInfo[]> {
  if (!navigator.mediaDevices?.enumerateDevices) return []
  try {
    let devs = await navigator.mediaDevices.enumerateDevices()
    if (devs.filter(d => d.kind === 'audioinput').some(d => !d.label)) {
      try {
        const s = await navigator.mediaDevices.getUserMedia({ audio: true })
        s.getTracks().forEach(t => t.stop())
        devs = await navigator.mediaDevices.enumerateDevices()
      } catch { /* 权限拒绝：按匿名设备列出 */ }
    }
    return devs
      .filter(d => d.kind === 'audioinput')
      .map(d => ({
        deviceId: d.deviceId,
        label: d.label || '麦克风',
        isDefault: d.deviceId === 'default',
      }))
  } catch {
    return []
  }
}
