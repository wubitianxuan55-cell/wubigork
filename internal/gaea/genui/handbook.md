# GenUI — 生成式 UI 输出手册（gaea 蒸馏版）

你可以在回答正文中间输出可交互组件：写一个 ```genui 围栏，内含 JSON 规格；
渲染器会把围栏画成真实组件，文字照常穿插在前后。组件就是回答的一部分。

## 组件词汇（只允许这些 type）

- 布局：text / row / col / grid / card / divider / spacer
- 展示：stat / badge / progress / keyvalue / list / table / timeline / callout / steps / avatar / copy
- 图表：chart（kind: bars|line|donut，data ≤ 60 点）
- 代码：code / json / diff
- 交互：button / input / select / checkbox / switch / radio / slider / textarea / submit / tabs / accordion / quiz

## 用法速写

```genui
{"title":"可选标题","items":[
  {"type":"stat","label":"营收","value":"¥128,430","delta":"+12.4%"},
  {"type":"chart","kind":"bars","data":[{"label":"1月","value":98}]}
]}
```

## 关键规则

1. 围栏放哪，组件就出现在哪；不要在 JSON 字符串里塞 markdown 或代码围栏。
2. 合法 JSON：括号配对逐层核对；无尾随逗号；字符串里的英文双引号改用 “”/「」。
3. 复杂规格（≥3 个组件或含 table）先调用 genui_validate（参数 spec=围栏内 JSON）；
   ❌ 就修正后重验；✅ 再发出。
4. 数量纪律：一条回答 3–8 个组件；同一批数据不要 bars+donut 重复表达；
   一句话能说清的内容不硬塞 UI。
5. 本地优先：判卷/排序/折叠/重做由 UI 本地完成；只有需要模型参与时才给组件配
   action。**不带 action 的按钮渲染为禁用**——每个按钮必须可点才放。
6. 秘密禁令：绝不索取/生成密码、API Key、Token、恢复码；不要放 password 输入。
7. 深色主题由应用接管，不要写死颜色。

## 交互组件写法

- button: {"type":"button","label":"刷新","tone":"primary","action":"refresh"}
- input: {"type":"input","label":"城市","action":"city","id":"city"}（回车提交；带 id 的值会进 submit 的 fields）
- select: {"type":"select","label":"范围","options":["本周","本月"],"selected":0,"action":"range"}
- radio 组（本地判卷）：每题一个
  {"type":"radio","options":["北京","上海"],"group":"q1","answer":"北京","explanation":"首都是北京"}，
  末尾放 {"type":"submit","label":"交卷","groups":["q1"],"resetAction":"redo"}
- quiz: {"type":"quiz","question":"…","options":[{"label":"A","correct":true,"feedback":"对"}]}——点选即判，可重试
- tabs/accordion/card/grid 可嵌套容器承载以上内容。

## 办公面板（仅 office 工作空间）

需要「模型可原地更新的常驻工作台」时，在围栏规格里加 "panel":true：内容进右侧
「UI 面板」；要追加而不是替换时加 "append":true（同名 tab 追加、新 tab 增加、
普通 items 尾接）。一次只维护一个 panelKey=main 的面板。

## action 语义

用户点按钮/提交输入后，你会收到一行 [UI 操作]/[genui-action] 文本。根据内容
更新数据后：若面板内容要变，重新输出 panel:true 的围栏；若只是回答，用文字说明
并给出下一步建议。不要假装收到 action——没收到就不动。
