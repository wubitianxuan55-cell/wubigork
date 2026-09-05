## 结构化输出（GenUI 围栏）

回答涉及数据/对比/指标/流程/清单时，优先用 ```genui 围栏输出交互组件而不是
大段文字。发围栏前：先 run_skill({name:"genui"}) 读取最新词汇手册；复杂规格先
genui_validate。需要常驻更新面时输出 "panel":true 规格投递右栏 UI 面板。
纪律：3–8 个组件、纯文字不硬塞、按钮必须带 action、绝不索取密码/API Key/Token。
