export interface BootAudio {
  context: AudioContext
  master: GainNode
  stop: () => void
}

type AudioContextCtor = new (contextOptions?: AudioContextOptions) => AudioContext

/**
 * Original procedural startup score. It deliberately avoids external audio
 * files and game music: warm facility drone, geological scan, five validation
 * pulses, and a low root-to-canopy resolution chord.
 */
export function createBootAudio(durationMs: number): BootAudio | null {
  const AudioCtor = (
    window.AudioContext ||
    (window as Window & { webkitAudioContext?: AudioContextCtor }).webkitAudioContext
  ) as AudioContextCtor | undefined
  if (!AudioCtor) return null

  let context: AudioContext
  try {
    context = new AudioCtor()
  } catch {
    return null
  }

  const master = context.createGain()
  const compressor = context.createDynamicsCompressor()
  master.gain.value = 0.28
  compressor.threshold.value = -22
  compressor.knee.value = 16
  compressor.ratio.value = 4
  compressor.attack.value = 0.004
  compressor.release.value = 0.26
  master.connect(compressor)
  compressor.connect(context.destination)

  const startAt = context.currentTime + 0.035
  const finishedAt = startAt + durationMs / 1000

  const ambient = context.createGain()
  const ambientFilter = context.createBiquadFilter()
  ambientFilter.type = 'lowpass'
  ambientFilter.frequency.value = 210
  ambientFilter.Q.value = 0.8
  ambient.gain.setValueAtTime(0.0001, startAt)
  ambient.gain.exponentialRampToValueAtTime(0.065, startAt + 0.8)
  ambient.gain.setValueAtTime(0.065, finishedAt - 1.0)
  ambient.gain.exponentialRampToValueAtTime(0.0001, finishedAt)
  ambient.connect(ambientFilter)
  ambientFilter.connect(master)

  const rootDrone = context.createOscillator()
  const soilDrone = context.createOscillator()
  rootDrone.type = 'sine'
  soilDrone.type = 'triangle'
  rootDrone.frequency.setValueAtTime(48.5, startAt)
  rootDrone.frequency.linearRampToValueAtTime(49.2, startAt + 3.4)
  soilDrone.frequency.setValueAtTime(72.2, startAt)
  soilDrone.detune.value = 4
  rootDrone.connect(ambient)
  soilDrone.connect(ambient)
  rootDrone.start(startAt)
  soilDrone.start(startAt)
  rootDrone.stop(finishedAt + 0.05)
  soilDrone.stop(finishedAt + 0.05)

  // Fine mineral air, kept quiet enough to feel like room tone.
  const noiseBuffer = context.createBuffer(1, context.sampleRate, context.sampleRate)
  const noiseData = noiseBuffer.getChannelData(0)
  for (let i = 0; i < noiseData.length; i += 1) {
    noiseData[i] = Math.random() * 2 - 1
  }
  const air = context.createBufferSource()
  const airFilter = context.createBiquadFilter()
  const airGain = context.createGain()
  air.buffer = noiseBuffer
  air.loop = true
  airFilter.type = 'bandpass'
  airFilter.frequency.value = 640
  airFilter.Q.value = 0.4
  airGain.gain.setValueAtTime(0.0001, startAt)
  airGain.gain.exponentialRampToValueAtTime(0.016, startAt + 1.3)
  airGain.gain.setValueAtTime(0.016, finishedAt - 1.2)
  airGain.gain.exponentialRampToValueAtTime(0.0001, finishedAt)
  air.connect(airFilter)
  airFilter.connect(airGain)
  airGain.connect(master)
  air.start(startAt)
  air.stop(finishedAt + 0.05)

  const scheduleTone = (
    frequency: number,
    at: number,
    duration: number,
    level: number,
    type: OscillatorType = 'sine',
  ) => {
    const oscillator = context.createOscillator()
    const gain = context.createGain()
    oscillator.type = type
    oscillator.frequency.setValueAtTime(frequency, at)
    gain.gain.setValueAtTime(0.0001, at)
    gain.gain.exponentialRampToValueAtTime(level, at + 0.014)
    gain.gain.exponentialRampToValueAtTime(0.0001, at + duration)
    oscillator.connect(gain)
    gain.connect(master)
    oscillator.start(at)
    oscillator.stop(at + duration + 0.035)
  }

  // Access scan: a rising geological sweep.
  const scanner = context.createOscillator()
  const scannerGain = context.createGain()
  const scannerFilter = context.createBiquadFilter()
  scanner.type = 'sawtooth'
  scanner.frequency.setValueAtTime(76, startAt + 0.08)
  scanner.frequency.exponentialRampToValueAtTime(286, startAt + 1.55)
  scannerFilter.type = 'lowpass'
  scannerFilter.frequency.value = 560
  scannerGain.gain.setValueAtTime(0.0001, startAt + 0.08)
  scannerGain.gain.exponentialRampToValueAtTime(0.032, startAt + 0.3)
  scannerGain.gain.exponentialRampToValueAtTime(0.0001, startAt + 1.7)
  scanner.connect(scannerFilter)
  scannerFilter.connect(scannerGain)
  scannerGain.connect(master)
  scanner.start(startAt + 0.08)
  scanner.stop(startAt + 1.74)

  const phaseTimes = [0, 0.9, 1.8, 3.0, 4.3]
  const phaseTones = [196, 261.63, 329.63, 392, 523.25]
  phaseTimes.forEach((offset, index) => {
    const at = startAt + offset
    scheduleTone(phaseTones[index], at, index === 4 ? 0.62 : 0.21, 0.055, 'triangle')
    if (index > 0) scheduleTone(phaseTones[index] * 2, at + 0.045, 0.1, 0.018, 'sine')
  })

  // Subterranean data ticks.
  const dataTicks = [0.48, 1.22, 2.28, 2.76, 3.66, 4.08, 5.18, 5.84, 6.42]
  dataTicks.forEach((offset, index) => {
    scheduleTone(980 + (index % 3) * 160, startAt + offset, 0.038, 0.011, 'square')
  })

  // Low root chord -> canopy overtone.
  const readyAt = startAt + 6.62
  scheduleTone(130.81, readyAt, 0.85, 0.04, 'triangle')
  scheduleTone(261.63, readyAt + 0.04, 0.76, 0.035, 'sine')
  scheduleTone(392, readyAt + 0.1, 0.65, 0.026, 'sine')
  scheduleTone(783.99, readyAt + 0.16, 0.5, 0.012, 'triangle')

  let stopped = false
  return {
    context,
    master,
    stop: () => {
      if (stopped) return
      stopped = true
      const now = context.currentTime
      master.gain.cancelScheduledValues(now)
      master.gain.setTargetAtTime(0.0001, now, 0.025)
      window.setTimeout(() => {
        if (context.state !== 'closed') void context.close().catch(() => {})
      }, 150)
    },
  }
}
