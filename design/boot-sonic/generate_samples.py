"""Generate three blind-listening candidates for the GAEA startup signature.

These are intentional sound prototypes, not final masters. They use only
procedural synthesis plus the locally generated one-word GAEA take.
"""

from __future__ import annotations

import math
import wave
from pathlib import Path

import numpy as np
from scipy import signal
from scipy.io import wavfile


SR = 48_000
ROOT = Path(__file__).resolve().parent
RAW_VOICE = ROOT / "raw-gaea-name.wav"
RNG = np.random.default_rng(42)


def seconds(duration: float) -> int:
    return int(round(duration * SR))


def place(target: np.ndarray, source: np.ndarray, at: float, gain: float = 1.0) -> None:
    start = seconds(at)
    end = min(len(target), start + len(source))
    if end > start:
        target[start:end] += source[: end - start] * gain


def tone(
    frequency: float,
    duration: float,
    *,
    gain: float = 1.0,
    attack: float = 0.02,
    decay: float = 0.5,
    phase: float = 0.0,
    shimmer: float = 0.18,
) -> np.ndarray:
    t = np.arange(seconds(duration), dtype=np.float64) / SR
    env = (1.0 - np.exp(-t / max(attack, 1e-4))) * np.exp(-t / max(decay, 1e-4))
    wave_data = (
        np.sin(2 * math.pi * frequency * t + phase)
        + shimmer * np.sin(2 * math.pi * frequency * 2.003 * t + phase * 0.4)
        + 0.08 * np.sin(2 * math.pi * frequency * 3.997 * t + phase * 0.7)
    )
    return env * wave_data * gain


def pad(
    frequency: float,
    duration: float,
    *,
    gain: float = 1.0,
    attack: float = 0.2,
    release: float = 1.0,
) -> np.ndarray:
    count = seconds(duration)
    t = np.arange(count, dtype=np.float64) / SR
    attack_env = np.minimum(1.0, t / attack)
    release_env = np.minimum(1.0, np.maximum(0.0, (duration - t) / release))
    lfo = 0.94 + 0.06 * np.sin(2 * math.pi * 0.17 * t + 0.8)
    return (
        np.sin(2 * math.pi * frequency * t)
        + 0.24 * np.sin(2 * math.pi * frequency * 2.002 * t + 0.35)
        + 0.09 * np.sin(2 * math.pi * frequency * 3.998 * t + 0.7)
    ) * attack_env * release_env * lfo * gain


def pulse(frequency: float, duration: float, gain: float = 1.0) -> np.ndarray:
    t = np.arange(seconds(duration), dtype=np.float64) / SR
    env = np.exp(-t / 0.09) * (1.0 - np.exp(-t / 0.004))
    return np.sin(2 * math.pi * frequency * t) * env * gain


def filtered_noise(
    duration: float,
    low: float,
    high: float,
    *,
    gain: float = 1.0,
    attack: float = 0.15,
    release: float = 1.2,
) -> np.ndarray:
    count = seconds(duration)
    t = np.arange(count, dtype=np.float64) / SR
    noise = RNG.standard_normal(count)
    sos = signal.butter(3, [low, high], btype="bandpass", fs=SR, output="sos")
    noise = signal.sosfilt(sos, noise)
    noise /= max(np.max(np.abs(noise)), 1e-8)
    env = np.minimum(1.0, t / attack) * np.minimum(1.0, np.maximum(0.0, (duration - t) / release))
    env *= 0.55 + 0.45 * np.sin(2 * math.pi * 0.23 * t + 0.2) ** 2
    return noise * env * gain


def glass(frequency: float, duration: float, gain: float = 1.0) -> np.ndarray:
    t = np.arange(seconds(duration), dtype=np.float64) / SR
    env = (1.0 - np.exp(-t / 0.006)) * np.exp(-t / 0.28)
    return (
        np.sin(2 * math.pi * frequency * t)
        + 0.34 * np.sin(2 * math.pi * frequency * 2.41 * t)
        + 0.12 * np.sin(2 * math.pi * frequency * 3.93 * t)
    ) * env * gain


def make_ir(duration: float = 1.4, decay: float = 0.38) -> np.ndarray:
    count = seconds(duration)
    t = np.arange(count, dtype=np.float64) / SR
    ir = np.empty((count, 2), dtype=np.float64)
    for channel in range(2):
        noise = RNG.standard_normal(count)
        sos = signal.butter(2, [280, 6200], btype="bandpass", fs=SR, output="sos")
        noise = signal.sosfilt(sos, noise)
        ir[:, channel] = noise * np.exp(-t / decay)
    ir /= np.max(np.abs(ir), axis=0, keepdims=True)
    return ir


IR = make_ir()


