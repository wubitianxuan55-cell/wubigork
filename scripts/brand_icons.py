#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""brand_icons.py — 从透明底的标母版生成全部品牌图标与 UI 标。

母版：build/brand/gaea-mark-2048.png（玻璃 G，RGBA）。换标只换母版再跑一次本脚本。

用法（仓根）：python scripts/brand_icons.py        依赖 Pillow + numpy

产出：
  build/appicon.png                 1024 圆角深色砖（exe 图标源）
  build/appicon.svg                 同图的 1024 SVG 封装
  build/appicon-small.svg           512 紧凑小图标（字形占 92%）
  build/windows/icon.ico            七档 16/24/32/48/64/128/256；≤48 用紧凑取景
  frontend/public/favicon.svg       默认 = 深色
  frontend/public/favicon-dark.svg  512 深色砖
  frontend/public/favicon-light.svg 512 浅色砖
  frontend/public/gaea-brand-icon.png = appicon.png
  frontend/public/gaea-brand-mark.webp
                                    UI 品牌标：1568×1264 画幅，字形落在旧明暗原图的同一框位，
                                    所以现有五处容器尺寸都不用动
  frontend/public/boot/mark.webp    开机动画贴图：1024 方形，字形占 86%
"""
import base64
import io
import os
import struct

import numpy as np
from PIL import Image, ImageDraw, ImageFilter

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MASTER = os.path.join(ROOT, 'build', 'brand', 'gaea-mark-2048.png')
ACCENT = (46, 214, 232)
RADIUS = 0.19
MARGIN = 1.16
# 旧明暗原图里字形的框位（1568×1264 画布）：x 406–1193、y 205–970
BRAND_CANVAS = (1568, 1264)
BRAND_BOX = (406, 205, 1193, 970)


def load_letterform():
    """母版 → 方形 RGBA：字形居中、四周留 16% 边。返回 (图, 母版里的字形 bbox)。"""
    mark = Image.open(MASTER).convert('RGBA')
    bbox = mark.split()[3].point(lambda v: 255 if v > 24 else 0).getbbox()
    side = int(round(max(bbox[2] - bbox[0], bbox[3] - bbox[1]) * MARGIN))
    cx, cy = (bbox[0] + bbox[2]) / 2, (bbox[1] + bbox[3]) / 2
    crop = Image.new('RGBA', (side, side), (0, 0, 0, 0))
    crop.paste(mark, (int(round(side / 2 - cx)), int(round(side / 2 - cy))))
    return crop, bbox, mark


def make_tile(size, letter, fill, light=False, corners=True):
    """圆角砖：径向渐变底 + 字形居中（字形高度 = size × fill）。"""
    yy, xx = np.mgrid[0:size, 0:size].astype(np.float32) / (size - 1)
    dist = np.clip(np.sqrt((xx - 0.5) ** 2 + ((yy - 0.42) * 0.95) ** 2) / 0.72, 0, 1)
    inner, outer = ((254, 254, 252), (228, 233, 240)) if light else ((16, 22, 38), (2, 3, 7))
    grad = np.array(inner, np.float32) + (np.array(outer, np.float32) - np.array(inner, np.float32)) * (dist[..., None] ** 1.2)
    tile = Image.fromarray(grad.round().astype('uint8'), 'RGB').convert('RGBA')
    target = int(size * fill * MARGIN)
    origin = ((size - target) // 2, (size - target) // 2 + int(size * 0.01))
    if light:
        shadow = Image.new('RGBA', (size, size), (0, 0, 0, 0))
        alpha = letter.split()[3].point(lambda v: int(v * 0.26))
        shadow.paste(Image.new('RGBA', letter.size, (24, 44, 84, 255)), (0, 0), alpha)
        shadow = shadow.resize((target, target), Image.LANCZOS).filter(ImageFilter.GaussianBlur(size * 0.022))
        tile.alpha_composite(shadow, (origin[0], origin[1] + int(size * 0.014)))
    else:
        glow = Image.new('RGBA', (size, size), (0, 0, 0, 0))
        pad = size * 0.16
        ImageDraw.Draw(glow).ellipse((pad, pad, size - pad, size - pad), fill=ACCENT + (64,))
        tile.alpha_composite(glow.filter(ImageFilter.GaussianBlur(size * 0.1)))
    tile.alpha_composite(letter.resize((target, target), Image.LANCZOS), origin)
    if corners:
        mask = Image.new('L', (size * 4, size * 4), 0)
        ImageDraw.Draw(mask).rounded_rectangle((0, 0, size * 4 - 1, size * 4 - 1), radius=int(size * 4 * RADIUS), fill=255)
        tile.putalpha(mask.resize((size, size), Image.LANCZOS))
    return tile


def svg_wrap(img, title):
    buf = io.BytesIO()
    img.save(buf, 'PNG', optimize=True)
    size = img.size[0]
    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" viewBox="0 0 {size} {size}" role="img" aria-label="gaea">\n'
        f'  <title>{title}</title>\n'
        f'  <image width="{size}" height="{size}" href="data:image/png;base64,{base64.b64encode(buf.getvalue()).decode("ascii")}"/>\n'
        f'</svg>\n'
    )


def png_bytes(img):
    buf = io.BytesIO()
    img.save(buf, 'PNG', optimize=True)
    return buf.getvalue()


def bmp_entry(img):
    """ICO 里的 32 位 BMP 帧：XOR 图像（BGRA、自下而上）+ AND 掩码（alpha 为 0 处置 1）。"""
    w, h = img.size
    a = np.asarray(img)[::-1]
    xor = np.dstack([a[..., 2], a[..., 1], a[..., 0], a[..., 3]]).tobytes()
    row_bytes = ((w + 31) // 32) * 4
    and_mask = b''.join(
        np.packbits((row == 0).astype(np.uint8), bitorder='big').tobytes().ljust(row_bytes, b'\x00')
        for row in a[..., 3]
    )
    header = struct.pack('<IiiHHIIiiII', 40, w, h * 2, 1, 32, 0, len(xor) + len(and_mask), 0, 0, 0, 0)
    return header + xor + and_mask


def ico_bytes(frames):
    """frames: [(size, RGBA 图)]；≤48 走 BMP，≥64 走 PNG（Windows 自带图标同款口径）。"""
    entries = [(size, bmp_entry(img) if size <= 48 else png_bytes(img)) for size, img in frames]
    head = struct.pack('<HHH', 0, 1, len(entries))
    offset = 6 + 16 * len(entries)
    directory, data = b'', b''
    for size, blob in entries:
        directory += struct.pack('<BBBBHHII', size % 256, size % 256, 0, 0, 1, 32, len(blob), offset)
        data += blob
        offset += len(blob)
    return head + directory + data


def write(path, data):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, 'wb') as fh:
        fh.write(data)
    print(f'{os.path.relpath(path, ROOT):58s} {len(data):>9,d} B')


def main():
    letter, bbox, master = load_letterform()
    print(f'母版 {master.size} 字形 bbox {bbox}')

    big = make_tile(1024, letter, 0.80)
    write(os.path.join(ROOT, 'build', 'appicon.png'), png_bytes(big))
    write(os.path.join(ROOT, 'build', 'appicon.svg'), svg_wrap(big, 'gaea — 玻璃 G 标志').encode('utf-8'))
    write(os.path.join(ROOT, 'build', 'appicon-small.svg'), svg_wrap(make_tile(512, letter, 0.92), 'gaea — 玻璃 G 标志').encode('utf-8'))
    compact = make_tile(1024, letter, 0.92)
    write(os.path.join(ROOT, 'build', 'windows', 'icon.ico'), ico_bytes(
        [(s, (compact if s <= 48 else big).resize((s, s), Image.LANCZOS)) for s in (16, 24, 32, 48, 64, 128, 256)]
    ))

    dark_svg = svg_wrap(make_tile(512, letter, 0.80), 'gaea — 深色玻璃 G 标志')
    write(os.path.join(ROOT, 'frontend', 'public', 'favicon.svg'), dark_svg.replace('深色', '').encode('utf-8'))
    write(os.path.join(ROOT, 'frontend', 'public', 'favicon-dark.svg'), dark_svg.encode('utf-8'))
    write(os.path.join(ROOT, 'frontend', 'public', 'favicon-light.svg'),
          svg_wrap(make_tile(512, letter, 0.80, light=True), 'gaea — 浅色玻璃 G 标志').encode('utf-8'))
    write(os.path.join(ROOT, 'frontend', 'public', 'gaea-brand-icon.png'), png_bytes(big))

    # UI 标：等比放进旧字形的框位，保持高度一致、水平居中
    lw, lh = bbox[2] - bbox[0], bbox[3] - bbox[1]
    bx0, by0, bx1, by1 = BRAND_BOX
    scale = min((bx1 - bx0) / lw, (by1 - by0) / lh)
    size = (int(round(lw * scale)), int(round(lh * scale)))
    canvas = Image.new('RGBA', BRAND_CANVAS, (0, 0, 0, 0))
    canvas.alpha_composite(master.crop(bbox).resize(size, Image.LANCZOS),
                           ((bx0 + bx1 - size[0]) // 2, (by0 + by1 - size[1]) // 2))
    write(os.path.join(ROOT, 'frontend', 'public', 'gaea-brand-mark.webp'),
          _webp(canvas.resize((BRAND_CANVAS[0] // 2, BRAND_CANVAS[1] // 2), Image.LANCZOS), 90))
    write(os.path.join(ROOT, 'frontend', 'public', 'boot', 'mark.webp'),
          _webp(letter.resize((1024, 1024), Image.LANCZOS), 92))


def _webp(img, quality):
    buf = io.BytesIO()
    img.save(buf, 'WEBP', quality=quality, method=6, exact=True)
    return buf.getvalue()


if __name__ == '__main__':
    main()
