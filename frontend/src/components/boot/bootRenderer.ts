/**
 * BootSplash WebGL2 renderer: one full-screen triangle and the fragment shader in bootShader.ts.
 *
 * createBootRenderer never throws. When WebGL2 is missing or a texture cannot be created
 * (GPU texture size limits) it returns null and the splash keeps the static plate cross-fade.
 */
import { FRAG, VERT } from './bootShader'
import type { BootPose } from './bootScene'

/** What a plate arrives as: an ImageBitmap (scaled off the main thread), a plain image, or a canvas. */
export type BootPlateSource = ImageBitmap | HTMLImageElement | HTMLCanvasElement

export interface BootTextures {
  mark: HTMLImageElement
  /** Decoded plates; ImageBitmap when the window is smaller than 4K (scaled off the main thread). */
  orbit: BootPlateSource
  valley: BootPlateSource
}

export type BootRgb = readonly [number, number, number]

export interface BootPalette {
  accent: BootRgb
  gold: BootRgb
}

export interface BootRenderer {
  /** Draws the frame for `pose` into a backing store of `width × height` device pixels. */
  render(width: number, height: number, pose: BootPose): void
  /** Frees the textures and the GL context right away — the splash is short-lived. */
  dispose(): void
}

const UNIFORMS = [
  'uRes', 'uT', 'uScene', 'uFlash', 'uAmb', 'uLogoA', 'uLogoBoost', 'uSweep', 'uComet', 'uRingA',
  'uCam1', 'uCam2', 'uAccent', 'uGold', 'uLogo', 'uPlate1', 'uPlate2', 'uLogoRect',
] as const

/** 16:9 composition: the mark is centred at 22.5% of the height and its letterform is 24.4% of it. */
const LOGO_CENTER_Y = 0.5 - 0.225
const LOGO_HALF = (0.244 * 1.16) / 2
/** Plates ship at 4K; a GPU with a smaller texture limit gets them scaled down on the CPU. */
const MAX_PLATE_EDGE = 3840

const CONTEXT_ATTRS: WebGLContextAttributes = {
  alpha: false,
  antialias: false,
  depth: false,
  stencil: false,
  powerPreference: 'high-performance',
}

function compile(gl: WebGL2RenderingContext, type: number, source: string): WebGLShader {
  const shader = gl.createShader(type)
  if (!shader) throw new Error('createShader failed')
  gl.shaderSource(shader, source)
  gl.compileShader(shader)
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    const log = gl.getShaderInfoLog(shader)
    gl.deleteShader(shader)
    throw new Error(`boot shader: ${log}`)
  }
  return shader
}

function createTexture(
  gl: WebGL2RenderingContext,
  unit: number,
  source: BootPlateSource,
  opt: { flipY: boolean; premultiply: boolean },
): WebGLTexture {
  const texture = gl.createTexture()
  if (!texture) throw new Error('createTexture failed')
  gl.activeTexture(gl.TEXTURE0 + unit)
  gl.bindTexture(gl.TEXTURE_2D, texture)
  gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, opt.flipY)
  gl.pixelStorei(gl.UNPACK_PREMULTIPLY_ALPHA_WEBGL, opt.premultiply)
  gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, source)
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
  // Both the mark (on small windows) and the plates (during the pull-back) are minified:
  // mipmaps keep them clean instead of shimmering.
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
  gl.generateMipmap(gl.TEXTURE_2D)
  return texture
}

function fitPlate(image: BootPlateSource, limit: number): BootPlateSource {
  const width = Number((image as HTMLImageElement).naturalWidth) || Number((image as ImageBitmap).width)
  const height = Number((image as HTMLImageElement).naturalHeight) || Number((image as ImageBitmap).height)
  if (!width || !height || Math.max(width, height) <= limit) return image
  const ratio = limit / Math.max(width, height)
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(width * ratio)
  canvas.height = Math.round(height * ratio)
  const ctx = canvas.getContext('2d')
  if (!ctx) return image
  ctx.drawImage(image, 0, 0, canvas.width, canvas.height)
  return canvas
}

