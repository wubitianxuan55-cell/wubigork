import React from 'react'

function lensChannelMap(channel: 'r' | 'g'): string {
  const vertical = channel === 'g'
  const color = (value: number) => vertical ? `rgb(0,${value},0)` : `rgb(${value},0,0)`
  const stops: [number, number][] = [
    [0, 232],
    [0.035, 160],
    [0.08, 128],
    [0.92, 128],
    [0.965, 96],
    [1, 24],
  ]
  const svg = [
    '<svg xmlns="http://www.w3.org/2000/svg" width="128" height="128"><defs>',
    `<linearGradient id="r" x1="0" y1="0" x2="${vertical ? 0 : 1}" y2="${vertical ? 1 : 0}">`,
    stops.map(([offset, value]) => `<stop offset="${offset}" stop-color="${color(value)}"/>`).join(''),
    '</linearGradient></defs><rect width="128" height="128" fill="url(#r)"/></svg>',
  ].join('')
  return `data:image/svg+xml,${encodeURIComponent(svg)}`
}

const LENS_RED_MAP = lensChannelMap('r')
const LENS_GREEN_MAP = lensChannelMap('g')

const FOG_FIELDS = {
  stream: [[0.12, 0.18, 0.27, 0.19, 1], [0.87, 0.82, 0.27, 0.19, 0.85]],
  wash: [
    [0.24, 0.10, 0.24, 0.15, 0.9],
    [0.78, 0.13, 0.22, 0.16, 0.85],
    [0.20, 0.88, 0.22, 0.15, 0.8],
    [0.78, 0.89, 0.23, 0.15, 0.85],
  ],
  clear: [[-0.02, 0.34, 0.21, 0.30, 0.55], [1.02, 0.66, 0.21, 0.30, 0.55]],
} as const

/**
 * SVG-only parts of Bloom 0.29: the liquid lens and the three light fog masks.
 * Keeping these as real SVG filters/masks preserves the upstream edge
 * refraction and irregular fog contours instead of approximating them with blur.
 */
const BloomSvgDefs: React.FC = () => (
  <svg
    width="0"
    height="0"
    aria-hidden="true"
    focusable="false"
    style={{ position: 'fixed', pointerEvents: 'none' }}
  >
    <defs>
      <filter
        id="gaea-liquid-lens"
        x="0%"
        y="0%"
        width="100%"
        height="100%"
        colorInterpolationFilters="sRGB"
        primitiveUnits="objectBoundingBox"
      >
        <feImage
          href={LENS_RED_MAP}
          xlinkHref={LENS_RED_MAP}
          width="1"
          height="1"
          preserveAspectRatio="none"
          result="r"
        />
        <feImage
          href={LENS_GREEN_MAP}
          xlinkHref={LENS_GREEN_MAP}
          width="1"
          height="1"
          preserveAspectRatio="none"
          result="g"
        />
        <feComposite in="r" in2="g" operator="arithmetic" k2="1" k3="1" result="lens" />
        <feComponentTransfer in="lens" result="neutral">
          <feFuncR type="linear" slope="1" intercept={String(0.5 - 128 / 255)} />
          <feFuncG type="linear" slope="1" intercept={String(0.5 - 128 / 255)} />
        </feComponentTransfer>
        <feDisplacementMap
          in="SourceGraphic"
          in2="neutral"
          scale="0.016"
          xChannelSelector="R"
          yChannelSelector="G"
        />
      </filter>

      <radialGradient id="gaea-fog-soft">
        {[
          ['0%', '.50'],
          ['25%', '.40'],
          ['65%', '.12'],
          ['100%', '0'],
        ].map(([offset, opacity]) => (
          <stop key={offset} offset={offset} stopColor="white" stopOpacity={opacity} />
        ))}
      </radialGradient>

      <filter
        id="gaea-fog-noise"
        filterUnits="userSpaceOnUse"
        primitiveUnits="userSpaceOnUse"
        x="-.15"
        y="-.15"
        width="1.3"
        height="1.3"
        colorInterpolationFilters="sRGB"
      >
        <feTurbulence type="fractalNoise" baseFrequency="8 10" numOctaves="2" seed="13" result="texture" />
        <feDisplacementMap
          in="SourceGraphic"
          in2="texture"
          scale=".09"
          xChannelSelector="R"
          yChannelSelector="G"
          result="broken"
        />
        <feColorMatrix
          in="texture"
          type="matrix"
          values="0 0 0 0 1  0 0 0 0 1  0 0 0 0 1  1.4 0 0 0 -.25"
          result="grain"
        />
        <feComposite in="broken" in2="grain" operator="in" />
      </filter>

      {(Object.entries(FOG_FIELDS) as [keyof typeof FOG_FIELDS, typeof FOG_FIELDS[keyof typeof FOG_FIELDS]][]).map(([name, fields]) => (
        <mask
          key={name}
          id={`gaea-fog-${name}`}
          maskUnits="objectBoundingBox"
          maskContentUnits="objectBoundingBox"
          x="0"
          y="0"
          width="1"
          height="1"
          style={{ maskType: 'luminance' }}
        >
          <g filter="url(#gaea-fog-noise)">
            {fields.map(([cx, cy, rx, ry, opacity], index) => (
              <ellipse
                key={index}
                cx={cx}
                cy={cy}
                rx={rx}
                ry={ry}
                opacity={opacity}
                fill="url(#gaea-fog-soft)"
              />
            ))}
          </g>
        </mask>
      ))}
    </defs>
  </svg>
)

export default BloomSvgDefs
