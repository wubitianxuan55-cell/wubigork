# <webpage_edit_selections> · 1 个版本（按长度降序）


## 版本 1（316 字符）

```text
{{- range $index, $sel := .WebpageEditSelections }}
  <selection index="{{index1 $index}}">
    <file>{{xmlText $sel.FilePath}}</file>
    <selector>{{xmlText $sel.Selector}}</selector>
    <tag>{{xmlText $sel.TagName}}</tag>
    <selected_text>{{xmlText $sel.SelectedText}}</selected_text>
  </selection>
{{- end }}
```
