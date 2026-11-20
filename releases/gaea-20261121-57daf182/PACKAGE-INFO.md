# gaea 功能包 gaea-20261121-57daf182

- **提交**：57daf182（四线合一：D1–D10 缺陷批 + 实测九刀 + AI 反馈机制 + 记忆重梳/缓存刀）
- **构建**：2026-11-21 干净工作树（.tmp/pkg-f96b9e3f @ 57daf182）
  - 前端 pnpm frozen install + `npm run build`（tsc 0，28.3s）
  - `PATH=ucrt64 CGO_ENABLED=1 wails build -s -ldflags "-s -w" -trimpath`（17.7s，Skip Bindings=false）
- **内容**：gaea.exe + onnxruntime.dll + sherpa-onnx-c-api.dll + sherpa-onnx-cxx-api.dll
  - DLL 与上一包（33e16fee）逐字节同源（第三方运行时）
- **冒烟**：装机位（C:\AI\gaea-app）/api/health 200（见 SHA256SUMS.txt 同目录记录与 progress.md）
- **要点**：auto-dream 全家删除 / remember 口径回调（主动调+确认卡）/
  项目简报跨天衰减两遍法（L1 前缀缓存恒定）/ ai_feedback 三板块 /
  md_validate / read_file section / doc_render total_pages / 回执 metrics 固化
