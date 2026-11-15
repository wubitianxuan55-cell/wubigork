/**
 * MemoryAtlasStage —— 记忆中枢「活体星图」场景（three.js / WebGL）。
 *
 * 中间不是贴图、不是 2D 剪影，也不是 Canvas 伪 3D：它加载的是 NIH 3D
 * 「Detailed Human Brain Model」的 18,511 顶点真实脑网格（CC BY 4.0），
 * 用 GPU 渲染成「发光脑壳 + 脑回线框 + 顶点粒子」三层，任何角度旋转都是
 * 同一个 3D 脑。左右半球按 x<0 / x>0 分金/紫；只有真实 GraphNode 才是
 * 大光点，点击/悬浮用 three Raycaster，不伪造记忆条目。
 *
 * 性能：26,265 三角面 + 43k 棱 + 18.5k 粒子全部走 GPU；DPR 上限 1.5，
 * 页面不可见时停画。3D 依赖只在打开记忆中枢时懒加载（页面侧 React.lazy），
 * GraphView 的 3d-force-graph 仍是独立 chunk。
 */
import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { DOMAIN_COLORS, DOMAIN_LABELS } from "../../lib/domainColors";
import type { GraphLink, GraphNode } from "../../lib/types";
import { ATLAS_NODE_LIMIT } from "./atlasConfig";
import { decodeBrainMesh } from "./brainMeshDecoder";

interface RingGeometry {
  left: THREE.Vector3[];
  right: THREE.Vector3[];
}

interface VoiceVisuals {
  pointMat: THREE.PointsMaterial;
  wireMat: THREE.LineBasicMaterial;
  fiberMat: THREE.LineBasicMaterial;
  haloGold: THREE.SpriteMaterial;
  haloViolet: THREE.SpriteMaterial;
  beam: THREE.SpriteMaterial;
  pulseGold: THREE.Sprite;
  pulseViolet: THREE.Sprite;
}

interface SceneState {
  renderer: THREE.WebGLRenderer | null;
  scene: THREE.Scene | null;
  camera: THREE.PerspectiveCamera | null;
  brainGroup: THREE.Group | null;
  nodeGroup: THREE.Group | null;
  platformGroup: THREE.Group | null;
  orbitGroup: THREE.Group | null;
  points: THREE.Points | null;
  nodeSprites: THREE.Sprite[];
  ringSprite: THREE.Sprite | null;
  surfaces: RingGeometry | null;
  paused: boolean;
  autoRotate: boolean;
  showRings: boolean;
  showDust: boolean;
  visible: boolean;
  dragging: boolean;
  dragMoved: boolean;
  lastX: number;
  lastY: number;
  pointerX: number;
  pointerY: number;
  selectedId: string | null;
  resetFrames: number;
  voiceVisuals: VoiceVisuals | null;
}

const FALLBACK_TYPE_COLOR = "#94a3b8"; // hex-exempt：未知领域色
const GOLD_HEX = "#ffb454"; // hex-exempt：左半球金
const VIOLET_HEX = "#bc76ff"; // hex-exempt：右半球紫
const WHITE_HEX = "#ffffff"; // hex-exempt：关系线/选中回退

