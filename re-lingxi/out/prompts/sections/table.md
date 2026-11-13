# <table> · 2 个版本（按长度降序）


## 版本 1（569 字符）

```text
{{range $b := .Buckets}}
{{if $b}}
  <tr>
    <td style="padding:0 0 0 0.25em">[</td>
    <td style="text-align:right;padding:0 0.25em">{{.Lower}},</td>
    <td style="text-align:right;padding:0 0.25em">{{.Upper}})</td>
    <td style="text-align:right;padding:0 0.25em">{{.N}}</td>
    <td style="text-align:right;padding:0 0.25em">{{printf "%#.3f" .Pct}}%</td>
    <td style="text-align:right;padding:0 0.25em">{{printf "%#.3f" .CumulativePct}}%</td>
    <td><div style="background-color: blue; height: 1em; width: {{.GraphWidth}};"></div></td>
  </tr>
{{end}}
{{end}}
```

## 版本 2（267 字符）

```text
<tr>
    <td style="padding:0.25em">Count: {{.Count}}</td>
    <td style="padding:0.25em">Mean: {{printf "%.0f" .Mean}}</td>
    <td style="padding:0.25em">StdDev: {{printf "%.0f" .StandardDeviation}}</td>
    <td style="padding:0.25em">Median: {{.Median}}</td>
</tr>
```
