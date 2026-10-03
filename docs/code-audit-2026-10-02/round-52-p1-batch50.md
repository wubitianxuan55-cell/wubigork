# 全仓审计第 50 批 · 观察池收口（WhisperSubgraph 键名大小写真机 bug 坐实修复）· 2026-10-03

> 接续 [round-51（批次四十九）](round-51-p1-batch49.md)。批 46 登记的观察池疑点静态坐实并修复：**真机关系图谱面板功能性死亡 bug**。申报行为修复（线上 JSON 键名变化），**不抬版本**。
> 快照：开工时 HEAD = `cd7a0690`（批次四十九）；工作树干净。

---

## 一、疑点坐实（纯代码链路判定，无需真机）

- **链路**：Go `whisper.Subgraph/GraphNode/GraphEdge` **无 json 标签** → Wails 线上序列化大写键（`Nodes/Edges/ID/Name/From/EmotionLabel`…）→ `app.WhisperGraphSubgraph` 直达 → `WhisperGraphPanel.tsx` 读 `graph.nodes ?? []`（契约小写）→ **恒 undefined**。
- **症状**：面板不崩（`?? []` 兜底）但**永远渲染空图**——轻语「关系图谱」查询在真机上自上线起功能性死亡；dev mock（批 46 起按契约小写）反而是好的。bridge 契约面（`WhisperSubgraph` 小写接口）与 Go 无标签类型的分叉是 X1-11 家族的又一实证。
- **修正方向**：Go 侧补 json 标签对齐契约（不动 UI、不动 mock）——`nodes/edges/id/name/type/weight/from/to/emotionLabel`；`EmotionLabel` 带 `omitempty`（空=中性，对齐契约可选标注与包内 types.go 既有 camelCase 标签风格——漏标是 memory_graph.go 的孤例异常）。
- **回归面核查**：大写键零消费方（`grep .Nodes/.Edges` 前端仅 wailsjs 生成物且无导入；Go 侧无键名断言）——线上键名变化无第二受害者。

## 二、落地与门禁

- `internal/whisper/memory_graph.go`：三结构补 json 标签 + 注释记因。
- `memory_graph_subgraph_test.go`：`TestSubgraphJSONKeysContract` 契约钉（marshal 后断言 nodes/edges/id/name/type/weight/from/to/emotionLabel 小写键全在——键名回退即红）。
- mock/memory.ts 观察池注释收口（疑点已解，指向本批）。
- gofmt 0 差 / `go build ./...` 绿 / whisper 全包 1.7s 绿（QuerySubgraph 九例既有语义零漂移）/ tsc -b 绿 / **全量 vitest 435 文件 3762 例全绿**（WhisperGraphPanel 17 例在列）。

## 三、台账

- **观察池清零**（唯一在册项 WhisperSubgraph 已收口）。
- 审计线终态：P0 25 全清；四簇/留池/上报未动池全清或归拍板；免拍板活池零。余=拍板池十项 + 109 绑定删除 + fsync 决策 + WhisperGraphPanel 真机复走（随下次真机窗口顺带验证图谱查询出图）。

## 四、下一批

- 请用户批量裁决拍板池；或 WhisperGraphPanel 真机复走（等闲置窗口）；或新审计周期。
