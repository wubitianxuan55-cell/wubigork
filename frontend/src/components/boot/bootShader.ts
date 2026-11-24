/**
 * BootSplash shaders (WebGL2 / GLSL ES 3.00). Every frame is a pure function of the uniforms,
 * so the sequence can be scrubbed, frozen for reduced motion and screenshot-tested.
 *
 * Frame space: the origin is the frame centre, the frame height is 1, y points up. The plates
 * cover the frame (16:9), so on wider windows they are scaled up by `cover` and everything
 * anchored to them lives in plate space `f = p / cover`.
 */

export const VERT = `#version 300 es
in vec2 aPos;
void main(){ gl_Position = vec4(aPos, 0.0, 1.0); }`

export const FRAG = `#version 300 es
precision highp float;
out vec4 O;

uniform vec2 uRes;
uniform float uT;
uniform float uScene;
uniform float uFlash;
uniform float uAmb;
uniform float uLogoA;
uniform float uLogoBoost;
uniform float uSweep;
uniform float uComet;
uniform float uRingA;
uniform vec4 uCam1;
uniform vec4 uCam2;
uniform vec3 uAccent;
uniform vec3 uGold;
uniform sampler2D uLogo;
uniform sampler2D uPlate1;
uniform sampler2D uPlate2;
uniform vec4 uLogoRect;

const float AR = 1.7777778;

float hash11(float p){ p = fract(p * 0.1031); p *= p + 33.33; p *= p + p; return fract(p); }
float hash12(vec2 p){ vec3 p3 = fract(vec3(p.xyx) * 0.1031); p3 += dot(p3, p3.yzx + 33.33); return fract((p3.x + p3.y) * p3.z); }
vec2 hash22(vec2 p){ vec3 p3 = fract(vec3(p.xyx) * vec3(0.1031, 0.1030, 0.0973)); p3 += dot(p3, p3.yzx + 33.33); return fract((p3.xx + p3.yz) * p3.zy); }
float noise2(vec2 p){
  vec2 i = floor(p); vec2 f = fract(p); vec2 u = f * f * (3.0 - 2.0 * f);
  return mix(mix(hash12(i), hash12(i + vec2(1.0, 0.0)), u.x), mix(hash12(i + vec2(0.0, 1.0)), hash12(i + vec2(1.0, 1.0)), u.x), u.y);
}
float fbm3(vec2 p){ float a = 0.5; float s = 0.0; for (int i = 0; i < 3; i++){ s += a * noise2(p); p = p * 2.03 + vec2(1.7, 9.2); a *= 0.5; } return s; }

/* Pin-point twinkles, three depths. Sizes are in device pixels so they survive any resolution. */
vec3 stars(vec2 p, float t, float density){
  float px = uRes.y;
  float sig = 0.72 + px / 2600.0;
  vec3 acc = vec3(0.0);
  for (int i = 0; i < 3; i++){
    float fi = float(i);
    float sc = 24.0 + fi * 13.0;
    vec2 g = p * sc;
    vec2 id = floor(g);
    vec2 f = fract(g) - 0.5;
    vec2 h = hash22(id + fi * 17.3);
    vec2 dpx = (f - (h - 0.5) * 0.9) * (px / sc);
    float r2 = dot(dpx, dpx);
    float m = hash12(id + 3.1 + fi * 5.7);
    float keep = step(m, density * (0.60 - fi * 0.14));
    float bri = pow(hash12(id + 8.8 + fi), 3.5) * (1.7 - fi * 0.5);
    float tw = 0.45 + 0.55 * (0.5 + 0.5 * sin(t * (1.5 + 5.0 * m) + m * 61.0));
    float core = exp(-r2 / (sig * sig));
    float halo = exp(-sqrt(r2) * 1.1) * 0.085;
    vec3 tint = mix(vec3(0.74, 0.84, 1.0), vec3(1.0, 0.92, 0.80), hash12(id + 9.0));
    acc += keep * bri * (1.0 - uAmb + uAmb * tw) * (core + halo) * tint;
  }
  return acc;
}

/* Frame point -> plate UV. The zoom is anchored; the far top scales a little less than the near bottom. */
vec2 plateUv(vec2 p, vec4 cam, vec2 anchor){
  float sy = clamp(0.5 - p.y, 0.0, 1.0);
  float s = cam.x * (1.0 + cam.w * (sy - 0.45));
  vec2 q = anchor + (p - vec2(cam.y, cam.z) - anchor) / s;
  return vec2(0.5 + q.x / AR, 0.5 - q.y);
}

/* Orbit plate: breathing sunrise, twinkles in the dark sky, shimmering limb. */
vec3 shotOrbit(vec2 p, float t){
  vec2 sun = vec2(0.0, -0.035);
  vec3 col = texture(uPlate1, plateUv(p, uCam1, sun)).rgb;
  float alt = length(p - vec2(0.0, -3.03)) - 2.99;
  float luma = dot(col, vec3(0.2126, 0.7152, 0.0722));
  float dark = smoothstep(0.10, 0.02, luma) * smoothstep(0.02, 0.10, alt);
  col += stars(p, t, 0.55) * dark * 0.55 * uAmb;
  float pulse = 1.0 + uAmb * 0.10 * sin(t * 2.0);
  float r = length((p - sun) * vec2(1.0, 1.9));
  col += vec3(1.0, 0.86, 0.66) * exp(-r * 22.0) * 0.50 * pulse;
  col += vec3(1.0, 0.70, 0.42) * exp(-r * 5.0) * 0.14 * pulse;
  col += vec3(0.80, 0.90, 1.0) * exp(-abs(p.y - sun.y) * 200.0) * exp(-abs(p.x) * 6.0) * 0.22 * smoothstep(-0.004, 0.004, alt);
  col += vec3(0.30, 0.70, 1.0) * exp(-abs(alt) * 110.0) * (0.14 + uAmb * 0.05 * sin(t * 1.4 + p.x * 3.0));
  return col;
}

/* Valley plate: rippling mirror, city lights, beam, wisps, mist, glitter. */
vec3 shotValley(vec2 p, float t){
  vec2 sun = vec2(0.064, -0.132);
  float sy = 0.5 - p.y;
  vec2 uv = plateUv(p, uCam2, sun);
  float lake = smoothstep(0.655, 0.680, sy) * smoothstep(0.990, 0.950, sy)
             * smoothstep(0.14, 0.21, uv.x) * smoothstep(0.90, 0.83, uv.x);
  float rip = noise2(vec2(uv.x * 22.0 + t * 0.05, uv.y * 260.0 - t * 1.6)) - 0.5;
  float rip2 = noise2(vec2(uv.x * 70.0 - t * 0.03, uv.y * 900.0 + t * 4.0)) - 0.5;
  uv.x += (rip * 0.0035 + rip2 * 0.0012) * lake * uAmb;
  uv.y += rip2 * 0.0008 * lake * uAmb;
  vec3 col = texture(uPlate2, uv).rgb;

  vec2 cp = p - vec2(0.565, -0.055);
  float box = smoothstep(0.135, 0.100, abs(cp.x)) * smoothstep(0.105, 0.075, abs(cp.y));
  col += stars(p + vec2(3.7, 1.3), t, 1.0) * box * vec3(1.0, 0.88, 0.70) * 0.8 * uAmb;
  col += vec3(0.70, 0.92, 1.0) * exp(-abs(p.x - 0.578) * 520.0) * smoothstep(-0.13, -0.05, p.y) * smoothstep(0.30, 0.10, p.y) * (0.10 + uAmb * 0.04 * sin(t * 1.7));

  vec2 pp = p - vec2(0.672, 0.290);
  float inside = smoothstep(0.155, 0.135, length(pp));
  col += vec3(0.85, 0.92, 1.0) * smoothstep(0.55, 0.80, fbm3(vec2(p.x * 2.2 - t * 0.035, p.y * 38.0) + 7.0)) * inside * 0.10 * uAmb;

  float luma = dot(col, vec3(0.2126, 0.7152, 0.0722));
  col += stars(p + vec2(0.0, 0.4), t, 0.5) * smoothstep(0.16, 0.05, luma) * smoothstep(0.22, 0.08, sy) * 0.5 * uAmb;

  float pulse = 1.0 + uAmb * 0.10 * sin(t * 2.0);
  float r = length((p - sun) * vec2(1.0, 2.2));
  col += vec3(1.0, 0.84, 0.62) * exp(-r * 26.0) * 0.40 * pulse + vec3(1.0, 0.66, 0.40) * exp(-r * 6.0) * 0.12 * pulse;
  float glit = pow(noise2(vec2(uv.x * 70.0 + t * 0.4, uv.y * 420.0 - t * 5.0)), 16.0);
  col += vec3(1.0, 0.84, 0.58) * glit * exp(-abs(uv.x - 0.536) * 16.0) * 5.0 * lake * uAmb;

  float mist = fbm3(vec2(p.x * 3.0 - t * 0.018, p.y * 26.0) + 5.0);
  col = mix(col, vec3(0.86, 0.90, 0.97), smoothstep(0.55, 0.85, mist) * exp(-pow((sy - 0.675) * 26.0, 2.0)) * 0.26 * uAmb);
  return col;
}

/* Glass mark (premultiplied texture) + orbital comet, in frame space. */
vec3 foreground(vec3 col, vec2 p){
  float px = uRes.y;
  vec2 q = p - uLogoRect.xy;
  vec2 lp = q / uLogoRect.zw;
  vec2 uv = lp * 0.5 + 0.5;
  float inRect = step(max(abs(lp.x), abs(lp.y)), 1.0);
  float mask = 0.0;
  vec3 lcol = vec3(0.0);
  if (inRect > 0.5){
    vec4 L = texture(uLogo, uv);
    mask = L.a;
    lcol = L.rgb;
  }
  float a = mix(0.500, 0.335, uScene);
  float ratio = mix(0.28, 0.30, uScene) + 0.04 * sin(uT * 0.9) * uAmb;
  float rot = mix(-0.34, -0.22, uScene) + 0.03 * sin(uT * 0.6) * uAmb;
  float cr = cos(rot);
  float sr = sin(rot);
  vec2 r = vec2(q.x * cr + q.y * sr, -q.x * sr + q.y * cr);
  vec2 e = r / vec2(a, a * ratio);
  float le = length(e);
  vec2 grad = vec2(e.x / a, e.y / (a * ratio)) / max(le, 1e-5);
  float dist = (le - 1.0) / max(length(grad), 1e-5);
  float th = atan(e.y, e.x);
  float depth = -sin(th) * sqrt(1.0 - ratio * ratio);
  float ang = fract((uComet - th) / 6.2831853);
  float w = 1.0 + px / 1500.0;
  float line = exp(-pow(dist * px / w, 2.0));
  float glowL = exp(-abs(dist) * px / 7.0) * 0.10;
  float tail = exp(-ang * mix(7.0, 1.9, uScene));
  vec3 ring = mix(vec3(0.60, 0.96, 1.0), uGold, uScene) * (line * (0.20 + 1.1 * tail) + glowL * (0.5 + tail));
  vec2 hr = vec2(cos(uComet), sin(uComet)) * vec2(a, a * ratio);
  vec2 hq = vec2(hr.x * cr - hr.y * sr, hr.x * sr + hr.y * cr);
  float hd = length(q - hq) * px;
  ring += mix(vec3(0.88, 1.0, 1.0), vec3(1.0, 0.96, 0.84), uScene) * (exp(-hd * hd / 6.0) * 1.6 + exp(-hd / 16.0) * 0.40);
  ring *= mix(1.0 - mask * uLogoA, 1.0, step(0.0, depth));
  col += ring * uRingA;

  float la = mask * uLogoA;
  col = col * (1.0 - la) + lcol * (uLogoBoost * uLogoA);
  if (length(lp) < 1.9){
    float bl = 0.0;
    for (int k = 0; k < 8; k++){
      float an = 6.2831853 * float(k) / 8.0;
      bl += texture(uLogo, clamp(uv + vec2(cos(an), sin(an)) * 0.085, 0.001, 0.999)).a;
    }
    bl += 2.0 * texture(uLogo, clamp(uv + vec2(0.05, 0.03), 0.001, 0.999)).a;
    col += mix(uAccent, vec3(1.0), 0.35) * (bl / 10.0) * (1.0 - la) * uLogoA * 0.38;
  }
  float sw = exp(-pow((lp.x * 0.7 + lp.y * 0.7 - uSweep) * 5.0, 2.0));
  col += lcol * sw * uLogoA * 0.55;
  return col;
}

void main(){
  vec2 p = (gl_FragCoord.xy - 0.5 * uRes) / uRes.y;
  vec2 f = p / max(1.0, (uRes.x / uRes.y) / AR);
  vec3 c1 = uScene < 1.0 ? shotOrbit(f, uT) : vec3(0.0);
  vec3 c2 = uScene > 0.0 ? shotValley(f, uT) : vec3(0.0);
  vec3 col = mix(c1, c2, uScene);
  col = foreground(col, p);
  col = mix(col, vec3(1.0, 0.95, 0.88), uFlash);
  vec2 v = (gl_FragCoord.xy / uRes - 0.5) * vec2(1.0, 1.25);
  col *= 1.0 - 0.28 * dot(v, v);
  O = vec4(min(col, vec3(1.0)), 1.0);
}`
