# gaea 板块蓝图 · weixin 微信助手「通讯枢纽」

> 覆盖 MASTER。v4.47.0 落地（WeixinPage.tsx + weixin-page.css），取代 misc.md 中
> 「服务型板块，无页面」的过时定位（v4.4 起 WeixinPage 已在 rail）。

## 视觉性格：通讯枢纽（Link Hub）
服务型工作台 = 实时运营台语义：通道状态一眼可读、绑定流渐进引导、提醒即代办
收件箱。对齐 Constellation OS 三分区语言（strip + 左轨道 + 主区），信息层用
`v3-panel`/`v3-card` 实底卡，容器层 Luminous Glass 2.0。

## 布局（wx-* 类，见 frontend/src/pages/weixin-page.css）
- **顶部细条 `.wx-strip`**（40px 玻璃）：WechatOutlined + 「微信助手」+ 一句话
  定位 + 通道遥测 meta（等宽数字 `3 通道 · 1 运行中 · 提醒 开`，role=status）
  + 刷新按钮。原整卡介绍收敛为一句副标语，完整说明移到指南视图。
- **左轨道 `.wx-rail`（196px）**：通道组 = 每助手一条 rail item（antd Avatar
  26px + 名字 + 状态字 + 状态点）；「+ 新增微信助手」dashed 项挂组尾；
  「支持」组 = 离线提醒（pending 数徽标）/ 使用指南。激活态 =
  primary-container 底 + 左缘 `--gaea-glow` 光条（对齐 hub-rail）。
- **主区 `.wx-main` 三视图**（rail 点击驱动）：
  1. 通道详情 `ChannelDetail`：身份头（Avatar 44 + 核心/人格 Tag + 绑定/删除
     操作）→ 会话过期内联 Alert → 三张键值卡（通道状态/微信绑定/启停 Switch）。
  2. 离线提醒：标题 + 全局「到点回推微信」Switch → 新建行（Input+DatePicker+
     创建）→ `wx-rem-item` 行（状态 Tag/文本/来源/重试/等宽时间/删除）。
  3. 使用指南：三张 `wx-guide-card`（设提醒示例句 `wx-cmd` 等宽 chip /
     收提醒 / 聊天）+ 脚注。

## 状态语义（三重传达：色 + 文字 + 详情 Tag）
`channelStatusOf`：running=success 发光脉冲（reduced-motion 关闭）/
expired=warning / stopped=次级 62% / unbound=虚线空心点。rail 状态字 +
`title` + 详情区完整 antd Tag 同口径。

## 关键流
- 扫码绑定 Modal：`wx-qr-steps` 三步指示（扫码→确认→完成，等宽字）+ 240px
  `wx-qr-frame` 二维码容器；相位文案/配对码/错误分支照旧。
- 新增助手 Modal（名字 + 人格 ID）→ 扫码 confirmed 落库 → 主区直接落在新通道。
- 会话过期：选中通道详情内联 Alert（全局任一过期的语义保留在状态字）。

## 纪律
- 零硬编码色值：只消费 --md-sys-* / --gaea-* / --color-* / --v3-* 与旧栈
  令牌，派生一律 color-mix。
- 功能零删减（v4.4 面逐项对齐）：多助手 CRUD/逐助手扫码/启停/核心禁删禁停/
  提醒 CRUD + 全局开关/使用说明/过期警示。
- 测试锁：WeixinPage.test.tsx 8 场景（轨道渲染/启停载荷/核心限制/删除/
  新增扫码流/提醒视图/指南视图/过期警示）。
- dev mock：`gaea/lib/mock/weixin.ts` 提供微信域离线数据（伪二维码 SVG、
  扫码相位推进 2→4 次轮询）。
