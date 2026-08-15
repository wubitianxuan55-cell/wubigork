# gaea 3.0 板块蓝图 · imagegen 绘梦

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。

## 现状（facts）
- 页面：`frontend/src/pages/ImageGenPage.tsx`（310 行）+ components/imagegen/* + hooks（useImageGenQueue/History/Config）
- 可折叠分区/底部生成栏/任务中心三 Tab/玻璃 HUD；生成队列/历史/灯箱（Lightbox）
- 多后端（本地 ComfyUI：Flux/Z-Image-Turbo/Krea2 + LoRA/风格预设 + 云端 xAI）；进度事件

## 目标态
- **视觉性格：画廊工作台**——历史瀑布（创作资产）+ 参数面板（工具）+ 生成队列（状态）。
- 历史瀑布：图片卡（圆角 `--radius-md` + 悬停 `--elevation-2` + 操作浮现：下载/剧照/删除）；
  - 灯箱（Lightbox）：全屏玻璃遮罩 + 图 + 元数据（模型/尺寸/耗时小字）。
- 参数面板：折叠分区（模型/尺寸/风格/LoRA），表单走令牌；生成按钮主色胶囊。
- 生成队列/任务：进度条主色；状态色（排队=secondary / 运行=primary 脉冲 / 完成=success /
  失败=destructive），色+图标+文字。
- 底部生成栏：玻璃输入条 + 提示词 + 附加（图生图）+ 生成按钮。

## 落地清单
- [ ] 历史卡/灯箱/队列令牌化（清硬编码）
- [ ] 生成状态四态语义色统一
- [ ] 参数面板 focus 环 + 生成按钮微交互

## 验收
- 12 主题下图片卡/灯箱对比度正常；队列状态可区分（色+图标）；焦点环可见。
