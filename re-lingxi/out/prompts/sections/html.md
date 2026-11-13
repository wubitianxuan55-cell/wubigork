# <html> · 2 个版本（按长度降序）


## 版本 1（3383 字符）

```text
<head>
	<title>/debug/requests</title>
	<style type="text/css">
		body {
			font-family: sans-serif;
		}
		table#tr-status td.family {
			padding-right: 2em;
		}
		table#tr-status td.active {
			padding-right: 1em;
		}
		table#tr-status td.latency-first {
			padding-left: 1em;
		}
		table#tr-status td.empty {
			color: #aaa;
		}
		table#reqs {
			margin-top: 1em;
		}
		table#reqs tr.first {
			{{if $.Expanded}}font-weight: bold;{{end}}
		}
		table#reqs td {
			font-family: monospace;
		}
		table#reqs td.when {
			text-align: right;
			white-space: nowrap;
		}
		table#reqs td.elapsed {
			padding: 0 0.5em;
			text-align: right;
			white-space: pre;
			width: 10em;
		}
		address {
			font-size: smaller;
			margin-top: 5em;
		}
	</style>
	</head>
	<body>

<h1>/debug/requests</h1>
{{end}} {{/* end of Prolog */}}

{{define "StatusTable"}}
<table id="tr-status">
	{{range $fam := .Families}}
	<tr>
		<td class="family">{{$fam}}</td>

		{{$n := index $.ActiveTraceCount $fam}}
		<td class="active {{if not $n}}empty{{end}}">
			{{if $n}}<a href="?fam={{$fam}}&b=-1{{if $.Expanded}}&exp=1{{end}}">{{end}}
			[{{$n}} active]
			{{if $n}}</a>{{end}}
		</td>

		{{$f := index $.CompletedTraces $fam}}
		{{range $i, $b := $f.Buckets}}
		{{$empty := $b.Empty}}
		<td {{if $empty}}class="empty"{{end}}>
		{{if not $empty}}<a href="?fam={{$fam}}&b={{$i}}{{if $.Expanded}}&exp=1{{end}}">{{end}}
		[{{.Cond}}]
		{{if not $empty}}</a>{{end}}
		</td>
		{{end}}

		{{$nb := len $f.Buckets}}
		<td class="latency-first">
		<a href="?fam={{$fam}}&b={{$nb}}">[minute]</a>
		</td>
		<td>
		<a href="?fam={{$fam}}&b={{add $nb 1}}">[hour]</a>
		</td>
		<td>
		<a href="?fam={{$fam}}&b={{add $nb 2}}">[total]</a>
		</td>

	</tr>
	{{end}}
</table>
{{end}} {{/* end of StatusTable */}}

{{define "Epilog"}}
{{if $.Traces}}
<hr />
<h3>Family: {{$.Family}}</h3>

{{if or $.Expanded $.Traced}}
  <a href="?fam={{$.Family}}&b={{$.Bucket}}">[Normal/Summary]</a>
{{else}}
  [Normal/Summary]
{{end}}

{{if or (not $.Expanded) $.Traced}}
  <a href="?fam={{$.Family}}&b={{$.Bucket}}&exp=1">[Normal/Expanded]</a>
{{else}}
  [Normal/Expanded]
{{end}}

{{if not $.Active}}
	{{if or $.Expanded (not $.Traced)}}
	<a href="?fam={{$.Family}}&b={{$.Bucket}}&rtraced=1">[Traced/Summary]</a>
	{{else}}
	[Traced/Summary]
	{{end}}
	{{if or (not $.Expanded) (not $.Traced)}}
	<a href="?fam={{$.Family}}&b={{$.Bucket}}&exp=1&rtraced=1">[Traced/Expanded]</a>
        {{else}}
	[Traced/Expanded]
	{{end}}
{{end}}

{{if $.Total}}
<p><em>Showing <b>{{len $.Traces}}</b> of <b>{{$.Total}}</b> traces.</em></p>
{{end}}

<table id="reqs">
	<caption>
		{{if $.Active}}Active{{else}}Completed{{end}} Requests
	</caption>
	<tr><th>When</th><th>Elapsed&nbsp;(s)</th></tr>
	{{range $tr := $.Traces}}
	<tr class="first">
		<td class="when">{{$tr.When}}</td>
		<td class="elapsed">{{$tr.ElapsedTime}}</td>
		<td>{{$tr.Title}}</td>
		{{/* TODO: include traceID/spanID */}}
	</tr>
	{{if $.Expanded}}
	{{range $tr.Events}}
	<tr>
		<td class="when">{{.WhenString}}</td>
		<td class="elapsed">{{elapsed .Elapsed}}</td>
		<td>{{if or $.ShowSensitive (not .Sensitive)}}... {{.What}}{{else}}<em>[redacted]</em>{{end}}</td>
	</tr>
	{{end}}
	{{end}}
	{{end}}
</table>
{{end}} {{/* if $.Traces */}}

{{if $.Histogram}}
<h4>Latency (&micro;s) of {{$.Family}} over {{$.HistogramWindow}}</h4>
{{$.Histogram}}
{{end}} {{/* if $.Histogram */}}

	</body>
```

