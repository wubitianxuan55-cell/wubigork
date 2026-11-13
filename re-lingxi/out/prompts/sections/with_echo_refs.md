# <with_echo_refs> · 1 个版本（按长度降序）


## 版本 1（181 字符）

```text
{{- range $index, $ref := .WithEchoRefs }}
  <with_echo_ref index="{{$index}}">
    <type>{{xmlText $ref.Type}}</type>
    <id>{{xmlText $ref.ID}}</id>
  </with_echo_ref>
{{- end }}
```