def reverb(signal_data: np.ndarray, wet: float) -> np.ndarray:
    source = np.asarray(signal_data, dtype=np.float64)
    if source.ndim == 1:
        source = np.column_stack((source, source))
    out = np.empty_like(source)
    for channel in range(2):
        tail = signal.fftconvolve(source[:, channel], IR[:, channel])[: len(source)]
        out[:, channel] = source[:, channel] * (1.0 - wet) + tail * wet
    return out


def read_voice() -> np.ndarray:
    with wave.open(str(RAW_VOICE), "rb") as source:
        if source.getnchannels() != 1:
            raise ValueError("The raw voice sample must be mono.")
        src_rate = source.getframerate()
        samples = np.frombuffer(source.readframes(source.getnframes()), dtype=np.int16)
    voice = samples.astype(np.float64) / 32768.0
    if src_rate != SR:
        gcd = math.gcd(src_rate, SR)
        voice = signal.resample_poly(voice, SR // gcd, src_rate // gcd)
    # Remove the generated tail and keep the short spoken word.
    keep = min(len(voice), seconds(0.68))
    voice = voice[:keep]
    fade = np.minimum(1.0, np.arange(len(voice)) / seconds(0.01))
    release = np.minimum(1.0, np.arange(len(voice))[::-1] / seconds(0.025))
    return voice * fade * release


def normalize(stereo: np.ndarray, peak: float = 0.93) -> np.ndarray:
    stereo = np.tanh(stereo * 1.12) / 1.12
    peak_value = np.max(np.abs(stereo))
    if peak_value > 0:
        stereo *= peak / peak_value
    return stereo


def write_wav(path: Path, stereo: np.ndarray) -> None:
    pcm = np.clip(stereo, -1.0, 1.0)
    wavfile.write(path, SR, (pcm * 32767).astype(np.int16))


def sample_a() -> np.ndarray:
    duration = 3.2
    left = np.zeros(seconds(duration), dtype=np.float64)
    right = np.zeros_like(left)

    sub = pad(48.0, 2.6, gain=0.11, attack=0.08, release=1.0)
    place(left, sub * 0.95, 0.05)
    place(right, sub, 0.05)
    place(left, pulse(54.0, 0.62, 0.38), 0.02)
    place(right, pulse(54.0, 0.62, 0.36), 0.02)

    for at, freq, gain in [
        (0.12, 98.0, 0.1),
        (0.22, 146.83, 0.08),
        (0.34, 196.0, 0.06),
        (0.5, 293.66, 0.035),
    ]:
        voice = pad(freq, 2.4 - at, gain=gain, attack=0.12, release=0.9)
        place(left, voice, at)
        place(right, voice * 0.96, at + 0.008)

    place(left, filtered_noise(2.6, 520, 4200, gain=0.022), 0.16)
    place(right, filtered_noise(2.6, 620, 4600, gain=0.022), 0.24)
    place(left, glass(783.99, 1.0, 0.035), 0.78)
    place(right, glass(1174.66, 0.9, 0.024), 1.18)
    place(right, glass(1567.98, 0.75, 0.016), 1.48)
    return normalize(reverb(np.column_stack((left, right)), wet=0.2))


def sample_b() -> np.ndarray:
    duration = 3.0
    left = np.zeros(seconds(duration), dtype=np.float64)
    right = np.zeros_like(left)

    place(left, pad(65.41, 2.5, gain=0.08, attack=0.12, release=0.9), 0.04)
    place(right, pad(65.41, 2.5, gain=0.075, attack=0.12, release=0.9), 0.06)
    for at, freq, gain in [
        (0.2, 130.81, 0.08),
        (0.35, 196.0, 0.055),
        (0.55, 293.66, 0.04),
        (0.82, 329.63, 0.026),
    ]:
        note = pad(freq, 2.2 - at, gain=gain, attack=0.08, release=0.8)
        place(left, note, at)
        place(right, note * 0.9, at + 0.012)

    for at, freq, gain, pan in [
        (0.08, 1046.5, 0.042, -0.35),
        (0.42, 783.99, 0.038, 0.45),
        (0.84, 1318.51, 0.03, -0.2),
        (1.28, 987.77, 0.027, 0.25),
    ]:
        note = glass(freq, 1.15, gain)
        place(left, note * (1 - max(pan, 0)), at)
        place(right, note * (1 + min(pan, 0)), at)

    place(left, filtered_noise(2.4, 1200, 7600, gain=0.014, attack=0.3), 0.26)
    place(right, filtered_noise(2.4, 1500, 8200, gain=0.013, attack=0.4), 0.34)
    return normalize(reverb(np.column_stack((left, right)), wet=0.32))


def sample_c() -> np.ndarray:
    duration = 3.2
    left = np.zeros(seconds(duration), dtype=np.float64)
    right = np.zeros_like(left)

    place(left, pad(48.0, 2.7, gain=0.075, attack=0.1, release=1.0), 0.02)
    place(right, pad(48.0, 2.7, gain=0.075, attack=0.1, release=1.0), 0.03)
    for at, freq, gain in [
        (0.16, 98.0, 0.055),
        (0.28, 146.83, 0.045),
        (0.6, 196.0, 0.035),
        (1.0, 293.66, 0.025),
    ]:
        note = pad(freq, 2.4 - at, gain=gain, attack=0.12, release=1.0)
        place(left, note, at)
        place(right, note * 0.96, at + 0.01)
    place(left, filtered_noise(2.3, 540, 4200, gain=0.014), 0.2)
    place(right, filtered_noise(2.3, 720, 5200, gain=0.014), 0.3)

    voice = read_voice()
    voice_gain = 0.5
    place(left, voice, 0.42, voice_gain)
    place(right, voice, 0.435, voice_gain * 0.92)
    voice_bed = reverb(voice, wet=0.48)
    place(left, voice_bed[:, 0] * 0.32, 0.42)
    place(right, voice_bed[:, 1] * 0.32, 0.435)

    place(left, glass(783.99, 1.0, 0.026), 0.94)
    place(right, glass(1174.66, 0.9, 0.019), 1.2)
    return normalize(reverb(np.column_stack((left, right)), wet=0.16), peak=0.9)


def read_standard_voice(name: str) -> np.ndarray:
    sample_rate, samples = wavfile.read(ROOT / "voice-takes" / name)
    if sample_rate != SR:
        raise ValueError(f"Unexpected voice sample rate: {sample_rate}")
    audio = samples.astype(np.float64)
    if np.max(np.abs(audio)) > 1.5:
        audio /= 32768.0
    if audio.ndim > 1:
        audio = audio.mean(axis=1)
    return audio


def voice_candidate(voice_name: str) -> np.ndarray:
    duration = 4.0
    count = seconds(duration)
    left = np.zeros(count, dtype=np.float64)
    right = np.zeros_like(left)

    # A restrained continuous bed keeps every voice take in the same brand
    # world. There is deliberately no convolution reverb or delayed voice copy.
    place(left, pad(48.0, 3.6, gain=0.025, attack=0.1, release=0.9), 0.0)
    place(right, pad(48.0, 3.6, gain=0.025, attack=0.1, release=0.9), 0.0)
    for at, frequency, gain in [
        (0.08, 98.0, 0.013),
        (0.18, 146.83, 0.01),
        (0.32, 196.0, 0.007),
    ]:
        note = pad(frequency, duration - at - 0.2, gain=gain, attack=0.14, release=0.8)
        place(left, note, at)
        place(right, note * 0.96, at + 0.01)
    place(left, filtered_noise(3.2, 620, 3800, gain=0.004, attack=0.3), 0.1)
    place(right, filtered_noise(3.2, 760, 4400, gain=0.004, attack=0.36), 0.16)

    voice = read_standard_voice(voice_name)
    voice = voice[: min(len(voice), seconds(3.35))]
    voice *= np.minimum(1.0, np.arange(len(voice)) / seconds(0.012))
    voice *= np.minimum(1.0, np.arange(len(voice))[::-1] / seconds(0.035))
    # Keep the voice completely dry and centered. The short envelope only
    # removes clicks at the edit points.
    stereo_voice = np.column_stack((voice, voice))
    voice_at = 0.24
    place(left, stereo_voice[:, 0] * 0.82, voice_at)
    place(right, stereo_voice[:, 1] * 0.82, voice_at)
    mixed = np.column_stack((left, right))
    fade_out = np.minimum(1.0, np.arange(seconds(0.25))[::-1] / seconds(0.25))
    mixed[-len(fade_out):] *= fade_out[:, None]
    return normalize(mixed, peak=0.92)


def main() -> None:
    outputs = {
        "sample-A.wav": sample_a(),
        "sample-B.wav": sample_b(),
        "sample-C.wav": sample_c(),
        "voice-takes/voice-candidate-01.wav": voice_candidate("std-voice-C.wav"),
        "voice-takes/voice-candidate-02.wav": voice_candidate("std-voice-A.wav"),
        "voice-takes/voice-candidate-03.wav": voice_candidate("std-voice-B.wav"),
    }
    for name, audio in outputs.items():
        path = ROOT / name
        path.parent.mkdir(parents=True, exist_ok=True)
        write_wav(path, audio)
        print(f"{name}: {len(audio) / SR:.2f}s")


if __name__ == "__main__":
    main()
