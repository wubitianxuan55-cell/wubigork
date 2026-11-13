# <wps_docs_context> · 1 个版本（按长度降序）


## 版本 1（419 字符）

```text
本轮涉及wps文档（如下所列）。
<references>
{{- range .WpsDocsContext.References }}
<doc link_id="{{xmlText .LinkID}}" doc_type="{{xmlText .DocType}}"{{ if .ActiveSheet }} active_sheet="{{xmlText .ActiveSheet}}"{{ end }}{{ if .SlideIndex }} slide_index="{{.SlideIndex}}"{{ end }}{{ if .SlideID }} slide_id="{{.SlideID}}"{{ end }} />
{{- end }}
</references>
active_sheet 是用户当前浏览的工作表，slide_index / slide_id 是当前幻灯片；请结合用户问题判断该位置是否为用户所指。
```