export function createBootRenderer(
  canvas: HTMLCanvasElement,
  textures: BootTextures,
  palette: BootPalette,
): BootRenderer | null {
  let gl: WebGL2RenderingContext | null = null
  try {
    gl = canvas.getContext('webgl2', CONTEXT_ATTRS)
  } catch {
    gl = null
  }
  if (!gl) return null
  const ctx = gl

  const texturesOut: WebGLTexture[] = []
  let buffer: WebGLBuffer | null = null
  let program: WebGLProgram | null = null
  const release = () => {
    texturesOut.forEach((tex) => ctx.deleteTexture(tex))
    if (buffer) ctx.deleteBuffer(buffer)
    if (program) ctx.deleteProgram(program)
    ctx.getExtension('WEBGL_lose_context')?.loseContext()
  }

  try {
    program = ctx.createProgram()
    if (!program) throw new Error('createProgram failed')
    ctx.attachShader(program, compile(ctx, ctx.VERTEX_SHADER, VERT))
    ctx.attachShader(program, compile(ctx, ctx.FRAGMENT_SHADER, FRAG))
    ctx.linkProgram(program)
    if (!ctx.getProgramParameter(program, ctx.LINK_STATUS)) {
      throw new Error(`boot program: ${ctx.getProgramInfoLog(program)}`)
    }
    ctx.useProgram(program)

    buffer = ctx.createBuffer()
    if (!buffer) throw new Error('createBuffer failed')
    ctx.bindBuffer(ctx.ARRAY_BUFFER, buffer)
    ctx.bufferData(ctx.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), ctx.STATIC_DRAW)
    const aPos = ctx.getAttribLocation(program, 'aPos')
    ctx.enableVertexAttribArray(aPos)
    ctx.vertexAttribPointer(aPos, 2, ctx.FLOAT, false, 0, 0)

    const limit = Math.min(MAX_PLATE_EDGE, ctx.getParameter(ctx.MAX_TEXTURE_SIZE) as number)
    // The mark is composited "over" with premultiplied RGB (see the shader's foreground()).
    texturesOut.push(createTexture(ctx, 0, textures.mark, { flipY: true, premultiply: true }))
    texturesOut.push(createTexture(ctx, 1, fitPlate(textures.orbit, limit), { flipY: false, premultiply: false }))
    texturesOut.push(createTexture(ctx, 2, fitPlate(textures.valley, limit), { flipY: false, premultiply: false }))
    if (ctx.getError() !== ctx.NO_ERROR) throw new Error('boot textures rejected')

    const u = {} as Record<(typeof UNIFORMS)[number], WebGLUniformLocation | null>
    for (const name of UNIFORMS) u[name] = ctx.getUniformLocation(program, name)
    ctx.uniform1i(u.uLogo, 0)
    ctx.uniform1i(u.uPlate1, 1)
    ctx.uniform1i(u.uPlate2, 2)

    return {
      render(width, height, pose) {
        if (canvas.width !== width || canvas.height !== height) {
          canvas.width = width
          canvas.height = height
        }
        ctx.viewport(0, 0, width, height)
        ctx.uniform2f(u.uRes, width, height)
        ctx.uniform1f(u.uT, pose.t)
        ctx.uniform1f(u.uScene, pose.scene)
        ctx.uniform1f(u.uFlash, pose.flash)
        ctx.uniform1f(u.uAmb, pose.amb)
        ctx.uniform1f(u.uLogoA, pose.logoA)
        ctx.uniform1f(u.uLogoBoost, pose.logoBoost)
        ctx.uniform1f(u.uSweep, pose.sweep)
        ctx.uniform1f(u.uComet, pose.comet)
        ctx.uniform1f(u.uRingA, pose.ringA)
        ctx.uniform4f(u.uCam1, ...pose.cam1)
        ctx.uniform4f(u.uCam2, ...pose.cam2)
        ctx.uniform3f(u.uAccent, ...palette.accent)
        ctx.uniform3f(u.uGold, ...palette.gold)
        ctx.uniform4f(u.uLogoRect, 0, LOGO_CENTER_Y, LOGO_HALF, LOGO_HALF)
        ctx.drawArrays(ctx.TRIANGLES, 0, 3)
      },
      dispose: release,
    }
  } catch {
    release()
    return null
  }
}
