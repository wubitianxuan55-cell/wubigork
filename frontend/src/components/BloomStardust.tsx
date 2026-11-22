import React, { useMemo } from 'react'

type CssVars = React.CSSProperties & Record<`--${string}`, string | number>

interface StardustNode {
  className: string
  style: CssVars
  tag?: 'b' | 'i'
}

function random(seed: number): () => number {
  let state = seed >>> 0
  return () => {
    state = (state * 1664525 + 1013904223) >>> 0
    return state / 4294967296
  }
}

function makeDot(
  next: () => number,
  cluster: string,
  size: number,
  dx: number,
  dy: number,
  durationBase: number,
  durationSpan: number,
  twinkleBase: number,
  twinkleSpan: number,
  tintRatio: number,
): StardustNode {
  const peak = 0.35 + next() * 0.65
  const duration = durationBase + next() * durationSpan
  const twinkle = twinkleBase + next() * twinkleSpan
  const tint = next() < tintRatio
  return {
    className: tint ? `${cluster} t` : cluster,
    style: {
      width: `${size.toFixed(2)}px`,
      height: `${size.toFixed(2)}px`,
      '--dx': `${dx.toFixed(1)}px`,
      '--dy': `${dy.toFixed(1)}px`,
      '--peak': peak.toFixed(2),
      '--glow': `${(size * 2.4).toFixed(1)}px`,
      animationDuration: `${duration.toFixed(2)}s, ${twinkle.toFixed(2)}s`,
      animationDelay: `${(-next() * duration).toFixed(2)}s, ${(-next() * twinkle).toFixed(2)}s`,
    },
  }
}

function createParticle(
  next: () => number,
  x: number,
  y: number,
  angle: number,
  distance: number,
  cluster: string,
): StardustNode {
  const galaxyOnly = cluster === 's-amb' || cluster === 's-galaxy-edge'
  const big = next() < (galaxyOnly ? 0.32 : 0.15)
  const size = big ? 1.8 + next() * 0.9 : 0.7 + next() * 1
  const dot = makeDot(
    next,
    cluster,
    size,
    Math.cos(angle) * distance,
    Math.sin(angle) * distance * 0.75,
    6.5,
    7.5,
    1.2,
    1.8,
    0.55,
  )
  if (galaxyOnly) {
    dot.style['--peak'] = Math.max(0.55, Number(dot.style['--peak'])).toFixed(2)
    dot.style['--glow'] = `${Math.max(3, size * 2.4).toFixed(1)}px`
  }
  dot.style.left = `${x.toFixed(2)}%`
  dot.style.top = `${y.toFixed(2)}%`
  return dot
}

function makeMeteor(next: () => number, index: number): StardustNode {
  const fromTop = next() < 0.62
  const left = fromTop ? 8 + next() * 100 : 70 + next() * 34
  const top = fromTop ? -6 + next() * 10 : next() * 62
  const length = 140 + next() * 200
  const duration = 4.5 + next() * 7
  return {
    className: `s-met m${(index % 3) + 1}`,
    tag: 'b',
    style: {
      left: `${left.toFixed(2)}%`,
      top: `${top.toFixed(2)}%`,
      width: `${length.toFixed(0)}px`,
      height: `${(1.8 + next() * 1.4).toFixed(2)}px`,
      '--rot': `${(-28 - next() * 16).toFixed(1)}deg`,
      '--travel': `${(920 + next() * 980).toFixed(0)}px`,
      '--peak': (0.75 + next() * 0.25).toFixed(2),
      animationDuration: `calc(${duration.toFixed(2)}s * var(--gaea-meteor-pace, 1))`,
      animationDelay: `calc(${(-next() * duration).toFixed(2)}s * var(--gaea-meteor-pace, 1))`,
    },
  }
}

function buildStardust(): StardustNode[] {
  const nodes: StardustNode[] = []
  const corner = random(7 * 9973 + 17)
  for (let index = 0; index < 64; index++) {
    const x = 88 + corner() * 12
    const y = -3 + corner() * 10
    const angle = ((95 + corner() * 90) * Math.PI) / 180
    const distanceSeed = corner()
    const distance = 50 + distanceSeed * distanceSeed * 560
    const big = corner() < 0.15
    const size = big ? 1.8 + corner() * 0.9 : 0.7 + corner() * 1
    const dot = makeDot(
      corner,
      's-tr',
      size,
      Math.cos(angle) * distance,
      Math.sin(angle) * distance * 0.75,
      6,
      7,
      1.1,
      1.6,
      0.55,
    )
    dot.style.right = `${(360 - (x / 100) * 360 - size).toFixed(2)}px`
    dot.style.top = `${((y / 100) * 220).toFixed(2)}px`
    nodes.push(dot)
  }

  const ambient = random(7 * 7919 + 3)
  for (let index = 0; index < 26; index++) {
    const x = 2 + ambient() * 24
    const y = 74 + ambient() * 24
    const angle = ((275 + ambient() * 85) * Math.PI) / 180
    const distance = 30 + Math.pow(ambient(), 1.6) * 460
    nodes.push(createParticle(ambient, x, y, angle, distance, 's-bl'))
  }
  for (let index = 0; index < 28; index++) {
    nodes.push(createParticle(
      ambient,
      ambient() * 100,
      ambient() * 100,
      ambient() * Math.PI * 2,
      25 + ambient() * 160,
      's-amb',
    ))
  }
  for (let index = 0; index < 36; index++) {
    const x = 68 + ambient() * 31
    const y = 58 + ambient() * 40
    const angle = ((150 + ambient() * 120) * Math.PI) / 180
    nodes.push(createParticle(
      ambient,
      x,
      y,
      angle,
      35 + Math.pow(ambient(), 1.6) * 420,
      's-amb',
    ))
  }
  for (let index = 0; index < 44; index++) {
    const x = 4 + ambient() * 92
    const y = 76 + ambient() * 23
    const angle = ((195 + ambient() * 150) * Math.PI) / 180
    nodes.push(createParticle(
      ambient,
      x,
      y,
      angle,
      25 + Math.pow(ambient(), 1.6) * 320,
      's-amb',
    ))
  }

  const edge = random(7 * 104729 + 11)
  for (let index = 0; index < 72; index++) {
    const side = index % 4
    const along = 2 + edge() * 96
    const inset = 1 + edge() * 16
    const x = side <= 1 ? along : side === 2 ? inset : 100 - inset
    const y = side >= 2 ? along : side === 0 ? inset : 100 - inset
    nodes.push(createParticle(
      edge,
      x,
      y,
      edge() * Math.PI * 2,
      20 + edge() * 150,
      's-galaxy-edge',
    ))
  }

  const meteor = random(7 * 6151 + 29)
  for (let index = 0; index < 60; index++) nodes.push(makeMeteor(meteor, index))
  return nodes
}

/** Exact deterministic particle set from Bloom 0.29 starlight.ts. */
const BloomStardust: React.FC = () => {
  const nodes = useMemo(buildStardust, [])
  return (
    <div className="gaea-stardust" aria-hidden="true">
      {nodes.map((node, index) => (
        node.tag === 'b'
          ? <b key={index} className={node.className} style={node.style} />
          : <i key={index} className={node.className} style={node.style} />
      ))}
    </div>
  )
}

export default BloomStardust
