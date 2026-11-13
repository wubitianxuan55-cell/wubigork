# <mask_strokes> · 1 个版本（按长度降序）


## 版本 1（216 字符）

```text
{{- range .CanvasImageEdit.MaskStrokes }}
    <stroke width="{{ .Width }}">{{ range $i, $p := .Points }}{{ if $i }} {{ end }}{{ range $j, $v := $p }}{{ if $j }},{{ end }}{{ $v }}{{ end }}{{ end }}</stroke>
{{- end }}
```