## 版本 2（2070 字符）

```text
<head>
		<title>events</title>
	</head>
	<style type="text/css">
		body {
			font-family: sans-serif;
		}
		table#req-status td.family {
			padding-right: 2em;
		}
		table#req-status td.active {
			padding-right: 1em;
		}
		table#req-status td.empty {
			color: #aaa;
		}
		table#reqs {
			margin-top: 1em;
		}
		table#reqs tr.first {
			{{if $.Expanded}}font-weight: bold;{{end}}
		}
		table#reqs td {
			font-family: monospace;
		}
		table#reqs td.when {
			text-align: right;
			white-space: nowrap;
		}
		table#reqs td.elapsed {
			padding: 0 0.5em;
			text-align: right;
			white-space: pre;
			width: 10em;
		}
		address {
			font-size: smaller;
			margin-top: 5em;
		}
	</style>
	<body>

<h1>/debug/events</h1>

<table id="req-status">
	{{range $i, $fam := .Families}}
	<tr>
		<td class="family">{{$fam}}</td>

	        {{range $j, $bucket := $.Buckets}}
	        {{$n := index $.Counts $i $j}}
		<td class="{{if not $bucket.MaxErrAge}}active{{end}}{{if not $n}}empty{{end}}">
	                {{if $n}}<a href="?fam={{$fam}}&b={{$j}}{{if $.Expanded}}&exp=1{{end}}">{{end}}
		        [{{$n}} {{$bucket.String}}]
			{{if $n}}</a>{{end}}
		</td>
                {{end}}

	</tr>{{end}}
</table>

{{if $.EventLogs}}
<hr />
<h3>Family: {{$.Family}}</h3>

{{if $.Expanded}}<a href="?fam={{$.Family}}&b={{$.Bucket}}">{{end}}
[Summary]{{if $.Expanded}}</a>{{end}}

{{if not $.Expanded}}<a href="?fam={{$.Family}}&b={{$.Bucket}}&exp=1">{{end}}
[Expanded]{{if not $.Expanded}}</a>{{end}}

<table id="reqs">
	<tr><th>When</th><th>Elapsed</th></tr>
	{{range $el := $.EventLogs}}
	<tr class="first">
		<td class="when">{{$el.When}}</td>
		<td class="elapsed">{{$el.ElapsedTime}}</td>
		<td>{{$el.Title}}</td>
	</tr>
	{{if $.Expanded}}
	<tr>
		<td class="when"></td>
		<td class="elapsed"></td>
		<td><pre>{{$el.Stack|trimSpace}}</pre></td>
	</tr>
	{{range $el.Events}}
	<tr>
		<td class="when">{{.WhenString}}</td>
		<td class="elapsed">{{elapsed .Elapsed}}</td>
		<td>.{{if .IsErr}}E{{else}}.{{end}}. {{.What}}</td>
	</tr>
	{{end}}
	{{end}}
	{{end}}
</table>
{{end}}
	</body>
```