function hashString(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

function isRightBrain(t: string): boolean {
  return t === "profile" || t === "whisper" || t === "digitallife" || t === "database";
}

function makeGlowTexture(size = 64): THREE.CanvasTexture {
  const c = document.createElement("canvas");
  c.width = size;
  c.height = size;
  const g = c.getContext("2d")!;
  const grad = g.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
  grad.addColorStop(0, "rgba(255,255,255,1)");
  grad.addColorStop(0.16, "rgba(255,255,255,0.9)");
  grad.addColorStop(0.4, "rgba(255,255,255,0.35)");
  grad.addColorStop(0.72, "rgba(255,255,255,0.08)");
  grad.addColorStop(1, "rgba(255,255,255,0)");
  g.fillStyle = grad;
  g.fillRect(0, 0, size, size);
  const tex = new THREE.CanvasTexture(c);
  tex.colorSpace = THREE.SRGBColorSpace;
  return tex;
}

function makeRingTexture(size = 96): THREE.CanvasTexture {
  const c = document.createElement("canvas");
  c.width = size;
  c.height = size;
  const g = c.getContext("2d")!;
  g.strokeStyle = "rgba(255,255,255,0.95)";
  g.lineWidth = 3;
  g.beginPath();
  g.arc(size / 2, size / 2, size * 0.38, 0, Math.PI * 2);
  g.stroke();
  g.strokeStyle = "rgba(255,255,255,0.25)";
  g.lineWidth = 1;
  g.beginPath();
  g.arc(size / 2, size / 2, size * 0.46, 0, Math.PI * 2);
  g.stroke();
  const tex = new THREE.CanvasTexture(c);
  tex.colorSpace = THREE.SRGBColorSpace;
  return tex;
}

function makeRadialTexture(rgb: string, alpha = 0.5): THREE.CanvasTexture {
  const size = 256;
  const c = document.createElement("canvas");
  c.width = size;
  c.height = size;
  const g = c.getContext("2d")!;
  const grad = g.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
  grad.addColorStop(0, `rgba(${rgb},${alpha})`);
  grad.addColorStop(0.35, `rgba(${rgb},${alpha * 0.42})`);
  grad.addColorStop(0.7, `rgba(${rgb},${alpha * 0.12})`);
  grad.addColorStop(1, `rgba(${rgb},0)`);
  g.fillStyle = grad;
  g.fillRect(0, 0, size, size);
  const tex = new THREE.CanvasTexture(c);
  tex.colorSpace = THREE.SRGBColorSpace;
  return tex;
}

function makeBeamTexture(): THREE.CanvasTexture {
  const w = 64;
  const h = 256;
  const c = document.createElement("canvas");
  c.width = w;
  c.height = h;
  const g = c.getContext("2d")!;
  const horiz = g.createLinearGradient(0, 0, w, 0);
  horiz.addColorStop(0, "rgba(150,240,255,0)");
  horiz.addColorStop(0.4, "rgba(200,250,255,0.55)");
  horiz.addColorStop(0.5, "rgba(255,255,255,0.95)");
  horiz.addColorStop(0.6, "rgba(200,250,255,0.55)");
  horiz.addColorStop(1, "rgba(150,240,255,0)");
  g.fillStyle = horiz;
  g.fillRect(0, 0, w, h);
  const vert = g.createLinearGradient(0, 0, 0, h);
  vert.addColorStop(0, "rgba(0,0,0,0)");
  vert.addColorStop(0.18, "rgba(0,0,0,1)");
  vert.addColorStop(0.82, "rgba(0,0,0,1)");
  vert.addColorStop(1, "rgba(0,0,0,0)");
  g.globalCompositeOperation = "destination-in";
  g.fillStyle = vert;
  g.fillRect(0, 0, w, h);
  const tex = new THREE.CanvasTexture(c);
  tex.colorSpace = THREE.SRGBColorSpace;
  return tex;
}

let cachedNodeGlow: THREE.CanvasTexture | null = null;
let cachedNodeRing: THREE.CanvasTexture | null = null;
const nodeGlowTexture = () => (cachedNodeGlow ??= makeGlowTexture());
const nodeRingTexture = () => (cachedNodeRing ??= makeRingTexture());

function pickColor(node: GraphNode, right: boolean): THREE.Color {
  const domain = new THREE.Color(DOMAIN_COLORS[node.type] ?? FALLBACK_TYPE_COLOR);
  const hemi = new THREE.Color(right ? VIOLET_HEX : GOLD_HEX);
  return domain.lerp(hemi, 0.32);
}

export interface MemoryAtlasStageProps {
  nodes: GraphNode[];
  links: GraphLink[];
  selectedId?: string | null;
  paused?: boolean;
  autoRotate?: boolean;
  resetSignal?: number;
  showRings?: boolean;
  showDust?: boolean;
  onSelect?: (id: string | null) => void;
  voicePhase?: "idle" | "listening" | "thinking" | "speaking" | "interrupted" | "error";
  voiceConnected?: boolean;
  voiceLevelRef?: { current: number };
  voicePlayLevelRef?: { current: number };
  onReady?: () => void;
}

export function MemoryAtlasStage(p: MemoryAtlasStageProps) {
  const {
    nodes,
    links,
    selectedId = null,
    paused = false,
    autoRotate = true,
    resetSignal = 0,
    showRings = true,
    showDust = true,
    onSelect,
    voicePhase,
    voiceConnected = false,
    voiceLevelRef,
    voicePlayLevelRef,
    onReady,
  } = p;

  const wrapRef = useRef<HTMLDivElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const [hover, setHover] = useState<{ id: string; name: string; type: string } | null>(null);
  const [ready, setReady] = useState(false);
  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;
  const nodesRef = useRef(nodes);
  nodesRef.current = nodes;
  const voiceInputRef = useRef({
    phase: voicePhase,
    connected: voiceConnected,
    levelRef: voiceLevelRef,
    playLevelRef: voicePlayLevelRef,
  });
  voiceInputRef.current = {
    phase: voicePhase,
    connected: voiceConnected,
    levelRef: voiceLevelRef,
    playLevelRef: voicePlayLevelRef,
  };
  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;

  const stateRef = useRef<SceneState>({
    renderer: null,
    scene: null,
    camera: null,
    brainGroup: null,
    nodeGroup: null,
    platformGroup: null,
    orbitGroup: null,
    points: null,
    nodeSprites: [],
    ringSprite: null,
    surfaces: null,
    paused,
    autoRotate,
    showRings,
    showDust,
    visible: true,
    dragging: false,
    dragMoved: false,
    lastX: 0,
    lastY: 0,
    pointerX: 0,
    pointerY: 0,
    selectedId,
    resetFrames: 0,
    voiceVisuals: null,
  });

  // 开关同步
  useEffect(() => {
    const s = stateRef.current;
    s.paused = paused;
    s.autoRotate = autoRotate;
    s.showRings = showRings;
    s.showDust = showDust;
    if (s.orbitGroup) s.orbitGroup.visible = showRings;
    if (s.points) s.points.visible = showDust;
    if (paused) {
      setHover(null);
      s.selectedId = selectedId;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paused, autoRotate, showRings, showDust]);

  useEffect(() => {
    if (resetSignal <= 0) return;
    stateRef.current.resetFrames = 34;
  }, [resetSignal]);

  // 场景构建（只做一次）
  useEffect(() => {
    const wrap = wrapRef.current;
    if (!wrap) return;
    const s = stateRef.current;

    const renderer = (() => {
      try {
        return new THREE.WebGLRenderer({ alpha: true, antialias: true, powerPreference: "high-performance" });
      } catch {
        // jsdom / 无 WebGL 环境：退化为空舞台，页面其余部分照常可用。
        return null;
      }
    })();
    if (!renderer) return;
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 1.5));
    renderer.setClearColor(0x000000, 0);
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    renderer.domElement.className = "atlas-canvas";
    wrap.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(38, 1, 0.1, 30);
    const voiceMode = Boolean(voiceInputRef.current.phase);
    const baseCameraZ = voiceMode ? 2.82 : 3.9;
    camera.position.set(0, 0.28, baseCameraZ);
    camera.lookAt(0, 0.02, 0);
    s.renderer = renderer;
    s.scene = scene;
    s.camera = camera;

    const resize = () => {
      const w = Math.max(1, wrap.clientWidth);
      const h = Math.max(1, wrap.clientHeight);
      renderer.setSize(w, h, false);
      camera.aspect = w / h;
      camera.updateProjectionMatrix();
    };
    resize();
    const ro = typeof ResizeObserver !== "undefined" ? new ResizeObserver(resize) : null;
    ro?.observe(wrap);
    window.addEventListener("resize", resize);
    const io = typeof IntersectionObserver !== "undefined"
      ? new IntersectionObserver((entries) => {
          s.visible = entries.some((e) => e.isIntersecting);
        }, { threshold: 0.01 })
      : null;
    io?.observe(wrap);

    // 数据与几何
    const mesh = decodeBrainMesh();
    const V = mesh.pos.length / 3;
    const geo = new THREE.BufferGeometry();
    geo.setAttribute("position", new THREE.BufferAttribute(mesh.pos, 3));
    geo.setIndex(new THREE.BufferAttribute(mesh.tri, 1));
    geo.computeVertexNormals();
    const vColors = new Float32Array(V * 3);
    const gold = new THREE.Color(GOLD_HEX);
    const violet = new THREE.Color(VIOLET_HEX);
    for (let i = 0; i < V; i++) {
      const c = (mesh.pos[i * 3] < 0 ? gold : violet).clone();
      c.multiplyScalar(0.5 + mesh.shade[i] * 0.72);
      vColors[i * 3] = c.r;
      vColors[i * 3 + 1] = c.g;
      vColors[i * 3 + 2] = c.b;
    }
    geo.setAttribute("color", new THREE.BufferAttribute(vColors, 3));

    const surfaceMat = new THREE.ShaderMaterial({
      vertexColors: true,
      transparent: true,
      depthWrite: false,
      blending: THREE.AdditiveBlending,
      side: THREE.DoubleSide,
      vertexShader: `
        attribute vec3 color;
        varying vec3 vColor;
        varying vec3 vNormal;
        varying vec3 vView;
        void main() {
          vColor = color;
          vNormal = normalize(normalMatrix * normal);
          vec4 mv = modelViewMatrix * vec4(position, 1.0);
          vView = normalize(-mv.xyz);
          gl_Position = projectionMatrix * mv;
        }
      `,
      fragmentShader: `
        varying vec3 vColor;
        varying vec3 vNormal;
        varying vec3 vView;
        void main() {
          float fres = pow(1.0 - abs(dot(normalize(vNormal), normalize(vView))), 2.1);
          float diff = 0.55 + 0.45 * max(dot(normalize(vNormal), normalize(vec3(0.3, 0.5, 0.82))), 0.0);
          vec3 col = vColor * diff * (0.55 + 1.5 * fres);
          gl_FragColor = vec4(col, 0.09 + 0.58 * fres);
        }
      `,
    });
    const surface = new THREE.Mesh(geo, surfaceMat);

    // 唯一棱 → 线框（脑回结构）
    const edgeKeys = new Set<number>();
    const eA: number[] = [];
    const eB: number[] = [];
    for (let t = 0; t < mesh.tri.length; t += 3) {
      const a = mesh.tri[t], b = mesh.tri[t + 1], c = mesh.tri[t + 2];
      for (const [u, v] of [[a, b], [b, c], [c, a]] as const) {
        const key = u < v ? u * V + v : v * V + u;
        if (edgeKeys.has(key)) continue;
        edgeKeys.add(key);
        eA.push(u);
        eB.push(v);
      }
    }
    const wirePos = new Float32Array(eA.length * 6);
    const wireCol = new Float32Array(eA.length * 6);
    for (let i = 0; i < eA.length; i++) {
      const a = eA[i], b = eB[i];
      wirePos[i * 6] = mesh.pos[a * 3];
      wirePos[i * 6 + 1] = mesh.pos[a * 3 + 1];
      wirePos[i * 6 + 2] = mesh.pos[a * 3 + 2];
      wirePos[i * 6 + 3] = mesh.pos[b * 3];
      wirePos[i * 6 + 4] = mesh.pos[b * 3 + 1];
      wirePos[i * 6 + 5] = mesh.pos[b * 3 + 2];
      const c = (mesh.pos[a * 3] + mesh.pos[b * 3]) / 2 < 0 ? gold : violet;
      for (let k = 0; k < 2; k++) {
        wireCol[i * 6 + k * 3] = c.r * 0.55;
        wireCol[i * 6 + k * 3 + 1] = c.g * 0.55;
        wireCol[i * 6 + k * 3 + 2] = c.b * 0.55;
      }
    }
    const wireGeo = new THREE.BufferGeometry();
    wireGeo.setAttribute("position", new THREE.BufferAttribute(wirePos, 3));
    wireGeo.setAttribute("color", new THREE.BufferAttribute(wireCol, 3));
    const wireMat = new THREE.LineBasicMaterial({
      vertexColors: true,
      transparent: true,
      opacity: 0.14,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    });
    const wire = new THREE.LineSegments(wireGeo, wireMat);

    // 脑回纤维（沿网格边的随机游走）
    const fiberSegs: number[] = [];
    const fiberColors: number[] = [];
    for (const path of mesh.fibers) {
      for (let i = 0; i < path.length - 1; i++) {
        const a = path[i], b = path[i + 1];
        fiberSegs.push(mesh.pos[a * 3], mesh.pos[a * 3 + 1], mesh.pos[a * 3 + 2]);
        fiberSegs.push(mesh.pos[b * 3], mesh.pos[b * 3 + 1], mesh.pos[b * 3 + 2]);
        const c = (mesh.pos[a * 3] + mesh.pos[b * 3]) / 2 < 0 ? gold : violet;
        fiberColors.push(c.r * 0.7, c.g * 0.7, c.b * 0.7);
        fiberColors.push(c.r * 0.7, c.g * 0.7, c.b * 0.7);
      }
    }
    const fiberGeo = new THREE.BufferGeometry();
    fiberGeo.setAttribute("position", new THREE.BufferAttribute(new Float32Array(fiberSegs), 3));
    fiberGeo.setAttribute("color", new THREE.BufferAttribute(new Float32Array(fiberColors), 3));
    const fiberMat = new THREE.LineBasicMaterial({
      vertexColors: true,
      transparent: true,
      opacity: 0.1,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
    });
    const fibers = new THREE.LineSegments(fiberGeo, fiberMat);

    // 顶点粒子
    const glowTex = makeGlowTexture();
    const pointMat = new THREE.PointsMaterial({
      size: 0.012,
      map: glowTex,
      vertexColors: true,
      transparent: true,
      opacity: 0.9,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
      sizeAttenuation: true,
      alphaTest: 0.01,
    });
    const points = new THREE.Points(geo, pointMat);
    s.points = points;

    const brainGroup = new THREE.Group();
    if (voiceMode) brainGroup.scale.setScalar(1.14);
    brainGroup.add(surface, wire, fibers, points);
    scene.add(brainGroup);
    s.brainGroup = brainGroup;

    // 节点落点：正面（+z）且明暗足够的顶点
    const left: THREE.Vector3[] = [];
    const right: THREE.Vector3[] = [];
    for (let i = 0; i < V; i++) {
      if (mesh.shade[i] < 0.55 || mesh.pos[i * 3 + 2] < 0.05) continue;
      const v = new THREE.Vector3(mesh.pos[i * 3], mesh.pos[i * 3 + 1], mesh.pos[i * 3 + 2]);
      if (v.x < 0) left.push(v);
      else right.push(v);
    }
    s.surfaces = {
      left: left.length ? left : [...Array(Math.min(300, V))].map((_, i) => new THREE.Vector3(mesh.pos[i * 3], mesh.pos[i * 3 + 1], mesh.pos[i * 3 + 2])),
      right: right.length ? right : [],
    };
    if (s.surfaces.right.length === 0) s.surfaces.right = s.surfaces.left;

    // 背景星点
    const starPos = new Float32Array(360 * 3);
    for (let i = 0; i < 360; i++) {
      starPos[i * 3] = (Math.random() - 0.5) * 9;
      starPos[i * 3 + 1] = (Math.random() - 0.5) * 5.5;
      starPos[i * 3 + 2] = -Math.random() * 5 - 0.5;
    }
    const starGeo = new THREE.BufferGeometry();
    starGeo.setAttribute("position", new THREE.BufferAttribute(starPos, 3));
    const stars = new THREE.Points(starGeo, new THREE.PointsMaterial({
      size: 0.012, map: glowTex, color: 0x9cc8ff, transparent: true, opacity: 0.5,
      blending: THREE.AdditiveBlending, depthWrite: false, sizeAttenuation: true,
    }));
    scene.add(stars);

    // 双色光晕
    const haloGold = new THREE.Sprite(new THREE.SpriteMaterial({
      map: makeRadialTexture("255,180,90", 0.42), transparent: true,
      blending: THREE.AdditiveBlending, depthWrite: false, color: 0xffb45c,
    }));
    haloGold.scale.set(2.4, 2.4, 1);
    haloGold.position.set(-0.45, 0.08, -0.45);
    const haloViolet = new THREE.Sprite(new THREE.SpriteMaterial({
      map: makeRadialTexture("180,110,255", 0.45), transparent: true,
      blending: THREE.AdditiveBlending, depthWrite: false, color: 0xb46bff,
    }));
    haloViolet.scale.set(2.4, 2.4, 1);
    haloViolet.position.set(0.45, 0.08, -0.45);
    scene.add(haloGold, haloViolet);

    const pulseTexture = makeRadialTexture("255,255,255", 0.7);
    const pulseGold = new THREE.Sprite(new THREE.SpriteMaterial({
      map: pulseTexture,
      color: 0xffb454,
      transparent: true,
      opacity: 0,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
      depthTest: false,
    }));
    pulseGold.position.set(-0.46, 0.06, 0.3);
    pulseGold.scale.set(1.1, 1.1, 1);
    pulseGold.renderOrder = 9;
    scene.add(pulseGold);

    const pulseViolet = new THREE.Sprite(new THREE.SpriteMaterial({
      map: pulseTexture,
      color: 0xb86dff,
      transparent: true,
      opacity: 0,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
      depthTest: false,
    }));
    pulseViolet.position.set(0.46, 0.06, 0.3);
    pulseViolet.scale.set(1.1, 1.1, 1);
    pulseViolet.renderOrder = 9;
    scene.add(pulseViolet);

    // 平台
    const platformGroup = new THREE.Group();
    platformGroup.position.y = -1.05;
    const ringDefs = [
      [0.36, 0.55], [0.5, 0.42], [0.66, 0.3], [0.84, 0.2], [1.04, 0.12], [1.2, 0.07],
    ] as const;
    for (const [radius, opacity] of ringDefs) {
      const pts: THREE.Vector3[] = [];
      for (let i = 0; i < 128; i++) {
        const a = (i / 128) * Math.PI * 2;
        pts.push(new THREE.Vector3(Math.cos(a) * radius, 0, Math.sin(a) * radius * 0.3));
      }
      const rg = new THREE.BufferGeometry().setFromPoints(pts);
      const rm = new THREE.LineBasicMaterial({
        color: 0x6ee7ff, transparent: true, opacity,
        blending: THREE.AdditiveBlending, depthWrite: false,
      });
      platformGroup.add(new THREE.LineLoop(rg, rm));
    }
    const disc = new THREE.Mesh(
      new THREE.PlaneGeometry(2.6, 2.6),
      new THREE.MeshBasicMaterial({
        map: makeRadialTexture("110,225,255", 0.4), transparent: true,
        blending: THREE.AdditiveBlending, depthWrite: false, side: THREE.DoubleSide,
      }),
    );
    disc.rotation.x = -Math.PI / 2;
    disc.position.y = 0.002;
    platformGroup.add(disc);
    scene.add(platformGroup);
    platformGroup.visible = !voiceMode;
    s.platformGroup = platformGroup;

    const beam = new THREE.Sprite(new THREE.SpriteMaterial({
      map: makeBeamTexture(), transparent: true, blending: THREE.AdditiveBlending,
      depthWrite: false, color: 0xbdf3ff, opacity: 0.6,
    }));
    beam.scale.set(0.42, 1.75, 1);
    beam.position.set(0, -0.38, -0.1);
    scene.add(beam);

    // 轨道环
    const orbitGroup = new THREE.Group();
    const orbitDefs = [
      [1.24, 0.46, 0.18, 0.2],
      [1.48, -0.4, -0.22, 0.12],
      [1.7, 0.62, 0.1, 0.07],
    ] as const;
    for (const [radius, rx, rz, opacity] of orbitDefs) {
      const pts: THREE.Vector3[] = [];
      for (let i = 0; i < 160; i++) {
        const a = (i / 160) * Math.PI * 2;
        pts.push(new THREE.Vector3(Math.cos(a) * radius, 0, Math.sin(a) * radius));
      }
      const og = new THREE.BufferGeometry().setFromPoints(pts);
      const om = new THREE.LineBasicMaterial({
        color: 0xffcf88, transparent: true, opacity,
        blending: THREE.AdditiveBlending, depthWrite: false,
      });
      const ring = new THREE.LineLoop(og, om);
      ring.rotation.x = rx;
      ring.rotation.z = rz;
      ring.position.y = 0.08;
      orbitGroup.add(ring);
    }
    scene.add(orbitGroup);
    s.orbitGroup = orbitGroup;

    s.voiceVisuals = {
      pointMat,
      wireMat,
      fiberMat,
      haloGold: haloGold.material,
      haloViolet: haloViolet.material,
      beam: beam.material,
      pulseGold,
      pulseViolet,
    };
    const ownedTextures = [
      pulseTexture,
      haloGold.material.map,
      haloViolet.material.map,
      disc.material.map,
      beam.material.map,
    ].filter((texture): texture is THREE.Texture => Boolean(texture));
    setReady(true);
    onReadyRef.current?.();

    // ── 交互 ──────────────────────────────────────────────
    const raycaster = new THREE.Raycaster();
    const pointer = new THREE.Vector2();
    const canvas = renderer.domElement;

    const hitNode = (clientX: number, clientY: number): THREE.Sprite | null => {
      const rect = canvas.getBoundingClientRect();
      pointer.x = ((clientX - rect.left) / rect.width) * 2 - 1;
      pointer.y = -((clientY - rect.top) / rect.height) * 2 + 1;
      raycaster.setFromCamera(pointer, camera);
      const hits = raycaster.intersectObjects(s.nodeSprites, false);
      return (hits[0]?.object as THREE.Sprite) ?? null;
    };

    const onPointerDown = (e: PointerEvent) => {
      if (e.button !== 0) return;
      s.dragging = true;
      s.dragMoved = false;
      s.lastX = e.clientX;
      s.lastY = e.clientY;
      canvas.setPointerCapture?.(e.pointerId);
    };

    const onPointerMove = (e: PointerEvent) => {
      s.pointerX = e.clientX;
      s.pointerY = e.clientY;
      if (s.dragging) {
        const dx = e.clientX - s.lastX;
        const dy = e.clientY - s.lastY;
        if (Math.abs(dx) + Math.abs(dy) > 2) s.dragMoved = true;
        if (s.brainGroup) {
          s.brainGroup.rotation.y += dx * 0.006;
          s.brainGroup.rotation.x = Math.max(-0.55, Math.min(0.55, s.brainGroup.rotation.x + dy * 0.004));
        }
        s.lastX = e.clientX;
        s.lastY = e.clientY;
        return;
      }
      const sprite = hitNode(e.clientX, e.clientY);
      const id = (sprite?.userData.id as string | undefined) ?? null;
      setHover((prev) => {
        if (!sprite) return prev === null ? prev : null;
        if (prev && prev.id === id) return prev;
        return {
          id: id ?? "",
          name: (sprite.userData.name as string) ?? "",
          type: (sprite.userData.type as string) ?? "",
        };
      });
      canvas.style.cursor = sprite ? "pointer" : "grab";
    };

    const onPointerUp = (e: PointerEvent) => {
      if (!s.dragging) return;
      s.dragging = false;
      try { canvas.releasePointerCapture?.(e.pointerId); } catch { /* ignore */ }
      if (!s.dragMoved) {
        const sprite = hitNode(e.clientX, e.clientY);
        onSelectRef.current?.((sprite?.userData.id as string | undefined) ?? null);
      }
    };

    const onPointerLeave = () => {
      if (!s.dragging) setHover(null);
    };

    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      camera.position.z = Math.max(2.65, Math.min(5.6, camera.position.z + e.deltaY * 0.0016));
      camera.lookAt(0, 0.02, 0);
    };

    canvas.addEventListener("pointerdown", onPointerDown);
    canvas.addEventListener("pointermove", onPointerMove);
    canvas.addEventListener("pointerup", onPointerUp);
    canvas.addEventListener("pointercancel", onPointerUp);
    canvas.addEventListener("pointerleave", onPointerLeave);
    canvas.addEventListener("wheel", onWheel, { passive: false });

    // ── 渲染循环 ──────────────────────────────────────────
    let raf = 0;
    let last = performance.now();
    const render = () => {
      raf = requestAnimationFrame(render);
      const now = performance.now();
      const dt = Math.min(50, now - last);
      last = now;
      if (!s.visible) return;

      if (!s.paused) {
        if (s.autoRotate && !s.dragging && s.brainGroup) s.brainGroup.rotation.y += dt * 0.00008;
        if (s.platformGroup) s.platformGroup.rotation.y += dt * 0.00003;
        if (s.orbitGroup) s.orbitGroup.rotation.y += dt * 0.00002;
      }
      if (s.resetFrames > 0 && s.brainGroup) {
        s.brainGroup.rotation.y += (0 - s.brainGroup.rotation.y) * 0.12;
        s.brainGroup.rotation.x += (0 - s.brainGroup.rotation.x) * 0.12;
        camera.position.z += (baseCameraZ - camera.position.z) * 0.12;
        s.resetFrames -= 1;
      }

      if (s.voiceVisuals) {
        const voice = voiceInputRef.current;
        const visuals = s.voiceVisuals;
        if (!voice.phase) {
          visuals.pulseGold.material.opacity = 0;
          visuals.pulseViolet.material.opacity = 0;
        } else {
          const micLevel = Math.max(0, Math.min(1, voice.levelRef?.current ?? 0));
          const playLevel = Math.max(0, Math.min(1, voice.playLevelRef?.current ?? 0));
          const perceptual = (value: number) => Math.pow(
            Math.max(0, Math.min(1, (value - 0.012) / 0.27)),
            0.58,
          );
          const micEnergy = perceptual(micLevel);
          const playEnergy = perceptual(playLevel);
          const live = voice.phase === "speaking" ? Math.max(micEnergy, playEnergy) : micEnergy;
          const connected = voice.connected ? 1 : 0.58;
          const thoughtWave = 0.5 + 0.5 * Math.sin(now * 0.0026);
          const leftEnergy = voice.phase === "listening"
            ? 0.14 + live * 0.86
            : voice.phase === "thinking"
              ? 0.16 + thoughtWave * 0.64
              : voice.phase === "error" || voice.phase === "interrupted"
                ? 0.18
                : 0.045;
          const rightEnergy = voice.phase === "speaking"
            ? 0.14 + live * 0.86
            : voice.phase === "thinking"
              ? 0.16 + (1 - thoughtWave) * 0.64
              : voice.phase === "error" || voice.phase === "interrupted"
                ? 0.18
                : 0.045;
          const peakEnergy = Math.max(leftEnergy, rightEnergy);
          const breathe = 1 + Math.sin(now * 0.0042) * 0.035;

          visuals.pulseGold.material.opacity = connected * leftEnergy * 0.64;
          visuals.pulseViolet.material.opacity = connected * rightEnergy * 0.64;
          visuals.pulseGold.scale.set(
            1.1 * breathe * (1 + leftEnergy * 0.28),
            1.1 * breathe * (1 + leftEnergy * 0.22),
            1,
          );
          visuals.pulseViolet.scale.set(
            1.1 * breathe * (1 + rightEnergy * 0.28),
            1.1 * breathe * (1 + rightEnergy * 0.22),
            1,
          );
          visuals.haloGold.opacity = 0.34 + leftEnergy * 0.28;
          visuals.haloViolet.opacity = 0.36 + rightEnergy * 0.3;
          visuals.pointMat.opacity = Math.min(1, 0.86 + peakEnergy * 0.16);
          visuals.wireMat.opacity = 0.13 + peakEnergy * 0.1;
          visuals.fiberMat.opacity = 0.09 + peakEnergy * 0.13;
          visuals.beam.opacity = 0.42 + peakEnergy * 0.34;
        }
      }
      renderer.render(scene, camera);
    };
    raf = requestAnimationFrame(render);

    return () => {
      cancelAnimationFrame(raf);
      ro?.disconnect();
      io?.disconnect();
      window.removeEventListener("resize", resize);
      canvas.removeEventListener("pointerdown", onPointerDown);
      canvas.removeEventListener("pointermove", onPointerMove);
      canvas.removeEventListener("pointerup", onPointerUp);
      canvas.removeEventListener("pointercancel", onPointerUp);
      canvas.removeEventListener("pointerleave", onPointerLeave);
      canvas.removeEventListener("wheel", onWheel);
      scene.traverse((obj) => {
        const m = obj as THREE.Mesh;
        m.geometry?.dispose?.();
        const mat = m.material as THREE.Material | THREE.Material[] | undefined;
        if (Array.isArray(mat)) mat.forEach((x) => x.dispose());
        else mat?.dispose?.();
      });
      glowTex.dispose();
      for (const texture of ownedTextures) texture.dispose();
      renderer.dispose();
      renderer.forceContextLoss();
      canvas.remove();
      s.renderer = null;
      s.scene = null;
      s.camera = null;
      s.brainGroup = null;
      s.nodeGroup = null;
      s.platformGroup = null;
      s.orbitGroup = null;
      s.points = null;
      s.surfaces = null;
      s.voiceVisuals = null;
    };
  }, []);

  // 真实记忆节点层（随 nodes/links 重建）
  useEffect(() => {
    const s = stateRef.current;
    if (!ready || !s.brainGroup || !s.surfaces) return;
    if (s.nodeGroup) {
      s.brainGroup.remove(s.nodeGroup);
      s.nodeGroup.traverse((obj) => {
        const m = obj as THREE.Mesh;
        m.geometry?.dispose?.();
        const mat = m.material as THREE.Material | THREE.Material[] | undefined;
        if (Array.isArray(mat)) mat.forEach((x) => x.dispose());
        else mat?.dispose?.();
      });
      s.nodeGroup = null;
    }
    const group = new THREE.Group();
    s.nodeGroup = group;
    s.brainGroup.add(group);
    s.nodeSprites = [];

    const glowTex = nodeGlowTexture();
    const ringTex = nodeRingTexture();
    const nodeColors = new Map<string, THREE.Color>();
    const used = nodes.slice(0, ATLAS_NODE_LIMIT);
    for (const node of used) {
      const right = isRightBrain(node.type);
      const pool = right ? s.surfaces.right : s.surfaces.left;
      if (pool.length === 0) continue;
      const base = pool[hashString(node.id || node.name || "") % pool.length];
      const sprite = new THREE.Sprite(new THREE.SpriteMaterial({
        map: glowTex,
        color: pickColor(node, right),
        transparent: true,
        blending: THREE.AdditiveBlending,
        depthWrite: false,
        depthTest: false,
      }));
      sprite.position.copy(base);
      sprite.scale.setScalar(0.11 + Math.max(0, Math.min(1.4, node.val ?? 1)) * 0.025);
      sprite.userData.id = node.id;
      sprite.userData.name = node.name;
      sprite.userData.type = node.type;
      sprite.userData.val = Math.max(0, Math.min(1.4, node.val ?? 1));
      sprite.renderOrder = 10;
      group.add(sprite);
      s.nodeSprites.push(sprite);
      nodeColors.set(node.id, sprite.material.color.clone());
    }

    // 真实关系线
    const positions: number[] = [];
    const colors: number[] = [];
    const byId = new Map(s.nodeSprites.map((sp) => [sp.userData.id as string, sp]));
    for (const link of links) {
      const a = byId.get(String(link.source));
      const b = byId.get(String(link.target));
      if (!a || !b) continue;
      positions.push(a.position.x, a.position.y, a.position.z, b.position.x, b.position.y, b.position.z);
      const ca = nodeColors.get(a.userData.id as string) ?? new THREE.Color(WHITE_HEX);
      const cb = nodeColors.get(b.userData.id as string) ?? new THREE.Color(WHITE_HEX);
      const c = ca.clone().lerp(cb, 0.5);
      colors.push(c.r * 0.7, c.g * 0.7, c.b * 0.7, c.r * 0.7, c.g * 0.7, c.b * 0.7);
    }
    if (positions.length) {
      const lg = new THREE.BufferGeometry();
      lg.setAttribute("position", new THREE.BufferAttribute(new Float32Array(positions), 3));
      lg.setAttribute("color", new THREE.BufferAttribute(new Float32Array(colors), 3));
      const lm = new THREE.LineBasicMaterial({
        vertexColors: true, transparent: true, opacity: 0.36,
        blending: THREE.AdditiveBlending, depthWrite: false,
      });
      group.add(new THREE.LineSegments(lg, lm));
    }

    // 选中环
    const ring = new THREE.Sprite(new THREE.SpriteMaterial({
      map: ringTex,
      color: 0x8df5ff,
      transparent: true,
      blending: THREE.AdditiveBlending,
      depthWrite: false,
      depthTest: false,
    }));
    ring.scale.setScalar(0.34);
    ring.visible = false;
    ring.renderOrder = 11;
    group.add(ring);
    s.ringSprite = ring;

    // 应用当前选中
    const sp = s.nodeSprites.find((x) => x.userData.id === s.selectedId);
    if (sp) {
      sp.scale.multiplyScalar(1.7);
      ring.position.copy(sp.position);
      ring.visible = true;
    }
  }, [ready, nodes, links]);

  // 选中同步
  useEffect(() => {
    const s = stateRef.current;
    s.selectedId = selectedId;
    for (const sp of s.nodeSprites) {
      const base = 0.11 + Math.max(0, Math.min(1.4, (sp.userData.val as number) ?? 1)) * 0.025;
      sp.scale.setScalar(sp.userData.id === selectedId ? base * 1.8 : base);
    }
    if (s.ringSprite) {
      const sp = s.nodeSprites.find((x) => x.userData.id === selectedId);
      s.ringSprite.visible = !!sp;
      if (sp) s.ringSprite.position.copy(sp.position);
    }
  }, [selectedId, ready, nodes]);

  // 悬浮提示跟随
  useEffect(() => {
    if (!hover) return;
    const el = tooltipRef.current;
    if (!el) return;
    const s = stateRef.current;
    el.style.transform = `translate3d(${s.pointerX + 16}px, ${s.pointerY + 16}px, 0)`;
    const move = (e: PointerEvent) => {
      el.style.transform = `translate3d(${e.clientX + 16}px, ${e.clientY + 16}px, 0)`;
    };
    window.addEventListener("pointermove", move);
    return () => window.removeEventListener("pointermove", move);
  }, [hover]);

  return (
    <div className="atlas-stage" ref={wrapRef} aria-hidden="true">
      {hover && (
        <div className="atlas-tip" ref={tooltipRef} role="presentation">
          <span className="atlas-tip-type" style={{ color: DOMAIN_COLORS[hover.type] ?? FALLBACK_TYPE_COLOR }}>
            {DOMAIN_LABELS[hover.type] ?? hover.type}
          </span>
          <strong>{hover.name}</strong>
          <small>点击查看详情</small>
        </div>
      )}
    </div>
  );
}
