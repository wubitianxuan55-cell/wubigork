# gaea 一致性评分 v1·文字锚点方案（v4.404.0）规格书

> 2026-09-23 立项。来源：用户拍板「一致性评分」（v4.400 观察池挂池项，
> v1 用 vision LLM 文字锚点评分的方案随拍板解锁）。目标版本：v4.404.0。

## 1. 论点

设定卡/剧照生成后「像不像这个角色」只能靠人眼。v1 用视觉 LLM 拿生成图对照
角色**文字设定**（Appearance/Figure）打分（0-100+总评+不一致点），给参考
图资产一个可比较的质量读数；低分驱动「补参考」提示——闭环的读取侧基建。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | **文字锚点方案**（vision LLM 拿图对照文字设定），非图 vs 图对比——复用既有 vision.RecognizeImage 单图链路零改动（多图输入本地 Qwen 视觉运行时未验证，挂观察池 V2）；文字锚点缺 Appearance/Figure 时可用（降级为通用形象评审） |
| Q2 | 新绑定 `CharacterScoreConsistency(chJSON, image) (string, error)`（CharlibB，716→717，play）：image 收本地路径或 data URL（前端新生成图是 data URL、已存参考是本地路径）；远端 URL 如实拒绝；返回规范化 JSON `{"score":0-100,"summary":"…","issues":[…]}` |
| Q3 | data URL→临时文件落盘复用既有视觉链（scoreImageToLocalPath 助手可独立测试，cleanup 即删）；视觉调用走 seam `characterScoreVision`（测试注入，对齐 visionRecognize 既有模式） |
| Q4 | prompt 纯函数 `buildConsistencyScorePrompt`：评审维度（面部/发型发色/体型/年龄感，服装仅当文字有）+严格 JSON 契约+四级评分标准（90+/70-89/50-69/<50）+锚点截断 160 rune（与设定卡同预算） |
| Q5 | 回复解析：ExtractJSON 提取（围栏容错）→score 数值校验+钳位 0-100（四舍五入）→issues 字符串清洗（非空、≤5 防御截断）；缺 score/非 JSON 如实报错带原始返回前 200 字符 |
| Q6 | 前端：CharacterLibEditor 参考图列表每张图加「评分」钮（AimOutlined，与剧照/移除并列）→结果 Modal.info（分数标题+总评+不一致点列表）；**score<60 提示「建议补充不同角度/光照的参考图后再生成」**（评分驱动补参考 v1）；评分进行中禁其他评分钮（单飞） |
| Q7 | V2 挂池：图 vs 参考图对比评分（需视觉运行时多图验证）；sin 侧评分入口；评分写回参考图元数据 |

## 3. 测试

- Go：prompt 纯函数（维度/JSON 契约/锚点/无外观可用）；解析（正常/围栏/
  钳位上下界/四舍五入/缺 score/非数值/非 JSON/issues 清洗）；绑定 seam 注入
  （规范 JSON 回传/data URL 原样给 seam/路径透传/临时文件助手落盘+清理/
  缺逗号报错/无名/空图/远端 URL/视觉失败透传/坏回复报错）。
- 前端：评分钮调用透传参考图+Modal 分数与 issues；高分无补参考提示；低分
  有提示；失败 message+按钮复位。

## 4. 门禁

全量 ci、版本漂移闸 OK@4.404.0、绑定面 717、spaceBindings 540。

## 5. 观察池

图 vs 参考图对比评分（视觉运行时多图输入验证）；sin 侧评分入口；评分写回
参考图元数据；评分驱动自动重生成。
