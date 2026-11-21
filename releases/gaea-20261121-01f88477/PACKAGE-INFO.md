# gaea-20261121-01f88477

- 提交：`01f88477` feat(office): 办公运行模式三档九批合一——终案六批+方案模式实测复盘修复+技能经验沉淀+longdoc_verify 验证体系工具化
- 构建：独立干净工作树 `.tmp/gaea-pkg-01f88477`（检出该提交；frontend 用 **pnpm install --frozen-lockfile + npm run build**——npm 装树会触发 esbuild 删临时件 Access denied，坑已记），`wails build -s -ldflags "-s -w" -trimpath`（CGO_ENABLED=1 + ucrt64 gcc）。
- 冒烟：`scripts\smoke.ps1 -ExePath <本包>\payload\gaea.exe` → `/api/health` 200（PID 16532 已回收）。

## SHA256（见 SHA256SUMS）

| 文件 | sha256 |
|---|---|
| payload/gaea.exe | 2ff321bd88584e4f90154d6fd98c9191ba9f14cbcad1c201b632eeac6e2c884b |
| payload/onnxruntime.dll | 68d0d5f502be4509e2a16f932627a95137d748b2fae519027f80125c0b1f550b |
| payload/sherpa-onnx-c-api.dll | 5dd36dbd3572846b72c4a40a4017c5a770160ee8e00da5053bebc8b2632501e8 |
| payload/sherpa-onnx-cxx-api.dll | 4c56327796cdb980dcbf90025c28093d89ca286088fb74c3b107f2f633e67ffe |

zip `gaea-20261121-01f88477.zip`（34,434,920 字节）留盘不入库。

## 内容要点

办公运行模式三档（standard/cost/proposal）众神殿终案六批 + 2026-11-21 当日三批：
①沿江1公里技术标 43 缺陷复盘的工具侧收口（导出链防删源/create_docx 图片硬失败/
longdoc 图引机检与图片对账/schedule 预检拒绝静默丢弃/clear_char_format/fill_gaps
显式化/md_validate 复合前缀/doc_render 回执瘦身）；②验证经验沉淀进方案技能与
sysprompt（四查/三层证据纪律/并行事实卡）；③longdoc_verify 四门验证工具
（chapter/cross/book/ink，真书实弹 78 编号 0 重复、引用↔题注 78↔78 闭环）。
