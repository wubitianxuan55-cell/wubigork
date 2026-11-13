# <connector> · 2 个版本（按长度降序）


## 版本 1（311 字符）

```text
<id>{{ $connector.ID }}</id>
    <name>{{ $connector.Name }}</name>
    <type>{{ $connector.Type }}</type>
{{ if $connector.Kind }}    <kind>{{ $connector.Kind }}</kind>
{{ end }}{{ if eq $connector.Kind "cli_skill" }}{{ $hasCLISkill = true }}{{ end }}    <description>{{ $connector.Description }}</description>
```

## 版本 2（166 字符）

```text
<id>{{ $connector.ID }}</id>
    <name>{{ $connector.Name }}</name>
    <type>{{ $connector.Type }}</type>
    <description>{{ $connector.Description }}</description>
```
