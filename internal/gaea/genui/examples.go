package genui

// examples.go — 每个组件的最小可用示例（单源）。
//
// 两个消费方共用这张表：①校验器报「缺少必填字段」时附上对应示例，模型照着改即可
// （2026-11-24 实弹：callout 缺 content 只能靠报错才知道必填）；②手册 handbook.md 的
// 必填字段表——由 TestHandbookExamplesInSync 保证与这里逐字一致，改这里必须同步改手册。
// 字段形态以 frontend/src/genui/spec.ts 为准；只列必填字段与一两个最常用的可选字段。

var minimalExample = map[string]string{
	"text":      `{"type":"text","content":"一句话"}`,
	"callout":   `{"type":"callout","title":"可选标题","content":"一句话","tone":"info"}`,
	"row":       `{"type":"row","items":[{"type":"text","content":"左"},{"type":"text","content":"右"}]}`,
	"col":       `{"type":"col","items":[{"type":"text","content":"上"},{"type":"text","content":"下"}]}`,
	"grid":      `{"type":"grid","cols":2,"items":[{"type":"text","content":"左"},{"type":"text","content":"右"}]}`,
	"card":      `{"type":"card","title":"可选标题","items":[{"type":"text","content":"内容"}]}`,
	"accordion": `{"type":"accordion","items":[{"title":"标题","items":[]}]}`,
	"tabs":      `{"type":"tabs","tabs":[{"label":"页签","items":[]}]}`,
	"divider":   `{"type":"divider"}`,
	"spacer":    `{"type":"spacer"}`,
	"stat":      `{"type":"stat","label":"营收","value":"¥128,430","delta":"+12.4%"}`,
	"badge":     `{"type":"badge","label":"进行中","tone":"success"}`,
	"progress":  `{"type":"progress","label":"完成率","value":62}`,
	"keyvalue":  `{"type":"keyvalue","pairs":[{"key":"工期","value":"90天"}]}`,
	"list":      `{"type":"list","items":["一项","两项"]}`,
	"table":     `{"type":"table","columns":["名称","数量"],"rows":[["降水井","12"]]}`,
	"timeline":  `{"type":"timeline","items":[{"title":"开工","time":"3月"}]}`,
	"steps":     `{"type":"steps","current":1,"steps":[{"title":"勘察"},{"title":"设计"}]}`,
	"avatar":    `{"type":"avatar","name":"张工"}`,
	"copy":      `{"type":"copy","label":"复制","text":"要复制的文本"}`,
	"chart":     `{"type":"chart","kind":"bars","data":[{"label":"1月","value":98}]}`,
	"code":      `{"type":"code","lang":"python","code":"print(1)"}`,
	"json":      `{"type":"json","value":{"a":1}}`,
	"diff":      `{"type":"diff","diffs":[{"path":"a.md","oldText":"旧","newText":"新"}]}`,
	"button":    `{"type":"button","label":"刷新","tone":"primary","action":"refresh"}`,
	"input":     `{"type":"input","label":"城市","action":"city","id":"city"}`,
	"select":    `{"type":"select","label":"范围","options":["本周","本月"],"selected":0,"action":"range"}`,
	"checkbox":  `{"type":"checkbox","label":"含税","checked":false,"action":"tax"}`,
	"switch":    `{"type":"switch","label":"自动保存","checked":true,"action":"autosave"}`,
	"radio":     `{"type":"radio","options":["北京","上海"],"group":"q1","answer":"北京","explanation":"首都是北京"}`,
	"slider":    `{"type":"slider","label":"置信度","min":0,"max":100,"value":70,"action":"conf"}`,
	"textarea":  `{"type":"textarea","label":"备注","rows":4,"action":"note","id":"note"}`,
	"submit":    `{"type":"submit","label":"交卷","groups":["q1"],"resetAction":"redo"}`,
	"quiz":      `{"type":"quiz","question":"首都是？","options":[{"label":"北京","correct":true,"feedback":"对"},{"label":"上海"}]}`,
}

// MinimalExample 某组件类型的最小可用示例；没有登记的返回空串。
func MinimalExample(typ string) string { return minimalExample[typ] }
