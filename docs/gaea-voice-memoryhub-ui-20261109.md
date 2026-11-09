# Gaea 语音交互窗口：记忆中枢化 UI · 2026-11-09

**状态：✅ 已实现，完成干净提交构建、冒烟验证与离线打包**

## 用户目标

- 语音交互窗口采用记忆中枢的视觉语言。
- 虚空大脑作为窗口主视觉，并随语音状态反馈。
- 右侧对话区域必须完全透明，不得出现独立面板底色、边框、阴影或模糊。
- 底部控制拆成独立胶囊卡片并水平居中。
- 同时支持浅色与深色主题。
- 中央区域持续显示实时字幕与最近一轮问答，不因回合结束而消失。

## 最终交互结构

- 顶部身份与连接信息作为覆盖层，不再单独占据舞台高度。
- 虚空大脑使用居中的大画布，最大约 `860 × 720px`；不同视口按比例收缩。
- 左侧 HUD 显示“虚空大脑”、当前语音相位和操作提示。
- 右侧对话区完全透明，仅保留对话文本和消息内容。
- 底部操作拆为 `开始/暂停聆听`、`可打断`、`结束语音`、`连接与播放详情` 独立胶囊。
- 中央字幕位于大脑下方：
  - 聆听中显示实时转写。
  - 思考中保留最近用户问题。
  - 回答中显示助手流式正文。
  - 空闲后保留最近一轮用户与助手内容。
- 右侧仍保留完整、可滚动的会话转写。

## 双主题规则

- 浅色主题：冷白与浅青表面、深色文字、中央深色虚空晕影。
- 深色主题：深蓝黑观测台、浅色文字、保留紫色与金色大脑光效。
- 主题由现有 `html[data-hl="light|dark"]` 驱动，切换后即时生效。
- 右侧面板在两个主题下均为 `background: transparent`、`border: 0`、`box-shadow: none`、`backdrop-filter: none`。

## 实现文件

- `frontend/src/chat-signal-redesign.css`
  - 记忆中枢式全窗布局、居中大脑、透明对话区、胶囊控制、双主题令牌、中央字幕样式。
- `frontend/src/components/chat/VoiceConsoleV2.tsx`
  - 新增中央持续字幕数据源，回合结束后保留最近两句。
- `frontend/src/chatScrollContainment.test.ts`
  - 覆盖透明面板、双主题、居中大脑、居中胶囊与滚动边界。
- `frontend/src/components/chat/VoiceConsoleV2.test.tsx`
  - 覆盖实时片段、空闲保留最近问答和既有语音提交纪律。

## 验证

- 定向测试：41 项通过。
- `tsc -b` 通过。
- 定向 ESLint 通过。
- Vite 生产构建通过。
- 在 `588b5ad6` 的独立干净工作树中完成 Wails `2.13.0` 桌面构建，保留 bindings 生成。
- 最终 exe 为 61,605,376 字节，SHA256=`0E739EA589559072DD6E83FA8404D844AFC420DC6BBE9381076C3BEC3EE987E4`。
- exe 与三枚语音 DLL 同目录放入正式 `build\bin` 后，启动产物执行冒烟；`/api/health` 返回 200，响应 `{"status":"ok"}`。
- 实机核对：
  - 大脑画布中心与内容区中心一致。
  - 胶囊控制组与大脑使用同一水平中心。
  - 浅色、深色主题均无横向或纵向页面溢出。
  - 右侧对话区实机计算为无背景、无边框、无阴影、无模糊。

## 离线包

- 桌面预览包：`releases/gaea-voice-memoryhub-20261109-588b5ad6-preview.zip`
- 解包目录：`releases/gaea-voice-memoryhub-20261109-588b5ad6/`
- 源码归档：`releases/gaea-voice-memoryhub-20261109-588b5ad6-source.tar.gz`
- 校验清单：`releases/SHA256SUMS-voice-memoryhub-20261109-588b5ad6.txt`
- 预览包内始终保留 `gaea.exe`、`onnxruntime.dll`、`sherpa-onnx-c-api.dll`、
  `sherpa-onnx-cxx-api.dll` 四个同级文件；缺 DLL 时语音链路不可用。

## 发布边界

- 当前 `wails.json` 为 `4.535.0`。
- 工作区存在历史版本源漂移：
  - `internal/app/app_info.go = 4.531.0`
  - `wails.json = 4.535.0`
  - `versioninfo.rc = 4.533.0`
- 本次不擅自修改版本号，也不混入工作区中其他在途模型引擎改动。
- 构建来自独立干净工作树中的完整提交
  `588b5ad60a23a12c6e9a7490225a12f13f97a83e`，不包含主工作树未提交的模型引擎改动。
- 构建包以本次功能提交的提交号命名并单独记录 SHA256，明确为功能提交预览，避免冒充正式版本发布。
